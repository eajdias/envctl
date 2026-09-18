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

// ProvisionProvidersUseCase is the "phase 0" preflight: before packages,
// toolchains or configs, make sure the agent CLIs themselves are present and
// current, so a fresh machine can reach the agents without a manual step.
//
// Policy per tool, learned from the machines this runs on: anything Volta owns
// is installed and updated through Volta (it resolves the npm latest, so
// "outdated" and "missing" are the same command); anything the OS package
// manager owns is reported, never shadowed — a second copy under ~/.local/bin
// would silently win on PATH and freeze the machine at whichever version envctl
// happened to install.
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
	if uc.logger != nil {
		uc.logger.Info(format, args...)
	}
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

// providerCLI is one agent-facing CLI phase 0 is responsible for.
type providerCLI struct {
	name      string // human name for diagnostics
	binary    string // binary that must resolve on PATH
	voltaPkg  string // Volta/npm package name, empty when the tool has its own installer
	wingetID  string // winget id used on Windows when there is no Volta package
	installer string // shell installer used on Linux when there is no Volta package
}

func providerCLIs() []providerCLI {
	return []providerCLI{
		{name: "CommandCode CLI", binary: "cmdc", voltaPkg: "command-code"},
		{
			name:      "OpenCode CLI",
			binary:    "opencode",
			wingetID:  "SST.opencode",
			installer: "curl -fsSL https://opencode.ai/install | bash",
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

	if !uc.ensureVolta(ctx, add) {
		// Without Volta there is no supported path to the Node-packaged CLIs.
		for _, tool := range providerCLIs() {
			if tool.voltaPkg != "" && uc.installedVersion(ctx, tool.binary) == "" {
				add(entity.DiagWarning, tool.name,
					"Not installed and Volta is unavailable to install it",
					"Install Volta, then run 'envctl run providers' again")
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
		uc.installStandaloneProvider(ctx, tool, add)

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
		add(entity.DiagOK, tool.name, fmt.Sprintf("v%s (%s; update it with its own manager)", installed, sourceLabel(source)), "")
	}
}

// installStandaloneProvider installs a provider that ships its own installer
// instead of an npm package (OpenCode on Linux, winget on Windows).
func (uc *ProvisionProvidersUseCase) installStandaloneProvider(ctx context.Context, tool providerCLI, add func(entity.DiagnosticStatus, string, string, string)) {
	if runtime.GOOS == "windows" && tool.wingetID != "" {
		if mgr, ok := uc.managers[entity.PackageTypeWinget]; ok && mgr.IsAvailable(ctx) {
			if err := mgr.Install(ctx, entity.Package{ID: tool.wingetID, Type: entity.PackageTypeWinget}); err == nil {
				add(entity.DiagOK, tool.name, "Installed via winget ("+tool.wingetID+")", "")
				return
			}
		}
		add(entity.DiagWarning, tool.name, "Not installed and winget could not install it",
			"Run 'envctl run winget' to install "+tool.wingetID)
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
	add(entity.DiagOK, tool.name, "Installed via its official installer", "")
}

// installedVersion runs `<binary> --version` and reduces the output to the first
// version-looking token (tools report "opencode v2.0.5", "1.55.1", ...).
func (uc *ProvisionProvidersUseCase) installedVersion(ctx context.Context, binary string) string {
	if _, err := exec.LookPath(binary); err != nil {
		return ""
	}
	out, err := runWithToolchain(ctx, binary, "--version")
	if err != nil {
		return ""
	}
	return firstVersionToken(out)
}

// installSource classifies where a binary comes from, which decides whether
// envctl may touch it.
func installSource(binary string) string {
	path, err := exec.LookPath(binary)
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
	case strings.HasPrefix(normalized, filepath.ToSlash(filepath.Join(home, ".local"))):
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
