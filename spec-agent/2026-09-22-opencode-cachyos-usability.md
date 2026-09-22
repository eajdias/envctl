# SPEC — OpenCode usability on CachyOS (ssh MCP, zscan removal, LSP handshake, memory enforcement, arch identity)

Date: 2026-09-22 · Scope: end-to-end (OpenCode + CommandCode, CachyOS + Windows, docs + doctor) · Answers: ssh-mirror-disabled / remove-zscan+doctor / lsp-handshake / memory-via-AGENTS / split-arch-vs-server

## Context (verified evidence, not assumptions)

- `envctl doctor` on this CachyOS host: 128/128 green. All 14 LSP binaries resolve via `which` (volta/pacman). Provisioning works; identity and runtime wiring do not.
- `configs/opencode.linux.json` NEVER had `ssh-manager` (since initial squash `aa160e9`); only Windows `configs/opencode.json:173-177` has it (`enabled: false`). No regression — day-one asymmetry. Matrix `docs/os-and-agent-matrix.md:80` documents OpenCode MCP as context7 + chrome-devtools only.
- `zscan` (`@eajdias/zscan-run`) is present since `aa160e9` in `configs/opencode.json:178-186` and `configs/commandcode/mcp.json:19-28` (both `enabled: false`). No later commit added it; nothing removed it. Removal is a new decision, not a revert.
- LSP removals (rust `854c511`, csharp `9a0f9ae`) were deliberate stack decisions co-authored by CommandCodeBot and recorded in CHANGELOG — not accidental deletions. `dcp`/`ponytail` removal (`7667593`) was an opencode-v2 breakage fix, also documented.
- Memory enforcement never regressed: `git log -S agent-memory -- configs/ manifests/` shows only `aa160e9`. Build-agent wording ("quando parecer repetir") was always weak; only `review`/`plan` prompts mandate one `agent-memory` load.
- `~/.config/opencode/AGENTS.md` on this host says "Ubuntu Server, usuário `ubuntu`" (`configs/AGENTS.linux.md:5`, described as "for Linux VPS servers" in `manifests/shell.yaml:54-60`) — wrong identity for a CachyOS desktop (`eadiasold`, fish, paru, gaming, Cursor).
- `ssh-manager server list` on this host: zero servers; `~/.config/opencode/extras/` empty. Inventory is per-machine and never versioned (correct), but this machine was never onboarded.

## Task 1 — ssh-manager MCP in `opencode.linux.json` (mirror Windows, disabled)

1. Add to `configs/opencode.linux.json` `mcp` section, mirroring `configs/opencode.json:173-177`,
   plus `"timeout": 30000` (MCP default 5s bursts on handshake — recorded lesson 2026-08-24; the neighboring
   `chrome-devtools` entry already pins 30000):
   `ssh-manager: { type: local, command: [mcp-ssh-manager], timeout: 30000, enabled: false }`.
2. Keep `enabled: false` (user activates per-session via `/mcp` when VPS work is needed).
3. Update `configs/AGENTS.linux.md` (and new arch variant, Task 5) MCP bullet: no longer "Context7 e ssh-manager no mesmo config" as if enabled — state exact entries + how to enable.
4. Acceptance: `envctl opencode` (or `run shell`) deploys it; after opencode restart, `opencode debug config`
   shows `ssh-manager` under `mcp` (a shell `which mcp-ssh-manager` is NOT sufficient — interactive-shell PATH
   differs from the opencode server env, especially on GUI launch).

## Task 2 — Remove zscan everywhere + doctor guard

1. Delete the `zscan` block from `configs/opencode.json:178-186` and `configs/commandcode/mcp.json:19-28`.
2. Update descriptions: `manifests/shell.yaml:139` (drop ", zscan"), `docs/os-and-agent-matrix.md:80` (both agent columns), `configs/REFERENCE.md` (the MCP mention only — VPS hostnames like `zscanproxyprod`/`zscanchatbot` are server inventory and MUST stay).
3. Add a doctor/cleanup check: if a deployed `opencode.json` or `mcp.json` still contains a `zscan` entry (user-edited or stale), report it as drift with fix (remove block). Decide location: extend `auditSkillTree`-style config audit or a `cleanup` entry — pick the existing pattern for agent-written-config drift (`29d6bd3`).
4. Explicitly OUT of scope (history is immutable): `CHANGELOG.md` entries, `.opencode/memory/lessons.md` historical lines, `internal/usecase/temp_hygiene.go:68` `zscan-` temp prefix (defensive hygiene for leftover tarballs, installs nothing — keep).
5. Acceptance: `envctl doctor` green; a stale deployed config with `zscan` is flagged; verification grep
   returns empty (all `configs/`, `manifests/`, `docs/` refs are removed; repo-root history in `CHANGELOG.md`,
   project-memory lines in `.opencode/memory/lessons.md`, and the defensive `zscan-` temp-hygiene prefix in
   `internal/usecase/temp_hygiene.go:68` intentionally remain):
   `grep -rn "zscan" configs/ manifests/ docs/ | grep -v "zscanchatbot\|zscanproxy\|zscanintranet\|zscansaclocal\|zscanchatcomercial"`

## Task 3 — LSP: from "binary in PATH" to "handshake real + no double-spawn + documented requirements"

1. Doctor today only checks binary presence — all 14 pass, yet user sees no activation while editing files. Add handshake verification: for each LSP in `manifests/lsp.yaml`, send a minimal `initialize` over `--stdio` with stdin closed (pattern from skill `lsp-smoke-test`, never trust `--version`). Precedent: taplo responds with `registered request handler method="initialize"`; `gopls version` reports `v0.23.0`.
2. Decide failure policy: handshake failure = warning with the exact failing binary + how to reproduce (not silent green), consistent with "zero tolerance" AGENTS rule — New `doctor` output must name the LSP.
3. Fix the confirmed builtin×custom collision: upstream builtin id is `yaml-ls` (opencode docs builtin table) but the config declares key `yaml` (`configs/opencode.linux.json` + `configs/opencode.json`) — a NON-matching key spawns a SECOND server alongside the builtin (merge: `item.extensions ?? existing?.extensions ?? []` in upstream `lsp.ts`), so `.yaml/.yml` fires twice. Rename the key to the builtin id (`yaml-ls`) or set `disabled` on the redundant one. Audit the remaining custom keys the same way via `opencode debug config` + session logs (`dockerfile`, `sql`, `toml`, `pylsp` — builtin ids unconfirmed, treat as suspects, not facts).
4. Do NOT touch the four extension-less entries (`typescript`, `pyright`, `gopls`, `bash`): all four match confirmed upstream builtin ids and inherit their extensions through the merge above — they trigger correctly.
5. Document per-server spawn requirements as the expected cause of "not activating" (upstream `server.ts` + builtin table): `typescript` only starts when the project resolves `typescript/lib/tsserver.js`; `pyright`/`eslint` require the project dependency; `gopls` requires the `go` command. Consequence: in this Go repo, `.ts`/`.py` files will NOT start those LSPs even when healthy — expected, not a bug.
6. Acceptance (per stack, not blanket): opening a `.go` file in an opencode session shows `gopls` active (`go` is present); opening `.ts` in a JS project with a `typescript` dep shows `typescript` active; opening `.ts` in this repo shows it correctly idle with the documented reason. `envctl doctor` reports per-LSP handshake status plus any duplicate-spawn pairs.

## Task 4 — Memory enforcement via AGENTS.md (both agents, all OS variants)

1. Strengthen the memory bullet in `configs/AGENTS.md`, `configs/AGENTS.linux.md` (+ new arch file, Task 5), `configs/commandcode/AGENTS.md`, `configs/commandcode/AGENTS.linux.md`: LOAD at task start (project → global, one pass), SAVE on error/correction/reusable fix/preference, REFLECT at close; route through `memory-promotion` on every write (reusable multi-step process → skill + remove from memory; else stays).
2. Keep on-demand skill mechanics (no prompt-injected full memory — context cost); the enforcement is the AGENTS.md instruction the build agent auto-loads every session, plus `SKILL-INDEX.md:44` row already present.
3. Seed CachyOS/global specifics if missing (machine peculiarities: no-AVX2, fish, paru, tailnet DNS) as `patterns.md` entries, never secrets.
4. Acceptance (observable proxy, not a vibe check): a fresh build-agent session issues a `skill` tool call loading `agent-memory` at task start; a deliberately introduced mistake becomes a new `lessons.md` entry in the same session.

## Task 5 — Split AGENTS identity: arch desktop vs ubuntu server

1. Create `configs/AGENTS.arch.md` (CachyOS/Arch desktop: fish + bash-posix for agent, pacman/paru, volta, gaming stack ref, Cursor, `/temp`, user-neutral wording — no hardcoded `ubuntu` user) vs keep `configs/AGENTS.linux.md` for Ubuntu/Debian servers. Manifest `os:` families already support it (`internal/domain/entity/platform.go:47-99`: `arch`, `cachyos`, `debian`, `ubuntu` + comma lists).
2. Wire in `manifests/shell.yaml`: `agents_manifest_linux` → `os: debian,ubuntu`; new `agents_manifest_arch` (`os: arch,cachyos`) → `~/.config/opencode/AGENTS.md`. Same split for `configs/commandcode/AGENTS.linux.md` if it carries server identity (check). Fallback semantics (mandatory): `provision_shell.go:203-206` writes EVERY os-matching entry with no dedup, and today `os: linux` covers all distros — after the split, a `DistroUnknown` host (fedora, etc.) would silently get NO `AGENTS.md`. Decide before implementing: either a third generic entry ordered to lose (fragile — last match wins, order-dependent), or an explicit doctor warning when no AGENTS variant matches the host. Document the chosen semantics in `docs/os-and-agent-matrix.md` §1.
3. Fix stale numbers while touching these files: Linux LSP count is 14, not 17 (`configs/AGENTS.linux.md:29`).
4. Acceptance: `envctl run shell` on this host deploys the arch identity (no "Ubuntu Server"/"`ubuntu`" strings); on a debian host the server identity deploys; on an unmatched distro `doctor` reports the missing identity instead of staying green. `envctl doctor` green on both primary targets.

## Task 6 — Docs, matrix, changelog, verify gate

1. Update `docs/os-and-agent-matrix.md` (§1 Arch column, §2 MCP rows per agent, §3 new asymmetry entries for ssh-in-opencode-linux + zscan removal + arch/server AGENTS split), `README.md`/`AGENTS.md` counts if they change, `CHANGELOG.md` new `[Unreleased]` section.
2. Run the gate: `go build ./...`, `go vet ./...`, `go test ./...`, `golangci-lint run --new-from-rev=origin/main` (or `envctl-verify --git-push`). Verifier-script changes must keep `internal/usecase/verify_script_test.go` passing.
3. Rollout order on this machine: `envctl run all` → `envctl doctor` → restart opencode → `opencode debug config` → open one file per stack → `ssh-manager server add` for the VPSs (per-machine inventory, never committed).
