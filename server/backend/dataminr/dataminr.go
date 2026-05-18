// Copyright (c) 2025-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package dataminr

import (
	"fmt"
	"sync"
	"time"

	"github.com/mattermost/mattermost/server/public/plugin"
	"github.com/mattermost/mattermost/server/public/pluginapi"

	"github.com/mattermost/mattermost-plugin-dataminr/server/backend"
)

// init registers the Dataminr backend factory
func init() {
	backend.RegisterBackendFactory("dataminr", func(config backend.Config, api *pluginapi.Client, papi plugin.API, poster backend.AlertPoster, deduplicator backend.Deduplicator, disableCallback backend.DisableCallback) (backend.Backend, error) {
		return New(config, api, papi, poster, deduplicator, disableCallback)
	})
}

// Backend implements the backend.Backend interface for Dataminr First Alert API
type Backend struct {
	config      backend.Config
	api         *pluginapi.Client
	papi        plugin.API
	poster      backend.AlertPoster
	authManager *AuthManager
	apiClient   *APIClient
	processor   *AlertProcessor
	stateStore  *StateStore
	poller      *Poller
	mu          sync.RWMutex
	running     bool
}

// New creates a new Dataminr backend instance
func New(config backend.Config, api *pluginapi.Client, papi plugin.API, poster backend.AlertPoster, deduplicator backend.Deduplicator, disableCallback backend.DisableCallback) (*Backend, error) {
	// Validate configuration
	if config.Type != "dataminr" {
		return nil, fmt.Errorf("invalid backend type: %s (expected: dataminr)", config.Type)
	}
	if config.ID == "" {
		return nil, fmt.Errorf("backend ID is required")
	}
	if config.URL == "" {
		return nil, fmt.Errorf("backend URL is required")
	}
	if config.APIId == "" {
		return nil, fmt.Errorf("API ID is required")
	}
	if config.APIKey == "" {
		return nil, fmt.Errorf("API key is required")
	}
	if config.ChannelID == "" {
		return nil, fmt.Errorf("channel ID is required")
	}

	// Create state store
	stateStore := NewStateStore(papi, config.ID)

	// Create auth manager
	authManager := NewAuthManager(
		config.URL,
		config.APIId,
		config.APIKey,
		papi,
		config.ID,
		api.Log,
	)

	// Create API client
	apiClient := NewAPIClient(config.URL, authManager, api.Log)

	// Create backend instance
	b := &Backend{
		config:      config,
		api:         api,
		papi:        papi,
		poster:      poster,
		authManager: authManager,
		apiClient:   apiClient,
		stateStore:  stateStore,
		running:     false,
	}

	// Create alert processor with poster, channel ID, and shared deduplicator
	b.processor = NewAlertProcessor(api, config.Type, config.Name, poster, config.ChannelID, deduplicator)

	// Create poller
	pollInterval := time.Duration(config.PollIntervalSeconds) * time.Second
	b.poller = NewPoller(
		api,
		papi,
		config.ID,
		config.Name,
		pollInterval,
		apiClient,
		b.processor,
		stateStore,
		disableCallback,
	)

	return b, nil
}

// Start begins the backend's polling lifecycle
func (b *Backend) Start() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.running {
		return fmt.Errorf("backend already running")
	}

	if !b.config.Enabled {
		return fmt.Errorf("backend is disabled")
	}

	// Reset failure state when starting an enabled backend
	// This ensures a fresh start when re-enabling after failures
	if err := b.stateStore.ResetFailures(); err != nil {
		b.api.Log.Warn("Failed to reset failure count on start", "id", b.config.ID, "error", err.Error())
	}
	if err := b.stateStore.SaveLastError(""); err != nil {
		b.api.Log.Warn("Failed to clear last error on start", "id", b.config.ID, "error", err.Error())
	}

	// Start the poller
	if err := b.poller.Start(); err != nil {
		return fmt.Errorf("failed to start poller: %w", err)
	}

	b.running = true
	b.api.Log.Info("Dataminr backend started", "id", b.config.ID, "name", b.config.Name)
	return nil
}

// Stop gracefully shuts down the backend
func (b *Backend) Stop() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if !b.running {
		return nil
	}

	// Stop the poller
	if err := b.poller.Stop(); err != nil {
		b.api.Log.Error("Failed to stop poller", "id", b.config.ID, "error", err.Error())
		return fmt.Errorf("failed to stop poller: %w", err)
	}

	b.running = false
	b.api.Log.Info("Dataminr backend stopped", "id", b.config.ID, "name", b.config.Name)
	return nil
}

// GetID returns the unique identifier for this backend
func (b *Backend) GetID() string {
	return b.config.ID
}

// GetName returns the display name for this backend
func (b *Backend) GetName() string {
	return b.config.Name
}

// GetType returns the backend type
func (b *Backend) GetType() string {
	return b.config.Type
}

// GetStatus returns the current operational status of the backend
func (b *Backend) GetStatus() backend.Status {
	b.mu.RLock()
	defer b.mu.RUnlock()

	status := backend.Status{
		Enabled: b.config.Enabled,
	}

	// Get last poll time from state
	lastPoll, err := b.stateStore.GetLastPoll()
	if err != nil {
		b.api.Log.Warn("Failed to get last poll time", "id", b.config.ID, "error", err.Error())
	} else {
		status.LastPollTime = lastPoll
	}

	// Get last success time from state
	lastSuccess, err := b.stateStore.GetLastSuccess()
	if err != nil {
		b.api.Log.Warn("Failed to get last success time", "id", b.config.ID, "error", err.Error())
	} else {
		status.LastSuccessTime = lastSuccess
	}

	// Get consecutive failures from state
	failures, err := b.stateStore.GetFailures()
	if err != nil {
		b.api.Log.Warn("Failed to get failure count", "id", b.config.ID, "error", err.Error())
	} else {
		status.ConsecutiveFailures = failures
	}

	// Get last error from state
	lastError, err := b.stateStore.GetLastError()
	if err != nil {
		b.api.Log.Warn("Failed to get last error", "id", b.config.ID, "error", err.Error())
	} else {
		status.LastError = lastError
	}

	// Check authentication status
	token, expiry, err := b.stateStore.GetAuthToken()
	if err != nil {
		b.api.Log.Warn("Failed to check auth token", "id", b.config.ID, "error", err.Error())
		status.IsAuthenticated = false
	} else {
		// Token is valid if it exists and hasn't expired
		status.IsAuthenticated = token != "" && time.Now().Before(expiry)
	}

	return status
}

// ClearOperationalState removes cursor and auth token state while preserving
// failure tracking for status display
func (b *Backend) ClearOperationalState() error {
	return b.stateStore.ClearOperationalState()
}
