# Mattermost Dataminr Plugin

**Version:** 0.9.3
**Date:** 2025-11-13
**Status:** Active Development

---

## Overview

Real-time alerting system that integrates Dataminr First Alert API with Mattermost. Polls configured backends for new events and posts them to designated channels.

### Key Design Principles

1. **Isolation**: Each backend operates independently with its own authentication, polling state, and error handling
2. **Resilience**: Backend failures don't affect other backends or plugin stability
3. **Extensibility**: Architecture supports future backend types beyond Dataminr
4. **Cluster-Aware**: Uses Mattermost's cluster scheduled job system to ensure only one server polls in multi-server deployments
5. **Deduplication**: Plugin-level deduplicator (24hr TTL) shared across backends prevents duplicate posts to the same channel while allowing the same alert to multiple channels

---

## Architecture

### Core Components

- **Backend Registry**: Thread-safe registry managing all backend instances
- **Backend Instance**: Self-contained unit with authentication, API client, cluster-aware poller, state storage, and alert processor
- **Alert Processor**: Normalizes alerts, uses shared deduplicator, posts to Mattermost
- **Deduplicator**: In-memory cache (24hr TTL) shared across all backends with namespaced alert IDs
- **Admin Console**: React component for backend configuration with real-time status display

### Backend Configuration

Backends are configured via `plugin.json` settings as a JSON array. Each backend requires:
- `id`: UUID v4 (immutable, auto-generated, used for KV store keys and job IDs)
- `name`: Display name (mutable, must be unique)
- `type`: "dataminr" (extensible)
- `enabled`: Boolean
- `url`, `apiId`, `apiKey`: Backend credentials
- `channelId`: Mattermost channel to post alerts
- `pollIntervalSeconds`: Poll interval (min: 10, default: 30)

**Important**: Backend `id` is immutable and used for all internal operations (KV keys, job IDs). Backend `name` can change without affecting state storage.

### Error Handling & Auto-Disable

- Backends track consecutive failures in KV store
- After 5 consecutive failures (`MaxConsecutiveFailures`), backend is automatically disabled
- Admin console displays status: ✅ Active, ⚠️ Warning (1-4 failures), ❌ Error (auto-disabled), ⚪ Disabled (manual)
- Re-enabling a backend resets failure state and clears operational state (cursor + auth token)

### Duplicate Alert Prevention

**Plugin-level deduplicator** (24hr TTL) prevents duplicate posts to the **same Mattermost channel** (including when multiple backends target that channel):
- Cache keys include backend type, alert id, and destination channel id (e.g., `dataminr:12345:channelId`) so the same alert can be posted to different channels independently
- Duplicates MAY occur if:
  - Backend disabled >24 hours (cache expired) then re-enabled
  - Plugin deactivated/server reboot (cache cleared, but cursor preserved)

---

## Critical Implementation Details

### ⚠️ Deadlock Prevention

1. **Configuration Lock**: NEVER hold `configurationLock` while calling `SavePluginConfig()` (triggers `OnConfigurationChange` which tries to acquire same lock)
   - Use `getConfiguration()` to clone config, modify clone, call `SavePluginConfig()` without holding locks

2. **Auto-Disable Callback**: MUST invoke `disableCallback` in a goroutine in poller.go
   - Without goroutine: poller calls callback → triggers `OnConfigurationChange` → tries to stop poller → poller waiting for callback = deadlock
   - With goroutine: poll cycle completes, then config change cleanly stops and re-registers backend

3. **Polling Wait Interval**: Cluster job scheduler's `nextWaitInterval()` must calculate elapsed time since last execution
   - Returning fixed interval causes job to never execute
   - Pattern: calculate elapsed time, return remaining wait if interval hasn't passed, return 0 if ready

### Dataminr API Specifics

**Authentication** (CRITICAL - must use form-urlencoded):
```
POST /auth/1/userAuthorization
Content-Type: application/x-www-form-urlencoded

grant_type=api_key&scope=first_alert_api&api_user_id=X&api_password=Y
```
- Token lifetime: 1 hour (refresh 5 min before expiry)
- Authorization header: `Dmauth {token}` (NOT "Bearer")

**Polling:**
```
GET /alerts/1/alerts?alertversion=19&from={cursor}
Authorization: Dmauth {token}
```
- Alert version hardcoded to 19 (changing requires updating parsing logic)
- Cursor-based pagination required
- Rate limit: 180 requests / 10 minutes

**Alert Types & Colors:**
- Flash: Red (#D24B4E) - Breaking news
- Urgent: Orange (#EC8832) - High priority
- Alert: Yellow (#FFBC1F) - Normal
- Unknown: Light Gray (#D3D3D3)

### Operational State Management

When backend is disabled (auto or manual), `ClearOperationalState()`:
- Removes cursor and auth token (ensures fresh start when re-enabled)
- Preserves failure tracking data (last poll, last success, consecutive failures, last error)

When backend is removed from config, `ClearAll()` removes all KV keys.

---

## Development Guidelines

### Testing

- **Unit Tests**: `testify` for assertions, `plugintest` for mocking Plugin API
- **HTTP Mocking**: `httptest` for mocking external APIs
- **Integration Tests**: Multiple components, `*_integration_test.go` files
- **Linting**: CRITICAL - Always run `make check-style` before committing
  - Use `npm run fix` in webapp directory for auto-fixes
  - Fix TypeScript errors manually (use `!` or `as` when appropriate)
- **Code Formatting**: Use `go fmt` or `make check-style` (do NOT use `goimports`)

### Commit Messages

- **Do NOT use conventional commit prefixes** (no `feat:`, `chore:`, `fix:`, etc.)
- Write clear, concise messages focusing on "why" rather than "what"
- All commits include attribution footer:
  ```
  🤖 Generated with [Claude Code](https://claude.com/claude-code)

  Co-Authored-By: Claude <noreply@anthropic.com>
  ```

### Webapp Implementation Notes

- **Validation**: Runs on mount and blur, tracks user changes with `useRef` to avoid re-validation during edits
- **Status Pills**: Use `var(..., fallback)` with explicit hex fallbacks for theme reliability
- **Channel Selector**: Uses Mattermost Client4 API, searches all channels (public/private) across teams
- **Status Polling**: Fetches `/api/v1/backends/status` every 30 seconds, merges with config data by backend ID

---

## Example Configuration

```json
[
  {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "name": "Production Alerts",
    "type": "dataminr",
    "enabled": true,
    "url": "https://firstalert-api.dataminr.com",
    "apiId": "your_api_id",
    "apiKey": "your_api_key",
    "channelId": "abc123channelid",
    "pollIntervalSeconds": 30
  }
]
```

---
