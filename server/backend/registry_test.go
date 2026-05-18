// Copyright (c) 2025-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package backend

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockBackend is a simple mock implementation of the Backend interface for testing
type mockBackend struct {
	id      string
	name    string
	typ     string
	stopped bool
	mu      sync.Mutex

	// Errors to return
	stopErr  error
	startErr error
}

func newMockBackend(id, name, typ string) *mockBackend {
	return &mockBackend{
		id:   id,
		name: name,
		typ:  typ,
	}
}

func (m *mockBackend) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.startErr
}

func (m *mockBackend) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stopped = true
	return m.stopErr
}

func (m *mockBackend) GetID() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.id
}

func (m *mockBackend) GetName() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.name
}

func (m *mockBackend) GetType() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.typ
}

func (m *mockBackend) GetStatus() Status {
	m.mu.Lock()
	defer m.mu.Unlock()
	return Status{}
}

func (m *mockBackend) ClearOperationalState() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return nil
}

func (m *mockBackend) isStopped() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.stopped
}

func TestNewRegistry(t *testing.T) {
	registry := NewRegistry()
	assert.NotNil(t, registry)
	assert.Equal(t, 0, registry.Count())
}

func TestRegistry_Register(t *testing.T) {
	registry := NewRegistry()
	backend := newMockBackend("backend1", "Test Backend", "dataminr")

	err := registry.Register(backend)
	assert.NoError(t, err)
	assert.Equal(t, 1, registry.Count())

	retrieved := registry.Get("backend1")
	assert.Equal(t, backend, retrieved)
}

func TestRegistry_RegisterNilBackend(t *testing.T) {
	registry := NewRegistry()

	err := registry.Register(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot register nil backend")
	assert.Equal(t, 0, registry.Count())
}

func TestRegistry_RegisterEmptyID(t *testing.T) {
	registry := NewRegistry()
	backend := newMockBackend("", "Test Backend", "dataminr")

	err := registry.Register(backend)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "backend ID cannot be empty")
	assert.Equal(t, 0, registry.Count())
}

func TestRegistry_RegisterDuplicateID(t *testing.T) {
	registry := NewRegistry()
	backend1 := newMockBackend("backend1", "Test Backend 1", "dataminr")
	backend2 := newMockBackend("backend1", "Test Backend 2", "dataminr")

	err := registry.Register(backend1)
	require.NoError(t, err)

	err = registry.Register(backend2)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
	assert.Equal(t, 1, registry.Count())

	// Verify the first backend is still registered
	retrieved := registry.Get("backend1")
	assert.Equal(t, backend1, retrieved)
}

func TestRegistry_Unregister(t *testing.T) {
	registry := NewRegistry()
	backend := newMockBackend("backend1", "Test Backend", "dataminr")

	err := registry.Register(backend)
	require.NoError(t, err)

	err = registry.Unregister("backend1")
	assert.NoError(t, err)
	assert.Equal(t, 0, registry.Count())
	assert.True(t, backend.isStopped())

	retrieved := registry.Get("backend1")
	assert.Nil(t, retrieved)
}

func TestRegistry_UnregisterNotFound(t *testing.T) {
	registry := NewRegistry()

	err := registry.Unregister("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestRegistry_UnregisterStopError(t *testing.T) {
	registry := NewRegistry()
	backend := newMockBackend("backend1", "Test Backend", "dataminr")
	backend.stopErr = fmt.Errorf("stop failed")

	err := registry.Register(backend)
	require.NoError(t, err)

	err = registry.Unregister("backend1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to stop backend")
	assert.Contains(t, err.Error(), "stop failed")

	// Backend should still be removed even if Stop failed
	assert.Equal(t, 0, registry.Count())
}

func TestRegistry_Get(t *testing.T) {
	registry := NewRegistry()
	backend1 := newMockBackend("backend1", "Test Backend 1", "dataminr")
	backend2 := newMockBackend("backend2", "Test Backend 2", "dataminr")

	err := registry.Register(backend1)
	require.NoError(t, err)
	err = registry.Register(backend2)
	require.NoError(t, err)

	retrieved := registry.Get("backend1")
	assert.Equal(t, backend1, retrieved)

	retrieved = registry.Get("backend2")
	assert.Equal(t, backend2, retrieved)

	retrieved = registry.Get("nonexistent")
	assert.Nil(t, retrieved)
}

func TestRegistry_List(t *testing.T) {
	registry := NewRegistry()

	// Empty registry
	list := registry.List()
	assert.NotNil(t, list)
	assert.Equal(t, 0, len(list))

	// Add backends
	backend1 := newMockBackend("backend1", "Test Backend 1", "dataminr")
	backend2 := newMockBackend("backend2", "Test Backend 2", "dataminr")
	backend3 := newMockBackend("backend3", "Test Backend 3", "dataminr")

	require.NoError(t, registry.Register(backend1))
	require.NoError(t, registry.Register(backend2))
	require.NoError(t, registry.Register(backend3))

	list = registry.List()
	assert.Equal(t, 3, len(list))

	// Verify all backends are in the list
	ids := make(map[string]bool)
	for _, b := range list {
		ids[b.GetID()] = true
	}
	assert.True(t, ids["backend1"])
	assert.True(t, ids["backend2"])
	assert.True(t, ids["backend3"])
}

func TestRegistry_UnregisterAll(t *testing.T) {
	registry := NewRegistry()

	backend1 := newMockBackend("backend1", "Test Backend 1", "dataminr")
	backend2 := newMockBackend("backend2", "Test Backend 2", "dataminr")
	backend3 := newMockBackend("backend3", "Test Backend 3", "dataminr")

	require.NoError(t, registry.Register(backend1))
	require.NoError(t, registry.Register(backend2))
	require.NoError(t, registry.Register(backend3))

	err := registry.UnregisterAll()
	assert.NoError(t, err)
	assert.Equal(t, 0, registry.Count())

	// Verify all backends were stopped
	assert.True(t, backend1.isStopped())
	assert.True(t, backend2.isStopped())
	assert.True(t, backend3.isStopped())
}

func TestRegistry_UnregisterAllWithErrors(t *testing.T) {
	registry := NewRegistry()

	backend1 := newMockBackend("backend1", "Test Backend 1", "dataminr")
	backend2 := newMockBackend("backend2", "Test Backend 2", "dataminr")
	backend2.stopErr = fmt.Errorf("backend2 stop failed")
	backend3 := newMockBackend("backend3", "Test Backend 3", "dataminr")

	require.NoError(t, registry.Register(backend1))
	require.NoError(t, registry.Register(backend2))
	require.NoError(t, registry.Register(backend3))

	err := registry.UnregisterAll()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to stop backend")

	// All backends should be removed despite errors
	assert.Equal(t, 0, registry.Count())
}

func TestRegistry_Count(t *testing.T) {
	registry := NewRegistry()
	assert.Equal(t, 0, registry.Count())

	backend1 := newMockBackend("backend1", "Test Backend 1", "dataminr")
	require.NoError(t, registry.Register(backend1))
	assert.Equal(t, 1, registry.Count())

	backend2 := newMockBackend("backend2", "Test Backend 2", "dataminr")
	require.NoError(t, registry.Register(backend2))
	assert.Equal(t, 2, registry.Count())

	require.NoError(t, registry.Unregister("backend1"))
	assert.Equal(t, 1, registry.Count())

	require.NoError(t, registry.Unregister("backend2"))
	assert.Equal(t, 0, registry.Count())
}
