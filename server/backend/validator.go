// Copyright (c) 2025-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package backend

import (
	"fmt"
	"net/url"

	"github.com/google/uuid"
)

// SupportedBackendTypes lists all backend types this plugin supports
var SupportedBackendTypes = map[string]bool{
	"dataminr": true,
}

// ValidateBackends validates backend configurations.
// This performs all validation steps defined in the specification.
func ValidateBackends(configs []Config) error {
	if len(configs) == 0 {
		// Empty configuration is valid - no backends configured
		return nil
	}

	// Step 2-8: Validate each backend and check for duplicates
	seenIDs := make(map[string]bool)
	seenNames := make(map[string]bool)

	for i, config := range configs {
		// Step 2: Required fields
		if err := validateRequiredFields(config); err != nil {
			return fmt.Errorf("backend configuration at position %d: %w", i+1, err)
		}

		// Step 3: UUID format
		if err := validateUUID(config.ID); err != nil {
			return fmt.Errorf("backend '%s': %w", config.Name, err)
		}

		// Step 4: Duplicate IDs
		if seenIDs[config.ID] {
			return fmt.Errorf("duplicate backend ID found: %s", config.ID)
		}
		seenIDs[config.ID] = true

		// Step 5: Duplicate names
		if seenNames[config.Name] {
			return fmt.Errorf("duplicate backend name found: '%s'", config.Name)
		}
		seenNames[config.Name] = true

		// Step 6: Type support
		if !SupportedBackendTypes[config.Type] {
			return fmt.Errorf("backend '%s': unsupported type '%s' (only 'dataminr' is currently supported)", config.Name, config.Type)
		}

		// Step 7: URL format
		if err := validateURL(config.URL); err != nil {
			return fmt.Errorf("backend '%s': %w", config.Name, err)
		}

		// Step 8: Poll interval minimum
		if config.PollIntervalSeconds < MinPollIntervalSeconds {
			return fmt.Errorf("backend '%s': poll interval must be at least %d seconds (got %d)",
				config.Name, MinPollIntervalSeconds, config.PollIntervalSeconds)
		}
	}

	return nil
}

// validateRequiredFields checks that all required fields are present and non-empty
func validateRequiredFields(config Config) error {
	if config.ID == "" {
		return fmt.Errorf("missing required field 'id'")
	}
	if config.Name == "" {
		return fmt.Errorf("missing required field 'name'")
	}
	if config.Type == "" {
		return fmt.Errorf("missing required field 'type'")
	}
	if config.URL == "" {
		return fmt.Errorf("missing required field 'url'")
	}
	if config.APIId == "" {
		return fmt.Errorf("missing required field 'apiId'")
	}
	if config.APIKey == "" {
		return fmt.Errorf("missing required field 'apiKey'")
	}
	if config.ChannelID == "" {
		return fmt.Errorf("missing required field 'channelId'")
	}
	if config.PollIntervalSeconds == 0 {
		return fmt.Errorf("missing required field 'pollIntervalSeconds'")
	}
	return nil
}

// validateUUID checks that the ID is a valid UUID v4
func validateUUID(id string) error {
	parsed, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid UUID format for id: %w", err)
	}

	// Only accept UUID v4
	if parsed.Version() != 4 {
		return fmt.Errorf("id must be a UUID v4 (got version %d)", parsed.Version())
	}

	return nil
}

// validateURL checks that the URL is valid and uses HTTPS
func validateURL(rawURL string) error {
	if rawURL == "" {
		return fmt.Errorf("url cannot be empty")
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid url format: %w", err)
	}

	if parsed.Scheme != "https" {
		return fmt.Errorf("url must use HTTPS (got %s)", parsed.Scheme)
	}

	if parsed.Host == "" {
		return fmt.Errorf("url must include a hostname")
	}

	return nil
}

// DiffBackendConfigs compares old and new backend configurations and returns IDs to add, update, and remove.
// Returns three slices: toAdd (new backend IDs), toUpdate (modified backend IDs), toRemove (deleted backend IDs).
func DiffBackendConfigs(oldConfigs, newConfigs []Config) (toAdd, toUpdate, toRemove []string) {
	oldMap := make(map[string]Config)
	newMap := make(map[string]Config)

	for _, cfg := range oldConfigs {
		oldMap[cfg.ID] = cfg
	}

	for _, cfg := range newConfigs {
		newMap[cfg.ID] = cfg
	}

	// Find backends to add or update
	for id, newCfg := range newMap {
		if oldCfg, exists := oldMap[id]; !exists {
			toAdd = append(toAdd, id)
		} else if oldCfg != newCfg {
			// Direct struct comparison works since all fields are primitive types
			toUpdate = append(toUpdate, id)
		}
	}

	// Find backends to remove
	for id := range oldMap {
		if _, exists := newMap[id]; !exists {
			toRemove = append(toRemove, id)
		}
	}

	return toAdd, toUpdate, toRemove
}
