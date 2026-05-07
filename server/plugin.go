package main

import (
	"encoding/json"
	"sync"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
	"github.com/mattermost/mattermost/server/public/pluginapi"
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-plugin-dataminr/server/backend"
	_ "github.com/mattermost/mattermost-plugin-dataminr/server/backend/dataminr" // Register dataminr backend factory
	"github.com/mattermost/mattermost-plugin-dataminr/server/poster"
)

// Plugin implements the interface expected by the Mattermost server to communicate between the server and plugin processes.
type Plugin struct {
	plugin.MattermostPlugin

	// client is the Mattermost server API client.
	client *pluginapi.Client

	// configurationLock synchronizes access to the configuration.
	configurationLock sync.RWMutex

	// configuration is the active plugin configuration. Consult getConfiguration and
	// setConfiguration for usage.
	configuration *configuration

	// registry manages all active backend instances.
	registry *backend.Registry

	// poster posts alerts to Mattermost channels.
	poster backend.AlertPoster

	// deduplicator is shared across all backends to prevent duplicate alerts
	deduplicator *Deduplicator
}

// OnActivate is invoked when the plugin is activated. If an error is returned, the plugin will be deactivated.
func (p *Plugin) OnActivate() error {
	p.client = pluginapi.NewClient(p.API, p.Driver)
	p.registry = backend.NewRegistry()
	p.deduplicator = NewDeduplicator(p.client)

	// Check license
	if !pluginapi.IsEnterpriseLicensedOrDevelopment(p.API.GetConfig(), p.API.GetLicense()) {
		err := errors.New("this plugin requires an Enterprise license")
		p.API.LogError("Cannot initialize plugin", "err", err)
		return err
	}

	// Get configuration
	config := p.getConfiguration()

	// Ensure bot user exists
	botUsername := config.BotUsername
	if botUsername == "" {
		botUsername = "dataminr-alerts"
	}
	botDisplayName := config.BotDisplayName
	if botDisplayName == "" {
		botDisplayName = "Dataminr Alerts"
	}

	botID, err := p.client.Bot.EnsureBot(&model.Bot{
		Username:    botUsername,
		DisplayName: botDisplayName,
		Description: "Bot for posting Dataminr alerts to Mattermost channels",
	}, pluginapi.ProfileImagePath("assets/profile.png"))
	if err != nil {
		return errors.Wrap(err, "failed to ensure bot user")
	}

	p.API.LogInfo("Bot user initialized", "botID", botID, "username", botUsername)

	// Create poster with bot ID
	p.poster = poster.New(p.API, botID)

	// Initialize backends from current configuration
	for _, backendConfig := range config.Backends {
		p.createAndStartBackend(backendConfig)
	}

	return nil
}

// OnDeactivate is invoked when the plugin is deactivated.
func (p *Plugin) OnDeactivate() error {
	if p.registry != nil {
		if err := p.registry.UnregisterAll(); err != nil {
			p.API.LogError("Failed to unregister all backends during deactivation", "error", err.Error())
			return err
		}
	}

	if p.deduplicator != nil {
		p.deduplicator.Stop()
	}

	return nil
}

// createAndStartBackend creates a backend instance and registers it.
// If the backend is enabled, it also starts the backend.
// Logs errors but does not fail - errors are non-fatal for individual backends.
func (p *Plugin) createAndStartBackend(config backend.Config) {
	// Create backend instance using factory, passing the shared deduplicator and disable callback
	b, err := backend.Create(config, p.client, p.API, p.poster, p.deduplicator, p.disableBackend)
	if err != nil {
		p.API.LogError("Failed to create backend", "id", config.ID, "name", config.Name, "error", err.Error())
		return
	}

	// Register backend (always register, even if disabled)
	if err := p.registry.Register(b); err != nil {
		p.API.LogError("Failed to register backend", "id", config.ID, "name", config.Name, "error", err.Error())
		return
	}

	// Only start the backend if it's enabled
	if !config.Enabled {
		// Clear cursor and auth token for disabled backends to ensure fresh start when re-enabled
		// This preserves failure tracking state for status display
		if err := b.ClearOperationalState(); err != nil {
			p.API.LogWarn("Failed to clear operational state for disabled backend", "id", config.ID, "name", config.Name, "error", err.Error())
		} else {
			p.API.LogInfo("Cleared operational state for disabled backend", "id", config.ID, "name", config.Name)
		}
		p.API.LogInfo("Backend registered but not started (disabled)", "id", config.ID, "name", config.Name)
		return
	}

	// Start backend
	if err := b.Start(); err != nil {
		p.API.LogError("Failed to start backend", "id", config.ID, "name", config.Name, "error", err.Error())
		// Keep backend registered even if start fails - it will show error state in status
		return
	}

	p.API.LogInfo("Backend started successfully", "id", config.ID, "name", config.Name, "type", config.Type)
}

// disableBackend sets a backend's enabled flag to false and persists the configuration change.
// This is called when a backend reaches MaxConsecutiveFailures and needs to be auto-disabled.
// The configuration change will trigger OnConfigurationChange, which will stop the backend.
func (p *Plugin) disableBackend(backendID string) error {
	// Get the current configuration (handles locking internally)
	config := p.getConfiguration()

	// Clone it to avoid modifying the active configuration
	configClone := config.Clone()

	// Find the backend in the cloned configuration
	found := false
	var backendName string
	for i := range configClone.Backends {
		if configClone.Backends[i].ID == backendID {
			// Set enabled to false
			configClone.Backends[i].Enabled = false
			backendName = configClone.Backends[i].Name
			found = true
			break
		}
	}

	if !found {
		return errors.Errorf("backend with ID %s not found in configuration", backendID)
	}

	p.API.LogInfo("Disabling backend in configuration", "id", backendID, "name", backendName)

	// Marshal the configuration to map[string]any for SavePluginConfig
	marshalBytes, err := json.Marshal(configClone)
	if err != nil {
		return errors.Wrap(err, "failed to marshal configuration")
	}

	configMap := make(map[string]any)
	if err := json.Unmarshal(marshalBytes, &configMap); err != nil {
		return errors.Wrap(err, "failed to unmarshal configuration to map")
	}

	// Persist the configuration change (this will trigger OnConfigurationChange)
	if err := p.client.Configuration.SavePluginConfig(configMap); err != nil {
		return errors.Wrap(err, "failed to save plugin configuration")
	}

	p.API.LogInfo("Backend disabled and configuration persisted", "id", backendID)
	return nil
}

// See https://developers.mattermost.com/extend/plugins/server/reference/
