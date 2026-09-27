package performance

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/eajdias/envctl/internal/domain/entity"
	"github.com/eajdias/envctl/internal/domain/repository"
)

type zramManager struct {
	hasDevice   func() bool
	hasNode     func() bool
	run         commandRunner
	elevate     bool
	wait        func(context.Context, func() bool) bool
	listDevices func() []string
}

// NewZRAMManager creates the production zram lifecycle adapter.
func NewZRAMManager() repository.ZRAMManager {
	manager := newZRAMManager(
		func() bool {
			data, err := os.ReadFile("/proc/swaps")
			return err == nil && procSwapsContainsZRAM0(string(data))
		},
		execCommand,
		needsElevation(),
		waitForDevice,
		listZRAMDevices,
	)
	manager.hasNode = func() bool {
		_, err := os.Stat("/dev/zram0")
		return err == nil
	}
	return manager
}

func newZRAMManager(
	hasDevice func() bool,
	run commandRunner,
	elevate bool,
	wait func(context.Context, func() bool) bool,
	listDevices ...func() []string,
) *zramManager {
	var list func() []string
	if len(listDevices) > 0 {
		list = listDevices[0]
	}
	return &zramManager{
		hasDevice:   hasDevice,
		hasNode:     func() bool { return true },
		run:         run,
		elevate:     elevate,
		wait:        wait,
		listDevices: list,
	}
}

// Ensure brings the compressed-RAM device up when the resolved policy wants it.
// An existing device is adopted, and a second pre-existing device is still
// refused rather than duplicated.
func (m *zramManager) Ensure(ctx context.Context, dryRun bool) ([]entity.Diagnostic, error) {
	if m.hasDevice != nil && m.hasDevice() {
		return []entity.Diagnostic{{
			Category: entity.DiagOK,
			System:   "Performance",
			Target:   "zram",
			Details:  "zram device is active",
		}}, nil
	}
	if other := otherZRAMDevices(m.existingDevices()); len(other) > 0 {
		return []entity.Diagnostic{{
			Category: entity.DiagInfo,
			System:   "Performance",
			Target:   "zram",
			Details:  "existing zram device(s) " + strings.Join(other, ", ") + " detected; refusing to create a second device",
		}}, nil
	}
	if dryRun {
		return []entity.Diagnostic{{
			Category: entity.DiagInfo,
			System:   "Performance",
			Target:   "zram",
			Details:  "zram device is absent; would start " + zramService,
		}}, nil
	}

	if out, err := m.command(ctx, "modprobe", "zram"); err != nil {
		return []entity.Diagnostic{{
			Category: entity.DiagError,
			System:   "Performance",
			Target:   "zram",
			Details:  fmt.Sprintf("load zram module failed: %v (%s)", err, strings.TrimSpace(string(out))),
		}}, err
	}
	if m.wait == nil || !m.wait(ctx, m.hasNode) {
		return []entity.Diagnostic{{
			Category: entity.DiagError,
			System:   "Performance",
			Target:   "zram",
			Details:  "zram module loaded but /dev/zram0 did not appear; inspect udev/module loading",
		}}, fmt.Errorf("zram module loaded without a device")
	}
	if other := otherZRAMDevices(m.existingDevices()); len(other) > 0 {
		return []entity.Diagnostic{{
			Category: entity.DiagInfo,
			System:   "Performance",
			Target:   "zram",
			Details:  "zram0 appeared alongside existing device(s) " + strings.Join(other, ", ") + "; refusing to start a second setup service",
		}}, nil
	}

	if out, err := m.command(ctx, "systemctl", "daemon-reload"); err != nil {
		return []entity.Diagnostic{{
			Category: entity.DiagError,
			System:   "Performance",
			Target:   "zram",
			Details:  fmt.Sprintf("reload systemd units for zram failed: %v (%s)", err, strings.TrimSpace(string(out))),
		}}, err
	}
	if out, err := m.command(ctx, "systemctl", "start", zramService); err != nil {
		return []entity.Diagnostic{{
			Category: entity.DiagError,
			System:   "Performance",
			Target:   "zram",
			Details:  fmt.Sprintf("start %s failed: %v (%s)", zramService, err, strings.TrimSpace(string(out))),
		}}, err
	}

	if m.wait == nil || !m.wait(ctx, m.hasDevice) {
		return []entity.Diagnostic{{
			Category: entity.DiagError,
			System:   "Performance",
			Target:   "zram",
			Details:  fmt.Sprintf("%s did not produce an active /dev/zram0 entry in /proc/swaps; reboot or inspect the generator", zramService),
		}}, fmt.Errorf("zram service completed without an active device")
	}
	return []entity.Diagnostic{{
		Category: entity.DiagOK,
		System:   "Performance",
		Target:   "zram",
		Details:  "zram device started successfully",
	}}, nil
}

func (m *zramManager) existingDevices() []string {
	if m.listDevices != nil {
		return m.listDevices()
	}
	if m.hasDevice != nil && m.hasDevice() {
		return []string{"/dev/zram0"}
	}
	return nil
}

func otherZRAMDevices(devices []string) []string {
	var other []string
	for _, device := range devices {
		if !strings.HasPrefix(filepath.Base(device), "zram0") {
			other = append(other, device)
		}
	}
	return other
}

func (m *zramManager) command(ctx context.Context, name string, args ...string) ([]byte, error) {
	if m.elevate {
		return m.run(ctx, "sudo", append([]string{"-n", name}, args...)...)
	}
	return m.run(ctx, name, args...)
}

func procSwapsContainsZRAM0(data string) bool {
	for _, line := range strings.Split(data, "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 3 && fields[0] == "/dev/zram0" {
			return true
		}
	}
	return false
}

func listZRAMDevices() []string {
	var devices []string
	for _, pattern := range []string{"/dev/zram*", "/sys/block/zram*"} {
		paths, err := filepath.Glob(pattern)
		if err != nil {
			continue
		}
		devices = append(devices, paths...)
	}
	sort.Strings(devices)
	return devices
}

// waitForDevice polls briefly after modprobe: the device node appears through
// udev, so it is not there the instant the module loads.
func waitForDevice(ctx context.Context, hasDevice func() bool) bool {
	deadline := time.NewTimer(2 * time.Second)
	defer deadline.Stop()
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		if hasDevice() {
			return true
		}
		select {
		case <-ctx.Done():
			return false
		case <-deadline.C:
			return hasDevice()
		case <-ticker.C:
		}
	}
}
