// Copyright (c) 2025-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package dataminr

import (
	"encoding/json"
	"fmt"
	"time"
)

// AuthRequest represents the authentication request payload
type AuthRequest struct {
	GrantType   string `json:"grant_type"`
	Scope       string `json:"scope"`
	APIUserID   string `json:"api_user_id"`
	APIPassword string `json:"api_password"`
}

// AuthResponse represents the authentication response from Dataminr API
type AuthResponse struct {
	AuthorizationToken string    `json:"authorizationToken"`
	ExpirationTime     time.Time `json:"-"` // Parsed from expirationTime milliseconds
	TOS                string    `json:"TOS"`
	ThirdPartyTerms    string    `json:"thirdPartyTerms"`
}

// UnmarshalJSON implements custom JSON unmarshaling for AuthResponse
// Converts expirationTime from milliseconds to time.Time
func (a *AuthResponse) UnmarshalJSON(data []byte) error {
	// Create type alias to avoid infinite recursion when calling json.Unmarshal
	// Without this alias, calling json.Unmarshal on AuthResponse would invoke this method again
	type Alias AuthResponse
	aux := &struct {
		ExpirationTimeMs int64 `json:"expirationTime"`
		*Alias
	}{
		Alias: (*Alias)(a),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// Convert milliseconds to time.Time in UTC
	a.ExpirationTime = time.Unix(0, aux.ExpirationTimeMs*int64(time.Millisecond)).UTC()
	return nil
}

// AuthError represents authentication error response
// Format: {"Error": "unauthorized_client", "error_description": "Invalid API User ID/Password"}
type AuthError struct {
	Error            string `json:"Error"`
	ErrorDescription string `json:"error_description"`
}

// APIError represents API error response
// Format: {"Error": [{"Code": "103", "message": "Authentication error. Invalid token"}]}
type APIError struct {
	Errors []ErrorDetail `json:"Error"`
}

// ErrorDetail represents details of an API error
type ErrorDetail struct {
	Code    string `json:"Code"`
	Message string `json:"message"`
}

// Error implements the error interface for APIError
func (e *APIError) Error() string {
	if len(e.Errors) > 0 {
		return fmt.Sprintf("Code %s: %s", e.Errors[0].Code, e.Errors[0].Message)
	}
	return "unknown API error"
}

// AlertsResponse represents the response from polling alerts endpoint
type AlertsResponse struct {
	Alerts []Alert `json:"alerts"`
	To     string  `json:"to"` // Cursor for next request
}

// Alert represents a complete alert object from Dataminr First Alert API
// All time and location fields are automatically parsed during JSON unmarshaling
type Alert struct {
	AlertID       string        `json:"alertId"`
	AlertType     AlertType     `json:"alertType"`
	EventTime     time.Time     `json:"-"` // Parsed from eventTime milliseconds
	Headline      string        `json:"headline"`
	PublicPost    *PublicPost   `json:"publicPost,omitempty"`
	FirstAlertURL string        `json:"firstAlertURL"`
	Location      *Location     `json:"-"` // Parsed from estimatedEventLocation array
	AlertLists    []AlertList   `json:"alertLists,omitempty"`
	AlertTopics   []AlertTopic  `json:"alertTopics,omitempty"`
	LinkedAlerts  []LinkedAlert `json:"linkedAlerts,omitempty"`
	SubHeadline   *SubHeadline  `json:"subHeadline,omitempty"`
	TermsOfUse    string        `json:"termsOfUse,omitempty"`
}

// UnmarshalJSON implements custom JSON unmarshaling for Alert
// Parses eventTime from milliseconds and estimatedEventLocation from array
func (r *Alert) UnmarshalJSON(data []byte) error {
	// Create type alias to avoid infinite recursion when calling json.Unmarshal
	// Without this alias, calling json.Unmarshal on Alert would invoke this method again
	type Alias Alert
	aux := &struct {
		EventTimeMs int64 `json:"eventTime"`
		LocationRaw []any `json:"estimatedEventLocation"`
		*Alias
	}{
		Alias: (*Alias)(r),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// Parse event time from milliseconds to UTC
	if aux.EventTimeMs > 0 {
		r.EventTime = time.Unix(0, aux.EventTimeMs*int64(time.Millisecond)).UTC()
	}

	// Parse location from array format: [address, lat, lon, confidence, mgrs]
	if len(aux.LocationRaw) >= 4 {
		loc := &Location{}

		if len(aux.LocationRaw) > 0 {
			if addr, ok := aux.LocationRaw[0].(string); ok {
				loc.Address = addr
			}
		}
		if len(aux.LocationRaw) > 1 {
			if lat, ok := aux.LocationRaw[1].(float64); ok {
				loc.Latitude = lat
			}
		}
		if len(aux.LocationRaw) > 2 {
			if lon, ok := aux.LocationRaw[2].(float64); ok {
				loc.Longitude = lon
			}
		}
		if len(aux.LocationRaw) > 3 {
			if conf, ok := aux.LocationRaw[3].(float64); ok {
				loc.ConfidenceRadiusMiles = conf
			}
		}
		if len(aux.LocationRaw) > 4 {
			if mgrs, ok := aux.LocationRaw[4].(string); ok {
				loc.MGRSCode = mgrs
			}
		}

		r.Location = loc
	}

	return nil
}

// AlertType represents the type and color of an alert
type AlertType struct {
	Name  string `json:"name"`  // Flash, Urgent, Alert
	Color string `json:"color"` // red, orange, yellow
}

// PublicPost represents the public source information (social media post)
type PublicPost struct {
	Link           string   `json:"link"`
	Text           string   `json:"text"`
	TranslatedText string   `json:"translatedText,omitempty"`
	Media          []string `json:"media,omitempty"` // URLs to images/media
}

// AlertTopic represents a topic associated with an alert
type AlertTopic struct {
	Name string `json:"name"`
	ID   string `json:"id"`
}

// AlertList represents an alert list that this alert belongs to
type AlertList struct {
	Name string `json:"name"`
}

// LinkedAlert represents linked/related alerts
type LinkedAlert struct {
	Count    int    `json:"count"`
	ParentID string `json:"parentId"`
}

// SubHeadline represents additional contextual information
type SubHeadline struct {
	Title        string `json:"title"`
	SubHeadlines string `json:"subHeadlines"` // Additional context text
}

// Location represents the parsed location data from Dataminr alerts
type Location struct {
	Address               string
	Latitude              float64
	Longitude             float64
	ConfidenceRadiusMiles float64 // Confidence radius in miles
	MGRSCode              string  // Military Grid Reference System code
}
