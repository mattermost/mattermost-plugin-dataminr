package backend

import (
	"fmt"

	"github.com/mattermost/mattermost/server/public/plugin"
	"github.com/mattermost/mattermost/server/public/pluginapi"
)

// AlertPoster is an interface for posting alerts to Mattermost channels.
// This abstraction allows backends to post alerts without directly depending on the poster package.
type AlertPoster interface {
	PostAlert(alert Alert, channelID string) error
}

// Deduplicator is an interface for tracking seen alerts per destination Mattermost channel
// across backends (shared cache), so duplicate posts to the same channel are suppressed.
type Deduplicator interface {
	// RecordAlert atomically checks if an alert is new for the given destination channel and marks it as seen if so.
	// Returns true if this is a new alert (successfully recorded), false if it's a duplicate.
	RecordAlert(backendType, alertID, channelID string) bool
}

// DisableCallback is a function type for disabling a backend when it reaches MaxConsecutiveFailures.
// The callback receives the backend ID and should persist the configuration change.
type DisableCallback func(backendID string) error

// Factory is a function type that creates a backend instance
type Factory func(config Config, api *pluginapi.Client, papi plugin.API, poster AlertPoster, deduplicator Deduplicator, disableCallback DisableCallback) (Backend, error)

// factoryRegistry maps backend types to their factory functions
var factoryRegistry = make(map[string]Factory)

// RegisterBackendFactory registers a backend factory for a given type.
// This allows backends to register themselves for creation.
func RegisterBackendFactory(backendType string, factory Factory) {
	factoryRegistry[backendType] = factory
}

// Create creates a new backend instance based on the provided configuration.
// Returns an error if the backend type is unknown or if creation fails.
func Create(config Config, api *pluginapi.Client, papi plugin.API, poster AlertPoster, deduplicator Deduplicator, disableCallback DisableCallback) (Backend, error) {
	if config.Type == "" {
		return nil, fmt.Errorf("backend type is required")
	}

	factory, exists := factoryRegistry[config.Type]
	if !exists {
		return nil, fmt.Errorf("unknown backend type: %s", config.Type)
	}

	return factory(config, api, papi, poster, deduplicator, disableCallback)
}
