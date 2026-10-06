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
	"github.com/eajdias/envctl/internal/infra/embedded"
	"github.com/eajdias/envctl/internal/infra/environment"
	"github.com/eajdias/envctl/internal/infra/executil"
	"github.com/eajdias/envctl/internal/infra/filesystem"
)

// ProvisionBootstrapUseCase installs the Linux toolchain required to replicate
// the global OpenCode and CommandCode environments on Ubuntu servers: Volta +
// Node, the OpenCode CLI, the CommandCode CLI, and the user-local CLI tools
// (gh, delta, yq, uv, ruff, stylelint, golangci-lint, fd).
// It is a no-op on Windows, where winget/volta packages cover the toolchain.
type ProvisionBootstrapUseCase struct {
	fsManager    *filesystem.FileSystemManager
	envManager   *environment.WindowsEnvManager
	manifestRepo *embedded.ManifestRepository
	managers     map[entity.PackageType]repository.PackageManager
	logger       repository.Logger
}

// BootstrapResult holds the diagnostics produced during bootstrap provisioning.
type BootstrapResult struct {
	Diagnostics []entity.Diagnostic
}

// NewProvisionBootstrapUseCase builds the Linux toolchain bootstrap use case.
func NewProvisionBootstrapUseCase(fsManager *filesystem.FileSystemManager, manifestRepo *embedded.ManifestRepository, envManager *environment.WindowsEnvManager, managers map[entity.PackageType]repository.PackageManager, logger repository.Logger) *ProvisionBootstrapUseCase {
	return &ProvisionBootstrapUseCase{fsManager: fsManager, envManager: envManager, manifestRepo: manifestRepo, managers: managers, logger: logger}
}

// userHome expands ~ to the current user's home directory.
func (uc *ProvisionBootstrapUseCase) userHome() string {
	//nolint:errcheck // an unresolvable home falls back to the toolchain env built from "".
	home, _ := uc.fsManager.ExpandUserPath("~")
	return home
}

// shellEnv returns an environment that makes Volta shims (~/.volta/bin) and
// user-local binaries (~/.local/bin) available on PATH, without mutating the
// process environment.
func (uc *ProvisionBootstrapUseCase) shellEnv() []string {
	return executil.ToolchainEnv(uc.userHome())
}

// toolAvailable reports whether a binary resolves on the platform PATH. On
// Linux it additionally resolves the toolchain dirs, which are not part of a
// non-login process PATH. Filesystem lookup replaces the previous
// `bash -lc "command -v"` spawn: same verdict for binary names, no subshell.
func toolAvailable(name string) bool {
	if runtime.GOOS == "linux" {
		if home, err := os.UserHomeDir(); err == nil && home != "" {
			_, err := executil.LookPathIn(executil.ToolchainPath(home), name)
			return err == nil
		}
	}
	_, err := exec.LookPath(name)
	return err == nil
}

// ensureProcessToolchainPath mutates the process environment so that Volta
// shims, user-local binaries and Go are resolvable by subsequent provisioning
// steps running in the same process.
func (uc *ProvisionBootstrapUseCase) ensureProcessToolchainPath() {
	home := uc.userHome()
	if home == "" {
		return
	}
	dirs := executil.ToolchainDirs(home)
	cur := os.Getenv("PATH")
	if !strings.Contains(cur, dirs[0]) || !strings.Contains(cur, dirs[1]) || !strings.Contains(cur, dirs[2]) {
		os.Setenv("PATH", strings.Join(append(dirs, cur), string(os.PathListSeparator)))
	}
	if os.Getenv("VOLTA_HOME") == "" {
		os.Setenv("VOLTA_HOME", filepath.Join(home, ".volta"))
	}
	if os.Getenv("GOPATH") == "" {
		os.Setenv("GOPATH", filepath.Join(home, "go"))
	}
}

// runShell executes a bash script with the Volta-aware environment.
func (uc *ProvisionBootstrapUseCase) runShell(ctx context.Context, script string) (string, error) {
	cmd := exec.CommandContext(ctx, "bash", "-lc", script)
	cmd.Env = uc.shellEnv()
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

// runShellStdout runs a script and captures stdout only: shell-profile noise on
// stderr (a broken rc line, a missing shim) must not corrupt a parsed value.
func (uc *ProvisionBootstrapUseCase) runShellStdout(ctx context.Context, script string) (string, error) {
	cmd := exec.CommandContext(ctx, "bash", "-lc", script)
	cmd.Env = uc.shellEnv()
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}

// hasTool reports whether a binary is resolvable on the Volta-aware PATH.
func (uc *ProvisionBootstrapUseCase) hasTool(ctx context.Context, name string) bool {
	_, err := uc.runShell(ctx, "command -v "+name+" >/dev/null 2>&1")
	return err == nil
}

// ensureStep runs script unless installed() already holds, recording the
// done/applied diagnostics with the caller's wording. Tool installs and
// config writes report different states honestly ("installed" vs "present"),
// so the wording stays at the call edge while the predicate, the script
// runner and the idempotency log are shared. Either predicate shape fits:
// a binary lookup (hasTool) or a check command (doneCheck) — both answer
// "is there anything to do", which is all the runner needs.
func (uc *ProvisionBootstrapUseCase) ensureStep(ctx context.Context, result *BootstrapResult, target string, installed func() bool, script, doneDetails, appliedDetails string) {
	if installed() {
		uc.logger.LogIdempotency("LinuxBootstrap", target, true, doneDetails)
		result.Diagnostics = append(result.Diagnostics, entity.OK(
			"LinuxBootstrap",
			target,
			doneDetails,
		))
		return
	}

	if err := uc.installStep(ctx, result, target, target, script); err != nil {
		return
	}
	uc.logger.LogIdempotency("LinuxBootstrap", target, false, appliedDetails)
	result.Diagnostics = append(result.Diagnostics, entity.OK(
		"LinuxBootstrap",
		target,
		appliedDetails,
	))
}

// step installs a tool when missing, or reports it as already available.
func (uc *ProvisionBootstrapUseCase) step(ctx context.Context, result *BootstrapResult, name, target, installScript string) {
	uc.ensureStep(ctx, result, target, func() bool { return uc.hasTool(ctx, name) },
		installScript, "Already installed and available on PATH", "Installed successfully")
}

// pathStep persists toolchain dirs in the shell profiles through the single
// PATH owner (envManager.EnsurePathEntry). One target emits one diagnostic,
// however many dirs it covers; the textual diags stay identical to the old
// shell-script steps so the run log reads the same.
func (uc *ProvisionBootstrapUseCase) pathStep(ctx context.Context, result *BootstrapResult, target string, dirs ...string) {
	if uc.envManager == nil {
		return
	}
	changedAny := false
	for _, dir := range dirs {
		changed, err := uc.envManager.EnsurePathEntry(ctx, dir)
		if err != nil {
			uc.logger.Error("LinuxBootstrap: failed to persist %s: %v", target, err)
			result.Diagnostics = append(result.Diagnostics, entity.Warn(
				"LinuxBootstrap",
				target,
				fmt.Sprintf("could not persist %s in shell profiles: %v", dir, err),
				"Run 'envctl run shell' after installing the toolchain",
			))
			return
		}
		changedAny = changedAny || changed
	}
	details := "Already present in the shell profiles"
	if changedAny {
		details = "Written to the shell profiles"
	}
	uc.logger.LogIdempotency("LinuxBootstrap", target, !changedAny, details)
	result.Diagnostics = append(result.Diagnostics, entity.OK(
		"LinuxBootstrap",
		target,
		details,
	))
}

// installStep runs an installer and records only its failure. Callers that
// need a post-install version check can add their own success diagnostic after
// verifying the resulting binary.
func (uc *ProvisionBootstrapUseCase) installStep(ctx context.Context, result *BootstrapResult, name, target, installScript string) error {
	uc.logger.Info("LinuxBootstrap: installing %s (%s)", name, target)
	out, err := uc.runShell(ctx, installScript)
	if err != nil {
		msg := fmt.Sprintf("Installation failed: %v", err)
		if out != "" {
			msg += ": " + out
		}
		uc.logger.Error("LinuxBootstrap: failed to install %s: %s", target, msg)
		result.Diagnostics = append(result.Diagnostics, entity.Warn(
			"LinuxBootstrap",
			target,
			msg,
			"Run the install command manually as your user",
		))
	}
	return err
}

// ensureOpenCodeV2 converges the OpenCode CLI to the V2-native channel. A
// package-owned binary stays authoritative: stale user-local copies are
// archived, and the package is updated through its manager. Ubuntu/Debian
// user-local and non-package system copies use the official V2 installer.
func (uc *ProvisionBootstrapUseCase) ensureOpenCodeV2(ctx context.Context, result *BootstrapResult) {
	installed := uc.toolVersion(ctx, "opencode")
	source := installSource("opencode")
	pacmanOwns := uc.pacmanOwnsOpenCode(ctx)

	var ok bool
	installed, source, ok = archiveShadowedUserCopies(ctx,
		func(format string, args ...any) { uc.logger.Info("LinuxBootstrap: "+format, args...) },
		"OpenCode CLI", "opencode",
		"Move the stale user-local opencode binary aside, then run bootstrap again",
		"Move the stale user-local binary aside, then run bootstrap again",
		pacmanOwns, installed, source,
		func(target, details, fixHint string) {
			result.Diagnostics = append(result.Diagnostics, entity.Warn("LinuxBootstrap", target, details, fixHint))
		})
	if !ok {
		return
	}

	if versionMajorAtLeast(installed, 2) {
		if source != sourceSystem && !uc.ensureOpenCodeShellPath(ctx, result) {
			return
		}
		uc.logger.LogIdempotency("LinuxBootstrap", "OpenCode CLI", true, "already installed")
		result.Diagnostics = append(result.Diagnostics, entity.OK(
			"LinuxBootstrap",
			"OpenCode CLI",
			fmt.Sprintf("OpenCode %s is already installed and available on PATH", printableVersion(installed)),
		))
		return
	}

	if pacmanOwns {
		if !uc.installPacmanOpenCode(ctx, result) {
			return
		}
	} else if uc.pacmanManagerAvailable(ctx) || uc.hasTool(ctx, "pacman") {
		if !uc.installPacmanOpenCode(ctx, result) {
			return
		}
	} else if err := uc.installStep(ctx, result, "opencode", "OpenCode CLI", openCodeLinuxV2Installer); err != nil {
		return
	}

	after := uc.toolVersion(ctx, "opencode")
	source = installSource("opencode")
	if !versionMajorAtLeast(after, 2) {
		result.Diagnostics = append(result.Diagnostics, entity.Warn(
			"LinuxBootstrap",
			"OpenCode CLI",
			fmt.Sprintf("installer completed but OpenCode %s was not verified", printableVersion(after)),
			"Run 'curl -fsSL https://opencode.ai/v2/install | bash' and verify 'opencode --version'",
		))
		return
	}
	if source != sourceSystem && !uc.ensureOpenCodeShellPath(ctx, result) {
		return
	}
	uc.logger.LogIdempotency("LinuxBootstrap", "OpenCode CLI", false, "installed successfully")
	result.Diagnostics = append(result.Diagnostics, entity.OK(
		"LinuxBootstrap",
		"OpenCode CLI",
		fmt.Sprintf("Installed/updated OpenCode %s", printableVersion(after)),
	))
}

func (uc *ProvisionBootstrapUseCase) pacmanManagerAvailable(ctx context.Context) bool {
	mgr, ok := uc.managers[entity.PackageTypePacman]
	return ok && mgr.IsAvailable(ctx)
}

func (uc *ProvisionBootstrapUseCase) pacmanOwnsOpenCode(ctx context.Context) bool {
	if !uc.pacmanManagerAvailable(ctx) {
		return false
	}
	mgr := uc.managers[entity.PackageTypePacman]
	installed, _, err := mgr.IsInstalled(ctx, entity.Package{ID: "opencode", Type: entity.PackageTypePacman})
	return err == nil && installed
}

func (uc *ProvisionBootstrapUseCase) installPacmanOpenCode(ctx context.Context, result *BootstrapResult) bool {
	mgr, ok := uc.managers[entity.PackageTypePacman]
	if !ok || !mgr.IsAvailable(ctx) {
		result.Diagnostics = append(result.Diagnostics, entity.Warn(
			"LinuxBootstrap",
			"OpenCode CLI",
			"pacman is required for the distro-owned opencode package but is unavailable",
			"Restore pacman, then run bootstrap again",
		))
		return false
	}
	if err := mgr.Install(ctx, entity.Package{ID: "opencode", Type: entity.PackageTypePacman}); err != nil {
		result.Diagnostics = append(result.Diagnostics, entity.Warn(
			"LinuxBootstrap",
			"OpenCode CLI",
			fmt.Sprintf("pacman could not install/update opencode: %v", err),
			"Run 'envctl run pacman' to install or update opencode",
		))
		return false
	}
	return true
}

func (uc *ProvisionBootstrapUseCase) ensureOpenCodeShellPath(ctx context.Context, result *BootstrapResult) bool {
	if uc.envManager == nil {
		return true
	}
	if _, err := uc.envManager.EnsurePathEntry(ctx, openCodePathDir); err != nil {
		result.Diagnostics = append(result.Diagnostics, entity.Warn(
			"LinuxBootstrap",
			"OpenCode CLI PATH",
			fmt.Sprintf("could not persist ~/.opencode/bin in shell profiles: %v", err),
			"Run 'envctl run shell' after installing OpenCode",
		))
		return false
	}
	return true
}

// Execute provisions the Linux toolchain. On Windows it is a no-op.
func (uc *ProvisionBootstrapUseCase) Execute(ctx context.Context) (*BootstrapResult, error) {
	result := &BootstrapResult{}

	if runtime.GOOS != "linux" {
		result.Diagnostics = append(result.Diagnostics, entity.OK(
			"LinuxBootstrap",
			"toolchain bootstrap",
			"Skipped on Windows (toolchain provisioned via winget/volta packages)",
		))
		return result, nil
	}

	// Expose the Volta/user-local toolchain dirs to the current process so that
	// subsequent provisioning steps (shell npm install, LSP installs) can resolve
	// the binaries installed below.
	uc.ensureProcessToolchainPath()

	// 1. Volta (mandatory) - official installer.
	uc.step(ctx, result, "volta", "Volta JS toolchain manager",
		"curl -fsSL https://get.volta.sh | bash")

	// 2. Node.js LTS + pnpm via Volta (mirrors the Windows pin from packages.yaml). Idempotent.
	nodeSpec := "node@24.19.0"
	if pkgs, err := uc.manifestRepo.LoadPackages(); err == nil {
		for _, p := range pkgs {
			if p.Type == entity.PackageTypeVolta && strings.HasPrefix(p.ID, "node@") {
				nodeSpec = p.ID
				break
			}
		}
	}
	if uc.hasTool(ctx, "volta") {
		uc.logger.Info("LinuxBootstrap: ensuring Node.js %s + pnpm via Volta", nodeSpec)
		out, err := uc.runShell(ctx, "volta install "+nodeSpec+" pnpm")
		if err != nil {
			uc.logger.Error("LinuxBootstrap: volta install failed: %s (%s)", out, err)
			result.Diagnostics = append(result.Diagnostics, entity.Warn(
				"LinuxBootstrap",
				"Node.js + pnpm",
				fmt.Sprintf("volta install failed: %v (%s)", err, out),
				"Run 'volta install "+nodeSpec+" pnpm' manually",
			))
		} else {
			result.Diagnostics = append(result.Diagnostics, entity.OK(
				"LinuxBootstrap",
				"Node.js + pnpm",
				"Provisioned via Volta ("+nodeSpec+")",
			))
		}
	}

	// 2b. Expose Volta on interactive shells. get.volta.sh can skip rc-file
	// integration when run non-interactively, leaving volta off the PATH of
	// future login shells. The $HOME form keeps the entry portable and matches
	// what EnsurePathEntry's guard looks for; VOLTA_HOME itself is exported by
	// the toolchain env (shellEnv), so only the PATH entry is persisted here.
	if uc.hasTool(ctx, "volta") {
		uc.logger.Info("LinuxBootstrap: ensuring Volta is exported on interactive shells")
		uc.pathStep(ctx, result, "Volta shell integration", "$HOME/.volta/bin")
	}

	// 2c. Bun runtime - fast JS/TS runtime. Browser automation CLIs
	// (playwright-cli) and MCP servers (chrome-devtools) launch through
	// `bunx <pkg>@<version>`, so bun must be resolvable on PATH.
	uc.step(ctx, result, "bun", "Bun JS/TS runtime",
		`set -e
export PATH="$HOME/.volta/bin:$HOME/.local/bin:$PATH"
mkdir -p "$HOME/.local/bin"
npm install -g --no-audit --no-fund --prefix "$HOME/.local" bun
ln -sf "$HOME/.local/bin/bun" "$HOME/.local/bin/bunx"`)

	// 2d. Playwright CLI browsers - headless browser builds used by
	// `playwright-cli` (the deterministic/token-efficient automation path;
	// interactive work goes through chrome-devtools MCP). Installed via the
	// CLI's own installer so versions never drift.
	if uc.hasTool(ctx, "bun") {
		uc.logger.Info("LinuxBootstrap: ensuring Playwright CLI browsers")
		out, err := uc.runShell(ctx, `export PATH="$HOME/.volta/bin:$HOME/.local/bin:$PATH"
bunx @playwright/cli@latest install-browser chromium`)
		if err != nil {
			uc.logger.Error("LinuxBootstrap: playwright install-browser failed: %s (%s)", out, err)
			result.Diagnostics = append(result.Diagnostics, entity.Warn(
				"LinuxBootstrap",
				"Playwright CLI browsers",
				fmt.Sprintf("install-browser chromium failed: %v (%s)", err, out),
				"Run 'bunx @playwright/cli@latest install-browser chromium' manually",
			))
		} else {
			result.Diagnostics = append(result.Diagnostics, entity.OK(
				"LinuxBootstrap",
				"Playwright CLI browsers",
				"Playwright browsers provisioned via CLI installer (~/.cache/ms-playwright)",
			))
		}
	}

	// 3. OpenCode CLI - official V2 installer into ~/.opencode/bin. The
	// bootstrap path also converges a user-local V1 install; a pacman-owned
	// binary is reported instead of being shadowed.
	uc.ensureOpenCodeV2(ctx, result)

	// 3.5. CommandCode CLI - npm global (user prefix).
	uc.step(ctx, result, "cmdc", "CommandCode CLI",
		`set -e
export PATH="$HOME/.volta/bin:$HOME/.local/bin:$PATH"
npm install -g --no-audit --no-fund --prefix "$HOME/.local" command-code`)

	// 4. GitHub CLI (gh) - official release tarball into ~/.local/bin.
	uc.step(ctx, result, "gh", "GitHub CLI",
		`set -e
ARCH=$(uname -m); case "$ARCH" in x86_64|amd64) GHA=amd64;; aarch64|arm64) GHA=arm64;; *) echo "Unsupported arch: $ARCH"; exit 1;; esac
VER=$(curl -fsSL https://api.github.com/repos/cli/cli/releases/latest | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p')
TVER=${VER#v}
curl -fsSL "https://github.com/cli/cli/releases/download/${VER}/gh_${TVER}_linux_${GHA}.tar.gz" -o /tmp/envctl-gh.tgz
tar -xzf /tmp/envctl-gh.tgz -C /tmp
cp /tmp/gh_${TVER}_linux_${GHA}/bin/gh "$HOME/.local/bin/gh"
chmod +x "$HOME/.local/bin/gh"
rm -rf /tmp/envctl-gh.tgz /tmp/gh_${TVER}_linux_${GHA}`)

	// 5. git-delta pager - official release tarball into ~/.local/bin.
	uc.step(ctx, result, "delta", "git-delta pager",
		`set -e
ARCH=$(uname -m); case "$ARCH" in x86_64|amd64) DELTA_ARCH=x86_64-unknown-linux-gnu;; aarch64|arm64) DELTA_ARCH=aarch64-unknown-linux-gnu;; *) echo "Unsupported arch: $ARCH"; exit 1;; esac
VER=$(curl -fsSL https://api.github.com/repos/dandavison/delta/releases/latest | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p')
curl -fsSL "https://github.com/dandavison/delta/releases/download/${VER}/delta-${VER}-${DELTA_ARCH}.tar.gz" -o /tmp/envctl-delta.tgz
tar -xzf /tmp/envctl-delta.tgz -C /tmp
cp /tmp/delta-${VER}-${DELTA_ARCH}/delta "$HOME/.local/bin/delta"
chmod +x "$HOME/.local/bin/delta"
rm -rf /tmp/envctl-delta.tgz /tmp/delta-${VER}-${DELTA_ARCH}`)

	// 6. yq - static release binary into ~/.local/bin.
	uc.step(ctx, result, "yq", "yq YAML/JSON processor",
		`set -e
ARCH=$(uname -m); case "$ARCH" in x86_64|amd64) YQ_ARCH=amd64;; aarch64|arm64) YQ_ARCH=arm64;; *) echo "Unsupported arch: $ARCH"; exit 1;; esac
curl -fsSL https://github.com/mikefarah/yq/releases/latest/download/yq_linux_${YQ_ARCH} -o "$HOME/.local/bin/yq"
chmod +x "$HOME/.local/bin/yq"`)

	// 7. uv - official installer (installs to ~/.local/bin).
	uc.step(ctx, result, "uv", "uv Python package manager",
		`set -e
curl -LsSf https://astral.sh/uv/install.sh | sh`)

	// 8. ruff - installed via uv (user-local tool).
	if uc.hasTool(ctx, "uv") && !uc.hasTool(ctx, "ruff") {
		uc.logger.Info("LinuxBootstrap: installing ruff via uv")
		out, err := uc.runShell(ctx, `"$HOME/.local/bin/uv" tool install ruff`)
		if err != nil {
			uc.logger.Error("LinuxBootstrap: uv tool install ruff failed: %s (%s)", out, err)
			result.Diagnostics = append(result.Diagnostics, entity.Warn(
				"LinuxBootstrap",
				"ruff",
				fmt.Sprintf("uv tool install ruff failed: %v (%s)", err, out),
				"Run 'uv tool install ruff' manually",
			))
		} else {
			result.Diagnostics = append(result.Diagnostics, entity.OK(
				"LinuxBootstrap",
				"ruff",
				"Installed successfully via uv",
			))
		}
	}

	// 9. fd - Debian exposes it as `fdfind`, Arch ships plain `fd`. Bootstrap
	// runs before the packages step, so install it here as a best-effort
	// fallback through whichever package manager the host actually has.
	if !uc.hasTool(ctx, "fd") {
		uc.logger.Info("LinuxBootstrap: provisioning fd")
		out, err := uc.runShell(ctx, `FDFIND=$(command -v fdfind || command -v fd || true)
if [ -z "$FDFIND" ]; then
  if command -v pacman >/dev/null 2>&1; then
    sudo -n pacman -S --noconfirm --needed fd >/dev/null 2>&1 || true
  elif command -v apt-get >/dev/null 2>&1; then
    sudo -n apt-get update >/dev/null 2>&1 || true
    sudo -n apt-get install -y --no-install-recommends fd-find >/dev/null 2>&1 || true
  fi
  FDFIND=$(command -v fdfind || command -v fd || true)
fi
if [ -n "$FDFIND" ] && [ ! -e "$HOME/.local/bin/fd" ]; then ln -sf "$FDFIND" "$HOME/.local/bin/fd"; fi`)
		if err != nil {
			uc.logger.Error("LinuxBootstrap: fd provisioning failed: %s (%s)", out, err)
			result.Diagnostics = append(result.Diagnostics, entity.Warn(
				"LinuxBootstrap",
				"fd (fdfind symlink)",
				fmt.Sprintf("fd provisioning failed: %v (%s)", err, out),
			))
		} else if uc.hasTool(ctx, "fd") {
			result.Diagnostics = append(result.Diagnostics, entity.OK(
				"LinuxBootstrap",
				"fd (fdfind symlink)",
				"Linked fdfind as fd",
			))
		} else {
			uc.logger.Warn("LinuxBootstrap: fd not available, symlink skipped")
			result.Diagnostics = append(result.Diagnostics, entity.OK(
				"LinuxBootstrap",
				"fd (fdfind symlink)",
				"Skipped (no fd package found; install it with the distro package manager)",
			))
		}
	}

	// 10. paru - the AUR helper the `type: paru` manifest entries need. CachyOS
	// ships it in its own repo; on plain Arch it is AUR-only, so build it from
	// source there. Everything here is a no-op off the Arch family.
	if !uc.hasTool(ctx, "paru") {
		uc.logger.Info("LinuxBootstrap: provisioning paru (AUR helper)")
		out, err := uc.runShell(ctx, `set -e
command -v pacman >/dev/null 2>&1 || exit 0
if sudo -n pacman -S --needed --noconfirm paru >/dev/null 2>&1; then exit 0; fi
sudo -n pacman -S --needed --noconfirm base-devel git >/dev/null 2>&1 || true
BUILD=$(mktemp -d)
git clone --depth 1 https://aur.archlinux.org/paru-bin.git "$BUILD/paru-bin" >/dev/null 2>&1
cd "$BUILD/paru-bin" && makepkg -si --noconfirm >/dev/null 2>&1`)
		if err != nil {
			uc.logger.Error("LinuxBootstrap: paru provisioning failed: %s (%s)", out, err)
			result.Diagnostics = append(result.Diagnostics, entity.Warn(
				"LinuxBootstrap",
				"paru (AUR helper)",
				fmt.Sprintf("paru provisioning failed: %v (%s)", err, out),
				"Install paru from your repo or build it from the AUR, then re-run",
			))
		} else if uc.hasTool(ctx, "paru") {
			result.Diagnostics = append(result.Diagnostics, entity.OK(
				"LinuxBootstrap",
				"paru (AUR helper)",
				"Installed successfully",
			))
		} else {
			result.Diagnostics = append(result.Diagnostics, entity.OK(
				"LinuxBootstrap",
				"paru (AUR helper)",
				"Skipped (not an Arch-family host)",
			))
		}
	}

	// 11. Stylelint - CSS/SCSS linter (mirrors the Windows volta global package).
	uc.step(ctx, result, "stylelint", "Stylelint CSS/SCSS linter",
		"volta install stylelint")

	// 12. golangci-lint - the Go lint gate used by CI and by envctl-verify.
	// Without it here, the verifier's lint check silently skips on a fresh
	// machine and lint findings only surface in the pipeline.
	uc.step(ctx, result, "golangci-lint", "golangci-lint (CI lint gate)",
		`set -e
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b "$HOME/.local/bin" >/dev/null
"$HOME/.local/bin/golangci-lint" --version`)

	// 13. Go SDK - official tarball into /usr/local/go (requires sudo).
	// The prior install must be removed first: extracting over an old SDK
	// leaves orphaned stdlib/packages that corrupt builds (official guidance).
	uc.step(ctx, result, "go", "Go programming language SDK",
		`set -e
ARCH=$(uname -m); case "$ARCH" in x86_64|amd64) GO_ARCH=amd64;; aarch64|arm64) GO_ARCH=arm64;; *) echo "Unsupported arch: $ARCH"; exit 1;; esac
GO_VER=$(curl -fsSL https://go.dev/VERSION?m=text | head -1)
curl -fsSL "https://go.dev/dl/${GO_VER}.linux-${GO_ARCH}.tar.gz" -o /tmp/envctl-go.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf /tmp/envctl-go.tar.gz
rm -f /tmp/envctl-go.tar.gz
echo "Installed ${GO_VER}"`)

	// 14. Persist the Go PATH in shell profiles so future login shells find go
	// and gopls. One target, two dirs — replaces the old goPathInstaller
	// shell script; each line is independently idempotent.
	uc.pathStep(ctx, result, "Persist Go PATH in shell profiles", "/usr/local/go/bin", "$HOME/go/bin")

	// 15. hadolint - Dockerfile linter (no apt/pacman package upstream; same
	// release-binary pattern as gh/delta/yq). envctl-verify lints changed
	// Dockerfiles with it.
	uc.step(ctx, result, "hadolint", "hadolint (Dockerfile linter)",
		`set -e
ARCH=$(uname -m); case "$ARCH" in x86_64|amd64) HAD_ARCH=x86_64;; aarch64|arm64) HAD_ARCH=arm64;; *) echo "Unsupported arch: $ARCH"; exit 1;; esac
curl -fsSL "https://github.com/hadolint/hadolint/releases/latest/download/hadolint-linux-${HAD_ARCH}" -o "$HOME/.local/bin/hadolint"
chmod +x "$HOME/.local/bin/hadolint"
"$HOME/.local/bin/hadolint" --version`)

	// 16. fzf - a distro build can predate the built-in directory walker
	// (0.47), and without it fzf falls back to `find`: slow and blind to
	// ignore-files. Install the current release into ~/.local/bin only when
	// needed, so an up-to-date distro package stays (it also ships the shell
	// bindings under /usr/share/fzf).
	if uc.fzfSupportsWalker(ctx) {
		result.Diagnostics = append(result.Diagnostics, entity.OK(
			"LinuxBootstrap",
			"fzf (built-in directory walker)",
			"The installed fzf already provides the built-in directory walker",
		))
	} else {
		uc.logger.Info("LinuxBootstrap: installing a current fzf (built-in directory walker)")
		out, err := uc.runShell(ctx, `set -e
ARCH=$(uname -m); case "$ARCH" in x86_64|amd64) FZF_ARCH=amd64;; aarch64|arm64) FZF_ARCH=arm64;; *) echo "Unsupported arch: $ARCH"; exit 1;; esac
VER=$(curl -fsSL https://api.github.com/repos/junegunn/fzf/releases/latest | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p')
TVER=${VER#v}
curl -fsSL "https://github.com/junegunn/fzf/releases/download/${VER}/fzf-${TVER}-linux-${FZF_ARCH}.tar.gz" -o /tmp/envctl-fzf.tgz 2>/dev/null || curl -fsSL "https://github.com/junegunn/fzf/releases/download/${VER}/fzf-${TVER}-linux_${FZF_ARCH}.tar.gz" -o /tmp/envctl-fzf.tgz
tar -xzf /tmp/envctl-fzf.tgz -C /tmp
install -m 0755 /tmp/fzf "$HOME/.local/bin/fzf"
rm -f /tmp/envctl-fzf.tgz /tmp/fzf
"$HOME/.local/bin/fzf" --version`)
		if err != nil {
			uc.logger.Error("LinuxBootstrap: fzf install failed: %s (%s)", out, err)
			result.Diagnostics = append(result.Diagnostics, entity.Warn(
				"LinuxBootstrap",
				"fzf (built-in directory walker)",
				fmt.Sprintf("fzf install failed: %v (%s)", err, out),
				"install a fzf >= 0.47 manually (it replaced the `find` fallback with a built-in walker)",
			))
		} else {
			result.Diagnostics = append(result.Diagnostics, entity.OK(
				"LinuxBootstrap",
				"fzf (built-in directory walker)",
				"Installed current fzf via release tarball",
			))
		}
	}

	return result, nil
}
