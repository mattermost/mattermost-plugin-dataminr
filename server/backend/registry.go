// Copyright (c) 2025-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package backend

import (
	"fmt"
	"sync"
)

// Registry manages all active backend instances.
// It provides thread-safe operations for registering, retrieving, and managing backends.
type Registry struct {
	mu       sync.RWMutex
	backends map[string]Backend
}

// NewRegistry creates a new backend registry.
func NewRegistry() *Registry {
	return &Registry{
		backends: make(map[string]Backend),
	}
}

// Register adds a backend to the registry.
// Returns an error if a backend with the same ID already exists.
func (r *Registry) Register(backend Backend) error {
	if backend == nil {
		return fmt.Errorf("cannot register nil backend")
	}

	id := backend.GetID()
	if id == "" {
		return fmt.Errorf("backend ID cannot be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.backends[id]; exists {
		return fmt.Errorf("backend with ID %s already registered", id)
	}

	r.backends[id] = backend
	return nil
}

// Unregister removes a backend from the registry and stops it.
// Returns an error if the backend doesn't exist or cannot be stopped.
// The backend is always removed from the registry, even if Stop fails.
func (r *Registry) Unregister(id string) error {
	r.mu.Lock()
	backend, exists := r.backends[id]
	if !exists {
		r.mu.Unlock()
		return fmt.Errorf("backend with ID %s not found", id)
	}

	// Remove the backend from the registry first
	delete(r.backends, id)
	r.mu.Unlock()

	// Stop the backend after releasing the lock to avoid blocking other registry operations
	if err := backend.Stop(); err != nil {
		return fmt.Errorf("failed to stop backend %s: %w", id, err)
	}

	return nil
}

// Get retrieves a backend by its ID.
// Returns nil if the backend doesn't exist.
func (r *Registry) Get(id string) Backend {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.backends[id]
}

// List returns all registered backends.
// Returns a copy of the backend slice to avoid race conditions.
func (r *Registry) List() []Backend {
	r.mu.RLock()
	defer r.mu.RUnlock()

	backends := make([]Backend, 0, len(r.backends))
	for _, backend := range r.backends {
		backends = append(backends, backend)
	}

	return backends
}

// UnregisterAll unregisters and stops all registered backends.
// Returns the first error encountered, but continues unregistering remaining backends.
func (r *Registry) UnregisterAll() error {
	r.mu.Lock()
	// Get all backends and clear the registry
	backends := make([]Backend, 0, len(r.backends))
	for id, backend := range r.backends {
		backends = append(backends, backend)
		delete(r.backends, id)
	}
	r.mu.Unlock()

	// Stop all backends after releasing the lock
	var firstError error
	for _, backend := range backends {
		if err := backend.Stop(); err != nil && firstError == nil {
			firstError = fmt.Errorf("failed to stop backend %s: %w", backend.GetID(), err)
		}
	}

	return firstError
}

// Count returns the number of registered backends.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.backends)
}
