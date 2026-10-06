package usecase

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/eajdias/envctl/internal/domain/entity"
	"github.com/eajdias/envctl/internal/infra/executil"
)

// 1. Audit Environment Variables
//
//nolint:errcheck // audit continues with an empty list when the manifest fails to load.
func (uc *DoctorAuditUseCase) auditEnvVars(addDiag func(entity.Diagnostic)) {
	envVars, _ := uc.manifestRepo.LoadEnvVars()
	for _, ev := range envVars {
		if !entity.MatchesOS(ev.OS) {
			continue
		}
		val, err := uc.envManager.GetEnvVar(ev.Scope, ev.Name)
		if err != nil || val != ev.Value {
			addDiag(entity.Warn(
				"Environment",
				ev.Name,
				fmt.Sprintf("Current: '%s' | Expected: '%s' (Scope: %s)", val, ev.Value, ev.Scope),
				fmt.Sprintf("run 'envctl run shell' to apply %s=%s", ev.Name, ev.Value),
			))
		} else {
			addDiag(entity.OK(
				"Environment",
				ev.Name,
				fmt.Sprintf("Set to '%s' (Scope: %s)", val, ev.Scope),
			))
		}
	}
}

// 1.5. Audit ~/.local/bin on PATH (provisioned helpers like `pw` live here).
func (uc *DoctorAuditUseCase) auditLocalBinPATH(addDiag func(entity.Diagnostic)) {
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
				//nolint:gosec // G703: rc paths are fixed $HOME/.profile and $HOME/.bashrc, not user input.
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
			addDiag(entity.OK(
				"Environment",
				"PATH (~/.local/bin)",
				"~/.local/bin is on PATH (provisioned helpers resolve)",
			))
		} else {
			addDiag(entity.Warn(
				"Environment",
				"PATH (~/.local/bin)",
				"~/.local/bin is not on PATH — provisioned helpers (pw) do not resolve by bare name",
				"run 'envctl run shell' to prepend ~/.local/bin to the user PATH",
			))
		}
	}

	// 1.6. Audit that envctl itself resolves on PATH. A fresh machine out of
	// bootstrap runs `envctl doctor` from a shell the bootstrap cannot refresh,
	// so the audit (not the install) is what confirms the binary is reachable.
	if envctlPath, err := exec.LookPath("envctl"); err != nil || envctlPath == "" {
		addDiag(entity.Warn(
			"Envctl",
			"PATH",
			"envctl is not on PATH — the bootstrap install dir was not persisted",
			"run 'bootstrap.ps1 -Force' (Windows) or add the install dir to PATH, then open a new shell",
		))
	} else {
		addDiag(entity.OK(
			"Envctl",
			"PATH",
			fmt.Sprintf("envctl resolves at %s", envctlPath),
		))
	}
}

// 2. Audit Git Global Configurations
//
//nolint:errcheck // audit continues with an empty list when the manifest fails to load.
func (uc *DoctorAuditUseCase) auditGitConfigs(ctx context.Context, addDiag func(entity.Diagnostic)) {
	gitConfigs, _ := uc.manifestRepo.LoadGitConfigs()
	for _, gc := range gitConfigs {
		if !entity.MatchesOS(gc.OS) {
			continue
		}
		val, err := uc.gitManager.GetGlobalConfig(ctx, gc.Key)
		if err != nil || val != gc.Value {
			addDiag(entity.Warn(
				"Git",
				gc.Key,
				fmt.Sprintf("Current: '%s' | Expected: '%s'", val, gc.Value),
				fmt.Sprintf("git config --global %s %s", gc.Key, gc.Value),
			))
		} else {
			addDiag(entity.OK(
				"Git",
				gc.Key,
				fmt.Sprintf("Configured: %s", val),
			))
		}
	}
}

// 3. Audit Config Files
//
//nolint:errcheck // audit continues with an empty list when the manifest fails to load.
func (uc *DoctorAuditUseCase) auditConfigFiles(addDiag func(entity.Diagnostic)) {
	configFiles, _ := uc.manifestRepo.LoadConfigFiles()
	for _, cf := range configFiles {
		if !entity.MatchesOS(cf.OS) {
			continue
		}
		if !uc.fsManager.Exists(cf.Destination) {
			addDiag(entity.Error(
				"ConfigFile",
				cf.Destination,
				"File missing on filesystem",
				"run 'envctl run shell'",
			))
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
						addDiag(entity.Warn(
							"ConfigFile",
							cf.Destination,
							"Content diverges from provisioned source",
							"run 'envctl run shell' to restore the provisioned content",
						))
						continue
					}
				}
			}
			addDiag(entity.OK(
				"ConfigFile",
				cf.Destination,
				details,
			))
		}
	}
}

// 8. Audit Windows 11 Registry Tweaks, Features & Fonts (Windows only)
func (uc *DoctorAuditUseCase) auditWindowsTweaks(ctx context.Context, addDiag func(entity.Diagnostic)) {
	if runtime.GOOS == "windows" && uc.tweaksManager != nil {
		//nolint:errcheck // audit continues with an empty list when the manifest fails to load.
		tweaks, _ := uc.manifestRepo.LoadWindowsTweaks()
		for _, tw := range tweaks {
			targetName := fmt.Sprintf("%s\\%s", tw.Path, tw.Name)
			if tw.Path == "" {
				targetName = fmt.Sprintf("[%s] %s", tw.Type, tw.Name)
			}
			ok, details, err := uc.tweaksManager.CheckTweak(ctx, tw)
			if err != nil || !ok {
				addDiag(entity.Warn(
					"Windows11",
					targetName,
					details,
					"run 'envctl run windows'",
				))
			} else {
				addDiag(entity.OK(
					"Windows11",
					targetName,
					details,
				))
			}
		}
	}

	// 8b. Audit opt-in Windows debloat (debloat.yaml). Never WARN: the stack
	// only applies via `run debloat`, so drift is informational. One line per
	// category keeps the report compact; per-tweak detail lives in the
	// `run debloat` output itself.
	uc.auditDebloat(ctx, addDiag)
}

// 10. Audit Git Worktree Support
// `git worktree list` exits 128 outside a git repository, which is expected
// and not a fault of the git installation. Only run the command from inside a repo.
func (uc *DoctorAuditUseCase) auditWorktreeSupport(ctx context.Context, addDiag func(entity.Diagnostic)) {
	if _, err := exec.LookPath("git"); err != nil {
		addDiag(entity.Warn(
			"Git",
			"git worktree",
			"git binary not found in PATH",
			"install git (winget install Git.Git / apt-get install -y git)",
		))
	} else if err := exec.CommandContext(ctx, "git", "rev-parse", "--is-inside-work-tree").Run(); err != nil {
		addDiag(entity.OK(
			"Git",
			"git worktree",
			"git worktree supported (command not run: current directory is not inside a git repository)",
		))
	} else if out, err := exec.CommandContext(ctx, "git", "worktree", "list", "--porcelain").CombinedOutput(); err != nil {
		addDiag(entity.Warn(
			"Git",
			"git worktree",
			fmt.Sprintf("Worktree check failed: %v", err),
			"Ensure git is installed and updated",
		))
	} else {
		addDiag(entity.OK(
			"Git",
			"git worktree",
			"Worktree command supported and active",
		))

		worktrees, parseErr := parseWorktreeListPorcelain(string(out))
		if parseErr != nil {
			addDiag(entity.Warn(
				"Git",
				"worktree report",
				fmt.Sprintf("Could not parse 'git worktree list --porcelain': %v", parseErr),
				"inspect the worktree list manually; do not prune or remove entries automatically",
			))
		} else {
			for _, diagnostic := range worktreeFindings(worktrees) {
				addDiag(diagnostic)
			}
		}
	}
}

// 10.5 Audit the local verification wiring: the same gates run by the
// CommandCode Stop hook and the git pre-push hook. Without them a broken
// tree only surfaces after the push, which is what this wiring exists to
// prevent.
func (uc *DoctorAuditUseCase) auditVerifyWiring(addDiag func(entity.Diagnostic)) {
	verifierPath, verifierErr := uc.fsManager.ExpandUserPath("~/.local/bin/envctl-verify")
	prePushPath, prePushErr := uc.fsManager.ExpandUserPath("~/.config/git/hooks/pre-push")
	if verifierErr != nil || prePushErr != nil {
		uc.logger.Warn("Could not resolve the verification paths: %v / %v", verifierErr, prePushErr)
	}
	verifierReady := executil.IsExecutableFile(verifierPath)
	prePushReady := executil.IsExecutableFile(prePushPath)

	switch {
	case verifierReady && prePushReady:
		addDiag(entity.OK(
			"Verify",
			"local quality gates",
			"envctl-verify deployed and the git pre-push hook is executable",
		))
	case verifierReady:
		addDiag(entity.Warn(
			"Verify",
			"git pre-push hook",
			"pre-push hook missing or not executable — pushes are not gated locally",
			"run 'envctl run shell' to deploy ~/.config/git/hooks/pre-push",
		))
	default:
		addDiag(entity.Warn(
			"Verify",
			"envctl-verify",
			"local verifier not deployed — lint/test failures surface only in CI",
			"run 'envctl run shell' to deploy ~/.local/bin/envctl-verify",
		))
	}
}

// 11.5. Audit WSL Ubuntu secondary shell (Windows only)
func (uc *DoctorAuditUseCase) auditWSLSecondary(ctx context.Context, addDiag func(entity.Diagnostic)) {
	if runtime.GOOS == "windows" {
		out, err := exec.CommandContext(ctx, "wsl.exe", "-l", "-q").CombinedOutput()
		// Windows console output is UTF-16: strip null bytes before matching.
		clean := strings.ReplaceAll(string(out), "\x00", "")
		if err != nil {
			addDiag(entity.Warn(
				"WSL",
				"Ubuntu",
				fmt.Sprintf("wsl.exe check failed: %v", err),
				"run 'wsl --install -d Ubuntu' (WSL2)",
			))
		} else if !strings.Contains(strings.ToLower(clean), "ubuntu") {
			addDiag(entity.Warn(
				"WSL",
				"Ubuntu",
				"WSL Ubuntu distro not found (secondary POSIX shell)",
				"run 'wsl --install -d Ubuntu' (WSL2)",
			))
		} else {
			addDiag(entity.OK(
				"WSL",
				"Ubuntu",
				"WSL Ubuntu available as secondary POSIX shell",
			))
		}
	}
}

// 11.6. Audit Windows console code page (UTF-8 required for Unicode glyph rendering)
func (uc *DoctorAuditUseCase) auditConsoleCodePage(ctx context.Context, addDiag func(entity.Diagnostic)) {
	if runtime.GOOS == "windows" {
		out, err := exec.CommandContext(ctx, "chcp").CombinedOutput()
		codePage := strings.TrimSpace(regexp.MustCompile(`\d+`).FindString(string(out)))
		if err != nil {
			addDiag(entity.Warn(
				"Console",
				"Code Page",
				fmt.Sprintf("chcp check failed: %v", err),
				"run 'envctl run shell' to reapply the UTF-8 PowerShell profile",
			))
		} else if codePage != "65001" {
			addDiag(entity.Warn(
				"Console",
				"Code Page",
				fmt.Sprintf("Console code page is %s — Unicode glyphs (emojis, checkmarks) render as U+FFFD", codePage),
				"restart Windows Terminal (its profile now sets chcp 65001) or run 'envctl run shell' to reapply the PowerShell profile",
			))
		} else {
			addDiag(entity.OK(
				"Console",
				"Code Page",
				"Console code page is UTF-8 (65001) — Unicode output safe",
			))
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
			addDiag(entity.Warn(
				"Console",
				"System Code Page",
				fmt.Sprintf("system code page check failed: %v", regErr),
				"enable 'Beta: Use Unicode UTF-8 for worldwide language support' (Settings > Time & Language > Language & region > Administrative language settings > Change system locale), then reboot",
			))
		} else if acp != "65001" || oemcp != "65001" {
			addDiag(entity.Warn(
				"Console",
				"System Code Page",
				fmt.Sprintf("System ACP/OEMCP is %s/%s — processes spawned without the PowerShell profile (opencode shell tool, cmd.exe, services) still emit CP%s and render as U+FFFD", acp, oemcp, oemcp),
				"enable 'Beta: Use Unicode UTF-8 for worldwide language support' (Settings > Time & Language > Language & region > Administrative language settings > Change system locale) or set HKLM\\SYSTEM\\CurrentControlSet\\Control\\Nls\\CodePage ACP/OEMCP=65001, then reboot",
			))
		} else {
			addDiag(entity.OK(
				"Console",
				"System Code Page",
				"System ACP/OEMCP is UTF-8 (65001) — Unicode safe for every process",
			))
		}
	}
}

// 12. Audit OpenCode storage accumulation & standardized temp folder
func (uc *DoctorAuditUseCase) auditTempAndScratch(addDiag func(entity.Diagnostic)) {
	homeDir, homeErr := uc.fsManager.ExpandUserPath("~")
	if homeErr != nil {
		uc.logger.Warn("Could not resolve the user home for the OpenCode store audit: %v", homeErr)
	}
	opencodeDataDir := openCodeDataDir(homeDir)

	dbPath := openCodeStorePath(homeDir)
	store, storeErr := InspectOpenCodeStore(dbPath)
	switch {
	case storeErr == nil && store.ExceedsThreshold():
		addDiag(entity.Warn(
			"OpenCode",
			"Database",
			fmt.Sprintf("opencode.db is %.1f MB (threshold: %d MB); %.1f MB reclaimable by VACUUM — the rest is live session history",
				float64(store.SizeBytes)/(1024*1024), openCodeStoreWarnBytes/(1024*1024), float64(store.ReclaimableBytes)/(1024*1024)),
			"run 'envctl run cleanup' to reclaim free pages; shrinking further means pruning sessions in OpenCode",
		))
	case storeErr == nil:
		addDiag(entity.OK(
			"OpenCode",
			"Database",
			fmt.Sprintf("Database OK (%.1f MB)", float64(store.SizeBytes)/(1024*1024)),
		))
	default:
		addDiag(entity.OK(
			"OpenCode",
			"Database",
			"Database not found (no opencode.db — clean state)",
		))
	}

	toolOutputDir := filepath.Join(opencodeDataDir, "tool-output")
	if toolSize := dirSize(toolOutputDir); toolSize > 50*1024*1024 {
		addDiag(entity.Warn(
			"OpenCode",
			"Tool Output",
			fmt.Sprintf("tool-output is %.1f MB (threshold: 50 MB)", float64(toolSize)/(1024*1024)),
			"run 'envctl run cleanup' to remove files >10 MB",
		))
	}

	// Stale config: standard config file is opencode.json; jsonc is a legacy conflict source.
	for _, stale := range []string{
		"~/.config/opencode/opencode.jsonc",
		"~/.config/opencode/opencode.linux.jsonc",
	} {
		if uc.fsManager.Exists(stale) {
			addDiag(entity.Warn(
				"OpenCode",
				stale,
				"Legacy config file found — arrays don't merge with opencode.json, causing duplicate LSPs/plugins",
				"run 'envctl run shell' to remove it (cleanup step)",
			))
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
		addDiag(entity.Warn(
			"TempFolder",
			tempDir,
			"Standardized agent temp folder (ENVCTL_TEMP) missing",
			"run 'envctl run shell' to create it",
		))
	} else if tempSize := dirSize(tempDir); tempSize > 500*1024*1024 {
		// A large temp folder is only a warning when the agent's own scratch is
		// the dominant owner. Third-party caches (Docker Desktop, WinGet, Brave
		// updaters) are regenerable by their owning app and should never keep
		// the doctor red — otherwise the operator learns to ignore the warning.
		dominant := dominantTempOwner(tempDir)
		category := entity.DiagInfo
		details := fmt.Sprintf("Agent temp folder is %.1f MB, mostly third-party caches — regenerable by their owning apps", float64(tempSize)/(1024*1024))
		if dominant == tempOwnerScratch {
			category = entity.DiagWarning
			details = fmt.Sprintf("Agent temp folder is %.1f MB — clean stale scratch", float64(tempSize)/(1024*1024))
		}
		addDiag(entity.Diagnostic{
			Category: category,
			System:   "TempFolder",
			Target:   tempDir,
			Details:  details,
			FixHint:  "run 'envctl run cleanup' or delete its contents manually",
		})
	} else {
		addDiag(entity.OK(
			"TempFolder",
			tempDir,
			"Standardized agent temp folder present (ENVCTL_TEMP)",
		))
	}
}

func (uc *DoctorAuditUseCase) auditEnvctlFreshness(ctx context.Context, addDiag func(entity.Diagnostic)) {
	if uc.envctlVersion == "" || uc.gitDescribe == nil {
		return
	}
	repoDir, err := repoRootFinder()
	if err != nil {
		return
	}
	describe, err := uc.gitDescribe(ctx, repoDir)
	if err != nil || strings.TrimSpace(describe) == "" {
		return
	}
	if envctlBinaryFresh(uc.envctlVersion, describe) {
		return
	}
	addDiag(entity.Warn(
		"Envctl",
		"Binary freshness",
		fmt.Sprintf("running binary reports %s but the repo checkout is at %s — embedded templates are stale", printableVersion(uc.envctlVersion), describe),
		`rebuild: go build -ldflags "-X main.Version=$(git describe --tags --always)" -o envctl.exe ./cmd/envctl`,
	))
}

// envctlBinaryFresh reports whether the running binary matches the repo describe.
// A "dev" build never matches a tagged checkout: an unversioned binary is the
// signature of a plain `go build` without the release ldflags, which is exactly
// the stale-template risk this check exists to catch.

func envctlBinaryFresh(builtVersion, repoDescribe string) bool {
	built := strings.TrimPrefix(strings.TrimSpace(builtVersion), "v")
	describe := strings.TrimPrefix(strings.TrimSpace(repoDescribe), "v")
	if built == "" || built == "dev" || describe == "" {
		return false
	}
	return built == describe
}

// findRepoRoot walks up from the working directory to the nearest .git, so the
// freshness audit compares against the checkout the user actually ran from.
// repoRootFinder is the package-level seam tests override to avoid depending on
// the host's working directory.

var repoRootFinder = findRepoRoot

func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no git checkout found above %s", dir)
		}
		dir = parent
	}
}

// removedMCPEntries lists MCP servers that envctl no longer provisions. A
// deployed config that still declares one is stale (user-edited or predating
// the removal) — flag it by name instead of hiding behind a generic drift diff.
