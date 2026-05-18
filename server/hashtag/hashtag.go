// Copyright (c) 2025-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package hashtag

import (
	"strings"

	"github.com/mattermost/mattermost-plugin-dataminr/server/backend"
)

// Generate creates formatted hashtag text from alert data.
//
// This is the main function that coordinates all hashtag extraction.
//
// Order of hashtags:
// 1. Alert level (#Flash, #Urgent, #Alert)
// 2. Countries (up to 2)
// 3. Topics (all, deduplicated)
//
// Returns formatted string (e.g., "🏷️ #Flash, #Ukraine, #Fire")
func Generate(alert backend.Alert) string {
	var allTags []string

	// 1. Alert level (always first)
	alertLevelTag := extractAlertLevelTag(alert.AlertType)
	allTags = append(allTags, alertLevelTag)

	// 2. Countries (if location available)
	if alert.Location != nil && alert.Location.Address != "" {
		countryTags := extractCountryTags(alert.Location.Address)
		allTags = append(allTags, countryTags...)
	}

	// 3. Topics (all topics, will be deduplicated)
	if len(alert.Topics) > 0 {
		topicTags := extractTopicTags(alert.Topics)
		allTags = append(allTags, topicTags...)
	}

	// Deduplicate while preserving order
	uniqueTags := deduplicateTags(allTags)

	// Format and return
	return formatHashtagText(uniqueTags)
}

// extractAlertLevelTag extracts hashtag from alert type.
func extractAlertLevelTag(alertType string) string {
	if alertType == "" {
		return "#Alert"
	}

	// Capitalize first letter, ensure clean format
	cleanType := strings.TrimSpace(alertType)
	if len(cleanType) > 0 {
		cleanType = strings.ToUpper(cleanType[:1]) + strings.ToLower(cleanType[1:])
	}
	return "#" + cleanType
}

// deduplicateTags removes duplicate tags (case-insensitive) while preserving order.
func deduplicateTags(tags []string) []string {
	seen := make(map[string]bool)
	var uniqueTags []string

	for _, tag := range tags {
		tagLower := strings.ToLower(tag)
		if !seen[tagLower] {
			uniqueTags = append(uniqueTags, tag)
			seen[tagLower] = true
		}
	}

	return uniqueTags
}

// formatHashtagText formats hashtags as comma-separated text with emoji prefix.
func formatHashtagText(tags []string) string {
	if len(tags) == 0 {
		return ""
	}

	return "🏷️ " + strings.Join(tags, ", ")
}

// camelCase converts text to CamelCase by capitalizing first letter of each word
// and removing spaces.
func camelCase(text string) string {
	words := strings.Fields(text)
	var result strings.Builder

	for _, word := range words {
		if len(word) > 0 {
			result.WriteString(strings.ToUpper(word[:1]))
			if len(word) > 1 {
				result.WriteString(word[1:])
			}
		}
	}

	return result.String()
}
