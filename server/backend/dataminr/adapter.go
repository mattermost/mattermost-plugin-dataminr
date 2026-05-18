// Copyright (c) 2025-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package dataminr

import (
	"fmt"

	"github.com/mattermost/mattermost-plugin-dataminr/server/backend"
)

const (
	// MilesToMeters conversion factor
	MilesToMeters = 1609.34
)

// NormalizeAlert converts a Dataminr alert to a normalized backend.Alert
func NormalizeAlert(alert Alert, backendName string) *backend.Alert {
	normalized := &backend.Alert{
		BackendName: backendName,
		AlertID:     alert.AlertID,
		Headline:    alert.Headline,
		AlertType:   alert.AlertType.Name,
		EventTime:   alert.EventTime,
		AlertURL:    alert.FirstAlertURL,
	}

	// Parse location and convert confidence radius from miles to meters
	if alert.Location != nil {
		normalized.Location = &backend.Location{
			Address:          alert.Location.Address,
			Latitude:         alert.Location.Latitude,
			Longitude:        alert.Location.Longitude,
			ConfidenceRadius: alert.Location.ConfidenceRadiusMiles * MilesToMeters,
		}
	}

	// Extract sub-headline
	if alert.SubHeadline != nil {
		subHeadline := ""
		if alert.SubHeadline.Title != "" {
			subHeadline = fmt.Sprintf("**%s**\n%s", alert.SubHeadline.Title, alert.SubHeadline.SubHeadlines)
		} else {
			subHeadline = alert.SubHeadline.SubHeadlines
		}
		normalized.SubHeadline = subHeadline
	}

	// Extract topics
	if len(alert.AlertTopics) > 0 {
		normalized.Topics = make([]string, len(alert.AlertTopics))
		for i, topic := range alert.AlertTopics {
			normalized.Topics[i] = topic.Name
		}
	}

	// Extract alert lists
	if len(alert.AlertLists) > 0 {
		normalized.AlertLists = make([]string, len(alert.AlertLists))
		for i, list := range alert.AlertLists {
			normalized.AlertLists[i] = list.Name
		}
	}

	// Extract linked alerts
	if len(alert.LinkedAlerts) > 0 {
		normalized.LinkedAlerts = make([]string, 0, len(alert.LinkedAlerts))
		for _, linked := range alert.LinkedAlerts {
			if linked.Count > 0 {
				normalized.LinkedAlerts = append(normalized.LinkedAlerts,
					fmt.Sprintf("%d linked alerts (parent: %s)", linked.Count, linked.ParentID))
			}
		}
	}

	// Extract public post data
	if alert.PublicPost != nil {
		normalized.SourceText = alert.PublicPost.Text
		normalized.TranslatedText = alert.PublicPost.TranslatedText
		normalized.PublicSourceURL = alert.PublicPost.Link
		normalized.MediaURLs = alert.PublicPost.Media
	}

	return normalized
}
