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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// createTestServerWithAuth creates a test server that handles authentication and custom alerts handling
// The alertsHandler is called for GET requests to /alerts/1/alerts
func createTestServerWithAuth(alertsHandler http.HandlerFunc) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handle auth endpoint
		if r.URL.Path == "/auth/1/userAuthorization" && r.Method == http.MethodPost {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"authorizationToken": "test-token",
				"expirationTime":     time.Now().Add(1 * time.Hour).UnixMilli(),
			})
			return
		}

		// Handle alerts endpoint with custom handler
		if r.URL.Path == "/alerts/1/alerts" && r.Method == http.MethodGet {
			alertsHandler(w, r)
			return
		}

		// Unknown endpoint
		w.WriteHeader(http.StatusNotFound)
	}))
}

func TestAPIClient_FetchAlerts_Success(t *testing.T) {
	// Mock alerts response
	mockResponse := AlertsResponse{
		Alerts: []Alert{
			{
				AlertID:   "alert-1",
				Headline:  "Test Alert 1",
				EventTime: time.Now().UTC(),
				AlertType: AlertType{Name: "Flash", Color: "red"},
			},
			{
				AlertID:   "alert-2",
				Headline:  "Test Alert 2",
				EventTime: time.Now().UTC(),
				AlertType: AlertType{Name: "Urgent", Color: "orange"},
			},
		},
		To: "cursor-123",
	}

	// Create test server with auth handling
	server := createTestServerWithAuth(func(w http.ResponseWriter, r *http.Request) {
		// Verify query parameters
		assert.Equal(t, "19", r.URL.Query().Get("alertversion"))

		// Verify authorization header
		authHeader := r.Header.Get("Authorization")
		assert.Contains(t, authHeader, "Dmauth")
		assert.Contains(t, authHeader, "test-token")

		// Return mock response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(mockResponse)
	})
	defer server.Close()

	// Create mock plugin API
	api := &plugintest.API{}
	api.On("LogDebug", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
	api.On("LogInfo", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
	api.On("KVGet", mock.Anything).Return(nil, nil).Maybe()
	api.On("KVSet", mock.Anything, mock.Anything).Return(nil).Maybe()

	// Create logger
	client := pluginapi.NewClient(api, &plugintest.Driver{})

	// Create auth manager (with cached token to avoid auth call)
	authManager := NewAuthManager(server.URL, "test-user", "test-pass", api, "test-backend", client.Log)

	// Create API client
	apiClient := NewAPIClient(server.URL, authManager, client.Log)

	// Test fetching without cursor
	resp, err := apiClient.FetchAlerts("")

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Len(t, resp.Alerts, 2)
	assert.Equal(t, "cursor-123", resp.To)
	assert.Equal(t, "alert-1", resp.Alerts[0].AlertID)
	assert.Equal(t, "alert-2", resp.Alerts[1].AlertID)
}

func TestAPIClient_FetchAlerts_WithCursor(t *testing.T) {
	// Create test server with auth handling
	server := createTestServerWithAuth(func(w http.ResponseWriter, r *http.Request) {
		// Verify cursor parameter is passed
		cursor := r.URL.Query().Get("from")
		assert.Equal(t, "previous-cursor", cursor)

		// Return mock response
		mockResponse := AlertsResponse{
			Alerts: []Alert{},
			To:     "new-cursor",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(mockResponse)
	})
	defer server.Close()

	// Create mock plugin API
	api := &plugintest.API{}
	api.On("LogDebug", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
	api.On("LogInfo", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
	api.On("KVGet", mock.Anything).Return(nil, nil).Maybe()
	api.On("KVSet", mock.Anything, mock.Anything).Return(nil).Maybe()

	// Create logger
	client := pluginapi.NewClient(api, &plugintest.Driver{})

	// Create auth manager and API client
	authManager := NewAuthManager(server.URL, "test-user", "test-pass", api, "test-backend", client.Log)
	apiClient := NewAPIClient(server.URL, authManager, client.Log)

	// Test fetching with cursor
	resp, err := apiClient.FetchAlerts("previous-cursor")

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "new-cursor", resp.To)
}

func TestAPIClient_FetchAlerts_Unauthorized(t *testing.T) {
	// Create test server with auth handling
	server := createTestServerWithAuth(func(w http.ResponseWriter, r *http.Request) {
		apiErr := APIError{
			Errors: []ErrorDetail{
				{Code: "103", Message: "Authentication error. Invalid token"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(apiErr)
	})
	defer server.Close()

	// Create mock plugin API
	api := &plugintest.API{}
	api.On("LogDebug", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
	api.On("LogInfo", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
	api.On("KVGet", mock.Anything).Return(nil, nil).Maybe()
	api.On("KVSet", mock.Anything, mock.Anything).Return(nil).Maybe()

	// Create logger
	client := pluginapi.NewClient(api, &plugintest.Driver{})

	// Create auth manager and API client
	authManager := NewAuthManager(server.URL, "test-user", "test-pass", api, "test-backend", client.Log)
	apiClient := NewAPIClient(server.URL, authManager, client.Log)

	// Test fetching - should return 401 error
	resp, err := apiClient.FetchAlerts("")

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "authentication error")
	assert.Contains(t, err.Error(), "401")
}

func TestAPIClient_FetchAlerts_RateLimitExceeded(t *testing.T) {
	// Create test server with auth handling
	server := createTestServerWithAuth(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	})
	defer server.Close()

	// Create mock plugin API
	api := &plugintest.API{}
	api.On("LogDebug", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
	api.On("LogInfo", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
	api.On("KVGet", mock.Anything).Return(nil, nil).Maybe()
	api.On("KVSet", mock.Anything, mock.Anything).Return(nil).Maybe()

	// Create logger
	client := pluginapi.NewClient(api, &plugintest.Driver{})

	// Create auth manager and API client
	authManager := NewAuthManager(server.URL, "test-user", "test-pass", api, "test-backend", client.Log)
	apiClient := NewAPIClient(server.URL, authManager, client.Log)

	// Test fetching - should return 429 error
	resp, err := apiClient.FetchAlerts("")

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "rate limit exceeded")
	assert.Contains(t, err.Error(), "429")
}

func TestAPIClient_FetchAlerts_ServerError(t *testing.T) {
	// Create test server with auth handling
	server := createTestServerWithAuth(func(w http.ResponseWriter, r *http.Request) {
		apiErr := APIError{
			Errors: []ErrorDetail{
				{Code: "500", Message: "Internal server error"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(apiErr)
	})
	defer server.Close()

	// Create mock plugin API
	api := &plugintest.API{}
	api.On("LogDebug", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
	api.On("LogInfo", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
	api.On("KVGet", mock.Anything).Return(nil, nil).Maybe()
	api.On("KVSet", mock.Anything, mock.Anything).Return(nil).Maybe()

	// Create logger
	client := pluginapi.NewClient(api, &plugintest.Driver{})

	// Create auth manager and API client
	authManager := NewAuthManager(server.URL, "test-user", "test-pass", api, "test-backend", client.Log)
	apiClient := NewAPIClient(server.URL, authManager, client.Log)

	// Test fetching - should return 500 error
	resp, err := apiClient.FetchAlerts("")

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "server error")
	assert.Contains(t, err.Error(), "500")
}

func TestAPIClient_FetchAlerts_BadRequest(t *testing.T) {
	// Create test server with auth handling
	server := createTestServerWithAuth(func(w http.ResponseWriter, r *http.Request) {
		apiErr := APIError{
			Errors: []ErrorDetail{
				{Code: "400", Message: "Invalid request parameters"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(apiErr)
	})
	defer server.Close()

	// Create mock plugin API
	api := &plugintest.API{}
	api.On("LogDebug", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
	api.On("LogInfo", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
	api.On("KVGet", mock.Anything).Return(nil, nil).Maybe()
	api.On("KVSet", mock.Anything, mock.Anything).Return(nil).Maybe()

	// Create logger
	client := pluginapi.NewClient(api, &plugintest.Driver{})

	// Create auth manager and API client
	authManager := NewAuthManager(server.URL, "test-user", "test-pass", api, "test-backend", client.Log)
	apiClient := NewAPIClient(server.URL, authManager, client.Log)

	// Test fetching - should return 400 error
	resp, err := apiClient.FetchAlerts("")

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "bad request")
	assert.Contains(t, err.Error(), "400")
}

func TestAPIClient_FetchAlerts_InvalidJSON(t *testing.T) {
	// Create test server with auth handling
	server := createTestServerWithAuth(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("invalid json"))
	})
	defer server.Close()

	// Create mock plugin API
	api := &plugintest.API{}
	api.On("LogDebug", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
	api.On("LogInfo", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
	api.On("KVGet", mock.Anything).Return(nil, nil).Maybe()
	api.On("KVSet", mock.Anything, mock.Anything).Return(nil).Maybe()

	// Create logger
	client := pluginapi.NewClient(api, &plugintest.Driver{})

	// Create auth manager and API client
	authManager := NewAuthManager(server.URL, "test-user", "test-pass", api, "test-backend", client.Log)
	apiClient := NewAPIClient(server.URL, authManager, client.Log)

	// Test fetching - should return parse error
	resp, err := apiClient.FetchAlerts("")

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "failed to parse")
}

func TestAPIClient_FetchAlerts_AuthenticationFailure(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Auth endpoint returns 401
		if r.URL.Path == "/auth/1/userAuthorization" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		// Alerts endpoint should not be reached
		t.Error("Should not reach alerts endpoint when auth fails")
	}))
	defer server.Close()

	// Create mock plugin API
	api := &plugintest.API{}
	api.On("LogDebug", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
	api.On("LogInfo", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
	api.On("LogWarn", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
	api.On("KVGet", mock.Anything).Return(nil, nil).Maybe()

	// Create logger
	client := pluginapi.NewClient(api, &plugintest.Driver{})

	// Create auth manager with no cached token
	authManager := NewAuthManager(server.URL, "test-user", "test-pass", api, "test-backend", client.Log)
	apiClient := NewAPIClient(server.URL, authManager, client.Log)

	// Test fetching - should fail during authentication
	resp, err := apiClient.FetchAlerts("")

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "failed to get auth token")
}

func TestAPIClient_FetchAlerts_EmptyAlerts(t *testing.T) {
	// Mock empty response (no new alerts)
	mockResponse := AlertsResponse{
		Alerts: []Alert{},
		To:     "cursor-456",
	}

	// Create test server with auth handling
	server := createTestServerWithAuth(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(mockResponse)
	})
	defer server.Close()

	// Create mock plugin API
	api := &plugintest.API{}
	api.On("LogDebug", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
	api.On("LogInfo", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
	api.On("KVGet", mock.Anything).Return(nil, nil).Maybe()
	api.On("KVSet", mock.Anything, mock.Anything).Return(nil).Maybe()

	// Create logger
	client := pluginapi.NewClient(api, &plugintest.Driver{})

	// Create auth manager and API client
	authManager := NewAuthManager(server.URL, "test-user", "test-pass", api, "test-backend", client.Log)
	apiClient := NewAPIClient(server.URL, authManager, client.Log)

	// Test fetching - should succeed with empty alerts
	resp, err := apiClient.FetchAlerts("cursor-123")

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Empty(t, resp.Alerts)
	assert.Equal(t, "cursor-456", resp.To)
}
