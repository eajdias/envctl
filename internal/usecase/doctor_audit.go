package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
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
		default:
			// DiagInfo and future informational categories count as passed:
			// they carry context, not problems.
			report.Passed++
			if uc.logger != nil {
				uc.logger.Info("[AUDIT-INFO] [%s] %s: %s", diag.System, diag.Target, diag.Details)
			}
		}
	}

	// 1. Audit Environment Variables
	envVars, _ := uc.manifestRepo.LoadEnvVars()
	for _, ev := range envVars {
		if !entity.MatchesOS(ev.OS) {
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

	// 1.5. Audit ~/.local/bin on PATH (provisioned helpers like `pw` live here).
	if localBin, err := uc.fsManager.ExpandUserPath("~/.local/bin"); err == nil && localBin != "" {
		onPath := false
		if runtime.GOOS == "windows" {
			if pathVal, err := uc.envManager.GetEnvVar("User", "Path"); err == nil {
				for _, p := range strings.Split(pathVal, ";") {
					if strings.EqualFold(strings.TrimSpace(p), localBin) {
						onPath = true
						break
					}
				}
			} else if p, err := exec.LookPath("pw.cmd"); err == nil && p != "" {
				onPath = true
			}
		} else {
			for _, rc := range []string{filepath.Join(os.Getenv("HOME"), ".profile"), filepath.Join(os.Getenv("HOME"), ".bashrc")} {
				if data, err := os.ReadFile(rc); err == nil && strings.Contains(string(data), localBin) {
					onPath = true
					break
				}
			}
			if !onPath {
				for _, seg := range strings.Split(os.Getenv("PATH"), string(os.PathListSeparator)) {
					if seg == localBin {
						onPath = true
						break
					}
				}
			}
		}
		if onPath {
			addDiag(entity.Diagnostic{
				Category: entity.DiagOK,
				System:   "Environment",
				Target:   "PATH (~/.local/bin)",
				Details:  "~/.local/bin is on PATH (provisioned helpers resolve)",
			})
		} else {
			addDiag(entity.Diagnostic{
				Category: entity.DiagWarning,
				System:   "Environment",
				Target:   "PATH (~/.local/bin)",
				Details:  "~/.local/bin is not on PATH — provisioned helpers (pw) do not resolve by bare name",
				FixHint:  "run 'envctl run shell' to prepend ~/.local/bin to the user PATH",
			})
		}
	}

	// 2. Audit Git Global Configurations
	gitConfigs, _ := uc.manifestRepo.LoadGitConfigs()
	for _, gc := range gitConfigs {
		if !entity.MatchesOS(gc.OS) {
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
		if !entity.MatchesOS(cf.OS) {
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
			details := "Present on disk"
			switch {
			case cf.SeedIfMissing:
				// Seed templates are intentionally local (per-machine additions),
				// so there is nothing to compare against.
			case cf.Merge != entity.MergeOverwrite:
				// Merge-mode files legitimately carry user content on top of the
				// template (ssh host stanzas, extra dependencies); a byte
				// comparison would always diverge.
				details = "Present on disk (merged with user content)"
			default:
				if src, err := uc.fsManager.ReadFile(cf.Source); err == nil {
					if dst, err := uc.fsManager.ReadFile(cf.Destination); err == nil && string(dst) != string(src) {
						addDiag(entity.Diagnostic{
							Category: entity.DiagWarning,
							System:   "ConfigFile",
							Target:   cf.Destination,
							Details:  "Content diverges from provisioned source",
							FixHint:  "run 'envctl run shell' to restore the provisioned content",
						})
						continue
					}
				}
			}
			addDiag(entity.Diagnostic{
				Category: entity.DiagOK,
				System:   "ConfigFile",
				Target:   cf.Destination,
				Details:  details,
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

	// 4.5. Audit OpenCode MCP file references: opencode fails hard at
	// startup when a `{file:...}` reference points to a missing file
	// (e.g. a secrets key that was never created on this machine), so a
	// "present on disk" opencode.json is not enough — every referenced
	// file must exist too.
	uc.auditOpenCodeFileRefs(addDiag)

	// 5. Audit Packages
	packages, _ := uc.manifestRepo.LoadPackages()
	for _, pkg := range packages {
		if !entity.MatchesOS(pkg.OS) {
			continue
		}

		mgr, ok := uc.managers[pkg.Type]
		if !ok || !mgr.IsAvailable(ctx) {
			// A package whose manager does not exist on this machine (e.g.
			// pacman entries on Ubuntu/Debian, winget entries on Linux) is
			// not applicable here — skip silently instead of warning, or
			// every Ubuntu doctor would drown in 20+ pacman warnings.
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

	// 6. Audit Skills (only the ones that belong on this OS and are enabled)
	skills, _ := uc.manifestRepo.LoadSkills()
	for _, s := range skills {
		if !s.Enabled || !s.AppliesToOS(runtime.GOOS) {
			continue
		}
		targetDir := s.TargetDir
		if targetDir == "" {
			targetDir = filepath.Join("~/.config/opencode/skills", s.Name)
		}

		if !uc.fsManager.Exists(targetDir) {
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
		if !entity.MatchesOS(lsp.OS) {
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

	// 9. Audit Browser & Playwright
	// 9.1 Audit Google Chrome (native system browser for chrome-devtools MCP
	// & CLI tools). Playwright CLI brings its own bundled Chromium, so a
	// missing system Chrome only matters for chrome-devtools-mcp — downgrade
	// to Info on Linux instead of warning on every headless VPS.
	isLinux := runtime.GOOS == "linux"
	chromePath := ""
	if runtime.GOOS == "windows" {
		candidates := []string{
			`C:\Program Files\Google\Chrome\Application\chrome.exe`,
			`C:\Program Files (x86)\Google\Chrome\Application\chrome.exe`,
		}
		if localAppData := os.Getenv("LOCALAPPDATA"); localAppData != "" {
			candidates = append(candidates, filepath.Join(localAppData, `Google\Chrome\Application\chrome.exe`))
		}
		for _, c := range candidates {
			if uc.fsManager.Exists(c) {
				chromePath = c
				break
			}
		}
	} else if runtime.GOOS == "darwin" {
		c := "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"
		if uc.fsManager.Exists(c) {
			chromePath = c
		}
	} else {
		candidates := []string{
			"/usr/bin/google-chrome",
			"/usr/bin/google-chrome-stable",
			"/usr/bin/chromium-browser",
			"/usr/bin/chromium",
		}
		for _, c := range candidates {
			if uc.fsManager.Exists(c) {
				chromePath = c
				break
			}
		}
	}
	if chromePath == "" {
		if p, err := exec.LookPath("google-chrome"); err == nil {
			chromePath = p
		} else if p, err := exec.LookPath("chrome"); err == nil {
			chromePath = p
		}
	}

	if chromePath == "" {
		fixHint := "install Google Chrome (winget install Google.Chrome / apt install google-chrome-stable)"
		if runtime.GOOS == "darwin" {
			fixHint = "install Google Chrome (brew install --cask google-chrome)"
		}
		category := entity.DiagWarning
		system := "Browser"
		details := "Google Chrome not detected (recommended for chrome-devtools-mcp)"
		if isLinux {
			category = entity.DiagInfo
			details = "Google Chrome not detected (only needed for chrome-devtools-mcp; Playwright CLI brings its own bundled Chromium)"
		}
		addDiag(entity.Diagnostic{
			Category: category,
			System:   system,
			Target:   "Google Chrome",
			Details:  details,
			FixHint:  fixHint,
		})
	} else {
		addDiag(entity.Diagnostic{
			Category: entity.DiagOK,
			System:   "Browser",
			Target:   "Google Chrome",
			Details:  fmt.Sprintf("Google Chrome detected at %s", chromePath),
		})
	}

	// 9.2 Audit Playwright CLI bundled Chromium + pw wrapper: deterministic
	// automation (`playwright-cli` via `pw`) runs on the bundled build, so
	// verify at least one usable binary exists under the playwright cache
	// (%LOCALAPPDATA%\ms-playwright on Windows, ~/.cache/ms-playwright on
	// POSIX), plus the provisioned wrapper itself.
	browserCacheDir := ""
	pwWrapper := ""
	if runtime.GOOS == "windows" {
		if localAppData := os.Getenv("LOCALAPPDATA"); localAppData != "" {
			browserCacheDir = filepath.Join(localAppData, "ms-playwright")
		}
		if homeDir, _ := uc.fsManager.ExpandUserPath("~"); homeDir != "" {
			pwWrapper = filepath.Join(homeDir, ".local", "bin", "pw.cjs")
		}
	} else {
		if homeDir, _ := uc.fsManager.ExpandUserPath("~"); homeDir != "" {
			browserCacheDir = filepath.Join(homeDir, ".cache", "ms-playwright")
			pwWrapper = filepath.Join(homeDir, ".local", "bin", "pw.cjs")
		}
	}
	if browserCacheDir != "" {
		found := false
		if entries, err := os.ReadDir(browserCacheDir); err == nil {
			for _, e := range entries {
				n := e.Name()
				if strings.HasPrefix(n, "chromium-") || strings.HasPrefix(n, "chromium_headless_shell-") {
					found = true
					break
				}
			}
		}
		if found {
			addDiag(entity.Diagnostic{
				Category: entity.DiagOK,
				System:   "Browser",
				Target:   "Playwright Chromium",
				Details:  fmt.Sprintf("Bundled Chromium present in %s (playwright-cli ready)", browserCacheDir),
			})
		} else {
			addDiag(entity.Diagnostic{
				Category: entity.DiagWarning,
				System:   "Browser",
				Target:   "Playwright Chromium",
				Details:  fmt.Sprintf("No bundled Chromium in %s — playwright-cli cannot launch a browser", browserCacheDir),
				FixHint:  "run 'bunx @playwright/cli@latest install-browser chromium'",
			})
		}
	}

	// 9.3 Audit pw wrapper: the hang-safe runner must be provisioned, or
	// agents fall back to raw playwright-cli and hang on Windows.
	if pwWrapper == "" {
		if homeDir, _ := uc.fsManager.ExpandUserPath("~"); homeDir != "" {
			pwWrapper = filepath.Join(homeDir, ".local", "bin", "pw.cjs")
		}
	}
	if pwWrapper != "" {
		if uc.fsManager.Exists(pwWrapper) {
			addDiag(entity.Diagnostic{
				Category: entity.DiagOK,
				System:   "Browser",
				Target:   "pw wrapper",
				Details:  fmt.Sprintf("Hang-safe runner present at %s", pwWrapper),
			})
		} else {
			addDiag(entity.Diagnostic{
				Category: entity.DiagWarning,
				System:   "Browser",
				Target:   "pw wrapper",
				Details:  "pw wrapper missing — agents have no hang-safe playwright-cli path on Windows",
				FixHint:  "run 'envctl run shell' to provision ~/.local/bin/pw.cjs",
			})
		}
	}

	// 10. Audit Git Worktree Support
	// `git worktree list` exits 128 outside a git repository, which is expected
	// and not a fault of the git installation. Only run the command from inside a repo.
	if _, err := exec.LookPath("git"); err != nil {
		addDiag(entity.Diagnostic{
			Category: entity.DiagWarning,
			System:   "Git",
			Target:   "git worktree",
			Details:  "git binary not found in PATH",
			FixHint:  "install git (winget install Git.Git / apt-get install -y git)",
		})
	} else if err := exec.CommandContext(ctx, "git", "rev-parse", "--is-inside-work-tree").Run(); err != nil {
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

	// 10.5 Audit the local verification wiring: the same gates run by the
	// CommandCode Stop hook and the git pre-push hook. Without them a broken
	// tree only surfaces after the push, which is what this wiring exists to
	// prevent.
	verifierPath, verifierErr := uc.fsManager.ExpandUserPath("~/.local/bin/envctl-verify")
	prePushPath, prePushErr := uc.fsManager.ExpandUserPath("~/.config/git/hooks/pre-push")
	if verifierErr != nil || prePushErr != nil {
		if uc.logger != nil {
			uc.logger.Warn("Could not resolve the verification paths: %v / %v", verifierErr, prePushErr)
		}
	}
	verifierReady := isExecutableFile(verifierPath)
	prePushReady := isExecutableFile(prePushPath)

	switch {
	case verifierReady && prePushReady:
		addDiag(entity.Diagnostic{
			Category: entity.DiagOK,
			System:   "Verify",
			Target:   "local quality gates",
			Details:  "envctl-verify deployed and the git pre-push hook is executable",
		})
	case verifierReady:
		addDiag(entity.Diagnostic{
			Category: entity.DiagWarning,
			System:   "Verify",
			Target:   "git pre-push hook",
			Details:  "pre-push hook missing or not executable — pushes are not gated locally",
			FixHint:  "run 'envctl run shell' to deploy ~/.config/git/hooks/pre-push",
		})
	default:
		addDiag(entity.Diagnostic{
			Category: entity.DiagWarning,
			System:   "Verify",
			Target:   "envctl-verify",
			Details:  "local verifier not deployed — lint/test failures surface only in CI",
			FixHint:  "run 'envctl run shell' to deploy ~/.local/bin/envctl-verify",
		})
	}

	// 11. Audit Linux Toolchain Bootstrap (Linux only)
	if runtime.GOOS == "linux" {
		userHomeDir, homeErr := uc.fsManager.ExpandUserPath("~")
		if homeErr != nil && uc.logger != nil {
			uc.logger.Warn("Could not expand the home directory for the Linux toolchain audit: %v", homeErr)
		}
		env := linuxToolchainEnv(userHomeDir)
		bootstrapTools := []struct {
			name string
			desc string
		}{
			{"volta", "Volta JS toolchain manager"},
			{"node", "Node.js (via Volta)"},
			{"opencode", "OpenCode CLI"},
			{"cmdc", "CommandCode CLI"},
			{"gh", "GitHub CLI"},
			{"delta", "git-delta pager"},
			{"yq", "yq YAML/JSON processor"},
			{"uv", "uv Python package manager"},
			{"ruff", "ruff linter (via uv)"},
			{"fd", "fd (fdfind symlink)"},
			{"pylsp", "python-lsp-server (via uv)"},
			{"stylelint", "Stylelint CSS/SCSS linter (via Volta)"},
			{"bun", "Bun JS/TS runtime (browser CLI/MCP launcher via bunx)"},
			{"playwright-chromium", "Playwright CLI bundled Chromium (deterministic automation via CLI installer)"},
			{"go", "Go programming language SDK"},
		}
		for _, t := range bootstrapTools {
			found := false
			if t.name == "playwright-chromium" {
				// Not a PATH binary: the bundled build lives under
				// ~/.cache/ms-playwright (see the Browser/Playwright
				// Chromium check). Mirror that logic here so both
				// diagnostics agree.
				if entries, err := os.ReadDir(filepath.Join(userHomeDir, ".cache", "ms-playwright")); err == nil {
					for _, e := range entries {
						n := e.Name()
						if strings.HasPrefix(n, "chromium-") || strings.HasPrefix(n, "chromium_headless_shell-") {
							found = true
							break
						}
					}
				}
			} else {
				found = func() bool {
					c := exec.CommandContext(ctx, "bash", "-lc", "command -v "+t.name+" >/dev/null 2>&1")
					c.Env = env
					return c.Run() == nil
				}()
			}
			if found {
				details := "Tool available on PATH (" + t.name + ")"
				if t.name == "playwright-chromium" {
					details = "Bundled Chromium present in ~/.cache/ms-playwright"
				}
				addDiag(entity.Diagnostic{
					Category: entity.DiagOK,
					System:   "LinuxBootstrap",
					Target:   t.desc,
					Details:  details,
				})
			} else {
				details := "Tool not found on PATH (" + t.name + ")"
				if t.name == "playwright-chromium" {
					details = "No bundled Chromium in ~/.cache/ms-playwright"
				}
				addDiag(entity.Diagnostic{
					Category: entity.DiagWarning,
					System:   "LinuxBootstrap",
					Target:   t.desc,
					Details:  details,
					FixHint:  "run 'envctl run bootstrap'",
				})
			}
		}
	}

	// 11.5. Audit WSL Ubuntu secondary shell (Windows only)
	if runtime.GOOS == "windows" {
		out, err := exec.CommandContext(ctx, "wsl.exe", "-l", "-q").CombinedOutput()
		// Windows console output is UTF-16: strip null bytes before matching.
		clean := strings.ReplaceAll(string(out), "\x00", "")
		if err != nil {
			addDiag(entity.Diagnostic{
				Category: entity.DiagWarning,
				System:   "WSL",
				Target:   "Ubuntu",
				Details:  fmt.Sprintf("wsl.exe check failed: %v", err),
				FixHint:  "run 'wsl --install -d Ubuntu' (WSL2)",
			})
		} else if !strings.Contains(strings.ToLower(clean), "ubuntu") {
			addDiag(entity.Diagnostic{
				Category: entity.DiagWarning,
				System:   "WSL",
				Target:   "Ubuntu",
				Details:  "WSL Ubuntu distro not found (secondary POSIX shell)",
				FixHint:  "run 'wsl --install -d Ubuntu' (WSL2)",
			})
		} else {
			addDiag(entity.Diagnostic{
				Category: entity.DiagOK,
				System:   "WSL",
				Target:   "Ubuntu",
				Details:  "WSL Ubuntu available as secondary POSIX shell",
			})
		}
	}

	// 11.6. Audit Windows console code page (UTF-8 required for Unicode glyph rendering)
	if runtime.GOOS == "windows" {
		out, err := exec.CommandContext(ctx, "chcp").CombinedOutput()
		codePage := strings.TrimSpace(regexp.MustCompile(`\d+`).FindString(string(out)))
		if err != nil {
			addDiag(entity.Diagnostic{
				Category: entity.DiagWarning,
				System:   "Console",
				Target:   "Code Page",
				Details:  fmt.Sprintf("chcp check failed: %v", err),
				FixHint:  "run 'envctl run shell' to reapply the UTF-8 PowerShell profile",
			})
		} else if codePage != "65001" {
			addDiag(entity.Diagnostic{
				Category: entity.DiagWarning,
				System:   "Console",
				Target:   "Code Page",
				Details:  fmt.Sprintf("Console code page is %s — Unicode glyphs (emojis, checkmarks) render as U+FFFD", codePage),
				FixHint:  "restart Windows Terminal (its profile now sets chcp 65001) or run 'envctl run shell' to reapply the PowerShell profile",
			})
		} else {
			addDiag(entity.Diagnostic{
				Category: entity.DiagOK,
				System:   "Console",
				Target:   "Code Page",
				Details:  "Console code page is UTF-8 (65001) — Unicode output safe",
			})
		}

		// 11.6.1. Audit system ACP/OEMCP — the durable root cause. The chcp result above
		// depends on how envctl was launched (the PowerShell profile sets 65001 only in
		// interactive shells); opencode spawns pwsh with -NoProfile, so the system code
		// page is the only layer covering every process (shell tool, cmd.exe, services).
		regOut, regErr := exec.CommandContext(ctx, "reg", "query", `HKLM\SYSTEM\CurrentControlSet\Control\Nls\CodePage`).CombinedOutput()
		acp, oemcp := "", ""
		for _, m := range regexp.MustCompile(`(\w+)\s+REG_\w+\s+(\d+)`).FindAllStringSubmatch(string(regOut), -1) {
			switch m[1] {
			case "ACP":
				acp = m[2]
			case "OEMCP":
				oemcp = m[2]
			}
		}
		if regErr != nil || acp == "" || oemcp == "" {
			addDiag(entity.Diagnostic{
				Category: entity.DiagWarning,
				System:   "Console",
				Target:   "System Code Page",
				Details:  fmt.Sprintf("system code page check failed: %v", regErr),
				FixHint:  "enable 'Beta: Use Unicode UTF-8 for worldwide language support' (Settings > Time & Language > Language & region > Administrative language settings > Change system locale), then reboot",
			})
		} else if acp != "65001" || oemcp != "65001" {
			addDiag(entity.Diagnostic{
				Category: entity.DiagWarning,
				System:   "Console",
				Target:   "System Code Page",
				Details:  fmt.Sprintf("System ACP/OEMCP is %s/%s — processes spawned without the PowerShell profile (opencode shell tool, cmd.exe, services) still emit CP%s and render as U+FFFD", acp, oemcp, oemcp),
				FixHint:  "enable 'Beta: Use Unicode UTF-8 for worldwide language support' (Settings > Time & Language > Language & region > Administrative language settings > Change system locale) or set HKLM\\SYSTEM\\CurrentControlSet\\Control\\Nls\\CodePage ACP/OEMCP=65001, then reboot",
			})
		} else {
			addDiag(entity.Diagnostic{
				Category: entity.DiagOK,
				System:   "Console",
				Target:   "System Code Page",
				Details:  "System ACP/OEMCP is UTF-8 (65001) — Unicode safe for every process",
			})
		}
	}

	// 12. Audit OpenCode storage accumulation & standardized temp folder
	homeDir, homeErr := uc.fsManager.ExpandUserPath("~")
	if homeErr != nil && uc.logger != nil {
		uc.logger.Warn("Could not resolve the user home for the OpenCode store audit: %v", homeErr)
	}
	opencodeDataDir := openCodeDataDir(homeDir)

	dbPath := openCodeStorePath(homeDir)
	store, storeErr := InspectOpenCodeStore(dbPath)
	switch {
	case storeErr == nil && store.ExceedsThreshold():
		addDiag(entity.Diagnostic{
			Category: entity.DiagWarning,
			System:   "OpenCode",
			Target:   "Database",
			Details: fmt.Sprintf("opencode.db is %.1f MB (threshold: %d MB); %.1f MB reclaimable by VACUUM — the rest is live session history",
				float64(store.SizeBytes)/(1024*1024), openCodeStoreWarnBytes/(1024*1024), float64(store.ReclaimableBytes)/(1024*1024)),
			FixHint: "run 'envctl run cleanup' to reclaim free pages; shrinking further means pruning sessions in OpenCode",
		})
	case storeErr == nil:
		addDiag(entity.Diagnostic{
			Category: entity.DiagOK,
			System:   "OpenCode",
			Target:   "Database",
			Details:  fmt.Sprintf("Database OK (%.1f MB)", float64(store.SizeBytes)/(1024*1024)),
		})
	default:
		addDiag(entity.Diagnostic{
			Category: entity.DiagOK,
			System:   "OpenCode",
			Target:   "Database",
			Details:  "Database not found (no opencode.db — clean state)",
		})
	}

	toolOutputDir := filepath.Join(opencodeDataDir, "tool-output")
	if toolSize, err := dirSize(toolOutputDir); err == nil && toolSize > 50*1024*1024 {
		addDiag(entity.Diagnostic{
			Category: entity.DiagWarning,
			System:   "OpenCode",
			Target:   "Tool Output",
			Details:  fmt.Sprintf("tool-output is %.1f MB (threshold: 50 MB)", float64(toolSize)/(1024*1024)),
			FixHint:  "run 'envctl run cleanup' to remove files >10 MB",
		})
	}

	// Stale config: standard config file is opencode.json; jsonc is a legacy conflict source.
	for _, stale := range []string{
		"~/.config/opencode/opencode.jsonc",
		"~/.config/opencode/opencode.linux.jsonc",
	} {
		if uc.fsManager.Exists(stale) {
			addDiag(entity.Diagnostic{
				Category: entity.DiagWarning,
				System:   "OpenCode",
				Target:   stale,
				Details:  "Legacy config file found — arrays don't merge with opencode.json, causing duplicate LSPs/plugins",
				FixHint:  "run 'envctl run shell' to remove it (cleanup step)",
			})
		}
	}

	// Standardized global temp folder for LLM agent scratch (ENVCTL_TEMP).
	var tempDir string
	if runtime.GOOS == "windows" {
		tempDir = `C:\temp`
	} else {
		tempDir = "/temp"
	}
	if !uc.fsManager.Exists(tempDir) {
		addDiag(entity.Diagnostic{
			Category: entity.DiagWarning,
			System:   "TempFolder",
			Target:   tempDir,
			Details:  "Standardized agent temp folder (ENVCTL_TEMP) missing",
			FixHint:  "run 'envctl run shell' to create it",
		})
	} else if tempSize, err := dirSize(tempDir); err == nil && tempSize > 500*1024*1024 {
		addDiag(entity.Diagnostic{
			Category: entity.DiagWarning,
			System:   "TempFolder",
			Target:   tempDir,
			Details:  fmt.Sprintf("Agent temp folder is %.1f MB — clean stale scratch", float64(tempSize)/(1024*1024)),
			FixHint:  "run 'envctl run cleanup' or delete its contents manually",
		})
	} else {
		addDiag(entity.Diagnostic{
			Category: entity.DiagOK,
			System:   "TempFolder",
			Target:   tempDir,
			Details:  "Standardized agent temp folder present (ENVCTL_TEMP)",
		})
	}

	// 13. Audit CommandCode Agent Health
	if runtime.GOOS == "linux" || runtime.GOOS == "windows" {
		var cmdPath string
		var cmdErr error
		if runtime.GOOS == "windows" {
			// On Windows bare `cmd` resolves to System32\cmd.exe even when
			// CommandCode is absent, so only cmdc/command-code prove install.
			cmdPath, cmdErr = exec.LookPath("cmdc")
			if cmdErr != nil {
				cmdPath, cmdErr = exec.LookPath("command-code")
			}
			if cmdErr != nil {
				cmdPath, cmdErr = exec.LookPath("commandcode")
			}
		} else {
			cmdPath, cmdErr = exec.LookPath("cmd")
			if cmdErr != nil {
				cmdPath, cmdErr = exec.LookPath("cmdc")
			}
		}
		if cmdErr != nil {
			fixHint := "Run 'envctl run bootstrap' or 'npm install -g command-code'"
			if runtime.GOOS == "windows" {
				fixHint = "Run 'envctl run volta' or 'volta install command-code'"
			}
			addDiag(entity.Diagnostic{
				Category: entity.DiagWarning,
				System:   "CommandCode",
				Target:   "CommandCode CLI",
				Details:  "CommandCode CLI not found in PATH",
				FixHint:  fixHint,
			})
		} else {
			addDiag(entity.Diagnostic{
				Category: entity.DiagOK,
				System:   "CommandCode",
				Target:   "CommandCode CLI",
				Details:  fmt.Sprintf("Found at %s", cmdPath),
			})
		}

		ccConfigDir, _ := uc.fsManager.ExpandUserPath("~/.commandcode")
		if uc.fsManager.Exists(ccConfigDir) {
			addDiag(entity.Diagnostic{
				Category: entity.DiagOK,
				System:   "CommandCode",
				Target:   "~/.commandcode/",
				Details:  "Config directory exists",
			})

			mcpPath := filepath.Join(ccConfigDir, "mcp.json")
			if uc.fsManager.Exists(mcpPath) {
				data, readErr := os.ReadFile(mcpPath)
				switch {
				case readErr != nil:
					addDiag(entity.Diagnostic{
						Category: entity.DiagWarning,
						System:   "CommandCode",
						Target:   "MCP config",
						Details:  fmt.Sprintf("mcp.json unreadable: %v", readErr),
						FixHint:  "Run 'envctl run shell' to re-provision CommandCode configs",
					})
				case !json.Valid(data):
					addDiag(entity.Diagnostic{
						Category: entity.DiagWarning,
						System:   "CommandCode",
						Target:   "MCP config",
						Details:  "mcp.json is not valid JSON — CommandCode cannot load the servers",
						FixHint:  "Run 'envctl run shell' to re-provision CommandCode configs",
					})
				default:
					addDiag(entity.Diagnostic{
						Category: entity.DiagOK,
						System:   "CommandCode",
						Target:   "MCP config",
						Details:  "mcp.json present and valid JSON",
					})
				}
			} else {
				addDiag(entity.Diagnostic{
					Category: entity.DiagWarning,
					System:   "CommandCode",
					Target:   "MCP config",
					Details:  "mcp.json not found",
					FixHint:  "Run 'envctl run shell' to provision CommandCode configs",
				})
			}

			settingsPath := filepath.Join(ccConfigDir, "settings.json")
			if uc.fsManager.Exists(settingsPath) {
				data, readErr := os.ReadFile(settingsPath)
				switch {
				case readErr != nil:
					addDiag(entity.Diagnostic{
						Category: entity.DiagWarning,
						System:   "CommandCode",
						Target:   "Settings",
						Details:  fmt.Sprintf("settings.json unreadable: %v", readErr),
						FixHint:  "Run 'envctl run shell' to re-provision CommandCode configs",
					})
				case !json.Valid(data):
					addDiag(entity.Diagnostic{
						Category: entity.DiagWarning,
						System:   "CommandCode",
						Target:   "Settings",
						Details:  "settings.json is not valid JSON — CommandCode refuses an invalid settings file",
						FixHint:  "Run 'envctl run shell' to re-provision CommandCode configs",
					})
				default:
					addDiag(entity.Diagnostic{
						Category: entity.DiagOK,
						System:   "CommandCode",
						Target:   "Settings",
						Details:  "settings.json present and valid JSON",
					})
				}
			}

			uc.auditCommandCodeAgents(ccConfigDir, addDiag)

			skillsDir := filepath.Join(ccConfigDir, "skills")
			if uc.fsManager.Exists(skillsDir) {
				entries, _ := os.ReadDir(skillsDir)
				deployed := 0
				var rejected []string
				for _, entry := range entries {
					if !entry.IsDir() {
						continue
					}
					name := entry.Name()
					content, readErr := os.ReadFile(filepath.Join(skillsDir, name, "SKILL.md"))
					if readErr != nil {
						rejected = append(rejected, name+" (SKILL.md missing)")
						continue
					}
					fm, ok := parseSkillFrontmatter(content)
					if !ok {
						rejected = append(rejected, name+" (frontmatter missing or invalid YAML)")
						continue
					}
					if vErr := validateSkillFrontmatter(name, fm); vErr != nil {
						rejected = append(rejected, fmt.Sprintf("%s (%v)", name, vErr))
						continue
					}
					deployed++
				}

				total := deployed + len(rejected)
				switch {
				case len(rejected) > 0:
					addDiag(entity.Diagnostic{
						Category: entity.DiagWarning,
						System:   "CommandCode",
						Target:   "Skills",
						Details:  fmt.Sprintf("%d of %d deployed skills will not load: %s", len(rejected), total, strings.Join(rejected, "; ")),
						FixHint:  "Fix the SKILL.md frontmatter (name must match the directory, description must be non-empty), then run 'envctl run skills'",
					})
				case deployed > 0:
					addDiag(entity.Diagnostic{
						Category: entity.DiagOK,
						System:   "CommandCode",
						Target:   "Skills",
						Details:  fmt.Sprintf("%d skills deployed, frontmatter valid", deployed),
					})
				default:
					addDiag(entity.Diagnostic{
						Category: entity.DiagWarning,
						System:   "CommandCode",
						Target:   "Skills",
						Details:  "Skills directory exists but holds no skill",
						FixHint:  "Run 'envctl run skills' to deploy the manifest skills",
					})
				}

				if total > 0 {
					if manifestSkills, err := uc.manifestRepo.LoadSkills(); err == nil {
						expected := 0
						for _, s := range manifestSkills {
							if s.Enabled && s.AppliesToOS(runtime.GOOS) {
								expected++
							}
						}
						if expected > 0 && total != expected {
							addDiag(entity.Diagnostic{
								Category: entity.DiagWarning,
								System:   "CommandCode",
								Target:   "Skills",
								Details:  fmt.Sprintf("%d skills deployed but the manifest declares %d", total, expected),
								FixHint:  "Run 'envctl run skills' to re-sync the deployed skills with the manifest",
							})
						}
					}
				}
			} else {
				addDiag(entity.Diagnostic{
					Category: entity.DiagWarning,
					System:   "CommandCode",
					Target:   "Skills",
					Details:  "Skills directory not found",
					FixHint:  "Run 'envctl run skills' to deploy skills",
				})
			}
		} else {
			addDiag(entity.Diagnostic{
				Category: entity.DiagInfo,
				System:   "CommandCode",
				Target:   "~/.commandcode/",
				Details:  "Config directory not yet created (CommandCode not provisioned)",
			})
		}
	}

	return report, nil
}

// openCodeFileRefPattern matches opencode `{file:...}` variable references
// embedded in config values (e.g. MCP headers pointing at a secrets file).
var openCodeFileRefPattern = regexp.MustCompile(`\{file:([^}]+)\}`)

// auditOpenCodeFileRefs resolves every `{file:...}` reference found in the
// deployed opencode.json and reports the ones pointing at missing files.
// opencode aborts with "Configuration is invalid: bad file reference" when
// any of them is absent, which breaks EVERY command (even `opencode models`),
// while a plain "file present" check on opencode.json stays green — exactly
// the blind spot that left a VPS with a dead opencode and a passing doctor.
func (uc *DoctorAuditUseCase) auditOpenCodeFileRefs(addDiag func(entity.Diagnostic)) {
	rawPath := "~/.config/opencode/opencode.json"
	expanded, err := uc.fsManager.ExpandUserPath(rawPath)
	if err != nil {
		return
	}
	data, err := os.ReadFile(expanded)
	if err != nil {
		return
	}
	matches := openCodeFileRefPattern.FindAllSubmatch(data, -1)
	if len(matches) == 0 {
		return
	}
	seen := map[string]bool{}
	var missing []string
	for _, m := range matches {
		ref := strings.TrimSpace(string(m[1]))
		if ref == "" || seen[ref] {
			continue
		}
		seen[ref] = true
		resolved, err := uc.fsManager.ExpandUserPath(ref)
		if err != nil {
			resolved = ref
		}
		if _, statErr := os.Stat(resolved); statErr != nil {
			missing = append(missing, ref)
		}
	}
	if len(missing) > 0 {
		addDiag(entity.Diagnostic{
			Category: entity.DiagError,
			System:   "OpenCode",
			Target:   "Config file references",
			Details:  fmt.Sprintf("opencode.json references %d missing file(s): %s — opencode refuses to start until they exist", len(missing), strings.Join(missing, "; ")),
			FixHint:  "create the missing file(s) or run 'envctl run shell' to re-provision opencode.json",
		})
		return
	}
	addDiag(entity.Diagnostic{
		Category: entity.DiagOK,
		System:   "OpenCode",
		Target:   "Config file references",
		Details:  fmt.Sprintf("%d {file:...} reference(s) resolve on disk", len(seen)),
	})
}

// auditCommandCodeAgents validates every custom agent definition under
// <ccConfigDir>/agents, so a broken file cannot silently disable delegation.
// Reserved names are skipped: CommandCode owns those and ignores a custom file.
func (uc *DoctorAuditUseCase) auditCommandCodeAgents(ccConfigDir string, addDiag func(entity.Diagnostic)) {
	agentsDir := filepath.Join(ccConfigDir, "agents")
	entries, err := os.ReadDir(agentsDir)
	if err != nil {
		return
	}

	reserved := map[string]bool{"explore": true, "plan": true, "review": true, "general": true}

	valid := 0
	var broken []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".md") {
			continue
		}
		id := strings.TrimSuffix(name, ".md")
		if reserved[id] {
			continue
		}

		content, readErr := os.ReadFile(filepath.Join(agentsDir, name))
		if readErr != nil {
			broken = append(broken, name+" (unreadable)")
			continue
		}
		fm, ok := parseSkillFrontmatter(content)
		if !ok || strings.TrimSpace(fm.Name) != id {
			broken = append(broken, name+" (frontmatter 'name' missing or different from the filename)")
			continue
		}
		valid++
	}

	if len(broken) > 0 {
		addDiag(entity.Diagnostic{
			Category: entity.DiagWarning,
			System:   "CommandCode",
			Target:   "Agents",
			Details:  fmt.Sprintf("%d agent file(s) will not load: %s", len(broken), strings.Join(broken, "; ")),
			FixHint:  "Fix the frontmatter or remove the stale file, then run 'envctl run shell'",
		})
		return
	}

	addDiag(entity.Diagnostic{
		Category: entity.DiagOK,
		System:   "CommandCode",
		Target:   "Agents",
		Details:  fmt.Sprintf("%d custom agent definition(s) valid", valid),
	})
}
