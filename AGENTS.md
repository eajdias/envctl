# envctl

Go CLI (Clean Architecture) that provisions and audits dev environments on
Windows 11, Ubuntu/Debian and Arch/CachyOS: system packages, shell/env/configs,
41 agent skills (OpenCode + CommandCode), 15 LSPs, Windows tweaks, and a local
verification gate wired both to the agent and to git.

Key entry points: `internal/ui/cli/` (cobra commands `run`, `doctor`,
`snapshot`, `commandcode`, `opencode`), `internal/usecase/` (business logic),
`internal/infra/` (platform adapters), `manifests/*.yaml` (declarative specs),
`configs/` (embedded templates, `//go:embed` via `assets.go`),
`configs/bin/envctl-verify` (the local gate, deployed to `~/.local/bin`).

## Project state (read this before changing anything)

- **`docs/os-and-agent-matrix.md`** — start here. What is provisioned on each OS
  and for each agent, the asymmetries found and their status, coverage per stack,
  and the checklist to touch every layer when adding a package, config file,
  skill or LSP.
- **`docs/verification.md`** — the local gate: per-stack checks, the scoping
  rules (linters on changed files, type checks/tests repo-wide), the modes
  (`--hook` static-only, `--git-push` complete, `--dry-run`) and the skips.
- **`docs/doctor-and-idempotency.md`** — what the audit verifies, `--fix`, the
  atomic backup and log conventions.
- **`CHANGELOG.md`** — released work, one section per release. Shipped work never
  stays under `[Unreleased]`.

Counts (packages, skills, LSPs) drift by design; the manifests are the source of
truth and `envctl doctor` asserts the machine against them. When a number in the
docs disagrees with a manifest, the manifest wins — fix the doc.

## Conventions

- Code, comments and commits in English; conversation with the user in PT-BR.
- Idempotent operations with atomic backup (`.bak.YYYYMMDD-HHMMSS`); never
  overwrite user-owned content — declare a `merge:` mode or `seed_if_missing`.
- Verify with `go build ./...`, `go vet ./...`, `go test ./...` and
  `golangci-lint run --new-from-rev=origin/main`, or simply `envctl-verify
  --git-push` (the same gate the hooks run; `--dry-run` shows what it would do).
- The verifier script has its own tests in `internal/usecase/verify_script_test.go`
  — a change to it must keep them passing, since it blocks pushes when broken.