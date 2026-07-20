# Storage Plan

This directory will hold the persistent repositories for:

- users
- nodes
- subscriptions
- rules
- templates
- proxy groups
- remote servers
- traffic snapshots
- certificate assets
- backups

The first persistence target is SQLite. Repositories should be defined behind interfaces so the feature services stay testable.

