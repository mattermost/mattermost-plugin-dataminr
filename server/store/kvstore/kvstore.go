// Copyright (c) 2025-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package kvstore

type KVStore interface {
	// Define your methods here. This package is used to access the KVStore pluginapi methods.
	GetTemplateData(userID string) (string, error)
}
