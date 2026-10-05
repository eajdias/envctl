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
)

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
	fsManager repository.FileSystemManager
	logger    repository.Logger
}

func NewProvisionGamingTuningUseCase(fsManager repository.FileSystemManager, logger repository.Logger) *ProvisionGamingTuningUseCase {
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
	if err := runPrivileged(ctx, "tee", scxLoaderConfigPath); err != nil {
		return fmt.Errorf("cannot write %s: %w", scxLoaderConfigPath, err)
	}
	// Write through a temp file then mv, so a partial write never lands on a
	// live config. The mkstemp approach gives us the sudo pattern: the whole
	// operation is one sudo call chain.
	if err := runPrivileged(ctx, "bash", "-c",
		fmt.Sprintf("printf '%s' > %s.envctl-tmp && mv %s.envctl-tmp %s", want, scxLoaderConfigPath, scxLoaderConfigPath, scxLoaderConfigPath)); err != nil {
		return fmt.Errorf("cannot write %s: %w", scxLoaderConfigPath, err)
	}
	res.ScxLoaderWritten = true
	uc.logger.Info("scx_loader config written (bpfland/Auto)")
	// Enable the loader service only when the file actually changed.
	if err := runPrivileged(ctx, "systemctl", "enable", "--now", "scx_loader"); err != nil {
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
	if err := runPrivileged(ctx, "bash", "-c",
		fmt.Sprintf("printf '%s' > %s.envctl-tmp && mv %s.envctl-tmp %s", want, lactConfigPath, lactConfigPath, lactConfigPath)); err != nil {
		return fmt.Errorf("cannot write %s: %w", lactConfigPath, err)
	}
	res.LactWritten = true
	uc.logger.Info("LACT config written (fan curve + auto performance level)")
	if err := runPrivileged(ctx, "systemctl", "enable", "--now", "lactd"); err != nil {
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
	if err := runPrivileged(ctx, "cp", "-a", limineDefault, limineDefault+".envctl-bak"); err != nil {
		return fmt.Errorf("cannot back up %s: %w", limineDefault, err)
	}
	if err := runPrivileged(ctx, "bash", "-c",
		fmt.Sprintf("printf '%s' > %s.envctl-tmp && mv %s.envctl-tmp %s", string(merged), limineDefault, limineDefault, limineDefault)); err != nil {
		return fmt.Errorf("cannot write %s: %w", limineDefault, err)
	}
	if err := runPrivileged(ctx, "limine-update"); err != nil {
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
