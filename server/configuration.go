// Copyright (c) 2025-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"reflect"

	"github.com/mattermost/mattermost/server/public/plugin"
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-plugin-dataminr/server/backend"
)

// configuration captures the plugin's external configuration as exposed in the Mattermost server
// configuration, as well as values computed from the configuration. Any public fields will be
// deserialized from the Mattermost server configuration in OnConfigurationChange.
//
// As plugins are inherently concurrent (hooks being called asynchronously), and the plugin
// configuration can change at any time, access to the configuration must be synchronized. The
// strategy used in this plugin is to guard a pointer to the configuration, and clone the entire
// struct whenever it changes. You may replace this with whatever strategy you choose.
//
// If you add non-reference types to your configuration struct, be sure to rewrite Clone as a deep
// copy appropriate for your types.
type configuration struct {
	// BotUsername is the username for the alert notification bot.
	BotUsername string `json:"botUsername"`

	// BotDisplayName is the display name for the alert notification bot.
	BotDisplayName string `json:"botDisplayName"`

	// Backends is an array of backend configurations.
	// Each backend defines a separate alert source to poll and monitor.
	Backends []backend.Config `json:"backends"`
}

// Clone creates a deep copy of the configuration.
// This ensures that slice modifications don't affect the original.
func (c *configuration) Clone() *configuration {
	clone := *c

	// Deep copy the Backends slice
	if c.Backends != nil {
		clone.Backends = make([]backend.Config, len(c.Backends))
		copy(clone.Backends, c.Backends)
	}

	return &clone
}

// getConfiguration retrieves the active configuration under lock, making it safe to use
// concurrently. The active configuration may change underneath the client of this method, but
// the struct returned by this API call is considered immutable.
func (p *Plugin) getConfiguration() *configuration {
	p.configurationLock.RLock()
	defer p.configurationLock.RUnlock()

	if p.configuration == nil {
		return &configuration{}
	}

	return p.configuration
}

// setConfiguration replaces the active configuration under lock.
//
// Do not call setConfiguration while holding the configurationLock, as sync.Mutex is not
// reentrant. In particular, avoid using the plugin API entirely, as this may in turn trigger a
// hook back into the plugin. If that hook attempts to acquire this lock, a deadlock may occur.
//
// This method panics if setConfiguration is called with the existing configuration. This almost
// certainly means that the configuration was modified without being cloned and may result in
// an unsafe access.
func (p *Plugin) setConfiguration(configuration *configuration) {
	p.configurationLock.Lock()
	defer p.configurationLock.Unlock()

	if configuration != nil && p.configuration == configuration {
		// Ignore assignment if the configuration struct is empty. Go will optimize the
		// allocation for same to point at the same memory address, breaking the check
		// above.
		if reflect.ValueOf(*configuration).NumField() == 0 {
			return
		}

		panic("setConfiguration called with the existing configuration")
	}

	p.configuration = configuration
}

// findBackendConfigByID finds a backend configuration by ID in a slice of configs.
// Returns the config and true if found, or an empty config and false if not found.
func findBackendConfigByID(configs []backend.Config, id string) (backend.Config, bool) {
	for _, cfg := range configs {
		if cfg.ID == id {
			return cfg, true
		}
	}
	return backend.Config{}, false
}

// unregisterBackend unregisters a backend from the registry and logs the result.
func unregisterBackend(registry *backend.Registry, api plugin.API, id string, reason string) {
	if err := registry.Unregister(id); err != nil {
		api.LogWarn("Failed to unregister backend", "id", id, "reason", reason, "error", err.Error())
	} else {
		api.LogInfo("Unregistered backend", "id", id, "reason", reason)
	}
}

// OnConfigurationChange is invoked when configuration changes may have been made.
func (p *Plugin) OnConfigurationChange() error {
	newConfig := new(configuration)

	// Load the public configuration fields from the Mattermost server configuration.
	if err := p.API.LoadPluginConfiguration(newConfig); err != nil {
		return errors.Wrap(err, "failed to load plugin configuration")
	}

	// Validate backend configurations
	if err := backend.ValidateBackends(newConfig.Backends); err != nil {
		return errors.Wrap(err, "invalid backend configuration")
	}

	// Get old configuration for comparison
	oldConfig := p.getConfiguration()

	// Determine which backends need to be added, updated, or removed
	toAdd, toUpdate, toRemove := backend.DiffBackendConfigs(oldConfig.Backends, newConfig.Backends)

	// Update the configuration before managing backends
	p.setConfiguration(newConfig)

	// Handle backend lifecycle changes
	if p.registry != nil {
		// Remove deleted backends
		for _, id := range toRemove {
			unregisterBackend(p.registry, p.API, id, "backend removed from configuration")
		}

		// Update modified backends (stop old, start new)
		for _, id := range toUpdate {
			unregisterBackend(p.registry, p.API, id, "backend configuration changed")
			if cfg, found := findBackendConfigByID(newConfig.Backends, id); found {
				p.createAndStartBackend(cfg)
			}
		}

		// Add new backends
		for _, id := range toAdd {
			if cfg, found := findBackendConfigByID(newConfig.Backends, id); found {
				p.createAndStartBackend(cfg)
			}
		}
	}

	return nil
}
