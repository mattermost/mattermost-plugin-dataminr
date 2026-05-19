// Copyright (c) 2025-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package backend

import "time"

// Constants for backend behavior and thresholds
const (
	// MaxConsecutiveFailures is the number of consecutive polling failures
	// before a backend is automatically disabled. This prevents runaway
	// error conditions and excessive API calls to failing backends.
	MaxConsecutiveFailures = 5

	// MinPollIntervalSeconds is the minimum allowed poll interval
	MinPollIntervalSeconds = 10

	// DefaultPollIntervalSeconds is the recommended default poll interval
	DefaultPollIntervalSeconds = 30

	// AuthTokenRefreshBuffer is how long before token expiry to refresh
	AuthTokenRefreshBuffer = 5 * time.Minute
)
