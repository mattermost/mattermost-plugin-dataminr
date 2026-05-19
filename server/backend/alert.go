// Copyright (c) 2025-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package backend

import "time"

// Location represents geographic location data for an alert.
type Location struct {
	// Address is the human-readable address or location description
	Address string `json:"address,omitempty"`

	// Latitude is the geographic latitude coordinate
	Latitude float64 `json:"latitude,omitempty"`

	// Longitude is the geographic longitude coordinate
	Longitude float64 `json:"longitude,omitempty"`

	// ConfidenceRadius is the uncertainty radius in meters
	ConfidenceRadius float64 `json:"confidenceRadius,omitempty"`
}

// Alert represents a normalized alert from any backend type.
// This is the common format that all backends must convert their alerts into.
type Alert struct {
	// BackendName is the name of the backend that produced this alert
	BackendName string `json:"backendName"`

	// AlertID is the unique identifier for this alert from the backend
	AlertID string `json:"alertId"`

	// Headline is the main alert title or summary
	Headline string `json:"headline"`

	// AlertType is the type/priority of the alert (e.g., "Flash", "Urgent", "Alert")
	AlertType string `json:"alertType"`

	// EventTime is when the event occurred
	EventTime time.Time `json:"eventTime"`

	// Location contains geographic data for the alert
	Location *Location `json:"location,omitempty"`

	// AlertURL is the link to view the full alert (if available)
	AlertURL string `json:"alertUrl,omitempty"`

	// SubHeadline provides additional context (if available)
	SubHeadline string `json:"subHeadline,omitempty"`

	// Topics is a list of topics/categories associated with this alert
	Topics []string `json:"topics,omitempty"`

	// AlertLists is a list of alert list names this alert belongs to
	AlertLists []string `json:"alertLists,omitempty"`

	// LinkedAlerts is a list of related alert IDs
	LinkedAlerts []string `json:"linkedAlerts,omitempty"`

	// SourceText is the original source text (may be truncated for display)
	SourceText string `json:"sourceText,omitempty"`

	// TranslatedText is the translated text (if available, may be truncated)
	TranslatedText string `json:"translatedText,omitempty"`

	// PublicSourceURL is a link to the public source (if available)
	PublicSourceURL string `json:"publicSourceUrl,omitempty"`

	// MediaURLs is a list of media URLs associated with this alert
	// The first media URL is typically displayed as an embedded image
	MediaURLs []string `json:"mediaUrls,omitempty"`
}
