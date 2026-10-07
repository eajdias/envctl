package usecase

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/eajdias/envctl/internal/domain/entity"
	"github.com/eajdias/envctl/internal/domain/repository"
	"github.com/eajdias/envctl/internal/infra/embedded"
	"github.com/eajdias/envctl/internal/infra/environment"
	"github.com/eajdias/envctl/internal/infra/executil"
	"github.com/eajdias/envctl/internal/infra/filesystem"
)

// ProvisionProvidersUseCase is the "phase 0" preflight: before packages,
// toolchains or configs, make sure the agent CLIs themselves are present and
// current, so a fresh machine can reach the agents without a manual step.
//
// Policy per tool, learned from the machines this runs on: anything npm owns
// is installed and updated through npm (it resolves the npm latest, so
// "outdated" and "missing" are the same command). A binary owned by the OS is
// authoritative: on Arch, an envctl-owned user copy is archived so pacman keeps
// ownership; on Ubuntu/Debian, a legacy system v1 is the one deliberate
// exception because it cannot load the V2-native config.
type ProvisionProvidersUseCase struct {
	manifestRepo *embedded.ManifestRepository
	fsManager    *filesystem.FileSystemManager
	envManager   *environment.WindowsEnvManager
	managers     map[entity.PackageType]repository.PackageManager
	logger       repository.Logger
	// latestVersionFn resolves the newest available version of a standalone
	// provider binary. It is a field (not a bare function) so tests can inject a
	// deterministic latest without hitting the network; production wires the
	// default below.
	latestVersionFn func(ctx context.Context, binary string) string
}

func NewProvisionProvidersUseCase(
	manifestRepo *embedded.ManifestRepository,
	fsManager *filesystem.FileSystemManager,
	envManager *environment.WindowsEnvManager,
	managers map[entity.PackageType]repository.PackageManager,
	logger repository.Logger,
) *ProvisionProvidersUseCase {
	return &ProvisionProvidersUseCase{
		manifestRepo:    manifestRepo,
		fsManager:       fsManager,
		envManager:      envManager,
		managers:        managers,
		logger:          logger,
		latestVersionFn: defaultLatestProviderVersion,
	}
}

// logInfo logs through the optional logger.
func (uc *ProvisionProvidersUseCase) logInfo(format string, args ...any) {
	uc.logger.Info(format, args...)
}

// toolchainEnv builds the environment used for provider probes, so mise shims
// resolve even when envctl runs from a non-login shell (ssh, systemd, agent).
// The directory list lives in executil; this stays a thin wrapper because a
// Windows process PATH is already complete and must pass through untouched.
func toolchainEnv() []string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" || runtime.GOOS == "windows" {
		return os.Environ()
	}
	return executil.ToolchainEnv(home)
}

// runWithToolchain runs a command against the toolchain PATH and returns its
// trimmed combined output.
func runWithToolchain(ctx context.Context, name string, args ...string) (string, error) {
	return runWithToolchainEnv(ctx, toolchainEnv(), name, args...)
}

// runWithToolchainEnv is the testable core: it resolves the binary against the
// PATH declared in the supplied environment and runs it with that environment.
//
// Resolving first is not optional. exec.Command resolves the binary against the
// PROCESS PATH at construction time, so assigning cmd.Env afterwards never
// affects which executable runs. A non-login shell (ssh, systemd, an agent) has
// a minimal process PATH, so a tool the profile had just installed into
// ~/.local/share/mise/shims was reported as "executable file not found in $PATH". That made
// the Node runtime and the CommandCode CLI fail on a freshly provisioned VPS and
// forced a second `run all` to converge.
func runWithToolchainEnv(ctx context.Context, env []string, name string, args ...string) (string, error) {
	// Fall back to the bare name so a genuinely absent tool still produces the
	// familiar error from exec, naming the binary the operator expects.
	resolved := name
	if path, err := executil.LookPathIn(pathValueFromEnv(env), name); err == nil {
		resolved = path
	}
	cmd := exec.CommandContext(ctx, resolved, args...)
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

// pathValueFromEnv returns the PATH declared in env, or "" when there is none.
func pathValueFromEnv(env []string) string {
	for _, kv := range env {
		if v, ok := strings.CutPrefix(kv, "PATH="); ok {
			return v
		}
	}
	return ""
}

// resolveOnToolchainPath resolves against the PATH built by toolchainEnv(), so
// probes see what execution sees. It shares executil.LookPathIn with the runner, which
// matters: a probe and the command it guards must resolve a tool identically or
// the probe reports a tool as absent while the command could have run it.
func resolveOnToolchainPath(name string) (string, error) {
	pathValue := pathValueFromEnv(toolchainEnv())
	if pathValue == "" {
		return exec.LookPath(name)
	}
	return executil.LookPathIn(pathValue, name)
}

// openCodeLinuxV2Installer installs the official OpenCode V2 channel. The
// legacy https://opencode.ai/install endpoint still serves the 1.x line; the
// V2-native configs deployed by this repository require the /v2 installer.
// The official installer selects the current architecture, baseline, and musl
// artifact and manages the user's PATH. Shared by phase 0 and Linux bootstrap.
const openCodeLinuxV2Installer = `set -e
export PATH="$HOME/.opencode/bin:$HOME/.local/share/mise/shims:$HOME/.local/bin:$PATH"
curl -fsSL https://opencode.ai/v2/install | bash`

// openCodePathDir is the official V2 install directory persisted in shell
// profiles. Written in the literal $HOME form so the idempotence guard matches
// the lines the legacy shell installer wrote.
const openCodePathDir = "$HOME/.opencode/bin"

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
	npmPkg           string // npm package name, installed via mise (npm: backend); empty when the tool has its own installer
	windowsInstaller string // PowerShell installer used on Windows when there is no npm package
	installer        string // shell installer used on Linux when there is no npm package
	requiredMajor    int    // minimum compatible major version, zero when unconstrained
}

func providerCLIs() []providerCLI {
	return []providerCLI{
		{name: "CommandCode CLI", binary: "cmdc", npmPkg: "command-code"},
		{
			name:             "OpenCode CLI",
			binary:           "opencode",
			installer:        openCodeLinuxV2Installer,
			windowsInstaller: `$ErrorActionPreference='Stop'; $bin=Join-Path $HOME '.local\bin'; New-Item -ItemType Directory -Force -Path $bin | Out-Null; $ver=(Invoke-RestMethod 'https://opencode.ai/update/api/latest/cli/npm').version; $arch='x64'; if ($env:PROCESSOR_ARCHITECTURE -match 'ARM64') { $arch='arm64' }; $variant="opencode-windows-$arch.zip"; try { Add-Type -MemberDefinition '[DllImport("kernel32.dll")] public static extern bool IsProcessorFeaturePresent(int f);' -Name K32 -Namespace W32 -PassThru | Out-Null; if ($arch -eq 'x64' -and -not [W32.K32]::IsProcessorFeaturePresent(40)) { $variant='opencode-windows-x64-baseline.zip' } } catch {}; $url="https://opencode.ai/files/bin/$ver/$variant"; $tmp=Join-Path $env:TEMP 'opencode-win.zip'; Invoke-WebRequest -Uri $url -OutFile $tmp -UseBasicParsing; Expand-Archive -Path $tmp -DestinationPath $bin -Force; Remove-Item $tmp -Force; $p=[Environment]::GetEnvironmentVariable('Path','User'); if ($p -notlike "*$bin*") { [Environment]::SetEnvironmentVariable('Path', "$p;$bin", 'User') }`,
			requiredMajor:    2,
		},
	}
}

// Execute ensures mise, a default Node runtime and the provider CLIs. It is
// idempotent: a converged machine reports OK for everything and changes nothing.
func (uc *ProvisionProvidersUseCase) Execute(ctx context.Context) ([]entity.Diagnostic, error) {
	var diags []entity.Diagnostic
	add := func(status entity.DiagnosticStatus, target, details, hint string) {
		diags = append(diags, entity.Diagnostic{
			Category: status, System: "Providers", Target: target, Details: details, FixHint: hint,
		})
	}

	miseReady := uc.ensureMise(ctx, add)
	if !miseReady {
		// Standalone providers do not depend on mise (notably OpenCode's
		// official Linux installer), so still converge them when the npm-backed
		// provider path is unavailable.
		for _, tool := range providerCLIs() {
			if tool.npmPkg == "" {
				uc.ensureProviderCLI(ctx, tool, add)
				continue
			}
			if installed := installedVersion(ctx, tool.binary); installed == "" {
				add(entity.DiagWarning, tool.name,
					"Not installed and mise/npm are unavailable to install it",
					"Install mise, then run 'envctl run providers' again")
			} else {
				add(entity.DiagOK, tool.name, fmt.Sprintf("v%s available; update requires npm", installed), "")
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

// ensureMise installs mise when it is missing and reports whether it is
// available afterwards. mise has no self-update command: updating it means
// re-running its installer, which the bootstrap already does when it is absent.
func (uc *ProvisionProvidersUseCase) ensureMise(ctx context.Context, add func(entity.DiagnosticStatus, string, string, string)) bool {
	if version := installedVersion(ctx, "mise"); version != "" {
		add(entity.DiagOK, "Mise", fmt.Sprintf("v%s available", version), "")
		return true
	}

	if runtime.GOOS == "windows" {
		if mgr, ok := uc.managers[entity.PackageTypeWinget]; ok && mgr.IsAvailable(ctx) {
			if err := mgr.Install(ctx, entity.Package{ID: "jdx.mise", Type: entity.PackageTypeWinget}); err == nil {
				add(entity.DiagOK, "Mise", "Installed via winget (jdx.mise)", "")
				return true
			}
		}
		add(entity.DiagWarning, "Mise", "Not installed and winget could not install it",
			"Run 'envctl run winget' to install jdx.mise")
		return false
	}

	uc.logInfo("Providers: installing mise")
	installer := `curl -fsSL https://mise.run | sh`
	cmd := exec.CommandContext(ctx, "bash", "-lc", installer)
	if out, err := cmd.CombinedOutput(); err != nil {
		add(entity.DiagWarning, "Mise", fmt.Sprintf("Installer failed: %v (%s)", err, strings.TrimSpace(string(out))),
			"Run 'curl -fsSL https://mise.run | sh' manually, then 'envctl run providers'")
		return false
	}
	add(entity.DiagOK, "Mise", "Installed via the official installer", "")
	return true
}

// ensureNodeRuntime guarantees a default Node for mise to build tool shims on,
// using the same spec the manifest declares so both stay in step.
func (uc *ProvisionProvidersUseCase) ensureNodeRuntime(ctx context.Context, add func(entity.DiagnosticStatus, string, string, string)) {
	spec := uc.manifestNodeSpec()
	if spec == "" {
		return
	}
	if _, err := runWithToolchain(ctx, "mise", "which", "node"); err == nil {
		return
	}
	uc.logInfo("Providers: installing the default Node runtime (%s)", spec)
	out, err := runWithToolchain(ctx, "mise", "use", "-g", spec)
	if err != nil {
		add(entity.DiagWarning, "Node runtime", fmt.Sprintf("mise use -g %s failed: %v (%s)", spec, err, out),
			"Run 'mise use -g "+spec+"' manually")
		return
	}
	add(entity.DiagOK, "Node runtime", fmt.Sprintf("Installed %s as the default runtime", spec), "")
}

// manifestNodeSpec returns the Node spec the manifest pins for mise, so phase 0
// never invents a different one.
func (uc *ProvisionProvidersUseCase) manifestNodeSpec() string {
	pkgs, err := uc.manifestRepo.LoadPackages()
	if err != nil {
		return ""
	}
	for _, p := range pkgs {
		if p.Type == entity.PackageTypeMise && strings.HasPrefix(p.ID, "node@") {
			return p.ID
		}
	}
	return ""
}

// ensureProviderCLI installs the CLI when missing and updates it when npm owns
// it and the registry has moved on; a tool owned by the OS is reported instead.
func (uc *ProvisionProvidersUseCase) ensureProviderCLI(ctx context.Context, tool providerCLI, add func(entity.DiagnosticStatus, string, string, string)) {
	installed := installedVersion(ctx, tool.binary)
	source := installSource(tool.binary)
	pacmanOwns := tool.binary == "opencode" && uc.pacmanOwnsOpenCode(ctx)

	// A user-local copy must never keep winning over a package that pacman
	// owns. Archive every active non-system copy, then let the next probe see
	// the system binary; the package path is verified below before success.
	var ok bool
	installed, source, ok = archiveShadowedUserCopies(ctx, uc.logInfo, tool.name, tool.binary,
		"Remove the stale user-local opencode binary, then run 'envctl run providers' again",
		"Move the stale user-local binary aside, then run 'envctl run providers' again",
		pacmanOwns, installed, source,
		func(target, details, fixHint string) { add(entity.DiagWarning, target, details, fixHint) })
	if !ok {
		return
	}

	switch {
	case installed == "" && tool.npmPkg != "":
		mgr, ok := uc.managers[entity.PackageTypeMise]
		if !ok {
			uc.logInfo("Providers: mise manager unavailable for %s", tool.name)
			add(entity.DiagWarning, tool.name, "Not installed and the mise manager is unavailable",
				"Run 'mise install npm:"+tool.npmPkg+"' manually")
			return
		}
		if err := mgr.Install(ctx, entity.Package{ID: "npm:" + tool.npmPkg, Type: entity.PackageTypeMise}); err != nil {
			add(entity.DiagWarning, tool.name, fmt.Sprintf("mise install npm:%s failed: %v", tool.npmPkg, err),
				"Run 'mise install npm:"+tool.npmPkg+"' manually")
			return
		}
		add(entity.DiagOK, tool.name, "Installed via mise (npm backend)", "")

	case installed == "" && tool.npmPkg == "":
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

	case tool.binary == "opencode" && tool.npmPkg == "":
		// Same major, but the official channel may have moved on within it
		// (2.0.15 -> 2.0.23). The required-major guard above already passed, so
		// only a genuine newer minor/patch triggers the standalone installer.
		latest := uc.latestVersionFn(ctx, tool.binary)
		if latest == "" {
			add(entity.DiagOK, tool.name, fmt.Sprintf("v%s (latest unknown — kept as-is)", installed), "")
			return
		}
		if !openCodeWithinMajorUpdateNeeded(installed, latest, tool.requiredMajor) {
			add(entity.DiagOK, tool.name, fmt.Sprintf("v%s (current)", installed), "")
			return
		}
		if !standaloneProviderCanReplace(source, pacmanOwns) {
			add(entity.DiagOK, tool.name, fmt.Sprintf("v%s (%s; newer v%s available, update with its own manager)", installed, sourceLabel(source), latest), "")
			return
		}
		uc.logInfo("Providers: updating %s %s -> %s", tool.name, installed, latest)
		uc.installStandaloneProvider(ctx, tool, add, source != sourceSystem)

	case tool.npmPkg != "" && source == sourceEnvctl:
		latest := npmLatest(ctx, tool.npmPkg)
		if latest == "" || !versionsDiffer(installed, latest) {
			add(entity.DiagOK, tool.name, fmt.Sprintf("v%s (mise, current)", installed), "")
			return
		}
		uc.logInfo("Providers: updating %s (%s -> %s)", tool.name, installed, latest)
		mgr, ok := uc.managers[entity.PackageTypeMise]
		if !ok {
			add(entity.DiagWarning, tool.name, fmt.Sprintf("update to v%s skipped: the mise manager is unavailable", latest),
				"Run 'mise install npm:"+tool.npmPkg+"@latest' manually")
			return
		}
		if err := mgr.Install(ctx, entity.Package{ID: "npm:" + tool.npmPkg + "@latest", Type: entity.PackageTypeMise}); err != nil {
			add(entity.DiagWarning, tool.name, fmt.Sprintf("update to v%s failed: %v", latest, err),
				"Run 'mise install npm:"+tool.npmPkg+"@latest' manually")
			return
		}
		after := installedVersion(ctx, tool.binary)
		if after == "" || after == installed {
			// The shim map can lag a beat behind its own install; never claim
			// "updated X -> X" — report what was asked for instead.
			after = latest
		}
		add(entity.DiagOK, tool.name, fmt.Sprintf("Updated v%s -> v%s via npm", installed, after), "")

	default:
		// Present but not npm's. Never shadow a package-managed binary: the
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
	if uc.envManager == nil {
		return true
	}
	if _, err := uc.envManager.EnsurePathEntry(ctx, openCodePathDir); err != nil {
		add(entity.DiagWarning, tool.name, fmt.Sprintf("could not persist ~/.opencode/bin in shell profiles: %v", err),
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
	after := installedVersion(ctx, tool.binary)
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

// requireStandaloneProviderVersion verifies the minimum major after an
// installer returns successfully. A successful download alone is not enough:
// the repository's V2-native config would still fail against a v1 binary.
func (uc *ProvisionProvidersUseCase) requireStandaloneProviderVersion(ctx context.Context, tool providerCLI, add func(entity.DiagnosticStatus, string, string, string)) bool {
	if tool.requiredMajor == 0 {
		return true
	}
	version := installedVersion(ctx, tool.binary)
	if versionMajorAtLeast(version, tool.requiredMajor) {
		return true
	}
	details := fmt.Sprintf("installer completed but opencode %s is not a verified v%d binary", printableVersion(version), tool.requiredMajor)
	add(entity.DiagWarning, tool.name, details,
		"Run 'curl -fsSL https://opencode.ai/v2/install | bash' and verify 'opencode --version'")
	return false
}

// Install sources. These are stable identifiers compared against in code; use
// sourceLabel for anything a human reads.
const (
	sourceAbsent = "absent"
	sourceMise   = "mise"
	sourceEnvctl = "envctl"
	sourceSystem = "system"
)

// sourceLabel renders an install source for diagnostics.
func sourceLabel(source string) string {
	switch source {
	case sourceMise:
		return "mise"
	case sourceEnvctl:
		return "envctl"
	default:
		return "system package"
	}
}
