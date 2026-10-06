package usecase

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/eajdias/envctl/internal/domain/entity"
)

// gamingKernelParams are the performance kernel parameters the gaming stack
// expects on /proc/cmdline. mitigations=off is deliberately NOT managed here:
// it is a Spectre/Meltdown trade-off that stays a manual, approved decision.
var gamingKernelParams = []string{"preempt=full", "split_lock_detect=off", "zswap.enabled=0"}

// gamingAMDKernelParams are the GPU parameters that only make sense on an AMD
// host. They are required only when the amdgpu module is loaded, so an Intel
// or NVIDIA machine never warns about them. Kept out of gamingKernelParams on
// purpose: that list must stay hardware-agnostic.

var gamingAMDKernelParams = []string{"amdgpu.runpm=0", "amdgpu.aspm=0", "pcie_aspm=off", "amdgpu.gpu_recovery=0"}

// gamingPanicParams are the panic-stability parameters the managed
// workstation uses (oops=panic turns an oops into a panic instead of killing
// the process; panic=10 bounds the reboot loop). They are a deliberate
// stability choice, not a performance prerequisite, so their absence is
// reported as informational and never as a warning.

var gamingPanicParams = []string{"oops=panic", "panic=10"}

// amdgpuModulePresent reports whether the amdgpu kernel module is loaded, which
// is the gate for the AMD-only cmdline parameters.

func amdgpuModulePresent(moduleDir string) bool {
	info, err := os.Stat(moduleDir)
	return err == nil && info.IsDir()
}

// gamingServices are the daemons the gaming stack needs active.

var gamingServices = []string{"scx_loader", "lactd", "ananicy-cpp", "power-profiles-daemon"}

// pendingPacnewFiles returns the .pacnew files under an /etc tree, relative to
// it and sorted. pacman writes these when a package ships a config the admin
// edited, so their presence means the file on disk differs from the packaged
// one. pacman.d is included because mirrorlist and the repo files live there.

func pendingPacnewFiles(etcRoot string) []string {
	var found []string
	for _, dir := range []string{etcRoot, filepath.Join(etcRoot, "pacman.d")} {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".pacnew") {
				continue
			}
			rel, err := filepath.Rel(etcRoot, filepath.Join(dir, entry.Name()))
			if err != nil {
				rel = entry.Name()
			}
			found = append(found, filepath.ToSlash(rel))
		}
	}
	sort.Strings(found)
	return found
}

// mangoHudPresetPath is the MangoHud configuration seeded by `run shell`.

const mangoHudPresetPath = "~/.config/MangoHud/MangoHud.conf"

// gamingConfPath is the systemd user environment preset seeded by `run shell`.

const gamingConfPath = "~/.config/environment.d/gaming.conf"

// gamingConfRequiredKeys are the settings the gaming environment preset must
// carry. Both are written by `envctl run shell` (seed_if_missing), and both
// change observable behaviour: the shader cache size avoids recompiling
// gigabytes of cache every boot, and RADV_PERFTEST pins the GPL pipeline
// library instead of leaving it to the Mesa default.

var gamingConfRequiredKeys = []string{"MESA_SHADER_CACHE_MAX_SIZE=", "RADV_PERFTEST="}

// missingGamingConfKeys returns the required keys the preset does not set.
// Commented-out lines do not count: a commented setting is not in effect.

func missingGamingConfKeys(data []byte) []string {
	var missing []string
	for _, key := range gamingConfRequiredKeys {
		found := false
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), key) {
				found = true
				break
			}
		}
		if !found {
			missing = append(missing, key)
		}
	}
	return missing
}

// ananicyTypesMarker is a file owned solely by the ananicy ruleset package
// (cachyos-ananicy-rules). The ananicy-cpp daemon only creates /etc/ananicy.d
// and ships no file in it, so the service can be active with no rules at all.
// The gaming package audit already covers "the ruleset package is not
// installed"; this marker covers the drift the package loop cannot see, namely
// the package present while its rules were removed.

const ananicyTypesMarker = "/etc/ananicy.d/00-types.types"

// lactConfigPath is the LACT GPU control daemon config. The fan curve is a
// manual, privileged decision (and the GPU id varies per machine), so the
// audit only reports whether the config exists — never its content, and never
// auto-fix.

const lactConfigPath = "/etc/lact/config.yaml"

// scxLoaderConfigPath is the sched_ext loader config. The scheduler choice is
// a manual decision; the audit only reports presence.

const scxLoaderConfigPath = "/etc/scx_loader/config.toml"

// kwinrcConfigPath is the KDE compositor config in the user profile. The
// fullscreen compositing bypass is what removes stutter in games on the
// validated Polaris host.

const kwinrcConfigPath = "~/.config/kwinrc"

// kwinrcCompositingEnabled reports whether kwinrc carries the fullscreen
// compositing bypass: both AllowBlockCompositing=true and
// UnredirectFullscreen=true inside the [Compositing] section. Keys outside
// that section do not count — the section is what scopes them.

func kwinrcCompositingEnabled(data []byte) bool {
	section := false
	haveBlock, haveUnredirect := false, false
	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		switch {
		case strings.HasPrefix(line, "["):
			section = line == "[Compositing]"
		case section && line == "AllowBlockCompositing=true":
			haveBlock = true
		case section && line == "UnredirectFullscreen=true":
			haveUnredirect = true
		}
	}
	return haveBlock && haveUnredirect
}

// emulatorConfigCheck pins one emulator config file and the key that proves
// the Vulkan backend tuning was applied on the validated host.

type emulatorConfigCheck struct {
	target string // doctor diagnostic target
	path   string // config file under the user profile
	key    string // line fragment that must be present, uncommented
}

// gamingEmulatorConfigs are the renderer settings the validated host applied
// by hand (Vulkan everywhere; resolution is the sane ceiling for its CPU).
// They are informational by design: each emulator has its own GUI and config
// conventions, so the audit never auto-fixes them.

var gamingEmulatorConfigs = []emulatorConfigCheck{
	{target: "emulator-config dolphin", path: "~/.config/dolphin-emu/Dolphin.ini", key: "GFXBackend = Vulkan"},
	{target: "emulator-config retroarch", path: "~/.config/retroarch/retroarch.cfg", key: `video_driver = "vulkan"`},
	{target: "emulator-config ppsspp", path: "~/.config/ppsspp/PSP/SYSTEM/ppsspp.ini", key: "GraphicsBackend = 3"},
	{target: "emulator-config pcsx2", path: "~/.config/PCSX2/inis/PCSX2.ini", key: "Renderer = 14"},
	{target: "emulator-config duckstation", path: "~/.config/duckstation/settings.ini", key: "Renderer = Vulkan"},
	{target: "emulator-config azahar", path: "~/.config/azahar-emu/qt-config.ini", key: "graphics_api=2"},
	{target: "emulator-config eden", path: "~/.config/eden/qt-config.ini", key: "backend=1"},
	{target: "emulator-config vita3k", path: "~/.config/Vita3K/config.yml", key: "backend-renderer: Vulkan"},
	{target: "emulator-config cemu", path: "~/.config/Cemu/settings.xml", key: "<api>1</api>"},
}

// configHasKey reports whether data contains line containing key, ignoring
// commented-out lines (# or ;). Emulator configs mix INI, YAML and XML; the
// fragment match is deliberately loose so the same helper works across them.

func configHasKey(data []byte, key string) bool {
	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.Contains(line, key) {
			return true
		}
	}
	return false
}

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
			addDiag(entity.Warn(
				"Gaming",
				pkg.ID,
				fmt.Sprintf("Check failed: %v", checkErr),
				"run 'envctl run gaming' to install the missing gaming packages",
			))
		case !installed:
			addDiag(entity.Warn(
				"Gaming",
				pkg.ID,
				"Not installed (gaming stack opted in via Steam)",
				"run 'envctl run gaming' to install the missing gaming packages",
			))
		default:
			addDiag(entity.OK(
				"Gaming",
				pkg.ID,
				fmt.Sprintf("Installed (%s)", info),
			))
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
				addDiag(entity.Warn(
					"Gaming",
					"service "+svc,
					"Service is not active",
					fmt.Sprintf("run 'systemctl enable --now %s' in your own terminal (password required)", svc),
				))
			} else {
				addDiag(entity.OK(
					"Gaming",
					"service "+svc,
					"Service is active",
				))
			}
		}
	}
	// CPU capability is informational context: it gates the emulator ceiling
	// (x86-64-v2 without AVX2 cannot run v3/v4 builds) and the Eden build
	// choice, but the hardware cannot be changed — INFO, never WARN.
	if hostHasAVX2() {
		addDiag(entity.Info(
			"Gaming",
			"cpu-capability",
			"CPU supports AVX2 (full emulator ceiling; Eden standard build)",
		))
	} else {
		addDiag(entity.Info(
			"Gaming",
			"cpu-capability",
			"CPU lacks AVX2 (x86-64-v2: keep generic repos and AppImage legacy builds; RPCS3/PS3, xemu, simple64 and Switch AAA are not viable)",
		))
	}
	if !uc.fsManager.Exists(ananicyTypesMarker) {
		addDiag(entity.Warn(
			"Gaming",
			"ananicy-rules",
			"ananicy-cpp has no ruleset; the daemon runs but every process gets the default priority",
			"run 'envctl run gaming' to install cachyos-ananicy-rules (the ananicy-cpp daemon alone ships zero rules)",
		))
	} else {
		addDiag(entity.OK(
			"Gaming",
			"ananicy-rules",
			"ananicy ruleset present",
		))
	}
	if data, err := os.ReadFile("/sys/kernel/sched_ext/state"); err == nil {
		if state := strings.TrimSpace(string(data)); state != "enabled" {
			addDiag(entity.Warn(
				"Gaming",
				"sched_ext",
				fmt.Sprintf("sched_ext state is '%s', scx schedulers are inert", state),
				"run 'systemctl enable --now scx_loader' in your own terminal (password required)",
			))
		} else {
			addDiag(entity.OK(
				"Gaming",
				"sched_ext",
				"sched_ext enabled (scx scheduler running)",
			))
		}
	}
	if data, err := os.ReadFile("/proc/cmdline"); err == nil {
		if missing := missingCmdlineParams(string(data), gamingKernelParams); len(missing) > 0 {
			addDiag(entity.Warn(
				"Gaming",
				"kernel cmdline",
				fmt.Sprintf("Missing performance parameters: %s", strings.Join(missing, ", ")),
				"edit KERNEL_CMDLINE in /etc/default/limine, run 'limine-update' and reboot (password required; see docs/guides/cachyos-gaming.md)",
			))
		} else {
			addDiag(entity.OK(
				"Gaming",
				"kernel cmdline",
				"Performance parameters present (preempt, split_lock, zswap)",
			))
		}

		// The AMD parameters are only required where the driver is present:
		// on Intel or NVIDIA there is nothing to configure and nothing to warn
		// about, so the check stays silent instead of reporting a false gap.
		if amdgpuModulePresent("/sys/module/amdgpu") {
			if missing := missingCmdlineParams(string(data), gamingAMDKernelParams); len(missing) > 0 {
				addDiag(entity.Warn(
					"Gaming",
					"kernel cmdline (amdgpu)",
					fmt.Sprintf("Missing AMD GPU parameters: %s", strings.Join(missing, ", ")),
					"edit KERNEL_CMDLINE in /etc/default/limine, run 'limine-update' and reboot (password required; see docs/guides/cachyos-gaming.md)",
				))
			} else {
				addDiag(entity.OK(
					"Gaming",
					"kernel cmdline (amdgpu)",
					fmt.Sprintf("AMD GPU parameters present (%d checked)", len(gamingAMDKernelParams)),
				))
			}
		}

		// Informational by design: panic=10 and oops=panic bound reboot loops
		// instead of improving performance, so a stock kernel is not a gap.
		if missing := missingCmdlineParams(string(data), gamingPanicParams); len(missing) > 0 {
			addDiag(entity.Info(
				"Gaming",
				"kernel cmdline (panic)",
				fmt.Sprintf("Optional stability parameters not set: %s", strings.Join(missing, ", ")),
				"optional: oops=panic panic=10 turn an oops into a panic and bound the reboot loop; add to KERNEL_CMDLINE in /etc/default/limine if you want crash visibility",
			))
		} else {
			addDiag(entity.OK(
				"Gaming",
				"kernel cmdline (panic)",
				"Panic-stability parameters present (oops=panic, panic=10)",
			))
		}
	}
	if vulkaninfo, err := exec.LookPath("vulkaninfo"); err == nil {
		if out, err := exec.CommandContext(ctx, vulkaninfo).Output(); err != nil || !strings.Contains(string(out), "RADV") {
			addDiag(entity.Warn(
				"Gaming",
				"Vulkan driver",
				"RADV not reported by vulkaninfo (Polaris must stay on RADV, never AMDVLK)",
				"check 'vulkaninfo | grep RADV' (see docs/guides/cachyos-gaming.md)",
			))
		} else {
			addDiag(entity.OK(
				"Gaming",
				"Vulkan driver",
				"RADV active",
			))
		}
	}
	if gamingConf, err := uc.fsManager.ExpandUserPath(gamingConfPath); err == nil && gamingConf != "" {
		data, readErr := uc.fsManager.ReadFile(gamingConf)
		if readErr != nil {
			addDiag(entity.Warn(
				"Gaming",
				"shader cache preset",
				fmt.Sprintf("Cannot read %s", gamingConf),
				"run 'envctl run shell' to seed the gaming environment preset, then relog",
			))
		} else if missing := missingGamingConfKeys(data); len(missing) > 0 {
			addDiag(entity.Warn(
				"Gaming",
				"shader cache preset",
				fmt.Sprintf("~/.config/environment.d/gaming.conf misses %s", strings.Join(missing, ", ")),
				"run 'envctl run shell' to seed the gaming environment preset, then relog",
			))
		} else {
			addDiag(entity.OK(
				"Gaming",
				"shader cache preset",
				fmt.Sprintf("gaming.conf configured (%s)", strings.Join(gamingConfRequiredKeys, ", ")),
			))
		}
	}
	if !uc.fsManager.Exists(mangoHudPresetPath) {
		addDiag(entity.Warn(
			"Gaming",
			"MangoHud preset",
			"~/.config/MangoHud/MangoHud.conf missing (no overlay metrics, no F12 toggle)",
			"run 'envctl run shell' to seed the MangoHud preset (goverlay can edit it afterwards)",
		))
	} else {
		addDiag(entity.OK(
			"Gaming",
			"MangoHud preset",
			"MangoHud preset present",
		))
	}
	if !uc.fsManager.Exists("/usr/bin/X") {
		addDiag(entity.Warn(
			"Gaming",
			"X11 session",
			"/usr/bin/X missing (plasma-x11-session cannot start without xorg-server)",
			"run 'envctl run gaming' to install xorg-server",
		))
	} else {
		addDiag(entity.OK(
			"Gaming",
			"X11 session",
			"/usr/bin/X present",
		))
	}
	if data, err := os.ReadFile("/etc/pacman.conf"); err == nil {
		if !multilibEnabled(string(data)) {
			addDiag(entity.Warn(
				"Gaming",
				"multilib repo",
				"[multilib] not enabled in /etc/pacman.conf (Steam and lib32-* require it)",
				"uncomment [multilib] in /etc/pacman.conf in your own terminal (password required)",
			))
		} else {
			addDiag(entity.OK(
				"Gaming",
				"multilib repo",
				"[multilib] enabled",
			))
		}
	}

	// Privileged tuning files are informational by design: the fan curve, the
	// scheduler choice and the compositor bypass are manual decisions (root or
	// reboot required), so absence is context, not a health problem. Presence
	// of the file means the owner did the manual step.
	if !uc.fsManager.Exists(lactConfigPath) {
		addDiag(entity.Info(
			"Gaming",
			"lact-config",
			"/etc/lact/config.yaml missing (no fan curve or power tuning applied)",
			"manual: set the fan curve in LACT (lact gui) — optional, see docs/guides/cachyos-gaming.md",
		))
	} else {
		addDiag(entity.OK(
			"Gaming",
			"lact-config",
			"LACT config present (fan curve/tuning applied)",
		))
	}
	if !uc.fsManager.Exists(scxLoaderConfigPath) {
		addDiag(entity.Info(
			"Gaming",
			"scx-loader-config",
			"/etc/scx_loader/config.toml missing (scheduler stayed at the package default)",
			"manual: choose the scheduler in /etc/scx_loader/config.toml — optional, see docs/guides/cachyos-gaming.md",
		))
	} else {
		addDiag(entity.OK(
			"Gaming",
			"scx-loader-config",
			"scx_loader config present (scheduler chosen)",
		))
	}
	if data, err := uc.fsManager.ReadFile(kwinrcConfigPath); err != nil || !kwinrcCompositingEnabled(data) {
		addDiag(entity.Info(
			"Gaming",
			"kwinrc-compositing",
			"~/.config/kwinrc missing the fullscreen compositing bypass (AllowBlockCompositing + UnredirectFullscreen in [Compositing])",
			"manual: set the two keys in [Compositing] on kwinrc and relog — optional, see docs/guides/cachyos-gaming.md",
		))
	} else {
		addDiag(entity.OK(
			"Gaming",
			"kwinrc-compositing",
			"fullscreen compositing bypass active",
		))
	}

	// Emulator renderer configs are informational by design: each emulator has
	// its own GUI, so the audit only records whether the Vulkan tuning the
	// validated host applies by hand is present.
	for _, check := range gamingEmulatorConfigs {
		data, err := uc.fsManager.ReadFile(check.path)
		if err != nil || !configHasKey(data, check.key) {
			addDiag(entity.Info(
				"Gaming",
				check.target,
				fmt.Sprintf("%s missing the Vulkan tuning (%s)", check.path, check.key),
				"manual: set the renderer to Vulkan in the emulator GUI — optional, see docs/guides/cachyos-gaming.md",
			))
		} else {
			addDiag(entity.OK(
				"Gaming",
				check.target,
				"Vulkan tuning applied",
			))
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
			addDiag(entity.OK(
				"Debloat",
				"category "+cat,
				fmt.Sprintf("Fully applied (%d/%d)", st.applied, st.total),
			))
		} else {
			addDiag(entity.Info(
				"Debloat",
				"category "+cat,
				fmt.Sprintf("Opt-in stack: %d/%d applied", st.applied, st.total),
				"run 'envctl run debloat' as Administrator to apply (opt-in, never auto-fixed)",
			))
		}
	}
}

// openCodeFileRefPattern matches opencode `{file:...}` variable references
// embedded in config values (e.g. MCP headers pointing at a secrets file).
