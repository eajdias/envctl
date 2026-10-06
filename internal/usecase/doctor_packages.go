package usecase

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/eajdias/envctl/internal/domain/entity"
	"github.com/eajdias/envctl/internal/infra/executil"
)

// 5. Audit Packages
//
//nolint:errcheck // audit continues with an empty list when the manifest fails to load.
func (uc *DoctorAuditUseCase) auditEnvPackages(ctx context.Context, addDiag func(entity.Diagnostic)) {
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

		//nolint:errcheck // an IsInstalled error is indistinguishable from "not installed" here; the diagnostic reflects it.
		installed, info, _ := mgr.IsInstalled(ctx, pkg)
		if !installed {
			addDiag(entity.Warn(
				string(pkg.Type),
				pkg.ID,
				"Not installed",
				fmt.Sprintf("run 'envctl run %s'", pkg.Type),
			))
		} else {
			addDiag(entity.OK(
				string(pkg.Type),
				pkg.ID,
				fmt.Sprintf("Installed (%s)", info),
			))
		}
	}
}

// 7. Audit LSPs
//
//nolint:errcheck // audit continues with an empty list when the manifest fails to load.
func (uc *DoctorAuditUseCase) auditLSPPresence(addDiag func(entity.Diagnostic)) {
	lsps, _ := uc.manifestRepo.LoadLSPs()
	for _, lsp := range lsps {
		if !entity.MatchesOS(lsp.OS) {
			continue
		}
		if lsp.CheckBinary != "" {
			if !toolAvailable(lsp.CheckBinary) {
				addDiag(entity.Warn(
					"LSP",
					lsp.ServerName,
					fmt.Sprintf("Binary '%s' not found in PATH", lsp.CheckBinary),
					fmt.Sprintf("run 'envctl run lsp' to install %s", lsp.InstallTarget),
				))
			} else {
				addDiag(entity.OK(
					"LSP",
					lsp.ServerName,
					fmt.Sprintf("Ready (%s in PATH)", lsp.CheckBinary),
				))
			}
		}
	}
}

// 9. Audit Browser & Playwright
// 9.1 Audit Google Chrome (native system browser for chrome-devtools MCP
// & CLI tools). Playwright CLI brings its own bundled Chromium, so a
// missing system Chrome only matters for chrome-devtools-mcp — downgrade
// to Info on Linux instead of warning on every headless VPS.
func (uc *DoctorAuditUseCase) auditBrowserStack(addDiag func(entity.Diagnostic)) {
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
		addDiag(entity.OK(
			"Browser",
			"Google Chrome",
			fmt.Sprintf("Google Chrome detected at %s", chromePath),
		))
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
		//nolint:errcheck // an unresolvable home yields an empty path and the check below is skipped.
		if homeDir, _ := uc.fsManager.ExpandUserPath("~"); homeDir != "" {
			pwWrapper = filepath.Join(homeDir, ".local", "bin", "pw.cjs")
		}
	} else {
		//nolint:errcheck // an unresolvable home yields an empty path and the check below is skipped.
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
			addDiag(entity.OK(
				"Browser",
				"Playwright Chromium",
				fmt.Sprintf("Bundled Chromium present in %s (playwright-cli ready)", browserCacheDir),
			))
		} else {
			addDiag(entity.Warn(
				"Browser",
				"Playwright Chromium",
				fmt.Sprintf("No bundled Chromium in %s — playwright-cli cannot launch a browser", browserCacheDir),
				"run 'bunx @playwright/cli@latest install-browser chromium'",
			))
		}
	}

	// 9.3 Audit pw wrapper: the hang-safe runner must be provisioned, or
	// agents fall back to raw playwright-cli and hang on Windows.
	if pwWrapper == "" {
		//nolint:errcheck // an unresolvable home yields an empty path and the check below is skipped.
		if homeDir, _ := uc.fsManager.ExpandUserPath("~"); homeDir != "" {
			pwWrapper = filepath.Join(homeDir, ".local", "bin", "pw.cjs")
		}
	}
	if pwWrapper != "" {
		if uc.fsManager.Exists(pwWrapper) {
			addDiag(entity.OK(
				"Browser",
				"pw wrapper",
				fmt.Sprintf("Hang-safe runner present at %s", pwWrapper),
			))
		} else {
			addDiag(entity.Warn(
				"Browser",
				"pw wrapper",
				"pw wrapper missing — agents have no hang-safe playwright-cli path on Windows",
				"run 'envctl run shell' to provision ~/.local/bin/pw.cjs",
			))
		}
	}
}

// 11. Audit Linux Toolchain Bootstrap (Linux only)
func (uc *DoctorAuditUseCase) auditLinuxToolchain(ctx context.Context, addDiag func(entity.Diagnostic)) {
	if runtime.GOOS == "linux" {
		userHomeDir, homeErr := uc.fsManager.ExpandUserPath("~")
		if homeErr != nil {
			uc.logger.Warn("Could not expand the home directory for the Linux toolchain audit: %v", homeErr)
		}
		env := executil.ToolchainEnv(userHomeDir)
		bootstrapTools := []struct {
			name string
			desc string
		}{
			{"mise", "mise dev-tool manager"},
			{"node", "Node.js (via mise)"},
			{"opencode", "OpenCode CLI"},
			{"cmdc", "CommandCode CLI"},
			{"gh", "GitHub CLI"},
			{"delta", "git-delta pager"},
			{"yq", "yq YAML/JSON processor"},
			{"uv", "uv Python package manager"},
			{"ruff", "ruff linter (via uv)"},
			{"fd", "fd (fdfind symlink)"},
			{"stylelint", "Stylelint CSS/SCSS linter (via npm)"},
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
					//nolint:gosec // G204: t.name is a manifest-declared tool name, not user input.
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
				addDiag(entity.OK(
					"LinuxBootstrap",
					t.desc,
					details,
				))
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
				addDiag(entity.Warn(
					"LinuxBootstrap",
					t.desc,
					details,
					"run 'envctl run bootstrap'",
				))
			}
		}
	}
}

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
		if !toolAvailable(lsp.CheckBinary) {
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
			addDiag(entity.Warn(
				"LSP",
				lsp.ServerName,
				fmt.Sprintf("stdio handshake failed (%s): %s", snippet, repro),
				fmt.Sprintf("run '%s' by hand; reinstall via 'envctl run lsp' (%s)", repro, lsp.InstallTarget),
			))
			break
		}
	}
}

// auditCommandCodeAgents validates every custom agent definition under
// <ccConfigDir>/agents, so a broken file cannot silently disable delegation.
// Two failure shapes are covered: a file that does not load at all (missing or
// mismatched frontmatter name) and a file that loads with a schema value the
// runtime will ignore, which quietly strips the agent of capabilities.
// Reserved names are skipped: CommandCode owns those and ignores a custom file.
