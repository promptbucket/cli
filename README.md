# PromptBucket CLI

**Stop re-explaining your project to AI every day.**

PromptBucket gives your AI a persistent memory of your project. Run it once per project. Your AI remembers your stack, your decisions, and your preferences — across every session.

Works with Claude Code, Cursor, Cline, and any MCP-compatible client.

---

## Install

```bash
brew install promptbucket/tap/promptbucket
```

Or download a binary from the [releases page](https://github.com/promptbucket/cli/releases).

---

## Quick start

**1. Scaffold a persona in your project**

```bash
cd your-project/
promptbucket init --name myteam/backend --role "Senior Backend Engineer" --expertise "Go, MongoDB, API design"
```

This creates `.promptbucket/` in your project:

```
.promptbucket/
├── persona.yaml     ← identity, model, memory config
├── system.md        ← your AI's instructions (edit this)
├── memory/          ← private, gitignored automatically
├── seed-memory/     ← committed, shared with teammates
└── README.md
```

**2. Edit `.promptbucket/system.md`**

Add what your AI should always know about your project:

```markdown
# Senior Backend Engineer

You are a senior backend engineer on this project.

## About this project

- Stack: Go 1.22, MongoDB, Echo framework, Docker
- Auth: JWT via golang-jwt/jwt
- Deploy: Docker Compose locally, GCP Cloud Run in production
- No ORMs — raw MongoDB driver queries only

## How you work

- Review error paths before happy paths
- Keep functions under 40 lines
- Always check for nil pointer dereferences
```

**3. Connect Claude Code**

Add to your project's `.claude/mcp.json` (create the file if it doesn't exist):

```json
{
  "mcpServers": {
    "persona": {
      "command": "promptbucket",
      "args": ["serve"]
    }
  }
}
```

Claude Code will now automatically get your persona's context on every session. No more re-explaining.

**4. Connect Cursor**

Add to `.cursor/mcp.json`:

```json
{
  "mcpServers": {
    "persona": {
      "command": "promptbucket",
      "args": ["serve"]
    }
  }
}
```

---

## Commands

| Command | What it does |
|---|---|
| `promptbucket init` | Scaffold `.promptbucket/` in the current project |
| `promptbucket serve` | Start MCP server (for Claude Code, Cursor, Cline) |
| `promptbucket test` | Run prompt tests (legacy mode) |
| `promptbucket version` | Show version |

### `promptbucket init` flags

| Flag | Default | Description |
|---|---|---|
| `--name` | `my-persona` | Persona name (format: `author/name`) |
| `--role` | `Software Engineer` | Role description |
| `--expertise` | `Go, API design` | Comma-separated expertise areas |
| `--tone` | `direct, pragmatic` | Communication style |
| `--provider` | `anthropic` | Model provider |
| `--model` | `claude-sonnet-4-6` | Preferred model ID |
| `--force` | false | Overwrite existing `.promptbucket/` |

### `promptbucket serve` flags

| Flag | Default | Description |
|---|---|---|
| `--persona` | `.promptbucket/` | Path to persona directory |

---

## How it works

`promptbucket serve` starts a local [MCP server](https://modelcontextprotocol.io) over stdio. Your MCP client (Claude Code, Cursor, etc.) connects to it and receives:

- **`persona://system-prompt`** — your full system prompt, combining identity + `system.md` + any seed memories
- **`persona://identity`** — identity fields from `persona.yaml` as JSON

Your AI reads this at session start. You never have to explain your stack, preferences, or constraints again.

---

## Memory

Memory is structured in layers:

| Layer | Location | What it stores | Committed? |
|---|---|---|---|
| Seed memory | `seed-memory/` | Curated facts your AI should always know | Yes — shared with teammates |
| Episodic | `memory/episodic/` | What happened in past sessions | No — local only |
| Semantic | `memory/semantic/` | Project facts and decisions | No — local only |
| Procedural | `memory/procedural/` | How your AI does recurring tasks | No — local only |

Private memory (`memory/`) is gitignored automatically. Only `seed-memory/` is committed.

---

## Persona format

`.promptbucket/persona.yaml` follows [PERSONA-SPEC v0.1](../PERSONA-SPEC.md). The spec is an open standard — any tool can implement a runtime for it.

---

## License

Apache-2.0
