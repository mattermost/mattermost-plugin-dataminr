// Copyright (c) 2025-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package dataminr

import (
	"fmt"
	"time"

	"github.com/mattermost/mattermost/server/public/plugin"
	"github.com/mattermost/mattermost/server/public/pluginapi"
	"github.com/mattermost/mattermost/server/public/pluginapi/cluster"

	"github.com/mattermost/mattermost-plugin-dataminr/server/backend"
)

// AlertFetcher is an interface for fetching alerts from the Dataminr API
type AlertFetcher interface {
	FetchAlerts(cursor string) (*AlertsResponse, error)
}

// Poller manages the cluster-aware scheduled polling job for a Dataminr backend
type Poller struct {
	api             *pluginapi.Client
	backendID       string
	backendName     string
	interval        time.Duration
	client          AlertFetcher
	processor       *AlertProcessor
	stateStore      *StateStore
	scheduler       JobScheduler
	job             Job
	disableCallback backend.DisableCallback
}

// NewPoller creates a new poller instance
func NewPoller(
	api *pluginapi.Client,
	papi plugin.API,
	backendID string,
	backendName string,
	interval time.Duration,
	client AlertFetcher,
	processor *AlertProcessor,
	stateStore *StateStore,
	disableCallback backend.DisableCallback,
) *Poller {
	return &Poller{
		api:             api,
		backendID:       backendID,
		backendName:     backendName,
		interval:        interval,
		client:          client,
		processor:       processor,
		stateStore:      stateStore,
		scheduler:       NewClusterJobScheduler(papi),
		disableCallback: disableCallback,
	}
}

// SetScheduler sets a custom job scheduler (useful for testing)
func (p *Poller) SetScheduler(scheduler JobScheduler) {
	p.scheduler = scheduler
}

// Start begins the polling job using Mattermost's cluster job system
// This ensures only one server instance polls in a multi-server cluster
func (p *Poller) Start() error {
	if p.job != nil {
		return fmt.Errorf("poller already running")
	}

	return p.startRegularJob()
}

// startRegularJob starts the regular polling job
func (p *Poller) startRegularJob() error {
	jobID := fmt.Sprintf("dataminr_poll_%s", p.backendID)

	// Schedule the recurring job with cluster awareness
	job, err := p.scheduler.Schedule(jobID, p.nextWaitInterval, p.run)
	if err != nil {
		return fmt.Errorf("failed to schedule cluster job: %w", err)
	}

	p.job = job
	p.api.Log.Info("Poller started", "backendId", p.backendID, "backendName", p.backendName, "interval", p.interval)
	return nil
}

// Stop gracefully stops the polling job
func (p *Poller) Stop() error {
	if p.job == nil {
		return nil
	}

	err := p.job.Close()
	p.job = nil

	if err != nil {
		p.api.Log.Error("Failed to close cluster job", "backendId", p.backendID, "error", err.Error())
		return fmt.Errorf("failed to close cluster job: %w", err)
	}

	p.api.Log.Info("Poller stopped", "backendId", p.backendID, "backendName", p.backendName)
	return nil
}

// nextWaitInterval is called by the cluster job scheduler to determine how long to wait
// until the next poll. The metadata.LastFinished is automatically set by the cluster scheduler.
func (p *Poller) nextWaitInterval(now time.Time, metadata cluster.JobMetadata) time.Duration {
	// For the first run, execute immediately
	if metadata.LastFinished.IsZero() {
		return 0
	}

	// Check if enough time has passed since last finished
	sinceLastFinished := now.Sub(metadata.LastFinished)
	if sinceLastFinished < p.interval {
		// Not enough time elapsed, return remaining wait time
		return p.interval - sinceLastFinished
	}

	// Enough time has passed, run immediately
	return 0
}

// run is called by the cluster job scheduler to execute a poll cycle
func (p *Poller) run() {
	p.api.Log.Debug("Starting poll cycle", "backendId", p.backendID, "backendName", p.backendName)

	// Update last poll time
	if err := p.stateStore.SaveLastPoll(time.Now()); err != nil {
		p.api.Log.Error("Failed to save last poll time", "backendId", p.backendID, "error", err.Error())
	}

	// Load cursor from state
	cursor, err := p.stateStore.GetCursor()
	if err != nil {
		p.handlePollError(fmt.Errorf("failed to load cursor: %w", err))
		return
	}

	// Fetch alerts from API
	response, err := p.client.FetchAlerts(cursor)
	if err != nil {
		p.handlePollError(fmt.Errorf("failed to fetch alerts: %w", err))
		return
	}

	// Process alerts
	newCount, err := p.processor.ProcessAlerts(response.Alerts)
	if err != nil {
		p.handlePollError(fmt.Errorf("failed to process alerts: %w", err))
		return
	}

	// Save new cursor
	if response.To != "" {
		if err := p.stateStore.SaveCursor(response.To); err != nil {
			p.handlePollError(fmt.Errorf("failed to save cursor: %w", err))
			return
		}
	}

	// Poll succeeded - update success state
	now := time.Now()
	if err := p.stateStore.SaveLastSuccess(now); err != nil {
		p.api.Log.Error("Failed to save last success time", "backendId", p.backendID, "error", err.Error())
	}

	// Reset failure counter
	if err := p.stateStore.ResetFailures(); err != nil {
		p.api.Log.Error("Failed to reset failure counter", "backendId", p.backendID, "error", err.Error())
	}

	// Clear last error on success
	if err := p.stateStore.SaveLastError(""); err != nil {
		p.api.Log.Error("Failed to clear last error", "backendId", p.backendID, "error", err.Error())
	}

	p.api.Log.Debug("Poll cycle completed",
		"backendId", p.backendID,
		"backendName", p.backendName,
		"totalAlerts", len(response.Alerts),
		"newAlerts", newCount,
		"cursor", response.To)
}

// handlePollError increments failure count and disables backend if threshold exceeded
func (p *Poller) handlePollError(err error) {
	errMsg := err.Error()

	p.api.Log.Error("Poll cycle failed",
		"backendId", p.backendID,
		"backendName", p.backendName,
		"error", errMsg)

	// Save error message
	if saveErr := p.stateStore.SaveLastError(errMsg); saveErr != nil {
		p.api.Log.Error("Failed to save last error",
			"backendId", p.backendID,
			"error", saveErr.Error())
	}

	// Increment failure counter
	failureCount, incrementErr := p.stateStore.IncrementFailures()
	if incrementErr != nil {
		p.api.Log.Error("Failed to increment failure counter",
			"backendId", p.backendID,
			"error", incrementErr.Error())
		return
	}

	// Check if backend should be disabled
	if failureCount >= backend.MaxConsecutiveFailures {
		p.api.Log.Error("Backend reached max consecutive failures",
			"backendId", p.backendID,
			"backendName", p.backendName,
			"consecutiveFailures", failureCount,
			"lastError", errMsg)

		// Call disable callback to persist the configuration change
		// This will trigger OnConfigurationChange which will stop the backend
		// Wrap in goroutine to avoid deadlock (callback triggers Stop() on this backend)
		if p.disableCallback != nil {
			go func() {
				if disableErr := p.disableCallback(p.backendID); disableErr != nil {
					p.api.Log.Error("Failed to disable backend in configuration",
						"backendId", p.backendID,
						"error", disableErr.Error())

					// Fallback: stop the poller locally if callback fails
					if stopErr := p.Stop(); stopErr != nil {
						p.api.Log.Error("Failed to stop poller after callback failure",
							"backendId", p.backendID,
							"error", stopErr.Error())
					}
				}
			}()
		} else {
			// Fallback: stop the poller if no callback is provided
			p.api.Log.Warn("No disable callback provided, stopping poller locally",
				"backendId", p.backendID)
			if stopErr := p.Stop(); stopErr != nil {
				p.api.Log.Error("Failed to stop poller",
					"backendId", p.backendID,
					"error", stopErr.Error())
			}
		}
	}
}
