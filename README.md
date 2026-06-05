# Mattermost Dataminr Plugin

Real-time alerting system that integrates the [Dataminr First Alert API](https://www.dataminr.com/) with Mattermost. The plugin continuously polls configured Dataminr backends for new events and posts rich, color-coded alerts directly into designated Mattermost channels.

## Features

- **Multi-Backend Support** — Configure multiple independent Dataminr backends, each with its own credentials, polling schedule, and target channel.
- **Rich Alert Formatting** — Alerts are posted as color-coded Slack-style attachments with embedded media, location data, source text, and deep links back to Dataminr.
- **Alert Priority Levels** — Visual indicators by severity:
  - 🔴 **Flash** (red) — Breaking news
  - 🟠 **Urgent** (orange) — High priority
  - 🟡 **Alert** (yellow) — Normal priority
- **Searchable Hashtags** — Each alert is tagged with hashtags for alert level, country, and topics (e.g., `#Flash`, `#Fire`) so teams can search and filter alerts in Mattermost.
- **Automatic Deduplication** — A shared in-memory cache (24-hour TTL) prevents duplicate alerts from being posted to the same channel, even when multiple backends target it. The same alert can still appear in different channels.
- **Auto-Disable on Failure** — Backends that accumulate 5 consecutive polling failures are automatically disabled to prevent runaway errors. Admins see real-time status indicators in the System Console.
- **Cluster-Aware Polling** — Uses Mattermost's cluster job scheduler so only one server instance polls each backend in high-availability deployments.
- **Cursor-Based Pagination** — Polling state is persisted in Mattermost's KV store, so the plugin picks up where it left off after restarts.

## Admin Setup

This plugin requires a **Mattermost Enterprise license**.

### 1. Install and enable the plugin

Build the bundle locally:

```bash
make dist
```

This produces a plugin archive under `dist/`, typically:

```text
dist/com.mattermost.plugin-dataminr-<version>.tar.gz
```

Upload that archive in the Mattermost System Console under **Plugins > Plugin Management** and enable the plugin.

### 2. Configure plugin settings

In the System Console, navigate to **Plugins > Dataminr Plugin** and configure:

#### Bot Username

The username for the bot account that posts alerts. Defaults to `dataminr-alerts`.

#### Bot Display Name

The display name shown on alert posts. Defaults to `Dataminr Alerts`.

#### Backend Configurations

Backends are managed through a custom admin UI. Each backend represents an independent connection to the Dataminr First Alert API. Click **Add Backend** to create a new one, then fill in the required fields:

| Field | Description |
|---|---|
| **Name** | A unique display name for this backend (e.g., "Production Alerts") |
| **Type** | Backend type — currently only `dataminr` is supported |
| **Enabled** | Whether this backend is actively polling |
| **URL** | Dataminr API base URL (e.g., `https://firstalert-api.dataminr.com`). Must use HTTPS. |
| **API ID** | Your Dataminr API user ID |
| **API Key** | Your Dataminr API key/password |
| **Channel** | The Mattermost channel where alerts will be posted (searchable across all teams) |
| **Poll Interval** | How often to poll for new alerts, in seconds (minimum: 10, default: 30) |

Each backend is assigned a stable UUID on creation. This ID is used internally for state storage and job scheduling and does not change even if you rename the backend.

### 3. Obtain Dataminr API credentials

Contact Dataminr to obtain First Alert API credentials. You will need:

- An **API User ID** (`api_user_id`)
- An **API Key** (`api_password`)
- Access to the **First Alert API** scope

The plugin authenticates using Dataminr's OAuth-style flow and automatically handles token refresh (tokens expire after 1 hour and are refreshed 5 minutes before expiry).

## Backend Status Indicators

The admin console shows real-time status for each backend:

| Status | Meaning |
|---|---|
| ✅ **Active** | Backend is enabled and polling successfully |
| ⚠️ **Warning** | Backend is enabled but has 1–4 consecutive failures |
| ❌ **Error** | Backend was auto-disabled after 5 consecutive failures |
| ⚪ **Disabled** | Backend was manually disabled by an admin |

Status is polled every 30 seconds from the admin console and includes the last poll time, last success time, failure count, and last error message.

## Alert Post Format

Each alert is posted as a rich Mattermost message containing:

- **Alert type badge** with emoji and color-coded sidebar (🔴 FLASH / 🟠 URGENT / 🟡 ALERT)
- **Headline** as an H3 header
- **Alert Link** — deep link to view the full alert in Dataminr
- **Public Source** — link to the original public source (if available)
- **Event Time** — when the event occurred
- **Location** — address and coordinates with confidence radius (if available)
- **Additional Context** — sub-headline with supplementary information
- **Original Source Text** — the source text, truncated to 500 characters
- **Translated Text** — machine-translated text for non-English sources
- **Topics** — categorized topic tags (e.g., Fire, Earthquake, Protest)
- **Alert Lists** — Dataminr alert list memberships
- **Embedded Media** — first image embedded directly; up to 3 additional media links
- **Hashtags** — searchable tags for alert level, country, and topics
- **Footer** — the backend name that produced the alert

## Operational Behavior

### Deduplication

The plugin maintains an in-memory deduplication cache shared across all backends. Cache keys include the backend type, alert ID, and destination channel ID, which means:

- The same alert **will not** be posted twice to the same channel, even from different backends
- The same alert **can** be posted to different channels if multiple backends target different channels

The cache has a 24-hour TTL and is cleaned up every 10 minutes. Duplicates may occur if:

- A backend is disabled for more than 24 hours (cache expires) and then re-enabled
- The plugin is deactivated or the server reboots (cache is cleared, but the poll cursor is preserved)

### Auto-Disable

After 5 consecutive polling failures, a backend is automatically disabled in the configuration. This prevents excessive API calls and error logging. To recover:

1. Investigate and resolve the underlying issue (credentials, network, API availability)
2. Re-enable the backend in the System Console

Re-enabling a backend resets its failure counter and clears operational state (cursor and auth token) to ensure a fresh start.

### State Management

Each backend persists the following state in Mattermost's KV store:

- **Poll cursor** — position in the Dataminr alert stream for cursor-based pagination
- **Auth token** — cached authentication token and expiry
- **Failure tracking** — last poll time, last success time, consecutive failure count, last error

When a backend is disabled, cursor and auth token are cleared but failure tracking is preserved for status display. When a backend is removed from configuration, all state is deleted.

## Configuration Reference

### `BotUsername`

The username for the bot account that posts alert notifications to channels.

- **Type**: Text
- **Default**: `dataminr-alerts`

### `BotDisplayName`

The display name shown on the bot's alert posts.

- **Type**: Text
- **Default**: `Dataminr Alerts`

### `Backends`

A JSON array of backend configuration objects. Each object has the following fields:

| Field | Type | Required | Description |
|---|---|---|---|
| `id` | UUID v4 | Yes | Unique, immutable identifier (auto-generated) |
| `name` | String | Yes | Display name (must be unique across backends) |
| `type` | String | Yes | Backend type (`dataminr`) |
| `enabled` | Boolean | Yes | Whether the backend is actively polling |
| `url` | String | Yes | Base API URL (must use HTTPS) |
| `apiId` | String | Yes | API user ID |
| `apiKey` | String | Yes | API key / password |
| `channelId` | String | Yes | Target Mattermost channel ID |
| `pollIntervalSeconds` | Integer | Yes | Poll interval in seconds (minimum: 10) |

Example:

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

## Development

### Requirements

- Go `1.25`
- Node.js `20.11` (see `.nvmrc`)
- npm compatible with that Node version

Use the repo's Node version with:

```bash
nvm use
```

Install webapp dependencies:

```bash
cd webapp && npm install
```

### Common commands

| Command | Description |
|---|---|
| `make test` | Run server and webapp unit tests |
| `make check-style` | Run ESLint, TypeScript type checking, `go vet`, and `golangci-lint` |
| `make dist` | Build and bundle the plugin for distribution |
| `make deploy` | Build and deploy the plugin to a running Mattermost server |
| `make watch` | Watch and rebuild webapp assets during development |
| `make coverage` | Generate a server code coverage report |
| `make clean` | Remove all build artifacts |
| `make logs` | View plugin logs from the server |
| `make logs-watch` | Tail plugin logs in real time |
| `make attach` | Attach `dlv` debugger to the running plugin process |

### Local deployment

If Mattermost local mode is enabled, `make deploy` can deploy directly to your local server.

You can also deploy with API credentials or a personal access token:

```bash
export MM_SERVICESETTINGS_SITEURL=http://localhost:8065
export MM_ADMIN_TOKEN=YOUR_TOKEN
make deploy
```

For watch mode:

```bash
export MM_SERVICESETTINGS_SITEURL=http://localhost:8065
export MM_ADMIN_TOKEN=YOUR_TOKEN
make watch
```

If `MM_SERVICESETTINGS_ENABLEDEVELOPER` is set, the server build is limited to the current platform architecture to speed up local iteration.

## Testing

Run the full validation suite:

```bash
make test
make check-style
```

Server tests cover authentication flow, alert processing, deduplication, state management, cursor-based polling, adapter normalization, and backend lifecycle. Webapp tests cover the admin console components and validation logic.

## Release

The plugin version is derived at build time from git tags.

Release helpers:

- `make patch` — bump patch version
- `make minor` — bump minor version
- `make major` — bump major version
- `make patch-rc` — bump patch release candidate
- `make minor-rc` — bump minor release candidate
- `make major-rc` — bump major release candidate

These targets create and push signed tags, so make sure you are on the correct branch and up to date before running them.

## Troubleshooting

### The plugin fails to activate

- Verify you have a valid **Mattermost Enterprise license**. The plugin will not activate without one.
- Check the Mattermost server logs for specific error messages.

### A backend shows ❌ Error status

The backend was auto-disabled after 5 consecutive failures. Common causes:

- **Invalid credentials** — verify `apiId` and `apiKey` are correct
- **Incorrect URL** — ensure the URL points to the correct Dataminr API endpoint
- **Network issues** — confirm the Mattermost server can reach the Dataminr API
- **Rate limiting** — Dataminr enforces 180 requests per 10 minutes; reduce the poll interval if necessary

To recover, fix the underlying issue and re-enable the backend in the System Console.

### Alerts appear duplicated

Duplicates are rare but can happen when:

- A backend was disabled for more than 24 hours (deduplication cache expires)
- The Mattermost server was restarted (in-memory cache is lost)
- Multiple plugin instances run outside of Mattermost's cluster scheduler

### Alerts are not appearing

- Confirm the backend is **enabled** and shows **Active** status
- Verify the target channel exists and the bot user has permission to post in it
- Check that the Dataminr API credentials have the `first_alert_api` scope
- Review plugin logs with `make logs` or `make logs-watch`
