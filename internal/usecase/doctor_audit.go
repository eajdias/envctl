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
	"time"

	"github.com/eajdias/envctl/internal/domain/entity"
	"github.com/eajdias/envctl/internal/domain/repository"
)

type DoctorAuditUseCase struct {
	manifestRepo         repository.ManifestRepository
	fsManager            repository.FileSystemManager
	envManager           repository.WindowsEnvManager
	gitManager           repository.GitManager
	tweaksManager        repository.WindowsTweaksManager
	managers             map[entity.PackageType]repository.PackageManager
	performanceInspector repository.PerformanceInspector
	logger               repository.Logger
}

func NewDoctorAuditUseCase(
	manifestRepo repository.ManifestRepository,
	fsManager repository.FileSystemManager,
	envManager repository.WindowsEnvManager,
	gitManager repository.GitManager,
	tweaksManager repository.WindowsTweaksManager,
	managers map[entity.PackageType]repository.PackageManager,
	logger repository.Logger,
	performanceInspectors ...repository.PerformanceInspector,
) *DoctorAuditUseCase {
	var performanceInspector repository.PerformanceInspector
	if len(performanceInspectors) > 0 {
		performanceInspector = performanceInspectors[0]
	}
	return &DoctorAuditUseCase{
		manifestRepo:         manifestRepo,
		fsManager:            fsManager,
		envManager:           envManager,
		gitManager:           gitManager,
		tweaksManager:        tweaksManager,
		managers:             managers,
		performanceInspector: performanceInspector,
		logger:               logger,
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
			case cf.RuntimeManaged:
				// The agent writes to this file while it runs (CommandCode
				// appends approved commands to its permission list).
				// Provisioning realigns it to the template, so a byte
				// comparison between runs would always report drift.
				details = "Present on disk (runtime-managed by the agent; provisioning realigns it)"
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
	uc.auditRemovedMCPEntries(addDiag)
	uc.auditAgentsIdentityCoverage(addDiag, configFiles)
	uc.auditOpenCodeVersionSkew(ctx, addDiag)

	// 5. Audit Packages
	packages, _ := uc.manifestRepo.LoadPackages()
	for _, pkg := range packages {
		if !entity.MatchesPackage(pkg) {
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

	// 5.5. Audit Gaming stack (Arch/CachyOS only, opt-in: silent unless Steam
	// is installed, so a non-gaming Arch box stays at zero warnings).
	uc.auditGamingStack(ctx, addDiag)

	// 5.6. Audit read-only OS performance state. Optional differences are
	// informational; this audit never applies a performance tweak.
	uc.auditLinuxPerformance(ctx, func(diagnostic entity.Diagnostic) {
		addDiag(diagnostic)
	})

	// 6. Audit Skills (only the ones that belong on this OS and are enabled).
	// Both agents share the validator: a skill whose frontmatter the loader
	// rejects is silently ignored at runtime, so an existence-only check would
	// report a healthy tree while the agent sees nothing.
	if skillsDir, expandErr := uc.fsManager.ExpandUserPath("~/.config/opencode/skills"); expandErr == nil {
		uc.auditSkillTree("Skills", skillsDir, addDiag)
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

	// 7.5. Audit LSP stdio handshakes (presence in PATH is not proof the
	// server speaks LSP — see skill lsp-smoke-test).
	uc.auditLSPHandshake(ctx, addDiag)

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

	// 8b. Audit opt-in Windows debloat (debloat.yaml). Never WARN: the stack
	// only applies via `run debloat`, so drift is informational. One line per
	// category keeps the report compact; per-tweak detail lives in the
	// `run debloat` output itself.
	uc.auditDebloat(ctx, addDiag)

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
			{"golangci-lint", "golangci-lint (CI lint gate, used by envctl-verify)"},
			{"bun", "Bun JS/TS runtime (browser CLI/MCP launcher via bunx)"},
			{"playwright-chromium", "Playwright CLI bundled Chromium (deterministic automation via CLI installer)"},
			{"go", "Go programming language SDK"},
			{"fzf", "fzf (built-in directory walker)"},
			{"hadolint", "hadolint (Dockerfile linter)"},
		}

		// fzf needs to be new enough to own its directory walker (0.47+):
		// older distro builds fall back to `find`, which is slow and blind to
		// ignore-files. Read the version once and reuse it in both branches.
		fzfVersion := ""
		if out, err := func() ([]byte, error) {
			c := exec.CommandContext(ctx, "bash", "-lc", "fzf --version 2>/dev/null | awk '{print $1}'")
			c.Env = env
			return c.Output()
		}(); err == nil {
			fzfVersion = strings.TrimSpace(string(out))
		}
		for _, t := range bootstrapTools {
			found := false
			if t.name == "fzf" {
				found = fzfHasWalker(fzfVersion)
			} else if t.name == "playwright-chromium" {
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
				if t.name == "fzf" {
					details = "fzf " + fzfVersion + " (built-in directory walker)"
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
				if t.name == "fzf" {
					if fzfVersion == "" {
						details = "fzf not found on PATH"
					} else {
						details = "fzf " + fzfVersion + " is older than 0.47 — no built-in directory walker (falls back to `find`)"
					}
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

			uc.auditSkillTree("CommandCode", filepath.Join(ccConfigDir, "skills"), addDiag)
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

// gamingKernelParams are the performance kernel parameters the gaming stack
// expects on /proc/cmdline. mitigations=off is deliberately NOT managed here:
// it is a Spectre/Meltdown trade-off that stays a manual, approved decision.
var gamingKernelParams = []string{"preempt=full", "split_lock_detect=off", "zswap.enabled=0"}

// gamingServices are the daemons the gaming stack needs active.
var gamingServices = []string{"scx_loader", "lactd", "ananicy-cpp", "power-profiles-daemon"}

// missingCmdlineParams returns the wanted kernel parameters absent from cmdline.
func missingCmdlineParams(cmdline string, wanted []string) []string {
	var missing []string
	for _, w := range wanted {
		if !strings.Contains(cmdline, w) {
			missing = append(missing, w)
		}
	}
	return missing
}

// multilibEnabled reports whether the [multilib] repo is active in pacman.conf
// (Steam and lib32-* packages require it).
func multilibEnabled(pacmanConf string) bool {
	for _, line := range strings.Split(pacmanConf, "\n") {
		if strings.TrimSpace(line) == "[multilib]" {
			return true
		}
	}
	return false
}

// auditGamingStack audits the opt-in gaming manifest (gaming.yaml) plus the
// tuning state the manifests cannot express (services, kernel cmdline,
// scheduler, GPU driver, provisioned presets). Everything tuning-related is
// warn-only: privileged files (/etc, the cmdline source) are never auto-fixed.
func (uc *DoctorAuditUseCase) auditGamingStack(ctx context.Context, addDiag func(entity.Diagnostic)) {
	if runtime.GOOS != "linux" || !entity.MatchesOS("arch,cachyos") {
		return
	}
	pkgs, err := uc.manifestRepo.LoadGamingPackages()
	if err != nil || len(pkgs) == 0 {
		return
	}
	// Presence gate: Steam installed means the owner opted into the stack.
	// Without it (or without a usable package manager) stay silent.
	steamOptedIn := false
	for _, pkg := range pkgs {
		if pkg.ID != "steam" || !entity.MatchesOS(pkg.OS) {
			continue
		}
		mgr, ok := uc.managers[pkg.Type]
		if !ok || !mgr.IsAvailable(ctx) {
			return
		}
		installed, _, gateErr := mgr.IsInstalled(ctx, pkg)
		if gateErr != nil || !installed {
			return
		}
		steamOptedIn = true
		break
	}
	if !steamOptedIn {
		return
	}
	for _, pkg := range pkgs {
		if !entity.MatchesOS(pkg.OS) {
			continue
		}
		mgr, ok := uc.managers[pkg.Type]
		if !ok || !mgr.IsAvailable(ctx) {
			continue
		}
		installed, info, checkErr := mgr.IsInstalled(ctx, pkg)
		switch {
		case checkErr != nil:
			addDiag(entity.Diagnostic{
				Category: entity.DiagWarning,
				System:   "Gaming",
				Target:   pkg.ID,
				Details:  fmt.Sprintf("Check failed: %v", checkErr),
				FixHint:  "run 'envctl run gaming' to install the missing gaming packages",
			})
		case !installed:
			addDiag(entity.Diagnostic{
				Category: entity.DiagWarning,
				System:   "Gaming",
				Target:   pkg.ID,
				Details:  "Not installed (gaming stack opted in via Steam)",
				FixHint:  "run 'envctl run gaming' to install the missing gaming packages",
			})
		default:
			addDiag(entity.Diagnostic{
				Category: entity.DiagOK,
				System:   "Gaming",
				Target:   pkg.ID,
				Details:  fmt.Sprintf("Installed (%s)", info),
			})
		}
	}
	uc.auditGamingTuning(ctx, addDiag)
}

// auditGamingTuning checks the read-only tuning state behind the gaming stack.
// Every finding is warn-only with a manual fix hint: the sources live in
// privileged files or need a reboot, so `doctor --fix` must never touch them.
func (uc *DoctorAuditUseCase) auditGamingTuning(ctx context.Context, addDiag func(entity.Diagnostic)) {
	if systemctl, err := exec.LookPath("systemctl"); err == nil {
		for _, svc := range gamingServices {
			if err := exec.CommandContext(ctx, systemctl, "is-active", svc).Run(); err != nil {
				addDiag(entity.Diagnostic{
					Category: entity.DiagWarning,
					System:   "Gaming",
					Target:   "service " + svc,
					Details:  "Service is not active",
					FixHint:  fmt.Sprintf("run 'systemctl enable --now %s' in your own terminal (password required)", svc),
				})
			} else {
				addDiag(entity.Diagnostic{
					Category: entity.DiagOK,
					System:   "Gaming",
					Target:   "service " + svc,
					Details:  "Service is active",
				})
			}
		}
	}
	if data, err := os.ReadFile("/sys/kernel/sched_ext/state"); err == nil {
		if state := strings.TrimSpace(string(data)); state != "enabled" {
			addDiag(entity.Diagnostic{
				Category: entity.DiagWarning,
				System:   "Gaming",
				Target:   "sched_ext",
				Details:  fmt.Sprintf("sched_ext state is '%s', scx schedulers are inert", state),
				FixHint:  "run 'systemctl enable --now scx_loader' in your own terminal (password required)",
			})
		} else {
			addDiag(entity.Diagnostic{
				Category: entity.DiagOK,
				System:   "Gaming",
				Target:   "sched_ext",
				Details:  "sched_ext enabled (scx scheduler running)",
			})
		}
	}
	if data, err := os.ReadFile("/proc/cmdline"); err == nil {
		if missing := missingCmdlineParams(string(data), gamingKernelParams); len(missing) > 0 {
			addDiag(entity.Diagnostic{
				Category: entity.DiagWarning,
				System:   "Gaming",
				Target:   "kernel cmdline",
				Details:  fmt.Sprintf("Missing performance parameters: %s", strings.Join(missing, ", ")),
				FixHint:  "edit KERNEL_CMDLINE in /etc/default/limine, run 'limine-update' and reboot (see skill cachyos-gaming-setup; password required)",
			})
		} else {
			addDiag(entity.Diagnostic{
				Category: entity.DiagOK,
				System:   "Gaming",
				Target:   "kernel cmdline",
				Details:  "Performance parameters present (preempt, split_lock, zswap)",
			})
		}
	}
	if vulkaninfo, err := exec.LookPath("vulkaninfo"); err == nil {
		if out, err := exec.CommandContext(ctx, vulkaninfo).Output(); err != nil || !strings.Contains(string(out), "RADV") {
			addDiag(entity.Diagnostic{
				Category: entity.DiagWarning,
				System:   "Gaming",
				Target:   "Vulkan driver",
				Details:  "RADV not reported by vulkaninfo (Polaris must stay on RADV, never AMDVLK)",
				FixHint:  "check 'vulkaninfo | grep RADV' (see skill cachyos-gaming-setup)",
			})
		} else {
			addDiag(entity.Diagnostic{
				Category: entity.DiagOK,
				System:   "Gaming",
				Target:   "Vulkan driver",
				Details:  "RADV active",
			})
		}
	}
	if gamingConf, err := uc.fsManager.ExpandUserPath("~/.config/environment.d/gaming.conf"); err == nil && gamingConf != "" {
		if data, err := uc.fsManager.ReadFile(gamingConf); err != nil || !strings.Contains(string(data), "MESA_SHADER_CACHE_MAX_SIZE=") {
			addDiag(entity.Diagnostic{
				Category: entity.DiagWarning,
				System:   "Gaming",
				Target:   "shader cache preset",
				Details:  "~/.config/environment.d/gaming.conf misses MESA_SHADER_CACHE_MAX_SIZE",
				FixHint:  "run 'envctl run shell' to seed the gaming environment preset, then relog",
			})
		} else {
			addDiag(entity.Diagnostic{
				Category: entity.DiagOK,
				System:   "Gaming",
				Target:   "shader cache preset",
				Details:  "gaming.conf shader cache configured",
			})
		}
	}
	if !uc.fsManager.Exists("/usr/bin/X") {
		addDiag(entity.Diagnostic{
			Category: entity.DiagWarning,
			System:   "Gaming",
			Target:   "X11 session",
			Details:  "/usr/bin/X missing (plasma-x11-session cannot start without xorg-server)",
			FixHint:  "run 'envctl run gaming' to install xorg-server",
		})
	} else {
		addDiag(entity.Diagnostic{
			Category: entity.DiagOK,
			System:   "Gaming",
			Target:   "X11 session",
			Details:  "/usr/bin/X present",
		})
	}
	if data, err := os.ReadFile("/etc/pacman.conf"); err == nil {
		if !multilibEnabled(string(data)) {
			addDiag(entity.Diagnostic{
				Category: entity.DiagWarning,
				System:   "Gaming",
				Target:   "multilib repo",
				Details:  "[multilib] not enabled in /etc/pacman.conf (Steam and lib32-* require it)",
				FixHint:  "uncomment [multilib] in /etc/pacman.conf in your own terminal (password required)",
			})
		} else {
			addDiag(entity.Diagnostic{
				Category: entity.DiagOK,
				System:   "Gaming",
				Target:   "multilib repo",
				Details:  "[multilib] enabled",
			})
		}
	}
}

// auditDebloat checks the opt-in Windows debloat stack (debloat.yaml) and
// reports one aggregated line per category. Drift is DiagInfo, never a
// warning: the stack only applies when the owner explicitly runs
// `run debloat`, so an unapplied tweak is not a health problem. Windows-only;
// anywhere else the section stays silent.
func (uc *DoctorAuditUseCase) auditDebloat(ctx context.Context, addDiag func(entity.Diagnostic)) {
	if runtime.GOOS != "windows" || uc.tweaksManager == nil {
		return
	}
	tweaks, err := uc.manifestRepo.LoadDebloatTweaks()
	if err != nil || len(tweaks) == 0 {
		return
	}
	type catStat struct {
		total   int
		applied int
	}
	order := []string{}
	stats := map[string]*catStat{}
	checks := uc.tweaksManager.CheckBatch(ctx, tweaks)
	for _, c := range checks {
		tw := c.Tweak
		st, ok := stats[tw.Category]
		if !ok {
			st = &catStat{}
			stats[tw.Category] = st
			order = append(order, tw.Category)
		}
		st.total++
		if c.Err == nil && c.OK {
			st.applied++
		}
	}
	for _, cat := range order {
		st := stats[cat]
		if st.applied == st.total {
			addDiag(entity.Diagnostic{
				Category: entity.DiagOK,
				System:   "Debloat",
				Target:   "category " + cat,
				Details:  fmt.Sprintf("Fully applied (%d/%d)", st.applied, st.total),
			})
		} else {
			addDiag(entity.Diagnostic{
				Category: entity.DiagInfo,
				System:   "Debloat",
				Target:   "category " + cat,
				Details:  fmt.Sprintf("Opt-in stack: %d/%d applied", st.applied, st.total),
				FixHint:  "run 'envctl run debloat' as Administrator to apply (opt-in, never auto-fixed)",
			})
		}
	}
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

// auditOpenCodeVersionSkew warns when the installed opencode CLI predates the
// V2-native config the repo deploys (review the 2026-09-22 migration): a v1
// binary discards unknown keys (strict()), inverts MCP flags and fails V2
// plugins with LoadError, while doctor would otherwise stay green.
func (uc *DoctorAuditUseCase) auditOpenCodeVersionSkew(_ context.Context, addDiag func(entity.Diagnostic)) {
	resolved, err := resolveOnToolchainPath("opencode")
	if err != nil {
		return
	}
	out, err := runWithToolchain(context.Background(), resolved, "--version")
	if err != nil {
		return
	}
	version := firstVersionToken(strings.TrimSpace(out))
	if version == "" {
		return
	}
	if !versionMajorAtLeast(version, 2) {
		addDiag(entity.Diagnostic{
			Category: entity.DiagWarning,
			System:   "OpenCode",
			Target:   "Version skew",
			Details:  fmt.Sprintf("opencode v%s + V2 config (agents/permissions/plugins/skills/mcp.servers): v1 discards keys, inverts MCP flags and fails V2 plugins", version),
			FixHint:  "upgrade opencode to 2.x (envctl run providers), then 'opencode debug config'",
		})
	}
}

// removedMCPEntries lists MCP servers that envctl no longer provisions. A
// deployed config that still declares one is stale (user-edited or predating
// the removal) — flag it by name instead of hiding behind a generic drift diff.
var removedMCPEntries = []string{"zscan"}

// auditRemovedMCPEntries reports deployed agent configs that still declare
// MCP servers envctl removed (see removedMCPEntries). The generic ConfigFile
// drift check would only say "content diverges"; naming the stale entry tells
// the user exactly what to delete.
func (uc *DoctorAuditUseCase) auditRemovedMCPEntries(addDiag func(entity.Diagnostic)) {
	targets := []struct {
		path    string
		section string
		system  string
	}{
		{"~/.config/opencode/opencode.json", "mcp", "OpenCode"},
		{"~/.commandcode/mcp.json", "mcpServers", "CommandCode"},
	}
	for _, tgt := range targets {
		expanded, err := uc.fsManager.ExpandUserPath(tgt.path)
		if err != nil {
			continue
		}
		data, err := os.ReadFile(expanded)
		if err != nil {
			continue
		}
		var root map[string]json.RawMessage
		if err := json.Unmarshal(data, &root); err != nil {
			continue
		}
		var servers map[string]json.RawMessage
		if err := json.Unmarshal(root[tgt.section], &servers); err != nil {
			continue
		}
		var stale []string
		for _, name := range removedMCPEntries {
			if _, ok := servers[name]; ok {
				stale = append(stale, name)
			}
		}
		if len(stale) > 0 {
			addDiag(entity.Diagnostic{
				Category: entity.DiagWarning,
				System:   tgt.system,
				Target:   "Removed MCP entries",
				Details:  fmt.Sprintf("%s still declares removed MCP server(s): %s — envctl no longer provisions them", tgt.path, strings.Join(stale, ", ")),
				FixHint:  "run 'envctl run shell' to re-provision, or delete the stale block(s) by hand",
			})
		}
	}
}

// auditAgentsIdentityCoverage warns when no AGENTS.md manifest variant matches
// this host (e.g. an unmatched distro after the debian/ubuntu + arch/cachyos
// split): provisioning would silently skip the global rules file and doctor
// would otherwise stay green.
func (uc *DoctorAuditUseCase) auditAgentsIdentityCoverage(addDiag func(entity.Diagnostic), configFiles []entity.ConfigFile) {
	matched := false
	for _, cf := range configFiles {
		if !strings.HasSuffix(cf.Destination, "AGENTS.md") {
			continue
		}
		if entity.MatchesOS(cf.OS) {
			matched = true
			break
		}
	}
	if !matched {
		addDiag(entity.Diagnostic{
			Category: entity.DiagWarning,
			System:   "OpenCode",
			Target:   "AGENTS.md (identity coverage)",
			Details:  "no AGENTS.md manifest variant matches this OS/distro — global rules would not be provisioned",
			FixHint:  "add a matching manifest variant or report the distro gap",
		})
	}
}

// lspConnectionMarkers identifies a server that failed to bind its stdio
// transport (skill lsp-smoke-test: exit codes lie — node servers exit 1 on
// EOF when healthy; only the absence of a connection error proves health).
var lspConnectionMarkers = []string{
	"input stream is not set",
	"connection input stream",
	"no stdin",
	"stdin is not",
	"failed to bind",
}

// auditLSPHandshake runs every provisioned LSP with closed stdin and reports
// the ones that cannot bind their stdio transport. A server that starts and
// waits (killed by timeout) or exits quietly on EOF is healthy; only
// connection-error output fails the check.
func (uc *DoctorAuditUseCase) auditLSPHandshake(ctx context.Context, addDiag func(entity.Diagnostic)) {
	lsps, err := uc.manifestRepo.LoadLSPs()
	if err != nil {
		return
	}
	for _, lsp := range lsps {
		if !entity.MatchesOS(lsp.OS) || lsp.CheckBinary == "" {
			continue
		}
		if !toolAvailable(ctx, lsp.CheckBinary) {
			continue // missing binary already reported by the LSP presence check
		}
		timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		//nolint:gosec // G204: command/args come from the local manifests (same trust level as package installs), not from user input.
		cmd := exec.CommandContext(timeoutCtx, lsp.Command, lsp.Args...)
		devNull, err := os.Open(os.DevNull)
		if err != nil {
			cancel()
			continue
		}
		cmd.Stdin = devNull
		out, cmdErr := cmd.CombinedOutput()
		devNull.Close()
		cancel()
		if cmdErr != nil && len(out) == 0 {
			// Process died before producing output. Quiet exit is the
			// HEALTHY shape for stdio servers (node servers exit 1 on
			// healthy EOF — calibrated live across the 14 Linux LSPs), and
			// it is indistinguishable from a spawn crash at this layer, so
			// silence is the safe default: a warning here would flag every
			// healthy quiet server and break the 0 WARN/0 ERRO contract.
			// (SPEC Task 11.3a proposed a warning; rejected on this
			// evidence — see TestDoctorAudit_LSPHandshakeQuietExitPasses.)
			continue
		}
		lowered := strings.ToLower(string(out))
		for _, marker := range lspConnectionMarkers {
			if !strings.Contains(lowered, marker) {
				continue
			}
			snippet := strings.TrimSpace(string(out))
			if len(snippet) > 200 {
				snippet = snippet[:200] + "…"
			}
			nullDev := "/dev/null"
			if runtime.GOOS == "windows" {
				nullDev = "NUL"
			}
			repro := strings.TrimSpace(lsp.Command + " " + strings.Join(lsp.Args, " ") + " < " + nullDev)
			addDiag(entity.Diagnostic{
				Category: entity.DiagWarning,
				System:   "LSP",
				Target:   lsp.ServerName,
				Details:  fmt.Sprintf("stdio handshake failed (%s): %s", snippet, repro),
				FixHint:  fmt.Sprintf("run '%s' by hand; reinstall via 'envctl run lsp' (%s)", repro, lsp.InstallTarget),
			})
			break
		}
	}
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

// auditSkillTree validates one agent's deployed skills: presence first, then the
// frontmatter its loader reads. Both agents share this implementation because a
// skill whose frontmatter is rejected is silently ignored at runtime — an
// existence-only check would call that tree healthy.
func (uc *DoctorAuditUseCase) auditSkillTree(system, skillsDir string, addDiag func(entity.Diagnostic)) {
	if skillsDir == "" {
		return
	}
	if !uc.fsManager.Exists(skillsDir) {
		addDiag(entity.Diagnostic{
			Category: entity.DiagWarning,
			System:   system,
			Target:   "Skills",
			Details:  "Skills directory not found",
			FixHint:  "Run 'envctl run skills' to deploy skills",
		})
		return
	}

	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		addDiag(entity.Diagnostic{
			Category: entity.DiagWarning,
			System:   system,
			Target:   "Skills",
			Details:  fmt.Sprintf("Skills directory unreadable: %v", err),
			FixHint:  "Check the directory permissions, then run 'envctl run skills'",
		})
		return
	}

	deployed := 0
	var rejected []string
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
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
			System:   system,
			Target:   "Skills",
			Details:  fmt.Sprintf("%d of %d deployed skills will not load: %s", len(rejected), total, strings.Join(rejected, "; ")),
			FixHint:  "Fix the SKILL.md frontmatter (name must match the directory, description must be non-empty), then run 'envctl run skills'",
		})
	case deployed > 0:
		addDiag(entity.Diagnostic{
			Category: entity.DiagOK,
			System:   system,
			Target:   "Skills",
			Details:  fmt.Sprintf("%d skills deployed, frontmatter valid", deployed),
		})
	default:
		addDiag(entity.Diagnostic{
			Category: entity.DiagWarning,
			System:   system,
			Target:   "Skills",
			Details:  "Skills directory exists but holds no skill",
			FixHint:  "Run 'envctl run skills' to deploy the manifest skills",
		})
	}

	if total == 0 {
		return
	}
	manifestSkills, err := uc.manifestRepo.LoadSkills()
	if err != nil {
		return
	}
	expected := 0
	for _, s := range manifestSkills {
		if s.Enabled && s.AppliesToOS(runtime.GOOS) {
			expected++
		}
	}
	if expected > 0 && total != expected {
		addDiag(entity.Diagnostic{
			Category: entity.DiagWarning,
			System:   system,
			Target:   "Skills",
			Details:  fmt.Sprintf("%d skills deployed but the manifest declares %d", total, expected),
			FixHint:  "Run 'envctl run skills' to re-sync the deployed skills with the manifest",
		})
	}
}
