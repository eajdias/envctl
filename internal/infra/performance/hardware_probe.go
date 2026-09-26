package performance

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/eajdias/envctl/internal/domain/entity"
	"github.com/eajdias/envctl/internal/domain/repository"
)

// fsStat is the injectable slice of statfs(2), so the swap policy can be tested
// against every filesystem without mounting one.
type fsStat struct {
	Type      string
	FreeBytes uint64
}

type statfsFunc func(path string) (fsStat, error)

type hardwareProbe struct {
	root   string
	statfs statfsFunc
	run    commandRunner
}

// NewHardwareProbe creates the production read-only hardware probe.
func NewHardwareProbe() repository.HardwareProbe {
	return newHardwareProbe("/", statfsRoot, func(ctx context.Context, name string, args ...string) ([]byte, error) {
		return nil, nil
	})
}

func newHardwareProbe(root string, statfs statfsFunc, run commandRunner) *hardwareProbe {
	return &hardwareProbe{root: root, statfs: statfs, run: run}
}

// Snapshot reads the host's hardware and swap topology. Every source is
// optional: a missing procfs entry, an unreadable statfs, or an unknown
// filesystem magic degrades to a zero value so the caller can decide policy
// rather than crash the run.
func (h *hardwareProbe) Snapshot(ctx context.Context) entity.HardwareState {
	memTotalKB := parseMemTotalKB(readTrimmed(filepath.Join(h.root, "proc", "meminfo")))

	rootFS, free := "", uint64(0)
	if h.statfs != nil {
		if stat, err := h.statfs(h.root); err == nil {
			rootFS, free = stat.Type, stat.FreeBytes
		}
	}
	if rootFS == "" && h.run != nil {
		// findmnt is the fallback for a filesystem whose magic is not in the
		// table: an unmapped type must still be named so the swap policy can
		// refuse it explicitly rather than assume it is safe.
		rootFS = strings.TrimSpace(h.output(ctx, "findmnt", "-no", "FSTYPE", h.root))
	}

	return entity.NewHardwareState(
		memTotalKB,
		runtimeCPUCount(),
		rootFS,
		free,
		readSwapTable(filepath.Join(h.root, "proc", "swaps")),
	)
}

func (h *hardwareProbe) output(ctx context.Context, name string, args ...string) string {
	if h.run == nil {
		return ""
	}
	out, err := h.run(ctx, name, args...)
	if err != nil && len(out) == 0 {
		return ""
	}
	return string(out)
}

// parseMemTotalKB reads the kernel's MemTotal line. The value is in kibibytes
// and is always short of the nominal instance size, which is why the tier
// policy converts it explicitly instead of comparing against a marketing number.
func parseMemTotalKB(meminfo string) uint64 {
	for _, line := range strings.Split(meminfo, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 || fields[0] != "MemTotal:" {
			continue
		}
		value, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			return 0
		}
		return value
	}
	return 0
}

// filesystemMagic maps the statfs f_type values the swap policy reasons about.
// Values come from linux/magic.h.
var filesystemMagic = map[int64]string{
	0xEF53:     "ext4",
	0x58465342: "xfs",
	0x9123683E: "btrfs",
	0x01021994: "tmpfs",
	0x794c7630: "overlay",
	0x2FC12FC1: "zfs",
	0x6969:     "nfs",
	0x01021997: "v9fs",
	0x5346544E: "ntfs",
	0xF2F52010: "f2fs",
	0x4d44:     "vfat",
}

func filesystemTypeName(magic int64) string {
	if name, ok := filesystemMagic[magic]; ok {
		return name
	}
	// Unknown filesystems get a stable, obviously-unknown name. The swap
	// policy refuses anything outside its allowlist, so a wrong guess here
	// would create a swapfile the policy meant to prevent.
	return "unknown-" + strconv.FormatInt(magic, 16)
}

// positiveInt64 narrows a signed kernel-reported value to uint64 after checking
// the sign, so a negative value becomes zero rather than wrapping.
func positiveInt64(value int64) uint64 {
	if value <= 0 {
		return 0
	}
	return uint64(value)
}

// statfsRoot reports the filesystem type and available bytes of a path.
func statfsRoot(path string) (fsStat, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return fsStat{}, err
	}
	// Bsize is signed and Bavail is unsigned, with the exact widths varying by
	// architecture. Both are narrowed through positiveInt64 so a negative or
	// absurd kernel value cannot wrap into a huge unsigned one and defeat the
	// swapfile size clamp.
	return fsStat{
		Type:      filesystemTypeName(stat.Type),
		FreeBytes: stat.Bavail * positiveInt64(stat.Bsize),
	}, nil
}

// runtimeCPUCount reports the CPUs this process may actually run on, which is
// the cgroup/affinity-aware count rather than the machine's core count.
func runtimeCPUCount() int {
	if n := len(cpuAffinity()); n > 0 {
		return n
	}
	return 1
}

func cpuAffinity() []int {
	data, err := os.ReadFile("/proc/self/status")
	if err != nil {
		return nil
	}
	for _, line := range strings.Split(string(data), "\n") {
		if !strings.HasPrefix(line, "Cpus_allowed_list:") {
			continue
		}
		return expandCPUSet(strings.TrimSpace(strings.TrimPrefix(line, "Cpus_allowed_list:")))
	}
	return nil
}

// expandCPUSet expands a "0-3,8" style list into its members. runtime.NumCPU
// would be simpler, but it is not injectable and this keeps the probe testable.
func expandCPUSet(list string) []int {
	if list == "" {
		return nil
	}
	var cpus []int
	for _, part := range strings.Split(list, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		low, high, isRange := strings.Cut(part, "-")
		start, err := strconv.Atoi(low)
		if err != nil {
			continue
		}
		if !isRange {
			cpus = append(cpus, start)
			continue
		}
		end, err := strconv.Atoi(high)
		if err != nil || end < start {
			continue
		}
		for cpu := start; cpu <= end; cpu++ {
			cpus = append(cpus, cpu)
		}
	}
	return cpus
}
