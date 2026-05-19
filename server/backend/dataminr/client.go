// Copyright (c) 2025-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package dataminr

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/mattermost/mattermost/server/public/pluginapi"
)

// APIClient handles communication with the Dataminr First Alert API
// for fetching alerts using cursor-based pagination
type APIClient struct {
	baseURL     string
	httpClient  *http.Client
	authManager *AuthManager
	logger      pluginapi.LogService
}

// NewAPIClient creates a new API client
func NewAPIClient(baseURL string, authManager *AuthManager, logger pluginapi.LogService) *APIClient {
	return &APIClient{
		baseURL:     baseURL,
		authManager: authManager,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

// FetchAlerts polls the Dataminr alerts endpoint with cursor-based pagination
// Returns the alerts response containing alerts array and new cursor, or an error
func (c *APIClient) FetchAlerts(cursor string) (*AlertsResponse, error) {
	// Get valid authentication token
	token, _, err := c.authManager.GetValidToken()
	if err != nil {
		return nil, fmt.Errorf("failed to get auth token: %w", err)
	}

	// Build request URL with hardcoded alertversion=19
	alertsURL := fmt.Sprintf("%s/alerts/1/alerts?alertversion=19", c.baseURL)
	if cursor != "" {
		alertsURL += fmt.Sprintf("&from=%s", url.QueryEscape(cursor))
	}

	req, err := http.NewRequest(http.MethodGet, alertsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create alerts request: %w", err)
	}

	// Set Dmauth authorization header (NOT Bearer)
	req.Header.Set("Authorization", fmt.Sprintf("Dmauth %s", token))
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("alerts request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Handle various HTTP error responses
	switch resp.StatusCode {
	case http.StatusOK:
		// Success - parse response below
	case http.StatusUnauthorized:
		// 401 - Token expired or invalid, suggest re-authentication
		var apiErr APIError
		if err := json.NewDecoder(resp.Body).Decode(&apiErr); err == nil {
			return nil, fmt.Errorf("authentication error (HTTP 401): %s", apiErr.Error())
		}
		return nil, fmt.Errorf("authentication error (HTTP 401): token invalid or expired")
	case http.StatusTooManyRequests:
		// 429 - Rate limit exceeded
		return nil, fmt.Errorf("rate limit exceeded (HTTP 429): too many requests")
	case http.StatusInternalServerError:
		// 500 - Server error
		var apiErr APIError
		if err := json.NewDecoder(resp.Body).Decode(&apiErr); err == nil {
			return nil, fmt.Errorf("server error (HTTP 500): %s", apiErr.Error())
		}
		return nil, fmt.Errorf("server error (HTTP 500): Dataminr API internal error")
	case http.StatusBadRequest:
		// 400 - Bad request (configuration issue)
		var apiErr APIError
		if err := json.NewDecoder(resp.Body).Decode(&apiErr); err == nil {
			return nil, fmt.Errorf("bad request (HTTP 400): %s", apiErr.Error())
		}
		return nil, fmt.Errorf("bad request (HTTP 400): invalid request parameters")
	default:
		// Other errors
		return nil, fmt.Errorf("unexpected HTTP status %d", resp.StatusCode)
	}

	// Parse successful response
	var alertsResp AlertsResponse
	if err := json.NewDecoder(resp.Body).Decode(&alertsResp); err != nil {
		return nil, fmt.Errorf("failed to parse alerts response: %w", err)
	}

	c.logger.Debug("Successfully fetched alerts",
		"alertCount", len(alertsResp.Alerts),
		"cursor", cursor,
		"newCursor", alertsResp.To)

	return &alertsResp, nil
}
