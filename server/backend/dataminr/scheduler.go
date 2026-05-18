// Copyright (c) 2025-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package dataminr

import (
	"github.com/mattermost/mattermost/server/public/plugin"
	"github.com/mattermost/mattermost/server/public/pluginapi/cluster"
)

// Job represents a scheduled job that can be closed
type Job interface {
	Close() error
}

// JobScheduler is an interface for scheduling cluster-aware jobs
type JobScheduler interface {
	Schedule(
		jobID string,
		nextWaitInterval cluster.NextWaitInterval,
		callback func(),
	) (Job, error)
}

// ClusterJobScheduler is the production implementation that uses Mattermost's cluster job system
type ClusterJobScheduler struct {
	api plugin.API
}

// NewClusterJobScheduler creates a new cluster job scheduler
func NewClusterJobScheduler(api plugin.API) *ClusterJobScheduler {
	return &ClusterJobScheduler{
		api: api,
	}
}

// Schedule creates a new cluster-aware scheduled job
func (s *ClusterJobScheduler) Schedule(
	jobID string,
	nextWaitInterval cluster.NextWaitInterval,
	callback func(),
) (Job, error) {
	return cluster.Schedule(s.api, jobID, nextWaitInterval, callback)
}
