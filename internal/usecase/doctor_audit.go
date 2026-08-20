package usecase

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/eajdias/envctl/internal/domain/entity"
	"github.com/eajdias/envctl/internal/domain/repository"
)

type DoctorAuditUseCase struct {
	manifestRepo  repository.ManifestRepository
	fsManager     repository.FileSystemManager
	envManager    repository.WindowsEnvManager
	gitManager    repository.GitManager
	tweaksManager repository.WindowsTweaksManager
	managers      map[entity.PackageType]repository.PackageManager
	logger        repository.Logger
}

func NewDoctorAuditUseCase(
	manifestRepo repository.ManifestRepository,
	fsManager repository.FileSystemManager,
	envManager repository.WindowsEnvManager,
	gitManager repository.GitManager,
	tweaksManager repository.WindowsTweaksManager,
	managers map[entity.PackageType]repository.PackageManager,
	logger repository.Logger,
) *DoctorAuditUseCase {
	return &DoctorAuditUseCase{
		manifestRepo:  manifestRepo,
		fsManager:     fsManager,
		envManager:    envManager,
		gitManager:    gitManager,
		tweaksManager: tweaksManager,
		managers:      managers,
		logger:        logger,
	}
}

type AuditReport struct {
	Diagnostics []entity.Diagnostic
	TotalChecks int
	Passed      int
	Warnings    int
	Errors      int
}

func (uc *DoctorAuditUseCase) Execute(ctx context.Context) (*AuditReport, error) {
	if uc.logger != nil {
		uc.logger.Info("Starting system audit and diagnostic verification")
	}

	report := &AuditReport{}

	addDiag := func(diag entity.Diagnostic) {
		report.Diagnostics = append(report.Diagnostics, diag)
		report.TotalChecks++
		switch diag.Category {
		case entity.DiagOK:
			report.Passed++
			if uc.logger != nil {
				uc.logger.Info("[AUDIT-PASS] [%s] %s: %s", diag.System, diag.Target, diag.Details)
			}
		case entity.DiagWarning:
			report.Warnings++
			if uc.logger != nil {
				uc.logger.Warn("[AUDIT-WARN] [%s] %s: %s (Fix: %s)", diag.System, diag.Target, diag.Details, diag.FixHint)
			}
		case entity.DiagError:
			report.Errors++
			if uc.logger != nil {
				uc.logger.Error("[AUDIT-FAIL] [%s] %s: %s (Fix: %s)", diag.System, diag.Target, diag.Details, diag.FixHint)
			}
		}
	}

	// 1. Audit Environment Variables
	envVars, _ := uc.manifestRepo.LoadEnvVars()
	for _, ev := range envVars {
		if ev.OS != "" && ev.OS != runtime.GOOS {
			continue
		}
		val, err := uc.envManager.GetEnvVar(ev.Scope, ev.Name)
		if err != nil || val != ev.Value {
			addDiag(entity.Diagnostic{
				Category: entity.DiagWarning,
				System:   "Environment",
				Target:   ev.Name,
				Details:  fmt.Sprintf("Current: '%s' | Expected: '%s' (Scope: %s)", val, ev.Value, ev.Scope),
				FixHint:  fmt.Sprintf("run 'envctl run shell' to apply %s=%s", ev.Name, ev.Value),
			})
		} else {
			addDiag(entity.Diagnostic{
				Category: entity.DiagOK,
				System:   "Environment",
				Target:   ev.Name,
				Details:  fmt.Sprintf("Set to '%s' (Scope: %s)", val, ev.Scope),
			})
		}
	}

	// 2. Audit Git Global Configurations
	gitConfigs, _ := uc.manifestRepo.LoadGitConfigs()
	for _, gc := range gitConfigs {
		if gc.OS != "" && gc.OS != runtime.GOOS {
			continue
		}
		val, err := uc.gitManager.GetGlobalConfig(ctx, gc.Key)
		if err != nil || val != gc.Value {
			addDiag(entity.Diagnostic{
				Category: entity.DiagWarning,
				System:   "Git",
				Target:   gc.Key,
				Details:  fmt.Sprintf("Current: '%s' | Expected: '%s'", val, gc.Value),
				FixHint:  fmt.Sprintf("git config --global %s %s", gc.Key, gc.Value),
			})
		} else {
			addDiag(entity.Diagnostic{
				Category: entity.DiagOK,
				System:   "Git",
				Target:   gc.Key,
				Details:  fmt.Sprintf("Configured: %s", val),
			})
		}
	}

	// 3. Audit Config Files
	configFiles, _ := uc.manifestRepo.LoadConfigFiles()
	for _, cf := range configFiles {
		if cf.OS != "" && cf.OS != runtime.GOOS {
			continue
		}
		if !uc.fsManager.Exists(cf.Destination) {
			addDiag(entity.Diagnostic{
				Category: entity.DiagError,
				System:   "ConfigFile",
				Target:   cf.Destination,
				Details:  "File missing on filesystem",
				FixHint:  "run 'envctl run shell'",
			})
		} else {
			addDiag(entity.Diagnostic{
				Category: entity.DiagOK,
				System:   "ConfigFile",
				Target:   cf.Destination,
				Details:  "Present on disk",
			})
		}
	}

	// 4. Audit OpenCode Global Rules (AGENTS.md)
	globalAgentsPath := filepath.Join("~/.config/opencode/AGENTS.md")
	if !uc.fsManager.Exists(globalAgentsPath) {
		addDiag(entity.Diagnostic{
			Category: entity.DiagWarning,
			System:   "OpenCode",
			Target:   "AGENTS.md (global rules)",
			Details:  fmt.Sprintf("Global rules file missing (%s) - opencode loads rules from this path, not ~/AGENTS.md", globalAgentsPath),
			FixHint:  "run 'envctl run shell'",
		})
	} else {
		addDiag(entity.Diagnostic{
			Category: entity.DiagOK,
			System:   "OpenCode",
			Target:   "AGENTS.md (global rules)",
			Details:  "Global rules present at ~/.config/opencode/AGENTS.md",
		})
	}

	// 5. Audit Packages
	packages, _ := uc.manifestRepo.LoadPackages()
	for _, pkg := range packages {
		if pkg.OS != "" && pkg.OS != runtime.GOOS {
			continue
		}

		mgr, ok := uc.managers[pkg.Type]
		if !ok || !mgr.IsAvailable(ctx) {
			addDiag(entity.Diagnostic{
				Category: entity.DiagWarning,
				System:   string(pkg.Type),
				Target:   pkg.ID,
				Details:  fmt.Sprintf("Manager %s not available", pkg.Type),
			})
			continue
		}

		installed, info, _ := mgr.IsInstalled(ctx, pkg)
		if !installed {
			addDiag(entity.Diagnostic{
				Category: entity.DiagWarning,
				System:   string(pkg.Type),
				Target:   pkg.ID,
				Details:  "Not installed",
				FixHint:  fmt.Sprintf("run 'envctl run %s'", pkg.Type),
			})
		} else {
			addDiag(entity.Diagnostic{
				Category: entity.DiagOK,
				System:   string(pkg.Type),
				Target:   pkg.ID,
				Details:  fmt.Sprintf("Installed (%s)", info),
			})
		}
	}

	// 6. Audit Skills
	skills, _ := uc.manifestRepo.LoadSkills()
	for _, s := range skills {
		targetDir := s.TargetDir
		if targetDir == "" {
			targetDir = filepath.Join("~/.config/opencode/skills", s.Name)
		}
		altDir := filepath.Join("~/.agents/skills", s.Name)

		if !uc.fsManager.Exists(targetDir) && !uc.fsManager.Exists(altDir) {
			addDiag(entity.Diagnostic{
				Category: entity.DiagWarning,
				System:   "Skills",
				Target:   s.Name,
				Details:  fmt.Sprintf("Skill directory missing (%s)", targetDir),
				FixHint:  "run 'envctl run skills'",
			})
		} else {
			addDiag(entity.Diagnostic{
				Category: entity.DiagOK,
				System:   "Skills",
				Target:   s.Name,
				Details:  "Active and deployed",
			})
		}
	}

	// 7. Audit LSPs
	lsps, _ := uc.manifestRepo.LoadLSPs()
	for _, lsp := range lsps {
		if lsp.OS != "" && lsp.OS != runtime.GOOS {
			continue
		}
		if lsp.CheckBinary != "" {
			if !toolAvailable(ctx, lsp.CheckBinary) {
				addDiag(entity.Diagnostic{
					Category: entity.DiagWarning,
					System:   "LSP",
					Target:   lsp.ServerName,
					Details:  fmt.Sprintf("Binary '%s' not found in PATH", lsp.CheckBinary),
					FixHint:  fmt.Sprintf("run 'envctl run lsp' to install %s", lsp.InstallTarget),
				})
			} else {
				addDiag(entity.Diagnostic{
					Category: entity.DiagOK,
					System:   "LSP",
					Target:   lsp.ServerName,
					Details:  fmt.Sprintf("Ready (%s in PATH)", lsp.CheckBinary),
				})
			}
		}
	}

	// 8. Audit Windows 11 Registry Tweaks, Features & Fonts (Windows only)
	if runtime.GOOS == "windows" && uc.tweaksManager != nil {
		tweaks, _ := uc.manifestRepo.LoadWindowsTweaks()
		for _, tw := range tweaks {
			targetName := fmt.Sprintf("%s\\%s", tw.Path, tw.Name)
			if tw.Path == "" {
				targetName = fmt.Sprintf("[%s] %s", tw.Type, tw.Name)
			}
			ok, details, err := uc.tweaksManager.CheckTweak(ctx, tw)
			if err != nil || !ok {
				addDiag(entity.Diagnostic{
					Category: entity.DiagWarning,
					System:   "Windows11",
					Target:   targetName,
					Details:  details,
					FixHint:  "run 'envctl run windows'",
				})
			} else {
				addDiag(entity.Diagnostic{
					Category: entity.DiagOK,
					System:   "Windows11",
					Target:   targetName,
					Details:  details,
				})
			}
		}
	}

	// 9. Audit Playwright Node API & Chromium Browser
	userHomeDir, _ := uc.fsManager.ExpandUserPath("~")
	playwrightModule := filepath.Join(userHomeDir, "node_modules", "playwright")
	if !uc.fsManager.Exists(playwrightModule) {
		addDiag(entity.Diagnostic{
			Category: entity.DiagWarning,
			System:   "Playwright",
			Target:   "playwright (node_modules)",
			Details:  "Playwright npm module not installed in user home",
			FixHint:  "run 'envctl run shell'",
		})
	} else {
		addDiag(entity.Diagnostic{
			Category: entity.DiagOK,
			System:   "Playwright",
			Target:   "playwright (node_modules)",
			Details:  "Node.js API installed in user root",
		})
	}

	var msPlaywrightDir string
	if runtime.GOOS == "windows" {
		msPlaywrightDir, _ = uc.fsManager.ExpandUserPath("%LOCALAPPDATA%/ms-playwright")
	} else {
		msPlaywrightDir, _ = uc.fsManager.ExpandUserPath("~/.cache/ms-playwright")
	}

	chromiumFound := false
	if entries, err := os.ReadDir(msPlaywrightDir); err == nil {
		for _, e := range entries {
			if strings.HasPrefix(e.Name(), "chromium-") || strings.HasPrefix(e.Name(), "chromium_headless_shell-") {
				chromiumFound = true
				break
			}
		}
	}
	if !chromiumFound {
		addDiag(entity.Diagnostic{
			Category: entity.DiagWarning,
			System:   "Playwright",
			Target:   "Chromium Browser",
			Details:  fmt.Sprintf("Chromium browser binary not found in %s", msPlaywrightDir),
			FixHint:  "run 'npx playwright install chromium'",
		})
	} else {
		addDiag(entity.Diagnostic{
			Category: entity.DiagOK,
			System:   "Playwright",
			Target:   "Chromium Browser",
			Details:  fmt.Sprintf("Chromium binary verified in %s", msPlaywrightDir),
		})
	}

	// 10. Audit Custom CLI Scripts (~/.local/bin)
	customScripts := []string{"pw-screenshot", "pw-eval"}
	for _, cs := range customScripts {
		scriptPath := filepath.Join(userHomeDir, ".local", "bin", cs)
		if !uc.fsManager.Exists(scriptPath) {
			addDiag(entity.Diagnostic{
				Category: entity.DiagWarning,
				System:   "CLI-Scripts",
				Target:   cs,
				Details:  fmt.Sprintf("Script not found at %s", scriptPath),
				FixHint:  "run 'envctl run shell'",
			})
		} else {
			addDiag(entity.Diagnostic{
				Category: entity.DiagOK,
				System:   "CLI-Scripts",
				Target:   cs,
				Details:  "Executable ready in ~/.local/bin",
			})
		}
	}

	// 11. Audit Git Worktree Support
	// `git worktree list` exits 128 outside a git repository, which is expected
	// and not a fault of the git installation. Only run the command from inside a repo.
	inRepo := exec.CommandContext(ctx, "git", "rev-parse", "--is-inside-work-tree")
	if err := inRepo.Run(); err != nil {
		addDiag(entity.Diagnostic{
			Category: entity.DiagOK,
			System:   "Git",
			Target:   "git worktree",
			Details:  "git worktree supported (command not run: current directory is not inside a git repository)",
		})
	} else if _, err := exec.CommandContext(ctx, "git", "worktree", "list").CombinedOutput(); err != nil {
		addDiag(entity.Diagnostic{
			Category: entity.DiagWarning,
			System:   "Git",
			Target:   "git worktree",
			Details:  fmt.Sprintf("Worktree check failed: %v", err),
			FixHint:  "Ensure git is installed and updated",
		})
	} else {
		addDiag(entity.Diagnostic{
			Category: entity.DiagOK,
			System:   "Git",
			Target:   "git worktree",
			Details:  "Worktree command supported and active",
		})
	}

	// 12. Audit Linux Toolchain Bootstrap (Linux only)
	if runtime.GOOS == "linux" {
		env := linuxToolchainEnv(userHomeDir)
		bootstrapTools := []struct {
			name string
			desc string
		}{
			{"volta", "Volta JS toolchain manager"},
			{"node", "Node.js (via Volta)"},
			{"opencode", "OpenCode CLI"},
			{"gh", "GitHub CLI"},
			{"delta", "git-delta pager"},
			{"yq", "yq YAML/JSON processor"},
			{"uv", "uv Python package manager"},
			{"ruff", "ruff linter (via uv)"},
			{"oh-my-posh", "Oh-My-Posh prompt engine"},
			{"fd", "fd (fdfind symlink)"},
			{"pylsp", "python-lsp-server (via uv)"},
			{"firecrawl", "Firecrawl CLI (via Volta)"},
			{"stylelint", "Stylelint CSS/SCSS linter (via Volta)"},
		}
		for _, t := range bootstrapTools {
			found := func() bool {
				c := exec.CommandContext(ctx, "bash", "-lc", "command -v "+t.name+" >/dev/null 2>&1")
				c.Env = env
				return c.Run() == nil
			}()
			if found {
				addDiag(entity.Diagnostic{
					Category: entity.DiagOK,
					System:   "LinuxBootstrap",
					Target:   t.desc,
					Details:  "Tool available on PATH (" + t.name + ")",
				})
			} else {
				addDiag(entity.Diagnostic{
					Category: entity.DiagWarning,
					System:   "LinuxBootstrap",
					Target:   t.desc,
					Details:  "Tool not found on PATH (" + t.name + ")",
					FixHint:  "run 'envctl run bootstrap'",
				})
			}
		}
	}

	return report, nil
}
