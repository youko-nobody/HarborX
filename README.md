# HarborX

A self-hosted control panel for personal Xray, subscription, and VPS operations — rebuilt without any `license`, `pro`, or feature-gating modules.

## Goals

- Keep the broad functional surface of a modern Xray and subscription control panel.
- Use a clean feature-oriented architecture that is easier to extend than a single large entrypoint.
- Ship with first-party rule studio and private templates that can be expanded later.
- Be production-safe: auditable, bounded inputs, session-capped, and container-ready.

## Functional Scope

- Authentication and user management (roles, session pruning, audit trail)
- Nodes and subscriptions (Clash / sing-box templates, token rotation)
- Rule templates and visual Clash rule editing
- Proxy groups and routing
- Xray config generation, preview, and snapshots
- Remote server and agent management
- Certificates and DNS providers
- Notifications, backups, and system settings
- Dashboard, traffic, and **audit visibility**

## Project Layout

- `cmd/agent`: remote server agent (heartbeat + task execution)
- `cmd/server`: Go API entrypoint
- `internal/app`: bootstrap and feature wiring
- `internal/config`: environment configuration
- `internal/httpapi`: HTTP router, auth, rate-limiting, audit endpoints
- `internal/features`: feature-oriented services
- `internal/storage`: SQLite storage + schema migration
- `web`: React + Vite frontend

## Non-Goals

- No `license` module, no `pro` checks, no paid gating.

## Quick Start (Docker)

```bash
git clone https://github.com/youko-nobody/HarborX.git
cd HarborX
cp .env.example .env
# Edit HARBORX_ADMIN_PASSWORD in .env
docker compose up -d
```

The server starts on port `18080` (override with `HARBORX_PORT`).

## Environment Variables

| Variable | Default | Description |
|---|---|---|
| `HARBORX_HOST` | `0.0.0.0` | Listen address |
| `HARBORX_PORT` | `18080` | Listen port |
| `HARBORX_DATA_DIR` | `/app/data` | Data directory |
| `HARBORX_DB_PATH` | `/app/data/harborx.sqlite` | SQLite DB path |
| `HARBORX_WEB_DIST_DIR` | `/app/web-dist` | Frontend dist directory |
| `HARBORX_ADMIN_PASSWORD` | — | Initial admin password (first run only) |
| `HARBORX_CORS_ORIGINS` | empty (permissive) | Comma-separated allowed origins; set in production |

## Agent

Register a remote server in the console, copy the one-time agent token, then run the agent.

```bash
export HARBORX_AGENT_BASE_URL="https://your-harborx.example.com"
export HARBORX_AGENT_TOKEN="hxa_..."
export HARBORX_AGENT_INTERVAL_SECONDS=10

# One-click install (Linux):
curl -fsSL https://raw.githubusercontent.com/youko-nobody/HarborX/main/scripts/install-agent.sh | sudo -E bash
```

### Agent Safety

- `shell-script` tasks are **disabled by default**. Enable only on servers you control:

  ```bash
  export HARBORX_AGENT_ALLOW_SHELL=1
  ```

- Even when enabled, commands pass a coarse **allow-list guard** that blocks obvious destructive and reverse-shell patterns (`rm -rf /`, `| bash`, `python3 -c`, `/dev/tcp/`, etc.). Prefer `systemctl restart xray`-style commands.
- The agent has no HTTP-based command execution; all writes go through the auth-protected server API.

## Audit Trail

Every sensitive operator action is recorded to the `audit_logs` table:

- Login (`auth.login`)
- User create / delete
- Node create / delete
- Subscription create / delete / **token rotate**
- Xray config apply
- Remote server create

View the audit trail via:

```
GET /api/v1/audit/summary
GET /api/v1/audit/events
```

Retention is **90 days**; on each server startup, entries older than the retention window are pruned automatically.

## Subscription Token Rotation

When a subscription's client token is compromised, rotate it with a single authenticated request. The old token is invalidated atomically at the instant the new one is issued.

```
POST /api/v1/subscriptions/{id}/token-rotate
```

## Security Defaults

- **Request body cap**: every JSON endpoint enforces a 4 MiB body limit (defends against memory-exhaustion via oversized payloads).
- **Error sanitisation**: database paths, table names, and SQL fragments are redacted from error responses (HTTP 500 → `internal error`; constraint failures → `resource conflict`).
- **CORS**: permissive by default for self-hosted deployments; restrict via `HARBORX_CORS_ORIGINS` in production.
- **Sessions**: each login prunes old web sessions (30-day window, up to 10 per user) so tokens do not accumulate forever.
- **SQLite**: WAL mode + `busy_timeout` so concurrent writes do not produce `database is locked`.
- **ID generation**: UUIDs (via `github.com/google/uuid`) — no timestamp-collision primary key violations.
- **Graceful shutdown**: server handles `SIGINT` / `SIGTERM` with a 30 s in-flight request drain.

## Current API Slice

- `POST /api/v1/auth/login`
- `GET/POST/PUT/DELETE /api/v1/users`
- `GET/POST/PUT/DELETE /api/v1/nodes`, `POST /api/v1/nodes/import`
- `GET/POST/PUT/DELETE /api/v1/rulesets`
- `GET/POST/PUT/DELETE /api/v1/templates`
- `GET/POST/PUT/DELETE /api/v1/subscriptions`
- `GET /api/v1/subscriptions/{id}/preview`
- `GET /api/v1/subscriptions/{id}/download`
- `POST /api/v1/subscriptions/{id}/token-rotate`
- `GET/POST/PUT/DELETE /api/v1/xray`, `GET /api/v1/xray/preview`
- `POST /api/v1/xray/snapshots/{id}/restore`
- `GET/POST/PUT/DELETE /api/v1/remote/servers`
- `GET/POST /api/v1/remote/servers/{id}/tasks`
- `POST /api/v1/agent/heartbeat`
- `POST /api/v1/agent/tasks/claim`
- `POST /api/v1/agent/tasks/{id}`
- `POST /api/v1/backups/export`
- `POST /api/v1/notifications/channels/{id}/test`
- `GET /api/v1/audit/summary`
- `GET /api/v1/audit/events`
- Summary and bootstrap endpoints for all feature domains

## CI

The `main` branch runs GitHub Actions on every push: `go vet ./...` → `go build ./...` → `go test ./...`.

## Next Steps

1. Real ACME issue/renew/deploy workers.
2. Traffic aggregation jobs and dashboard charts.
3. Frontend audit-events view.
4. Frontend subscription token-rotate action.