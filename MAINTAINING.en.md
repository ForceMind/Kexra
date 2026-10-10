# Kexra Maintainer Guide

The primary maintained handbook is [MAINTAINING.md](MAINTAINING.md) in Chinese, together with [development checks](docs/DEVELOPMENT.md), [deployment status](docs/DEPLOYMENT.md), and [migration issues](docs/MIGRATION.md).

## Operating rules

- Maintain Kexra from the `ForceMind/Kexra` repository, not an auto-synced upstream fork.
- Preserve applicable AGPL and MIT legal notices, original copyrights, and import provenance.
- Record successful, failed, **and not executed** CI checks explicitly.
- Treat an unavailable GitHub Actions runner as an infrastructure issue, not proof of compilation success.
- Validate Go module/CLI names, release archive paths, updater, Docker image namespaces, database backup and rollback before creating a `v*` tag.
- Do not use old upstream releases or images as Kexra artifacts.
- Never run active security testing outside an explicit written authorization and isolated environment.

The former Korean upstream-sync maintainer manual is historical material, not the current Kexra release policy.
