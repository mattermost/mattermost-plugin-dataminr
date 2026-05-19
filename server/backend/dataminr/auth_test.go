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

func TestAuthManager_GetValidToken_SuccessfulAuthentication(t *testing.T) {
	// Create test server that simulates Dataminr auth endpoint
	expiryTime := time.Now().Add(1 * time.Hour)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/auth/1/userAuthorization", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"))

		// Parse form data
		err := r.ParseForm()
		require.NoError(t, err)

		assert.Equal(t, "api_key", r.PostFormValue("grant_type"))
		assert.Equal(t, "first_alert_api", r.PostFormValue("scope"))
		assert.Equal(t, "test_user", r.PostFormValue("api_user_id"))
		assert.Equal(t, "test_password", r.PostFormValue("api_password"))

		// Return successful auth response
		resp := AuthResponse{
			AuthorizationToken: "test_token_12345",
			TOS:                "https://www.dataminr.com/tos",
			ThirdPartyTerms:    "https://www.dataminr.com/thirdparty",
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// Manually construct JSON with expirationTime in milliseconds
		jsonResp := map[string]any{
			"authorizationToken": resp.AuthorizationToken,
			"expirationTime":     expiryTime.UnixNano() / int64(time.Millisecond),
			"TOS":                resp.TOS,
			"thirdPartyTerms":    resp.ThirdPartyTerms,
		}
		_ = json.NewEncoder(w).Encode(jsonResp)
	}))
	defer server.Close()

	// Mock plugin API
	api := &plugintest.API{}
	api.On("LogInfo", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
	api.On("LogDebug", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
	api.On("LogWarn", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
	api.On("KVGet", "backend_test-backend-id_auth").Return(nil, nil).Once()
	api.On("KVSet", "backend_test-backend-id_auth", mock.Anything).Return(nil).Once()

	// Create logger
	client := pluginapi.NewClient(api, &plugintest.Driver{})

	// Create auth manager
	authManager := NewAuthManager(server.URL, "test_user", "test_password", api, "test-backend-id", client.Log)

	// Get valid token
	token, expiry, err := authManager.GetValidToken()

	// Assertions
	require.NoError(t, err)
	assert.Equal(t, "test_token_12345", token)
	assert.WithinDuration(t, expiryTime, expiry, 1*time.Second)

	api.AssertExpectations(t)
}

func TestAuthManager_GetValidToken_UsesCachedToken(t *testing.T) {
	// Mock plugin API with cached token
	api := &plugintest.API{}
	api.On("LogInfo", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
	api.On("LogDebug", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
	api.On("LogWarn", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()

	futureExpiry := time.Now().Add(30 * time.Minute)
	cachedState := AuthTokenState{
		Token:  "cached_token_67890",
		Expiry: futureExpiry,
	}
	cachedData, _ := json.Marshal(cachedState)

	api.On("KVGet", "backend_test-backend-id_auth").Return(cachedData, nil).Once()

	// Create logger
	client := pluginapi.NewClient(api, &plugintest.Driver{})

	// Create auth manager (server URL doesn't matter - shouldn't be called)
	authManager := NewAuthManager("http://should-not-call", "test_user", "test_password", api, "test-backend-id", client.Log)

	// Get valid token
	token, expiry, err := authManager.GetValidToken()

	// Assertions - should use cached token without making HTTP request
	require.NoError(t, err)
	assert.Equal(t, "cached_token_67890", token)
	assert.WithinDuration(t, futureExpiry, expiry, 1*time.Second)

	api.AssertExpectations(t)
}

func TestAuthManager_GetValidToken_RefreshesExpiringToken(t *testing.T) {
	// Mock plugin API with expiring token
	api := &plugintest.API{}
	api.On("LogInfo", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
	api.On("LogDebug", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
	api.On("LogWarn", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()

	// Token expires in 3 minutes (less than 5 minute buffer)
	expiringExpiry := time.Now().Add(3 * time.Minute)
	cachedState := AuthTokenState{
		Token:  "expiring_token",
		Expiry: expiringExpiry,
	}
	cachedData, _ := json.Marshal(cachedState)

	api.On("KVGet", "backend_test-backend-id_auth").Return(cachedData, nil).Once()

	// Create test server for refresh
	newExpiry := time.Now().Add(1 * time.Hour)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		jsonResp := map[string]any{
			"authorizationToken": "refreshed_token_99999",
			"expirationTime":     newExpiry.UnixNano() / int64(time.Millisecond),
		}
		_ = json.NewEncoder(w).Encode(jsonResp)
	}))
	defer server.Close()

	api.On("KVSet", "backend_test-backend-id_auth", mock.Anything).Return(nil).Once()

	// Create logger
	client := pluginapi.NewClient(api, &plugintest.Driver{})

	// Create auth manager
	authManager := NewAuthManager(server.URL, "test_user", "test_password", api, "test-backend-id", client.Log)

	// Get valid token - should trigger refresh
	token, expiry, err := authManager.GetValidToken()

	// Assertions - should get new token
	require.NoError(t, err)
	assert.Equal(t, "refreshed_token_99999", token)
	assert.WithinDuration(t, newExpiry, expiry, 1*time.Second)

	api.AssertExpectations(t)
}

func TestAuthManager_Authenticate_InvalidCredentials(t *testing.T) {
	// Create test server that returns 401
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)

		authErr := AuthError{
			Error:            "unauthorized_client",
			ErrorDescription: "Invalid API User ID/Password",
		}
		_ = json.NewEncoder(w).Encode(authErr)
	}))
	defer server.Close()

	// Mock plugin API
	api := &plugintest.API{}
	api.On("LogInfo", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
	api.On("LogWarn", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
	api.On("KVGet", "backend_test-backend-id_auth").Return(nil, nil).Once()

	// Create logger
	client := pluginapi.NewClient(api, &plugintest.Driver{})

	// Create auth manager
	authManager := NewAuthManager(server.URL, "bad_user", "bad_password", api, "test-backend-id", client.Log)

	// Get valid token - should fail
	token, expiry, err := authManager.GetValidToken()

	// Assertions
	require.Error(t, err)
	assert.Contains(t, err.Error(), "authentication failed")
	assert.Contains(t, err.Error(), "unauthorized_client")
	assert.Contains(t, err.Error(), "Invalid API User ID/Password")
	assert.Equal(t, "", token)
	assert.True(t, expiry.IsZero())

	api.AssertExpectations(t)
}

func TestAuthManager_Authenticate_ServerError(t *testing.T) {
	// Create test server that returns 500
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	// Mock plugin API
	api := &plugintest.API{}
	api.On("LogInfo", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
	api.On("LogWarn", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
	api.On("KVGet", "backend_test-backend-id_auth").Return(nil, nil).Once()

	// Create logger
	client := pluginapi.NewClient(api, &plugintest.Driver{})

	// Create auth manager
	authManager := NewAuthManager(server.URL, "test_user", "test_password", api, "test-backend-id", client.Log)

	// Get valid token - should fail
	token, expiry, err := authManager.GetValidToken()

	// Assertions
	require.Error(t, err)
	assert.Contains(t, err.Error(), "authentication failed with HTTP 500")
	assert.Equal(t, "", token)
	assert.True(t, expiry.IsZero())

	api.AssertExpectations(t)
}

func TestAuthManager_Authenticate_MissingToken(t *testing.T) {
	// Create test server that returns 200 but no token
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// Empty response
		_ = json.NewEncoder(w).Encode(map[string]any{})
	}))
	defer server.Close()

	// Mock plugin API
	api := &plugintest.API{}
	api.On("LogInfo", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
	api.On("LogWarn", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
	api.On("KVGet", "backend_test-backend-id_auth").Return(nil, nil).Once()

	// Create logger
	client := pluginapi.NewClient(api, &plugintest.Driver{})

	// Create auth manager
	authManager := NewAuthManager(server.URL, "test_user", "test_password", api, "test-backend-id", client.Log)

	// Get valid token - should fail
	token, expiry, err := authManager.GetValidToken()

	// Assertions
	require.Error(t, err)
	assert.Contains(t, err.Error(), "auth response missing token")
	assert.Equal(t, "", token)
	assert.True(t, expiry.IsZero())

	api.AssertExpectations(t)
}

func TestAuthManager_Authenticate_InvalidJSON(t *testing.T) {
	// Create test server that returns invalid JSON
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("invalid json {"))
	}))
	defer server.Close()

	// Mock plugin API
	api := &plugintest.API{}
	api.On("LogInfo", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
	api.On("LogWarn", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
	api.On("KVGet", "backend_test-backend-id_auth").Return(nil, nil).Once()

	// Create logger
	client := pluginapi.NewClient(api, &plugintest.Driver{})

	// Create auth manager
	authManager := NewAuthManager(server.URL, "test_user", "test_password", api, "test-backend-id", client.Log)

	// Get valid token - should fail
	token, expiry, err := authManager.GetValidToken()

	// Assertions
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse auth response")
	assert.Equal(t, "", token)
	assert.True(t, expiry.IsZero())

	api.AssertExpectations(t)
}

func TestAuthManager_ClearCachedToken(t *testing.T) {
	// Mock plugin API
	api := &plugintest.API{}
	api.On("KVSet", "backend_test-backend-id_auth", mock.Anything).Return(nil).Once()

	// Create logger
	logger := pluginapi.LogService{}

	// Create auth manager
	authManager := NewAuthManager("http://test", "test_user", "test_password", api, "test-backend-id", logger)

	// Clear cached token
	err := authManager.ClearCachedToken()

	// Assertions
	require.NoError(t, err)
	api.AssertExpectations(t)
}

func TestAuthManager_isTokenValid(t *testing.T) {
	tests := []struct {
		name     string
		expiry   time.Time
		expected bool
	}{
		{
			name:     "valid token with 30 minutes remaining",
			expiry:   time.Now().Add(30 * time.Minute),
			expected: true,
		},
		{
			name:     "token expiring in 10 minutes (valid)",
			expiry:   time.Now().Add(10 * time.Minute),
			expected: true,
		},
		{
			name:     "token expiring in 4 minutes (should refresh)",
			expiry:   time.Now().Add(4 * time.Minute),
			expected: false,
		},
		{
			name:     "expired token",
			expiry:   time.Now().Add(-10 * time.Minute),
			expected: false,
		},
		{
			name:     "zero time",
			expiry:   time.Time{},
			expected: false,
		},
	}

	// Mock plugin API (not used in this test)
	api := &plugintest.API{}
	logger := pluginapi.LogService{}

	authManager := NewAuthManager("http://test", "test_user", "test_password", api, "test-backend-id", logger)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := authManager.isTokenValid(tt.expiry)
			assert.Equal(t, tt.expected, result)
		})
	}
}
