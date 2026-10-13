# Kexra

**An AI multi-agent workspace for authorized security assessment and evidence management.**

Kexra integrates task orchestration, asset and exploration graphs, traceable findings, evidence review, reports, and human oversight into a Web UI. The default UI build is Simplified Chinese.

> **Status:** The GitHub repository is now `ForceMind/Kexra`. The [independent-brand PR #1](https://github.com/ForceMind/Kexra/pull/1) has been merged into `main` after Go, Web, PostgreSQL, detection, five-target packaging, and isolated local authentication smoke checks. No Kexra Docker image or production release has been verified.
>
> **Scope:** Work only in your own systems, explicitly authorized environments, or isolated labs. Application approval and interception features are not a hardened sandbox.

## Main capabilities

- Planner/worker orchestration and human interaction
- Persistent asset, task, fact and finding relationships
- Evidence tracking, report generation, and retesting
- Go backend, Next.js/React frontend, PostgreSQL
- LLM integrations, MCP/Skills, notification and traffic components

## Developer quick start

~~~bash
git clone https://github.com/ForceMind/Kexra.git
cd Kexra
cd web && npm ci && npm run check && npm run build:static && cd ..
go build ./...
go test ./... -count=1
~~~

These commands verify source compilation, **not** application deployment or the safety of live tests. The Go entrypoint is `cmd/kexra` and the module path is `github.com/ForceMind/Kexra`. Existing legacy database/configuration identifiers remain for compatibility.

For detailed instructions see [installation](docs/INSTALLATION.md), [configuration](docs/CONFIGURATION.md), [architecture](docs/ARCHITECTURE.md), [development](docs/DEVELOPMENT.md), [security policy](SECURITY.md), and the [Chinese documentation index](docs/README.md).

## Licensing

This is independently maintained software based on existing open-source code, **not a clean-room rewrite**. Preserve the applicable [AGPL-3.0](LICENSE), [MIT notices for frontend components](web/LICENSE), and [source provenance](docs/IMPORT_PROVENANCE.md). See [third-party notices](THIRD_PARTY_NOTICES.md). A new brand does not remove inherited rights and obligations.
