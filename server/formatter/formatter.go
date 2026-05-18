// Copyright (c) 2025-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package formatter

import (
	"fmt"
	"strings"
	"time"

	"github.com/mattermost/mattermost/server/public/model"

	"github.com/mattermost/mattermost-plugin-dataminr/server/backend"
)

// Alert type colors
const (
	ColorFlash   = "#D24B4E" // Red 🔴
	ColorUrgent  = "#EC8832" // Orange 🟠
	ColorAlert   = "#FFBC1F" // Yellow 🟡
	ColorUnknown = "#D3D3D3" // Light Gray ⚪
)

// Alert type emojis
const (
	EmojiFlash   = "🔴"
	EmojiUrgent  = "🟠"
	EmojiAlert   = "🟡"
	EmojiUnknown = "⚪"
)

// GetAlertTypeText returns the formatted alert type text with emoji
func GetAlertTypeText(alertType string) string {
	return fmt.Sprintf("%s **%s**", getAlertEmoji(alertType), strings.ToUpper(alertType))
}

// FormatAlert creates a single alert post attachment with all alert information.
func FormatAlert(alert backend.Alert) *model.SlackAttachment {
	attachment := &model.SlackAttachment{}

	// Set text with title - use markdown H3 header for emphasis
	attachment.Text = fmt.Sprintf("### %s", alert.Headline)

	// Set color based on alert type
	attachment.Color = getAlertColor(alert.AlertType)

	// Build all fields
	var fields []*model.SlackAttachmentField

	// Alert Link and Public Source (side by side at top)
	if alert.AlertURL != "" {
		fields = append(fields, &model.SlackAttachmentField{
			Title: "Alert Link",
			Value: fmt.Sprintf("**[Open in Dataminr](%s)**", alert.AlertURL),
			Short: true,
		})
	}

	if alert.PublicSourceURL != "" {
		fields = append(fields, &model.SlackAttachmentField{
			Title: "Public Source",
			Value: fmt.Sprintf("**[Open Public Link](%s)**", alert.PublicSourceURL),
			Short: true,
		})
	}

	// Event Time and Location (side by side)
	fields = append(fields,
		&model.SlackAttachmentField{
			Title: "Event Time",
			Value: formatTime(alert.EventTime),
			Short: true,
		},
	)

	if alert.Location != nil && alert.Location.Address != "" {
		fields = append(fields, &model.SlackAttachmentField{
			Title: "Location",
			Value: formatLocation(alert.Location),
			Short: true,
		})
	}

	// Additional Context (sub-headline if available)
	if alert.SubHeadline != "" {
		fields = append(fields, &model.SlackAttachmentField{
			Title: "Additional Context",
			Value: alert.SubHeadline,
			Short: false,
		})
	}

	// Original Source Text (truncate at 500 chars)
	if alert.SourceText != "" {
		fields = append(fields, &model.SlackAttachmentField{
			Title: "Original Source Text",
			Value: truncateText(alert.SourceText, 500),
			Short: false,
		})
	}

	// Translated Text (truncate at 500 chars)
	if alert.TranslatedText != "" {
		fields = append(fields, &model.SlackAttachmentField{
			Title: "Translated Text",
			Value: truncateText(alert.TranslatedText, 500),
			Short: false,
		})
	}

	// Topics (bulleted list, full width)
	if len(alert.Topics) > 0 {
		fields = append(fields, &model.SlackAttachmentField{
			Title: "Topics",
			Value: formatBulletList(alert.Topics),
			Short: false,
		})
	}

	// Alert Lists (bulleted list, full width)
	if len(alert.AlertLists) > 0 {
		fields = append(fields, &model.SlackAttachmentField{
			Title: "Alert Lists",
			Value: formatBulletList(alert.AlertLists),
			Short: false,
		})
	}

	// Additional Media (links to media 2-4)
	if len(alert.MediaURLs) > 1 {
		additionalMedia := alert.MediaURLs[1:]
		if len(additionalMedia) > 3 {
			additionalMedia = additionalMedia[:3] // Limit to 3 additional media
		}
		fields = append(fields, &model.SlackAttachmentField{
			Title: "Additional Media",
			Value: formatMediaLinks(additionalMedia),
			Short: false,
		})
	}

	attachment.Fields = fields

	// Set image URL: First media item embedded
	if len(alert.MediaURLs) > 0 {
		attachment.ImageURL = alert.MediaURLs[0]
	}

	// Set footer: Backend name
	attachment.Footer = alert.BackendName

	return attachment
}

// getAlertColor returns the color code for an alert type
func getAlertColor(alertType string) string {
	switch strings.ToLower(alertType) {
	case "flash":
		return ColorFlash
	case "urgent":
		return ColorUrgent
	case "alert":
		return ColorAlert
	default:
		return ColorUnknown
	}
}

// getAlertEmoji returns the emoji for an alert type
func getAlertEmoji(alertType string) string {
	switch strings.ToLower(alertType) {
	case "flash":
		return EmojiFlash
	case "urgent":
		return EmojiUrgent
	case "alert":
		return EmojiAlert
	default:
		return EmojiUnknown
	}
}

// formatTime formats a time.Time to a readable string
func formatTime(t time.Time) string {
	return t.Format("2006-01-02 15:04:05 MST")
}

// formatLocation formats a Location struct to a readable string
func formatLocation(loc *backend.Location) string {
	parts := []string{}

	if loc.Address != "" {
		parts = append(parts, loc.Address)
	}

	if loc.Latitude != 0 || loc.Longitude != 0 {
		parts = append(parts, fmt.Sprintf("(%.6f, %.6f)", loc.Latitude, loc.Longitude))
	}

	if loc.ConfidenceRadius > 0 {
		parts = append(parts, fmt.Sprintf("±%.0fm", loc.ConfidenceRadius))
	}

	return strings.Join(parts, " ")
}

// formatBulletList formats a slice of strings as a bulleted list
func formatBulletList(items []string) string {
	bullets := make([]string, len(items))
	for i, item := range items {
		bullets[i] = fmt.Sprintf("• %s", item)
	}
	return strings.Join(bullets, "\n")
}

// truncateText truncates text to maxLen characters, adding "..." if truncated
func truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen] + "..."
}

// formatMediaLinks formats media URLs as markdown links
func formatMediaLinks(urls []string) string {
	links := make([]string, len(urls))
	for i, url := range urls {
		links[i] = fmt.Sprintf("[Media %d](%s)", i+2, url) // Start at 2 since first media is embedded
	}
	return strings.Join(links, " | ")
}
