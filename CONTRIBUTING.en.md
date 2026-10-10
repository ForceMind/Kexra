# Contributing to Kexra

Kexra is an independently maintained AI multi-agent workspace for **authorized** security assessment and evidence management. Its code includes third-party software governed by AGPL-3.0 and MIT notices.

## Before contributing

Read the [Chinese contribution guide](CONTRIBUTING.md), [development workflow](docs/DEVELOPMENT.md), [security policy](SECURITY.md), and [license notices](THIRD_PARTY_NOTICES.md).

1. Keep PRs narrowly scoped and describe their compatibility impact.
2. Work only with assets that you own or are explicitly authorized to test. Never include client data, actual tokens, target lists, or private traffic in issues.
3. Do not replace real evidence, detection signatures, source code references or protocol fields simply to remove inherited naming.
4. Preserve original license files and copyright statements attached to incorporated code.
5. Report the exact build/test commands you ran, distinguishing passed, failed and unexecuted checks.

~~~bash
go build ./...
go vet ./...
go test ./... -count=1
cd web && npm ci && npm run check && npm run build:static && cd ..
python3 -I scripts/check-doc-links.py
~~~

The ongoing brand migration is not fully validated: see [migration](docs/MIGRATION.md) and [draft PR #1](https://github.com/ForceMind/Kexra/pull/1). The Chinese guide is the primary maintained reference.
