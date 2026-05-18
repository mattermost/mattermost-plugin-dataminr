// Copyright (c) 2025-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package dataminr

import (
	"github.com/mattermost/mattermost/server/public/pluginapi"

	"github.com/mattermost/mattermost-plugin-dataminr/server/backend"
)

// AlertProcessor orchestrates alert normalization and deduplication
type AlertProcessor struct {
	api          *pluginapi.Client
	backendType  string
	backendName  string
	poster       backend.AlertPoster
	channelID    string
	deduplicator backend.Deduplicator
}

// NewAlertProcessor creates a new alert processor
func NewAlertProcessor(api *pluginapi.Client, backendType, backendName string, poster backend.AlertPoster, channelID string, deduplicator backend.Deduplicator) *AlertProcessor {
	return &AlertProcessor{
		api:          api,
		backendType:  backendType,
		backendName:  backendName,
		poster:       poster,
		channelID:    channelID,
		deduplicator: deduplicator,
	}
}

// ProcessAlerts processes a batch of Dataminr alerts
// Returns the number of new alerts processed (after deduplication)
func (p *AlertProcessor) ProcessAlerts(alerts []Alert) (int, error) {
	newCount := 0

	for _, alert := range alerts {
		// Atomically check and record alert (prevents race conditions)
		isNew := p.deduplicator.RecordAlert(p.backendType, alert.AlertID, p.channelID)
		if !isNew {
			p.api.Log.Debug("Skipping duplicate alert", "backendType", p.backendType, "alertId", alert.AlertID)
			continue
		}

		// Normalize to backend.Alert
		normalized := NormalizeAlert(alert, p.backendName)

		// Post alert to Mattermost channel
		if err := p.poster.PostAlert(*normalized, p.channelID); err != nil {
			p.api.Log.Error("Failed to post alert", "alertId", alert.AlertID, "channelId", p.channelID, "error", err.Error())
			continue
		}

		p.api.Log.Debug("Successfully posted alert", "alertId", alert.AlertID, "channelId", p.channelID)
		newCount++
	}

	return newCount, nil
}
