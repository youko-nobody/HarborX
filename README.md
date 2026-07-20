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

## Current Foundation

- SQLite bootstrap with schema and seed templates
- Real CRUD endpoints for nodes, rule sets, templates, and subscriptions
- Feature-oriented backend modules covering the full product surface
- React operator console scaffold with live bootstrap loading

## Current API Slice

- `GET/POST /api/v1/nodes`
- `DELETE /api/v1/nodes/{id}`
- `GET/POST /api/v1/rulesets`
- `GET/POST /api/v1/templates`
- `GET/POST /api/v1/subscriptions`
- Summary and bootstrap endpoints for all planned feature domains

## Next Steps

1. Add auth sessions, users, and dashboard persistence.
2. Wire the frontend rule studio to the CRUD APIs.
3. Add Xray config rendering and snapshot management.
4. Implement remote agent workflows and server orchestration.
