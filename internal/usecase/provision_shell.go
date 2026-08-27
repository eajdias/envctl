package usecase

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/eajdias/envctl/internal/domain/entity"
	"github.com/eajdias/envctl/internal/domain/repository"
)

type ProvisionShellUseCase struct {
	manifestRepo repository.ManifestRepository
	fsManager    repository.FileSystemManager
	envManager   repository.WindowsEnvManager
	gitManager   repository.GitManager
	embeddedFS   fs.FS
	logger       repository.Logger
}

func NewProvisionShellUseCase(
	manifestRepo repository.ManifestRepository,
	fsManager repository.FileSystemManager,
	envManager repository.WindowsEnvManager,
	gitManager repository.GitManager,
	embeddedFS fs.FS,
	logger repository.Logger,
) *ProvisionShellUseCase {
	return &ProvisionShellUseCase{
		manifestRepo: manifestRepo,
		fsManager:    fsManager,
		envManager:   envManager,
		gitManager:   gitManager,
		embeddedFS:   embeddedFS,
		logger:       logger,
	}
}

type ProvisionShellResult struct {
	EnvDiagnostics    []entity.Diagnostic
	GitDiagnostics    []entity.Diagnostic
	CreatedBackups    map[string]string // destPath -> backupPath
	ConfigDiagnostics []entity.Diagnostic
	RestrictedDirs    []string
}

func (uc *ProvisionShellUseCase) Execute(ctx context.Context) (*ProvisionShellResult, error) {
	if uc.logger != nil {
		uc.logger.Info("Starting shell, environment, git, and configs provisioning")
	}

	result := &ProvisionShellResult{
		CreatedBackups: make(map[string]string),
	}

	// 1. Environment Variables
	envVars, err := uc.manifestRepo.LoadEnvVars()
	if err == nil && len(envVars) > 0 {
		diags, _ := uc.envManager.EnsureEnvVars(ctx, envVars)
		result.EnvDiagnostics = diags
		for _, d := range diags {
			if uc.logger != nil {
				uc.logger.LogIdempotency("Environment", d.Target, d.Category == entity.DiagOK, d.Details)
			}
		}
	}

	// 2. Git Performance Configurations
	gitConfigs, err := uc.manifestRepo.LoadGitConfigs()
	if err == nil && len(gitConfigs) > 0 {
		var applicable []entity.GitConfig
		for _, gc := range gitConfigs {
			if gc.OS != "" && gc.OS != runtime.GOOS {
				continue
			}
			applicable = append(applicable, gc)
		}
		diags, _ := uc.gitManager.EnsureGlobalConfigs(ctx, applicable)
		result.GitDiagnostics = diags
		for _, d := range diags {
			if uc.logger != nil {
				uc.logger.LogIdempotency("Git", d.Target, d.Category == entity.DiagOK, d.Details)
			}
		}
	}

	// 3. Restricted Directories (SSH Keys, Secrets, OpenCode, Projects)
	dirs, dirsErr := uc.manifestRepo.LoadDirectories()
	if dirsErr != nil || len(dirs) == 0 {
		dirs = []entity.RestrictedDir{
			{Path: "~/Documents/SSH-keys", StrictACL: true, Description: "Restricted directory for VPS and server SSH private keys"},
			{Path: "~/.ssh-manager", StrictACL: true, Description: "Restricted directory for SSH Manager state and configs"},
			{Path: "~/.ssh", StrictACL: true, Description: "Standard user SSH configuration directory"},
			{Path: "~/.config/opencode/skills", Description: "OpenCode agent skills directory"},
			{Path: "~/projetos/git-privado", Description: "Directory for private git projects"},
			{Path: "~/projetos/git-publico", Description: "Directory for public open-source git projects"},
		}
		if runtime.GOOS == "windows" {
			dirs[4].Path = "C:/projetos/git-privado"
			dirs[5].Path = "C:/projetos/git-publico"
		}
	}

	for _, dir := range dirs {
		if dir.OS != "" && dir.OS != runtime.GOOS {
			continue
		}

		dirErr := uc.fsManager.EnsureDirectory(dir.Path, 0700)
		if dirErr != nil && runtime.GOOS == "linux" {
			// Root-level directories (e.g. /temp) require sudo; retry via
			// NOPASSWD sudo and leave a world-writable sticky scratch folder.
			if _, serr := exec.Command("sudo", "-n", "mkdir", "-p", dir.Path).CombinedOutput(); serr == nil {
				_ = exec.Command("sudo", "-n", "chmod", "1777", dir.Path).Run()
				dirErr = nil
				if uc.logger != nil {
					uc.logger.Info("Created root-level directory '%s' via sudo", dir.Path)
				}
			}
		}
		if dirErr != nil {
			if uc.logger != nil {
				uc.logger.Error("Failed to ensure directory '%s': %v", dir.Path, dirErr)
			}
			result.ConfigDiagnostics = append(result.ConfigDiagnostics, entity.Diagnostic{
				Category: entity.DiagError,
				System:   "Directory",
				Target:   dir.Path,
				Details:  fmt.Sprintf("Failed to create directory: %v", dirErr),
			})
			continue
		}

		if dir.StrictACL {
			if err := uc.fsManager.SetStrictWindowsACL(dir.Path); err != nil {
				if uc.logger != nil {
					uc.logger.Warn("Could not apply strict ACLs on '%s': %v", dir.Path, err)
				}
			} else {
				if uc.logger != nil {
					uc.logger.Info("Applied strict ACLs (current user only) to '%s'", dir.Path)
				}
			}
			result.RestrictedDirs = append(result.RestrictedDirs, dir.Path)
		}

		result.ConfigDiagnostics = append(result.ConfigDiagnostics, entity.Diagnostic{
			Category: entity.DiagOK,
			System:   "Directory",
			Target:   dir.Path,
			Details:  "Directory verified and permissions secured",
		})
	}

	// 4. Configuration Files (.bashrc, .bash_profile, nsswitch.conf, opencode.jsonc, AGENTS.md)
	configFiles, err := uc.manifestRepo.LoadConfigFiles()
	if err != nil {
		if uc.logger != nil {
			uc.logger.Error("Failed to load config files manifest: %v", err)
		}
		return result, fmt.Errorf("failed to load config files manifest: %w", err)
	}

	for _, cf := range configFiles {
		if cf.OS != "" && cf.OS != runtime.GOOS {
			continue
		}

		// Read source from disk if exists, otherwise embedded FS
		var content []byte
		var readErr error

		// Try disk relative to configs/
		content, readErr = uc.fsManager.ReadFile(cf.Source)
		if readErr != nil {
			// Read from embedded FS
			embeddedPath := filepath.ToSlash(cf.Source)
			content, readErr = fs.ReadFile(uc.embeddedFS, embeddedPath)
		}

		if readErr != nil {
			if uc.logger != nil {
				uc.logger.Error("Source file missing for '%s' (%s): %v", cf.Destination, cf.Source, readErr)
			}
			result.ConfigDiagnostics = append(result.ConfigDiagnostics, entity.Diagnostic{
				Category: entity.DiagError,
				System:   "ConfigFile",
				Target:   cf.Destination,
				Details:  fmt.Sprintf("Source file missing (%s): %v", cf.Source, readErr),
			})
			continue
		}

		// Write with atomic backup; sensitive files get strict permissions.
		perm := os.FileMode(0644)
		if cf.StrictACL {
			perm = 0600
		}

		// Seed mode: write the baseline only when the destination does not
		// exist yet (e.g. agent memory templates — per-machine additions must
		// never be overwritten by provisioning).
		if cf.SeedIfMissing && uc.fsManager.Exists(cf.Destination) {
			if uc.logger != nil {
				uc.logger.LogIdempotency("ConfigFile", cf.Destination, true, "seed baseline skipped (destination already exists with per-machine content)")
			}
			result.ConfigDiagnostics = append(result.ConfigDiagnostics, entity.Diagnostic{
				Category: entity.DiagOK,
				System:   "ConfigFile",
				Target:   cf.Destination,
				Details:  "Seed baseline present (destination already exists — per-machine content preserved)",
			})
			continue
		}

		backupPath, writeErr := uc.fsManager.WriteWithBackup(cf.Destination, content, perm)
		if writeErr != nil {
			if uc.logger != nil {
				uc.logger.Error("Failed to write config file '%s': %v", cf.Destination, writeErr)
			}
			result.ConfigDiagnostics = append(result.ConfigDiagnostics, entity.Diagnostic{
				Category: entity.DiagError,
				System:   "ConfigFile",
				Target:   cf.Destination,
				Details:  fmt.Sprintf("Failed to write config: %v", writeErr),
			})
		} else {
			if cf.StrictACL {
				if err := uc.fsManager.SetStrictWindowsACL(cf.Destination); err != nil {
					if uc.logger != nil {
						uc.logger.Warn("Could not apply strict ACLs to '%s': %v", cf.Destination, err)
					}
				}
			}

			detail := "Config written successfully"
			if backupPath != "" {
				result.CreatedBackups[cf.Destination] = backupPath
				detail = fmt.Sprintf("Updated (Backup saved to %s)", filepath.Base(backupPath))
				if uc.logger != nil {
					uc.logger.LogIdempotency("ConfigFile", cf.Destination, false, fmt.Sprintf("content updated, backup created at %s", backupPath))
				}
			} else {
				detail = "Already up to date"
				if uc.logger != nil {
					uc.logger.LogIdempotency("ConfigFile", cf.Destination, true, "content byte-for-byte identical, skipped backup/write")
				}
			}

			result.ConfigDiagnostics = append(result.ConfigDiagnostics, entity.Diagnostic{
				Category: entity.DiagOK,
				System:   "ConfigFile",
				Target:   cf.Destination,
				Details:  detail,
			})
		}
	}

	// 4.5. Cleanup stale files that conflict with the current provisioning
	// (e.g. the legacy opencode.jsonc after standardizing on opencode.json).
	cleanupItems, cleanupErr := uc.manifestRepo.LoadCleanupItems()
	if cleanupErr == nil {
		for _, item := range cleanupItems {
			if item.OS != "" && item.OS != runtime.GOOS {
				continue
			}
			expandedPath, err := uc.fsManager.ExpandUserPath(item.Path)
			if err != nil {
				continue
			}
			if !uc.fsManager.Exists(expandedPath) {
				continue
			}
			if err := os.Remove(expandedPath); err != nil {
				if uc.logger != nil {
					uc.logger.Warn("Failed to remove stale file '%s': %v", expandedPath, err)
				}
			} else {
				if uc.logger != nil {
					uc.logger.Info("Removed stale file: %s (%s)", expandedPath, item.Description)
				}
				result.ConfigDiagnostics = append(result.ConfigDiagnostics, entity.Diagnostic{
					Category: entity.DiagOK,
					System:   "Cleanup",
					Target:   expandedPath,
					Details:  "Stale file removed: " + item.Description,
				})
			}
		}
	}

	// 5. OpenCode Plugins npm dependencies installation
	opencodeConfigDir, _ := uc.fsManager.ExpandUserPath("~/.config/opencode")
	packageJsonPath := filepath.Join(opencodeConfigDir, "package.json")
	if uc.fsManager.Exists(packageJsonPath) {
		nodeModulesPath := filepath.Join(opencodeConfigDir, "node_modules")
		if !uc.fsManager.Exists(nodeModulesPath) {
			if uc.logger != nil {
				uc.logger.Info("Installing OpenCode plugin dependencies in ~/.config/opencode via npm")
			}
			cmd := exec.CommandContext(ctx, "npm", "install", "--no-audit", "--no-fund")
			cmd.Dir = opencodeConfigDir
			out, err := cmd.CombinedOutput()
			if err != nil {
				if uc.logger != nil {
					uc.logger.Warn("Failed to install OpenCode plugins via npm: %s (%v)", string(out), err)
				}
				result.ConfigDiagnostics = append(result.ConfigDiagnostics, entity.Diagnostic{
					Category: entity.DiagWarning,
					System:   "OpenCodePlugins",
					Target:   packageJsonPath,
					Details:  fmt.Sprintf("npm install warning: %v", err),
					FixHint:  "Run 'npm install' manually inside ~/.config/opencode",
				})
			} else {
				if uc.logger != nil {
					uc.logger.Info("Successfully installed OpenCode plugins in ~/.config/opencode")
					uc.logger.LogIdempotency("OpenCodePlugins", packageJsonPath, false, "Installed plugins successfully")
				}
				result.ConfigDiagnostics = append(result.ConfigDiagnostics, entity.Diagnostic{
					Category: entity.DiagOK,
					System:   "OpenCodePlugins",
					Target:   packageJsonPath,
					Details:  "OpenCode plugins installed (@opencode-ai/plugin)",
				})
			}
		} else {
			if uc.logger != nil {
				uc.logger.LogIdempotency("OpenCodePlugins", packageJsonPath, true, "node_modules already exists in ~/.config/opencode")
			}
		}
	}

	// 6. User Home npm dependencies & Playwright Browser Installation
	userHomeDir, _ := uc.fsManager.ExpandUserPath("~")
	userPackageJsonPath := filepath.Join(userHomeDir, "package.json")
	if uc.fsManager.Exists(userPackageJsonPath) {
		userNodeModulesPath := filepath.Join(userHomeDir, "node_modules")
		playwrightInstalled := uc.fsManager.Exists(filepath.Join(userNodeModulesPath, "playwright"))
		pkgJsonInfo, _ := os.Stat(userPackageJsonPath)
		nmInfo, _ := os.Stat(userNodeModulesPath)
		depsOutdated := pkgJsonInfo != nil && nmInfo != nil && pkgJsonInfo.ModTime().After(nmInfo.ModTime())
		if !playwrightInstalled || depsOutdated {
			if uc.logger != nil {
				uc.logger.Info("Installing user root dependencies (Playwright) in %s via npm", userHomeDir)
			}
			cmd := exec.CommandContext(ctx, "npm", "install", "--no-audit", "--no-fund")
			cmd.Dir = userHomeDir
			out, err := cmd.CombinedOutput()
			if err != nil {
				if uc.logger != nil {
					uc.logger.Warn("Failed to install user root dependencies via npm: %s (%v)", string(out), err)
				}
				result.ConfigDiagnostics = append(result.ConfigDiagnostics, entity.Diagnostic{
					Category: entity.DiagWarning,
					System:   "PlaywrightRuntime",
					Target:   userPackageJsonPath,
					Details:  fmt.Sprintf("npm install warning: %v", err),
					FixHint:  "Run 'npm install' in user home directory",
				})
			} else {
				if uc.logger != nil {
					uc.logger.Info("Successfully installed user root npm dependencies")
					uc.logger.LogIdempotency("PlaywrightRuntime", userPackageJsonPath, false, "Installed user root dependencies")
				}
				result.ConfigDiagnostics = append(result.ConfigDiagnostics, entity.Diagnostic{
					Category: entity.DiagOK,
					System:   "PlaywrightRuntime",
					Target:   userPackageJsonPath,
					Details:  "User root npm dependencies installed (playwright)",
				})
			}
		} else {
			if uc.logger != nil {
				uc.logger.LogIdempotency("PlaywrightRuntime", userPackageJsonPath, true, "playwright already installed in user node_modules")
			}
			result.ConfigDiagnostics = append(result.ConfigDiagnostics, entity.Diagnostic{
				Category: entity.DiagOK,
				System:   "PlaywrightRuntime",
				Target:   userPackageJsonPath,
				Details:  "Playwright Node.js API runtime verified in user root",
			})
		}

		// Ensure Playwright Chromium browser binaries are installed (only if missing).
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
		if chromiumFound {
			if uc.logger != nil {
				uc.logger.LogIdempotency("PlaywrightBrowser", "chromium", true, "Chromium already installed in "+msPlaywrightDir)
			}
			result.ConfigDiagnostics = append(result.ConfigDiagnostics, entity.Diagnostic{
				Category: entity.DiagOK,
				System:   "PlaywrightBrowser",
				Target:   "chromium",
				Details:  "Playwright Chromium browser binary already installed",
			})
		} else {
			if uc.logger != nil {
				uc.logger.Info("Ensuring Playwright Chromium browser binary is installed")
			}
			cmdBrowser := exec.CommandContext(ctx, "npx", "playwright", "install", "chromium")
			cmdBrowser.Dir = userHomeDir
			outBrowser, errBrowser := cmdBrowser.CombinedOutput()
			if errBrowser != nil {
				if uc.logger != nil {
					uc.logger.Warn("Failed to install Playwright Chromium: %s (%v)", string(outBrowser), errBrowser)
				}
				result.ConfigDiagnostics = append(result.ConfigDiagnostics, entity.Diagnostic{
					Category: entity.DiagWarning,
					System:   "PlaywrightBrowser",
					Target:   "chromium",
					Details:  fmt.Sprintf("playwright install chromium warning: %v", errBrowser),
					FixHint:  "Run 'npx playwright install chromium' in user home directory",
				})
			} else {
				if uc.logger != nil {
					uc.logger.Info("Playwright Chromium browser verified/installed successfully")
					uc.logger.LogIdempotency("PlaywrightBrowser", "chromium", true, "Playwright Chromium browser ready")
				}
				result.ConfigDiagnostics = append(result.ConfigDiagnostics, entity.Diagnostic{
					Category: entity.DiagOK,
					System:   "PlaywrightBrowser",
					Target:   "chromium",
					Details:  "Playwright Chromium browser binary installed and verified",
				})

				// On Linux, install the system libraries Chromium needs to run
				// headless (libnss3, libatk, ...). Best-effort with sudo -n.
				if runtime.GOOS == "linux" {
					depsCmd := exec.CommandContext(ctx, "sudo", "-n", "env", "PATH="+os.Getenv("PATH"), "npx", "playwright", "install-deps", "chromium")
					depsCmd.Dir = userHomeDir
					outDeps, errDeps := depsCmd.CombinedOutput()
					if errDeps != nil {
						if uc.logger != nil {
							uc.logger.Warn("Failed to install Playwright Chromium system deps (best-effort): %s (%v)", string(outDeps), errDeps)
						}
						result.ConfigDiagnostics = append(result.ConfigDiagnostics, entity.Diagnostic{
							Category: entity.DiagWarning,
							System:   "PlaywrightBrowser",
							Target:   "chromium system deps",
							Details:  fmt.Sprintf("playwright install-deps warning (best-effort): %v", errDeps),
							FixHint:  "Run 'sudo npx playwright install-deps chromium' in user home directory",
						})
					} else {
						if uc.logger != nil {
							uc.logger.Info("Playwright Chromium system dependencies installed")
						}
						result.ConfigDiagnostics = append(result.ConfigDiagnostics, entity.Diagnostic{
							Category: entity.DiagOK,
							System:   "PlaywrightBrowser",
							Target:   "chromium system deps",
							Details:  "Playwright Chromium system dependencies installed",
						})
					}
				}
			}
		}
	}

	return result, nil
}
