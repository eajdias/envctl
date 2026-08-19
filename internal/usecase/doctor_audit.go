package usecase

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/eajdias/win11-new/internal/domain/entity"
	"github.com/eajdias/win11-new/internal/domain/repository"
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
		val, err := uc.envManager.GetEnvVar(ev.Scope, ev.Name)
		if err != nil || val != ev.Value {
			addDiag(entity.Diagnostic{
				Category: entity.DiagWarning,
				System:   "Environment",
				Target:   ev.Name,
				Details:  fmt.Sprintf("Current: '%s' | Expected: '%s' (Scope: %s)", val, ev.Value, ev.Scope),
				FixHint:  fmt.Sprintf("run 'win11-new run shell' to apply %s=%s", ev.Name, ev.Value),
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
		if !uc.fsManager.Exists(cf.Destination) {
			addDiag(entity.Diagnostic{
				Category: entity.DiagError,
				System:   "ConfigFile",
				Target:   cf.Destination,
				Details:  "File missing on filesystem",
				FixHint:  "run 'win11-new run shell'",
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

	// 4. Audit Packages
	packages, _ := uc.manifestRepo.LoadPackages()
	for _, pkg := range packages {
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
				FixHint:  fmt.Sprintf("run 'win11-new run %s'", pkg.Type),
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

	// 5. Audit Skills
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
				FixHint:  "run 'win11-new run skills'",
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

	// 6. Audit LSPs
	lsps, _ := uc.manifestRepo.LoadLSPs()
	for _, lsp := range lsps {
		if lsp.CheckBinary != "" {
			if _, lookErr := exec.LookPath(lsp.CheckBinary); lookErr != nil {
				addDiag(entity.Diagnostic{
					Category: entity.DiagWarning,
					System:   "LSP",
					Target:   lsp.ServerName,
					Details:  fmt.Sprintf("Binary '%s' not found in PATH", lsp.CheckBinary),
					FixHint:  fmt.Sprintf("run 'win11-new run lsp' to install %s", lsp.InstallTarget),
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

	// 7. Audit Windows 11 Registry Tweaks, Features & Fonts
	if uc.tweaksManager != nil {
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
					FixHint:  "run 'win11-new run windows'",
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

	// 8. Audit Playwright Node API & Chromium Browser
	userHomeDir, _ := uc.fsManager.ExpandUserPath("~")
	playwrightModule := filepath.Join(userHomeDir, "node_modules", "playwright")
	if !uc.fsManager.Exists(playwrightModule) {
		addDiag(entity.Diagnostic{
			Category: entity.DiagWarning,
			System:   "Playwright",
			Target:   "playwright (node_modules)",
			Details:  "Playwright npm module not installed in user home",
			FixHint:  "run 'win11-new run shell'",
		})
	} else {
		addDiag(entity.Diagnostic{
			Category: entity.DiagOK,
			System:   "Playwright",
			Target:   "playwright (node_modules)",
			Details:  "Node.js API installed in user root",
		})
	}

	msPlaywrightDir, _ := uc.fsManager.ExpandUserPath("%LOCALAPPDATA%/ms-playwright")
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
			Details:  "Chromium browser binary not found in %LOCALAPPDATA%/ms-playwright",
			FixHint:  "run 'npx playwright install chromium'",
		})
	} else {
		addDiag(entity.Diagnostic{
			Category: entity.DiagOK,
			System:   "Playwright",
			Target:   "Chromium Browser",
			Details:  "Chromium binary verified in %LOCALAPPDATA%/ms-playwright",
		})
	}

	// 9. Audit Custom CLI Scripts (~/.local/bin)
	customScripts := []string{"pw-screenshot", "pw-eval"}
	for _, cs := range customScripts {
		scriptPath := filepath.Join(userHomeDir, ".local", "bin", cs)
		if !uc.fsManager.Exists(scriptPath) {
			addDiag(entity.Diagnostic{
				Category: entity.DiagWarning,
				System:   "CLI-Scripts",
				Target:   cs,
				Details:  fmt.Sprintf("Script not found at %s", scriptPath),
				FixHint:  "run 'win11-new run shell'",
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

	// 10. Audit Git Worktree Support
	if gitOut, err := exec.CommandContext(ctx, "git", "worktree", "list").CombinedOutput(); err != nil {
		addDiag(entity.Diagnostic{
			Category: entity.DiagWarning,
			System:   "Git",
			Target:   "git worktree",
			Details:  fmt.Sprintf("Worktree check failed: %v", err),
			FixHint:  "Ensure git is installed and updated",
		})
	} else {
		_ = gitOut
		addDiag(entity.Diagnostic{
			Category: entity.DiagOK,
			System:   "Git",
			Target:   "git worktree",
			Details:  "Worktree command supported and active",
		})
	}

	return report, nil
}
