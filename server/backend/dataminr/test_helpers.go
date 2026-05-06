package dataminr

import (
	"github.com/mattermost/mattermost-plugin-dataminr/server/backend"
)

// MockPoster is a mock poster implementation for testing
type MockPoster struct {
	PostAlertFn func(alert backend.Alert, channelID string) error
}

// PostAlert calls the mock function
func (m *MockPoster) PostAlert(alert backend.Alert, channelID string) error {
	if m.PostAlertFn != nil {
		return m.PostAlertFn(alert, channelID)
	}
	return nil
}

// MockDeduplicator is a mock implementation of backend.Deduplicator for testing
type MockDeduplicator struct {
	RecordAlertFn func(backendType, alertID, channelID string) bool
	seenAlerts    map[string]bool
}

// NewMockDeduplicator creates a new mock deduplicator with default behavior
func NewMockDeduplicator() *MockDeduplicator {
	return &MockDeduplicator{
		seenAlerts: make(map[string]bool),
	}
}

// RecordAlert calls the mock function or uses default behavior
func (m *MockDeduplicator) RecordAlert(backendType, alertID, channelID string) bool {
	if m.RecordAlertFn != nil {
		return m.RecordAlertFn(backendType, alertID, channelID)
	}
	// Default behavior: track in-memory
	key := backendType + ":" + alertID + ":" + channelID
	if _, exists := m.seenAlerts[key]; exists {
		return false
	}
	m.seenAlerts[key] = true
	return true
}
