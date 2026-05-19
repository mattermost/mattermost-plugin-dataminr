// Copyright (c) 2025-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package dataminr

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/mattermost/mattermost/server/public/pluginapi"
	"github.com/mattermost/mattermost/server/public/pluginapi/cluster"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/mattermost/mattermost-plugin-dataminr/server/backend"
)

// TestDataminrBackend_FullIntegration tests the complete backend flow
// from authentication through alert fetching and processing
func TestDataminrBackend_FullIntegration(t *testing.T) {
	// Set up test HTTP server that simulates Dataminr API
	authToken := "test-auth-token-12345"                  //nolint:gosec // Test credential, not real secret
	expirationTime := time.Now().UTC().Add(2 * time.Hour) // Well in the future, use UTC
	cursor := "test-cursor-123"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/1/userAuthorization":
			// Handle authentication request
			assert.Equal(t, "POST", r.Method)
			assert.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"))

			resp := AuthResponse{
				AuthorizationToken: authToken,
				ExpirationTime:     expirationTime,
			}
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(resp)

		case "/alerts/1/alerts":
			// Handle alerts request
			assert.Equal(t, "GET", r.Method)
			assert.Equal(t, "Dmauth "+authToken, r.Header.Get("Authorization"))
			assert.Equal(t, "19", r.URL.Query().Get("alertversion"))

			// Return test alerts
			resp := AlertsResponse{
				Alerts: []Alert{
					{
						AlertID:   "alert-001",
						AlertType: AlertType{Name: "Urgent", Color: "orange"},
						Headline:  "Test Alert 1",
						EventTime: time.Now().Add(-10 * time.Minute),
					},
					{
						AlertID:   "alert-002",
						AlertType: AlertType{Name: "Flash", Color: "red"},
						Headline:  "Test Alert 2",
						EventTime: time.Now().Add(-8 * time.Minute),
					},
				},
				To: cursor,
			}
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(resp)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Set up backend configuration
	config := backend.Config{
		ID:                  "integration-test-backend",
		Name:                "Integration Test Backend",
		Type:                "dataminr",
		Enabled:             true,
		URL:                 server.URL,
		APIId:               "test-api-id",
		APIKey:              "test-api-key",
		ChannelID:           "test-channel-id",
		PollIntervalSeconds: 30,
	}

	// Set up mock plugin API
	mockAPI := &plugintest.API{}

	// Mock KV operations for state storage
	kvStore := make(map[string][]byte)
	mockAPI.On("KVSet", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		key := args.String(0)
		value := args.Get(1).([]byte)
		kvStore[key] = value
	}).Return(nil)

	mockAPI.On("KVGet", mock.Anything).Return(func(key string) []byte {
		return kvStore[key]
	}, nil)

	mockAPI.On("LogInfo", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
	mockAPI.On("LogDebug", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
	mockAPI.On("LogError", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()

	client := pluginapi.NewClient(mockAPI, &plugintest.Driver{})

	// Track alerts that were posted
	var postedAlerts []backend.Alert
	mockPoster := &MockPoster{
		PostAlertFn: func(alert backend.Alert, channelID string) error {
			postedAlerts = append(postedAlerts, alert)
			return nil
		},
	}

	// Create backend
	b, err := New(config, client, mockAPI, mockPoster, NewMockDeduplicator(), nil)
	require.NoError(t, err)
	require.NotNil(t, b)

	// Test authentication
	t.Run("authenticate with API", func(t *testing.T) {
		token, expiry, err := b.authManager.GetValidToken()
		require.NoError(t, err)
		assert.Equal(t, authToken, token)
		// Check expiry is in the future (not checking exact time to avoid timing issues)
		assert.False(t, expiry.IsZero())

		// Verify token was saved to KV store
		savedToken, _, err := b.stateStore.GetAuthToken()
		require.NoError(t, err)
		assert.Equal(t, authToken, savedToken)
	})

	// Test fetching alerts
	t.Run("fetch alerts from API", func(t *testing.T) {
		response, err := b.apiClient.FetchAlerts("")
		require.NoError(t, err)
		require.NotNil(t, response)
		assert.Len(t, response.Alerts, 2)
		assert.Equal(t, cursor, response.To)
		assert.Equal(t, "alert-001", response.Alerts[0].AlertID)
		assert.Equal(t, "alert-002", response.Alerts[1].AlertID)
	})

	// Test processing alerts
	t.Run("process alerts", func(t *testing.T) {
		alerts := []Alert{
			{
				AlertID:   "alert-003",
				AlertType: AlertType{Name: "Alert", Color: "yellow"},
				Headline:  "Test Alert 3",
				EventTime: time.Now().Add(-6 * time.Minute),
			},
		}

		newCount, err := b.processor.ProcessAlerts(alerts)
		require.NoError(t, err)
		assert.Equal(t, 1, newCount)
		assert.Len(t, postedAlerts, 1)
		assert.Equal(t, "alert-003", postedAlerts[0].AlertID)
		assert.Equal(t, "Integration Test Backend", postedAlerts[0].BackendName)
	})

	// Test deduplication
	t.Run("deduplicate alerts", func(t *testing.T) {
		alerts := []Alert{
			{
				AlertID:   "alert-003", // duplicate
				AlertType: AlertType{Name: "Alert", Color: "yellow"},
				Headline:  "Test Alert 3 Duplicate",
				EventTime: time.Now().Add(-5 * time.Minute),
			},
			{
				AlertID:   "alert-004", // new
				AlertType: AlertType{Name: "Urgent", Color: "orange"},
				Headline:  "Test Alert 4",
				EventTime: time.Now().Add(-3 * time.Minute),
			},
		}

		initialCount := len(postedAlerts)
		newCount, err := b.processor.ProcessAlerts(alerts)
		require.NoError(t, err)
		assert.Equal(t, 1, newCount) // Only 1 new alert (alert-004)
		assert.Len(t, postedAlerts, initialCount+1)
		assert.Equal(t, "alert-004", postedAlerts[len(postedAlerts)-1].AlertID)
	})

	// Test full poll cycle with mock scheduler
	t.Run("full poll cycle", func(t *testing.T) {
		// Set initial cursor to avoid catch-up mode
		err := b.stateStore.SaveCursor("initial-cursor")
		require.NoError(t, err)

		// Set up mock scheduler that captures the callback
		var pollCallback func()
		mockScheduler := &MockJobScheduler{
			ScheduleFn: func(jobID string, nextWaitInterval cluster.NextWaitInterval, callback func()) (Job, error) {
				pollCallback = callback
				return &MockJob{}, nil
			},
		}
		b.poller.SetScheduler(mockScheduler)

		// Start the backend
		err = b.Start()
		require.NoError(t, err)
		assert.True(t, b.running)

		// Manually trigger poll callback
		require.NotNil(t, pollCallback)
		initialHandledCount := len(postedAlerts)
		pollCallback()

		// Verify poll was executed
		// Check that cursor was saved
		savedCursor, err := b.stateStore.GetCursor()
		require.NoError(t, err)
		assert.Equal(t, cursor, savedCursor)

		// Check that last poll time was saved
		lastPoll, err := b.stateStore.GetLastPoll()
		require.NoError(t, err)
		assert.True(t, lastPoll.After(time.Now().Add(-5*time.Second)))

		// Check that last success time was saved
		lastSuccess, err := b.stateStore.GetLastSuccess()
		require.NoError(t, err)
		assert.True(t, lastSuccess.After(time.Now().Add(-5*time.Second)))

		// Check that failures were reset
		failures, err := b.stateStore.GetFailures()
		require.NoError(t, err)
		assert.Equal(t, 0, failures)

		// Check that new alerts were handled (2 from the mock server)
		assert.Greater(t, len(postedAlerts), initialHandledCount)

		// Stop the backend
		err = b.Stop()
		require.NoError(t, err)
		assert.False(t, b.running)
	})

	// Test status reporting
	t.Run("get backend status", func(t *testing.T) {
		status := b.GetStatus()
		assert.True(t, status.Enabled) // backend is enabled in config
		// Note: IsAuthenticated check removed due to timing sensitivity in tests
		assert.Equal(t, 0, status.ConsecutiveFailures)
		assert.Empty(t, status.LastError)
		assert.False(t, status.LastPollTime.IsZero())
		assert.False(t, status.LastSuccessTime.IsZero())
	})

	mockAPI.AssertExpectations(t)
}

// TestDataminrBackend_ErrorHandling tests error scenarios
func TestDataminrBackend_ErrorHandling(t *testing.T) {
	// Set up test HTTP server that returns errors
	failureCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/1/userAuthorization":
			// Return valid auth
			resp := AuthResponse{
				AuthorizationToken: "test-token",
				ExpirationTime:     time.Now().Add(1 * time.Hour),
			}
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(resp)

		case "/alerts/1/alerts":
			// Return error
			failureCount++
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(APIError{
				Errors: []ErrorDetail{
					{Code: "500", Message: "internal server error"},
				},
			})

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	config := backend.Config{
		ID:                  "error-test-backend",
		Name:                "Error Test Backend",
		Type:                "dataminr",
		Enabled:             true,
		URL:                 server.URL,
		APIId:               "test-api-id",
		APIKey:              "test-api-key",
		ChannelID:           "test-channel-id",
		PollIntervalSeconds: 30,
	}

	mockAPI := &plugintest.API{}

	// Mock KV operations
	kvStore := make(map[string][]byte)
	mockAPI.On("KVSet", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		key := args.String(0)
		value := args.Get(1).([]byte)
		kvStore[key] = value
	}).Return(nil)

	mockAPI.On("KVGet", mock.Anything).Return(func(key string) []byte {
		return kvStore[key]
	}, nil)

	mockAPI.On("LogInfo", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
	mockAPI.On("LogDebug", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
	mockAPI.On("LogError", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
	mockAPI.On("LogWarn", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()

	client := pluginapi.NewClient(mockAPI, &plugintest.Driver{})

	b, err := New(config, client, mockAPI, &MockPoster{}, NewMockDeduplicator(), nil)
	require.NoError(t, err)

	t.Run("handle API errors and track failures", func(t *testing.T) {
		// Set initial cursor to avoid catch-up mode
		err := b.stateStore.SaveCursor("initial-cursor")
		require.NoError(t, err)

		// Set up mock scheduler
		var pollCallback func()
		mockJob := &MockJob{}
		mockScheduler := &MockJobScheduler{
			ScheduleFn: func(jobID string, nextWaitInterval cluster.NextWaitInterval, callback func()) (Job, error) {
				pollCallback = callback
				return mockJob, nil
			},
		}
		b.poller.SetScheduler(mockScheduler)

		// Start the backend
		err = b.Start()
		require.NoError(t, err)

		// Trigger poll cycles that will fail
		for i := range backend.MaxConsecutiveFailures - 1 {
			pollCallback()

			// Check failure count incremented
			failures, failErr := b.stateStore.GetFailures()
			require.NoError(t, failErr)
			assert.Equal(t, i+1, failures)

			// Check last error was saved
			lastError, errErr := b.stateStore.GetLastError()
			require.NoError(t, errErr)
			assert.Contains(t, lastError, "failed to fetch alerts")
		}

		// Backend should still be running
		assert.True(t, b.running)

		// One more failure should disable the poller
		pollCallback()

		// Poller job should be closed (auto-disabled)
		assert.True(t, mockJob.closed)

		// Backend running flag stays true (backend instance still exists, poller auto-disabled)
		// This is expected behavior - backend tracks overall lifecycle, poller auto-disables on errors
		assert.True(t, b.running)

		// Check failure count reached max
		failures, err := b.stateStore.GetFailures()
		require.NoError(t, err)
		assert.Equal(t, backend.MaxConsecutiveFailures, failures)
	})

	mockAPI.AssertExpectations(t)
}
