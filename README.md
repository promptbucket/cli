# PromptBucket

Open-source CLI for testing and evaluating AI prompts across models.

Define tests in YAML. Run them against GPT, Claude, Gemini. Assert quality. Catch regressions. Track cost.

## Installation

### Homebrew (macOS/Linux)

```bash
brew install promptbucket/tap/promptbucket
```

### Go Install

```bash
go install github.com/promptbucket/cli@latest
```

### Binary

Download the appropriate binary for your platform from the [releases page](https://github.com/promptbucket/cli/releases).

## Quick Start

```bash
# 1. Create a starter config
promptbucket init

# 2. Add your API keys
export OPENAI_API_KEY=sk-...
export ANTHROPIC_API_KEY=sk-ant-...

# 3. Run tests
promptbucket test
```

## YAML Config Reference

Tests are defined in `promptbucket.yaml` (or specify a custom path with `--config`):

```yaml
tests:
  - name: "basic greeting"
    prompt: "Say hello in a friendly way"
    system: "You are a helpful assistant."  # optional, can be a file path
    models:
      - gpt-4o-mini
      - claude-sonnet-4-20250514
    assert:
      - type: contains
        value: "hello"
      - type: cost-below
        max: 0.01
```

### Fields

| Field    | Required | Description                                          |
|----------|----------|------------------------------------------------------|
| `name`   | yes      | Human-readable test name                             |
| `prompt` | yes      | The user prompt to send                              |
| `system` | no       | System prompt (string or path to .txt/.md file)      |
| `models` | yes      | List of models to test against                       |
| `assert` | no       | List of assertions to run on each response           |

### System Prompt as File

If `system` contains a `/` or ends in `.txt` or `.md`, it is treated as a file path and its contents are used as the system prompt:

```yaml
system: "prompts/code-reviewer.md"
```

## Assertion Types

| Type           | Fields     | Description                                       |
|----------------|------------|---------------------------------------------------|
| `contains`     | `value`    | Response contains substring (case-insensitive)    |
| `not-contains` | `value`    | Response does NOT contain substring               |
| `regex`        | `pattern`  | Response matches regular expression               |
| `cost-below`   | `max`      | API call cost is below threshold (USD)            |
| `llm-judge`    | `criteria` | LLM evaluates response against criteria           |

### Examples

```yaml
assert:
  # String matching
  - type: contains
    value: "def"

  # Negative matching
  - type: not-contains
    value: "TODO"

  # Regex pattern
  - type: regex
    pattern: "def\\s+\\w+\\("

  # Cost guard
  - type: cost-below
    max: 0.05

  # LLM-as-judge
  - type: llm-judge
    criteria: "The code correctly implements a prime checker with edge cases"
```

## Supported Models

| Provider  | Models                                                                 | Env Variable       |
|-----------|------------------------------------------------------------------------|---------------------|
| OpenAI    | gpt-4o, gpt-4o-mini, gpt-4.1, gpt-4.1-mini, o1, o3, o3-mini, o4-mini | `OPENAI_API_KEY`    |
| Anthropic | claude-sonnet-4-20250514, claude-haiku-4-5-20251001, claude-opus-4-20250514       | `ANTHROPIC_API_KEY` |
| Google    | gemini-2.5-pro, gemini-2.5-flash, gemini-2.0-flash                    | `GOOGLE_API_KEY`    |

## CLI Reference

```
promptbucket init          Create a starter promptbucket.yaml
promptbucket test          Run all tests
promptbucket version       Print version info

Flags:
  --config string      Config file path (default "promptbucket.yaml")
  --ci                 CI mode: no color, exit 1 on failure
  --concurrency int    Max concurrent provider calls (default 4)
  --filter string      Filter tests by name substring (test command only)
  --force              Overwrite existing config (init command only)
```

## CI/CD Usage

### GitHub Actions

```yaml
name: Prompt Tests
on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Install PromptBucket
        run: |
          curl -sSL https://github.com/promptbucket/cli/releases/latest/download/promptbucket_Linux_x86_64.tar.gz | tar xz
          sudo mv promptbucket /usr/local/bin/

      - name: Run prompt tests
        env:
          OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}
          ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}
        run: promptbucket test --ci
```

### Environment Variables

Create a `.env` file (gitignored) for local development:

```bash
OPENAI_API_KEY=sk-...
ANTHROPIC_API_KEY=sk-ant-...
GOOGLE_API_KEY=AI...
PROMPTBUCKET_JUDGE_MODEL=gpt-4o-mini  # model used for llm-judge assertions
```

The CLI automatically loads `.env` and `.env.local` files.

## Contributing

```bash
# Clone and build
git clone https://github.com/promptbucket/cli.git
cd cli
make build

# Run tests
make test

# Lint
make lint
```

## License

MIT
