# Feature Map
This file tracks the breadth target so the project does not silently shrink over time.

## Shipped (implemented and wired into the API)
- Dashboard and operator overview
- Admin auth, password login, and API-token based authentication
- User management and roles
- Node management and source import
- Subscription generation and delivery
- Clash rule visualization and editing
- Proxy groups and routing policies
- Built-in and private templates
- Xray config generation, preview, and snapshots
- Remote server registry and agent orchestration
- Remote Xray and Nginx management via agent tasks
- Traffic sample ingestion, rollups, and dashboard metrics endpoints
- Notifications: Telegram and webhook channels (test delivery implemented;
  scheduled/alert automation is on the roadmap — see "Planned" below)
- Backups: database export and manual restore (no in-UI restore endpoint yet)
- System settings and customisation

## Planned (modelled in code, not yet fully implemented)
These domains have a DB table and service skeleton, but their core action
workflows are not wired end-to-end yet. They are intentionally kept out of the
"Shipped" list so this file does not over-promise.

- **ACME certificates**: storage exists, but issue / renew / deploy workers are
  not implemented. For now, upload an already-issued PEM pair via the
  certificates API.
- **DNS providers**: storage and provider CRUD exist, but record/zone actions
  against Cloudflare / AliDNS / DNSPod / Tencent / GoDaddy / NameSilo are not
  implemented.
- **Notification automation**: Telegram + webhook test delivery works. Traffic
  thresholds, server-status alerts, and daily summaries are planned but not
  scheduled yet.
- **WARP / remote network tooling**: removed as a product commitment. There is
  no WARP plan for HarborX.

## Explicitly excluded

- License enforcement
- Pro gating
- Paid feature entitlement logic
- Two-factor (TOTP) authentication — deliberately not on the roadmap; the
  authentication surface is password + API token only.

