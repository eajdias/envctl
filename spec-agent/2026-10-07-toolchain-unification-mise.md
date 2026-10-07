# Spec: toolchain unification on mise (absorb npm, drop go)

- Date: 2026-10-07 · Branch: `feat/toolchain-unification-mise` (to create)
- Rule: repo implementation flow (spec → branch → implement → PR → merge → release).
- Owner directive: prioritize standardization, simplification, unification;
  npm absorbed into mise; the go LSP is removed and Go standardizes on mise.

## 1. Objective
One toolchain manager for everything mise can own. After this change the only
runtimes/packages managers are `mise` (runtimes + npm-backed tools),
`pip`/`uv` (Python, PEP 668-aware, untouched), and the OS managers
(winget/apt/pacman/paru). `NpmManager`, `GoManager`, `PackageTypeNpm`,
`PackageTypeGo`, `GroupNpm`, `GroupGo` and the `gopls` LSP entry are deleted.

## 2. Non-goals
- `PipManager` stays: already optimal (`uv tool install` first, PEP 668
  probe + `--break-system-packages` fallback, `toolchain_managers.go:153-187`).
- No change to `run pip`, OS managers, providers Fase 0 architecture, or
  CommandCode (`${VAR}` env expansion untouched).
- No new `run` targets; `run mise` absorbs the npm workload.

## 3. Ground-truth inventory (verified 2026-10-07)
- `manifests/packages.yaml`: 8× `type: npm` (pnpm, `@playwright/cli`,
  mcp-ssh-manager, stylelint, typescript, prettier, sqllens-language-server,
  command-code), 10× `type: pip`, 1× `type: mise` (node@24.19.0), 0× `type: go`.
- `manifests/lsp.yaml` (14): 12× `install_type: npm`, 1× `install_type: go`
  (gopls, `lsp.yaml:26-33`), 1× `install_type: winget` (powershell).
- Code: `NpmManager` + `UserLocalPrefix` (`toolchain_managers.go:18-78`),
  `GoManager` (`:213+`), registration (`ui/cli/root.go:103-105`),
  `update.go` groups `GroupNpm`/`GroupGo` (`update.go:21-24,29-42,44-64`),
  providers npm install/update for CommandCode CLI
  (`provision_providers.go:301-314,353-370`, `npmPkg: "command-code"`),
  `cli/update.go:63,123` group lists.
- Deliberate decision being reversed: `mise_manager.go:14-15`
  ("mise only owns runtimes here, never packages").

## 4. Design
- **ID syntax**: migrated entries use the mise backend prefix —
  packages `id: npm:<pkg>` (e.g. `npm:pnpm`, `npm:@playwright/cli`,
  `npm:command-code`); LSP `install_target: npm:<target>`.
  `MiseManager.Install` (`mise install <id>`) needs no change.
- **`miseToolName` becomes backend-aware** (`mise_manager.go:55-66`): the
  current last-`@` strip mangles scoped IDs (`npm:@playwright/cli` → `npm:`).
  Rule: keep the `backend:` prefix for the `mise ls` query; strip an
  `@suffix` only when the tail looks like a version (starts with a digit)
  and the head is non-empty. Cases: `node@24.19.0`→`node`,
  `npm:typescript@5.6`→`npm:typescript`, `npm:@playwright/cli` unchanged,
  `npm:pnpm` unchanged. Unit-test every case (TDD, M1-style).
- **Providers**: CommandCode CLI install/update moves from `PackageTypeNpm`
  to `MiseManager` with `npm:command-code`; update-resolution reuses the
  mise latest path (prove with test; `npmLatest` stays only if still
  referenced, else delete).
- **`update.go`**: delete `GroupNpm`/`GroupGo`, their `automatableGroup`
  cases and `updateCommand` branches. npm/go LSP targets flow through
  `GroupMise` (`mise install <target>` already handles `npm:` targets and
  the `@latest` guard).
- **Deletions**: `NpmManager`, `UserLocalPrefix` (iff no other caller —
  verify with grep at implementation), `GoManager`, `PackageTypeNpm`,
  `PackageTypeGo`, root.go registrations, `update_test.go:55-64,143` npm/go
  cases, `toolchain_managers_test.go` Go test, doctor/provision fixtures
  carrying npm/go types (revealed by `go test`).
- **`provision_lsp.go`**: generic dispatch over `InstallType` — no change
  expected; confirm by build + tests.

## 5. Manifest changes
- `packages.yaml`: 8 npm entries `type: npm` → `type: mise`,
  `category: npm` → `category: mise`, `id: npm:<pkg>`. node entry untouched.
- `lsp.yaml`: 12 npm entries `install_type: npm` → `install_type: mise`,
  `install_target: npm:<target>`; **delete the gopls entry** (accepted
  tradeoff: Go developers lose provisioned gopls; manual
  `go install golang.org/x/tools/gopls@latest` via the mise-managed Go
  toolchain remains available).
- `TestLSPSingleSource` keeps passing (lsp.yaml stays the single source).

## 6. Migration on existing machines (the load-bearing part)
`ToolchainDirs` ranks `~/.local/bin` (rank 2, where `NpmManager --prefix
~/.local` put binaries) **above** mise shims (rank 3)
(`executil/toolchain.go:18-25`). Without cleanup, stale npm copies shadow
the new shims and freeze old versions — the same class as the Arch
user-local rule in `AGENTS.md`.
- `run mise` performs a one-time legacy sweep: for each migrated binary
  (basenames from manifest `check_command`/`check_binary`), if a legacy
  file exists at `~/.local/bin/<bin>` (and `~/.local/lib/node_modules/<pkg>`
  dir) while the mise shim exists, move the legacy aside following the
  `archiveShadowedUserCopies` precedent (`provision_providers.go:291`) and
  log it. Rationale for move-aside over `.bak`: reinstallable artifacts,
  not user content; convention preserved via archive, not timestamp backup.
- Applies on Windows too (`UserLocalPrefix` is cross-platform `~/.local`).
- `~/go/bin` (rank 4) is out of scope: Go toolchain stays, only the gopls
  binary goes stale if present — remove `~/go/bin/gopls` in the same sweep
  iff it exists (it could only have come from envctl).

## 7. Docs updates (docs-sync)
- `docs/architecture.md`: managers list (`MiseManager` absorbs npm;
  `GoManager`/`VoltaManager` gone).
- Matrix + guides: only if they name npm/go provisioning paths (verify with
  grep at implementation; `run mise`/`run pip` lines already exist).
- `CHANGELOG.md`: entry under `[Unreleased]` at merge time.

## 8. Verification
- `go build ./...`, `go vet ./...`, `go test ./...`,
  `golangci-lint run --new-from-rev=origin/main`, `envctl-verify --git-push`.
- Live on this machine (owner's workstation): rebuild, `run mise`,
  `run lsp`, `doctor` green; prove shim-wins with `Get-Command <bin>`
  for every migrated binary (no `~/.local/bin` shadow); prove
  `mise ls npm:<pkg> --json` parses via the new unit tests.
- Pre-flip proof: `mise install npm:@playwright/cli` (scoped) and a
  multi-bin package (`vscode-langservers-extracted`, 5 servers) on a
  scratch basis before flipping the manifests.

## 9. Rollback
Revert the merge commit. Machines already migrated keep working mise copies;
doctor on the old binary would reinstall npm copies that shadow shims again
— document `run mise` re-run as the forward fix, not rollback.

## 10. Risks
- mise npm-backend edge cases (scoped IDs, multi-bin packages, cmdc
  latest-resolution) — mitigated by §8 pre-flip proofs (bite-sized, one
  package at a time).
- Windows mise behavior differences — covered by the CI windows leg, but
  live-proof on this workstation is the real gate.
- Stale-shadow misses (binaries installed outside the manifest list) —
  sweep is allowlist-based; anything else surfaces via doctor version
  checks, not silently.
