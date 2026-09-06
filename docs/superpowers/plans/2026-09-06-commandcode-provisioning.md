# CommandCode Provisioning Support in envctl

> **For agentic workers:** implement this plan task-by-task — dispatch a fresh `general` subagent per task via the task tool, or execute inline with checkpoints. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make envctl provision both OpenCode AND CommandCode equivalently — same skills, equivalent MCP/agents/config, so users can choose either agent.

**Architecture:** Extend the existing provisioning pipeline (manifests → use cases → CLI) to dual-target CommandCode. Skills deploy to both `~/.config/opencode/skills/` and `~/.commandcode/skills/` (same SKILL.md format). Config templates for CommandCode settings, MCP, and agents are added to `configs/commandcode/` and deployed via the existing `provision_shell.go` config-file pipeline. Bootstrap installs CommandCode CLI alongside OpenCode on Linux.

**LSP Note:** CommandCode has **built-in LSP integration** — it auto-detects and manages language servers internally through its IDE connection (tool: `get_diagnostics`). There is no explicit LSP configuration in `settings.json` or any CommandCode config file. Therefore, LSP provisioning (`lsp.yaml` + `provision_lsp.go`) remains **OpenCode-only**. CommandCode users get LSP support automatically when connected to VS Code or similar IDE. No changes needed to the LSP provisioning pipeline.

**Tech Stack:** Go 1.24, Cobra, PTerm, YAML manifests, `//go:embed`

**Spec:** Web research on CommandCode architecture (https://commandcode.ai/docs)

## Global Constraints

- Go module: `github.com/eajdias/envctl`
- All new files under `internal/` follow Clean Architecture: domain → usecase → infrastructure → CLI
- Config templates go in `configs/` (embedded via `//go:embed`)
- Manifests go in `manifests/` (YAML, embedded)
- Commands use non-interactive flags only (no prompts, no `yes |`)
- Code/comments in English; user-facing strings can reference both tools
- `envctl doctor` and `envctl run all` must remain idempotent
- Zero warnings tolerance: `go vet`, `go build` must pass clean

## LSP Provisioning Decision

**OpenCode** requires explicit LSP configuration in `opencode.json` under the `"lsp"` key. envctl provisions 18 LSP servers via `lsp.yaml` and `provision_lsp.go` — each with `command`, `args`, `install_type`, `check_binary`. This is OpenCode-specific and remains unchanged.

**CommandCode** has **built-in LSP integration** BUT it is **IDE-dependent** (source: https://commandcode.ai/docs/ide-integration + https://commandcode.ai/docs/reference/tools):

- The `get_diagnostics` tool "pulls your editor's own lint and type-checking output" — **only available while the IDE is connected** (VS Code, Cursor, Windsurf).
- **Without IDE connection** (terminal-only usage): "diagnostics simply aren't there and CommandCode works as usual, **running your project's own lint or typecheck commands if you ask it to**."
- CommandCode does NOT have a standalone LSP client. It relies on the IDE's LSP infrastructure.

**Impact for terminal-only usage:**
- `get_diagnostics` tool is **not advertised** in terminal-only sessions
- CommandCode falls back to running lint/typecheck via `shell_command` (e.g., `pyright`, `tsc --noEmit`, `go vet`)
- The LSP binaries provisioned by envctl are still useful — CommandCode can invoke them via shell when asked

**Conclusion:** LSP provisioning does NOT need to be extended to CommandCode's config. The existing `lsp.yaml` + `provision_lsp.go` pipeline stays OpenCode-only. For terminal-only CommandCode users, LSP works indirectly through shell commands. If they later connect through an IDE terminal, `get_diagnostics` becomes available automatically.

---

## File Structure

### New Files
| File | Responsibility |
|------|---------------|
| `configs/commandcode/settings.json` | CommandCode project settings (permissions, minimal config) |
| `configs/commandcode/AGENTS.md` | CommandCode global rules (mirrors OpenCode's AGENTS.md) |
| `configs/commandcode/mcp.json` | CommandCode user-scope MCP servers (translated from OpenCode) |
| `configs/commandcode/agents/review.md` | Review agent (translated from OpenCode JSON) |
| `configs/commandcode/agents/plan.md` | Plan agent (translated from OpenCode JSON) |
| `configs/commandcode/agents/goal.md` | Goal agent (translated from OpenCode JSON) |
| `internal/usecase/cleanup_commandcode.go` | CommandCode-specific stale artifact cleanup |

### Modified Files
| File | Change |
|------|--------|
| `manifests/packages.yaml` | Add `command-code` volta package |
| `manifests/shell.yaml` | Add CommandCode config files, directories, cleanup entries |
| `internal/ui/cli/run.go` | Update `runSkillsProvisioning()` to dual-target, update `runAllProvisioning()` labels, update `runCleanup()` |
| `internal/ui/cli/root.go` | Add `CleanupCommandCodeUC` to `AppContext` and `InitApp()`; update descriptions |
| `internal/ui/cli/doctor.go` | Update `--fix` labels to reflect dual-agent provisioning |
| `internal/usecase/provision_bootstrap.go` | Add CommandCode CLI installation step on Linux |
| `internal/usecase/doctor_audit.go` | Add CommandCode health checks |

---

## Task 1: Add `command-code` to packages.yaml

**Files:**
- Modify: `manifests/packages.yaml` (~line 450, volta section)

**Interfaces:**
- Consumes: existing `Package` struct (no changes needed)
- Produces: `command-code` package entry installable via `envctl run volta`

- [ ] **Step 1: Add the package entry**

Add after the existing volta packages (near the end of the volta section, before the pip section):

```yaml
  # ============================================================================
  # AI Coding Agents
  # ============================================================================
  - id: command-code
    name: CommandCode CLI
    type: volta
    os: linux
    category: ai-agents
    check_command: cmd --version
```

Note: `cmd` is the Linux alias; on Windows it would be `cmdc`. Since volta manages node globals and the bootstrap runs on Linux, `os: linux` is correct. On Windows, the user installs manually or via a future winget entry.

- [ ] **Step 2: Verify it parses**

Run: `go build ./...`
Expected: clean build, no errors

- [ ] **Step 3: Commit**

```bash
git add manifests/packages.yaml
git commit -m "feat(provisioning): add command-code to volta packages manifest"
```

---

## Task 2: Create CommandCode config templates

**Files:**
- Create: `configs/commandcode/settings.json`
- Create: `configs/commandcode/AGENTS.md`
- Create: `configs/commandcode/mcp.json`
- Create: `configs/commandcode/agents/review.md`
- Create: `configs/commandcode/agents/plan.md`
- Create: `configs/commandcode/agents/goal.md`

**Interfaces:**
- Consumes: OpenCode's MCP server definitions (from `configs/opencode.json`), OpenCode's agent definitions (from `configs/opencode.json`), OpenCode's AGENTS.md rules
- Produces: 6 config template files deployable via `provision_shell.go`

- [ ] **Step 1: Create `configs/commandcode/settings.json`**

```json
{
  "$schema": "https://commandcode.ai/schema/settings.json",
  "permissions": {
    "allow": [
      "read",
      "edit",
      "write",
      "glob",
      "grep",
      "bash(git:*)",
      "bash(go:*)",
      "bash(npm:*)",
      "bash(npx:*)",
      "bash(pnpm:*)",
      "bash(bun:*)",
      "bash(bunx:*)",
      "bash(pytest:*)",
      "bash(go test:*)",
      "bash(go vet:*)",
      "bash(cargo test:*)",
      "bash(dotnet test:*)",
      "bash(docker:*)",
      "skill",
      "webfetch",
      "websearch",
      "todowrite",
      "todoread",
      "task"
    ],
    "ask": [
      "bash(rm:*)",
      "bash(sudo:*)",
      "bash(git push:*)",
      "bash(git commit:*)"
    ],
    "deny": []
  }
}
```

- [ ] **Step 2: Create `configs/commandcode/AGENTS.md`**

Copy the content from `configs/AGENTS.md` — it's the same global rules manifesto. CommandCode loads `~/.commandcode/AGENTS.md` as its global memory file.

Run: `cp configs/AGENTS.md configs/commandcode/AGENTS.md`

- [ ] **Step 3: Create `configs/commandcode/mcp.json`**

This is the user-scope MCP config for CommandCode. Translate OpenCode's MCP servers (from `configs/opencode.json` `"mcp"` key) to CommandCode's format:

```json
{
  "mcpServers": {
    "context7": {
      "command": "npx",
      "args": ["-y", "@upstash/context7-mcp@latest"],
      "type": "stdio"
    },
    "playwright": {
      "command": "npx",
      "args": ["-y", "@playwright/mcp@latest"],
      "type": "stdio"
    }
  }
}
```

Note: GitHub and Notion MCPs use HTTP transport with auth tokens. We provision the stdio servers (context7, playwright) that don't require auth. The user can add HTTP servers manually via `cmd mcp add-json`.

- [ ] **Step 4: Create `configs/commandcode/agents/review.md`**

Translate from OpenCode's `review` agent (JSON in `configs/opencode.json`):

```markdown
---
name: review
description: "Use when you need to review code for correctness, security, performance, or style — read-only analysis with evidence"
tools:
  - read
  - grep
  - glob
  - webfetch
  - websearch
model: inherit
reasoningEffort: high
maxTurns: 40
background: false
---

You are a code review agent. You perform read-only analysis of code changes.

## Rules
- Never modify files — read-only analysis only
- Provide evidence with file:line references
- Check for correctness, security, performance, style
- Use `go vet`, `go build`, `go test` (read-only) to verify
- Report findings with severity (error/warning/info)
- If you find issues, provide exact fix suggestions with code blocks
```

- [ ] **Step 5: Create `configs/commandcode/agents/plan.md`**

```markdown
---
name: plan
description: "Use for implementation planning — read-only research, file analysis, and plan generation"
tools:
  - read
  - grep
  - glob
  - webfetch
  - websearch
model: inherit
reasoningEffort: high
maxTurns: 30
background: false
---

You are a planning agent. You create implementation plans based on research and analysis.

## Rules
- Never modify files — read-only analysis only
- Use bash read-only commands (git log, git diff, go test, etc.) to gather evidence
- Create plans with exact file paths, line numbers, and code snippets
- Follow the writing-plans skill format when available
- Save plans to `docs/superpowers/plans/` directory
- Each task must be bite-sized (2-5 minutes) with testable deliverables
- No placeholders — every step needs actual content
```

- [ ] **Step 6: Create `configs/commandcode/agents/goal.md`**

```markdown
---
name: goal
description: "Autonomous agent that can execute multi-step tasks with full tool access — use for complex automated workflows"
tools:
  - "*"
model: inherit
reasoningEffort: high
maxTurns: 100
background: true
---

You are an autonomous coding agent with full tool access.

## Rules
- Execute tasks end-to-end without stopping for approval
- Use all available tools as needed
- Follow the project's coding conventions
- Run tests before declaring work complete
- Commit with conventional commit messages
- If blocked, report the blocker clearly and stop
```

- [ ] **Step 7: Verify all templates are valid**

Run: `go build ./...`
Expected: clean build (templates are embedded via `//go:embed`)

- [ ] **Step 8: Commit**

```bash
git add configs/commandcode/
git commit -m "feat(provisioning): add CommandCode config templates (settings, MCP, agents, AGENTS.md)"
```

---

## Task 3: Add CommandCode entries to shell.yaml

**Files:**
- Modify: `manifests/shell.yaml` (add to config_files, directories, cleanup sections)

**Interfaces:**
- Consumes: config templates from Task 2
- Produces: manifest entries that `provision_shell.go` deploys automatically

- [ ] **Step 1: Add CommandCode config files to `config_files` section**

Add after the existing OpenCode config file entries (after `memory_patterns_template` at line ~90, before `user_package_json` at line ~92):

```yaml
  # --- CommandCode Configs ---
  - id: commandcode_settings
    description: "CommandCode project settings (permissions, hooks)"
    source: "configs/commandcode/settings.json"
    destination: "~/.commandcode/settings.json"
    strict_acl: false
    category: "commandcode"

  - id: commandcode_agents_manifest
    description: "CommandCode global rules manifesto (auto-loaded from ~/.commandcode/AGENTS.md)"
    source: "configs/commandcode/AGENTS.md"
    destination: "~/.commandcode/AGENTS.md"
    strict_acl: false
    category: "commandcode"

  - id: commandcode_mcp
    description: "CommandCode user-scope MCP servers (context7, playwright)"
    source: "configs/commandcode/mcp.json"
    destination: "~/.commandcode/mcp.json"
    strict_acl: false
    category: "commandcode"

  - id: commandcode_agent_review
    description: "CommandCode review agent definition"
    source: "configs/commandcode/agents/review.md"
    destination: "~/.commandcode/agents/review.md"
    strict_acl: false
    category: "commandcode"

  - id: commandcode_agent_plan
    description: "CommandCode plan agent definition"
    source: "configs/commandcode/agents/plan.md"
    destination: "~/.commandcode/agents/plan.md"
    strict_acl: false
    category: "commandcode"

  - id: commandcode_agent_goal
    description: "CommandCode goal agent definition"
    source: "configs/commandcode/agents/goal.md"
    destination: "~/.commandcode/agents/goal.md"
    strict_acl: false
    category: "commandcode"
```

- [ ] **Step 2: Add CommandCode directories to `directories` section**

Add after the existing OpenCode directory entries (after `~/.config/opencode/extras` at line ~203):

```yaml
  - path: "~/.commandcode/skills"
    strict_acl: false
    description: "CommandCode agent skills directory"
  - path: "~/.commandcode/agents"
    strict_acl: false
    description: "CommandCode agent definitions directory"
  - path: "~/.commandcode/memory"
    strict_acl: false
    description: "CommandCode agent memory directory"
```

- [ ] **Step 3: Add CommandCode cleanup entries**

Add to the `cleanup` section (after the existing OpenCode cleanup items):

```yaml
  - id: stale_commandcode_settings_jsonc
    description: "Remove stale CommandCode settings.jsonc (standardized on settings.json)"
    path: "~/.commandcode/settings.jsonc"
    category: "commandcode"
```

- [ ] **Step 4: Verify YAML parses correctly**

Run: `go build ./...`
Expected: clean build

- [ ] **Step 5: Commit**

```bash
git add manifests/shell.yaml
git commit -m "feat(provisioning): add CommandCode entries to shell.yaml (configs, dirs, cleanup)"
```

---

## Task 4: Extend skills provisioning to dual-target

**Files:**
- Modify: `internal/ui/cli/run.go:275-295` (`runSkillsProvisioning`)

**Interfaces:**
- Consumes: `ProvisionSkillsUC.Execute(ctx, targetDir)` — already supports custom target
- Produces: Skills deployed to BOTH `~/.config/opencode/skills/` AND `~/.commandcode/skills/`

- [ ] **Step 1: Modify `runSkillsProvisioning()` to deploy to both targets**

Replace the function at `run.go:275-295`:

```go
func runSkillsProvisioning() {
	spinner, _ := pterm.DefaultSpinner.Start("Deploying agent skills to OpenCode & CommandCode...")
	ctx := context.Background()

	// Deploy to OpenCode (default target)
	results, err := appCtx.ProvisionSkillsUC.Execute(ctx, "")
	if err != nil {
		spinner.Fail(fmt.Sprintf("Failed skills deployment to OpenCode: %v", err))
		return
	}

	deployedOC := 0
	for _, r := range results {
		if r.Status == entity.DiagOK {
			deployedOC++
		} else {
			pterm.Warning.Printf("  • [OpenCode] Skill %s failed: %s\n", r.SkillName, r.ErrorMessage)
		}
	}

	// Deploy to CommandCode
	resultsCC, err := appCtx.ProvisionSkillsUC.Execute(ctx, "~/.commandcode/skills")
	if err != nil {
		spinner.Fail(fmt.Sprintf("Failed skills deployment to CommandCode: %v", err))
		return
	}

	deployedCC := 0
	for _, r := range resultsCC {
		if r.Status == entity.DiagOK {
			deployedCC++
		} else {
			pterm.Warning.Printf("  • [CommandCode] Skill %s failed: %s\n", r.SkillName, r.ErrorMessage)
		}
	}

	spinner.Success(fmt.Sprintf("Deployed %d skills to OpenCode, %d to CommandCode", deployedOC, deployedCC))
}
```

- [ ] **Step 2: Update `runAllProvisioning()` label**

Change `run.go:185` from:
```go
PrintSection(section(5, "Provisioning OpenCode Agent Skills"))
```
to:
```go
PrintSection(section(5, "Provisioning Agent Skills (OpenCode + CommandCode)"))
```

- [ ] **Step 3: Update `doctor.go` auto-fix label**

Change `doctor.go:106` from:
```go
PrintSection("4/5 Remediating OpenCode Agent Skills")
```
to:
```go
PrintSection("4/5 Remediating Agent Skills (OpenCode + CommandCode)")
```

- [ ] **Step 4: Verify build**

Run: `go build ./...`
Expected: clean build

- [ ] **Step 5: Commit**

```bash
git add internal/ui/cli/run.go internal/ui/cli/doctor.go
git commit -m "feat(provisioning): deploy skills to both OpenCode and CommandCode"
```

---

## Task 5: Add CommandCode CLI installation to bootstrap

**Files:**
- Modify: `internal/usecase/provision_bootstrap.go:237-243` (after OpenCode CLI step)

**Interfaces:**
- Consumes: existing `step()` helper, npm global install pattern
- Produces: `cmd` (CommandCode CLI) installed alongside `opencode` on Linux

- [ ] **Step 1: Add CommandCode CLI installation step**

Add after the OpenCode CLI step (line 243) and before the GitHub CLI step (line 245):

```go
	// 3.5. CommandCode CLI - npm global (user prefix).
	uc.step(ctx, result, "command-code", "CommandCode CLI",
		`set -e
export PATH="$HOME/.volta/bin:$HOME/.local/bin:$PATH"
npm install -g --no-audit --no-fund --prefix "$HOME/.local" command-code`)
```

- [ ] **Step 2: Update bootstrap description**

Change the `Use:` field in `run.go:58` from:
```go
Short: "Provision the Linux toolchain (Volta, Node, OpenCode CLI, gh, delta, yq, uv, ruff, oh-my-posh, fd)",
```
to:
```go
Short: "Provision the Linux toolchain (Volta, Node, OpenCode + CommandCode CLI, gh, delta, yq, uv, ruff, oh-my-posh, fd)",
```

- [ ] **Step 3: Verify build**

Run: `go build ./...`
Expected: clean build

- [ ] **Step 4: Commit**

```bash
git add internal/usecase/provision_bootstrap.go internal/ui/cli/run.go
git commit -m "feat(provisioning): install CommandCode CLI alongside OpenCode in Linux bootstrap"
```

---

## Task 6: Add CommandCode health checks to doctor

**Files:**
- Modify: `internal/usecase/doctor_audit.go` (add CommandCode section after existing checks)

**Interfaces:**
- Consumes: existing `AuditReport`, `Diagnostic` structs
- Produces: CommandCode-specific health diagnostics

- [ ] **Step 1: Add CommandCode checks to `Execute()`**

Add a new section at the end of `Execute()` (before the return statement). Add `"os/exec"` to imports if not already present:

```go
	// N. CommandCode Agent Health
	if runtime.GOOS == "linux" || runtime.GOOS == "windows" {
		// Check if CommandCode CLI is installed
		cmdPath, cmdErr := exec.LookPath("cmd")
		if cmdErr != nil {
			// Try Windows alias
			cmdPath, cmdErr = exec.LookPath("cmdc")
		}
		if cmdErr != nil {
			report.Diagnostics = append(report.Diagnostics, entity.Diagnostic{
				Category: entity.DiagWarning,
				System:   "CommandCode",
				Target:   "CommandCode CLI",
				Details:  "CommandCode CLI not found in PATH",
				FixHint:  "Run 'envctl run bootstrap' or 'npm install -g command-code'",
			})
		} else {
			report.Diagnostics = append(report.Diagnostics, entity.Diagnostic{
				Category: entity.DiagOK,
				System:   "CommandCode",
				Target:   "CommandCode CLI",
				Details:  fmt.Sprintf("Found at %s", cmdPath),
			})
		}

		// Check CommandCode config directory
		ccConfigDir, _ := uc.fsManager.ExpandUserPath("~/.commandcode")
		if uc.fsManager.Exists(ccConfigDir) {
			report.Diagnostics = append(report.Diagnostics, entity.Diagnostic{
				Category: entity.DiagOK,
				System:   "CommandCode",
				Target:   "~/.commandcode/",
				Details:  "Config directory exists",
			})

			// Check MCP config
			mcpPath := filepath.Join(ccConfigDir, "mcp.json")
			if uc.fsManager.Exists(mcpPath) {
				report.Diagnostics = append(report.Diagnostics, entity.Diagnostic{
					Category: entity.DiagOK,
					System:   "CommandCode",
					Target:   "MCP config",
					Details:  "mcp.json present",
				})
			} else {
				report.Diagnostics = append(report.Diagnostics, entity.Diagnostic{
					Category: entity.DiagWarning,
					System:   "CommandCode",
					Target:   "MCP config",
					Details:  "mcp.json not found",
					FixHint:  "Run 'envctl run shell' to provision CommandCode configs",
				})
			}

			// Check skills directory
			skillsDir := filepath.Join(ccConfigDir, "skills")
			if uc.fsManager.Exists(skillsDir) {
				entries, _ := os.ReadDir(skillsDir)
				report.Diagnostics = append(report.Diagnostics, entity.Diagnostic{
					Category: entity.DiagOK,
					System:   "CommandCode",
					Target:   "Skills",
					Details:  fmt.Sprintf("%d skills deployed", len(entries)),
				})
			} else {
				report.Diagnostics = append(report.Diagnostics, entity.Diagnostic{
					Category: entity.DiagWarning,
					System:   "CommandCode",
					Target:   "Skills",
					Details:  "Skills directory not found",
					FixHint:  "Run 'envctl run skills' to deploy skills",
				})
			}
		} else {
			report.Diagnostics = append(report.Diagnostics, entity.Diagnostic{
				Category: entity.DiagInfo,
				System:   "CommandCode",
				Target:   "~/.commandcode/",
				Details:  "Config directory not yet created (CommandCode not provisioned)",
			})
		}
	}
```

- [ ] **Step 2: Verify build and vet**

Run: `go build ./... && go vet ./...`
Expected: clean

- [ ] **Step 3: Commit**

```bash
git add internal/usecase/doctor_audit.go
git commit -m "feat(doctor): add CommandCode health checks (CLI, config, MCP, skills)"
```

---

## Task 7: Add CommandCode cleanup use case

**Files:**
- Create: `internal/usecase/cleanup_commandcode.go`
- Modify: `internal/ui/cli/root.go` (add to AppContext)
- Modify: `internal/ui/cli/run.go:318-352` (`runCleanup`)

**Interfaces:**
- Consumes: `repository.FileSystemManager` (existing)
- Produces: `CleanupCommandCodeUseCase` integrated into cleanup pipeline

- [ ] **Step 1: Create `cleanup_commandcode.go`**

```go
package usecase

import (
	"context"
	"os"
	"path/filepath"

	"github.com/eajdias/envctl/internal/domain/repository"
)

// CleanupCommandCodeUseCase prunes CommandCode storage accumulation:
// stale config variants and oversized artifacts.
type CleanupCommandCodeUseCase struct {
	fsManager repository.FileSystemManager
	logger    repository.Logger
}

func NewCleanupCommandCodeUseCase(
	fsManager repository.FileSystemManager,
	logger repository.Logger,
) *CleanupCommandCodeUseCase {
	return &CleanupCommandCodeUseCase{
		fsManager: fsManager,
		logger:    logger,
	}
}

func (uc *CleanupCommandCodeUseCase) Execute(ctx context.Context) (*CleanupResult, error) {
	result := &CleanupResult{}

	homeDir, _ := uc.fsManager.ExpandUserPath("~")
	ccDir := filepath.Join(homeDir, ".commandcode")

	if !uc.fsManager.Exists(ccDir) {
		return result, nil
	}

	// Remove stale settings.jsonc (standardized on settings.json)
	stale := filepath.Join(ccDir, "settings.jsonc")
	if uc.fsManager.Exists(stale) {
		if info, err := os.Stat(stale); err == nil {
			os.Remove(stale)
			result.RemovedFiles = append(result.RemovedFiles, stale)
			result.FreedBytes += info.Size()
			if uc.logger != nil {
				uc.logger.Info("[CLEANUP] removed stale CommandCode config %s", stale)
			}
		}
	}

	return result, nil
}
```

- [ ] **Step 2: Add to `AppContext` in `root.go`**

Add field at `root.go:45` (after `CleanupOpenCodeUC`):
```go
CleanupCommandCodeUC *usecase.CleanupCommandCodeUseCase
```

Add initialization at `root.go:118` (after `CleanupOpenCodeUC` initialization):
```go
CleanupCommandCodeUC: usecase.NewCleanupCommandCodeUseCase(fsManager, fileLogger),
```

- [ ] **Step 3: Update `runCleanup()` in `run.go`**

Add CommandCode cleanup after the TempHygiene block at `run.go:331-341`. Insert before the `len(removed) == 0` check:

```go
	if appCtx.CleanupCommandCodeUC != nil {
		ccReport, ccErr := appCtx.CleanupCommandCodeUC.Execute(ctx)
		if ccErr == nil && len(ccReport.RemovedFiles) > 0 {
			removed = append(removed, ccReport.RemovedFiles...)
			freed += ccReport.FreedBytes
		}
	}
```

Also update the spinner label at `run.go:319` from:
```go
spinner, _ := pterm.DefaultSpinner.Start("Cleaning OpenCode storage accumulation...")
```
to:
```go
spinner, _ := pterm.DefaultSpinner.Start("Cleaning OpenCode + CommandCode storage accumulation...")
```

- [ ] **Step 4: Verify build**

Run: `go build ./... && go vet ./...`
Expected: clean

- [ ] **Step 5: Commit**

```bash
git add internal/usecase/cleanup_commandcode.go internal/ui/cli/root.go internal/ui/cli/run.go
git commit -m "feat(cleanup): add CommandCode stale artifact cleanup"
```

---

## Task 8: Update version command and root description

**Files:**
- Modify: `internal/ui/cli/root.go:53-54` (root command description)
- Modify: `internal/ui/cli/root.go:144` (version command)

**Interfaces:**
- Consumes: none
- Produces: CLI reflects dual-agent support

- [ ] **Step 1: Update root command description**

Change `root.go:53` from:
```go
Short: "envctl: Universal Development Environment Provisioner (Windows 11 PRO & Ubuntu Linux / OpenCode)",
```
to:
```go
Short: "envctl: Universal Development Environment Provisioner (Windows 11 PRO & Ubuntu Linux / OpenCode + CommandCode)",
```

Change `root.go:54` from:
```go
Long:  `envctl is an idempotent, Clean Architecture CLI tool designed to provision, audit, and synchronize your development environments across Windows 11 PRO workstations and Ubuntu Linux VPS servers.`,
```
to:
```go
Long:  `envctl is an idempotent, Clean Architecture CLI tool designed to provision, audit, and synchronize your development environments across Windows 11 PRO workstations and Ubuntu Linux VPS servers. Supports both OpenCode and CommandCode AI agents.`,
```

- [ ] **Step 2: Update version command**

Change `root.go:144` from:
```go
PrintInfo(fmt.Sprintf("envctl %s (Windows 11 PRO & Ubuntu Linux / OpenCode Ecosystem)", appVersion))
```
to:
```go
PrintInfo(fmt.Sprintf("envctl %s (Windows 11 PRO & Ubuntu Linux / OpenCode + CommandCode)", appVersion))
```

- [ ] **Step 3: Verify build**

Run: `go build ./...`
Expected: clean

- [ ] **Step 4: Commit**

```bash
git add internal/ui/cli/root.go
git commit -m "feat(cli): update descriptions to reflect dual-agent support"
```

---

## Task 9: Full build verification and integration test

**Files:**
- None (verification only)

**Interfaces:**
- Consumes: all previous tasks
- Produces: clean build, vet, and basic smoke test

- [ ] **Step 1: Full build**

Run: `go build -o envctl-test.exe ./cmd/envctl/`
Expected: binary produced without errors

- [ ] **Step 2: Vet**

Run: `go vet ./...`
Expected: no warnings

- [ ] **Step 3: Smoke test — version**

Run: `./envctl-test.exe version`
Expected: output shows "OpenCode + CommandCode"

- [ ] **Step 4: Smoke test — doctor (dry run)**

Run: `./envctl-test.exe doctor`
Expected: audit runs, includes CommandCode checks

- [ ] **Step 5: Clean up test binary**

Run: `Remove-Item envctl-test.exe`

- [ ] **Step 6: Final commit if any fixes needed**

```bash
git add -A
git commit -m "fix: address review findings from integration verification"
```

---

## Self-Review Checklist

1. **Spec coverage:** All CommandCode config locations covered (settings, MCP, agents, AGENTS.md, skills, memory). Package install covered. Doctor checks covered. Cleanup covered. Bootstrap covered. **LSP: correctly excluded** — CommandCode's LSP is IDE-dependent (`get_diagnostics` only works with VS Code/Cursor/Windsurf connected). Terminal-only usage falls back to shell commands. No envctl changes needed.

2. **Placeholder scan:** No TBD/TODO/placeholders in any task. All code blocks are complete.

3. **Type consistency:** `ProvisionSkillsUC.Execute(ctx, targetDir)` signature unchanged. `CleanupResult` reused for CommandCode cleanup. `Diagnostic` struct reused for doctor checks. All new use cases follow existing patterns.

4. **Idempotency:** All provisioning uses existing idempotent mechanisms (byte-for-byte comparison, atomic backup, seed-if-missing). CommandCode entries are additions, not mutations.

5. **Backwards compatibility:** No existing behavior changed. OpenCode provisioning is untouched. CommandCode is purely additive. `envctl run all` works the same, just deploys to additional locations.
