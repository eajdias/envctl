package usecase

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/eajdias/envctl/internal/domain/repository"
	"github.com/eajdias/envctl/internal/infra/executil"
	"github.com/eajdias/envctl/internal/infra/filesystem"
)

// mergeKwinrcCompositing ensures the [Compositing] section of a KDE kwinrc
// carries the fullscreen compositing bypass (AllowBlockCompositing=true +
// UnredirectFullscreen=true), preserving every other section and key. It is
// idempotent: an already-configured file comes back unchanged. Keeping the
// merge scoped to those two keys is what makes provisioning safe — a blind
// overwrite of kwinrc would discard the user's desktop preferences.
func mergeKwinrcCompositing(data []byte) string {
	lines := strings.Split(string(data), "\n")
	out := make([]string, 0, len(lines)+3)
	inCompositing := false
	haveBlock := false
	haveUnredirect := false

	for _, raw := range lines {
		line := raw
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[") {
			// Leaving the section: flush any missing keys before the next one.
			if inCompositing {
				out = append(out, missingCompositingKeys(haveBlock, haveUnredirect)...)
			}
			inCompositing = trimmed == "[Compositing]"
			haveBlock, haveUnredirect = false, false
			out = append(out, line)
			continue
		}
		if inCompositing {
			switch trimmed {
			case "AllowBlockCompositing=true":
				haveBlock = true
			case "UnredirectFullscreen=true":
				haveUnredirect = true
			}
		}
		out = append(out, line)
	}
	if inCompositing {
		out = append(out, missingCompositingKeys(haveBlock, haveUnredirect)...)
	}

	merged := strings.Join(out, "\n")
	if !strings.Contains(merged, "[Compositing]") {
		merged += "\n\n[Compositing]\nAllowBlockCompositing=true\nUnredirectFullscreen=true\n"
	}
	return merged
}

// missingCompositingKeys returns the bypass keys the [Compositing] section
// still lacks, ready to append.
func missingCompositingKeys(haveBlock, haveUnredirect bool) []string {
	var missing []string
	if !haveBlock {
		missing = append(missing, "AllowBlockCompositing=true")
	}
	if !haveUnredirect {
		missing = append(missing, "UnredirectFullscreen=true")
	}
	return missing
}

// scxLoaderConfig renders the sched_ext loader config the validated host
// applies: bpfland in Auto mode. The scheduler stays with the package default
// when the file already exists — the merge is a full overwrite of a tiny
// file, but only through the privileged apply step.
func scxLoaderConfig() []byte {
	return []byte(`default_sched = "scx_bpfland"
default_mode = "Auto"
`)
}

// lactConfigFor renders the LACT GPU control config for a detected AMD
// device. The fan curve is the conservative Polaris profile the validated
// host runs (ramp to 70°C, aggressive 75°C+, ceiling 85°C); performance level
// stays auto so clocks are only raised when the workload asks. The device id
// varies per machine — it always comes from the sysfs probe, never from a
// hardcoded PCI id.
func lactConfigFor(deviceID string) []byte {
	return []byte(`version: 7
daemon:
  log_level: info
  admin_group: wheel
  disable_clocks_cleanup: false
apply_settings_timer: 5
current_profile: null
auto_switch_profiles: false
gpus:
  "` + deviceID + `":
    fan_control_enabled: true
    fan_control_settings:
      mode: curve
      temperature_key: edge
      interval_ms: 500
      curve:
        40: 0.2
        55: 0.35
        65: 0.55
        75: 0.8
        85: 1.0
    performance_level: auto
`)
}

// applyCmdlineParams adds the wanted kernel parameters to an existing cmdline
// line, keeping every parameter already present and the parameter order of
// the original. Parameters are matched by their full `key=value` or bare key
// form; a wanted list is idempotent (second call is a no-op).
func applyCmdlineParams(current string, wanted []string) string {
	already := strings.Fields(current)
	have := make(map[string]bool, len(already))
	for _, p := range already {
		have[p] = true
	}
	var missing []string
	for _, w := range wanted {
		if !have[w] {
			missing = append(missing, w)
		}
	}
	if len(missing) == 0 {
		return current
	}
	merged := make([]string, 0, len(already)+len(missing))
	merged = append(merged, already...)
	merged = append(merged, missing...)
	return strings.Join(merged, " ")
}

// edenURL returns the pinned Eden AppImage download URL, choosing the legacy
// build (pre-AVX2 CPUs) or the standard build by the host capability.
// Version and asset names are pinned: the validated host runs v0.2.1 legacy.
func edenURL(avx2 bool, version string) string {
	base := "https://git.eden-emu.dev/eden-emu/eden/releases/download/" + version + "/"
	if avx2 {
		return base + "Eden-Linux-v" + version + "-gcc-standard.AppImage"
	}
	return base + "Eden-Linux-v" + version + "-legacy-gcc-standard.AppImage"
}

// applyLimineCmdline applies the wanted kernel parameters to every
// KERNEL_CMDLINE entry of a /etc/default/limine file. Limine entries look
// like `KERNEL_CMDLINE[default]+="<cmdline>"`; each quoted cmdline is merged
// with applyCmdlineParams and re-quoted in place. Entries already carrying
// the wanted parameters come back unchanged, so the file is only rewritten
// when something actually changes.
func applyLimineCmdline(data []byte, wanted []string) []byte {
	lines := strings.Split(string(data), "\n")
	changed := false
	for i, line := range lines {
		if !strings.Contains(line, "KERNEL_CMDLINE") {
			continue
		}
		openIdx := strings.Index(line, "\"")
		if openIdx < 0 {
			continue // malformed entry; leave it alone
		}
		closeIdx := strings.LastIndex(line, "\"")
		if closeIdx <= openIdx {
			continue
		}
		quoted := line[openIdx+1 : closeIdx]
		merged := applyCmdlineParams(quoted, wanted)
		if merged == quoted {
			continue
		}
		lines[i] = line[:openIdx+1] + merged + line[closeIdx:]
		changed = true
	}
	if !changed {
		return data
	}
	return []byte(strings.Join(lines, "\n"))
}

// cpuinfoPath is the proc file carrying the CPU flags. It is a variable so
// tests can point it at a fixture.
var cpuinfoPath = "/proc/cpuinfo"

// cpuHasAVX2 reports whether the host CPU supports AVX2, read from the
// `flags` line of /proc/cpuinfo. AVX2 gates the emulator ceiling (builds
// marked x86-64-v3/v4 are unusable without it) and the Eden build choice.
func cpuHasAVX2(cpuinfo string) bool {
	for _, raw := range strings.Split(cpuinfo, "\n") {
		line := strings.TrimSpace(raw)
		if !strings.HasPrefix(line, "flags") {
			continue
		}
		for _, flag := range strings.Fields(line) {
			if flag == "avx2" {
				return true
			}
		}
	}
	return false
}

// hostHasAVX2 reads the real /proc/cpuinfo and reports AVX2 support. It
// returns false on any read error (machines without the proc file cannot
// possibly run AVX2 builds anyway).
func hostHasAVX2() bool {
	data, err := os.ReadFile(cpuinfoPath)
	if err != nil {
		return false
	}
	return cpuHasAVX2(string(data))
}

// amdgpuSysfsDir is the sysfs directory where DRM devices register. It is a
// variable so tests can point it at a fixture.
var amdgpuSysfsDir = "/sys/class/drm"

// hostAmdgpuDevice returns the sysfs device path of the first AMD GPU on the
// host, or "" when none is present. The LACT config step is gated on it: a
// machine without an AMD GPU skips the tuned config entirely.
func hostAmdgpuDevice() string {
	devices := amdgpuDevices(amdgpuSysfsDir)
	if len(devices) == 0 {
		return ""
	}
	return devices[0]
}

// amdDeviceID is the PCI vendor ID for AMD. Matching is done on the vendor
// file content, which sysfs writes as a zero-padded hex string ("0x1002").
const amdDeviceID = "0x1002"

// amdgpuDevices returns the sysfs device paths whose PCI vendor is AMD. A
// host without an AMD GPU yields an empty slice; the caller skips the
// AMD-only tuning steps (LACT config) in that case.
func amdgpuDevices(drmRoot string) []string {
	entries, err := os.ReadDir(drmRoot)
	if err != nil {
		return nil
	}
	var devices []string
	for _, entry := range entries {
		// card0, card1... are the render-capable devices; link names like
		// renderD128 are handled through the card symlink.
		if !strings.HasPrefix(entry.Name(), "card") {
			continue
		}
		deviceDir := filepath.Join(drmRoot, entry.Name(), "device")
		vendor, err := os.ReadFile(filepath.Join(deviceDir, "vendor"))
		if err != nil {
			continue
		}
		if strings.TrimSpace(string(vendor)) == amdDeviceID {
			devices = append(devices, deviceDir)
		}
	}
	return devices
}

// GamingTuningResult reports what the privileged tuning pass changed, so the
// CLI can tell the user what still needs a reboot or a manual confirmation.
type GamingTuningResult struct {
	KernelCmdlineChanged bool
	ScxLoaderWritten     bool
	LactWritten          bool
	KwinrcWritten        bool
	EdenInstalled        bool
	RebootRequired       bool
}

// ProvisionGamingTuningUseCase applies the manual tuning the validated
// CachyOS gaming stack needs, turning the audit-only checks of the doctor
// into provisioned state. Privileged files (/etc) are written through
// `sudo -n` after an interactive `sudo -v`; user files (kwinrc) go through
// the fs manager with its atomic backup.
type ProvisionGamingTuningUseCase struct {
	fsManager *filesystem.FileSystemManager
	logger    repository.Logger
}

func NewProvisionGamingTuningUseCase(fsManager *filesystem.FileSystemManager, logger repository.Logger) *ProvisionGamingTuningUseCase {
	return &ProvisionGamingTuningUseCase{fsManager: fsManager, logger: logger}
}

// Provision applies every step in dependency order. Each step is
// individually idempotent and skips silently when the machine does not need
// it (no AMD GPU, config already correct). A step failing on sudo does not
// stop the user-level steps that follow it.
func (uc *ProvisionGamingTuningUseCase) Provision(ctx context.Context) (*GamingTuningResult, error) {
	res := &GamingTuningResult{}

	if err := uc.provisionScxLoader(ctx, res); err != nil {
		uc.logger.Warn("scx_loader config step failed: %v", err)
	}
	if err := uc.provisionLact(ctx, res); err != nil {
		uc.logger.Warn("LACT config step failed: %v", err)
	}
	if err := uc.provisionKernelCmdline(ctx, res); err != nil {
		uc.logger.Warn("kernel cmdline step failed: %v", err)
	}
	if err := uc.provisionKwinrc(res); err != nil {
		uc.logger.Warn("kwinrc step failed: %v", err)
	}
	if err := uc.provisionEden(ctx, res); err != nil {
		uc.logger.Warn("Eden step failed: %v", err)
	}

	return res, nil
}

// provisionScxLoader writes /etc/scx_loader/config.toml (scheduler choice)
// when it does not already carry the wanted config.
func (uc *ProvisionGamingTuningUseCase) provisionScxLoader(ctx context.Context, res *GamingTuningResult) error {
	want := string(scxLoaderConfig())
	if data, err := os.ReadFile(scxLoaderConfigPath); err == nil && string(data) == want {
		uc.logger.Info("scx_loader config already correct")
		return nil
	}
	if err := executil.RunPrivileged(ctx, "tee", scxLoaderConfigPath); err != nil {
		return fmt.Errorf("cannot write %s: %w", scxLoaderConfigPath, err)
	}
	// Write through a temp file then mv, so a partial write never lands on a
	// live config. The mkstemp approach gives us the sudo pattern: the whole
	// operation is one sudo call chain.
	if err := executil.RunPrivileged(ctx, "bash", "-c",
		fmt.Sprintf("printf '%s' > %s.envctl-tmp && mv %s.envctl-tmp %s", want, scxLoaderConfigPath, scxLoaderConfigPath, scxLoaderConfigPath)); err != nil {
		return fmt.Errorf("cannot write %s: %w", scxLoaderConfigPath, err)
	}
	res.ScxLoaderWritten = true
	uc.logger.Info("scx_loader config written (bpfland/Auto)")
	// Enable the loader service only when the file actually changed.
	if err := executil.RunPrivileged(ctx, "systemctl", "enable", "--now", "scx_loader"); err != nil {
		return fmt.Errorf("cannot enable scx_loader: %w", err)
	}
	return nil
}

// provisionLact writes /etc/lact/config.yaml for the first AMD GPU found,
// with the conservative fan curve. Skips entirely on hosts without AMD.
func (uc *ProvisionGamingTuningUseCase) provisionLact(ctx context.Context, res *GamingTuningResult) error {
	device := hostAmdgpuDevice()
	if device == "" {
		uc.logger.Info("no AMD GPU detected; LACT config skipped")
		return nil
	}
	// The device id registered by LACT is the PCI id, not the sysfs path:
	// derive "VVVV:DDDD-..." from the sysfs device tree when available, else
	// fall back to the sysfs path (LACT accepts both).
	deviceID := device
	if data, err := os.ReadFile(filepath.Join(device, "uevent")); err == nil {
		deviceID = parsePCIIDFromUevent(string(data), deviceID)
	}
	uc.logger.Info("LACT device: %s", deviceID)

	want := string(lactConfigFor(deviceID))
	if data, err := os.ReadFile(lactConfigPath); err == nil && string(data) == want {
		uc.logger.Info("LACT config already correct")
		return nil
	}
	if err := executil.RunPrivileged(ctx, "bash", "-c",
		fmt.Sprintf("printf '%s' > %s.envctl-tmp && mv %s.envctl-tmp %s", want, lactConfigPath, lactConfigPath, lactConfigPath)); err != nil {
		return fmt.Errorf("cannot write %s: %w", lactConfigPath, err)
	}
	res.LactWritten = true
	uc.logger.Info("LACT config written (fan curve + auto performance level)")
	if err := executil.RunPrivileged(ctx, "systemctl", "enable", "--now", "lactd"); err != nil {
		return fmt.Errorf("cannot enable lactd: %w", err)
	}
	return nil
}

// provisionKernelCmdline appends the wanted params to every Limine
// KERNEL_CMDLINE entry and runs limine-update when the file changed.
func (uc *ProvisionGamingTuningUseCase) provisionKernelCmdline(ctx context.Context, res *GamingTuningResult) error {
	const limineDefault = "/etc/default/limine"
	data, err := os.ReadFile(limineDefault)
	if err != nil {
		return fmt.Errorf("cannot read %s: %w", limineDefault, err)
	}
	wanted := append([]string{}, gamingKernelParams...)
	if amdgpuModulePresent("/sys/module/amdgpu") {
		wanted = append(wanted, gamingAMDKernelParams...)
	}
	wanted = append(wanted, gamingPanicParams...)
	merged := applyLimineCmdline(data, wanted)
	if string(merged) == string(data) {
		uc.logger.Info("kernel cmdline already carries the wanted parameters")
		return nil
	}
	// Backup, write, update: three sudo calls, each independently verifiable.
	if err := executil.RunPrivileged(ctx, "cp", "-a", limineDefault, limineDefault+".envctl-bak"); err != nil {
		return fmt.Errorf("cannot back up %s: %w", limineDefault, err)
	}
	if err := executil.RunPrivileged(ctx, "bash", "-c",
		fmt.Sprintf("printf '%s' > %s.envctl-tmp && mv %s.envctl-tmp %s", string(merged), limineDefault, limineDefault, limineDefault)); err != nil {
		return fmt.Errorf("cannot write %s: %w", limineDefault, err)
	}
	if err := executil.RunPrivileged(ctx, "limine-update"); err != nil {
		return fmt.Errorf("limine-update failed (bootloader not rebuilt): %w", err)
	}
	res.KernelCmdlineChanged = true
	res.RebootRequired = true
	uc.logger.Info("kernel cmdline updated; reboot required for the new parameters")
	return nil
}

// provisionKwinrc merges the compositing bypass into the user kwinrc through
// the fs manager, preserving every other section (atomic backup built in).
func (uc *ProvisionGamingTuningUseCase) provisionKwinrc(res *GamingTuningResult) error {
	path, err := uc.fsManager.ExpandUserPath(kwinrcConfigPath)
	if err != nil || path == "" {
		return fmt.Errorf("cannot resolve %s: %w", kwinrcConfigPath, err)
	}
	data, readErr := uc.fsManager.ReadFile(kwinrcConfigPath)
	if readErr != nil {
		data = nil // file absent: create from scratch
	}
	merged := mergeKwinrcCompositing(data)
	if readErr == nil && string(data) == merged {
		uc.logger.Info("kwinrc compositing bypass already present")
		return nil
	}
	if _, err := uc.fsManager.WriteWithBackup(path, []byte(merged), 0o644); err != nil {
		return fmt.Errorf("cannot write %s: %w", kwinrcConfigPath, err)
	}
	res.KwinrcWritten = true
	uc.logger.Info("kwinrc compositing bypass merged")
	return nil
}

// edenVersion is the pinned Eden release the provisioner downloads. The
// validated host runs v0.2.1; the URL is generated by edenURL() with the
// legacy or standard asset chosen by the AVX2 capability.
const edenVersion = "v0.2.1"

// edenDir is where the AppImage and its launcher live. It is user-level, so
// the step needs no privilege.
const edenDir = "~/Applications/eden"

// provisionEden downloads the pinned Eden AppImage when it is not already in
// place, fingerprints it with a short SIGILL smoke test (the legacy build
// must run on this CPU), seeds the launcher and prints the manual firmware
// reminder. The download itself only writes user files.
func (uc *ProvisionGamingTuningUseCase) provisionEden(ctx context.Context, res *GamingTuningResult) error {
	dir, err := uc.fsManager.ExpandUserPath(edenDir)
	if err != nil || dir == "" {
		return fmt.Errorf("cannot resolve %s: %w", edenDir, err)
	}
	if err := uc.fsManager.EnsureDirectory(dir, 0o755); err != nil {
		return fmt.Errorf("cannot create %s: %w", dir, err)
	}

	appImage := filepath.Join(dir, fmt.Sprintf("Eden-Linux-v%s.AppImage", edenVersion))
	if uc.fsManager.Exists(appImage) {
		uc.logger.Info("Eden AppImage already present; smoke test re-run")
		if err := uc.smokeEden(ctx, appImage); err != nil {
			return err
		}
		return nil
	}

	url := edenURL(hostHasAVX2(), edenVersion)
	uc.logger.Info("downloading Eden %s from %s (AVX2: %t)", edenVersion, url, hostHasAVX2())
	if err := uc.downloadEden(ctx, url, appImage); err != nil {
		return fmt.Errorf("download %s failed: %w — download it manually and place it at %s", url, err, appImage)
	}
	if err := uc.smokeEden(ctx, appImage); err != nil {
		return fmt.Errorf("eden smoke test failed: %w", err)
	}
	if err := uc.seedEdenLauncher(appImage); err != nil {
		return err
	}
	res.EdenInstalled = true
	uc.logger.Info("Eden installed at %s", appImage)
	return nil
}

// downloadEden streams the pinned AppImage to disk with a progress-capable
// write (plain io.Copy) and chmods it executable. No credential, no PII.
func (uc *ProvisionGamingTuningUseCase) downloadEden(ctx context.Context, url, dest string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %s", resp.Status)
	}
	tmp := dest + ".part"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	if err := os.Chmod(tmp, 0o755); err != nil { //nolint:gosec // AppImage must be world-executable to launch from the .desktop entry
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, dest)
}

// smokeEden runs the AppImage with --version and a short timeout. The legacy
// build must execute on this CPU; a SIGILL (illegal instruction) means the
// wrong build asset was chosen and the file is unusable.
func (uc *ProvisionGamingTuningUseCase) smokeEden(ctx context.Context, appImage string) error {
	runCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(runCtx, appImage, "--version")
	out, err := cmd.CombinedOutput()
	if err != nil {
		// Illegal instruction is the fingerprint of an AVX2-only build on a
		// pre-AVX2 CPU. Any other execution failure is also fatal here.
		if strings.Contains(string(out), "Illegal instruction") || strings.Contains(string(out), "SIGILL") {
			return fmt.Errorf("eden build is not executable on this CPU (illegal instruction): %s", strings.TrimSpace(string(out)))
		}
		return fmt.Errorf("eden smoke test failed: %v (%s)", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// seedEdenLauncher writes the .desktop entry pointing at the AppImage, unless
// it already exists (user's own launcher wins).
func (uc *ProvisionGamingTuningUseCase) seedEdenLauncher(appImage string) error {
	appsDir, err := uc.fsManager.ExpandUserPath("~/.local/share/applications")
	if err != nil || appsDir == "" {
		return fmt.Errorf("cannot resolve applications dir: %w", err)
	}
	if err := uc.fsManager.EnsureDirectory(appsDir, 0o755); err != nil {
		return err
	}
	dest := filepath.Join(appsDir, "eden.desktop")
	if uc.fsManager.Exists(dest) {
		return nil
	}
	content := fmt.Sprintf(`[Desktop Entry]
Type=Application
Name=Eden (Switch)
Comment=Eden Switch emulator (pinned build)
Exec="%s" %%f
Terminal=false
Categories=Game;Emulator;
`, appImage)
	_, err = uc.fsManager.WriteWithBackup(dest, []byte(content), 0o644)
	return err
}

// parsePCIIDFromUevent extracts the full LACT device id from a sysfs uevent
// file. LACT v7 keys GPUs by "PCI_ID-PCI_SUBSYS_ID-PCI_SLOT_NAME" (for
// example "1002:6FDF-1002:0B31-0000:01:00.0"); a bare PCI_ID would not match
// the device and the fan curve would stay inert. Falls back to the given
// default when a required field is absent.
func parsePCIIDFromUevent(uevent, fallback string) string {
	fields := map[string]string{}
	for _, line := range strings.Split(uevent, "\n") {
		if k, v, ok := strings.Cut(line, "="); ok {
			fields[k] = v
		}
	}
	id := fields["PCI_ID"]
	if id == "" {
		return fallback
	}
	subsys := fields["PCI_SUBSYS_ID"]
	slot := fields["PCI_SLOT_NAME"]
	if subsys != "" {
		id += "-" + subsys
	}
	if slot != "" {
		id += "-" + slot
	}
	return id
}
