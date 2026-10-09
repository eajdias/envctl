# Spec: mise-managed MCP servers + deploy-time absolute shim paths

- Date: 2026-10-09 (expanded scope; supersedes `2026-10-09-mcp-ssh-manager-shim-path.md`, deleted)
- Branch: `fix/mcp-servers-mise-shim-path` (from `origin/main`)
- Skills loaded: `writing-plans`, `agent-memory`, `code-playbooks` (`references/go.md`, `references/docs-sync.md`)
- Status: spec only — no production code touched

## 1. Problem (verified evidence, 2026-10-09 session)

`opencode mcp list` reported:

```text
✗ ssh-manager  failed: Connection closed: MCP server process exited with code 1:
'mcp-ssh-manager' não é reconhecido como um comando interno ou externo
```

Root cause, proven in-session:

1. `configs/opencode.json:784-794` deploys `"command": ["mcp-ssh-manager"]` (bare name, mise shim).
2. The mise shims dir is injected into the *interactive* PATH only; the background
   service (`opencode.exe serve --service`, started 2026-10-05) inherited an env
   without it, so the MCP child fails to spawn.
3. The binary itself is healthy (6 servers / 37 tools via absolute path; `ssh_list_servers` live).
4. Doctor check `PATH (mise shims)` (1.5b, `internal/usecase/doctor_system.go:164-194`)
   is green — the audit cannot see the stale service env. Fixing PATH alone does
   not revive the MCP; only `opencode service restart` does (config is not hot-reload either).

Local hotfix (deployed file only): absolute shim path + `disabled: false` →
`✓ ssh-manager connected`. Not durable: `manifests/shell.yaml:29-43` has no
`merge:`/`seed_if_missing` → `MergeOverwrite` (`internal/domain/entity/models.go:64`);
live proof: `envctl doctor` reports
`⚠ WARN ConfigFile ~/.config/opencode/opencode.json — Content diverges from provisioned source`
(139 checks: 136 passed, 3 warnings, 0 errors; other 2 WARNs pre-existing, unrelated).

Scope expansion (owner decision 2026-10-09): the same fragility class covers all
local MCP servers. `configs/commandcode/mcp.json:1-32` shows the second runtime
uses different launchers for the same servers (brave via `npx`, chrome via `bunx`),
and `bunx`/`npx` are themselves bare PATH lookups. All three local servers go
mise-managed with absolute deploy-time paths. `context7`/`exa` are remote — untouched.
`context7`'s `@latest` pin is a known smell, explicitly out of scope.

## 2. Goal / non-goals

Goal: all three local MCP servers (`ssh-manager`, `brave`, `chrome-devtools`) on
both runtimes resolve via mise-managed version-pinned shims with absolute
deploy-time paths, independent of the spawning process's PATH age. Templates stay
portable (no usernames/hosts in git); `doctor` stays green.

Non-goals (explicit):

- N1: `disabled: true` / `"enabled": false` defaults untouched everywhere (matrix §3,
  asymmetry #28: intentional opt-in via `/mcp`). The local `disabled: false` flip
  is reverted by design on next provisioning.
- N2: remote servers (`context7`, `exa`) and `context7`'s `@latest` — separate follow-up.
- N3: doctor detection of stale-service env — unobservable; the guidance note in
  `docs/doctor-and-idempotency.md` (uncommitted, rides in this branch) stands.

## 3. Design

### 3.1 New manifest packages (`manifests/packages.yaml`, `type: mise`, no `os:` = all OSes)

| id | bin (verified via `npm view`) | template pin it must match |
|---|---|---|
| `npm:@brave/brave-search-mcp-server@2.1.4` | `brave-search-mcp-server` (single bin) | opencode `bunx @brave/brave-search-mcp-server@2.1.4`; commandcode `npx -y @brave/brave-search-mcp-server@2.1.4` |
| `npm:chrome-devtools-mcp@1.8.0` | `chrome-devtools-mcp` (bin key matches package name — the one `bunx`/`npx` executes) | opencode `bunx chrome-devtools-mcp@1.8.0`; commandcode `bunx chrome-devtools-mcp@1.8.0` |

Precedent for versioned IDs: `node@` handling (`provision_providers.go:273`);
`Install` passes `pkg.ID` verbatim to `mise install --yes` + `mise use -g`
(`mise_manager.go:120-140`, `--yes` covers the aube reputation gate), and
`miseToolName` keeps scoped `@version` IDs queryable (`mise_manager.go:74-86`).
Verified live: `mise ls-remote` resolves both packages (brave up to 2.1.4).
`IsInstalled` for backend IDs uses `mise ls` installed&&active
(`mise_manager.go:32-48`) — `check_command` is `omitempty` (`models.go:42`) and
unneeded; `auditEnvPackages` (`doctor_packages.go:20-53`) picks new IDs up with
zero doctor code changes. Category: `mise` (precedent: `npm:mcp-ssh-manager`,
`packages.yaml:255-259`).

Side benefit (stated, verified at implementation): `Oven-sh.Bun` is
Windows-only (`packages.yaml:105-110`), so bunx-launched MCPs are broken on
Linux today; mise is cross-platform, so this migration also repairs Linux.
`MiseShimDir` (`toolchain.go:27-36`) abstracts the per-OS shim dir; no username
is ever hardcoded.

### 3.2 Deploy-time overlay (both runtimes, table-driven)

Precedent: `withWindowsShellOverlay` (`opencode_config_shape.go:159-188`,
applied in `provision_shell.go:237-239`, mirrored in doctor at
`doctor_system.go:276-293`). New pure function
`withMCPShimOverlay(content []byte, patches map[string]string) []byte`
(new file `internal/usecase/mcp_shim_overlay.go`), where each patch maps an
exact bare fragment → absolute replacement. Per-server patches (templates unchanged):

opencode (`configs/opencode.json`):
- brave `["bunx", "@brave/brave-search-mcp-server@2.1.4", "--transport", "stdio"]`
  → `["<shim>/brave-search-mcp-server[.exe]", "--transport", "stdio"]`
- chrome `["bunx", "chrome-devtools-mcp@1.8.0", "--no-usage-statistics"]`
  → `["<shim>/chrome-devtools-mcp[.exe]", "--no-usage-statistics"]`
- ssh `["mcp-ssh-manager"]` → `["<shim>/mcp-ssh-manager[.exe]"]`

commandcode (`configs/commandcode/mcp.json`):
- brave `"command": "npx", "args": ["-y", "@brave/brave-search-mcp-server@2.1.4", "--transport", "stdio"]`
  → `"command": "<shim>/brave-search-mcp-server[.exe]"`, args minus the npx `-y` + package spec
- chrome `"command": "bunx", "args": ["chrome-devtools-mcp@1.8.0", "--no-usage-statistics"]`
  → `"command": "<shim>/chrome-devtools-mcp[.exe]"`, args minus the package spec
- ssh `"command": "mcp-ssh-manager"` → `"command": "<shim>/mcp-ssh-manager[.exe]"`

Rules (all locked by tests): apply a patch only on exact single match, else pass
through; `json.Valid` gate, invalid → original; CRLF→LF first (lesson 2026-10-06);
idempotent (no bare token left after run). Shim resolution at call site:
`filepath.Join(executil.MiseShimDir(home), bin)` / `bin+".exe"`, whichever
`executil.IsExecutableFile` (`toolchain.go:124-130`) confirms; none exists →
omit that server's patch (fresh machine before `run mise`; doctor 1.5b + package
WARNs cover it). In full profiles packages precede shell (`run.go:259-266`), so
shims exist. `BRAVE_API_KEY` env wiring untouched on both runtimes.

### 3.3 Doctor parity (same change, mandatory)

Extend the `cf.ID == "opencode_config"` mirror block (`doctor_system.go:281-283`)
to apply the identical overlay for `opencode_config`, `opencode_config_linux`,
`commandcode_mcp` before byte comparison — else permanent drift WARN (already
demonstrated live, §1).

## 4. Files affected

| File | Change |
|---|---|
| `manifests/packages.yaml` | +2 mise IDs (§3.1) |
| `internal/usecase/mcp_shim_overlay.go` | NEW: pure table-driven overlay |
| `internal/usecase/mcp_shim_overlay_test.go` | NEW: unit + golden tests (TDD) |
| `internal/usecase/provision_shell.go` | wire overlay for the 3 config IDs |
| `internal/usecase/doctor_system.go` | mirror overlay in `auditConfigFiles` |
| doctor drift test (locate: `grep -rn auditConfigFiles internal/usecase/*_test.go`) | overlay-aware no-drift case |
| `docs/doctor-and-idempotency.md` | 2026-10-09 note already present (uncommitted — rides along) |
| `docs/os-and-agent-matrix.md` | §3 asymmetry row for this migration |
| `CHANGELOG.md` | `Fixed:` entry under `[Unreleased]` |

## 5. Tasks (bite-sized, TDD, real commands)

Conventions: branch `fix/mcp-servers-mise-shim-path` from `origin/main`;
`.worktrees/fix-mcp-servers-mise-shim-path` + `session_move` before editing
(lesson 2026-09-26); re-check `git status --short` + `git worktree list`;
rebuild via `go build -o envctl.exe ./cmd/envctl` from repo root before any
`envctl run ...` (go:embed staleness — lessons 2026-08-26); `fix:` commits added
by path, never blind `git add -A`.

- [x] T0 — Baseline + locate: with MCPs still on bunx, record `opencode mcp list`
  and per-server tool counts (ssh 37, chrome 29 known; brave: record now — it is
  the post-migration equivalence oracle). Locate the doctor drift test file.
  Result: baseline numbers + test path, no assumptions forward.
  DONE 2026-10-09 (paced stdio handshake — servers exit on early EOF, so stdin
  stays open with 1.5s pacing): brave=8, chrome-devtools=29, ssh-manager=37.
  No dedicated drift test exists (only `doctor_system.go:245` + `doctor_audit.go:124`
  reference `auditConfigFiles`) → T3 adds coverage in `doctor_audit_test.go`.
- [x] T1 — Manifest: add the 2 mise IDs. `go build ./...`, manifest os-lint +
  `go test ./internal/domain/...` green. Then live: rebuilt `envctl run mise`,
  `mise ls --json` shows both pinned versions installed&&active,
  `envctl doctor` package rows OK. If a pinned version is gone from the registry,
  STOP and propose a template+manifest bump (owner approval — it changes pins).
  Rollback: `git checkout -- manifests/` + `mise uninstall <id>` (leave `use -g` config intact).
  DONE 2026-10-09: both installed pinned+active
  (`brave-search-mcp-server.exe`, `chrome-devtools-mcp.exe` + `chrome-devtools.exe`
  shims on disk); no registry-miss, no bump needed.
- [ ] T2 — RED: `mcp_shim_overlay_test.go` — all 6 patches (§3.2), `json.Valid`
  output, idempotency, pass-through on missing shim/invalid JSON/zero-or-multiple
  matches, CRLF fixture (pattern `ssh_overlay_test.go:139-155`).
  `go test ./internal/usecase/ -run TestMCPShimOverlay -v` → FAIL (undefined). Rollback: delete file.
- [ ] T3 — GREEN: implement `withMCPShimOverlay` + wire provision (3 IDs) + doctor
  mirror. `gofmt`, `go build ./...`, `go vet ./...`,
  `go test ./internal/usecase/ -run TestMCPShimOverlay -v` → PASS.
  Locks untouched and green:
  `go test ./internal/usecase/ -run 'TestShippedOpenCodeTemplatesHaveNativeShape|TestMCPServerParityAcrossAgents' -v`.
  Rollback: revert hunks; next `run shell` restores deployed files (atomic `.bak.`).
- [ ] T4 — Gate + docs: `go test ./...`, `golangci-lint run --new-from-rev=origin/main`
  (advisories per `docs/verification.md`); matrix §3 row; CHANGELOG `[Unreleased]`
  entry; docs-sync verification (claims cite `file:line`; no usernames/absolute
  paths in repo). Rollback: `git checkout -- docs/ CHANGELOG.md`.
- [ ] T5 — Live validation (Windows): `envctl run shell`, `envctl doctor`
  (ConfigFile rows OK, no new WARN vs 136/3/0 baseline of 2026-10-09);
  `opencode service restart`, enable all three via `/mcp`, `opencode mcp list`
  → 3× connected, tool counts equal T0 baseline; `opencode debug config` shows
  absolute commands. Linux proof optional via `vm-validation` skill. Rollback:
  restore provisioning `.bak.`, service restart.
- [ ] T6 — Close-out: `review` on Tab, `gh pr create --base main --head fix/mcp-servers-mise-shim-path`,
  merge, delete this spec (decisions live in matrix/memory/CHANGELOG), release only
  via manual Release Pipeline (never local tags).

## 6. Risks / unknowns / breaking changes

- Risks:
  - R1 Pinned version vanished from npm (chrome 1.8.0 < latest 1.10.1): T1 fails
    closed with a bump proposal instead of silently floating. Prob: low.
  - R2 Wrong default bin for `chrome-devtools-mcp` (two bins in package):
    `chrome-devtools-mcp` matches the invoked package name (npx/bunx convention);
    T0 baseline vs T5 catalog comparison is the real proof. Prob: low, impact:
    MCP catalog mismatch caught before merge.
  - R3 Windows `.exe` vs POSIX bare shim names: resolved at deploy via
    `IsExecutableFile` probe of both forms; unit tests cover both strings.
  - R4 Concurrent workstreams in this repo: T0 worktree/status check + own worktree.
- Unknowns: none blocking (T0/T1 resolve the two pointers above with owner gate on version bumps).
- Breaking changes: none. Disabled defaults untouched (everything stays off until
  `/mcp`); template IDs/shapes untouched; Linux behavior strictly improves
  (bunx-launched MCPs gain a working launcher).

## 7. Definition of Done

- [ ] T0 baseline numbers recorded; T2 RED observed before GREEN.
- [ ] `go build/vet/test ./...` green fresh; golangci-lint no new blocking findings.
- [ ] Shape + parity locks green unmodified; manifest os-lint green.
- [ ] `mise ls` shows both packages pinned installed&&active; doctor package rows OK.
- [ ] `envctl doctor`: ConfigFile rows OK, no new WARN vs baseline.
- [ ] Live `opencode mcp list`: 3/3 connected, catalog counts equal baseline.
- [ ] CHANGELOG + matrix updated; docs-sync clean (no PII/absolute paths in repo).
- [ ] `review` passed, PR merged, this spec deleted; diff limited to §4 files.
