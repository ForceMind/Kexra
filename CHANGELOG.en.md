# Kexra Changelog

For the full record see [CHANGELOG.md](CHANGELOG.md). Kexra independently maintains derived software, but the old ARTEX project’s releases are not Kexra releases.

## Unreleased — Independent brand merged on 2026-10-14

- [PR #1](https://github.com/ForceMind/Kexra/pull/1) merged Kexra branding, SVG icons, `cmd/kexra` entrypoint and `github.com/ForceMind/Kexra` module, scripts and package names.
- Compatibility retained for legacy `ARTEX_*` settings, original database contents and user sessions. Applicable AGPL-3.0 and MIT licenses, copyright and source provenance preserved.
- Verified Go build/vet/tests, six PostgreSQL 16 integration packages, Chinese Web build, detection regression, five-platform package checks and isolated service/login/restart behavior. See [migration evidence](docs/MIGRATION.md).
- The six normal GitHub Actions workflows remain manual-only for cost control.

**Still pending:** complete Docker image build/runtime, first official Release and updater rollback, legacy language cleanup, and a public repository security/license review. No production-ready release has been published.

## Inherited history

Older project releases are kept as historical reference in [CHANGELOG.zh.md](CHANGELOG.zh.md) with origin details in [docs/IMPORT_PROVENANCE.md](docs/IMPORT_PROVENANCE.md).
