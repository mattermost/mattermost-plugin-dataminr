// Copyright (c) 2025-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package dataminr

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeAlert(t *testing.T) {
	eventTime := time.Now().UTC()

	t.Run("full alert with all fields", func(t *testing.T) {
		alert := Alert{
			AlertID: "alert-123",
			AlertType: AlertType{
				Name:  "Flash",
				Color: "red",
			},
			EventTime:     eventTime,
			Headline:      "Breaking News Event",
			FirstAlertURL: "https://app.dataminr.com/alert/123",
			Location: &Location{
				Address:               "123 Main St, City, State",
				Latitude:              40.7128,
				Longitude:             -74.0060,
				ConfidenceRadiusMiles: 2.5,
				MGRSCode:              "18TWL8308",
			},
			SubHeadline: &SubHeadline{
				Title:        "Additional Context",
				SubHeadlines: "More details about the event",
			},
			AlertTopics: []AlertTopic{
				{Name: "Weather", ID: "topic-1"},
				{Name: "Emergency", ID: "topic-2"},
			},
			AlertLists: []AlertList{
				{Name: "Critical Alerts"},
				{Name: "Region: North America"},
			},
			LinkedAlerts: []LinkedAlert{
				{Count: 3, ParentID: "parent-alert-456"},
			},
			PublicPost: &PublicPost{
				Link:           "https://twitter.com/user/status/123",
				Text:           "Original tweet text",
				TranslatedText: "Translated tweet text",
				Media:          []string{"https://example.com/image1.jpg", "https://example.com/image2.jpg"},
			},
		}

		normalized := NormalizeAlert(alert, "Test Backend")

		assert.Equal(t, "Test Backend", normalized.BackendName)
		assert.Equal(t, "alert-123", normalized.AlertID)
		assert.Equal(t, "Breaking News Event", normalized.Headline)
		assert.Equal(t, "Flash", normalized.AlertType)
		assert.Equal(t, eventTime, normalized.EventTime)
		assert.Equal(t, "https://app.dataminr.com/alert/123", normalized.AlertURL)

		// Check location with miles to meters conversion
		assert.NotNil(t, normalized.Location)
		assert.Equal(t, "123 Main St, City, State", normalized.Location.Address)
		assert.Equal(t, 40.7128, normalized.Location.Latitude)
		assert.Equal(t, -74.0060, normalized.Location.Longitude)
		assert.InDelta(t, 4023.35, normalized.Location.ConfidenceRadius, 0.01) // 2.5 miles in meters

		// Check sub-headline formatting
		assert.Equal(t, "**Additional Context**\nMore details about the event", normalized.SubHeadline)

		// Check topics
		assert.Equal(t, []string{"Weather", "Emergency"}, normalized.Topics)

		// Check alert lists
		assert.Equal(t, []string{"Critical Alerts", "Region: North America"}, normalized.AlertLists)

		// Check linked alerts
		assert.Equal(t, []string{"3 linked alerts (parent: parent-alert-456)"}, normalized.LinkedAlerts)

		// Check public post
		assert.Equal(t, "Original tweet text", normalized.SourceText)
		assert.Equal(t, "Translated tweet text", normalized.TranslatedText)
		assert.Equal(t, "https://twitter.com/user/status/123", normalized.PublicSourceURL)
		assert.Equal(t, []string{"https://example.com/image1.jpg", "https://example.com/image2.jpg"}, normalized.MediaURLs)
	})

	t.Run("minimal alert with only required fields", func(t *testing.T) {
		alert := Alert{
			AlertID: "alert-456",
			AlertType: AlertType{
				Name:  "Alert",
				Color: "yellow",
			},
			EventTime:     eventTime,
			Headline:      "Simple Alert",
			FirstAlertURL: "https://app.dataminr.com/alert/456",
		}

		normalized := NormalizeAlert(alert, "Test Backend")

		assert.Equal(t, "Test Backend", normalized.BackendName)
		assert.Equal(t, "alert-456", normalized.AlertID)
		assert.Equal(t, "Simple Alert", normalized.Headline)
		assert.Equal(t, "Alert", normalized.AlertType)
		assert.Equal(t, eventTime, normalized.EventTime)
		assert.Equal(t, "https://app.dataminr.com/alert/456", normalized.AlertURL)

		// Optional fields should be empty/nil
		assert.Nil(t, normalized.Location)
		assert.Empty(t, normalized.SubHeadline)
		assert.Empty(t, normalized.Topics)
		assert.Empty(t, normalized.AlertLists)
		assert.Empty(t, normalized.LinkedAlerts)
		assert.Empty(t, normalized.SourceText)
		assert.Empty(t, normalized.TranslatedText)
		assert.Empty(t, normalized.PublicSourceURL)
		assert.Empty(t, normalized.MediaURLs)
	})

	t.Run("sub-headline without title", func(t *testing.T) {
		alert := Alert{
			AlertID: "alert-789",
			AlertType: AlertType{
				Name: "Urgent",
			},
			EventTime:     eventTime,
			Headline:      "Test Alert",
			FirstAlertURL: "https://app.dataminr.com/alert/789",
			SubHeadline: &SubHeadline{
				SubHeadlines: "Just the sub-headline text",
			},
		}

		normalized := NormalizeAlert(alert, "Test Backend")

		assert.Equal(t, "Just the sub-headline text", normalized.SubHeadline)
	})

	t.Run("linked alerts with zero count", func(t *testing.T) {
		alert := Alert{
			AlertID: "alert-999",
			AlertType: AlertType{
				Name: "Alert",
			},
			EventTime:     eventTime,
			Headline:      "Test Alert",
			FirstAlertURL: "https://app.dataminr.com/alert/999",
			LinkedAlerts: []LinkedAlert{
				{Count: 0, ParentID: "parent-123"}, // Zero count should be excluded
				{Count: 5, ParentID: "parent-456"},
			},
		}

		normalized := NormalizeAlert(alert, "Test Backend")

		// Only the linked alert with count > 0 should be included
		assert.Equal(t, []string{"5 linked alerts (parent: parent-456)"}, normalized.LinkedAlerts)
	})

	t.Run("location without optional fields", func(t *testing.T) {
		alert := Alert{
			AlertID: "alert-111",
			AlertType: AlertType{
				Name: "Alert",
			},
			EventTime:     eventTime,
			Headline:      "Test Alert",
			FirstAlertURL: "https://app.dataminr.com/alert/111",
			Location: &Location{
				Address:   "City Name",
				Latitude:  35.0,
				Longitude: -120.0,
				// No confidence radius or MGRS code
			},
		}

		normalized := NormalizeAlert(alert, "Test Backend")

		assert.NotNil(t, normalized.Location)
		assert.Equal(t, "City Name", normalized.Location.Address)
		assert.Equal(t, 35.0, normalized.Location.Latitude)
		assert.Equal(t, -120.0, normalized.Location.Longitude)
		assert.Equal(t, 0.0, normalized.Location.ConfidenceRadius) // Zero confidence radius
	})

	t.Run("miles to meters conversion", func(t *testing.T) {
		testCases := []struct {
			miles  float64
			meters float64
		}{
			{1.0, 1609.34},
			{0.5, 804.67},
			{10.0, 16093.4},
			{0.0, 0.0},
		}

		for _, tc := range testCases {
			alert := Alert{
				AlertID:   "test",
				AlertType: AlertType{Name: "Alert"},
				EventTime: eventTime,
				Headline:  "Test",
				Location: &Location{
					ConfidenceRadiusMiles: tc.miles,
				},
			}

			normalized := NormalizeAlert(alert, "Test")
			assert.InDelta(t, tc.meters, normalized.Location.ConfidenceRadius, 0.01)
		}
	})
}
