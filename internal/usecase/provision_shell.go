package usecase

import (
	"context"
	"fmt"
	"io/fs"
	"os/exec"
	"path/filepath"

	"github.com/eajdias/win11-new/internal/domain/entity"
	"github.com/eajdias/win11-new/internal/domain/repository"
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
		diags, _ := uc.gitManager.EnsureGlobalConfigs(ctx, gitConfigs)
		result.GitDiagnostics = diags
		for _, d := range diags {
			if uc.logger != nil {
				uc.logger.LogIdempotency("Git", d.Target, d.Category == entity.DiagOK, d.Details)
			}
		}
	}

	// 3. Restricted Directories (SSH Keys, Secrets)
	restrictedDirs := []string{
		"~/Documents/SSH-keys",
		"~/.ssh-manager",
		"~/.ssh",
		"~/.config/opencode/skills",
		"C:/projetos/git-privado",
		"C:/projetos/git-publico",
	}

	for _, dir := range restrictedDirs {
		if err := uc.fsManager.EnsureDirectory(dir, 0700); err != nil {
			if uc.logger != nil {
				uc.logger.Error("Failed to ensure directory '%s': %v", dir, err)
			}
			result.ConfigDiagnostics = append(result.ConfigDiagnostics, entity.Diagnostic{
				Category: entity.DiagError,
				System:   "Directory",
				Target:   dir,
				Details:  fmt.Sprintf("Failed to create directory: %v", err),
			})
			continue
		}

		if dir == "~/Documents/SSH-keys" || dir == "~/.ssh-manager" || dir == "~/.ssh" {
			if err := uc.fsManager.SetStrictWindowsACL(dir); err != nil {
				if uc.logger != nil {
					uc.logger.Warn("Could not apply strict ACLs on '%s': %v", dir, err)
				}
			} else {
				if uc.logger != nil {
					uc.logger.Info("Applied strict ACLs (current user only) to '%s'", dir)
				}
			}
			result.RestrictedDirs = append(result.RestrictedDirs, dir)
		}

		result.ConfigDiagnostics = append(result.ConfigDiagnostics, entity.Diagnostic{
			Category: entity.DiagOK,
			System:   "Directory",
			Target:   dir,
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

		// Write with atomic backup
		backupPath, writeErr := uc.fsManager.WriteWithBackup(cf.Destination, content, 0644)
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

	return result, nil
}
