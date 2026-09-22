# Built-in Plan Migration (Phase 1: minimal override) — Implementation Plan

> **For agentic workers:** implement this plan task-by-task, verifying after each task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the 100-line custom `plan` override with the upstream built-in `plan` plus the smallest possible delta, keeping versioned specs in `spec-agent/`.

**Architecture:** The built-in `plan` (`opencode.plan`: primary, `question allow`, `edit * deny` + `edit ~/.opencode/plan/* allow`) already covers mode, questions, and read-only posture. Our config keeps ONE appended rule (`edit spec-agent/** allow`, last-match-wins) so specs stay versioned in-repo. All teaching (spec format, skills, shell allowlist) moves to `AGENTS.md` templates + top-level `permissions`, which attach to the built-in automatically.

**Spec:** This document (the migration IS the spec; no separate design doc).

## Global Constraints

- Customs only with NEW IDs — `plan` keeps its ID solely as the grandfathered minimal exception (matrix checklist documents it).
- No behavior loss: specs still land in `spec-agent/YYYY-MM-DD-<f>.md`, shell read-only stays ask-free, `writing-plans` + `agent-memory` still load.
- `review` agent untouched (works very well — out of scope).
- Verify with `go build ./...`, `go vet ./...`, `go test ./...`, `golangci-lint run --new-from-rev=origin/main`, `opencode debug config` (zero diagnostics).
- JSON edited via script or exact-string edits (prompts are long — never retype).

---

### Task 1: Shrink `plan` to the single edit exception

**Files:**
- Modify: `configs/opencode.json` (`agents.plan` → `{"permissions": [{"action": "edit", "resource": "spec-agent/**", "effect": "allow"}]}`)
- Modify: `configs/opencode.linux.json` (same)

**Interfaces:**
- Consumes: built-in `plan` (mode, description, question/edit-base rules)
- Produces: merged effective plan = built-in + spec-agent write access

- [ ] **Step 1: Apply the trim**
  - [ ] Delete `description`, `request`, `steps`, `system` from `agents.plan` in both files, keeping only the `permissions` array with the one `spec-agent/**` rule
  - [ ] Run: `python3 -c "import json; [json.load(open(f)) for f in ['configs/opencode.json','configs/opencode.linux.json']]; print('JSON OK')"`
  - [ ] Expected: `JSON OK`

- [ ] **Step 2: Verify the merge**
  - [ ] Run: backup deployed config, copy `configs/opencode.linux.json` over it, `opencode debug config`, restore backup (same procedure as 2026-09-22 validation)
  - [ ] Expected: `agents.plan.permissions` == `[..., {edit * deny}, {edit ~/.opencode/plan/* allow}, {edit spec-agent/** allow}]`, zero normalization diagnostics

### Task 2: Move shell allowlist to top-level `permissions`

**Files:**
- Modify: `configs/opencode.json` (add top-level `permissions`)
- Modify: `configs/opencode.linux.json` (same)

**Interfaces:**
- Consumes: the read-only shell allowlist currently inside `agents.review.permissions` / `agents.plan.permissions`
- Produces: global `permissions` = `[{shell * ask}, {shell <each-read-only-cmd> allow}...]` (broad first, exceptions last — last match wins)

- [ ] **Step 1: Copy (don't move) the allowlist up**
  - [ ] Copy the exact `shell` rules (the `* ask` + all per-command `allow`s) from `agents.review.permissions` into a new top-level `permissions` array in both files. Do NOT delete the agent-level copies (review keeps its own; duplication is harmless — agent rules append after global)
  - [ ] Run: `opencode debug config` (same backup/restore procedure)
  - [ ] Expected: top-level `permissions` present, zero diagnostics

### Task 3: Teach spec conventions via global AGENTS templates

**Files:**
- Modify: `configs/AGENTS.md`, `configs/AGENTS.linux.md`, `configs/AGENTS.arch.md` (add identical planning section)

**Interfaces:**
- Consumes: `writing-plans` skill (carries `spec-agent/` convention + TDD format), `agent-memory`
- Produces: built-in `plan` learns specs without a custom system prompt

- [ ] **Step 1: Add the planning section**
  - [ ] Append (before `## Referências`): planning = Tab `plan` agent; specs in `spec-agent/YYYY-MM-DD-<f>.md`; load `writing-plans` + `agent-memory` first; TDD bite-sized tasks; real test commands; verify before declaring done
  - [ ] Keep to ≤10 lines per file (AGENTS.md is auto-loaded every turn — brevity matters)

### Task 4: Sync docs + memory

**Files:**
- Modify: `configs/REFERENCE.md` (plan bullet: override mínimo de 1 regra, resto built-in)
- Modify: `docs/os-and-agent-matrix.md` (checklist `Agente custom`: plan = exceção mínima documentada, 1 regra de edit)
- Modify: `CHANGELOG.md` (`[Unreleased]`: append bullets — do NOT rewrite the 2026-09-22 section)
- Modify: `.opencode/memory/lessons.md` (append 1 lesson with the outcome)

- [ ] **Step 1: Apply doc edits**
  - [ ] Run: `grep -rn "plan.*override\|override.*plan" configs/REFERENCE.md docs/os-and-agent-matrix.md | head`
  - [ ] Expected: each file mentions the single-rule exception exactly once, no stale "full custom plan" claims

### Task 5: Live experiment (costs tokens — confirm with owner first)

**Files:**
- None (runtime validation only)

- [ ] **Step 1: Real planning session**
  - [ ] Run: `envctl run shell` from repo root (rebuild + deploy), restart opencode, switch to `plan` agent, ask for a spec of a small feature
  - [ ] Expected: loads `writing-plans`, writes `spec-agent/<date>-<f>.md` without permission errors, shell read-only commands run without asks

- [ ] **Step 2: Rollback check**
  - [ ] Run: `git stash && envctl run shell` restores previous behavior (proves the change is revertible in one command)

### Task 6: Full gate

- [ ] **Step 1: Run the gate**
  - [ ] Run: `go build ./... && go vet ./... && go test ./...`
  - [ ] Expected: all `ok`, zero failures
  - [ ] Run: `golangci-lint run --new-from-rev=origin/main`
  - [ ] Expected: `0 issues.`

---

## Out of scope

- `review` agent (works very well — do not touch).
- Renaming `plan` to a new ID (breaks Tab muscle memory + all AGENTS references; revisit only if upstream changes built-in plan semantics).
- `cli.json` management (user-owned, documented in REFERENCE.md).
