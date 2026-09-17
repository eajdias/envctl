# Contributing to envctl

Thank you for your interest in contributing to envctl!

## How to Contribute

1. **Fork** the repository
2. **Clone** your fork locally
3. Create a **branch** for your feature or fix:
   ```bash
   git checkout -b feat/my-feature
   ```
4. Make your changes and **commit** with conventional messages:
   ```bash
   git commit -m "feat: add support for X"
   ```
5. **Push** to your fork:
   ```bash
   git push origin feat/my-feature
   ```
6. Open a **Pull Request** against the main repository

## Commit Convention

Use [Conventional Commits](https://www.conventionalcommits.org/) format:

| Prefix | Purpose |
|--------|---------|
| `feat:` | New feature |
| `fix:` | Bug fix |
| `chore:` | Maintenance, deps, tooling |
| `docs:` | Documentation only |
| `refactor:` | Code change that neither fixes a bug nor adds a feature |
| `test:` | Adding or updating tests |
| `perf:` | Performance improvement |

Examples:
```
feat: add Docker Engine provisioning for Linux
fix: correct LSP detection on ARM64
chore: update Go to 1.22
```

## Branch Naming

Use semantic branch names:

- `feat/description` — new features
- `fix/description` — bug fixes
- `docs/description` — documentation changes
- `chore/description` — maintenance

## Development Setup

### Linux / macOS

```bash
./bootstrap.sh
envctl run all
envctl doctor
```

### Windows

```powershell
.\bootstrap.ps1
envctl run all
envctl doctor
```

### Building from Source

```bash
go build -o envctl ./cmd/envctl
```

## Code Style

- **Language:** Code and comments in English
- **Architecture:** Clean Architecture with SOLID principles
- **Typing:** Strict typing, no `any` unless necessary
- **Error handling:** Always handle errors explicitly
- **Idempotency:** Every operation must be safe to run multiple times

## Running Checks

Before submitting a PR, ensure:

```bash
go build ./...          # Compiles without errors
go vet ./...            # No static analysis issues
go test ./...           # All tests pass
envctl doctor           # Environment health check
envctl doctor --fix     # Auto-remediate known issues (optional)
```

## Project Structure

```
cmd/envctl/             Entry point
internal/
  ui/cli/               Cobra commands (run, doctor, snapshot, version)
  usecase/              Business logic (provisioning, audit, cleanup)
  infra/                Platform adapters (apt, winget, git, filesystem)
  domain/               Entities and repository interfaces
manifests/              Declarative YAML specs (packages, shell, lsp, skills)
configs/                Config file templates (opencode.json, AGENTS.md, etc.)
```

## What NOT to Submit

- Secrets, API keys, or credentials
- Platform-specific tweaks that only work on your machine
- Dependencies without clear justification
- Changes that break idempotency

## Questions?

Open an issue or start a discussion on GitHub.
