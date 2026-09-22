# SPEC — Distro-strict OS scope + review fixes (no generic `linux`)

Date: 2026-09-22 · Scope: manifests + Go + skills + docs · Source: deep review 2026-09-22
(2 subagents `general` OK; 2 `explore` failed on free-tier provider — OS/gate axes re-verified by direct reads)
· For: a builder LLM implementing end-to-end in THIS repo (`/home/eadiasold/Projetos/envctl`)

## How to read this SPEC (builder instructions)

- Work top-down: Policy 0 first (it constrains every later task), then BLOCKERs (1–3), then MAJORs (4–10), then MINORs (11), then gate (12).
- Each task has: **Why** (evidence with `file:line`), **Files** (exact paths), **Current** (what the code/manifest looks like today), **Change** (exact edit, copy-pasteable), **Tests**, **Acceptance** (literal commands whose output is the proof). No step is a placeholder — if a value must be recounted, the recount command is given.
- Language rules (repo convention): code, comments, commits in English. SPEC itself in English. Chat with the user in PT-BR (not this file).
- Idempotency + atomic backup (`.bak.YYYYMMDD-HHMMSS`) are load-bearing conventions — never overwrite user-owned content without `merge:`/`seed_if_missing`/backup.
- The Go binary embeds `manifests/` + `configs/` (`assets.go`, `//go:embed`). After ANY manifest/config edit you MUST `go build -o envctl .` (or `go build ./...`) from the repo root and run the NEW binary from the repo root — otherwise `fsManager.ReadFile` silently serves the OLD embedded tree (lesson 2026-08-26).
- Verify with: `go build ./... && go vet ./... && go test ./...` and `golangci-lint run --new-from-rev=origin/main`, or `envctl-verify --git-push` (same gate the hooks run). A verifier-script change must keep `internal/usecase/verify_script_test.go` green.
- Do NOT invent new distros, packages, skills, or LSPs. Closed distro set (user statement): `windows`, `arch,cachyos` (= CachyOS desktop), `debian,ubuntu` (= Ubuntu Server VPS). Anything else is out of scope.

## Distro model (closed set — the user will explicitly request any addition)

| Host | `runtime.GOOS` | `DetectedDistro()` (`platform.go:39-54`, via `/etc/os-release` ID+ID_LIKE) | Manifest tokens to use |
|---|---|---|---|
| Windows 11 desktop | `windows` | `windows` | `windows` |
| CachyOS desktop (also covers Arch-family hosts IF they ever appear; today only CachyOS matters) | `linux` | `arch` (maps `arch,archlinux,cachyos,endeavouros,manjaro,garuda,artix`) | `arch,cachyos` (always BOTH tokens — `platform.go:99` maps either to `DistroArch`; writing only `arch` works today but `arch,cachyos` is the repo convention, e.g. `shell.yaml:68`) |
| Ubuntu Server VPS | `linux` | `debian` (maps `debian,ubuntu,linuxmint,pop,raspbian,kali,zorin`) | `debian,ubuntu` (always BOTH tokens) |
| Shared POSIX bytes (identical file on both Linux distros) | `linux` | either | `arch,cachyos,debian,ubuntu` (explicit 4-list — verbose ON PURPOSE, see Policy 0.4) |
| Truly portable (volta/pip/go toolchains, OS-agnostic skills) | any | any | OMIT `os:` entirely (empty = all platforms, `platform.go:86-87`) |

Engine note (do NOT change): `MatchOS("linux","linux",*)==true` stays supported in `platform.go` — the ban is manifest-side (linted), not engine-side. Existing `TestMatchOS` cases (incl. `{"exact platform","linux","linux",DistroUnknown,true}` at `platform_test.go:14`) KEEP passing.

---

## Policy 0 — Distro-strict `os:` (THE rule; implement FIRST, it gates Tasks 4–6)

**Why.** Bare `os: linux` matches every Linux host, so distro-specific content leaks across the desktop/server boundary (gaming + AUR skills deploy to VPS; apt/pacman warn on the wrong distro). The user decision: no generic `linux` anywhere — explicit distro lists only.

**Files.** `manifests/*.yaml`, `configs/skills/*/SKILL.md` (frontmatter — verified below: no change needed), `internal/domain/entity/platform_test.go` (or a new manifest-lint test file), `docs/os-and-agent-matrix.md` §5.

**Current (complete inventory of bare-`os: linux` — verified by grep, 2026-09-22).**

- `manifests/shell.yaml` — 7 occurrences: `15` (NODE_PATH linux row), `27` (ENVCTL_TEMP linux row), `44` (`opencode_config_linux`), `194` (`pw_wrapper_sh`), `291` (`ssh_config_linux`), `352` (`/temp` dir), `431` (`stale_ohmyposh_binary`).
- `manifests/packages.yaml` — apt block `415-599` (every entry `os: linux`: `git,curl,wget,ripgrep,fd-find,fzf,bat,tree,jq,unzip,zip,rsync,build-essential,python3-pip,python3-venv,sshpass,shellcheck,shfmt,software-properties-common,apt-transport-https,ca-certificates,gnupg,docker.io,python3-yaml,python3-requests,python3-openpyxl,python3-bs4`); pacman block `608-803` (every entry `os: linux`: `paru,base-devel,git,fish,go,git-delta,yq,dust,hyperfine,docker,docker-compose,docker-buildx,shellcheck,shfmt,ruff,tree,fzf,bat,fd,ripgrep,taplo-cli,python-yaml,python-requests,python-openpyxl,python-beautifulsoup4,python-pypdf,python-docx,python-lxml`); standalone `opencode` pacman entry `328-333` (`os: linux`).
- `manifests/gaming.yaml` — all 21 entries `os: linux` (`10-125`: `steam,gamescope,mangohud,lib32-mangohud,ntfs-3g,dolphin-emu,retroarch,ppsspp,mgba-qt,snes9x-gtk,mednafen,lact` pacman + `pcsx2-latest-bin,duckstation-qt-bin,azahar-appimage,melonds-bin,flycast-bin,vita3k-bin,cemu-bin` paru).
- `manifests/skills.yaml` — 3 occurrences: `aur-headless-install:52-57`, `cachyos-gaming-setup:73-78`, `headless-gui-probe:104-109` (all `os: linux`). `windows-admin:175-180` (`os: windows`) is correct, keep.
- `configs/skills/*/SKILL.md` frontmatter — VERIFIED: `aur-headless-install/SKILL.md:1-6` and `cachyos-gaming-setup/SKILL.md:1-6` contain only `name/description/license` (no `os:` key) → NO frontmatter edit needed. Do NOT invent an `os:` frontmatter key; the manifest is the single source of truth for skill scoping.
- `manifests/lsp.yaml` — only `powershell:113-121` has `os: windows` (correct, keep); all other 14 LSPs omit `os:` (portable, keep — installers are toolchain-scoped: volta/npm/pip/go, NOT distro-scoped).

**Change.**

1. Add the lint test FIRST (red), then fix manifests (green). New test file `internal/domain/entity/manifest_os_lint_test.go` (or extend `platform_test.go` — prefer a NEW file so `TestMatchOS` stays untouched), which:
   - walks `manifests/*.yaml` (relative to repo root — resolve root by searching upward for `go.mod`, NOT by CWD assumption; test working dir is the package dir),
   - parses each `os:` scalar, splits on `,`/` `/`|` (same as `MatchOS`: `strings.FieldsFunc`), lowercases,
   - FAILS if any token equals bare `linux` (so `linux`, `arch,linux`, `Linux` all fail; `arch,cachyos,debian,ubuntu` passes; empty/missing `os:` passes),
   - FAILS if any `check_command:` value contains a `"` character (Task 7 enforcement — same walk, `packages.yaml`/`lsp.yaml`),
   - FAILS on unknown tokens outside the closed set `{windows,linux(docs-only-legacy-NEVER-new),darwin(legacy-test-only),arch,archlinux,cachyos,debian,ubuntu,""(absent)}` — i.e. warn on anything else (future distro needs a SPEC + user approval before extending this set). Note: `darwin` appears only in `platform_test.go:26` (engine test, not a manifest) — manifests MUST NOT use `darwin`.
2. Then apply the manifest edits in Tasks 4–6 + shell.yaml shared-POSIX list below:
   - `shell.yaml:15,27,44,194,291,352,431` → `os: arch,cachyos,debian,ubuntu` (7 edits; content is byte-identical on both distros — the 4-list is intentional, see rule 4).
   - Rationale to put in a code comment or commit message (NOT in the YAML — keep manifests comment-light): a third distro later gets NOTHING (safe fail-closed) + `doctor` warns via `AGENTS.md (identity coverage)` pattern, instead of silently inheriting a "linux" bucket.
3. Document in `docs/os-and-agent-matrix.md` §5 checklist item 2 (exact replacement):
   - OLD: "`os:` correto — `windows`/`linux`, ou a família de distro (`os: arch`) quando o pacote só existe lá (AUR via `type: paru`)."
   - NEW: "`os:` explícito por distro, NUNCA bare `linux` — `windows` · `arch,cachyos` (desktop CachyOS) · `debian,ubuntu` (VPS Ubuntu Server) · conteúdo POSIX idêntico nas duas distros usa a lista `arch,cachyos,debian,ubuntu` · portável de verdade omite `os:`. Conjunto fechado: nova distro só com pedido explícito do dono (o lint `manifest_os_lint_test.go` falha em `os: linux` e em tokens desconhecidos)."

**Tests.** New lint test (above) + `go test ./internal/domain/entity/ -v` (old + new green).

**Acceptance (literal).**
```bash
grep -rn "os: linux" manifests/ | grep -v "os: arch,cachyos,debian,ubuntu"; echo "exit=$? (want 1 = no bare-linux left)"
grep -rn "os:" configs/skills/*/SKILL.md; echo "(want empty: frontmatter carries no os:)"
go test ./internal/domain/entity/ -v
```

**Pitfalls (do NOT).** Do NOT remove `linux` support from `MatchOS` engine code. Do NOT add `darwin`/new distros. Do NOT add `os:` to SKILL.md frontmatters.

---

## Task 1 — Phase 0 PATH parity (BLOCKER: non-login shells reinstall providers every run)

**Why.** `installedVersion` (`provision_providers.go:274-283`) and `installSource` (`287-297`) probe with process-PATH `exec.LookPath`, but every install/run uses Volta-aware `toolchainEnv()` (`56-62` → `linuxToolchainEnv(home)` in `provision_bootstrap.go:53-72`, prepending `~/.local/bin:~/.volta/bin:/usr/local/go/bin:~/go/bin`). Under ssh/systemd/agent non-login shells the process PATH lacks `~/.volta/bin` → phase 0 reports volta/node/cmdc "not installed" and reinstalls on EVERY `run providers`/`run all` (breaks the repo's strict-idempotency contract; lesson 2026-08-28 re-occurs in a new file — `toolchain_managers.go:21-49` already solves this correctly with `execTool`+`lookPathWithEnv`).

**Files.** `internal/usecase/provision_providers.go` (edit), `internal/usecase/provision_providers_test.go` (extend; check existing tests first — CHANGELOG `v1.2.144` mentions 5 tests there: version token, version compare, install-source classification, cmdc-regression).

**Current.**
```go
func (uc *ProvisionProvidersUseCase) installedVersion(ctx context.Context, binary string) string {
	if _, err := exec.LookPath(binary); err != nil {  // ← process PATH (WRONG env)
		return ""
	}
	out, err := runWithToolchain(ctx, binary, "--version")  // ← toolchain env (RIGHT env)
	...
}
func installSource(binary string) string {
	path, err := exec.LookPath(binary)  // ← same mismatch
	...
}
```

**Change (exact).**

1. Add an unexported helper in the `usecase` package (same package as `linuxToolchainEnv` — do NOT import the `toolchain` infra package: `lookPathWithEnv` there at `toolchain_managers.go:53-69` is unexported and cross-package import would tangle layers; duplicating 15 lines is the approved pattern here):
```go
// resolveOnToolchainPath mirrors toolchain.lookPathWithEnv against the PATH
// built by toolchainEnv(), so probes see what execution sees (non-login
// shells have a minimal process PATH; the toolchain PATH prepends
// ~/.local/bin, ~/.volta/bin, /usr/local/go/bin, ~/go/bin).
func resolveOnToolchainPath(name string) (string, error) {
	env := toolchainEnv()
	path := ""
	for _, kv := range env {
		if v, ok := strings.CutPrefix(kv, "PATH="); ok {
			path = v
			break
		}
	}
	if path == "" {
		return exec.LookPath(name)
	}
	if filepath.IsAbs(name) {
		return name, nil
	}
	for _, dir := range filepath.SplitList(path) {
		if dir == "" {
			dir = "."
		}
		candidate := filepath.Join(dir, name)
		if fi, err := os.Stat(candidate); err == nil && !fi.IsDir() && fi.Mode()&0111 != 0 {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("executable %q not found on toolchain PATH", name)
}
```
(`os`, `os/exec`, `path/filepath`, `strings`, `fmt` are already imported in `provision_providers.go:3-10` — verify before adding imports.)
2. In `installedVersion`: replace `exec.LookPath(binary)` with `resolveOnToolchainPath(binary)` for the existence probe. Keep `runWithToolchain(ctx, binary, "--version")` as-is (it already sets `cmd.Env`; note `exec.Cmd.Env` does NOT affect binary RESOLUTION — `exec.LookPath` uses the process PATH — which is exactly why the probe must be resolved manually and ideally the resolved absolute path passed to `runWithToolchain`; minimal fix: probe with the helper, execute by name as today. If you pass the absolute path, keep `--version` behavior identical.)
3. In `installSource`: replace `exec.LookPath(binary)` with `resolveOnToolchainPath(binary)`. Keep `classifyInstallSource(path, home)` unchanged.
4. Windows behavior unchanged: `toolchainEnv()` returns `os.Environ()` on Windows (`provision_providers.go:58-60`), so the helper degrades to process-PATH resolution there — correct.

**Tests.** Add `TestInstalledVersionResolvesVoltaShim`-style regression in `provision_providers_test.go`: fake `HOME` to a temp dir containing `.volta/bin/fakecli` (executable bit `0755`, prints a version on `--version`), strip the temp dir from process `PATH`, assert `installedVersion` still finds it AND `installSource` classifies it as `volta` (via `classifyInstallSource`, already unit-testable). Keep the 5 existing tests green.

**Acceptance (literal).**
```bash
go build ./... && go vet ./... && go test ./internal/usecase/ -run TestProvisionProviders -v
# live (both distros + non-login shell):
env -i HOME="$HOME" bash -lc '"$PWD/envctl" run providers'   # run TWICE; 2nd run must be all-DiagOK, zero installs
```

**Pitfalls.** Do NOT "fix" by mutating the process `os.Setenv("PATH",...)` globally (lesson: `exec.Cmd.Env` doesn't affect resolution; global mutation leaks into unrelated probes). Do NOT touch `runWithToolchain` semantics.

---

## Task 2 — Phase 0 pacman branch for opencode (BLOCKER: curl shadows the `extra` package)

**Why.** `installStandaloneProvider` (`provision_providers.go:243-270`) handles only winget (Windows) vs curl-installer (everything else). But the manifest declares `opencode` as Arch `extra` pacman package (`packages.yaml:328-333`, matrix §1 "Arch: pacote `extra`") and the repo rule (matrix §1 + checklist §5 item 2 + asymmetry #9) is: an OS-owned binary is REPORTED, never shadowed — a curl copy in `~/.local/bin` wins on PATH and freezes the version. Today a CachyOS box without opencode gets the curl copy instead of `pacman -S opencode`.

**Files.** `internal/usecase/provision_providers.go` (edit).

**Current** (`243-270`): `if windows → winget; if installer=="" → warn; else bash -lc installer`.

**Change (exact).** Insert AFTER the Windows winget block, BEFORE the `tool.installer==""` check:
```go
// Arch/CachyOS owns opencode via the `extra` repo (manifests/packages.yaml):
// prefer the distro channel over the curl installer so phase 0 never
// shadows a system package with a ~/.local/bin copy.
if mgr, ok := uc.managers[entity.PackageTypePacman]; ok && mgr.IsAvailable(ctx) {
	if err := mgr.Install(ctx, entity.Package{ID: "opencode", Type: entity.PackageTypePacman}); err == nil {
		add(entity.DiagOK, tool.name, "Installed via pacman (extra)", "")
		return
	}
	add(entity.DiagWarning, tool.name, "pacman could not install opencode",
		"Run 'envctl run pacman' to install opencode")
	return
}
```
Verify the constant name first: `grep -rn "PackageTypePacman" internal/domain/entity/` (used in `run.go:62` as `entity.PackageTypePacman` — confirmed). Keep curl-installer as the Debian/Ubuntu (+unknown-distro) path. Windows path untouched.

**Tests.** Existing provider tests green; if the file has a fake-managers harness, add: pacman-available + opencode-missing → asserts pacman `Install` called with `ID=="opencode"`, curl NOT invoked. (If no harness exists, do NOT build one from scratch — cover via the acceptance runs below and note it in the commit message.)

**Acceptance (literal).**
```bash
go test ./internal/usecase/ -v 2>&1 | tail -5
# CachyOS: with pacman present and opencode absent → installs via pacman, NO ~/.local/bin/opencode created:
ls ~/.local/bin/opencode; echo "(want: no such file after providers on Arch)"
# Ubuntu: still curl-installs (unchanged path).
```

**Pitfalls.** Do NOT add a paru branch (opencode is `extra`/pacman, not AUR). Do NOT change the `command-code` (Volta) path.

---

## Task 3 — VPS dispatch skill: drop the npm fallback (BLOCKER: teaches the forbidden downgrade)

**Why.** `configs/skills/vps-agent-dispatch/SKILL.md:75` lists `npm install -g opencode-ai` as an install fallback. The npm channel lags upstream (1.18.x vs 2.x tags that Arch `extra` 2.0.5 packages) — installing via npm DOWNGRADES the box (asymmetry #9, `packages.yaml:324-327` comment, `provision_bootstrap.go` history). A skill teaching the downgrade will eventually be followed.

**Files.** `configs/skills/vps-agent-dispatch/SKILL.md` (edit lines ~73-75 only).

**Current** (line 75): ``- Se ausente, sugira provisionar via `envctl run all` (padrão) ou `npm install -g opencode-ai` / script oficial de instalação.``

**Change (exact replacement).**
```md
- Se ausente, provisione pelo canal da distro (NUNCA por npm — o pacote `opencode-ai` rebaixa a linha 2.x para 1.18.x, assimetria #9): preferido `envctl run all` (ou `envctl run providers` só para os CLIs); fallback por OS — Ubuntu/Debian: `curl -fsSL https://opencode.ai/install | bash` · Arch/CachyOS: `sudo pacman -S opencode` (repo `extra`) · Windows: `winget install SST.opencode`.
```

**Tests.** Frontmatter parse check (description uses `>-` block scalar — keep it; a `:` in plain scalar would silently drop the skill, lesson 2026-08-29).

**Acceptance (literal).**
```bash
grep -rn "npm install.*opencode" configs/ manifests/ internal/; echo "exit=$? (want 1 = gone)"
python3 -c "import yaml,glob; [yaml.safe_load(open(f).read().split('---')[1]) for f in glob.glob('configs/skills/*/SKILL.md')]; print('frontmatter OK')"
```

---

## Task 4 — Gaming is Arch-only (`gaming.yaml` + runtime guard)

**Why.** All 21 `gaming.yaml` entries are `os: linux` (`10-125`), and neither `ExecuteGaming` (`provision_packages.go:49-63`) nor `runGamingProvisioning` (`run.go:315-333`) checks the distro — `MatchesOS("linux")` is true on Ubuntu, so `run gaming` on a VPS attempts Steam/gamescope/emulators/LACT (only saved today by absent pacman/paru). Docs (§1) and CLI help ("on Arch/CachyOS") promise Arch-only.

**Files.** `manifests/gaming.yaml` (21 edits), `internal/usecase/provision_packages.go` (guard), optionally `internal/ui/cli/run.go` (message only — prefer the usecase guard so both CLI and future callers are covered).

**Change.**

1. `gaming.yaml`: replace EVERY `os: linux` → `os: arch,cachyos` (mechanical; verify count stays 21):
```bash
sed -i 's/^\(\s*\)os: linux$/\1os: arch,cachyos/' manifests/gaming.yaml
grep -c "os: arch,cachyos" manifests/gaming.yaml   # want 21
grep -rn "os: linux" manifests/gaming.yaml; echo "exit=$? (want 1)"
```
2. Guard in `ExecuteGaming` (after `LoadGamingPackages`, before `provisionList`):
```go
if entity.DetectedDistro() != entity.DistroArch && runtime.GOOS == "linux" {
	return nil, fmt.Errorf("gaming stack is Arch/CachyOS-only (this host: %q); refusing to install Steam/GUI packages on a server", entity.DetectedDistro())
}
```
Add `"runtime"` + `entity` imports if missing (check header: `provision_packages.go:3-9` currently imports context/fmt/entity/repository — `runtime` needs adding). Return type is `([]entity.Package, error)` and the CLI prints the error via `spinner.Fail` (`run.go:327-330`) — correct UX. Windows already can't reach here meaningfully (pacman/paru managers unavailable → but explicit refusal is clearer; keep the guard Linux-focused and let manager-availability handle Windows).

**Tests.** If a gaming test exists, extend with: distro=debian + `os: arch,cachyos` fixture → 0 packages processed (or error per guard); arch → all processed. Else cover via acceptance.

**Acceptance (literal).**
```bash
grep -c "os: arch,cachyos" manifests/gaming.yaml   # want 21
go build -o envctl . && ./envctl run gaming        # CachyOS: "Processed 21 gaming packages" (or install flow); Ubuntu: clean refusal, zero pacman/paru warnings
```

---

## Task 5 — apt/pacman distro scoping (kills the `run all` warning spam)

**Why.** apt block (`packages.yaml:415-599`) and pacman block (`608-803`) both say `os: linux`, so each distro matches the OTHER's 27–29 packages; `provisionList` (`provision_packages.go:92-102`) then warns "manager not available" per package. `doctor` hides it (silent skip, `doctor_audit.go:281-286`), `run all` doesn't.

**Files.** `manifests/packages.yaml` (only `os:` lines change — do NOT touch ids/types/checks here; quotes are Task 7).

**Change (mechanical, verify each with grep after).**
```bash
# 1) standalone opencode pacman entry (extra channel):
#   lines ~328-333:  os: linux  →  os: arch,cachyos
# 2) apt block 415-599 (git…python3-bs4, ~27 entries): os: linux → os: debian,ubuntu
# 3) pacman block 608-803 (paru…python-lxml, ~29 entries): os: linux → os: arch,cachyos
# 4) KEEP cursor-bin 384-388 (os: arch, paru) — already correct, Arch-desktop-only, no Cursor on Ubuntu server.
python3 - <<'EOF'
import re
p = 'manifests/packages.yaml'
s = open(p).read().splitlines(keepends=True)
# Apply by line ranges is brittle — instead: inside the apt section (between
# '# Debian / Ubuntu APT' and '# Arch Linux / CachyOS (pacman)') map os: linux->debian,ubuntu;
# inside the pacman section (to EOF, excluding the already-correct cursor-bin os: arch) map to arch,cachyos.
out, mode = [], None
for line in s:
    if 'Debian / Ubuntu APT' in line: mode = 'apt'
    elif 'Arch Linux / CachyOS (pacman)' in line: mode = 'pacman'
    if re.match(r'\s*os: linux\s*$', line) and mode == 'apt':
        line = line.replace('os: linux', 'os: debian,ubuntu')
    elif re.match(r'\s*os: linux\s*$', line) and mode == 'pacman':
        line = line.replace('os: linux', 'os: arch,cachyos')
    out.append(line)
open(p, 'w').writelines(out)
EOF
grep -n "os: linux" manifests/packages.yaml  # expect ONLY the opencode standalone entry if you didn't hand-edit it — then fix 328-333 manually to arch,cachyos; final want: zero bare-linux lines
```
Then hand-fix `328-333` (`opencode` pacman) to `os: arch,cachyos`, and confirm `SST.opencode` winget `335-340` stays `os: windows`, volta/pip entries stay portable (no `os:`).

**Dual-channel note (taplo — DOCUMENT, don't re-plumb).** `lsp.yaml:132-139` (`toml`, `install_type: npm @taplo/cli`) vs `packages.yaml:749-754` (`taplo-cli` pacman): same dual-channel shape as the old opencode problem, but on Arch `run lsp` npm-installs what pacman already provides. Out of scope to rewire installers in THIS SPEC — instead document it as an intentional exception in the matrix (same wording pattern as the `pylsp` exception): matrix §1 or §3 new row: "`taplo` ships both channels (pacman `taplo-cli` + npm `@taplo/cli`); `run lsp` prefers npm today — same owner as the file, no shadowing across managers. Revisit only if version skew is observed." If the implementer sees actual skew, STOP and ask the user (do not redesign `run lsp` speculatively).

**Tests.** Manifest load tests green (`TestLoadManifestsFromDiskOrEmbed`-family — find exact name via `grep -rn "func Test" internal/ | grep -i manifest`); Policy-0 lint test green.

**Acceptance (literal).**
```bash
grep -rn "os: linux" manifests/packages.yaml; echo "exit=$? (want 1)"
go build -o envctl . && ./envctl run all 2>&1 | grep -c "manager not available"; echo "(want 0 on both CachyOS and Ubuntu)"
```

---

## Task 6 — Skills distro scoping (VPS stops receiving desktop-only skills)

**Why.** `skills.yaml:52-57` (`aur-headless-install`) + `73-78` (`cachyos-gaming-setup`) are `os: linux` → `AppliesToOS` (`models.go:105-109`, distro-aware via `DetectedDistro()`) deploys them to Ubuntu VPS. AUR helper skills and GPU/gaming tuning are meaningless on a server (matrix §0: "Ubuntu Server, sem GUI").

**Files.** `manifests/skills.yaml` (3 edits).

**Change (exact).**
- `aur-headless-install` (`52-57`): `os: linux` → `os: arch,cachyos`.
- `cachyos-gaming-setup` (`73-78`): `os: linux` → `os: arch,cachyos`.
- `headless-gui-probe` (`104-109`, `os: linux`): → `os: arch,cachyos,debian,ubuntu` (Policy 0 shared-POSIX 4-list — offscreen GUI probing is legitimately useful on headless servers).
- Keep `windows-admin` (`os: windows`) and all 37 portable entries (no `os:`) untouched.
- SKILL.md frontmatters: NO change (verified — they carry no `os:` key; manifest is authoritative).

**Resulting applicable counts (for Task 8 docs):** total 41 = 37 portable + 4 scoped → Windows 38 (37+windows-admin) · Ubuntu 38 (37+headless-gui-probe) · CachyOS 40 (37+aur+gaming+headless). VERIFY with the counter in Task 8 — do not trust this comment blindly.

**Tests.** Skills manifest↔disk consistency tests green; Policy-0 lint green.

**Acceptance (literal).**
```bash
go build -o envctl . && ./envctl run skills   # Ubuntu: deploys 38 to OpenCode (no aur-headless-install, no cachyos-gaming-setup); CachyOS: 40
ls ~/.config/opencode/skills | sort
```

---

## Task 7 — `check_command` without quotes (dead probes)

**Why.** Every manager splits the custom check with `strings.Fields` (`apt_manager.go:47`, `toolchain_managers.go:89`, `volta_manager.go:31`, `paru_manager.go:48`, `winget_manager.go:35`). A payload like `python3 -c "import yaml"` tokenizes to `["python3","-c","\"import","yaml\""]` and ALWAYS fails; only the `dpkg -Q`/`pacman -Q` ID fallback (`apt_manager.go:57-71`) saves the audit. Violates matrix §5 item 3 ("`check_command` sem aspas").

**Files.** `manifests/packages.yaml` (deletions only).

**Change.** DELETE the `check_command:` line on every entry whose value contains a `"`: the `python3 -c "import …"` apt probes (`python3-yaml,python3-requests,python3-openpyxl,python3-bs4`) and the `python3 -c "import …"` pacman probes (`python-yaml,python-requests,python-openpyxl,python-beautifulsoup4,python-pypdf,python-docx,python-lxml`). Do NOT replace with another probe (a `Fields`-compatible `-c` payload is impossible — quoting is mandatory for `-c`, hence unrepresentable here). Detection (robust to miscounts — do NOT hardcode "8"):
```bash
grep -n 'check_command.*"' manifests/packages.yaml   # every hit loses its check_command line
```

**Tests.** Policy-0 lint extension (Task Policy-0.1) fails on `"` in `check_command` going forward. `doctor` package checks for those IDs must stay green via the ID fallback.

**Acceptance (literal).**
```bash
grep -rn 'check_command.*"' manifests/; echo "exit=$? (want 1)"
go build -o envctl . && ./envctl doctor 2>&1 | grep -Ei "python3?-?(yaml|requests|openpyxl|bs4|pypdf|docx|lxml)" | head -20  # want OK-via-fallback, no false-missing
```

---

## Task 8 — Docs/matrix/README recount (numbers are a contract)

**Why.** The matrix header says it: "Todos os números vêm dos manifestos — se divergirem, um dos dois mudou." All six below diverge (verified against disk 2026-09-22).

**Files.** `README.md`, `docs/os-and-agent-matrix.md`, `docs/doctor-and-idempotency.md`.

**Change — exact replacements (recompute AFTER Tasks 4–6; commands to recount are below — never copy numbers blindly).**

1. `README.md:78`: `envctl run lsp          # 18 Servidores de Linguagem (LSP)` → `# 15 Servidores de Linguagem (LSP)` (15 ids in `lsp.yaml`; removals 18→16→15 per CHANGELOG v1.2.126/v1.2.129).
2. Matrix §2 `Plugins`: `3 + deps npm (package.json)` → `1 + deps npm (goal-plugin only; dcp.jsonc kept provisioned for return)` (both `configs/opencode.json:174-176` + `opencode.linux.json:174-176` list only `@prevalentware/opencode-goal-plugin`; dcp+ponytail removed 2026-09-19, opencode-v2 breakage).
3. Matrix §2 `Arquivos declarados`: `10 / 6` → `11 / 7` (split `AGENTS.arch.md` added 1 per agent — asymmetry #13).
4. Matrix §1 `Variáveis de ambiente`: `3 | 2 | 2` → `2 | 2 | 2` (`shell.yaml:4-27`: only `NODE_PATH` + `ENVCTL_TEMP` per OS; `FZF_DEFAULT_COMMAND` left with asymmetry #6).
5. Matrix §1 `Git global`: `5 (3+2) | 3 | 3` → `6 (4+2 win-only) | 4 | 4` (`git.yaml` has 6 entries incl. `core.hooksPath` from gate v1.2.126).
6. Matrix §1 `Configs aplicáveis` (`21 / 19 / 19`): RECOUNT with this method (document the method in-matrix in one parenthetical): count `config_files` in `shell.yaml` with `MatchesOS` per distro AFTER the Policy-0 4-list edits. Expected direction: ~28 Win / 26 Debian / 26 Arch (prior audit) — but RECOMPUTE, the Task 4–6 edits move entries.
7. Matrix §0 Ubuntu row: `Sem IDE — o LSP/editor não se aplica` → `Sem Cursor/IDE (servidor, sem GUI) — LSPs do agente aplicam-se normalmente (headless)` (LSPs are agent runtime, not IDE chrome; `lsp.yaml` intentionally has no distro filter except `pwsh`).
8. `docs/doctor-and-idempotency.md:22`: `todos os 50+ binários` → `45–55 binários conforme o OS (45 Ubuntu / 48 Arch / 55 Win — matrix §1)` (recompute if Task 5 changes totals — it shouldn't, scoping doesn't add/remove entries).
9. Same doc `27-31`: `15 LSPs` → `15 no manifesto (14 aplicáveis no Linux — `pwsh` é windows-only)`; `41 Skills em ~/.config/opencode/skills` → `38 Win / 38 Ubuntu / 40 CachyOS` (+ CommandCode mirror note) — recompute per Task 6 result.

**Recount commands (paste outputs into the commit message).**
```bash
python3 - <<'EOF'
import yaml
pkgs = yaml.safe_load(open('manifests/packages.yaml'))['packages']
from collections import Counter
print('packages total:', len(pkgs), Counter((p.get('type'), p.get('os','<portable>')) for p in pkgs))
skills = yaml.safe_load(open('manifests/skills.yaml'))['skills']
print('skills total:', len(skills), [(s['name'], s.get('os','<portable>')) for s in skills if s.get('os')])
lsps = yaml.safe_load(open('manifests/lsp.yaml'))['lsps']
print('lsps total:', len(lsps), [l['id'] for l in lsps])
shell = yaml.safe_load(open('manifests/shell.yaml'))
print('config_files:', len(shell['config_files']), 'dirs:', len(shell['directories']))
EOF
```

**Acceptance.** `grep -rn "18 Servidores\|3 + deps\|10.*6.*declarados\|FZF_DEFAULT" README.md docs/os-and-agent-matrix.md` shows only historical/asymmetry references, no live claims; every §1/§2 number has its recount command in the commit message.

---

## Task 9 — Tweaks PS quoting (security: manifest-driven script injection surface)

**Why.** All six PS builders interpolate raw (`tweaks_manager.go:33-36` feature-check, `48-50` psmodule-check, `62-75` registry-check, `109` feature-apply, `127-132` psmodule-apply, `152-161` registry-apply). `env_manager.go:24` already owns the correct pattern (`psQuote`: `'` → `''`). Today's manifests are static ints/known names (low live risk) — but the lens is mandatory: any future string tweak value with `"`/`$` breaks the script or escapes it.

**Files.** `internal/infra/windows/tweaks_manager.go` (edit), new/extend `*_test.go` in same package (check what exists: `ls internal/infra/windows/`).

**Change (exact).**

1. Add (duplicate — `psQuote` in `environment` is unexported; cross-package import for 3 lines is overkill; keep a local copy with a comment pointing at the canonical one):
```go
// psQuote mirrors environment.psQuote (single-quote escape for PowerShell
// string literals); duplicated to avoid an infra→infra import for 3 lines.
func psQuote(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}
```
(`strings` is already imported — verify.)
2. Convert every `"%s"` interpolation of `tweak.Name`/`tweak.Path` to `'%s'` + `psQuote(...)`:
   - check-feature: `... -FeatureName "%s" ...` → `... -FeatureName '%s' ...", psQuote(tweak.Name)`
   - check-psmodule: `-Name "%s"` → `-Name '%s'", psQuote(tweak.Name)`
   - check-registry: `$path = "%s" / $name = "%s"` → `'%s'` + psQuote both.
   - apply-feature/psmodule/registry: same conversion.
3. Registry value formatting — replace `$val = %v` with a typed formatter:
```go
func psValue(v any) string {
	switch t := v.(type) {
	case string:
		return "'" + psQuote(t) + "'"
	case bool:
		if t {
			return "$true"
		}
		return "$false"
	default:
		return fmt.Sprintf("%v", v) // ints/floats: bare (DWord-compatible)
	}
}
```
and `$val = %v", tweak.Value` → `$val = %s", psValue(tweak.Value)`. Check `entity.WindowsTweak.Value` type first (`grep -n "Value" internal/domain/entity/*.go`) — if it is `any`/`interface{}`, the switch above is correct; if typed `int`, keep ints bare and still route through `psValue` for future-proofing.
4. `valType` (`tweak.Type`, `152-161` `$type = "%s"`) → also single-quote + psQuote (defense in depth; type comes from the same manifest).

**Tests.** Adversarial unit test: `WindowsTweak{Name: "a'b\"c$d", Path: "HKCU:\\a'b", Value: "x'y"}` → assert generated script contains `'a''b"c$d'` (single-quotes doubled, double-quote harmless inside single-quoted PS string, `$` inert inside single quotes) and NO bare `"...` interpolation of the payload.

**Acceptance.** `go test ./internal/infra/windows/ -v` green; `golangci-lint run --new-from-rev=origin/main` clean (watch `gosec G204` — the file may need the existing `//nolint:gosec` pattern used in `doctor_audit.go:1197` if lint flags the `exec.CommandContext(ctx, "powershell.exe", ..., psScript)` call; copy that exact nolint comment + justification rather than weakening lint globally).

---

## Task 10 — Atomic backup + skills overwrite guard (data-loss)

**Why.** (a) `WriteWithBackup` (`fs_manager.go:120-155`) writes the destination DIRECTLY (`os.WriteFile`, line 150) — a mid-write kill leaves a truncated live file — and the backup suffix has 1-second resolution (`20060102-150405`, line 141): two writes in the same second destroy the first backup. The docblock promises "atomic timestamped backup". (b) `CopyEmbeddedTree` (`fs_manager.go:196-241`, sole skill caller `provision_skills.go:92`) writes `0644` unconditionally (line 232): no content-diff, no `.bak`, drops the exec bit — a user-customized `SKILL.md` is silently destroyed, violating the repo's never-overwrite-user-content rule.

**Files.** `internal/infra/filesystem/fs_manager.go` (edit), `internal/usecase/provision_skills.go` (log wording only if counting semantics change), tests in `internal/infra/filesystem/`.

**Change (exact).**

1. `WriteWithBackup` rewrite:
   - keep: expand path, `MkdirAll` dir, read-existing → identical bytes ⇒ return `"", nil` (no-op, unchanged).
   - backup: `stamp := time.Now().Format("20060102-150405")`, `backupPath := expanded + ".bak." + stamp`; if that path EXISTS (same-second second write), append `-1`, `-2`, … (loop `os.Stat`, first free wins). Write backup with the SAME `perm` (keep current behavior) — do NOT `Chmod` the live file (preserve-mode hardening is out of scope; note it in the commit).
   - atomic write: `tmp, err := os.CreateTemp(dir, ".envctl-tmp-*")`; `tmp.Write(content)`; `tmp.Chmod(perm)`; `tmp.Close()`; `os.Rename(tmp.Name(), expanded)`. On rename error: remove tmp, return error. (`os.Rename` is atomic on POSIX same-dir and on Windows for this size class — document that in a comment.)
   - return `backupPath, nil` as today (callers unchanged).
2. `CopyEmbeddedTree` rewrite (lines 220-238):
   - after reading embedded `data`: `os.ReadFile(targetPath)` — if exists AND `bytes.Equal(existing, data)` → skip WITHOUT counting (return nil early, do NOT increment `count`); if exists AND differs → create `.bak.YYYYMMDD-HHMMSS(-N)` exactly like (1) (same stamp/counter logic — extract a shared `backupPathFor(path string) string` helper in the same file and use it in both functions), then tmp+rename write.
   - preserve exec bit: `mode := os.FileMode(0644)`; if existing file has any exec bit (`fi.Mode()&0111 != 0`) OR embedded `d.Info()` reports exec → `mode = 0755`. (Embedded `fs.FS` from `embed` reports `0444`-ish modes — the EXISTING target's bit is the source of truth when present; for new files keep `0644`, or `0755` when the SOURCE path ends in `.sh`/has no extension under `configs/bin/`? Simplest deterministic rule, document it: new file under `configs/bin/` or with `.sh`/no-extension + shebang ⇒ `0755`, else `0644`. Pick ONE rule, implement it, test it.)
   - `count` now means "files written/updated" (was "files visited"). Update the `provision_skills.go:105` log line ONLY if it claims otherwise — it says `Deployed skill '%s' (%d files)` which stays truthful under the new meaning. Add a debug-level `skipped identical` counter in the walker for observability (log via returned count? keep signature `(int, error)` — do NOT change the `FileSystemManager` interface; extra observability goes to the logger if available, else dropped).
3. Interface check: `CopyEmbeddedTree` is part of `repository.FileSystemManager` — do NOT change its signature (callers + mocks). Verify with `grep -rn "CopyEmbeddedTree" internal/`.

**Tests.** `fs_manager` tests: (i) same-second double-write keeps BOTH backups (`.bak.<stamp>` + `.bak.<stamp>-1`); (ii) identical rewrite is a no-op (mtime unchanged, no new `.bak`); (iii) user-edited skill file survives `CopyEmbeddedTree` as `.bak` + updated content; (iv) exec bit preserved (seed target `0755`, embedded `0644` → still `0755` after copy).

**Acceptance (literal).**
```bash
go test ./internal/infra/filesystem/ -v
go test ./internal/usecase/ -run TestProvisionSkills -v 2>&1 | tail -3
```

---

## Task 11 — MINORs (same PR, no separate rollout)

1. `internal/ui/cli/run.go:27` — error string lists `all, winget, apt, pacman, paru, gaming, bootstrap, volta, pip, shell, skills, lsp, windows, cleanup` but omits `providers` (a real subcommand, `run.go:84-91`, and the documented phase 0). Change to `valid: all, providers, winget, apt, pacman, paru, gaming, bootstrap, volta, pip, shell, skills, lsp, windows, cleanup`.
2. `internal/infra/logger/file_logger.go:42` — `os.OpenFile(..., 0644)` → `0600` (header at `53-62` logs host/user/OS + full PS command lines via `LogCommand`; multi-user boxes shouldn't world-read it). Leave dir `0755` (line 35) untouched.
3. `internal/usecase/doctor_audit.go:1208-1211,1222` — (a) ~~crash-with-no-output (`cmdErr != nil && len(out)==0`) currently `continue`s SILENTLY (the most-broken server is the quietest): emit `DiagWarning` naming `lsp.ServerName` + `CheckBinary` ("crashed on spawn with no output") with FixHint `envctl run lsp`~~ **REJECTED on evidence during implementation (2026-09-22)**: quiet exit IS the healthy stdio shape (node servers exit 1 on healthy EOF — calibrated live across the 14 Linux LSPs; `TestDoctorAudit_LSPHandshakeQuietExitPasses` pins it with `sh -c "exit 1"` → zero diags). Crash and healthy-quiet are indistinguishable at this layer (`cmdErr`+empty `out` both ways), and a warning here would flag every healthy quiet server, breaking the 0 WARN/0 ERRO contract. Kept the silent `continue` + documented why in a code comment. (b) `repro` string (line 1222) hardcodes `< /dev/null`: build it per-platform — `" < NUL"` when `runtime.GOOS=="windows"` else `" < /dev/null"` (verify `runtime` is imported in `doctor_audit.go`; `os` already is — line 1199 uses `os.DevNull`, which is ALREADY correct at runtime, only the displayed string is wrong). IMPLEMENTED. (c) Keep the sequential 10s×N loop (worst case ~140s for 14 LSPs) — parallelization is explicitly OUT of scope; note it in the commit message as future work.

**Acceptance.** `go build ./... && ./envctl run badname 2>&1 | grep providers` (lists it); `ls -l ~/.envctl/logs/envctl-*.log | awk '{print $1}'` (want `-rw-------` on new logs); `TestDoctorAudit_LSPHandshakeQuietExitPasses` still green (silent-quiet kept by design — see 3a rejection).

---

## Task 12 — Gate + changelog + rollout (close-out)

1. `CHANGELOG.md` — append a NEW `###` subsection under the existing `## [Unreleased]` (do NOT create a second `[Unreleased]` header; file head verified: `CHANGELOG.md:10` is `## [Unreleased]` with two subsections already):
```md
### 🔒 Distro-strict OS scope + phase-0 and safety fixes (review 2026-09-22)

- **Changed**: bare `os: linux` banned from manifests — explicit `arch,cachyos` (CachyOS desktop) · `debian,ubuntu` (Ubuntu Server VPS) · shared-POSIX 4-list `arch,cachyos,debian,ubuntu` · portable omits `os:`; lint test fails on bare-`linux`/quoted-`check_command`/unknown tokens (Policy 0).
- **Fixed**: phase 0 probes resolve on the toolchain PATH (non-login shells no longer reinstall Volta/Node/CLIs every run); `installStandaloneProvider` prefers pacman `extra` on Arch (no more curl shadow); `vps-agent-dispatch` drops the `npm install -g opencode-ai` fallback (per-distro channels).
- **Fixed**: `gaming.yaml` is Arch-only + runtime refusal off-Arch; apt↔pacman cross-distro warning spam gone; AUR/gaming skills no longer deploy to Ubuntu (38 Win / 38 Ubuntu / 40 CachyOS); 8 quoted `check_command`s removed (ID fallback owns those checks).
- **Fixed**: Windows tweaks PS-quote `Name/Path` + typed `Value`; `WriteWithBackup` tmp+rename with unique same-second suffixes; `CopyEmbeddedTree` diff-gated with `.bak` + exec-bit preservation.
- **Docs**: matrix/README/doctor-doc numbers recounted from manifests (method in commit message); Ubuntu row clarified (no Cursor/IDE, agent LSPs apply headless); `taplo` dual-channel documented as intentional exception.
```
2. Gate (in this order; paste outputs into the PR/commit message):
```bash
go build ./... && go vet ./... && go test ./...
golangci-lint run --new-from-rev=origin/main
~/.local/bin/envctl-verify --git-push   # or: envctl-verify --git-push (repo-root build only)
```
`internal/usecase/verify_script_test.go` must stay green if the verifier was touched (it wasn't in this SPEC — if you touch `configs/bin/envctl-verify`, re-read `docs/verification.md` first).
3. Rollout (rebuild + run from repo root EVERY time — embedded manifests!):
   - CachyOS desktop: `go build -o envctl . && ./envctl run all && ./envctl doctor` → expect ZERO "manager not available", arch AGENTS identity (no "Ubuntu Server"/`` `ubuntu` `` strings in `~/.config/opencode/AGENTS.md`), 40 skills in `~/.config/opencode/skills`, 15 `~/.config/opencode/opencode.json` LSP keys (13 active + `powershell` absent-by-OS + `pylsp`-absent-by-design — count keys, not activations).
   - Ubuntu VPS (fresh or existing): same two commands → server AGENTS identity, 38 skills, `./envctl run gaming` refuses cleanly, `doctor` green (Chrome-absent is Info on Linux by design).
   - Windows 11: `go build -o envctl.exe` + `.\envctl.exe run all` → `doctor` → 8 tweaks OK, Cursor present, 38 skills.
4. Commit in ONE conventional commit (repo rule): e.g. `fix: distro-strict os scope, phase-0 PATH/pacman, backup atomicity + docs recount`. NEVER push secrets; never commit `~/.config/opencode/extras/` or `secrets/` content.

## Explicit non-goals (do NOT do these in this SPEC)

- No new distros, packages, skills, LSPs, or `darwin` support. No `run lsp` installer redesign for taplo (documented exception only). No doctor-handshake parallelization. No `GOPATH`-preserve change (`provision_bootstrap.go:62` overwrites `GOPATH` — known MINOR from review, deferred: changing Go env semantics needs its own SPEC). No `EnsureTweaks`/dead-code (`ListInstalled`, `PrintSuccess/...`, `StatusMissing/...`) removal — deferred YAGNI cleanup, separate commit.
