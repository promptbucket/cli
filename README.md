# PromptBucket CLI

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

### Package Management
- `promptbucket init` – scaffold `promptbucket.yaml`.
- `promptbucket build` – create a `.pbt` archive.
- `promptbucket completion` – generate shell completions.

### Authentication
- `promptbucket login` – authenticate with PromptBucket using browser OAuth.
- `promptbucket logout` – sign out and clear stored credentials.
- `promptbucket whoami` – show current user information.

#### Authentication Examples

```bash
# Login with Google (default)
promptbucket login

# Login with GitHub
promptbucket login --provider github

# Login with custom callback port
promptbucket login --port 3456

# Check current user
promptbucket whoami

# Logout
promptbucket logout
```

#### How Authentication Works

1. **CLI starts local callback server** on port 3456 (configurable)
2. **Opens browser** to OAuth endpoint with CLI callback URL: `/v1/auth/google?redirect_uri=http://localhost:3456/callback`
3. **User authenticates** with Google/GitHub through OAuth provider
4. **Server redirects** back to CLI callback server with auth data as URL parameters
5. **CLI displays success page** that auto-closes after 2 seconds
6. **CLI saves token** to `~/.promptbucket/token.json` with secure permissions
7. **Future API calls** use stored token for authentication

#### Backend Integration Requirements

For OAuth authentication to work, your backend needs to:

- **Accept redirect_uri parameter** in OAuth endpoints (`/v1/auth/google`, `/v1/auth/github`)
- **Return auth data as URL parameters** in the callback:
  ```
  http://localhost:3456/callback?token=JWT_TOKEN&user_id=123&email=user@example.com&name=John+Doe&provider=google
  ```
- **Handle errors** by redirecting with error parameter:
  ```
  http://localhost:3456/callback?error=Authentication+failed
  ```

#### Configuration

Environment variables:
- `PROMPTBUCKET_BASE_URL` - API base URL (default: https://harbor.promptbucket.co)
- `PROMPTBUCKET_API_VERSION` - API version (default: v1)

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
