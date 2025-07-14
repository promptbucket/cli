# PromptBucket CLI (`pbt`)

Package and manage prompts using simple YAML manifests.

## Installation

### Homebrew (macOS/Linux)

```bash
brew install promptbucket/tap/promptbucket
```

### Go Install

```bash
go install github.com/promptbucket/cli@latest
```

### Debian/Ubuntu

```bash
# Download the .deb package from releases
wget https://github.com/promptbucket/cli/releases/latest/download/promptbucket_Linux_x86_64.deb
sudo dpkg -i promptbucket_Linux_x86_64.deb
```

### Manual Installation

Download the appropriate binary for your platform from the [releases page](https://github.com/promptbucket/cli/releases).

## Commands

- `pbt init` – scaffold `promptbucket.yaml`.
- `pbt build` – create a `.pbt` archive.
- `pbt completion` – generate shell completions.

## Development

### Building

```bash
make vet test build
```

### Releasing

Create a new tag and push it to trigger the release:

```bash
git tag v0.1.0
git push origin v0.1.0
```

## Example

See [examples/hello](examples/hello) for a sample manifest.
