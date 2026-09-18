# envctl

Go CLI (Clean Architecture) that provisions and audits dev environments on
Windows 11 and Ubuntu/Debian: system packages, shell/env/configs, 41 agent
skills (OpenCode + CommandCode), 15 LSPs, Windows tweaks.

Key entry points: `internal/ui/cli/` (cobra commands `run`, `doctor`,
`snapshot`, `commandcode`, `opencode`), `internal/usecase/` (business logic),
`internal/infra/` (platform adapters), `manifests/*.yaml` (declarative specs),
`configs/` (embedded templates, `//go:embed` via `assets.go`).

Conventions: code/comments/commits in English; idempotent operations with
atomic backup (`.bak.YYYYMMDD-HHMMSS`); verify with `go build ./...`,
`go vet ./...`, `go test ./...`, `envctl doctor` before declaring done.