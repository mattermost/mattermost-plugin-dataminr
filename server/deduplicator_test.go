// Copyright (c) 2025-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"strconv"
	"testing"
	"time"

	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/mattermost/mattermost/server/public/pluginapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const testDedupChannelID = "ch1"

func TestDeduplicator(t *testing.T) {
	t.Run("new alert is recorded successfully", func(t *testing.T) {
		api := plugintest.NewAPI(t)
		client := pluginapi.NewClient(api, &plugintest.Driver{})

		dedup := NewDeduplicator(client)
		defer dedup.Stop()

		isNew := dedup.RecordAlert("dataminr", "alert-1", testDedupChannelID)
		assert.True(t, isNew, "First occurrence should be recorded as new")
	})

	t.Run("duplicate alert is rejected", func(t *testing.T) {
		api := plugintest.NewAPI(t)
		client := pluginapi.NewClient(api, &plugintest.Driver{})

		dedup := NewDeduplicator(client)
		defer dedup.Stop()

		// Record first time
		isNew := dedup.RecordAlert("dataminr", "alert-1", testDedupChannelID)
		assert.True(t, isNew, "First occurrence should be recorded as new")

		// Try to record again
		isNew = dedup.RecordAlert("dataminr", "alert-1", testDedupChannelID)
		assert.False(t, isNew, "Second occurrence should be rejected as duplicate")
	})

	t.Run("same alert to different channels is not duplicate", func(t *testing.T) {
		api := plugintest.NewAPI(t)
		client := pluginapi.NewClient(api, &plugintest.Driver{})

		dedup := NewDeduplicator(client)
		defer dedup.Stop()

		assert.True(t, dedup.RecordAlert("dataminr", "alert-1", "channel-a"))
		assert.False(t, dedup.RecordAlert("dataminr", "alert-1", "channel-a"))
		assert.True(t, dedup.RecordAlert("dataminr", "alert-1", "channel-b"))
		assert.False(t, dedup.RecordAlert("dataminr", "alert-1", "channel-b"))
	})

	t.Run("backend type namespacing prevents collisions", func(t *testing.T) {
		api := plugintest.NewAPI(t)
		client := pluginapi.NewClient(api, &plugintest.Driver{})

		dedup := NewDeduplicator(client)
		defer dedup.Stop()

		// Record alert for dataminr backend
		isNew := dedup.RecordAlert("dataminr", "alert-1", testDedupChannelID)
		assert.True(t, isNew, "Alert from dataminr should be new")

		// Same alert ID from different backend should also be new
		isNew = dedup.RecordAlert("other-backend", "alert-1", testDedupChannelID)
		assert.True(t, isNew, "Same alert ID from different backend should be new")

		// Second attempt for dataminr should fail
		isNew = dedup.RecordAlert("dataminr", "alert-1", testDedupChannelID)
		assert.False(t, isNew, "Duplicate alert from dataminr should be rejected")

		// Second attempt for other backend should also fail
		isNew = dedup.RecordAlert("other-backend", "alert-1", testDedupChannelID)
		assert.False(t, isNew, "Duplicate alert from other backend should be rejected")
	})

	t.Run("multiple different alerts", func(t *testing.T) {
		api := plugintest.NewAPI(t)
		client := pluginapi.NewClient(api, &plugintest.Driver{})

		dedup := NewDeduplicator(client)
		defer dedup.Stop()

		// Record several alerts
		assert.True(t, dedup.RecordAlert("dataminr", "alert-1", testDedupChannelID))
		assert.True(t, dedup.RecordAlert("dataminr", "alert-2", testDedupChannelID))
		assert.True(t, dedup.RecordAlert("dataminr", "alert-3", testDedupChannelID))

		// All should be rejected on second attempt
		assert.False(t, dedup.RecordAlert("dataminr", "alert-1", testDedupChannelID))
		assert.False(t, dedup.RecordAlert("dataminr", "alert-2", testDedupChannelID))
		assert.False(t, dedup.RecordAlert("dataminr", "alert-3", testDedupChannelID))

		// New alert should be accepted
		assert.True(t, dedup.RecordAlert("dataminr", "alert-4", testDedupChannelID))
	})

	t.Run("cleanup removes expired entries", func(t *testing.T) {
		api := plugintest.NewAPI(t)
		api.On("LogDebug", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
		client := pluginapi.NewClient(api, &plugintest.Driver{})

		dedup := NewDeduplicator(client)
		defer dedup.Stop()

		// Record an alert
		isNew := dedup.RecordAlert("dataminr", "alert-1", testDedupChannelID)
		assert.True(t, isNew)

		// Manually set the seen time to be older than TTL (25 hours, TTL is 24 hours)
		dedup.mu.Lock()
		dedup.seenAlerts["dataminr:alert-1:"+testDedupChannelID] = time.Now().Add(-25 * time.Hour)
		dedup.mu.Unlock()

		// Run cleanup
		dedup.cleanup()

		// Alert should be accepted again (expired entry was removed)
		isNew = dedup.RecordAlert("dataminr", "alert-1", testDedupChannelID)
		assert.True(t, isNew, "Alert should be new again after expiration")
	})

	t.Run("cleanup keeps recent entries", func(t *testing.T) {
		api := plugintest.NewAPI(t)
		client := pluginapi.NewClient(api, &plugintest.Driver{})

		dedup := NewDeduplicator(client)
		defer dedup.Stop()

		// Record an alert
		isNew := dedup.RecordAlert("dataminr", "alert-1", testDedupChannelID)
		assert.True(t, isNew)

		// Run cleanup (should not remove recent entry)
		dedup.cleanup()

		// Recent alert should still be rejected as duplicate
		isNew = dedup.RecordAlert("dataminr", "alert-1", testDedupChannelID)
		assert.False(t, isNew, "Recent alert should still be duplicate")
	})

	t.Run("cleanup with mixed expiration", func(t *testing.T) {
		api := plugintest.NewAPI(t)
		api.On("LogDebug", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
		client := pluginapi.NewClient(api, &plugintest.Driver{})

		dedup := NewDeduplicator(client)
		defer dedup.Stop()

		// Add recent alerts
		assert.True(t, dedup.RecordAlert("dataminr", "alert-recent-1", testDedupChannelID))
		assert.True(t, dedup.RecordAlert("dataminr", "alert-recent-2", testDedupChannelID))

		// Add old alerts
		dedup.mu.Lock()
		dedup.seenAlerts["dataminr:alert-old-1:"+testDedupChannelID] = time.Now().Add(-25 * time.Hour)
		dedup.seenAlerts["dataminr:alert-old-2:"+testDedupChannelID] = time.Now().Add(-26 * time.Hour)
		dedup.mu.Unlock()

		// Run cleanup
		dedup.cleanup()

		// Recent alerts should still be rejected
		assert.False(t, dedup.RecordAlert("dataminr", "alert-recent-1", testDedupChannelID))
		assert.False(t, dedup.RecordAlert("dataminr", "alert-recent-2", testDedupChannelID))

		// Old alerts should be accepted again (expired entries removed)
		assert.True(t, dedup.RecordAlert("dataminr", "alert-old-1", testDedupChannelID))
		assert.True(t, dedup.RecordAlert("dataminr", "alert-old-2", testDedupChannelID))
	})

	t.Run("stop waits for cleanup goroutine", func(t *testing.T) {
		api := plugintest.NewAPI(t)
		client := pluginapi.NewClient(api, &plugintest.Driver{})

		dedup := NewDeduplicator(client)

		// Stop should not block indefinitely
		done := make(chan struct{})
		go func() {
			dedup.Stop()
			close(done)
		}()

		select {
		case <-done:
			// Success - Stop completed
		case <-time.After(1 * time.Second):
			t.Fatal("Stop() did not complete within timeout")
		}
	})

	t.Run("concurrent access is safe", func(t *testing.T) {
		api := plugintest.NewAPI(t)
		api.On("LogDebug", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
		client := pluginapi.NewClient(api, &plugintest.Driver{})

		dedup := NewDeduplicator(client)
		defer dedup.Stop()

		done := make(chan struct{})

		// Writer goroutine
		go func() {
			for i := range 100 {
				dedup.RecordAlert("dataminr", "alert-"+strconv.Itoa(i), testDedupChannelID)
			}
			done <- struct{}{}
		}()

		// Reader goroutine (also uses RecordAlert since it's atomic)
		go func() {
			for i := range 100 {
				dedup.RecordAlert("other-backend", "alert-"+strconv.Itoa(i), testDedupChannelID)
			}
			done <- struct{}{}
		}()

		// Wait for both goroutines
		<-done
		<-done

		// If we get here without a panic, concurrent access is safe
	})
}
