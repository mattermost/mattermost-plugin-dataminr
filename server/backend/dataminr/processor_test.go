// Copyright (c) 2025-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package dataminr

import (
	"errors"
	"testing"
	"time"

	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/mattermost/mattermost/server/public/pluginapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/mattermost/mattermost-plugin-dataminr/server/backend"
)

func TestAlertProcessor_ProcessAlerts(t *testing.T) {
	eventTime := time.Now().UTC()

	t.Run("processes new alerts successfully", func(t *testing.T) {
		api := plugintest.NewAPI(t)
		api.On("LogDebug", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
		client := pluginapi.NewClient(api, &plugintest.Driver{})

		postedAlerts := []backend.Alert{}
		mockPoster := &MockPoster{
			PostAlertFn: func(alert backend.Alert, channelID string) error {
				postedAlerts = append(postedAlerts, alert)
				return nil
			},
		}

		mockDedup := NewMockDeduplicator()
		processor := NewAlertProcessor(client, "dataminr", "Test Backend", mockPoster, "test-channel-id", mockDedup)

		alerts := []Alert{
			{
				AlertID:       "alert-1",
				AlertType:     AlertType{Name: "Flash"},
				EventTime:     eventTime,
				Headline:      "Test Alert 1",
				FirstAlertURL: "https://example.com/alert/1",
			},
			{
				AlertID:       "alert-2",
				AlertType:     AlertType{Name: "Urgent"},
				EventTime:     eventTime,
				Headline:      "Test Alert 2",
				FirstAlertURL: "https://example.com/alert/2",
			},
		}

		count, err := processor.ProcessAlerts(alerts)

		assert.NoError(t, err)
		assert.Equal(t, 2, count)
		assert.Len(t, postedAlerts, 2)
		assert.Equal(t, "alert-1", postedAlerts[0].AlertID)
		assert.Equal(t, "alert-2", postedAlerts[1].AlertID)
	})

	t.Run("skips duplicate alerts", func(t *testing.T) {
		api := plugintest.NewAPI(t)
		api.On("LogDebug", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
		client := pluginapi.NewClient(api, &plugintest.Driver{})

		postedAlerts := []backend.Alert{}
		mockPoster := &MockPoster{
			PostAlertFn: func(alert backend.Alert, channelID string) error {
				postedAlerts = append(postedAlerts, alert)
				return nil
			},
		}

		mockDedup := NewMockDeduplicator()
		processor := NewAlertProcessor(client, "dataminr", "Test Backend", mockPoster, "test-channel-id", mockDedup)

		alerts := []Alert{
			{
				AlertID:       "alert-1",
				AlertType:     AlertType{Name: "Flash"},
				EventTime:     eventTime,
				Headline:      "Test Alert 1",
				FirstAlertURL: "https://example.com/alert/1",
			},
			{
				AlertID:       "alert-1", // Duplicate
				AlertType:     AlertType{Name: "Flash"},
				EventTime:     eventTime,
				Headline:      "Test Alert 1",
				FirstAlertURL: "https://example.com/alert/1",
			},
		}

		count, err := processor.ProcessAlerts(alerts)

		assert.NoError(t, err)
		assert.Equal(t, 1, count)
		assert.Len(t, postedAlerts, 1)
	})

	t.Run("skips duplicates across batches", func(t *testing.T) {
		api := plugintest.NewAPI(t)
		api.On("LogDebug", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
		client := pluginapi.NewClient(api, &plugintest.Driver{})

		postedAlerts := []backend.Alert{}
		mockPoster := &MockPoster{
			PostAlertFn: func(alert backend.Alert, channelID string) error {
				postedAlerts = append(postedAlerts, alert)
				return nil
			},
		}

		mockDedup := NewMockDeduplicator()
		processor := NewAlertProcessor(client, "dataminr", "Test Backend", mockPoster, "test-channel-id", mockDedup)

		// First batch
		batch1 := []Alert{
			{
				AlertID:       "alert-1",
				AlertType:     AlertType{Name: "Flash"},
				EventTime:     eventTime,
				Headline:      "Test Alert 1",
				FirstAlertURL: "https://example.com/alert/1",
			},
		}

		count1, err := processor.ProcessAlerts(batch1)
		assert.NoError(t, err)
		assert.Equal(t, 1, count1)

		// Second batch with duplicate
		batch2 := []Alert{
			{
				AlertID:       "alert-1", // Duplicate from batch 1
				AlertType:     AlertType{Name: "Flash"},
				EventTime:     eventTime,
				Headline:      "Test Alert 1",
				FirstAlertURL: "https://example.com/alert/1",
			},
			{
				AlertID:       "alert-2", // New alert
				AlertType:     AlertType{Name: "Urgent"},
				EventTime:     eventTime,
				Headline:      "Test Alert 2",
				FirstAlertURL: "https://example.com/alert/2",
			},
		}

		count2, err := processor.ProcessAlerts(batch2)
		assert.NoError(t, err)
		assert.Equal(t, 1, count2) // Only alert-2 should be processed

		assert.Len(t, postedAlerts, 2)
		assert.Equal(t, "alert-1", postedAlerts[0].AlertID)
		assert.Equal(t, "alert-2", postedAlerts[1].AlertID)
	})

	t.Run("continues processing on handler error", func(t *testing.T) {
		api := plugintest.NewAPI(t)
		api.On("LogDebug", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
		api.On("LogError", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
		client := pluginapi.NewClient(api, &plugintest.Driver{})

		postedAlerts := []backend.Alert{}
		callCount := 0
		mockPoster := &MockPoster{
			PostAlertFn: func(alert backend.Alert, channelID string) error {
				callCount++
				if alert.AlertID == "alert-2" {
					return errors.New("handler error")
				}
				postedAlerts = append(postedAlerts, alert)
				return nil
			},
		}

		mockDedup := NewMockDeduplicator()
		processor := NewAlertProcessor(client, "dataminr", "Test Backend", mockPoster, "test-channel-id", mockDedup)

		alerts := []Alert{
			{
				AlertID:       "alert-1",
				AlertType:     AlertType{Name: "Flash"},
				EventTime:     eventTime,
				Headline:      "Test Alert 1",
				FirstAlertURL: "https://example.com/alert/1",
			},
			{
				AlertID:       "alert-2",
				AlertType:     AlertType{Name: "Urgent"},
				EventTime:     eventTime,
				Headline:      "Test Alert 2",
				FirstAlertURL: "https://example.com/alert/2",
			},
			{
				AlertID:       "alert-3",
				AlertType:     AlertType{Name: "Alert"},
				EventTime:     eventTime,
				Headline:      "Test Alert 3",
				FirstAlertURL: "https://example.com/alert/3",
			},
		}

		count, err := processor.ProcessAlerts(alerts)

		assert.NoError(t, err)
		assert.Equal(t, 2, count) // alert-1 and alert-3 succeeded
		assert.Len(t, postedAlerts, 2)
		assert.Equal(t, 3, callCount) // Handler was called for all 3
	})

	t.Run("handles empty alert batch", func(t *testing.T) {
		api := plugintest.NewAPI(t)
		client := pluginapi.NewClient(api, &plugintest.Driver{})

		postedAlerts := []backend.Alert{}
		mockPoster := &MockPoster{
			PostAlertFn: func(alert backend.Alert, channelID string) error {
				postedAlerts = append(postedAlerts, alert)
				return nil
			},
		}

		mockDedup := NewMockDeduplicator()
		processor := NewAlertProcessor(client, "dataminr", "Test Backend", mockPoster, "test-channel-id", mockDedup)

		count, err := processor.ProcessAlerts([]Alert{})

		assert.NoError(t, err)
		assert.Equal(t, 0, count)
		assert.Len(t, postedAlerts, 0)
	})

	t.Run("poster is called for each alert", func(t *testing.T) {
		api := plugintest.NewAPI(t)
		api.On("LogDebug", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
		client := pluginapi.NewClient(api, &plugintest.Driver{})

		mockPoster := &MockPoster{}

		mockDedup := NewMockDeduplicator()
		processor := NewAlertProcessor(client, "dataminr", "Test Backend", mockPoster, "test-channel-id", mockDedup)

		alerts := []Alert{
			{
				AlertID:       "alert-1",
				AlertType:     AlertType{Name: "Flash"},
				EventTime:     eventTime,
				Headline:      "Test Alert 1",
				FirstAlertURL: "https://example.com/alert/1",
			},
		}

		count, err := processor.ProcessAlerts(alerts)

		assert.NoError(t, err)
		assert.Equal(t, 1, count)
	})

	t.Run("normalizes alerts correctly", func(t *testing.T) {
		api := plugintest.NewAPI(t)
		api.On("LogDebug", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
		client := pluginapi.NewClient(api, &plugintest.Driver{})

		var capturedAlert backend.Alert
		mockPoster := &MockPoster{
			PostAlertFn: func(alert backend.Alert, channelID string) error {
				capturedAlert = alert
				return nil
			},
		}

		mockDedup := NewMockDeduplicator()
		processor := NewAlertProcessor(client, "dataminr", "Test Backend", mockPoster, "test-channel-id", mockDedup)

		alerts := []Alert{
			{
				AlertID:       "alert-1",
				AlertType:     AlertType{Name: "Flash"},
				EventTime:     eventTime,
				Headline:      "Test Alert",
				FirstAlertURL: "https://example.com/alert/1",
				Location: &Location{
					Address:               "123 Main St",
					Latitude:              40.7128,
					Longitude:             -74.0060,
					ConfidenceRadiusMiles: 1.0,
				},
			},
		}

		count, err := processor.ProcessAlerts(alerts)

		assert.NoError(t, err)
		assert.Equal(t, 1, count)
		assert.Equal(t, "Test Backend", capturedAlert.BackendName)
		assert.Equal(t, "alert-1", capturedAlert.AlertID)
		assert.Equal(t, "Flash", capturedAlert.AlertType)
		assert.NotNil(t, capturedAlert.Location)
		assert.InDelta(t, 1609.34, capturedAlert.Location.ConfidenceRadius, 0.01)
	})
}
