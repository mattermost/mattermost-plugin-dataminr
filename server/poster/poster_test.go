// Copyright (c) 2025-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package poster

import (
	"testing"
	"time"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/mattermost/mattermost-plugin-dataminr/server/backend"
)

func TestPostAlert_Success(t *testing.T) {
	// Create mock API
	api := &plugintest.API{}
	defer api.AssertExpectations(t)

	botID := "bot-user-id"
	channelID := "channel-id"
	postID := "post-id"

	// Create test alert
	alert := backend.Alert{
		BackendName: "Test Backend",
		AlertID:     "alert-123",
		AlertType:   "Flash",
		Headline:    "Test Alert",
		EventTime:   time.Now(),
	}

	// Mock CreatePost to succeed
	api.On("CreatePost", mock.MatchedBy(func(post *model.Post) bool {
		// Verify post fields
		assert.Equal(t, botID, post.UserId, "Post should use bot user ID")
		assert.Equal(t, channelID, post.ChannelId, "Post should target correct channel")
		assert.Equal(t, model.PostTypeSlackAttachment, post.Type, "Post should be slack_attachment type")
		assert.NotNil(t, post.Props, "Post should have props")

		// Verify attachment was added to props
		attachments, ok := post.Props["attachments"]
		assert.True(t, ok, "Post props should contain attachments")
		assert.NotNil(t, attachments, "Post attachments should not be nil")

		return true
	})).Return(&model.Post{Id: postID}, nil).Once()

	// Create poster and call PostAlert
	poster := New(api, botID)
	err := poster.PostAlert(alert, channelID)

	// Verify no error
	require.NoError(t, err)
}

func TestPostAlert_PostError(t *testing.T) {
	// Create mock API
	api := &plugintest.API{}
	defer api.AssertExpectations(t)

	botID := "bot-user-id"
	channelID := "channel-id"

	// Create test alert
	alert := backend.Alert{
		BackendName: "Test Backend",
		AlertID:     "alert-123",
		AlertType:   "Urgent",
		Headline:    "Test Alert",
		EventTime:   time.Now(),
	}

	// Mock CreatePost to fail
	expectedErr := &model.AppError{
		Id:      "app.post.create.error",
		Message: "Failed to create post",
	}
	api.On("CreatePost", mock.Anything).Return(nil, expectedErr).Once()

	// Create poster and call PostAlert
	poster := New(api, botID)
	err := poster.PostAlert(alert, channelID)

	// Verify error is returned
	require.Error(t, err)
	assert.Equal(t, expectedErr, err)
}

func TestPostAlert_ChannelNotFound(t *testing.T) {
	// Create mock API
	api := &plugintest.API{}
	defer api.AssertExpectations(t)

	botID := "bot-user-id"
	channelID := "nonexistent-channel"

	// Create test alert
	alert := backend.Alert{
		BackendName: "Test Backend",
		AlertID:     "alert-123",
		AlertType:   "Alert",
		Headline:    "Test Alert",
		EventTime:   time.Now(),
	}

	// Mock CreatePost to fail with 404
	expectedErr := &model.AppError{
		Id:         "app.channel.get.find.app_error",
		Message:    "Channel not found",
		StatusCode: 404,
	}
	api.On("CreatePost", mock.Anything).Return(nil, expectedErr)

	// Create poster and call PostAlert
	poster := New(api, botID)
	err := poster.PostAlert(alert, channelID)

	// Verify error is returned
	require.Error(t, err)
	assert.Equal(t, expectedErr, err)
}

func TestPostAlert_PermissionError(t *testing.T) {
	// Create mock API
	api := &plugintest.API{}
	defer api.AssertExpectations(t)

	botID := "bot-user-id"
	channelID := "private-channel"

	// Create test alert
	alert := backend.Alert{
		BackendName: "Test Backend",
		AlertID:     "alert-123",
		AlertType:   "Flash",
		Headline:    "Test Alert",
		EventTime:   time.Now(),
	}

	// Mock CreatePost to fail with 403 permission error
	expectedErr := &model.AppError{
		Id:         "api.context.permissions.app_error",
		Message:    "You do not have permission",
		StatusCode: 403,
	}
	api.On("CreatePost", mock.Anything).Return(nil, expectedErr)

	// Create poster and call PostAlert
	poster := New(api, botID)
	err := poster.PostAlert(alert, channelID)

	// Verify error is returned
	require.Error(t, err)
	assert.Equal(t, expectedErr, err)
}

func TestPostAlert_WithRichAlert(t *testing.T) {
	// Create mock API
	api := &plugintest.API{}
	defer api.AssertExpectations(t)

	botID := "bot-user-id"
	channelID := "channel-id"
	postID := "post-id"

	// Create rich alert with all fields
	alert := backend.Alert{
		BackendName:     "Test Backend",
		AlertID:         "alert-123",
		AlertType:       "Flash",
		Headline:        "Breaking News",
		SubHeadline:     "This is a sub-headline with details",
		EventTime:       time.Now(),
		AlertURL:        "https://example.com/alert/123",
		PublicSourceURL: "https://news.example.com/article",
		SourceText:      "Original source text here",
		TranslatedText:  "Translated text here",
		Topics:          []string{"Topic1", "Topic2"},
		AlertLists:      []string{"List1", "List2"},
		LinkedAlerts:    []string{"alert-124", "alert-125"},
		MediaURLs:       []string{"https://example.com/image1.jpg", "https://example.com/image2.jpg"},
		Location: &backend.Location{
			Address:          "123 Main St, City, Country",
			Latitude:         40.7128,
			Longitude:        -74.0060,
			ConfidenceRadius: 5000,
		},
	}

	// Mock CreatePost to succeed
	api.On("CreatePost", mock.MatchedBy(func(post *model.Post) bool {
		// Verify post fields
		assert.Equal(t, botID, post.UserId)
		assert.Equal(t, channelID, post.ChannelId)
		assert.Equal(t, model.PostTypeSlackAttachment, post.Type)
		assert.NotNil(t, post.Props)

		// Verify attachment was added to props
		attachments, ok := post.Props["attachments"]
		assert.True(t, ok)
		assert.NotNil(t, attachments)

		return true
	})).Return(&model.Post{Id: postID}, nil).Once()

	// Create poster and call PostAlert
	poster := New(api, botID)
	err := poster.PostAlert(alert, channelID)

	// Verify no error
	require.NoError(t, err)
}
