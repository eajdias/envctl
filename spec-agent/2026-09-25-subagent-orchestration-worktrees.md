# Subagent Context Isolation, Skill Activation, and Worktree Safety

## Goal

Improve agent ergonomics without inflating the default context or token bill:

1. expose one dispatchable planning subagent while preserving the built-in `plan` primary agent;
2. make skill selection proactive through concise routing rules and trigger-rich descriptions;
3. standardize repository worktrees under a project-local OpenCode configuration and report stale/prunable worktrees;
4. replace stale supervision vocabulary with the real OpenCode V2 session-interrupt path.

The default remains inline execution. A subagent is justified only when it isolates bulky research/planning context and returns a compact artifact.

## Scope and Non-Goals

### In scope

- Add a new `planner` custom agent with `mode: "subagent"` to both OpenCode config templates.
- Add static tests that validate the shipped OpenCode config shape and the planner contract.
- Add one concise proactive-skill rule to project and provisioned agent instructions.
- Sharpen a small, high-value set of skill descriptions and fix the non-folded VPS description.
- Add a project-local `.opencode/opencode.json` worktree directory convention.
- Add ignored `.worktrees/` paths to the repository `.gitignore`.
- Extend `doctor` with a pure `git worktree list --porcelain` parser and stale/prunable diagnostics.
- Rewrite `subagent-supervision` to use OpenCode V2 session IDs and the documented interrupt endpoint; do not claim a nonexistent `agent_output` tool.
- Keep docs, `REFERENCE.md`, matrix, and memory aligned.

### Not in scope

- A new V2 plugin or custom MCP server for subagent supervision. The repository has no OpenCode server client or plugin test harness; the first implementation uses the documented CLI/API path and is validated without a live destructive session.
- A state-changing `envctl worktree add/remove` command. Worktree creation remains explicit and consent-based; this pass only standardizes configuration, documentation, and audit.
- A large roster of specialized auditor/implementer subagents. The user preference is inline-first and subagents only for context isolation.

## Current Evidence

- `opencode v2.0.14` is installed locally; `opencode debug agents` shows `plan` and `review` as `primary`, and `explore`/`general` as `subagent`.
- `configs/opencode.json` and `configs/opencode.linux.json` contain the same `agents` block; `plan` intentionally has no `mode` so the built-in primary behavior is preserved.
- The catalog of skills is injected from frontmatter descriptions; the deployed AGENTS files currently say skills are on-demand but do not explicitly instruct the model to load a matching skill before acting.
- `configs/skills/vps-agent-dispatch/SKILL.md` is the only description using a plain scalar instead of the established folded block form.
- `doctor_audit.go` currently checks only that `git worktree list` succeeds and discards its output.
- `subagent-supervision` references `agent_output`, `agent_id`, and `run_in_background`, which are not present in the current OpenCode V2 toolset.
- The OpenCode V2 API documents `POST /api/session/{sessionID}/interrupt`; the local `opencode api` command reaches that route and returns `SessionNotFoundError` for an unknown ID.

## Design Decisions

### Planner agent

Use a new ID, `planner`, rather than changing the built-in `plan` ID:

- `mode: "subagent"` makes it dispatchable without adding another Tab primary;
- reuse the read-only shell allowlist and `spec-agent/**` write boundary from `plan`;
- include a non-empty `description` because the description is what the parent model uses to choose a subagent;
- keep the system prompt short and return a compact plan artifact;
- keep `experimental.subagent_depth: 2` unchanged for the existing review/plan -> explore/general topology.

### Proactive skills

Do not paste the full skill index into every prompt. Add one compact rule: if a skill description matches the current task, load it with the `skill` tool before acting; use `SKILL-INDEX.md` only to break ties. Strengthen only the high-value descriptions whose trigger boundary is currently implicit, and keep every description folded/valid.

### Worktrees

Use `.worktrees/<type>-<slug>` as the project convention, matching the existing linked worktree and keeping the configuration local to this repository. Ignore `/.worktrees/` in the repository so another clone does not treat local worktrees as project files. The doctor reports prunable entries as warnings and locked entries as informational, never auto-removing anything.

### Supervision

Describe the actual V2 lifecycle:

- foreground child: parent interruption cancels the child;
- background child: keep the returned child session ID and interrupt it with `opencode api post /api/session/<sessionID>/interrupt`;
- hard process termination is a separate operation and requires an explicitly tracked PID;
- never invent `agent_output`, `agent_id`, or a model-facing kill tool;
- retry at most once with a refined scope, then escalate;
- preserve worktree changes for inspection.

## Risks and Unknowns

### Risks

- **Prompt growth:** copying the full skill index into AGENTS would increase every turn's context. Mitigation: one routing sentence and sharper existing descriptions only.
- **Config merge:** the OpenCode configs are overwritten by provisioning; tests must read the embedded source templates, not only the deployed machine copy.
- **Worktree false positives:** `locked` is a valid user state and must not be a warning; prunable output is actionable but must not be auto-pruned.
- **API semantics:** `sessionID` interruption is documented, but a destructive end-to-end test is not safe in the shared server. Mitigation: test route existence with a nonexistent ID and document the live procedure separately.
- **Cross-agent parity:** CommandCode has its own agent frontmatter and built-ins; do not copy OpenCode JSON agents into CommandCode without a separate contract.

### Unknowns / decisions resolved for this pass

- Whether to build a full supervision plugin: **deferred**. The V2 API/CLI path is sufficient for the documented workflow and avoids a new dependency/runtime surface.
- Whether to add a worktree create/remove CLI: **deferred**. The user asked for standardization, not new state-changing commands; native OpenCode and Git already provide the operation.
- Whether to create extra auditor agents: **deferred**. Inline execution is the default and the project must measure a real context-isolation benefit first.

## Tasks

### Task 1 — Add the dispatchable planner and config contract tests

**Files:**

- Modify: `configs/opencode.json`
- Modify: `configs/opencode.linux.json`
- Create: `internal/infra/embedded/opencode_config_test.go`

**Interfaces:**

- Both templates expose `agents.planner.mode == "subagent"`;
- `agents.planner.description` is non-empty and bounded by the skill/config description limit;
- the planner does not collide with built-in IDs `build`, `plan`, `general`, or `explore`;
- permission entries use the current V2 `action` vocabulary.

**TDD:**

1. Write a table test that reads both embedded config templates and asserts the planner contract.
2. Run `go test ./internal/infra/embedded -run TestOpenCodeConfigTemplates -v`; expected RED because `planner` does not exist.
3. Add the minimal `planner` definition to both JSON templates.
4. Re-run the focused test; expected GREEN.

**Verification:**

- `go test ./internal/infra/embedded -v`
- `opencode debug config` and `opencode debug agents` after a controlled local config copy/reload; confirm `planner` is a subagent and `plan` remains primary.

**Rollback:** revert only the two config files and the new test.

### Task 2 — Make skill activation proactive and guard descriptions

**Files (as implemented):**

- Modify: `AGENTS.md`
- Modify: `configs/AGENTS.md`, `configs/AGENTS.linux.md`, `configs/AGENTS.arch.md`
- Modify: `configs/commandcode/AGENTS.md`, `configs/commandcode/AGENTS.linux.md`, `configs/commandcode/AGENTS.arch.md`
- Modify: `configs/skills/{writing-plans,verification-before-completion,systematic-debugging,test-driven-development,docs-sync,vps-agent-dispatch}/SKILL.md` (description only)
- Modify: `configs/skills/git-workflow/SKILL.md` (description + worktree convention body)
- Modify: `configs/skills/subagent-routing/SKILL.md` (description + inline-first body)
- Create: `internal/usecase/skill_content_contract_test.go`

**Note:** `context7-auto` and `grill-me` were audited and left unchanged — their descriptions already carry mandatory/imperative triggers, so rewriting them would only add prompt bytes.

**Interfaces:**

- every instruction file carries the same concise proactive-loading rule;
- descriptions stay within the 1024-rune loader limit and use valid YAML frontmatter;
- the skill catalog remains the matching mechanism; no index table is copied into every prompt.

**TDD:**

1. Add a test for the folded-description and trigger-marker contract, plus a regression case for a plain scalar containing `: `.
2. Run the focused test and observe the expected RED for the current VPS description.
3. Update the instruction sentence and descriptions; convert `vps-agent-dispatch` to `description: >-`.
4. Run `go test ./internal/usecase -run 'Test(SkillFrontmatter|SkillDescription)' -v` and the embedded manifest test; expected GREEN.

**Verified (2026-09-25):** RED observed on all six AGENTS variants, on `subagent-routing` (no inline rule) and on `subagent-supervision` (stale `agent_output`/`agent_id`/`run_in_background`). GREEN after the edits; deployed copy re-extracted with 0 skills pruned, and the live skill catalog in the next session shows the new descriptions.

**Verification:**

- `go test ./internal/usecase ./internal/infra/embedded -v`
- `go build ./...`
- `go vet ./...`
- `go test ./...`

**Rollback:** revert the instruction/description files and the contract test; no runtime data is changed.

### Task 3 — Standardize worktree configuration and add doctor hygiene

**Files (as implemented):**

- Create: `.opencode/opencode.json`
- Modify: `.gitignore`
- Create: `internal/usecase/worktree_report.go`
- Create: `internal/usecase/worktree_report_test.go`
- Modify: `internal/usecase/doctor_audit.go`

**Added beyond the original plan:** `internal/usecase/opencode_config_shape.go` + `_test.go` (the `Config shape` doctor check and the shipped-template contract), because a silent V1-shaped config is exactly the failure mode that produced the stale `agent_output` documentation.

**Interfaces:**

- `parseWorktreeListPorcelain(string) ([]gitWorktree, error)` parses blank-line-separated records with `worktree`, `HEAD`, `branch`, `locked`, and `prunable` fields;
- `worktreeFindings([]gitWorktree) []entity.Diagnostic` reports prunable entries as warnings and locked entries as informational;
- the existing git availability/inside-repository guard remains unchanged;
- no command removes, prunes, commits, or resets a worktree.

**TDD:**

1. Write table tests for valid records, detached/bare records, locked entries, prunable entries, malformed input, and empty output.
2. Run `go test ./internal/usecase -run 'Test(ParseWorktreeListPorcelain|WorktreeFindings)' -v`; expected RED because the parser does not exist.
3. Implement the pure parser and findings function.
4. Wire the existing `git worktree list --porcelain` output into the doctor block.
5. Re-run the focused tests; expected GREEN.

**Verification:**

- `go test ./internal/usecase -run 'Worktree' -v`
- `go test ./...`
- run `envctl doctor` from a disposable repository fixture or the current repository and confirm the new diagnostics have the intended severity.

**Rollback:** remove the project config/ignore entry and the new parser/audit wiring; do not touch existing worktree directories.

### Task 4 — Correct subagent supervision to the real V2 lifecycle

**Files:**

- Modify: `configs/skills/subagent-supervision/SKILL.md`
- Modify: `configs/skills/task-hang-watchdog/SKILL.md` if its wording still implies nonexistent OpenCode tools
- Modify: `configs/REFERENCE.md`
- Modify: `docs/skills.md`
- Create: a focused content contract test for the supervision skill if needed

**Interfaces:**

- the skill distinguishes session interruption from OS-process termination;
- the OpenCode path is documented as `opencode api post /api/session/<sessionID>/interrupt`;
- CommandCode-specific vocabulary is not presented as an OpenCode tool;
- no `agent_output`, `agent_id`, `run_in_background`, `kill_shell`, or `monitor_command` claim remains in the OpenCode-facing supervision guidance.

**TDD:**

1. Add a regression test asserting the supervision skill contains the V2 interrupt path and does not contain the stale tool names.
2. Run it and observe RED against the current skill.
3. Rewrite the skill body and the operational reference.
4. Re-run the test; expected GREEN.

**Verification:**

- focused content test;
- `opencode api get /api/info` and a safe nonexistent-session interrupt probe (expect a structured `SessionNotFoundError`, proving route discovery without touching a live session);
- no live session is killed during automated verification.

**Rollback:** restore the previous skill/reference text; the API path is documented, not a server-side mutation.

### Task 5 — Synchronize documentation and memory

**Files:**

- Modify: `docs/os-and-agent-matrix.md`
- Modify: `docs/verification.md` if the new doctor checks need documenting
- Modify: `configs/REFERENCE.md`
- Modify: `CHANGELOG.md` under `[Unreleased]` only
- Modify: `.opencode/memory/lessons.md` and/or `.opencode/memory/patterns.md` when a verified lesson is learned

**Requirements:** no new skill is added, so the manifest skill count remains 50 and no `SKILL-INDEX` row is required. Counts in docs must match manifests; release-please remains the only owner of version sections/tags.

### Task 6 — Final verification and runtime smoke checks

**Commands:**

```bash
gofmt -w <changed-go-files>
go test ./internal/usecase -run 'Test(ParseWorktreeListPorcelain|WorktreeFindings|SkillFrontmatter|SkillDescription)' -v
go test ./internal/infra/embedded -run 'Test(OpenCodeConfigTemplates|LoadManifestsFromDiskOrEmbed)' -v
go build ./...
go vet ./...
go test ./...
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build ./cmd/envctl
git diff --check
git status --short --branch --untracked-files=all
```

Then validate the local OpenCode config with `opencode debug config` and `opencode debug agents`. A fresh session is required for the new `planner` and skill-routing instructions to be observed.

## Definition of Done

- [x] `planner` is a real dispatchable OpenCode subagent in both shipped templates — `opencode debug agents` shows `planner mode=subagent, hidden=false` and `plan mode=primary`.
- [x] The built-in `plan` remains primary and is not overridden (`plan mode=<none>` in the resolved config).
- [x] Proactive skill loading is explicit but adds no full index to the per-turn prompt (one sentence per AGENTS file).
- [x] All modified skill descriptions parse and satisfy the loader contract (`go test ./internal/usecase ./internal/infra/embedded`).
- [x] Worktree path convention is project-local, ignored, documented, and auditable (`.opencode/opencode.json` + `.gitignore` + skill + matrix).
- [x] Doctor detects prunable worktrees without mutating or force-removing anything — proven end-to-end on a disposable repo: `Worktree "…" is prunable: gitdir file points to non-existent location` as WARN, 208 checks / 0 errors.
- [x] Supervision documentation uses the real V2 interrupt path and no invented tools (content contract test).
- [x] Focused tests, full Go suite, build, vet, cross-build, and `git diff --check` have fresh evidence — `go build ./...`, `go vet ./...`, `go test ./...` all green; `golangci-lint run --new-from-rev=origin/main` → 0 issues; `GOOS=windows CGO_ENABLED=0 go build` OK.
- [x] Machine re-provisioned from this tree (`envctl opencode` + `envctl commandcode` with the freshly built binary) and `envctl doctor` → 207/207, 0 warnings, 0 errors.
- [x] No secrets, machine-specific paths, force operations, or out-of-scope files enter the diff.
- [x] No commit or push is performed without explicit user confirmation.
