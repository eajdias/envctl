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

// ProvisionBootstrapUseCase installs the Linux toolchain required to replicate
// the global OpenCode environment on Ubuntu servers: Volta + Node, the OpenCode
// CLI and the user-local CLI tools (gh, delta, yq, uv, ruff, oh-my-posh, fd).
// It is a no-op on Windows, where winget/volta packages cover the toolchain.
type ProvisionBootstrapUseCase struct {
	fsManager    repository.FileSystemManager
	manifestRepo repository.ManifestRepository
	logger       repository.Logger
}

// BootstrapResult holds the diagnostics produced during bootstrap provisioning.
type BootstrapResult struct {
	Diagnostics []entity.Diagnostic
}

// NewProvisionBootstrapUseCase builds the Linux toolchain bootstrap use case.
func NewProvisionBootstrapUseCase(fsManager repository.FileSystemManager, manifestRepo repository.ManifestRepository, logger repository.Logger) *ProvisionBootstrapUseCase {
	return &ProvisionBootstrapUseCase{fsManager: fsManager, manifestRepo: manifestRepo, logger: logger}
}

// userHome expands ~ to the current user's home directory.
func (uc *ProvisionBootstrapUseCase) userHome() string {
	home, _ := uc.fsManager.ExpandUserPath("~")
	return home
}

// shellEnv returns an environment that makes Volta shims (~/.volta/bin) and
// user-local binaries (~/.local/bin) available on PATH, without mutating the
// process environment.
func (uc *ProvisionBootstrapUseCase) shellEnv() []string {
	return linuxToolchainEnv(uc.userHome())
}

// linuxToolchainEnv builds an environment that resolves Volta shims,
// user-local binaries, Go, and Rustup/Cargo, shared by the bootstrap and
// doctor use cases.
func linuxToolchainEnv(home string) []string {
	localBin := filepath.Join(home, ".local", "bin")
	voltaBin := filepath.Join(home, ".volta", "bin")
	cargoBin := filepath.Join(home, ".cargo", "bin")
	goBin := "/usr/local/go/bin"
	userGoBin := filepath.Join(home, "go", "bin")
	path := strings.Join([]string{localBin, voltaBin, cargoBin, goBin, userGoBin, os.Getenv("PATH")}, string(os.PathListSeparator))
	env := []string{
		"PATH=" + path,
		"VOLTA_HOME=" + filepath.Join(home, ".volta"),
		"GOPATH=" + filepath.Join(home, "go"),
	}
	for _, kv := range os.Environ() {
		key := kv[:strings.IndexByte(kv, '=')]
		if key == "PATH" || key == "VOLTA_HOME" {
			continue
		}
		env = append(env, kv)
	}
	return env
}

// toolAvailable reports whether a binary resolves on the platform PATH. On
// Linux it additionally resolves Volta shims (~/.volta/bin) and user-local
// binaries (~/.local/bin), which are not part of the process PATH.
func toolAvailable(ctx context.Context, name string) bool {
	if runtime.GOOS == "linux" {
		if home, err := os.UserHomeDir(); err == nil && home != "" {
			c := exec.CommandContext(ctx, "bash", "-lc", "command -v "+name+" >/dev/null 2>&1")
			c.Env = linuxToolchainEnv(home)
			return c.Run() == nil
		}
	}
	_, err := exec.LookPath(name)
	return err == nil
}

// ensureProcessToolchainPath mutates the process environment so that Volta
// shims, user-local binaries, Go, and Rustup/Cargo are resolvable by
// subsequent provisioning steps running in the same process.
func (uc *ProvisionBootstrapUseCase) ensureProcessToolchainPath() {
	home := uc.userHome()
	if home == "" {
		return
	}
	localBin := filepath.Join(home, ".local", "bin")
	voltaBin := filepath.Join(home, ".volta", "bin")
	cargoBin := filepath.Join(home, ".cargo", "bin")
	goBin := "/usr/local/go/bin"
	userGoBin := filepath.Join(home, "go", "bin")
	cur := os.Getenv("PATH")
	if !strings.Contains(cur, localBin) || !strings.Contains(cur, voltaBin) || !strings.Contains(cur, cargoBin) {
		os.Setenv("PATH", strings.Join([]string{localBin, voltaBin, cargoBin, goBin, userGoBin, cur}, string(os.PathListSeparator)))
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

// hasTool reports whether a binary is resolvable on the Volta-aware PATH.
func (uc *ProvisionBootstrapUseCase) hasTool(ctx context.Context, name string) bool {
	_, err := uc.runShell(ctx, "command -v "+name+" >/dev/null 2>&1")
	return err == nil
}

// step installs a tool when missing, or reports it as already available.
func (uc *ProvisionBootstrapUseCase) step(ctx context.Context, result *BootstrapResult, name, target, installScript string) {
	if uc.hasTool(ctx, name) {
		uc.logger.LogIdempotency("LinuxBootstrap", target, true, "already installed")
		result.Diagnostics = append(result.Diagnostics, entity.Diagnostic{
			Category: entity.DiagOK, System: "LinuxBootstrap", Target: target,
			Details: "Already installed and available on PATH",
		})
		return
	}

	uc.logger.Info("LinuxBootstrap: installing %s (%s)", name, target)
	out, err := uc.runShell(ctx, installScript)
	if err != nil {
		msg := fmt.Sprintf("Installation failed: %v", err)
		if out != "" {
			msg += ": " + out
		}
		uc.logger.Error("LinuxBootstrap: failed to install %s: %s", target, msg)
		result.Diagnostics = append(result.Diagnostics, entity.Diagnostic{
			Category: entity.DiagWarning, System: "LinuxBootstrap", Target: target,
			Details: msg, FixHint: "Run the install command manually as your user",
		})
		return
	}
	uc.logger.LogIdempotency("LinuxBootstrap", target, false, "installed successfully")
	result.Diagnostics = append(result.Diagnostics, entity.Diagnostic{
		Category: entity.DiagOK, System: "LinuxBootstrap", Target: target,
		Details: "Installed successfully",
	})
}

// Execute provisions the Linux toolchain. On Windows it is a no-op.
func (uc *ProvisionBootstrapUseCase) Execute(ctx context.Context) (*BootstrapResult, error) {
	result := &BootstrapResult{}

	if runtime.GOOS != "linux" {
		result.Diagnostics = append(result.Diagnostics, entity.Diagnostic{
			Category: entity.DiagOK, System: "LinuxBootstrap", Target: "toolchain bootstrap",
			Details: "Skipped on Windows (toolchain provisioned via winget/volta packages)",
		})
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
			result.Diagnostics = append(result.Diagnostics, entity.Diagnostic{
				Category: entity.DiagWarning, System: "LinuxBootstrap", Target: "Node.js + pnpm",
				Details: fmt.Sprintf("volta install failed: %v (%s)", err, out),
				FixHint: "Run 'volta install " + nodeSpec + " pnpm' manually",
			})
		} else {
			result.Diagnostics = append(result.Diagnostics, entity.Diagnostic{
				Category: entity.DiagOK, System: "LinuxBootstrap", Target: "Node.js + pnpm",
				Details: "Provisioned via Volta (" + nodeSpec + ")",
			})
		}
	}

	// 2b. Expose Volta on interactive shells. get.volta.sh can skip rc-file
	// integration when run non-interactively, leaving volta off the PATH of
	// future login shells. Append the standard exports if missing.
	if uc.hasTool(ctx, "volta") {
		uc.logger.Info("LinuxBootstrap: ensuring Volta is exported on interactive shells")
		out, err := uc.runShell(ctx, `set -e
VOLTA_LINES='export VOLTA_HOME="$HOME/.volta"
export PATH="$VOLTA_HOME/bin:$PATH"'
for f in "$HOME/.bashrc" "$HOME/.profile"; do
  if [ -f "$f" ] && ! grep -q "VOLTA_HOME" "$f"; then
    printf '\n# Volta (via envctl bootstrap)\n%s\n' "$VOLTA_LINES" >> "$f"
  fi
done`)
		if err != nil {
			uc.logger.Warn("LinuxBootstrap: failed to add Volta to shell rc files: %s (%s)", out, err)
			result.Diagnostics = append(result.Diagnostics, entity.Diagnostic{
				Category: entity.DiagWarning, System: "LinuxBootstrap", Target: "Volta shell integration",
				Details: fmt.Sprintf("failed to append Volta exports to ~/.bashrc/~/.profile: %v (%s)", err, out),
				FixHint: "Append 'export VOLTA_HOME=$HOME/.volta' and 'export PATH=$VOLTA_HOME/bin:$PATH' to ~/.bashrc",
			})
		} else {
			result.Diagnostics = append(result.Diagnostics, entity.Diagnostic{
				Category: entity.DiagOK, System: "LinuxBootstrap", Target: "Volta shell integration",
				Details: "Volta exports ensured in ~/.bashrc and ~/.profile",
			})
		}
	}

	// 3. OpenCode CLI - npm global (user prefix) with official script fallback.
	uc.step(ctx, result, "opencode", "OpenCode CLI",
		`set -e
export PATH="$HOME/.volta/bin:$HOME/.local/bin:$PATH"
if ! npm install -g --no-audit --no-fund --prefix "$HOME/.local" opencode-ai >/tmp/envctl-opencode-npm.log 2>&1; then
  curl -fsSL https://opencode.ai/install | bash >/tmp/envctl-opencode-curl.log 2>&1
fi`)

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

	// 8. oh-my-posh - official installer (installs to ~/.local/bin).
	uc.step(ctx, result, "oh-my-posh", "Oh-My-Posh prompt engine",
		`set -e
curl -s https://ohmyposh.dev/install.sh | bash -s`)

	// 9. ruff - installed via uv (user-local tool).
	if uc.hasTool(ctx, "uv") && !uc.hasTool(ctx, "ruff") {
		uc.logger.Info("LinuxBootstrap: installing ruff via uv")
		out, err := uc.runShell(ctx, `"$HOME/.local/bin/uv" tool install ruff`)
		if err != nil {
			uc.logger.Error("LinuxBootstrap: uv tool install ruff failed: %s (%s)", out, err)
			result.Diagnostics = append(result.Diagnostics, entity.Diagnostic{
				Category: entity.DiagWarning, System: "LinuxBootstrap", Target: "ruff",
				Details: fmt.Sprintf("uv tool install ruff failed: %v (%s)", err, out),
				FixHint: "Run 'uv tool install ruff' manually",
			})
		} else {
			result.Diagnostics = append(result.Diagnostics, entity.Diagnostic{
				Category: entity.DiagOK, System: "LinuxBootstrap", Target: "ruff",
				Details: "Installed successfully via uv",
			})
		}
	}

	// 10. fd symlink - the apt fd-find package exposes `fdfind`; expose it as `fd`.
	// Bootstrap runs before the apt packages step, so on fresh VPSs fdfind may
	// not exist yet: install fd-find here as a best-effort fallback.
	if !uc.hasTool(ctx, "fd") {
		uc.logger.Info("LinuxBootstrap: linking fdfind as fd")
		out, err := uc.runShell(ctx, `FDFIND=$(command -v fdfind || true)
if [ -z "$FDFIND" ]; then
  sudo -n apt-get update >/dev/null 2>&1 || true
  sudo -n apt-get install -y --no-install-recommends fd-find >/dev/null 2>&1 || true
  FDFIND=$(command -v fdfind || true)
fi
if [ -n "$FDFIND" ] && [ ! -e "$HOME/.local/bin/fd" ]; then ln -sf "$FDFIND" "$HOME/.local/bin/fd"; fi`)
		if err != nil {
			uc.logger.Error("LinuxBootstrap: fd symlink failed: %s (%s)", out, err)
			result.Diagnostics = append(result.Diagnostics, entity.Diagnostic{
				Category: entity.DiagWarning, System: "LinuxBootstrap", Target: "fd (fdfind symlink)",
				Details: fmt.Sprintf("fd symlink failed: %v (%s)", err, out),
			})
		} else if uc.hasTool(ctx, "fd") {
			result.Diagnostics = append(result.Diagnostics, entity.Diagnostic{
				Category: entity.DiagOK, System: "LinuxBootstrap", Target: "fd (fdfind symlink)",
				Details: "Linked fdfind as fd",
			})
		} else {
			uc.logger.Warn("LinuxBootstrap: fdfind not available, fd symlink skipped")
			result.Diagnostics = append(result.Diagnostics, entity.Diagnostic{
				Category: entity.DiagOK, System: "LinuxBootstrap", Target: "fd (fdfind symlink)",
				Details: "Skipped (fdfind not found; install via 'apt install fd-find')",
			})
		}
	}

	// 11. python-lsp-server - installed via uv so the `pylsp` binary lands in
	// ~/.local/bin. Ubuntu 24.04 blocks system pip installs (PEP 668), so pip is
	// not a viable installer on Linux.
	if uc.hasTool(ctx, "uv") && !uc.hasTool(ctx, "pylsp") {
		uc.logger.Info("LinuxBootstrap: installing python-lsp-server via uv")
		out, err := uc.runShell(ctx, `"$HOME/.local/bin/uv" tool install python-lsp-server`)
		if err != nil {
			uc.logger.Error("LinuxBootstrap: uv tool install python-lsp-server failed: %s (%s)", out, err)
			result.Diagnostics = append(result.Diagnostics, entity.Diagnostic{
				Category: entity.DiagWarning, System: "LinuxBootstrap", Target: "pylsp (python-lsp-server)",
				Details: fmt.Sprintf("uv tool install python-lsp-server failed: %v (%s)", err, out),
				FixHint: "Run 'uv tool install python-lsp-server' manually",
			})
		} else {
			result.Diagnostics = append(result.Diagnostics, entity.Diagnostic{
				Category: entity.DiagOK, System: "LinuxBootstrap", Target: "pylsp (python-lsp-server)",
				Details: "Installed successfully via uv",
			})
		}
	}

	// 12. Firecrawl CLI - global npm tool used by the firecrawl-* agent skills
	// (mirrors the Windows volta global package).
	uc.step(ctx, result, "firecrawl", "Firecrawl CLI",
		"volta install firecrawl-cli")

	// 13. Stylelint - CSS/SCSS linter (mirrors the Windows volta global package).
	uc.step(ctx, result, "stylelint", "Stylelint CSS/SCSS linter",
		"volta install stylelint")

	// 14. Go SDK - official tarball into /usr/local/go (requires sudo).
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

	// 15. Rustup - official non-interactive installer (installs to ~/.cargo).
	uc.step(ctx, result, "rustup", "Rustup Rust toolchain manager",
		`set -e
curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y
source "$HOME/.cargo/env"
rustup default stable
rustup component add rust-analyzer`)

	// 16. Persist Go and Cargo PATH in shell profiles so future login shells
	// find go, gopls, rustc, cargo, rust-analyzer, etc.
	uc.step(ctx, result, "shell-path", "Persist Go/Cargo/Rust PATH in shell profiles",
		`set -e
PATH_LINES='
# Go SDK (via envctl bootstrap)
export PATH="/usr/local/go/bin:$HOME/go/bin:$PATH"
# Rust/Cargo (via envctl bootstrap)
export PATH="$HOME/.cargo/bin:$PATH"'
for f in "$HOME/.bashrc" "$HOME/.profile"; do
  if [ -f "$f" ] && ! grep -q "/usr/local/go/bin" "$f"; then
    printf '%s\n' "$PATH_LINES" >> "$f"
  fi
done
echo "Go and Cargo PATH persisted to ~/.bashrc and ~/.profile"`)

	return result, nil
}
