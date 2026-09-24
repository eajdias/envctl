package usecase

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
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

// categoryAllowed reports whether an entry belongs to one of the requested
// agent subsystems. An empty filter provisions everything (machine-level
// entries included, which is what `envctl run shell`/`run all` do); a filter
// limits the run to the listed categories, so `envctl commandcode` never
// touches opencode files and vice versa.
func categoryAllowed(category string, filter []string) bool {
	if len(filter) == 0 {
		return true
	}
	for _, f := range filter {
		if f == category {
			return true
		}
	}
	return false
}

// Execute provisions the shell, environment and config layers. Pass category
// filters ("opencode", "commandcode") to restrict the run to one agent.
func (uc *ProvisionShellUseCase) Execute(ctx context.Context, categories ...string) (*ProvisionShellResult, error) {
	if uc.logger != nil {
		uc.logger.Info("Starting shell, environment, git, and configs provisioning (categories: %v)", categories)
	}

	result := &ProvisionShellResult{
		CreatedBackups: make(map[string]string),
	}

	// 1. Environment Variables (machine-level: skipped when a category filter
	// limits the run to one agent subsystem).
	envVars, err := uc.manifestRepo.LoadEnvVars()
	if err == nil && len(envVars) > 0 && len(categories) == 0 {
		diags, _ := uc.envManager.EnsureEnvVars(ctx, envVars)
		result.EnvDiagnostics = diags
		for _, d := range diags {
			if uc.logger != nil {
				uc.logger.LogIdempotency("Environment", d.Target, d.Category == entity.DiagOK, d.Details)
			}
		}
		if localBin, err := uc.fsManager.ExpandUserPath("~/.local/bin"); err == nil {
			if changed, err := uc.envManager.EnsurePathEntry(ctx, localBin); err != nil {
				if uc.logger != nil {
					uc.logger.Warn("Failed to ensure ~/.local/bin on PATH: %v", err)
				}
			} else if changed && uc.logger != nil {
				uc.logger.LogIdempotency("Environment", "PATH", false, "~/.local/bin prepended to user PATH")
			}
		}
	}

	// 2. Git Performance Configurations (machine-level: skipped when a category
	// filter limits the run to one agent subsystem).
	gitConfigs, err := uc.manifestRepo.LoadGitConfigs()
	if err == nil && len(gitConfigs) > 0 && len(categories) == 0 {
		var applicable []entity.GitConfig
		for _, gc := range gitConfigs {
			if !entity.MatchesOS(gc.OS) {
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
		if !entity.MatchesOS(dir.OS) {
			continue
		}
		if !categoryAllowed(dir.Category, categories) {
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
		if !entity.MatchesOS(cf.OS) {
			continue
		}
		if !categoryAllowed(cf.Category, categories) {
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

		// Merge modes combine the template with whatever is already on disk
		// instead of replacing it, so user-owned entries (ssh host stanzas,
		// extra npm dependencies) are never silently discarded.
		existedBefore := uc.fsManager.Exists(cf.Destination)
		if cf.Merge != entity.MergeOverwrite && existedBefore {
			existingContent, readErr := uc.fsManager.ReadFile(cf.Destination)
			if readErr == nil {
				switch cf.Merge {
				case entity.MergeSSHHosts:
					content = mergeSSHHosts(content, existingContent)
				case entity.MergeJSONDeps:
					mergedContent, mergeErr := mergeJSONDeps(content, existingContent)
					if mergeErr != nil {
						// Never replace an unparseable user file with the template.
						if uc.logger != nil {
							uc.logger.Warn("Keeping '%s' untouched: %v", cf.Destination, mergeErr)
						}
						result.ConfigDiagnostics = append(result.ConfigDiagnostics, entity.Diagnostic{
							Category: entity.DiagWarning,
							System:   "ConfigFile",
							Target:   cf.Destination,
							Details:  fmt.Sprintf("Preserved user content (not mergeable: %v)", mergeErr),
							FixHint:  "Fix the JSON syntax so provisioning can merge the managed baseline",
						})
						continue
					}
					content = mergedContent
				}
			}
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
			// Executable scripts (e.g. ~/.local/bin helpers): ensure the
			// POSIX exec bit survives provisioning (WriteWithBackup writes
			// 0644; Windows ignores the bit harmlessly).
			if cf.Executable && runtime.GOOS != "windows" {
				if expanded, err := uc.fsManager.ExpandUserPath(cf.Destination); err == nil {
					if err := os.Chmod(expanded, 0755); err != nil && uc.logger != nil {
						uc.logger.Warn("Could not set executable bit on '%s': %v", cf.Destination, err)
					}
				}
			}

			// Report what actually happened: a newly created file and an
			// already-identical one both skip the backup, but only the latter
			// is "up to date".
			detail := "Config written successfully"
			switch {
			case backupPath != "":
				result.CreatedBackups[cf.Destination] = backupPath
				detail = fmt.Sprintf("Updated (Backup saved to %s)", filepath.Base(backupPath))
				if uc.logger != nil {
					uc.logger.LogIdempotency("ConfigFile", cf.Destination, false, fmt.Sprintf("content updated, backup created at %s", backupPath))
				}
			case !existedBefore:
				detail = "Created"
				if uc.logger != nil {
					uc.logger.LogIdempotency("ConfigFile", cf.Destination, false, "file created")
				}
			default:
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
			if !entity.MatchesOS(item.OS) {
				continue
			}
			if !categoryAllowed(item.Category, categories) {
				continue
			}
			expandedPath, err := uc.fsManager.ExpandUserPath(item.Path)
			if err != nil {
				continue
			}
			if !uc.fsManager.Exists(expandedPath) {
				continue
			}
			if item.KeepNewest > 0 {
				pruned, err := pruneTimestampedBackups(expandedPath, item.KeepNewest)
				if err != nil {
					if uc.logger != nil {
						uc.logger.Warn("Failed to prune backups in '%s': %v", expandedPath, err)
					}
					continue
				}
				if uc.logger != nil {
					uc.logger.Info("Pruned %d old backup(s) in %s (%s)", len(pruned), expandedPath, item.Description)
				}
				if len(pruned) > 0 {
					result.ConfigDiagnostics = append(result.ConfigDiagnostics, entity.Diagnostic{
						Category: entity.DiagOK,
						System:   "Cleanup",
						Target:   expandedPath,
						Details:  fmt.Sprintf("Pruned %d old backup(s), kept newest %d per file: %s", len(pruned), item.KeepNewest, item.Description),
					})
				}
				continue
			}
			if item.Recursive {
				err = os.RemoveAll(expandedPath)
			} else {
				err = os.Remove(expandedPath)
			}
			if err != nil {
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

	// 5. OpenCode Plugins npm dependencies installation (opencode subsystem only)
	opencodeConfigDir, _ := uc.fsManager.ExpandUserPath("~/.config/opencode")
	packageJsonPath := filepath.Join(opencodeConfigDir, "package.json")
	if categoryAllowed("opencode", categories) && uc.fsManager.Exists(packageJsonPath) {
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
					Details:  "OpenCode plugins installed (@opencode/plugin)",
				})
			}
		} else {
			if uc.logger != nil {
				uc.logger.LogIdempotency("OpenCodePlugins", packageJsonPath, true, "node_modules already exists in ~/.config/opencode")
			}
		}
	}

	// 6. User Home npm dependencies (agent automation libs: axios, cheerio, papaparse).
	// Browser automation is MCP-only with the bundled browser — no Playwright
	// Node API module and no separate Chromium provisioning here.
	userHomeDir, _ := uc.fsManager.ExpandUserPath("~")
	userPackageJsonPath := filepath.Join(userHomeDir, "package.json")
	if categoryAllowed("runtimes", categories) && uc.fsManager.Exists(userPackageJsonPath) {
		userNodeModulesPath := filepath.Join(userHomeDir, "node_modules")
		nodeModulesMissing := !uc.fsManager.Exists(userNodeModulesPath)
		pkgJsonInfo, _ := os.Stat(userPackageJsonPath)
		nmInfo, _ := os.Stat(userNodeModulesPath)
		depsOutdated := pkgJsonInfo != nil && nmInfo != nil && pkgJsonInfo.ModTime().After(nmInfo.ModTime())
		if nodeModulesMissing || depsOutdated {
			if uc.logger != nil {
				uc.logger.Info("Installing user root dependencies (agent libs) in %s via npm", userHomeDir)
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
					System:   "UserRuntime",
					Target:   userPackageJsonPath,
					Details:  fmt.Sprintf("npm install warning: %v", err),
					FixHint:  "Run 'npm install' in user home directory",
				})
			} else {
				if uc.logger != nil {
					uc.logger.Info("Successfully installed user root npm dependencies")
					uc.logger.LogIdempotency("UserRuntime", userPackageJsonPath, false, "Installed user root dependencies")
				}
				result.ConfigDiagnostics = append(result.ConfigDiagnostics, entity.Diagnostic{
					Category: entity.DiagOK,
					System:   "UserRuntime",
					Target:   userPackageJsonPath,
					Details:  "User root npm dependencies installed (axios, cheerio, papaparse)",
				})
			}
		} else {
			if uc.logger != nil {
				uc.logger.LogIdempotency("UserRuntime", userPackageJsonPath, true, "user node_modules already up to date")
			}
			result.ConfigDiagnostics = append(result.ConfigDiagnostics, entity.Diagnostic{
				Category: entity.DiagOK,
				System:   "UserRuntime",
				Target:   userPackageJsonPath,
				Details:  "User root npm dependencies verified in user root",
			})
		}

	}

	return result, nil
}

// pruneTimestampedBackups removes `<name>.bak.YYYYMMDD-HHMMSS` files inside dir,
// keeping the newest keep per original file. Returns the removed file names.
func pruneTimestampedBackups(dir string, keep int) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	groups := make(map[string][]os.DirEntry)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		idx := strings.Index(name, ".bak.")
		if idx < 0 || !validBackupSuffix(name[idx+len(".bak."):]) {
			continue
		}
		key := name[:idx]
		groups[key] = append(groups[key], e)
	}
	var removed []string
	for _, files := range groups {
		if len(files) <= keep {
			continue
		}
		sort.Slice(files, func(i, j int) bool {
			ii, iErr := files[i].Info()
			jj, jErr := files[j].Info()
			if iErr != nil || jErr != nil {
				return files[i].Name() > files[j].Name()
			}
			if ii.ModTime().Equal(jj.ModTime()) {
				return files[i].Name() > files[j].Name()
			}
			return ii.ModTime().After(jj.ModTime())
		})
		for _, f := range files[keep:] {
			if err := os.Remove(filepath.Join(dir, f.Name())); err != nil {
				return removed, err
			}
			removed = append(removed, f.Name())
		}
	}
	sort.Strings(removed)
	return removed, nil
}

// validBackupSuffix reports whether s matches the provisioning backup
// timestamp format YYYYMMDD-HHMMSS (e.g. 20260916-140849).
func validBackupSuffix(s string) bool {
	if len(s) != 15 || s[8] != '-' {
		return false
	}
	for i, c := range s {
		if i == 8 {
			continue
		}
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
