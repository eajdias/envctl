# STATE — SPEC opencode-cachyos-usability (2026-09-22)

Source of truth for implementation progress. Update after every task/step.
Resume rule: read this file + the SPEC, then continue at the first unchecked box.

## Decisions (locked)
- ssh-manager in `opencode.linux.json`: mirror Windows entry + `timeout: 30000`, `enabled: false`. (Task 1)
- zscan: delete from `configs/opencode.json` + `configs/commandcode/mcp.json`; keep CHANGELOG/lessons/`temp_hygiene.go:68`; VPS hostnames (`zscan*`) stay. (Task 2)
- LSP: rename `yaml`→`yaml-ls` (both opencode configs); do NOT touch extension-less entries; per-stack acceptance. (Task 3)
- Memory: enforce via AGENTS.md wording; observable proxy = `skill agent-memory` tool call. (Task 4)
- Identity: new `configs/AGENTS.arch.md` (`os: arch,cachyos`); linux file → `os: debian,ubuntu`; unmatched-distro fallback = doctor warning (no silent green). (Task 5)
- Scope: end-to-end. Gate: `go build/vet/test` + `golangci-lint --new-from-rev=origin/main`. (Task 6)

## Progress
- [x] Task 5.1 — `configs/AGENTS.arch.md` + `configs/commandcode/AGENTS.arch.md` created (final memory/MCP wording already in)
- [x] Task 5.2 — `manifests/shell.yaml` wiring (`debian,ubuntu` + `arch,cachyos`); fallback doctor check PENDING (bundle with Task 3 doctor work)
- [x] Task 5.3 — stale LSP count fixed (14) in `configs/AGENTS.linux.md:29`
- [x] Task 1 — `ssh-manager` (+timeout) in `configs/opencode.linux.json`; MCP bullet in linux AGENTS (arch file already final)
- [x] Task 2 — zscan blocks deleted; descriptions updated; doctor drift check (`auditRemovedMCPEntries` + 2 tests)
- [x] Task 4 — memory bullets strengthened (remaining 4 AGENTS files; arch files already final)
- [x] Task 3 — doctor handshake checks (+ AGENTS fallback check); `yaml`→`yaml-ls` rename; suspect audit vs binary ids; requirements documented (matrix #14)
- [x] Task 6 — matrix (#11-#14)/CHANGELOG [Unreleased]; full gate GREEN (build/vet/test/lint-0-issues)
- [x] Rollout on this machine (`run shell` → doctor 128/128 zero warnings → `debug config` shows ssh-manager)
- [x] VPS on-boarded: `vps_oracle_1` + `vps_oracle_2` in `~/.ssh-manager/.env` (600) + `extras/ssh_servers.md`; `server test` OK both via CLI
- [x] MCP RULE (user-locked 2026-09-22): default SEMPRE `false` + caminho padrão = CLI (zero restart); skill `ssh-vps` atualizada (seção MCP + workflow cadastro) e deployada nos 2 agentes; memória com 3 lições (stale-MCP + rebuild-binário + nunca-insista); regra `Nunca deduza, nunca insista` nos 6 AGENTS + deployada; doctor 128/128
- [x] MCP VERIFIED 2026-09-22: after user toggle OFF→ON, `ssh_list_servers` shows both VPSs; `ssh_execute(hostname && uptime)` success code 0 on both (instance-20260917-0100 / -0107). CLI + MCP paths both proven.
- [x] OPEN DECISION resolved (matrix #14): `pylsp` removed from both opencode configs (13/14 entries) — research: 2026 consensus = pyright (types) + ruff (lint); binary still provisioned
- [ ] USER-SIDE (needs new opencode session + interaction, cannot be done by agent here):
  - restart opencode (MCP/LSP/AGENTS are read at session start — `/mcp` then shows ssh-manager toggle)
  - TOGGLE ssh-manager OFF→ON in `/mcp` once (MCP process cached empty config from before registration), then MCP `ssh_list_servers`/`ssh_execute` work
  - open one file per stack and confirm LSP activation (`.go`→gopls must fire; `.ts` in this Go repo correctly idle)
  - optional: `envctl run gaming` / Cursor / `ssh-manager server add` for more hosts

## Verification log
- 2026-09-22 `go build ./...` → OK (after arch files + shell.yaml wiring + LSP count fix)
- 2026-09-22 `go test -count=1 ./internal/infra/embedded/ -run TestLoadManifestsFromDiskOrEmbed` → PASS (disk/embed consistent)
- 2026-09-22 `go test ./internal/domain/entity/ -run TestMatchOS` → PASS
- 2026-09-22 Task 1: `configs/opencode.linux.json` mcp keys = context7/ssh-manager/chrome-devtools (JSON valid, `go build` OK)
- 2026-09-22 Task 2: zscan blocks deleted (both MCP configs); `go vet ./internal/usecase/` OK; 4 new doctor tests PASS
  (also fixed a collapsed newline in doctor_audit_test.go:317 from an earlier edit)
- 2026-09-22 Task 3/6: gate GREEN (build/vet/full-test/lint 0 issues); ROLLOUT: `run shell` deployed arch AGENTS.md + opencode.json (ssh-manager/yaml-ls) + cmdc mcp.json (no zscan); `doctor` 128/128 zero warnings (zscan drift auto-resolved by re-provision); `debug config` lists ssh-manager; handshake silent-green on all 14 live LSPs
