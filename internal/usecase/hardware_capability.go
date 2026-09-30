package usecase

import (
	"os"
	"path/filepath"
	"strings"
)

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