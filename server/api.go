// Copyright (c) 2025-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"

	"github.com/mattermost/mattermost-plugin-dataminr/server/backend"
)

// ServeHTTP handles HTTP requests for the plugin.
// All endpoints require system admin permissions.
func (p *Plugin) ServeHTTP(_ *plugin.Context, w http.ResponseWriter, r *http.Request) {
	// All HTTP endpoints require a logged-in user
	userID := r.Header.Get("Mattermost-User-ID")
	if userID == "" {
		http.Error(w, "Not authorized", http.StatusUnauthorized)
		return
	}

	// All HTTP endpoints require system admin permissions
	if !p.client.User.HasPermissionTo(userID, model.PermissionManageSystem) {
		http.Error(w, "Not authorized", http.StatusUnauthorized)
		return
	}

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/backends/status", p.getBackendsStatus).Methods(http.MethodGet)

	router.ServeHTTP(w, r)
}

// getBackendsStatus returns the status of all configured backends.
// Response is a map of backend ID (UUID) to status object.
func (p *Plugin) getBackendsStatus(w http.ResponseWriter, _ *http.Request) {
	// Get all registered backends
	backends := p.registry.List()

	// Build status map
	statusMap := make(map[string]backend.Status)
	for _, b := range backends {
		statusMap[b.GetID()] = b.GetStatus()
	}

	// Marshal and return response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(statusMap); err != nil {
		p.API.LogError("Failed to encode backend status response", "error", err.Error())
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}
