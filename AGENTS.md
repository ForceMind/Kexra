# Artex — instructions for coding agents

## Repository context

- This is `ForceMind/Artex`, an independently maintained **private import** of `jiwoochris/artex-ko` (a Korean localization of `Autumn-27/ARTEX`). The original source commit, license and tree hash are documented in `docs/IMPORT_PROVENANCE.md`.
- Go (`cmd/`, `server/`, `agent/`, `db/`, etc.) + Next.js/TypeScript (`web/`) + PostgreSQL. The Go module's original import path is deliberately retained; do not mass-rename it without a complete migration.
- For repository-specific development instructions, read `docs/REPOSITORY_GUIDE.zh-CN.md`. The default product language is Simplified Chinese (zh); Chinese README is the main entrypoint, while historical Korean material remains available as README.ko.md.

## Working agreements

- Make incremental, reviewable changes. Do not reformat unrelated localization code or rewrite embedded agent prompts just to tidy the repository.
- Preserve attribution, AGPL-3.0 license, security notices, and existing CLI/config compatibility.
- Never commit local `.env*` secrets, credentials, JWT keys, runtime data, customer/target assets, evidence or captured traffic. Do not paste secrets into CI logs or issues.
- Do not run autonomous penetration tests, active scans or exploit chains against third-party hosts. Testing must be limited to explicitly authorized, isolated fixtures; code review and static analysis are preferred.
- When changing behavior, extend relevant Go or Web unit tests, and update the corresponding documentation.
- The default UI locale is zh (Simplified Chinese), with ko retained as an explicitly selectable build-time compatibility locale. Prefer next-intl keys over hardcoded user-facing strings, keep both locale JSON key sets in sync, and preserve API protocol identifiers exactly.
- Do not assume the original upstream Docker Hub image is accessible, or that the new repository has published a working image.
- Do not publish releases, Docker images, or change production credentials without verifying the target namespace and approvals.

## Verification commands

```bash
go build ./...
go vet ./...
go test ./... -count=1
python3 -I scripts/check-doc-links.py
cd web && npm ci && npm run check && npm run build:static
```

Go commands need the toolchain from `go.mod` and dependencies. DB integration tests need an isolated PostgreSQL 16 service. When a command cannot run due to missing toolchains, network, DB or secrets, report it explicitly and do not mark it green.

## Handoff

Record changed files, tests attempted/results, security implications, migration steps and any remaining limitations in the PR description. Avoid auto-merging unvalidated server-side or release changes.
