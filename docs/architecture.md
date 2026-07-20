# Architecture Notes

## Principles

- Feature-oriented backend modules instead of one giant handler package.
- Frontend routes organized around operator workflows.
- API-first development so CLI, UI, and future agents can share capabilities.
- Self-hosted by default, with no feature entitlements.

## Backend Modules

- `auth`: login, sessions, TOTP, API tokens
- `users`: admin and member management
- `dashboard`: summary metrics and status
- `nodes`: inbound and outbound node inventory
- `subscriptions`: subscribe endpoints, output formats, user views
- `rules`: Clash-style rule editor, ordering, validation
- `templates`: built-in and private templates
- `proxygroups`: policy groups and routing targets
- `xray`: config generation, snapshots, apply plans
- `remote`: remote server registry and agent orchestration
- `traffic`: usage and aggregation
- `certificates`: ACME and deployment workflows
- `dns`: DNS provider integrations
- `notifications`: Telegram and future channels
- `backups`: export, restore, retention
- `system`: platform settings and defaults

## Frontend Areas

- Dashboard
- Nodes
- Subscriptions
- Rules Studio
- Templates
- Remote Servers
- Xray
- Certificates
- Notifications
- Settings

## Implementation Strategy

Build breadth first, then deepen one vertical slice at a time:

1. API shell and data contracts
2. SQLite repositories
3. Dashboard, nodes, subscriptions
4. Rule studio and templates
5. Xray rendering
6. Remote agent workflows
7. Certificates and automation

