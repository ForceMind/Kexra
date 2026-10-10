# Kexra

**AI multi-agent security assessment and evidence-management workspace.**

Kexra is independently maintained under the ForceMind organization, based on previously open-source work. It is not a clean-room original implementation. Only test systems you own or are explicitly authorized to assess.

Key capabilities: agent planning/execution, asset and exploration graphs, traceable findings and retesting, reports, a Go API and a Next.js web dashboard. Existing approval and interception rules must not be considered a hardened security sandbox.

## Local development

Go 1.26, Node.js 22, npm and PostgreSQL 16 are required. From the project root:

```bash
cd web && npm ci && npm run build:static && cd ..
go build ./...
go test ./... -count=1
```

Repository rename is pending admin action: `ForceMind/Artex` → `ForceMind/Kexra`. There is no verified Kexra release image yet.

## Licensing and acknowledgments

The project contains AGPL-3.0-covered code, and the web application contains separately MIT-licensed material. Original license and copyright notices remain in [LICENSE](LICENSE), [web/LICENSE](web/LICENSE), and [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md). The original source snapshot is recorded in [docs/IMPORT_PROVENANCE.md](docs/IMPORT_PROVENANCE.md).
