package performance

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/eajdias/envctl/internal/domain/entity"
	"github.com/eajdias/envctl/internal/domain/repository"
)

type performanceInspector struct {
	root string
	run  commandRunner
}

// NewPerformanceInspector creates a read-only Linux performance inspector.
func NewPerformanceInspector() repository.PerformanceInspector {
	return newPerformanceInspector("/", func(ctx context.Context, name string, args ...string) ([]byte, error) {
		return exec.CommandContext(ctx, name, args...).CombinedOutput()
	})
}

func newPerformanceInspector(root string, run commandRunner) *performanceInspector {
	return &performanceInspector{root: root, run: run}
}

func (i *performanceInspector) Snapshot(ctx context.Context) entity.PerformanceSnapshot {
	journaldConfig := i.output(ctx, "systemd-analyze", "cat-config", "systemd/journald.conf")
	snapshot := entity.PerformanceSnapshot{
		Swap:            readSwapTable(filepath.Join(i.root, "proc/swaps")),
		ZRAM:            i.readZRAM(ctx),
		CPUGovernors:    readCPUGovernors(filepath.Join(i.root, "sys/devices/system/cpu")),
		BlockSchedulers: readBlockSchedulers(filepath.Join(i.root, "sys/block")),
		Journald: entity.JournaldState{
			DiskUsage:    parseJournaldDiskUsage(i.output(ctx, "journalctl", "--disk-usage")),
			Storage:      parseJournaldValue(journaldConfig, "Storage="),
			SystemMaxUse: parseJournaldValue(journaldConfig, "SystemMaxUse="),
		},
		FSTRIMTimer: i.readTimer(ctx, "fstrim.timer"),
		Services:    i.readServices(ctx, []string{"scx_loader", "lactd", "ananicy-cpp", "power-profiles-daemon", "systemd-oomd"}),
	}
	if snapshot.ZRAM.Present && snapshot.ZRAM.Priority == 0 {
		for _, device := range snapshot.Swap {
			if device.Name == snapshot.ZRAM.Name {
				snapshot.ZRAM.Priority = device.Priority
				break
			}
		}
	}
	return snapshot
}

func (i *performanceInspector) output(ctx context.Context, name string, args ...string) string {
	if i.run == nil {
		return ""
	}
	out, err := i.run(ctx, name, args...)
	if err != nil && len(out) == 0 {
		return ""
	}
	return string(out)
}

func (i *performanceInspector) readZRAM(ctx context.Context) entity.ZRAMState {
	root := filepath.Join(i.root, "sys/block/zram0")
	algorithm := readTrimmed(filepath.Join(root, "comp_algorithm"))
	if selected, _ := parseScheduler(algorithm); selected != "" {
		algorithm = selected
	}
	if algorithm == "" {
		return parseZramctl(i.output(ctx, "zramctl"))
	}

	state := entity.ZRAMState{
		Present:    true,
		Name:       "/dev/zram0",
		Algorithm:  algorithm,
		SizeBytes:  readUint(filepath.Join(root, "disksize")),
		DataBytes:  readUint(filepath.Join(root, "mem_used_total")),
		TotalBytes: readUint(filepath.Join(root, "mem_total")),
		Priority:   safeUintToInt(readUint(filepath.Join(root, "priority"))),
	}
	return state
}

func (i *performanceInspector) readTimer(ctx context.Context, name string) entity.TimerState {
	return entity.TimerState{
		Name:    name,
		Enabled: strings.TrimSpace(i.output(ctx, "systemctl", "is-enabled", name)),
		Active:  strings.TrimSpace(i.output(ctx, "systemctl", "is-active", name)),
	}
}

func (i *performanceInspector) readServices(ctx context.Context, names []string) []entity.ServiceState {
	services := make([]entity.ServiceState, 0, len(names))
	for _, name := range names {
		services = append(services, entity.ServiceState{
			Name:    name,
			Enabled: strings.TrimSpace(i.output(ctx, "systemctl", "is-enabled", name)),
			Active:  strings.TrimSpace(i.output(ctx, "systemctl", "is-active", name)),
		})
	}
	return services
}

func readSwapTable(path string) []entity.SwapDevice {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	return parseSwapTable(string(data))
}

func parseSwapTable(data string) []entity.SwapDevice {
	var devices []entity.SwapDevice
	for _, line := range strings.Split(data, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 5 || fields[0] == "Filename" {
			continue
		}
		size, sizeErr := strconv.ParseUint(fields[2], 10, 64)
		if sizeErr != nil {
			continue
		}
		used, usedErr := strconv.ParseUint(fields[3], 10, 64)
		if usedErr != nil {
			continue
		}
		priority, priorityErr := strconv.Atoi(fields[4])
		if priorityErr != nil {
			continue
		}
		devices = append(devices, entity.SwapDevice{
			Name:     fields[0],
			Type:     fields[1],
			SizeKB:   size,
			UsedKB:   used,
			Priority: priority,
		})
	}
	return devices
}

func parseZramctl(data string) entity.ZRAMState {
	for _, line := range strings.Split(data, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 6 || fields[0] == "NAME" || !strings.HasPrefix(fields[0], "/dev/zram") {
			continue
		}
		return entity.ZRAMState{
			Present:    true,
			Name:       fields[0],
			Algorithm:  fields[1],
			SizeBytes:  parseSizeBytes(fields[2]),
			DataBytes:  parseSizeBytes(fields[3]),
			TotalBytes: parseSizeBytes(fields[5]),
		}
	}
	return entity.ZRAMState{}
}

func parseScheduler(value string) (selected, available string) {
	// Re-read the original value so bracketed scheduler names remain exact.
	available = strings.Join(strings.Fields(value), " ")
	for _, field := range strings.Fields(value) {
		if strings.HasPrefix(field, "[") && strings.HasSuffix(field, "]") {
			selected = strings.Trim(field, "[]")
		}
	}
	if selected == "" {
		return "", available
	}
	for _, field := range strings.Fields(value) {
		clean := strings.Trim(field, "[]")
		available = strings.ReplaceAll(available, "["+clean+"]", clean)
	}
	return selected, available
}

var journaldUsagePattern = regexp.MustCompile(`take up\s+([0-9]+(?:[.,][0-9]+)?[KMGTP]?B?)`)

func parseJournaldDiskUsage(data string) string {
	matches := journaldUsagePattern.FindStringSubmatch(data)
	if len(matches) == 2 {
		return strings.Replace(matches[1], ",", ".", 1)
	}
	return ""
}

func parseJournaldValue(data, key string) string {
	for _, line := range strings.Split(data, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, key) {
			return strings.TrimSpace(strings.TrimPrefix(line, key))
		}
	}
	return ""
}

func readCPUGovernors(root string) []string {
	paths, err := filepath.Glob(filepath.Join(root, "cpu*", "cpufreq", "scaling_governor"))
	if err != nil {
		return nil
	}
	seen := map[string]bool{}
	var governors []string
	for _, path := range paths {
		governor := readTrimmed(path)
		if governor == "" || seen[governor] {
			continue
		}
		seen[governor] = true
		governors = append(governors, governor)
	}
	sort.Strings(governors)
	return governors
}

func readBlockSchedulers(root string) []entity.BlockScheduler {
	paths, err := filepath.Glob(filepath.Join(root, "*", "queue", "scheduler"))
	if err != nil {
		return nil
	}
	var schedulers []entity.BlockScheduler
	for _, path := range paths {
		selected, available := parseScheduler(readTrimmed(path))
		if selected == "" {
			continue
		}
		device := filepath.Base(filepath.Dir(filepath.Dir(path)))
		schedulers = append(schedulers, entity.BlockScheduler{Device: device, Selected: selected, Available: available})
	}
	sort.Slice(schedulers, func(i, j int) bool { return schedulers[i].Device < schedulers[j].Device })
	return schedulers
}

func readTrimmed(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func readUint(path string) uint64 {
	value, err := strconv.ParseUint(readTrimmed(path), 10, 64)
	if err != nil {
		return 0
	}
	return value
}

func safeUintToInt(value uint64) int {
	maxInt := uint64(^uint(0) >> 1)
	if value > maxInt {
		return int(maxInt)
	}
	return int(value)
}

func parseSizeBytes(value string) uint64 {
	value = strings.TrimSpace(strings.Replace(value, ",", ".", 1))
	if value == "" {
		return 0
	}
	multiplier := uint64(1)
	last := value[len(value)-1]
	switch last {
	case 'K', 'k':
		multiplier = 1_000
		value = value[:len(value)-1]
	case 'M', 'm':
		multiplier = 1_000_000
		value = value[:len(value)-1]
	case 'G', 'g':
		multiplier = 1_000_000_000
		value = value[:len(value)-1]
	case 'T', 't':
		multiplier = 1_000_000_000_000
		value = value[:len(value)-1]
	}
	number, err := strconv.ParseFloat(value, 64)
	if err != nil || number < 0 {
		return 0
	}
	return uint64(number * float64(multiplier))
}
