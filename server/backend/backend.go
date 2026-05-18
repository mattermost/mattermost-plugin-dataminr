// Copyright (c) 2025-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package backend

import "time"

// Config represents the configuration for a backend instance.
// Each backend is uniquely identified by its ID (UUID v4).
type Config struct {
	// ID is the unique stable identifier for this backend (UUID v4, immutable)
	ID string `json:"id"`

	// Name is the display name for this backend (mutable, must be unique)
	Name string `json:"name"`

	// Type is the backend type (e.g., "dataminr")
	Type string `json:"type"`

	// Enabled indicates whether this backend should be actively polling
	Enabled bool `json:"enabled"`

	// URL is the base API URL for this backend
	URL string `json:"url"`

	// APIId is the API user ID or client ID (backend-specific)
	APIId string `json:"apiId"`

	// APIKey is the API key or password (backend-specific)
	APIKey string `json:"apiKey"`

	// ChannelID is the Mattermost channel ID to post alerts to
	ChannelID string `json:"channelId"`

	// PollIntervalSeconds is how often to poll this backend (minimum: MinPollIntervalSeconds)
	PollIntervalSeconds int `json:"pollIntervalSeconds"`
}

// Status represents the current operational status of a backend instance.
type Status struct {
	// Enabled indicates whether the backend is enabled in configuration
	Enabled bool `json:"enabled"`

	// LastPollTime is the timestamp of the last poll attempt
	LastPollTime time.Time `json:"lastPollTime"`

	// LastSuccessTime is the timestamp of the last successful poll
	LastSuccessTime time.Time `json:"lastSuccessTime"`

	// ConsecutiveFailures is the count of consecutive polling failures
	ConsecutiveFailures int `json:"consecutiveFailures"`

	// IsAuthenticated indicates whether the backend has a valid auth token
	IsAuthenticated bool `json:"isAuthenticated"`

	// LastError contains the error message from the most recent failure (empty if no error)
	LastError string `json:"lastError"`
}
