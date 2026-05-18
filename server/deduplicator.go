// Copyright (c) 2025-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/mattermost/mattermost/server/public/pluginapi"
)

const (
	// DeduplicationCacheTTL is how long to keep alert IDs in the deduplication cache
	DeduplicationCacheTTL = 24 * time.Hour

	// DeduplicationCleanupInterval is how often to clean up expired entries
	DeduplicationCleanupInterval = 10 * time.Minute
)

// Deduplicator tracks seen alert IDs to prevent duplicate processing across all backends
type Deduplicator struct {
	api         *pluginapi.Client
	seenAlerts  map[string]time.Time
	mu          sync.RWMutex
	stopCleanup chan struct{}
	cleanupDone chan struct{}
}

// NewDeduplicator creates a new deduplicator and starts the cleanup loop
func NewDeduplicator(api *pluginapi.Client) *Deduplicator {
	d := &Deduplicator{
		api:         api,
		seenAlerts:  make(map[string]time.Time),
		stopCleanup: make(chan struct{}),
		cleanupDone: make(chan struct{}),
	}

	go d.cleanupLoop()

	return d
}

// RecordAlert atomically checks if an alert is new and marks it as seen if so.
// Returns true if this is a new alert (successfully recorded), false if it's a duplicate.
// This operation is atomic to prevent race conditions between checking and marking.
func (d *Deduplicator) RecordAlert(backendType, alertID string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()

	namespacedID := d.namespaceAlertID(backendType, alertID)

	// Check if already seen
	if _, exists := d.seenAlerts[namespacedID]; exists {
		return false // Duplicate
	}

	// Mark as seen
	d.seenAlerts[namespacedID] = time.Now()
	return true // New alert
}

// namespaceAlertID creates a namespaced alert ID to prevent collisions between backend types
func (d *Deduplicator) namespaceAlertID(backendType, alertID string) string {
	return fmt.Sprintf("%s:%s", backendType, alertID)
}

// cleanupLoop periodically removes expired entries from the cache
func (d *Deduplicator) cleanupLoop() {
	ticker := time.NewTicker(DeduplicationCleanupInterval)
	defer ticker.Stop()
	defer close(d.cleanupDone)

	for {
		select {
		case <-ticker.C:
			d.cleanup()
		case <-d.stopCleanup:
			return
		}
	}
}

// cleanup removes entries older than DeduplicationCacheTTL
func (d *Deduplicator) cleanup() {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := time.Now()
	expired := 0

	for alertID, seenTime := range d.seenAlerts {
		if now.Sub(seenTime) > DeduplicationCacheTTL {
			delete(d.seenAlerts, alertID)
			expired++
		}
	}

	if expired > 0 {
		d.api.Log.Debug("Cleaned up expired deduplication cache entries",
			"expired", expired,
			"remaining", len(d.seenAlerts))
	}
}

// Stop stops the cleanup goroutine and waits for it to finish
func (d *Deduplicator) Stop() {
	close(d.stopCleanup)
	<-d.cleanupDone
}
