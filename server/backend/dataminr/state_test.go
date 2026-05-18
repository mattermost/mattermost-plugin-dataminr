// Copyright (c) 2025-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package dataminr

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStateStore_AuthToken(t *testing.T) {
	t.Run("save and retrieve auth token", func(t *testing.T) {
		api := &plugintest.API{}
		backendID := "test-backend-123"
		store := NewStateStore(api, backendID)

		token := "test_token_abc123"
		expiry := time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC)

		// Mock KVSet
		expectedKey := "backend_test-backend-123_auth"
		expectedState := AuthTokenState{Token: token, Expiry: expiry}
		expectedData, _ := json.Marshal(expectedState)

		api.On("KVSet", expectedKey, expectedData).Return(nil)

		// Save
		err := store.SaveAuthToken(token, expiry)
		require.NoError(t, err)
		api.AssertExpectations(t)

		// Mock KVGet
		api.On("KVGet", expectedKey).Return(expectedData, nil)

		// Retrieve
		gotToken, gotExpiry, err := store.GetAuthToken()
		require.NoError(t, err)
		assert.Equal(t, token, gotToken)
		assert.Equal(t, expiry, gotExpiry)
		api.AssertExpectations(t)
	})

	t.Run("get auth token when none stored", func(t *testing.T) {
		api := &plugintest.API{}
		backendID := "test-backend-123"
		store := NewStateStore(api, backendID)

		expectedKey := "backend_test-backend-123_auth"
		api.On("KVGet", expectedKey).Return(nil, nil)

		token, expiry, err := store.GetAuthToken()
		require.NoError(t, err)
		assert.Empty(t, token)
		assert.True(t, expiry.IsZero())
		api.AssertExpectations(t)
	})

	t.Run("get auth token with corrupted data", func(t *testing.T) {
		api := &plugintest.API{}
		backendID := "test-backend-123"
		store := NewStateStore(api, backendID)

		expectedKey := "backend_test-backend-123_auth"
		api.On("KVGet", expectedKey).Return([]byte("invalid json"), nil)

		token, expiry, err := store.GetAuthToken()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to unmarshal")
		assert.Empty(t, token)
		assert.True(t, expiry.IsZero())
		api.AssertExpectations(t)
	})
}

func TestStateStore_Cursor(t *testing.T) {
	t.Run("save and retrieve cursor", func(t *testing.T) {
		api := &plugintest.API{}
		backendID := "test-backend-456"
		store := NewStateStore(api, backendID)

		cursor := "cursor_abc123xyz"
		expectedKey := "backend_test-backend-456_cursor"

		api.On("KVSet", expectedKey, []byte(cursor)).Return(nil)

		// Save
		err := store.SaveCursor(cursor)
		require.NoError(t, err)
		api.AssertExpectations(t)

		// Mock KVGet
		api.On("KVGet", expectedKey).Return([]byte(cursor), nil)

		// Retrieve
		gotCursor, err := store.GetCursor()
		require.NoError(t, err)
		assert.Equal(t, cursor, gotCursor)
		api.AssertExpectations(t)
	})

	t.Run("get cursor when none stored", func(t *testing.T) {
		api := &plugintest.API{}
		backendID := "test-backend-456"
		store := NewStateStore(api, backendID)

		expectedKey := "backend_test-backend-456_cursor"
		api.On("KVGet", expectedKey).Return(nil, nil)

		cursor, err := store.GetCursor()
		require.NoError(t, err)
		assert.Empty(t, cursor)
		api.AssertExpectations(t)
	})
}

func TestStateStore_LastPoll(t *testing.T) {
	t.Run("save and retrieve last poll time", func(t *testing.T) {
		api := &plugintest.API{}
		backendID := "test-backend-789"
		store := NewStateStore(api, backendID)

		pollTime := time.Date(2025, 1, 15, 14, 30, 45, 0, time.UTC)
		expectedKey := "backend_test-backend-789_last_poll"
		expectedData, _ := json.Marshal(pollTime)

		api.On("KVSet", expectedKey, expectedData).Return(nil)

		// Save
		err := store.SaveLastPoll(pollTime)
		require.NoError(t, err)
		api.AssertExpectations(t)

		// Mock KVGet
		api.On("KVGet", expectedKey).Return(expectedData, nil)

		// Retrieve
		gotTime, err := store.GetLastPoll()
		require.NoError(t, err)
		assert.Equal(t, pollTime, gotTime)
		api.AssertExpectations(t)
	})

	t.Run("get last poll when none stored", func(t *testing.T) {
		api := &plugintest.API{}
		backendID := "test-backend-789"
		store := NewStateStore(api, backendID)

		expectedKey := "backend_test-backend-789_last_poll"
		api.On("KVGet", expectedKey).Return(nil, nil)

		pollTime, err := store.GetLastPoll()
		require.NoError(t, err)
		assert.True(t, pollTime.IsZero())
		api.AssertExpectations(t)
	})
}

func TestStateStore_Failures(t *testing.T) {
	t.Run("increment failures from zero", func(t *testing.T) {
		api := &plugintest.API{}
		backendID := "test-backend-abc"
		store := NewStateStore(api, backendID)

		expectedKey := "backend_test-backend-abc_failures"

		// First call to GetFailures (inside IncrementFailures)
		api.On("KVGet", expectedKey).Return(nil, nil).Once()

		// Save incremented count
		expectedData, _ := json.Marshal(1)
		api.On("KVSet", expectedKey, expectedData).Return(nil)

		count, err := store.IncrementFailures()
		require.NoError(t, err)
		assert.Equal(t, 1, count)
		api.AssertExpectations(t)
	})

	t.Run("increment failures from existing count", func(t *testing.T) {
		api := &plugintest.API{}
		backendID := "test-backend-abc"
		store := NewStateStore(api, backendID)

		expectedKey := "backend_test-backend-abc_failures"

		// Mock existing count of 3
		existingData, _ := json.Marshal(3)
		api.On("KVGet", expectedKey).Return(existingData, nil).Once()

		// Save incremented count (4)
		newData, _ := json.Marshal(4)
		api.On("KVSet", expectedKey, newData).Return(nil)

		count, err := store.IncrementFailures()
		require.NoError(t, err)
		assert.Equal(t, 4, count)
		api.AssertExpectations(t)
	})

	t.Run("reset failures", func(t *testing.T) {
		api := &plugintest.API{}
		backendID := "test-backend-abc"
		store := NewStateStore(api, backendID)

		expectedKey := "backend_test-backend-abc_failures"
		expectedData, _ := json.Marshal(0)

		api.On("KVSet", expectedKey, expectedData).Return(nil)

		err := store.ResetFailures()
		require.NoError(t, err)
		api.AssertExpectations(t)
	})

	t.Run("get failures when none stored", func(t *testing.T) {
		api := &plugintest.API{}
		backendID := "test-backend-abc"
		store := NewStateStore(api, backendID)

		expectedKey := "backend_test-backend-abc_failures"
		api.On("KVGet", expectedKey).Return(nil, nil)

		count, err := store.GetFailures()
		require.NoError(t, err)
		assert.Equal(t, 0, count)
		api.AssertExpectations(t)
	})

	t.Run("get failures with existing count", func(t *testing.T) {
		api := &plugintest.API{}
		backendID := "test-backend-abc"
		store := NewStateStore(api, backendID)

		expectedKey := "backend_test-backend-abc_failures"
		existingData, _ := json.Marshal(5)
		api.On("KVGet", expectedKey).Return(existingData, nil)

		count, err := store.GetFailures()
		require.NoError(t, err)
		assert.Equal(t, 5, count)
		api.AssertExpectations(t)
	})
}

func TestStateStore_ClearOperationalState(t *testing.T) {
	t.Run("clears only cursor and auth token", func(t *testing.T) {
		api := &plugintest.API{}
		backendID := "test-backend-xyz"
		store := NewStateStore(api, backendID)

		// Should only delete cursor and auth, not failure tracking state
		expectedKeys := []string{
			"backend_test-backend-xyz_auth",
			"backend_test-backend-xyz_cursor",
		}

		for _, key := range expectedKeys {
			api.On("KVDelete", key).Return(nil)
		}

		err := store.ClearOperationalState()
		require.NoError(t, err)
		api.AssertExpectations(t)
	})
}

func TestStateStore_ClearAll(t *testing.T) {
	t.Run("clears all state keys", func(t *testing.T) {
		api := &plugintest.API{}
		backendID := "test-backend-xyz"
		store := NewStateStore(api, backendID)

		expectedKeys := []string{
			"backend_test-backend-xyz_auth",
			"backend_test-backend-xyz_cursor",
			"backend_test-backend-xyz_last_poll",
			"backend_test-backend-xyz_last_success",
			"backend_test-backend-xyz_failures",
			"backend_test-backend-xyz_last_error",
		}

		for _, key := range expectedKeys {
			api.On("KVDelete", key).Return(nil)
		}

		err := store.ClearAll()
		require.NoError(t, err)
		api.AssertExpectations(t)
	})
}

func TestStateStore_KeyIsolation(t *testing.T) {
	t.Run("different backend IDs use different keys", func(t *testing.T) {
		api := &plugintest.API{}

		store1 := NewStateStore(api, "backend-1")
		store2 := NewStateStore(api, "backend-2")

		cursor1 := "cursor_for_backend_1"
		cursor2 := "cursor_for_backend_2"

		// Each should use a different key
		api.On("KVSet", "backend_backend-1_cursor", []byte(cursor1)).Return(nil)
		api.On("KVSet", "backend_backend-2_cursor", []byte(cursor2)).Return(nil)

		err := store1.SaveCursor(cursor1)
		require.NoError(t, err)

		err = store2.SaveCursor(cursor2)
		require.NoError(t, err)

		api.AssertExpectations(t)

		// Verify retrievals use correct keys
		api.On("KVGet", "backend_backend-1_cursor").Return([]byte(cursor1), nil)
		api.On("KVGet", "backend_backend-2_cursor").Return([]byte(cursor2), nil)

		got1, err := store1.GetCursor()
		require.NoError(t, err)
		assert.Equal(t, cursor1, got1)

		got2, err := store2.GetCursor()
		require.NoError(t, err)
		assert.Equal(t, cursor2, got2)

		api.AssertExpectations(t)
	})
}
