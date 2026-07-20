# HarborX

`HarborX` is a self-hosted control panel inspired by `miaomiaowuX`, rebuilt for personal use without any `license`, `pro`, or feature-gating modules.

## Goals

- Keep the broad functional surface of a modern Xray and subscription control panel.
- Use a clean feature-oriented architecture that is easier to extend than a single large entrypoint.
- Build a first-party rule studio with visual Clash rule editing.
- Ship with built-in private templates that can be expanded later.

## Functional Scope

This project is being built as a full self-hosted control plane and already includes the first persistence-backed slice for several core domains.

- Authentication and user management
- Nodes and subscriptions
- Rule templates and visual Clash rule editing
- Proxy groups and routing
- Xray config generation and snapshots
- Remote server and agent management
- Certificates and DNS providers
- Notifications, backups, and system settings
- Dashboard, traffic, and audit visibility

## Project Layout

- `cmd/agent`: remote server agent loop for heartbeat and queued tasks
- `cmd/server`: Go API entrypoint
- `internal/app`: app bootstrap and feature wiring
- `internal/config`: environment configuration
- `internal/httpapi`: HTTP router and handlers
- `internal/features`: feature-oriented services
- `internal/storage`: storage interfaces and bootstrapping
- `web`: React and Vite frontend
- `docs`: architecture and roadmap notes

## Non-Goals

- No `license` module
- No `pro` feature checks
- No paid gating or entitlement logic

## VPS One-Click Deploy

On a fresh VPS, run:

```bash
curl -fsSL https://raw.githubusercontent.com/youko-nobody/HarborX/main/scripts/deploy-vps.sh | sudo bash
```

Optional environment variables:

```bash
export HARBORX_PORT=18080
export HARBORX_ADMIN_PASSWORD="replace-with-a-strong-password"
export HARBORX_INSTALL_DIR=/opt/harborx
```

The script installs Docker, pulls this repository, writes `.env`, builds the image, and starts HarborX with Docker Compose.

## Agent Quick Start

Register a remote server in the HarborX console and copy the one-time agent token.

```bash
export HARBORX_AGENT_BASE_URL="https://your-harborx.example.com"
export HARBORX_AGENT_TOKEN="hxa_..."
export HARBORX_AGENT_INTERVAL_SECONDS=10
curl -fsSL https://raw.githubusercontent.com/youko-nobody/HarborX/main/scripts/install-agent.sh | sudo -E bash
```

`shell-script` tasks are disabled by default. Enable them only on servers you control:

```bash
export HARBORX_AGENT_ALLOW_SHELL=1
```

## Current Foundation

- SQLite bootstrap with schema and seed templates
- Admin login, API-token sessions, and protected mutation endpoints
- CRUD endpoints for users, nodes, rule sets, templates, subscriptions, proxy groups, DNS providers, certificates, notifications, backups, system settings, traffic samples, and remote servers
- Subscription rendering for Clash-like and sing-box templates
- Share-link import for common `vmess://`, `vless://`, `trojan://`, and `ss://` node links
- Xray configuration preview from saved nodes and rules
- Remote server enrollment tokens, task queues, and agent heartbeat/task APIs
- Agent executors for Xray restart/reload/install, Nginx install, certificate renewal, WARP script launch, and opt-in shell scripts
- Database backup export with SQLite `VACUUM INTO`
- Telegram and webhook notification test delivery
- React operator console with live bootstrap loading and no license/pro gating

## Current API Slice

- `POST /api/v1/auth/login`
- `GET/POST/PUT/DELETE /api/v1/users`
- `GET/POST/PUT/DELETE /api/v1/nodes`
- `POST /api/v1/nodes/import`
- `GET/POST/PUT/DELETE /api/v1/rulesets`
- `GET/POST/PUT/DELETE /api/v1/templates`
- `GET/POST/PUT/DELETE /api/v1/subscriptions`
- `GET /api/v1/subscriptions/{id}/preview`
- `GET /api/v1/subscriptions/{id}/download`
- `GET /api/v1/xray/preview`
- `GET/POST/PUT/DELETE /api/v1/remote/servers`
- `GET/POST /api/v1/remote/servers/{id}/tasks`
- `POST /api/v1/agent/heartbeat`
- `POST /api/v1/agent/tasks/claim`
- `POST /api/v1/agent/tasks/{id}`
- `POST /api/v1/backups/export`
- `POST /api/v1/notifications/channels/{id}/test`
- Summary and bootstrap endpoints for all feature domains

## Next Steps

1. Add the standalone remote agent binary that consumes the agent task API.
2. Implement real ACME issue/renew/deploy workers.
3. Add traffic aggregation jobs and dashboard charts.
4. Add importers for existing subscription/node formats.
