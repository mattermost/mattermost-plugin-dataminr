// Copyright (c) 2025-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package dataminr

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthResponse_TimeConversion(t *testing.T) {
	t.Run("converts expiration time from milliseconds", func(t *testing.T) {
		// Create a known time
		expectedTime := time.Date(2021, 6, 13, 20, 14, 19, 980000000, time.UTC)
		expirationMs := expectedTime.UnixNano() / int64(time.Millisecond)

		// Build JSON with milliseconds
		jsonData := fmt.Sprintf(`{
			"authorizationToken": "c1b748aa49cb4dfaa602fbdc1912e9f1",
			"expirationTime": %d,
			"TOS": "By using the API...",
			"thirdPartyTerms": "https://www.dataminr.com/terms"
		}`, expirationMs)

		var resp AuthResponse
		err := json.Unmarshal([]byte(jsonData), &resp)
		require.NoError(t, err)

		// Verify we get the exact same time back
		assert.Equal(t, expectedTime, resp.ExpirationTime)
		assert.Equal(t, "c1b748aa49cb4dfaa602fbdc1912e9f1", resp.AuthorizationToken)
	})

	t.Run("handles zero expiration time", func(t *testing.T) {
		jsonData := `{
			"authorizationToken": "token123",
			"expirationTime": 0
		}`

		var resp AuthResponse
		err := json.Unmarshal([]byte(jsonData), &resp)
		require.NoError(t, err)

		assert.Equal(t, time.Unix(0, 0).UTC(), resp.ExpirationTime)
	})
}

func TestAPIError_ErrorMethod(t *testing.T) {
	t.Run("formats error message with code", func(t *testing.T) {
		apiErr := APIError{
			Errors: []ErrorDetail{
				{Code: "103", Message: "Authentication error. Invalid token"},
			},
		}

		assert.Equal(t, "Code 103: Authentication error. Invalid token", apiErr.Error())
	})

	t.Run("returns first error when multiple", func(t *testing.T) {
		apiErr := APIError{
			Errors: []ErrorDetail{
				{Code: "101", Message: "First error"},
				{Code: "102", Message: "Second error"},
			},
		}

		assert.Equal(t, "Code 101: First error", apiErr.Error())
	})

	t.Run("returns unknown error when empty", func(t *testing.T) {
		apiErr := APIError{Errors: []ErrorDetail{}}
		assert.Equal(t, "unknown API error", apiErr.Error())
	})
}

func TestAlert_TimeConversion(t *testing.T) {
	t.Run("converts event time from milliseconds", func(t *testing.T) {
		// Create a known time
		expectedTime := time.Date(2021, 3, 24, 20, 27, 33, 472000000, time.UTC)
		eventTimeMs := expectedTime.UnixNano() / int64(time.Millisecond)

		// Build JSON with milliseconds
		jsonData := fmt.Sprintf(`{
			"alertId": "test123",
			"alertType": {"name": "Urgent", "color": "orange"},
			"eventTime": %d,
			"headline": "Test alert",
			"firstAlertURL": "https://example.com"
		}`, eventTimeMs)

		var alert Alert
		err := json.Unmarshal([]byte(jsonData), &alert)
		require.NoError(t, err)

		// Verify we get the exact same time back
		assert.Equal(t, expectedTime, alert.EventTime)
	})

	t.Run("handles zero event time", func(t *testing.T) {
		jsonData := `{
			"alertId": "test123",
			"alertType": {"name": "Alert", "color": "yellow"},
			"eventTime": 0,
			"headline": "Test",
			"firstAlertURL": "https://example.com"
		}`

		var alert Alert
		err := json.Unmarshal([]byte(jsonData), &alert)
		require.NoError(t, err)

		// Zero time should not be parsed (kept as zero value)
		assert.True(t, alert.EventTime.IsZero())
	})
}

func TestAlert_LocationParsing(t *testing.T) {
	t.Run("parses complete location array", func(t *testing.T) {
		jsonData := `{
			"alertId": "test123",
			"alertType": {"name": "Alert", "color": "yellow"},
			"eventTime": 1616621253472,
			"headline": "Test",
			"firstAlertURL": "https://example.com",
			"estimatedEventLocation": [
				"123 Main St, San Francisco, CA 94102, USA",
				37.7749,
				-122.4194,
				5.2,
				"10SEG12345678"
			]
		}`

		var alert Alert
		err := json.Unmarshal([]byte(jsonData), &alert)
		require.NoError(t, err)

		require.NotNil(t, alert.Location)
		assert.Equal(t, "123 Main St, San Francisco, CA 94102, USA", alert.Location.Address)
		assert.Equal(t, 37.7749, alert.Location.Latitude)
		assert.Equal(t, -122.4194, alert.Location.Longitude)
		assert.Equal(t, 5.2, alert.Location.ConfidenceRadiusMiles)
		assert.Equal(t, "10SEG12345678", alert.Location.MGRSCode)
	})

	t.Run("parses location without optional MGRS code", func(t *testing.T) {
		jsonData := `{
			"alertId": "test123",
			"alertType": {"name": "Alert", "color": "yellow"},
			"eventTime": 1616621253472,
			"headline": "Test",
			"firstAlertURL": "https://example.com",
			"estimatedEventLocation": [
				"Address Only",
				37.7749,
				-122.4194,
				5.2
			]
		}`

		var alert Alert
		err := json.Unmarshal([]byte(jsonData), &alert)
		require.NoError(t, err)

		require.NotNil(t, alert.Location)
		assert.Equal(t, "Address Only", alert.Location.Address)
		assert.Equal(t, 37.7749, alert.Location.Latitude)
		assert.Equal(t, -122.4194, alert.Location.Longitude)
		assert.Equal(t, 5.2, alert.Location.ConfidenceRadiusMiles)
		assert.Empty(t, alert.Location.MGRSCode)
	})

	t.Run("returns nil for insufficient location data", func(t *testing.T) {
		jsonData := `{
			"alertId": "test123",
			"alertType": {"name": "Alert", "color": "yellow"},
			"eventTime": 1616621253472,
			"headline": "Test",
			"firstAlertURL": "https://example.com",
			"estimatedEventLocation": ["Address", 37.7749, -122.4194]
		}`

		var alert Alert
		err := json.Unmarshal([]byte(jsonData), &alert)
		require.NoError(t, err)

		// Need at least 4 elements for valid location
		assert.Nil(t, alert.Location)
	})

	t.Run("returns nil for empty location array", func(t *testing.T) {
		jsonData := `{
			"alertId": "test123",
			"alertType": {"name": "Alert", "color": "yellow"},
			"eventTime": 1616621253472,
			"headline": "Test",
			"firstAlertURL": "https://example.com",
			"estimatedEventLocation": []
		}`

		var alert Alert
		err := json.Unmarshal([]byte(jsonData), &alert)
		require.NoError(t, err)

		assert.Nil(t, alert.Location)
	})

	t.Run("returns nil when location field missing", func(t *testing.T) {
		jsonData := `{
			"alertId": "test123",
			"alertType": {"name": "Alert", "color": "yellow"},
			"eventTime": 1616621253472,
			"headline": "Test",
			"firstAlertURL": "https://example.com"
		}`

		var alert Alert
		err := json.Unmarshal([]byte(jsonData), &alert)
		require.NoError(t, err)

		assert.Nil(t, alert.Location)
	})

	t.Run("handles type mismatches gracefully", func(t *testing.T) {
		jsonData := `{
			"alertId": "test123",
			"alertType": {"name": "Alert", "color": "yellow"},
			"eventTime": 1616621253472,
			"headline": "Test",
			"firstAlertURL": "https://example.com",
			"estimatedEventLocation": [
				123,
				"not_a_float",
				-122.4194,
				"not_a_float"
			]
		}`

		var alert Alert
		err := json.Unmarshal([]byte(jsonData), &alert)
		require.NoError(t, err)

		require.NotNil(t, alert.Location)
		// Wrong types should result in zero values
		assert.Empty(t, alert.Location.Address)
		assert.Equal(t, 0.0, alert.Location.Latitude)
		assert.Equal(t, -122.4194, alert.Location.Longitude)
		assert.Equal(t, 0.0, alert.Location.ConfidenceRadiusMiles)
	})

	t.Run("handles negative coordinates", func(t *testing.T) {
		jsonData := `{
			"alertId": "test123",
			"alertType": {"name": "Alert", "color": "yellow"},
			"eventTime": 1616621253472,
			"headline": "Test",
			"firstAlertURL": "https://example.com",
			"estimatedEventLocation": [
				"Southern Hemisphere Location",
				-33.8688,
				151.2093,
				10.5
			]
		}`

		var alert Alert
		err := json.Unmarshal([]byte(jsonData), &alert)
		require.NoError(t, err)

		require.NotNil(t, alert.Location)
		assert.Equal(t, -33.8688, alert.Location.Latitude)
		assert.Equal(t, 151.2093, alert.Location.Longitude)
	})
}

func TestAlert_CombinedTimeAndLocation(t *testing.T) {
	t.Run("parses both time and location correctly", func(t *testing.T) {
		// Create known time
		expectedTime := time.Date(2021, 3, 24, 20, 27, 33, 472000000, time.UTC)
		eventTimeMs := expectedTime.UnixNano() / int64(time.Millisecond)

		jsonData := fmt.Sprintf(`{
			"alertId": "complete-test",
			"alertType": {"name": "Flash", "color": "red"},
			"eventTime": %d,
			"headline": "Complete test alert",
			"firstAlertURL": "https://example.com",
			"estimatedEventLocation": [
				"Test Location",
				40.7128,
				-74.0060,
				2.5,
				"MGRS123"
			]
		}`, eventTimeMs)

		var alert Alert
		err := json.Unmarshal([]byte(jsonData), &alert)
		require.NoError(t, err)

		// Verify time
		assert.Equal(t, expectedTime, alert.EventTime)

		// Verify location
		require.NotNil(t, alert.Location)
		assert.Equal(t, "Test Location", alert.Location.Address)
		assert.Equal(t, 40.7128, alert.Location.Latitude)
		assert.Equal(t, -74.0060, alert.Location.Longitude)
		assert.Equal(t, 2.5, alert.Location.ConfidenceRadiusMiles)
		assert.Equal(t, "MGRS123", alert.Location.MGRSCode)
	})
}
