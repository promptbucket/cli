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
- `promptbucket build` – create a `.promptbucket` archive.
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

The CLI supports configuration through environment variables and `.env` files.

**Environment Variables:**
- `PROMPTBUCKET_BASE_URL` - API base URL (default: https://harbor.promptbucket.co)
- `PROMPTBUCKET_API_VERSION` - API version (default: v1)

**Environment Files:**
The CLI automatically loads environment variables from `.env` files in the following order:

1. `.env` - Base environment file (should be committed)
2. `.env.local` - Local overrides (should be gitignored)
3. `.env.{ENVIRONMENT}` - Environment-specific (e.g., `.env.production`)
4. `.env.{ENVIRONMENT}.local` - Environment-specific local overrides

Later files override earlier ones. Set `DEBUG=true` to see which files are loaded.

**Example .env file:**
```bash
# .env
PROMPTBUCKET_BASE_URL=https://api.promptbucket.io
PROMPTBUCKET_API_VERSION=v2
ENVIRONMENT=development
```

**Example .env.local file:**
```bash
# .env.local (gitignored - for local development only)
PROMPTBUCKET_BASE_URL=http://localhost:8080
DEBUG=true
```

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

## Examples

### Basic Example
See [examples/hello](examples/hello) for a simple greeting persona.

### Advanced Examples
- [examples/code-reviewer](examples/code-reviewer) - Senior software engineer for code reviews
- [examples/data-analyst](examples/data-analyst) - Expert data analyst with statistical background

## Manifest Structure

### Basic Fields
```yaml
name: my-persona          # Required: lowercase, alphanumeric with dashes/underscores
version: 0.1.0           # Required: semantic versioning
licence: Apache-2.0      # Required: license identifier
description: Brief description of the persona
authors: [Your Name]     # Optional: list of authors
tags: [ai, assistant]    # Optional: categorization tags
```

### Persona Configuration
Define rich character details that eliminate the need for "You are..." statements:

```yaml
persona:
  # Identity
  name: Alex the Code Reviewer        # Persona's name
  role: Senior Software Engineer      # Professional role
  personality: [thorough, friendly]   # Character traits
  
  # Expertise & Background
  expertise: [Python, Go, Security]   # Technical skills
  experience: 8+ years               # Experience level
  background: CS degree with focus on distributed systems
  
  # Communication Style
  tone: professional                 # Communication tone
  style: structured                  # Response style
  language_level: expert-level       # Technical complexity
  interaction_style: asks clarifying questions
  
  # Behavioral Traits
  approach: systematic               # Problem-solving method
  focus: [security, performance]    # Priority areas
  constraints: [always explain reasoning]
  preferences: [provide code examples]
  output_format: structured markdown
```

### Variables & Prompts
```yaml
variables:
  - name: language
    description: Programming language being reviewed
    example: Python
    enum: [Python, Go, JavaScript]  # Optional: restrict to specific values

prompt: |-
  Review the provided {{language}} code.
  Focus on code quality and best practices.
```

## How Personas Work

When you build or run a prompt with a persona defined, the system automatically generates a comprehensive character description that precedes your main prompt:

```markdown
# Identity
You are Alex the Code Reviewer, a Senior Software Engineer.

# Background & Expertise
Experience: 8+ years
Areas of expertise: Python, Go, Security

# Communication Style
Personality traits: thorough, friendly
Tone: professional
Communication style: structured

# Guidelines
Constraints:
- always explain reasoning

Preferences:
- provide code examples

---

Review the provided Python code.
Focus on code quality and best practices.
```

This eliminates repetitive "You are..." statements and creates consistent, well-defined AI assistants.
