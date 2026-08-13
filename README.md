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

### Supported Shell-Script Commands

When `HARBORX_AGENT_ALLOW_SHELL=1`, shell-script tasks are **not** restricted to a
fixed allow-list. The guard is a coarse **deny-list**: anything that does not
contain a known-bad pattern is allowed to run via `sh -lc`. This is intentional
because operational commands vary widely (`systemctl restart xray`, `nginx -t`,
`certbot renew`, `curl ... -o /tmp/x.sh`, `apt-get install -y <pkg>`, etc.).

The patterns that are actively **blocked** are the ones that would let an
attacker destroy the host or open a reverse shell. The deny-list (see
`cmd/agent/main.go`, function `isAllowedShellCommand`) is:

| Blocked pattern | Why |
|---|---|
| ` rm -rf ` / `rm -rf /` / `rm -rf /*` | destructive deletion |
| ` \| bash ` / ` \| sh ` / `\| bash` / `\| sh` | pipe-to-shell (e.g. `curl install.sh \| bash`) |
| `bash -i` / `sh -i` | interactive reverse shell |
| `/dev/tcp/` / `/dev/udp/` | bash TCP/UDP reverse shell |
| `python3 -c` / `python -c` / `perl -e` / `ruby -e` | one-liner code execution |
| `nc -e` / `ncat -e` | netcat reverse shell |
| `mkfifo` | classic FIFO shell trick |
| `base64 -d \|` | decode-and-run payloads |

Whitespace-normalised before matching, so `rm -rf  /` (doubled space) still trips
the `rm -rf` rule.

Because the guard is coarse, treat `HARBORX_AGENT_ALLOW_SHELL=1` as "this host
is fully trusted and the operator is responsible for what they type". If you
want a true allow-list instead, narrow the logic in `cmd/agent/main.go` and push.

## Notifications

The notifications module supports two channel kinds. Configure them from the
`/api/v1/notifications/channels` endpoints or from the settings UI.

| Channel kind | Required config fields | Purpose |
|---|---|---|
| `telegram` | `botToken`, `chatId` | Send alerts to a Telegram chat via Bot API |
| `webhook` | `url` | POST `{"message":"…","source":"harborx"}` to any HTTPS endpoint |

Test a channel before wiring it into alerts:

```
POST /api/v1/notifications/channels/{id}/test
{"message": "connection ok"}
```

Notifications are currently delivered only when you explicitly test or trigger
them from the console; automatic alert scheduling (traffic thresholds, server
status, daily summary) is on the roadmap and is not yet implemented. The
`Summary()` endpoint therefore lists these as *planned capabilities*, not
currently active features.

## Backup and Restore

### Exporting a backup

The server ships a database-export endpoint. Call it (as admin) to create a
point-in-time copy of the SQLite database under `$HARBORX_DATA_DIR/backups/`:

```
POST /api/v1/backups/export
{"backupKind": "database", "summary": "pre-upgrade snapshot"}
```

The returned `filePath` is the absolute path to the `.sqlite` file inside the
container or on the host, depending on how the volume is mounted. List existing
backups with `GET /api/v1/backups`.

### Restoring from a backup

There is currently **no in-UI restore endpoint**. Restore is a manual operation:
replace the live database file with the backed-up one, then restart the server.
Because the application uses SQLite, this is atomic as long as the process is
stopped while you swap the file.

**Docker (recommended):**

```bash
# 1. Stop the server so it releases the DB file
docker stop harborx

# 2. The DB lives in the named volume `harborx-data`, mounted at /app/data
#    inside the container. Copy the backup into place from a helper container.
docker run --rm -v harborx-data:/app/data -v $(pwd):/tmp/backup alpine:3.21 sh -c '
  cp /app/data/harborx.sqlite /app/data/harborx.sqlite.bak.$(date +%s)
  cp /tmp/backup/harborx-db-20260814-120000.sqlite /app/data/harborx.sqlite
'

# 3. Start again — on boot the server will run any pending schema migrations
docker start harborx
```

**Bare-metal (`go run ./cmd/server` or binary):**

```bash
systemctl stop harborx          # or kill the running server process
cd /path/to/your/HARBORX_DATA_DIR
cp harborx.sqlite harborx.sqlite.bak.$(date +%s)
cp /path/to/backup/harborx-db-YYYYMMDD.sqlite harborx.sqlite
systemctl start harborx
```

### Notes

- Before upgrading a major version, always take a backup first — the schema
  migration table (`schema_migrations`) makes forward upgrades idempotent, but
  there is no built-in downgrade path.
- Backups are SQLite *file copies*. Restore by replacing `harborx.sqlite`; do
  not attempt to `ATTACH` / `INSERT` the backup into the live DB.
- Templates and rule sets live in the same database, so a DB restore fully
  covers them. `xray` snapshots and remote-server state are also in the DB.

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

Roadmap items that are not yet implemented. Frontend work for the audit-events
view and subscription token-rotate action has landed; do not repeat it.

1. ACME issue / renew / deploy workers — real certificate automation end-to-end
   (storage and provider CRUD exist; the workers do not).
2. DNS provider record / zone actions — wire Cloudflare / AliDNS / DNSPod /
   Tencent / GoDaddy / NameSilo so DNS is more than a credential store.
3. Notification automation — schedule traffic-threshold, server-status and
   daily-summary alerts against the Telegram / webhook channels.
4. Traffic dashboard charts — turn the rollup endpoints into a real chart view
   in the frontend.
5. In-UI backup restore — add a server-side restore endpoint so operators do
   not have to swap the SQLite file by hand.
6. Hardening the agent shell-script guard from a coarse deny-list to a real
   allow-list (see `cmd/agent/main.go`, `isAllowedShellCommand`).