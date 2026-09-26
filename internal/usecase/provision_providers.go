package usecase

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/eajdias/envctl/internal/domain/entity"
	"github.com/eajdias/envctl/internal/domain/repository"
)

// ProvisionProvidersUseCase is the "phase 0" preflight: before packages,
// toolchains or configs, make sure the agent CLIs themselves are present and
// current, so a fresh machine can reach the agents without a manual step.
//
// Policy per tool, learned from the machines this runs on: anything Volta owns
// is installed and updated through Volta (it resolves the npm latest, so
// "outdated" and "missing" are the same command). A binary owned by the OS is
// authoritative: on Arch, an envctl-owned user copy is archived so pacman keeps
// ownership; on Ubuntu/Debian, a legacy system v1 is the one deliberate
// exception because it cannot load the V2-native config.
type ProvisionProvidersUseCase struct {
	manifestRepo repository.ManifestRepository
	fsManager    repository.FileSystemManager
	managers     map[entity.PackageType]repository.PackageManager
	logger       repository.Logger
}

func NewProvisionProvidersUseCase(
	manifestRepo repository.ManifestRepository,
	fsManager repository.FileSystemManager,
	managers map[entity.PackageType]repository.PackageManager,
	logger repository.Logger,
) *ProvisionProvidersUseCase {
	return &ProvisionProvidersUseCase{
		manifestRepo: manifestRepo,
		fsManager:    fsManager,
		managers:     managers,
		logger:       logger,
	}
}

// logInfo logs through the optional logger.
func (uc *ProvisionProvidersUseCase) logInfo(format string, args ...any) {
	uc.logger.Info(format, args...)
}

// toolchainEnv builds the environment used for provider probes, so Volta shims
// resolve even when envctl runs from a non-login shell (ssh, systemd, agent).
func toolchainEnv() []string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" || runtime.GOOS == "windows" {
		return os.Environ()
	}
	return linuxToolchainEnv(home)
}

// runWithToolchain runs a command against the toolchain PATH and returns its
// trimmed combined output.
func runWithToolchain(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = toolchainEnv()
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

// resolveOnToolchainPath mirrors toolchain.lookPathWithEnv against the PATH
// built by toolchainEnv(), so probes see what execution sees (non-login
// shells have a minimal process PATH; the toolchain PATH prepends
// ~/.local/bin, ~/.volta/bin, /usr/local/go/bin, ~/go/bin).
func resolveOnToolchainPath(name string) (string, error) {
	env := toolchainEnv()
	path := ""
	for _, kv := range env {
		if v, ok := strings.CutPrefix(kv, "PATH="); ok {
			path = v
			break
		}
	}
	if path == "" {
		return exec.LookPath(name)
	}
	if filepath.IsAbs(name) {
		return name, nil
	}
	for _, dir := range filepath.SplitList(path) {
		if dir == "" {
			dir = "."
		}
		candidate := filepath.Join(dir, name)
		if fi, err := os.Stat(candidate); err == nil && !fi.IsDir() && fi.Mode()&0111 != 0 {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("executable %q not found on toolchain PATH", name)
}

// openCodeLinuxV2Installer installs the official OpenCode V2 channel. The
// legacy https://opencode.ai/install endpoint still serves the 1.x line; the
// V2-native configs deployed by this repository require the /v2 installer.
// The official installer selects the current architecture, baseline, and musl
// artifact and manages the user's PATH. Shared by phase 0 and Linux bootstrap.
const openCodeLinuxV2Installer = `set -e
export PATH="$HOME/.opencode/bin:$HOME/.volta/bin:$HOME/.local/bin:$PATH"
curl -fsSL https://opencode.ai/v2/install | bash`

// openCodePathInstaller persists the official V2 install directory even when
// envctl has already prepended it to the child PATH. The upstream installer
// otherwise treats that temporary PATH as an existing shell configuration and
// skips writing the rc entry on a fresh machine.
const openCodePathInstaller = `set -e
for f in "$HOME/.bashrc" "$HOME/.profile"; do
  if [ -f "$f" ] && ! grep -Fq '.opencode/bin' "$f"; then
    printf '\n# OpenCode (via envctl)\nexport PATH="$HOME/.opencode/bin:$PATH"\n' >> "$f"
  fi
done
if command -v fish >/dev/null 2>&1; then
  f="$HOME/.config/fish/config.fish"
  mkdir -p "$(dirname "$f")"
  if [ ! -f "$f" ] || ! grep -Fq "$HOME/.opencode/bin" "$f"; then
    printf '\n# OpenCode (via envctl)\nset -gx PATH "$HOME/.opencode/bin" $PATH\n' >> "$f"
  fi
fi`

// versionMajorAtLeast reports whether version's numeric major is at least
// major. It intentionally rejects malformed versions instead of treating an
// unparseable provider as compatible with the V2-native configuration.
func versionMajorAtLeast(version string, major int) bool {
	normalized := normalizeVersion(version)
	majorToken, _, _ := strings.Cut(normalized, ".")
	parsed, err := strconv.Atoi(majorToken)
	return err == nil && parsed >= major
}

// standaloneProviderCanReplace reports whether envctl may replace a standalone
// binary. User-local installs are owned by envctl. A system-owned binary is
// normally left alone, except on hosts without a pacman-owned opencode:
// Ubuntu/Debian may have a legacy v1 npm copy that must not shadow the
// V2-native configuration.
func standaloneProviderCanReplace(source string, pacmanOwns bool) bool {
	return !pacmanOwns && source != sourceAbsent
}

// providerCLI is one agent-facing CLI phase 0 is responsible for.
type providerCLI struct {
	name             string // human name for diagnostics
	binary           string // binary that must resolve on PATH
	voltaPkg         string // Volta/npm package name, empty when the tool has its own installer
	windowsInstaller string // PowerShell installer used on Windows when there is no Volta package
	installer        string // shell installer used on Linux when there is no Volta package
	requiredMajor    int    // minimum compatible major version, zero when unconstrained
}

func providerCLIs() []providerCLI {
	return []providerCLI{
		{name: "CommandCode CLI", binary: "cmdc", voltaPkg: "command-code"},
		{
			name:             "OpenCode CLI",
			binary:           "opencode",
			installer:        openCodeLinuxV2Installer,
			windowsInstaller: `$ErrorActionPreference='Stop'; $bin=Join-Path $HOME '.local\bin'; New-Item -ItemType Directory -Force -Path $bin | Out-Null; $ver=(Invoke-RestMethod 'https://opencode.ai/update/api/latest/cli/npm').version; $arch='x64'; if ($env:PROCESSOR_ARCHITECTURE -match 'ARM64') { $arch='arm64' }; $variant="opencode-windows-$arch.zip"; try { Add-Type -MemberDefinition '[DllImport("kernel32.dll")] public static extern bool IsProcessorFeaturePresent(int f);' -Name K32 -Namespace W32 -PassThru | Out-Null; if ($arch -eq 'x64' -and -not [W32.K32]::IsProcessorFeaturePresent(40)) { $variant='opencode-windows-x64-baseline.zip' } } catch {}; $url="https://opencode.ai/files/bin/$ver/$variant"; $tmp=Join-Path $env:TEMP 'opencode-win.zip'; Invoke-WebRequest -Uri $url -OutFile $tmp -UseBasicParsing; Expand-Archive -Path $tmp -DestinationPath $bin -Force; Remove-Item $tmp -Force; $p=[Environment]::GetEnvironmentVariable('Path','User'); if ($p -notlike "*$bin*") { [Environment]::SetEnvironmentVariable('Path', "$p;$bin", 'User') }`,
			requiredMajor:    2,
		},
	}
}

// Execute ensures Volta, a default Node runtime and the provider CLIs. It is
// idempotent: a converged machine reports OK for everything and changes nothing.
func (uc *ProvisionProvidersUseCase) Execute(ctx context.Context) ([]entity.Diagnostic, error) {
	var diags []entity.Diagnostic
	add := func(status entity.DiagnosticStatus, target, details, hint string) {
		diags = append(diags, entity.Diagnostic{
			Category: status, System: "Providers", Target: target, Details: details, FixHint: hint,
		})
	}

	voltaReady := uc.ensureVolta(ctx, add)
	if !voltaReady {
		// Standalone providers do not depend on Volta (notably OpenCode's
		// official Linux installer), so still converge them when the npm-backed
		// provider path is unavailable.
		for _, tool := range providerCLIs() {
			if tool.voltaPkg == "" {
				uc.ensureProviderCLI(ctx, tool, add)
				continue
			}
			if installed := uc.installedVersion(ctx, tool.binary); installed == "" {
				add(entity.DiagWarning, tool.name,
					"Not installed and Volta is unavailable to install it",
					"Install Volta, then run 'envctl run providers' again")
			} else {
				add(entity.DiagOK, tool.name, fmt.Sprintf("v%s available; update requires Volta", installed), "")
			}
		}
		return diags, nil
	}

	uc.ensureNodeRuntime(ctx, add)

	for _, tool := range providerCLIs() {
		uc.ensureProviderCLI(ctx, tool, add)
	}
	return diags, nil
}

// ensureVolta installs Volta when it is missing and reports whether it is
// available afterwards. Volta has no self-update command: updating it means
// re-running its installer, which the bootstrap already does when it is absent.
func (uc *ProvisionProvidersUseCase) ensureVolta(ctx context.Context, add func(entity.DiagnosticStatus, string, string, string)) bool {
	if version := uc.installedVersion(ctx, "volta"); version != "" {
		add(entity.DiagOK, "Volta", fmt.Sprintf("v%s available", version), "")
		return true
	}

	if runtime.GOOS == "windows" {
		if mgr, ok := uc.managers[entity.PackageTypeWinget]; ok && mgr.IsAvailable(ctx) {
			if err := mgr.Install(ctx, entity.Package{ID: "Volta.Volta", Type: entity.PackageTypeWinget}); err == nil {
				add(entity.DiagOK, "Volta", "Installed via winget (Volta.Volta)", "")
				return true
			}
		}
		add(entity.DiagWarning, "Volta", "Not installed and winget could not install it",
			"Run 'envctl run winget' to install Volta.Volta")
		return false
	}

	uc.logInfo("Providers: installing Volta")
	installer := `curl -fsSL https://get.volta.sh | bash`
	cmd := exec.CommandContext(ctx, "bash", "-lc", installer)
	if out, err := cmd.CombinedOutput(); err != nil {
		add(entity.DiagWarning, "Volta", fmt.Sprintf("Installer failed: %v (%s)", err, strings.TrimSpace(string(out))),
			"Run 'curl -fsSL https://get.volta.sh | bash' manually, then 'envctl run providers'")
		return false
	}
	add(entity.DiagOK, "Volta", "Installed via the official installer", "")
	return true
}

// ensureNodeRuntime guarantees a default Node for Volta to build tool shims on,
// using the same spec the manifest declares so both stay in step.
func (uc *ProvisionProvidersUseCase) ensureNodeRuntime(ctx context.Context, add func(entity.DiagnosticStatus, string, string, string)) {
	spec := uc.manifestNodeSpec()
	if spec == "" {
		return
	}
	cmd := exec.CommandContext(ctx, "volta", "which", "node")
	cmd.Env = toolchainEnv()
	if cmd.Run() == nil {
		return
	}
	uc.logInfo("Providers: installing the default Node runtime (%s)", spec)
	out, err := runWithToolchain(ctx, "volta", "install", spec)
	if err != nil {
		add(entity.DiagWarning, "Node runtime", fmt.Sprintf("volta install %s failed: %v (%s)", spec, err, out),
			"Run 'volta install "+spec+"' manually")
		return
	}
	add(entity.DiagOK, "Node runtime", fmt.Sprintf("Installed %s as the default runtime", spec), "")
}

// manifestNodeSpec returns the Node spec the manifest pins for Volta, so phase 0
// never invents a different one.
func (uc *ProvisionProvidersUseCase) manifestNodeSpec() string {
	pkgs, err := uc.manifestRepo.LoadPackages()
	if err != nil {
		return ""
	}
	for _, p := range pkgs {
		if p.Type == entity.PackageTypeVolta && strings.HasPrefix(p.ID, "node@") {
			return p.ID
		}
	}
	return ""
}

// ensureProviderCLI installs the CLI when missing and updates it when Volta owns
// it and the registry has moved on; a tool owned by the OS is reported instead.
func (uc *ProvisionProvidersUseCase) ensureProviderCLI(ctx context.Context, tool providerCLI, add func(entity.DiagnosticStatus, string, string, string)) {
	installed := uc.installedVersion(ctx, tool.binary)
	source := installSource(tool.binary)
	pacmanOwns := tool.binary == "opencode" && uc.pacmanOwnsOpenCode(ctx)

	// A user-local copy must never keep winning over a package that pacman
	// owns. Archive every active non-system copy, then let the next probe see
	// the system binary; the package path is verified below before success.
	for pacmanOwns && source != sourceSystem {
		path, err := resolveOnToolchainPath(tool.binary)
		if err != nil {
			add(entity.DiagWarning, tool.name, fmt.Sprintf("could not locate the user-local binary to archive: %v", err),
				"Remove the stale user-local opencode binary, then run 'envctl run providers' again")
			return
		}
		backup, err := archiveUserOpenCode(path)
		if err != nil {
			add(entity.DiagWarning, tool.name, fmt.Sprintf("could not archive user-local opencode at %s: %v", path, err),
				"Move the stale user-local binary aside, then run 'envctl run providers' again")
			return
		}
		uc.logInfo("Providers: archived user-local opencode at %s; pacman remains authoritative", backup)
		installed = uc.installedVersion(ctx, tool.binary)
		source = installSource(tool.binary)
	}

	switch {
	case installed == "" && tool.voltaPkg != "":
		out, err := runWithToolchain(ctx, "volta", "install", tool.voltaPkg)
		if err != nil {
			add(entity.DiagWarning, tool.name, fmt.Sprintf("volta install %s failed: %v (%s)", tool.voltaPkg, err, out),
				"Run 'volta install "+tool.voltaPkg+"' manually")
			return
		}
		add(entity.DiagOK, tool.name, "Installed via Volta", "")

	case installed == "" && tool.voltaPkg == "":
		uc.installStandaloneProvider(ctx, tool, add, false)

	case tool.requiredMajor > 0 && !versionMajorAtLeast(installed, tool.requiredMajor):
		if pacmanOwns {
			uc.updatePacmanOpenCode(ctx, tool, add)
			return
		}
		if !standaloneProviderCanReplace(source, pacmanOwns) {
			add(entity.DiagWarning, tool.name,
				fmt.Sprintf("v%s is managed by the system package manager; it cannot load the V2-native configuration", installed),
				"Update the distro-owned opencode package, then run 'envctl run providers' again")
			return
		}
		uc.logInfo("Providers: upgrading %s to major v%d (%s -> official installer)", tool.name, tool.requiredMajor, installed)
		uc.installStandaloneProvider(ctx, tool, add, source != sourceSystem)

	case tool.voltaPkg != "" && source == sourceVolta:
		latest := uc.npmLatest(ctx, tool.voltaPkg)
		if latest == "" || !versionsDiffer(installed, latest) {
			add(entity.DiagOK, tool.name, fmt.Sprintf("v%s (Volta, current)", installed), "")
			return
		}
		uc.logInfo("Providers: updating %s (%s -> %s)", tool.name, installed, latest)
		out, err := runWithToolchain(ctx, "volta", "install", tool.voltaPkg)
		if err != nil {
			add(entity.DiagWarning, tool.name, fmt.Sprintf("update to v%s failed: %v (%s)", latest, err, out),
				"Run 'volta install "+tool.voltaPkg+"' manually")
			return
		}
		after := uc.installedVersion(ctx, tool.binary)
		if after == "" || after == installed {
			// Volta's shim map can lag a beat behind its own install; never claim
			// "updated X -> X" — report what was asked for instead.
			after = latest
		}
		add(entity.DiagOK, tool.name, fmt.Sprintf("Updated v%s -> v%s via Volta", installed, after), "")

	default:
		// Present but not Volta's. Never shadow a package-managed binary: the
		// copy envctl placed would win on PATH and freeze that version.
		if tool.binary == "opencode" && source != sourceSystem && !uc.ensureOpenCodeShellPath(ctx, tool, add) {
			return
		}
		add(entity.DiagOK, tool.name, fmt.Sprintf("v%s (%s; update it with its own manager)", installed, sourceLabel(source)), "")
	}
}

// ensureOpenCodeShellPath persists the official Linux install directory in
// POSIX and fish profiles. The upstream installer may see envctl's temporary
// toolchain PATH and skip its own rc write, so envctl owns this idempotent
// persistence step explicitly.
func (uc *ProvisionProvidersUseCase) ensureOpenCodeShellPath(ctx context.Context, tool providerCLI, add func(entity.DiagnosticStatus, string, string, string)) bool {
	if runtime.GOOS != "linux" || tool.binary != "opencode" {
		return true
	}
	out, err := runWithToolchain(ctx, "bash", "-lc", openCodePathInstaller)
	if err != nil {
		add(entity.DiagWarning, tool.name, fmt.Sprintf("could not persist ~/.opencode/bin in shell profiles: %v (%s)", err, out),
			"Run 'envctl run shell' after installing OpenCode")
		return false
	}
	return true
}

// installStandaloneProvider installs a provider that ships its own installer
// instead of an npm package (OpenCode: official v2 installer on Linux,
// PowerShell-native official zip on Windows, pacman on Arch).
func (uc *ProvisionProvidersUseCase) installStandaloneProvider(ctx context.Context, tool providerCLI, add func(entity.DiagnosticStatus, string, string, string), preferUserInstaller bool) {
	if runtime.GOOS == "windows" && tool.windowsInstaller != "" {
		uc.logInfo("Providers: installing %s", tool.name)
		//nolint:gosec // G204: command/args come from the local provider table, not user input.
		cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-Command", tool.windowsInstaller)
		if out, err := cmd.CombinedOutput(); err != nil {
			add(entity.DiagWarning, tool.name, fmt.Sprintf("installer failed: %v (%s)", err, strings.TrimSpace(string(out))),
				"Run the Windows installer manually (see docs/os-and-agent-matrix.md Fase 0)")
			return
		}
		if !uc.requireStandaloneProviderVersion(ctx, tool, add) {
			return
		}
		add(entity.DiagOK, tool.name, "Installed via the official v2 installer (PowerShell, ~/.local/bin)", "")
		return
	}

	// Arch/CachyOS owns opencode via the `extra` repo (manifests/packages.yaml):
	// prefer the distro channel for a fresh install so phase 0 never shadows a
	// system package. An existing user-local V1 is different: it must be replaced
	// in ~/.opencode/bin, otherwise PATH would continue to select that stale copy.
	if uc.pacmanManagerAvailable(ctx) && !preferUserInstaller {
		mgr, ok := uc.managers[entity.PackageTypePacman]
		if !ok {
			add(entity.DiagWarning, tool.name, "pacman is available but no package manager was configured",
				"Run 'envctl run pacman' after restoring the package manager configuration")
			return
		}
		if err := mgr.Install(ctx, entity.Package{ID: "opencode", Type: entity.PackageTypePacman}); err == nil {
			if !uc.requireStandaloneProviderVersion(ctx, tool, add) {
				return
			}
			add(entity.DiagOK, tool.name, "Installed via pacman (extra)", "")
			return
		}
		add(entity.DiagWarning, tool.name, "pacman could not install opencode",
			"Run 'envctl run pacman' to install opencode")
		return
	}

	if tool.installer == "" {
		add(entity.DiagWarning, tool.name, "Not installed and no installer is known for this platform",
			"Install "+tool.name+" manually")
		return
	}

	uc.logInfo("Providers: installing %s", tool.name)
	out, err := runWithToolchain(ctx, "bash", "-lc", tool.installer)
	if err != nil {
		add(entity.DiagWarning, tool.name, fmt.Sprintf("installer failed: %v (%s)", err, out),
			"Run '"+tool.installer+"' manually")
		return
	}
	if !uc.requireStandaloneProviderVersion(ctx, tool, add) || !uc.ensureOpenCodeShellPath(ctx, tool, add) {
		return
	}
	add(entity.DiagOK, tool.name, "Installed via the official v2 installer", "")
}

// updatePacmanOpenCode updates the distro-owned package and verifies that the
// active system binary is compatible. It never falls back to a user-local copy.
func (uc *ProvisionProvidersUseCase) updatePacmanOpenCode(ctx context.Context, tool providerCLI, add func(entity.DiagnosticStatus, string, string, string)) {
	mgr, ok := uc.managers[entity.PackageTypePacman]
	if !ok {
		add(entity.DiagWarning, tool.name, "pacman owns opencode but its package manager is unavailable",
			"Run 'envctl run pacman' after restoring the pacman manager")
		return
	}
	if err := mgr.Install(ctx, entity.Package{ID: "opencode", Type: entity.PackageTypePacman}); err != nil {
		add(entity.DiagWarning, tool.name, fmt.Sprintf("pacman could not update opencode: %v", err),
			"Run 'envctl run pacman' to update opencode")
		return
	}
	after := uc.installedVersion(ctx, tool.binary)
	if !versionMajorAtLeast(after, tool.requiredMajor) {
		add(entity.DiagWarning, tool.name,
			fmt.Sprintf("pacman update completed but OpenCode %s is not a verified v%d binary", printableVersion(after), tool.requiredMajor),
			"Update the distro repository/package, then run 'envctl run providers' again")
		return
	}
	add(entity.DiagOK, tool.name, fmt.Sprintf("Updated/verified OpenCode %s via pacman (extra)", printableVersion(after)), "")
}

// pacmanManagerAvailable reports whether this Linux host has a usable pacman
// manager. It is only an installation capability check, not proof that the
// opencode package is installed.
func (uc *ProvisionProvidersUseCase) pacmanManagerAvailable(ctx context.Context) bool {
	if runtime.GOOS != "linux" {
		return false
	}
	if mgr, ok := uc.managers[entity.PackageTypePacman]; ok {
		return mgr.IsAvailable(ctx)
	}
	_, err := resolveOnToolchainPath("pacman")
	return err == nil
}

// pacmanOwnsOpenCode queries pacman's package database directly. Leaving
// CheckCommand empty is intentional: a stale ~/.opencode/bin/opencode can make
// the generic check pass even when no pacman package is installed.
func (uc *ProvisionProvidersUseCase) pacmanOwnsOpenCode(ctx context.Context) bool {
	if runtime.GOOS != "linux" {
		return false
	}
	mgr, ok := uc.managers[entity.PackageTypePacman]
	if !ok || !mgr.IsAvailable(ctx) {
		return false
	}
	installed, _, err := mgr.IsInstalled(ctx, entity.Package{ID: "opencode", Type: entity.PackageTypePacman})
	return err == nil && installed
}

// archiveUserOpenCode moves an envctl-owned user-local binary aside when the
// distro package is authoritative. The timestamped backup follows the same
// convention as config provisioning and keeps a rollback path.
func archiveUserOpenCode(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("empty opencode path")
	}
	if _, err := os.Lstat(path); err != nil {
		return "", err
	}
	stamp := time.Now().Format("20060102-150405")
	candidate := fmt.Sprintf("%s.bak.%s", path, stamp)
	for i := 1; ; i++ {
		if _, err := os.Lstat(candidate); os.IsNotExist(err) {
			break
		}
		candidate = fmt.Sprintf("%s.bak.%s-%d", path, stamp, i)
	}
	if err := os.Rename(path, candidate); err != nil {
		return "", err
	}
	return candidate, nil
}

// requireStandaloneProviderVersion verifies the minimum major after an
// installer returns successfully. A successful download alone is not enough:
// the repository's V2-native config would still fail against a v1 binary.
func (uc *ProvisionProvidersUseCase) requireStandaloneProviderVersion(ctx context.Context, tool providerCLI, add func(entity.DiagnosticStatus, string, string, string)) bool {
	if tool.requiredMajor == 0 {
		return true
	}
	version := uc.installedVersion(ctx, tool.binary)
	if versionMajorAtLeast(version, tool.requiredMajor) {
		return true
	}
	details := fmt.Sprintf("installer completed but opencode %s is not a verified v%d binary", printableVersion(version), tool.requiredMajor)
	add(entity.DiagWarning, tool.name, details,
		"Run 'curl -fsSL https://opencode.ai/v2/install | bash' and verify 'opencode --version'")
	return false
}

func printableVersion(version string) string {
	if version == "" {
		return "(missing)"
	}
	return "v" + version
}

// installedVersion runs `<binary> --version` and reduces the output to the first
// version-looking token (tools report "opencode v2.0.5", "1.55.1", ...).
func (uc *ProvisionProvidersUseCase) installedVersion(ctx context.Context, binary string) string {
	resolved, err := resolveOnToolchainPath(binary)
	if err != nil {
		return ""
	}
	// exec.Cmd.Env does not affect binary resolution (LookPath uses the
	// process PATH), so execute the resolved absolute path.
	out, err := runWithToolchain(ctx, resolved, "--version")
	if err != nil {
		return ""
	}
	return firstVersionToken(out)
}

// installSource classifies where a binary comes from, which decides whether
// envctl may touch it.
func installSource(binary string) string {
	path, err := resolveOnToolchainPath(binary)
	if err != nil {
		return sourceAbsent
	}
	home, err := os.UserHomeDir()
	if err != nil {
		home = ""
	}
	return classifyInstallSource(path, home)
}

// npmLatest reads the "latest" dist-tag of an npm package.
func (uc *ProvisionProvidersUseCase) npmLatest(ctx context.Context, pkg string) string {
	out, err := runWithToolchain(ctx, "curl", "-fsSL", "https://registry.npmjs.org/"+pkg+"/latest")
	if err != nil {
		return ""
	}
	// Deliberately dependency-free: the registry answers with a JSON object whose
	// "version" field is all this needs.
	idx := strings.Index(out, `"version":"`)
	if idx < 0 {
		return ""
	}
	rest := out[idx+len(`"version":"`):]
	if end := strings.Index(rest, `"`); end > 0 {
		return rest[:end]
	}
	return ""
}

// firstVersionToken extracts the first token that starts with a digit, dropping
// the tool name and any leading "v".
func firstVersionToken(output string) string {
	for _, field := range strings.Fields(output) {
		candidate := strings.TrimPrefix(field, "v")
		if candidate == "" {
			continue
		}
		if candidate[0] >= '0' && candidate[0] <= '9' {
			return strings.TrimRight(candidate, ".,;")
		}
	}
	return ""
}

// versionsDiffer reports whether two version strings disagree once normalized
// (leading "v" and surrounding whitespace removed).
func versionsDiffer(installed, latest string) bool {
	return normalizeVersion(installed) != normalizeVersion(latest)
}

func normalizeVersion(version string) string {
	return strings.TrimPrefix(strings.TrimSpace(version), "v")
}

// Install sources. These are stable identifiers compared against in code; use
// sourceLabel for anything a human reads.
const (
	sourceAbsent = "absent"
	sourceVolta  = "volta"
	sourceEnvctl = "envctl"
	sourceSystem = "system"
)

// classifyInstallSource names the owner of a binary path: Volta's tool image,
// envctl's own prefix, or the system. Only the envctl prefix and Volta are safe
// for envctl to replace.
func classifyInstallSource(path, home string) string {
	normalized := filepath.ToSlash(path)
	switch {
	case home == "":
		return sourceSystem
	case strings.HasPrefix(normalized, filepath.ToSlash(filepath.Join(home, ".volta"))):
		return sourceVolta
	case strings.HasPrefix(normalized, filepath.ToSlash(filepath.Join(home, ".local"))),
		strings.HasPrefix(normalized, filepath.ToSlash(filepath.Join(home, ".opencode"))):
		return sourceEnvctl
	default:
		return sourceSystem
	}
}

// sourceLabel renders an install source for diagnostics.
func sourceLabel(source string) string {
	switch source {
	case sourceVolta:
		return "Volta"
	case sourceEnvctl:
		return "envctl"
	default:
		return "system package"
	}
}
