// Copyright (c) 2025-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package poster

import (
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"

	"github.com/mattermost/mattermost-plugin-dataminr/server/backend"
	"github.com/mattermost/mattermost-plugin-dataminr/server/formatter"
	"github.com/mattermost/mattermost-plugin-dataminr/server/hashtag"
)

// Poster posts alerts to Mattermost channels.
// This struct is stateless - it only holds immutable configuration (API and botID).
type Poster struct {
	api   plugin.API
	botID string
}

// New creates a new Poster instance.
func New(api plugin.API, botID string) *Poster {
	return &Poster{
		api:   api,
		botID: botID,
	}
}

// PostAlert posts a formatted alert to a Mattermost channel as a single post.
//
// Parameters:
//   - alert: The normalized alert to post
//   - channelID: The target channel ID
//
// Returns an error if the post fails.
func (p *Poster) PostAlert(alert backend.Alert, channelID string) error {
	// Format alert attachment with all fields
	attachment := formatter.FormatAlert(alert)

	// Generate alert type text and hashtags for searchability
	alertTypeText := formatter.GetAlertTypeText(alert.AlertType)
	hashtagText := hashtag.Generate(alert)

	// Create post with alert type and hashtags in message
	message := alertTypeText
	if hashtagText != "" {
		message += " " + hashtagText
	}

	post := &model.Post{
		UserId:    p.botID,
		ChannelId: channelID,
		Type:      model.PostTypeSlackAttachment,
		Message:   message, // Alert type + hashtags
		Props:     model.StringInterface{},
	}

	// Add attachment to post props
	model.ParseSlackAttachment(post, []*model.SlackAttachment{attachment})

	// Post to channel
	_, err := p.api.CreatePost(post)
	if err != nil {
		return err
	}
	return nil
}
