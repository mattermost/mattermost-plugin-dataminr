// Copyright (c) 2025-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package dataminr

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/mattermost/mattermost/server/public/plugin"
	"github.com/mattermost/mattermost/server/public/pluginapi"

	"github.com/mattermost/mattermost-plugin-dataminr/server/backend"
)

// AuthManager handles authentication with the Dataminr First Alert API
// including token acquisition, caching, and proactive refresh
type AuthManager struct {
	baseURL     string
	apiUserID   string
	apiPassword string
	httpClient  *http.Client
	stateStore  *StateStore
	logger      pluginapi.LogService
}

// NewAuthManager creates a new authentication manager
func NewAuthManager(baseURL, apiUserID, apiPassword string, api plugin.API, backendID string, logger pluginapi.LogService) *AuthManager {
	return &AuthManager{
		baseURL:     baseURL,
		apiUserID:   apiUserID,
		apiPassword: apiPassword,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		stateStore: NewStateStore(api, backendID),
		logger:     logger,
	}
}

// GetValidToken returns a valid authentication token, refreshing if necessary
// Returns the token string and expiry time, or an error if authentication fails
func (a *AuthManager) GetValidToken() (string, time.Time, error) {
	cachedToken, cachedExpiry, err := a.stateStore.GetAuthToken()
	if err != nil {
		a.logger.Warn("Failed to load cached auth token", "error", err)
	}

	if cachedToken != "" && a.isTokenValid(cachedExpiry) {
		a.logger.Debug("Using cached authentication token")
		return cachedToken, cachedExpiry, nil
	}

	a.logger.Info("Acquiring new authentication token")
	return a.authenticate()
}

// authenticate performs the authentication flow with Dataminr API
func (a *AuthManager) authenticate() (string, time.Time, error) {
	authURL := fmt.Sprintf("%s/auth/1/userAuthorization", a.baseURL)

	formData := url.Values{}
	formData.Set("grant_type", "api_key")
	formData.Set("scope", "first_alert_api")
	formData.Set("api_user_id", a.apiUserID)
	formData.Set("api_password", a.apiPassword)

	req, err := http.NewRequest(http.MethodPost, authURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to create auth request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("authentication request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		var authErr AuthError
		if err := json.NewDecoder(resp.Body).Decode(&authErr); err == nil && authErr.Error != "" {
			return "", time.Time{}, fmt.Errorf("authentication failed (HTTP %d): %s - %s", resp.StatusCode, authErr.Error, authErr.ErrorDescription)
		}
		return "", time.Time{}, fmt.Errorf("authentication failed with HTTP %d", resp.StatusCode)
	}

	var authResp AuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return "", time.Time{}, fmt.Errorf("failed to parse auth response: %w", err)
	}

	if authResp.AuthorizationToken == "" {
		return "", time.Time{}, fmt.Errorf("auth response missing token")
	}

	if err := a.stateStore.SaveAuthToken(authResp.AuthorizationToken, authResp.ExpirationTime); err != nil {
		a.logger.Warn("Failed to save auth token to state store", "error", err)
	}

	a.logger.Info("Successfully authenticated with Dataminr", "expiry", authResp.ExpirationTime.Format(time.RFC3339))
	return authResp.AuthorizationToken, authResp.ExpirationTime, nil
}

// isTokenValid checks if a token is valid and not expiring soon
// Returns true if the token has more than AuthTokenRefreshBuffer (5 minutes) remaining
func (a *AuthManager) isTokenValid(expiry time.Time) bool {
	if expiry.IsZero() {
		return false
	}

	timeUntilExpiry := time.Until(expiry)
	return timeUntilExpiry > backend.AuthTokenRefreshBuffer
}

// ClearCachedToken removes the cached authentication token from the state store
// Useful for testing or forcing re-authentication
func (a *AuthManager) ClearCachedToken() error {
	return a.stateStore.SaveAuthToken("", time.Time{})
}
