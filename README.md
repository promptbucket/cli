# PromptBucket CLI (`pbt`)

*Package, sign & publish AI-prompt “personas” as versioned `.pbt` archives.*

[![CI](https://github.com/promptbucket/cli/actions/workflows/ci.yml/badge.svg)](https://github.com/promptbucket/cli/actions/workflows/ci.yml)
[![Go Report](https://goreportcard.com/badge/github.com/promptbucket/cli)](https://goreportcard.com/report/github.com/promptbucket/cli)
[![License](https://img.shields.io/badge/license-Apache--2.0-blue.svg)](LICENSE)

---

## What’s inside a `.pbt` file?

my-persona.pbt
└─ promptbucket.yaml # manifest + prompt text + variables
tests/ … # optional Promptfoo cases

yaml
Copy
Edit

A **single YAML manifest** keeps everything—metadata *and* prompt text—in one
place. No more `ppkg.json` + `prompt.md` split.

---

## ✨ Commands

| Command | Purpose |
|---------|---------|
| `pbt init` | Scaffold `promptbucket.yaml` with a boilerplate prompt. |
| `pbt build` | Tar + gzip → prepend magic **PBKT␀** → compute digest → output `.pbt`. |
| `pbt push` | Push to any OCI registry (GHCR, Harbor, GCR); optional Cosign sign. |
| `pbt pull` | Pull, verify digest & magic bytes, extract locally. |
| `pbt lint` | Validate YAML against schema, licence & size limits. |
| `pbt sign` / `pbt verify` | Attach / verify Sigstore signature. |
| `pbt inspect` | Pretty-print manifest, variables, template names. |
| `pbt test` | Run Promptfoo regression tests in the package. |
| `pbt completion` | Generate shell completions (bash, zsh, fish, pwsh). |

---

## 🚀 Quick start

```bash
# 1 · Install (Go ≥1.22)
go install github.com/promptbucket/cli@latest

# 2 · Create a package
mkdir hello && cd hello
pbt init        # writes promptbucket.yaml
# edit promptbucket.yaml -> see example below
pbt build       # hello-0.1.0.pbt

# 3 · Publish (GitHub Container Registry)
echo $GITHUB_TOKEN | pbt login ghcr.io --username $USER --password-stdin
pbt push myorg/hello:0.1.0 hello-0.1.0.pbt --sign
Sample promptbucket.yaml

yaml
Copy
Edit
name: hello
version: 0.1.0
licence: Apache-2.0
authors: [slipdisk]
description: Friendly greeting persona
variables:
  - name: name
    description: Person to greet
    example: Alice
prompt: |-
  {{system}} You are a friendly assistant.
  {{user}}  Say hello to {{name}} enthusiastically!
🔧 Installation options
Platform	Command
Homebrew (macOS/Linux)	brew install promptbucket/tap/pbt
Go ≥ 1.22	go install github.com/promptbucket/cli@latest
Pre-built binaries	Download from Releases
Docker	docker run --rm ghcr.io/promptbucket/cli:latest pbt …

🗂 Repo layout
csharp
Copy
Edit
cli/
├── cmd/             # Cobra commands
│   ├── root.go
│   ├── build.go
│   ├── init.go
│   └── …
├── internal/        # packager, registry, signer (phases 2+)
├── spec/            # manifest.schema.yaml (JSON-Schema on YAML)
├── examples/hello/  # promptbucket.yaml sample
├── Makefile
├── .github/…        # CI workflow
└── README.md
🧪 Tests & CI
bash
Copy
Edit
make lint test build   # golangci-lint, race detector, unit tests
GitHub Actions runs on Ubuntu, macOS & Windows for every push / PR.

🤝 Contributing
Fork → branch feat/<topic>

Ensure make lint test passes

Use Conventional Commits (e.g. feat: add sign cmd)

Open a PR → CI must be green before merge

📜 License
Apache License 2.0 – see LICENSE.

Happy prompt-packing!

yaml
Copy
Edit

---

#### `.gitignore` (copy if you don’t have one yet)

```gitignore
# Build artifacts
/bin/
/dist/
/*.pbt
*.sbom
*.sig

# Go cache & deps
vendor/
.gomodcache/
.gocache/

# Editor
.idea/
.vscode/
*.code-workspace

# OS
.DS_Store
Thumbs.db

# Logs
*.log

# Env
.env
.env.*

# Tests
*.test
*.out