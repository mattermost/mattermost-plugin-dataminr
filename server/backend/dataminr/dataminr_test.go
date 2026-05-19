// Copyright (c) 2025-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package dataminr

import (
	"encoding/json"
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

// MockJobScheduler is a mock implementation for testing
type MockJobScheduler struct {
	ScheduleFn func(jobID string, nextWaitInterval cluster.NextWaitInterval, callback func()) (Job, error)
}

// Schedule calls the mock function
func (m *MockJobScheduler) Schedule(
	jobID string,
	nextWaitInterval cluster.NextWaitInterval,
	callback func(),
) (Job, error) {
	if m.ScheduleFn != nil {
		return m.ScheduleFn(jobID, nextWaitInterval, callback)
	}
	return &MockJob{}, nil
}

// MockJob is a mock job implementation for testing
type MockJob struct {
	CloseFn func() error
	closed  bool
}

// Close calls the mock function
func (m *MockJob) Close() error {
	m.closed = true
	if m.CloseFn != nil {
		return m.CloseFn()
	}
	return nil
}

func TestNew(t *testing.T) {
	validConfig := backend.Config{
		ID:                  "test-id-123",
		Name:                "Test Backend",
		Type:                "dataminr",
		Enabled:             true,
		URL:                 "https://api.dataminr.com",
		APIId:               "test-api-id",
		APIKey:              "test-api-key",
		ChannelID:           "channel123",
		PollIntervalSeconds: 30,
	}

	tests := []struct {
		name        string
		config      backend.Config
		wantErr     bool
		errContains string
	}{
		{
			name:    "valid config",
			config:  validConfig,
			wantErr: false,
		},
		{
			name: "invalid type",
			config: backend.Config{
				ID:                  "test-id",
				Name:                "Test",
				Type:                "invalid",
				URL:                 "https://api.dataminr.com",
				APIId:               "id",
				APIKey:              "key",
				ChannelID:           "ch123",
				PollIntervalSeconds: 30,
			},
			wantErr:     true,
			errContains: "invalid backend type",
		},
		{
			name: "missing ID",
			config: backend.Config{
				Name:                "Test",
				Type:                "dataminr",
				URL:                 "https://api.dataminr.com",
				APIId:               "id",
				APIKey:              "key",
				ChannelID:           "ch123",
				PollIntervalSeconds: 30,
			},
			wantErr:     true,
			errContains: "backend ID is required",
		},
		{
			name: "missing URL",
			config: backend.Config{
				ID:                  "test-id",
				Name:                "Test",
				Type:                "dataminr",
				APIId:               "id",
				APIKey:              "key",
				ChannelID:           "ch123",
				PollIntervalSeconds: 30,
			},
			wantErr:     true,
			errContains: "backend URL is required",
		},
		{
			name: "missing API ID",
			config: backend.Config{
				ID:                  "test-id",
				Name:                "Test",
				Type:                "dataminr",
				URL:                 "https://api.dataminr.com",
				APIKey:              "key",
				ChannelID:           "ch123",
				PollIntervalSeconds: 30,
			},
			wantErr:     true,
			errContains: "API ID is required",
		},
		{
			name: "missing API key",
			config: backend.Config{
				ID:                  "test-id",
				Name:                "Test",
				Type:                "dataminr",
				URL:                 "https://api.dataminr.com",
				APIId:               "id",
				ChannelID:           "ch123",
				PollIntervalSeconds: 30,
			},
			wantErr:     true,
			errContains: "API key is required",
		},
		{
			name: "missing channel ID",
			config: backend.Config{
				ID:                  "test-id",
				Name:                "Test",
				Type:                "dataminr",
				URL:                 "https://api.dataminr.com",
				APIId:               "id",
				APIKey:              "key",
				PollIntervalSeconds: 30,
			},
			wantErr:     true,
			errContains: "channel ID is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAPI := &plugintest.API{}
			client := pluginapi.NewClient(mockAPI, &plugintest.Driver{})

			b, err := New(tt.config, client, mockAPI, &MockPoster{}, NewMockDeduplicator(), nil)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				assert.Nil(t, b)
			} else {
				require.NoError(t, err)
				require.NotNil(t, b)
				assert.Equal(t, tt.config.ID, b.GetID())
				assert.Equal(t, tt.config.Name, b.GetName())
				assert.Equal(t, tt.config.Type, b.GetType())
				assert.NotNil(t, b.authManager)
				assert.NotNil(t, b.apiClient)
				assert.NotNil(t, b.processor)
				assert.NotNil(t, b.stateStore)
				assert.NotNil(t, b.poller)
				assert.False(t, b.running)
			}
		})
	}
}

func TestDataminrBackend_Getters(t *testing.T) {
	config := backend.Config{
		ID:                  "backend-123",
		Name:                "Production Alerts",
		Type:                "dataminr",
		Enabled:             true,
		URL:                 "https://api.dataminr.com",
		APIId:               "test-id",
		APIKey:              "test-key",
		ChannelID:           "channel123",
		PollIntervalSeconds: 30,
	}

	mockAPI := &plugintest.API{}
	client := pluginapi.NewClient(mockAPI, &plugintest.Driver{})

	b, err := New(config, client, mockAPI, &MockPoster{}, NewMockDeduplicator(), nil)
	require.NoError(t, err)

	assert.Equal(t, "backend-123", b.GetID())
	assert.Equal(t, "Production Alerts", b.GetName())
	assert.Equal(t, "dataminr", b.GetType())
}

func TestDataminrBackend_Start(t *testing.T) {
	tests := []struct {
		name        string
		enabled     bool
		mockJob     *MockJob
		scheduleErr error
		wantErr     bool
		errContains string
		checkJob    bool
	}{
		{
			name:     "successful start",
			enabled:  true,
			mockJob:  &MockJob{},
			wantErr:  false,
			checkJob: true,
		},
		{
			name:        "backend disabled",
			enabled:     false,
			wantErr:     true,
			errContains: "backend is disabled",
			checkJob:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := backend.Config{
				ID:                  "test-backend",
				Name:                "Test Backend",
				Type:                "dataminr",
				Enabled:             tt.enabled,
				URL:                 "https://api.dataminr.com",
				APIId:               "test-id",
				APIKey:              "test-key",
				ChannelID:           "channel123",
				PollIntervalSeconds: 30,
			}

			mockAPI := &plugintest.API{}
			mockAPI.On("LogInfo", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
			// Mock GetCursor for Start() check (return existing cursor to avoid catch-up)
			mockAPI.On("KVGet", "backend_test-backend_cursor").Return([]byte("existing-cursor"), nil).Maybe()
			// When enabled, expect KVSet calls to reset failure state
			if tt.enabled {
				mockAPI.On("KVSet", "backend_test-backend_failures", []byte("0")).Return(nil)
				mockAPI.On("KVSet", "backend_test-backend_last_error", []byte("")).Return(nil)
			}
			client := pluginapi.NewClient(mockAPI, &plugintest.Driver{})

			b, err := New(config, client, mockAPI, &MockPoster{}, NewMockDeduplicator(), nil)
			require.NoError(t, err)

			// Inject mock scheduler
			mockScheduler := &MockJobScheduler{
				ScheduleFn: func(jobID string, nextWaitInterval cluster.NextWaitInterval, callback func()) (Job, error) {
					if tt.scheduleErr != nil {
						return nil, tt.scheduleErr
					}
					return tt.mockJob, nil
				},
			}
			b.poller.SetScheduler(mockScheduler)

			err = b.Start()

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				assert.False(t, b.running)
			} else {
				require.NoError(t, err)
				assert.True(t, b.running)
			}

			if tt.checkJob && tt.mockJob != nil {
				assert.NotNil(t, b.poller.job)
			}

			mockAPI.AssertExpectations(t)
		})
	}
}

func TestDataminrBackend_StartAlreadyRunning(t *testing.T) {
	config := backend.Config{
		ID:                  "test-backend",
		Name:                "Test Backend",
		Type:                "dataminr",
		Enabled:             true,
		URL:                 "https://api.dataminr.com",
		APIId:               "test-id",
		APIKey:              "test-key",
		ChannelID:           "channel123",
		PollIntervalSeconds: 30,
	}

	mockAPI := &plugintest.API{}
	client := pluginapi.NewClient(mockAPI, &plugintest.Driver{})

	b, err := New(config, client, mockAPI, &MockPoster{}, NewMockDeduplicator(), nil)
	require.NoError(t, err)

	// Manually set running to true
	b.mu.Lock()
	b.running = true
	b.mu.Unlock()

	err = b.Start()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "backend already running")
}

func TestDataminrBackend_StartResetsFailureState(t *testing.T) {
	config := backend.Config{
		ID:                  "test-backend",
		Name:                "Test Backend",
		Type:                "dataminr",
		Enabled:             true,
		URL:                 "https://api.dataminr.com",
		APIId:               "test-id",
		APIKey:              "test-key",
		ChannelID:           "channel123",
		PollIntervalSeconds: 30,
	}

	mockAPI := &plugintest.API{}
	mockAPI.On("LogInfo", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
	// Expect KVSet calls to reset failures and clear error
	mockAPI.On("KVSet", "backend_test-backend_failures", []byte("0")).Return(nil)
	mockAPI.On("KVSet", "backend_test-backend_last_error", []byte("")).Return(nil)
	client := pluginapi.NewClient(mockAPI, &plugintest.Driver{})

	b, err := New(config, client, mockAPI, &MockPoster{}, NewMockDeduplicator(), nil)
	require.NoError(t, err)

	// Inject mock scheduler
	mockJob := &MockJob{}
	mockScheduler := &MockJobScheduler{
		ScheduleFn: func(jobID string, nextWaitInterval cluster.NextWaitInterval, callback func()) (Job, error) {
			return mockJob, nil
		},
	}
	b.poller.SetScheduler(mockScheduler)

	err = b.Start()
	require.NoError(t, err)
	assert.True(t, b.running)

	// Verify that failure state was reset
	mockAPI.AssertExpectations(t)
}

func TestDataminrBackend_Stop(t *testing.T) {
	config := backend.Config{
		ID:                  "test-backend",
		Name:                "Test Backend",
		Type:                "dataminr",
		Enabled:             true,
		URL:                 "https://api.dataminr.com",
		APIId:               "test-id",
		APIKey:              "test-key",
		ChannelID:           "channel123",
		PollIntervalSeconds: 30,
	}

	t.Run("stop when not running", func(t *testing.T) {
		mockAPI := &plugintest.API{}
		client := pluginapi.NewClient(mockAPI, &plugintest.Driver{})

		b, err := New(config, client, mockAPI, &MockPoster{}, NewMockDeduplicator(), nil)
		require.NoError(t, err)

		err = b.Stop()
		assert.NoError(t, err)
		assert.False(t, b.running)
	})

	t.Run("stop when running", func(t *testing.T) {
		mockAPI := &plugintest.API{}
		mockAPI.On("LogInfo", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
		// Expect KVSet calls when Start resets failure state
		mockAPI.On("KVSet", "backend_test-backend_failures", []byte("0")).Return(nil)
		mockAPI.On("KVSet", "backend_test-backend_last_error", []byte("")).Return(nil)
		client := pluginapi.NewClient(mockAPI, &plugintest.Driver{})

		b, err := New(config, client, mockAPI, &MockPoster{}, NewMockDeduplicator(), nil)
		require.NoError(t, err)

		// Set up mock job
		mockJob := &MockJob{}
		mockScheduler := &MockJobScheduler{
			ScheduleFn: func(jobID string, nextWaitInterval cluster.NextWaitInterval, callback func()) (Job, error) {
				return mockJob, nil
			},
		}
		b.poller.SetScheduler(mockScheduler)

		// Start the backend
		err = b.Start()
		require.NoError(t, err)
		assert.True(t, b.running)

		// Stop the backend
		err = b.Stop()
		assert.NoError(t, err)
		assert.False(t, b.running)
		assert.True(t, mockJob.closed)

		mockAPI.AssertExpectations(t)
	})
}

func TestDataminrBackend_GetStatus(t *testing.T) {
	config := backend.Config{
		ID:                  "test-backend",
		Name:                "Test Backend",
		Type:                "dataminr",
		Enabled:             true,
		URL:                 "https://api.dataminr.com",
		APIId:               "test-id",
		APIKey:              "test-key",
		ChannelID:           "channel123",
		PollIntervalSeconds: 30,
	}

	t.Run("status with no state", func(t *testing.T) {
		mockAPI := &plugintest.API{}
		// All KVGet calls return nil (no data)
		mockAPI.On("KVGet", mock.Anything).Return(nil, nil)

		client := pluginapi.NewClient(mockAPI, &plugintest.Driver{})

		b, err := New(config, client, mockAPI, &MockPoster{}, NewMockDeduplicator(), nil)
		require.NoError(t, err)

		status := b.GetStatus()

		assert.True(t, status.Enabled) // backend is enabled in config
		assert.True(t, status.LastPollTime.IsZero())
		assert.True(t, status.LastSuccessTime.IsZero())
		assert.Equal(t, 0, status.ConsecutiveFailures)
		assert.False(t, status.IsAuthenticated)
		assert.Empty(t, status.LastError)

		mockAPI.AssertExpectations(t)
	})

	t.Run("status with existing state and running", func(t *testing.T) {
		mockAPI := &plugintest.API{}
		now := time.Now()
		lastPoll := now.Add(-1 * time.Minute)
		lastSuccess := now.Add(-2 * time.Minute)
		tokenExpiry := now.Add(30 * time.Minute)

		// Mock KVGet responses
		mockAPI.On("KVGet", "backend_test-backend_last_poll").Return(mustMarshalTime(lastPoll), nil)
		mockAPI.On("KVGet", "backend_test-backend_last_success").Return(mustMarshalTime(lastSuccess), nil)
		mockAPI.On("KVGet", "backend_test-backend_failures").Return([]byte(`3`), nil)
		mockAPI.On("KVGet", "backend_test-backend_last_error").Return([]byte("rate limit exceeded"), nil)
		mockAPI.On("KVGet", "backend_test-backend_auth").Return(mustMarshalAuthToken("test-token", tokenExpiry), nil)

		client := pluginapi.NewClient(mockAPI, &plugintest.Driver{})

		b, err := New(config, client, mockAPI, &MockPoster{}, NewMockDeduplicator(), nil)
		require.NoError(t, err)

		// Set backend as running
		b.mu.Lock()
		b.running = true
		b.mu.Unlock()

		status := b.GetStatus()

		assert.True(t, status.Enabled)
		assert.Equal(t, lastPoll.Unix(), status.LastPollTime.Unix())
		assert.Equal(t, lastSuccess.Unix(), status.LastSuccessTime.Unix())
		assert.Equal(t, 3, status.ConsecutiveFailures)
		assert.True(t, status.IsAuthenticated)
		assert.Equal(t, "rate limit exceeded", status.LastError)

		mockAPI.AssertExpectations(t)
	})

	t.Run("status with expired token", func(t *testing.T) {
		mockAPI := &plugintest.API{}
		now := time.Now()
		tokenExpiry := now.Add(-10 * time.Minute) // expired

		mockAPI.On("KVGet", "backend_test-backend_last_poll").Return(nil, nil)
		mockAPI.On("KVGet", "backend_test-backend_last_success").Return(nil, nil)
		mockAPI.On("KVGet", "backend_test-backend_failures").Return(nil, nil)
		mockAPI.On("KVGet", "backend_test-backend_last_error").Return(nil, nil)
		mockAPI.On("KVGet", "backend_test-backend_auth").Return(mustMarshalAuthToken("expired-token", tokenExpiry), nil)

		client := pluginapi.NewClient(mockAPI, &plugintest.Driver{})

		b, err := New(config, client, mockAPI, &MockPoster{}, NewMockDeduplicator(), nil)
		require.NoError(t, err)

		status := b.GetStatus()

		assert.False(t, status.IsAuthenticated)

		mockAPI.AssertExpectations(t)
	})
}

// Helper functions for marshaling test data
func mustMarshalTime(t time.Time) []byte {
	data, err := json.Marshal(t)
	if err != nil {
		panic(err)
	}
	return data
}

func mustMarshalAuthToken(token string, expiry time.Time) []byte {
	state := AuthTokenState{
		Token:  token,
		Expiry: expiry,
	}
	data, err := json.Marshal(state)
	if err != nil {
		panic(err)
	}
	return data
}
