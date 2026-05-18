// Copyright (c) 2025-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package formatter

import (
	"strings"
	"testing"
	"time"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mattermost/mattermost-plugin-dataminr/server/backend"
)

func TestFormatAlert_FullAlert(t *testing.T) {
	alert := backend.Alert{
		BackendName:     "Test Backend",
		AlertID:         "test-123",
		Headline:        "Breaking News",
		AlertType:       "Flash",
		EventTime:       time.Date(2025, 10, 30, 14, 30, 0, 0, time.UTC),
		AlertURL:        "https://example.com/alert/123",
		SubHeadline:     "Additional important context",
		Topics:          []string{"Politics", "Economy"},
		AlertLists:      []string{"Critical", "Breaking"},
		LinkedAlerts:    []string{"alert-1", "alert-2"},
		SourceText:      "Original source text here",
		TranslatedText:  "Translated text here",
		PublicSourceURL: "https://example.com/source",
		MediaURLs:       []string{"https://example.com/image1.jpg", "https://example.com/image2.jpg"},
		Location: &backend.Location{
			Address:          "123 Main St, City",
			Latitude:         40.7128,
			Longitude:        -74.0060,
			ConfidenceRadius: 100.5,
		},
	}

	attachment := FormatAlert(alert)

	// Verify basic structure
	assert.Contains(t, attachment.Text, "Breaking News")
	assert.NotContains(t, attachment.Text, "https://example.com/alert/123")
	assert.Empty(t, attachment.Pretext)
	assert.Empty(t, attachment.Title)
	assert.Empty(t, attachment.TitleLink)
	assert.Equal(t, ColorFlash, attachment.Color)
	assert.Equal(t, "https://example.com/image1.jpg", attachment.ImageURL)
	assert.Equal(t, "Test Backend", attachment.Footer)

	// Verify all fields exist in correct order
	require.Len(t, attachment.Fields, 10)

	// Field 0: Alert Link
	assert.Equal(t, "Alert Link", attachment.Fields[0].Title)
	assert.Equal(t, "**[Open in Dataminr](https://example.com/alert/123)**", attachment.Fields[0].Value)
	assert.Equal(t, model.SlackCompatibleBool(true), attachment.Fields[0].Short)

	// Field 1: Public Source
	assert.Equal(t, "Public Source", attachment.Fields[1].Title)
	assert.Equal(t, "**[Open Public Link](https://example.com/source)**", attachment.Fields[1].Value)
	assert.Equal(t, model.SlackCompatibleBool(true), attachment.Fields[1].Short)

	// Field 2: Event Time
	assert.Equal(t, "Event Time", attachment.Fields[2].Title)
	assert.Equal(t, "2025-10-30 14:30:00 UTC", attachment.Fields[2].Value)
	assert.Equal(t, model.SlackCompatibleBool(true), attachment.Fields[2].Short)

	// Field 3: Location
	assert.Equal(t, "Location", attachment.Fields[3].Title)
	assert.Contains(t, attachment.Fields[3].Value, "123 Main St, City")
	assert.Contains(t, attachment.Fields[3].Value, "(40.712800, -74.006000)")
	assert.Contains(t, attachment.Fields[3].Value, "±100m")
	assert.Equal(t, model.SlackCompatibleBool(true), attachment.Fields[3].Short)

	// Field 4: Additional Context
	assert.Equal(t, "Additional Context", attachment.Fields[4].Title)
	assert.Equal(t, "Additional important context", attachment.Fields[4].Value)
	assert.Equal(t, model.SlackCompatibleBool(false), attachment.Fields[4].Short)

	// Field 5: Original Source Text
	assert.Equal(t, "Original Source Text", attachment.Fields[5].Title)
	assert.Equal(t, "Original source text here", attachment.Fields[5].Value)
	assert.Equal(t, model.SlackCompatibleBool(false), attachment.Fields[5].Short)

	// Field 6: Translated Text
	assert.Equal(t, "Translated Text", attachment.Fields[6].Title)
	assert.Equal(t, "Translated text here", attachment.Fields[6].Value)
	assert.Equal(t, model.SlackCompatibleBool(false), attachment.Fields[6].Short)

	// Field 7: Topics
	assert.Equal(t, "Topics", attachment.Fields[7].Title)
	assert.Contains(t, attachment.Fields[7].Value, "• Politics")
	assert.Contains(t, attachment.Fields[7].Value, "• Economy")
	assert.Equal(t, model.SlackCompatibleBool(false), attachment.Fields[7].Short)

	// Field 8: Alert Lists
	assert.Equal(t, "Alert Lists", attachment.Fields[8].Title)
	assert.Contains(t, attachment.Fields[8].Value, "• Critical")
	assert.Contains(t, attachment.Fields[8].Value, "• Breaking")
	assert.Equal(t, model.SlackCompatibleBool(false), attachment.Fields[8].Short)

	// Field 9: Additional Media
	assert.Equal(t, "Additional Media", attachment.Fields[9].Title)
	assert.Equal(t, "[Media 2](https://example.com/image2.jpg)", attachment.Fields[9].Value)
	assert.Equal(t, model.SlackCompatibleBool(false), attachment.Fields[9].Short)
}

func TestFormatAlert_MinimalAlert(t *testing.T) {
	alert := backend.Alert{
		BackendName: "Test Backend",
		AlertID:     "test-123",
		Headline:    "Simple Alert",
		AlertType:   "Alert",
		EventTime:   time.Date(2025, 10, 30, 14, 30, 0, 0, time.UTC),
	}

	attachment := FormatAlert(alert)

	// Verify basic structure
	assert.Contains(t, attachment.Text, "Simple Alert")
	assert.Empty(t, attachment.Pretext)
	assert.Empty(t, attachment.Title)
	assert.Empty(t, attachment.TitleLink)
	assert.Equal(t, ColorAlert, attachment.Color)
	assert.Empty(t, attachment.ImageURL)
	assert.Equal(t, "Test Backend", attachment.Footer)

	// Verify only Event Time field (no Alert Link since no AlertURL, no Alert Type field)
	require.Len(t, attachment.Fields, 1)

	// Field 0: Event Time
	assert.Equal(t, "Event Time", attachment.Fields[0].Title)
	assert.Equal(t, "2025-10-30 14:30:00 UTC", attachment.Fields[0].Value)
	assert.Equal(t, model.SlackCompatibleBool(true), attachment.Fields[0].Short)
}

func TestGetAlertColor(t *testing.T) {
	tests := []struct {
		name      string
		alertType string
		expected  string
	}{
		{"Flash lowercase", "flash", ColorFlash},
		{"Flash uppercase", "FLASH", ColorFlash},
		{"Flash mixed case", "FlAsH", ColorFlash},
		{"Urgent lowercase", "urgent", ColorUrgent},
		{"Urgent uppercase", "URGENT", ColorUrgent},
		{"Alert lowercase", "alert", ColorAlert},
		{"Alert uppercase", "ALERT", ColorAlert},
		{"Unknown type", "unknown", ColorUnknown},
		{"Random type", "random", ColorUnknown},
		{"Empty type", "", ColorUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getAlertColor(tt.alertType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetAlertEmoji(t *testing.T) {
	tests := []struct {
		name      string
		alertType string
		expected  string
	}{
		{"Flash lowercase", "flash", EmojiFlash},
		{"Flash uppercase", "FLASH", EmojiFlash},
		{"Urgent lowercase", "urgent", EmojiUrgent},
		{"Urgent uppercase", "URGENT", EmojiUrgent},
		{"Alert lowercase", "alert", EmojiAlert},
		{"Alert uppercase", "ALERT", EmojiAlert},
		{"Unknown type", "unknown", EmojiUnknown},
		{"Empty type", "", EmojiUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getAlertEmoji(tt.alertType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatTime(t *testing.T) {
	// Test with UTC time
	utcTime := time.Date(2025, 10, 30, 14, 30, 45, 0, time.UTC)
	result := formatTime(utcTime)
	assert.Equal(t, "2025-10-30 14:30:45 UTC", result)

	// Test with different timezone
	loc, err := time.LoadLocation("America/New_York")
	require.NoError(t, err)
	nyTime := time.Date(2025, 10, 30, 14, 30, 45, 0, loc)
	result = formatTime(nyTime)
	assert.Contains(t, result, "2025-10-30 14:30:45")
	assert.Contains(t, result, "E") // EDT or EST
}

func TestFormatLocation(t *testing.T) {
	tests := []struct {
		name     string
		location *backend.Location
		expected string
	}{
		{
			name: "Full location",
			location: &backend.Location{
				Address:          "123 Main St",
				Latitude:         40.7128,
				Longitude:        -74.0060,
				ConfidenceRadius: 100.5,
			},
			expected: "123 Main St (40.712800, -74.006000) ±100m",
		},
		{
			name: "Address only",
			location: &backend.Location{
				Address: "123 Main St",
			},
			expected: "123 Main St",
		},
		{
			name: "Coordinates only",
			location: &backend.Location{
				Latitude:  40.7128,
				Longitude: -74.0060,
			},
			expected: "(40.712800, -74.006000)",
		},
		{
			name: "Address and coordinates",
			location: &backend.Location{
				Address:   "123 Main St",
				Latitude:  40.7128,
				Longitude: -74.0060,
			},
			expected: "123 Main St (40.712800, -74.006000)",
		},
		{
			name: "With confidence radius",
			location: &backend.Location{
				Latitude:         40.7128,
				Longitude:        -74.0060,
				ConfidenceRadius: 250.75,
			},
			expected: "(40.712800, -74.006000) ±251m",
		},
		{
			name:     "Empty location",
			location: &backend.Location{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatLocation(tt.location)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatBulletList(t *testing.T) {
	tests := []struct {
		name     string
		items    []string
		expected string
	}{
		{
			name:     "Single item",
			items:    []string{"Item 1"},
			expected: "• Item 1",
		},
		{
			name:     "Multiple items",
			items:    []string{"Item 1", "Item 2", "Item 3"},
			expected: "• Item 1\n• Item 2\n• Item 3",
		},
		{
			name:     "Empty slice",
			items:    []string{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatBulletList(tt.items)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTruncateText(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		maxLen   int
		expected string
	}{
		{
			name:     "Text shorter than max",
			text:     "Short text",
			maxLen:   100,
			expected: "Short text",
		},
		{
			name:     "Text exactly at max",
			text:     "Exact",
			maxLen:   5,
			expected: "Exact",
		},
		{
			name:     "Text longer than max",
			text:     "This is a very long text that should be truncated",
			maxLen:   20,
			expected: "This is a very long ...",
		},
		{
			name:     "Truncate at 500 chars",
			text:     strings.Repeat("a", 600),
			maxLen:   500,
			expected: strings.Repeat("a", 500) + "...",
		},
		{
			name:     "Empty text",
			text:     "",
			maxLen:   100,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateText(tt.text, tt.maxLen)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatMediaLinks(t *testing.T) {
	tests := []struct {
		name     string
		urls     []string
		expected string
	}{
		{
			name:     "Single URL",
			urls:     []string{"https://example.com/image2.jpg"},
			expected: "[Media 2](https://example.com/image2.jpg)",
		},
		{
			name:     "Multiple URLs",
			urls:     []string{"https://example.com/image2.jpg", "https://example.com/image3.jpg"},
			expected: "[Media 2](https://example.com/image2.jpg) | [Media 3](https://example.com/image3.jpg)",
		},
		{
			name:     "Three URLs",
			urls:     []string{"https://example.com/image2.jpg", "https://example.com/image3.jpg", "https://example.com/image4.jpg"},
			expected: "[Media 2](https://example.com/image2.jpg) | [Media 3](https://example.com/image3.jpg) | [Media 4](https://example.com/image4.jpg)",
		},
		{
			name:     "Empty slice",
			urls:     []string{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatMediaLinks(tt.urls)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatAlert_MultipleMediaURLs(t *testing.T) {
	tests := []struct {
		name                  string
		mediaURLs             []string
		expectedImageURL      string
		expectAdditionalMedia bool
		expectedMediaCount    int
	}{
		{
			name:                  "No media",
			mediaURLs:             []string{},
			expectedImageURL:      "",
			expectAdditionalMedia: false,
		},
		{
			name:                  "One media",
			mediaURLs:             []string{"https://example.com/image1.jpg"},
			expectedImageURL:      "https://example.com/image1.jpg",
			expectAdditionalMedia: false,
		},
		{
			name:                  "Two media",
			mediaURLs:             []string{"https://example.com/image1.jpg", "https://example.com/image2.jpg"},
			expectedImageURL:      "https://example.com/image1.jpg",
			expectAdditionalMedia: true,
			expectedMediaCount:    1,
		},
		{
			name:                  "Four media (max 3 additional)",
			mediaURLs:             []string{"https://example.com/image1.jpg", "https://example.com/image2.jpg", "https://example.com/image3.jpg", "https://example.com/image4.jpg"},
			expectedImageURL:      "https://example.com/image1.jpg",
			expectAdditionalMedia: true,
			expectedMediaCount:    3,
		},
		{
			name:                  "Five media (only 3 additional shown)",
			mediaURLs:             []string{"https://example.com/image1.jpg", "https://example.com/image2.jpg", "https://example.com/image3.jpg", "https://example.com/image4.jpg", "https://example.com/image5.jpg"},
			expectedImageURL:      "https://example.com/image1.jpg",
			expectAdditionalMedia: true,
			expectedMediaCount:    3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			alert := backend.Alert{
				BackendName: "Test",
				AlertID:     "123",
				Headline:    "Test",
				AlertType:   "Alert",
				EventTime:   time.Now(),
				MediaURLs:   tt.mediaURLs,
			}

			attachment := FormatAlert(alert)

			assert.Equal(t, tt.expectedImageURL, attachment.ImageURL)

			// Check for Additional Media field
			hasAdditionalMedia := false
			var additionalMediaField string
			for _, field := range attachment.Fields {
				if field.Title == "Additional Media" {
					hasAdditionalMedia = true
					additionalMediaField = field.Value.(string)
					break
				}
			}

			assert.Equal(t, tt.expectAdditionalMedia, hasAdditionalMedia)

			if tt.expectAdditionalMedia {
				require.NotEmpty(t, additionalMediaField)
				// Count the number of media links
				mediaCount := strings.Count(additionalMediaField, "[Media")
				assert.Equal(t, tt.expectedMediaCount, mediaCount)
			}
		})
	}
}

func TestFormatAlert_AlertTypeVariations(t *testing.T) {
	tests := []struct {
		alertType     string
		expectedColor string
		expectedEmoji string
	}{
		{"Flash", ColorFlash, EmojiFlash},
		{"flash", ColorFlash, EmojiFlash},
		{"FLASH", ColorFlash, EmojiFlash},
		{"Urgent", ColorUrgent, EmojiUrgent},
		{"urgent", ColorUrgent, EmojiUrgent},
		{"URGENT", ColorUrgent, EmojiUrgent},
		{"Alert", ColorAlert, EmojiAlert},
		{"alert", ColorAlert, EmojiAlert},
		{"ALERT", ColorAlert, EmojiAlert},
		{"Unknown", ColorUnknown, EmojiUnknown},
		{"", ColorUnknown, EmojiUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.alertType, func(t *testing.T) {
			alert := backend.Alert{
				BackendName: "Test",
				AlertID:     "123",
				Headline:    "Test",
				AlertType:   tt.alertType,
				EventTime:   time.Now(),
			}

			attachment := FormatAlert(alert)

			assert.Equal(t, tt.expectedColor, attachment.Color)
			assert.Contains(t, attachment.Text, "Test") // Text contains headline
		})
	}
}

func TestFormatAlert_SourceTextTruncation(t *testing.T) {
	longText := strings.Repeat("a", 600)

	alert := backend.Alert{
		BackendName:    "Test",
		AlertID:        "123",
		Headline:       "Test",
		AlertType:      "Alert",
		EventTime:      time.Now(),
		SourceText:     longText,
		TranslatedText: longText,
	}

	attachment := FormatAlert(alert)

	// Find source text and translated text fields
	var sourceTextField, translatedTextField string
	for _, field := range attachment.Fields {
		if field.Title == "Original Source Text" {
			sourceTextField = field.Value.(string)
		}
		if field.Title == "Translated Text" {
			translatedTextField = field.Value.(string)
		}
	}

	require.NotEmpty(t, sourceTextField)
	require.NotEmpty(t, translatedTextField)

	// Both should be truncated to 500 chars + "..."
	assert.Equal(t, 503, len(sourceTextField))
	assert.True(t, strings.HasSuffix(sourceTextField, "..."))

	assert.Equal(t, 503, len(translatedTextField))
	assert.True(t, strings.HasSuffix(translatedTextField, "..."))
}
