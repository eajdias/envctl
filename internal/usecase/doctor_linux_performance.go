package usecase

import (
	"context"
	"fmt"
	"runtime"
	"strings"

	"github.com/eajdias/envctl/internal/domain/entity"
)

func (uc *DoctorAuditUseCase) auditLinuxPerformance(ctx context.Context, addDiag func(entity.Diagnostic)) {
	if runtime.GOOS != "linux" || uc.performanceInspector == nil {
		return
	}

	snapshot := uc.performanceInspector.Snapshot(ctx)

	// A pending .pacnew means a packaged config differs from the one on disk.
	// Informational on purpose: it is not a performance defect and not every
	// pending file deserves action, so warning would be noise.
	if pending := pendingPacnewFiles("/etc"); len(pending) > 0 {
		addDiag(entity.Diagnostic{
			Category: entity.DiagInfo,
			System:   "Performance",
			Target:   "pacnew",
			Details:  fmt.Sprintf("%d pending .pacnew file(s): %s", len(pending), strings.Join(pending, ", ")),
			FixHint:  "review with 'pacdiff', then merge or delete each file; leaving them is safe but the packaged change never lands",
		})
	} else {
		addDiag(entity.Diagnostic{
			Category: entity.DiagOK,
			System:   "Performance",
			Target:   "pacnew",
			Details:  "no pending .pacnew files",
		})
	}

	if len(snapshot.Swap) == 0 {
		addDiag(entity.Diagnostic{
			Category: entity.DiagInfo,
			System:   "Performance",
			Target:   "swap",
			Details:  "no active swap device detected (informational; disk swap is not created automatically)",
		})
	} else {
		names := make([]string, 0, len(snapshot.Swap))
		for _, device := range snapshot.Swap {
			names = append(names, fmt.Sprintf("%s (%s, priority %d)", device.Name, formatPerformanceBytes(device.SizeKB*1024), device.Priority))
		}
		addDiag(entity.Diagnostic{
			Category: entity.DiagOK,
			System:   "Performance",
			Target:   "swap",
			Details:  "active devices: " + strings.Join(names, ", "),
		})
	}

	if snapshot.ZRAM.Present {
		addDiag(entity.Diagnostic{
			Category: entity.DiagOK,
			System:   "Performance",
			Target:   "zram",
			Details: fmt.Sprintf("%s active (%s, %s, priority %d)",
				snapshot.ZRAM.Name, snapshot.ZRAM.Algorithm, formatPerformanceBytes(snapshot.ZRAM.SizeBytes), snapshot.ZRAM.Priority),
		})
	} else {
		addDiag(entity.Diagnostic{
			Category: entity.DiagInfo,
			System:   "Performance",
			Target:   "zram",
			Details:  "zram device not detected (run the OS-specific performance profile if desired)",
		})
	}

	if len(snapshot.CPUGovernors) == 0 {
		addDiag(entity.Diagnostic{
			Category: entity.DiagInfo,
			System:   "Performance",
			Target:   "cpu-governor",
			Details:  "CPU frequency governor unavailable",
		})
	} else {
		addDiag(entity.Diagnostic{
			Category: entity.DiagInfo,
			System:   "Performance",
			Target:   "cpu-governor",
			Details:  "current governor(s): " + strings.Join(snapshot.CPUGovernors, ", "),
		})
	}

	if len(snapshot.BlockSchedulers) == 0 {
		addDiag(entity.Diagnostic{
			Category: entity.DiagInfo,
			System:   "Performance",
			Target:   "io-scheduler",
			Details:  "block scheduler information unavailable",
		})
	} else {
		details := make([]string, 0, len(snapshot.BlockSchedulers))
		for _, scheduler := range snapshot.BlockSchedulers {
			details = append(details, fmt.Sprintf("%s=%s", scheduler.Device, scheduler.Selected))
		}
		addDiag(entity.Diagnostic{
			Category: entity.DiagInfo,
			System:   "Performance",
			Target:   "io-scheduler",
			Details:  "current scheduler(s): " + strings.Join(details, ", "),
		})
	}

	journaldDetails := make([]string, 0, 3)
	if snapshot.Journald.DiskUsage != "" {
		journaldDetails = append(journaldDetails, "disk="+snapshot.Journald.DiskUsage)
	}
	if snapshot.Journald.Storage != "" {
		journaldDetails = append(journaldDetails, "storage="+snapshot.Journald.Storage)
	}
	if snapshot.Journald.SystemMaxUse != "" {
		journaldDetails = append(journaldDetails, "max="+snapshot.Journald.SystemMaxUse)
	}
	if len(journaldDetails) == 0 {
		journaldDetails = append(journaldDetails, "configuration unavailable")
	}
	addDiag(entity.Diagnostic{
		Category: entity.DiagInfo,
		System:   "Performance",
		Target:   "journald",
		Details:  strings.Join(journaldDetails, ", "),
	})

	fstrim := snapshot.FSTRIMTimer
	if fstrim.Name == "" {
		addDiag(entity.Diagnostic{
			Category: entity.DiagInfo,
			System:   "Performance",
			Target:   "fstrim.timer",
			Details:  "timer information unavailable",
		})
	} else {
		addDiag(entity.Diagnostic{
			Category: entity.DiagInfo,
			System:   "Performance",
			Target:   "fstrim.timer",
			Details:  fmt.Sprintf("enabled=%s active=%s", valueOrUnknown(fstrim.Enabled), valueOrUnknown(fstrim.Active)),
		})
	}

	if len(snapshot.Services) == 0 {
		addDiag(entity.Diagnostic{
			Category: entity.DiagInfo,
			System:   "Performance",
			Target:   "services",
			Details:  "service state unavailable",
		})
	} else {
		details := make([]string, 0, len(snapshot.Services))
		for _, service := range snapshot.Services {
			details = append(details, fmt.Sprintf("%s(active=%s,enabled=%s)", service.Name, valueOrUnknown(service.Active), valueOrUnknown(service.Enabled)))
		}
		addDiag(entity.Diagnostic{
			Category: entity.DiagInfo,
			System:   "Performance",
			Target:   "services",
			Details:  strings.Join(details, ", "),
		})
	}

	// What the profile wants, against what the host will actually apply at the
	// next boot. The profile's own file being correct proves nothing about the
	// running host, which is how a vendor file with a later filename kept
	// vm.swappiness at 10 across a reboot with a green audit.
	uc.auditSysctlIntent(snapshot, addDiag)

	// A pending reboot is the one performance line allowed to warn: the host
	// genuinely is not in the state the profile would converge it to.
	if state := ProbeRebootPending(nil); state.Pending {
		addDiag(entity.Diagnostic{
			Category: entity.DiagWarning,
			System:   "Performance",
			Target:   "reboot",
			Details:  state.Detail,
			FixHint:  "sudo reboot, then re-run envctl run performance",
		})
	}
}

func formatPerformanceBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	divisor := uint64(unit)
	exponent := 0
	for value := bytes / unit; value >= unit && exponent < 4; value /= unit {
		divisor *= unit
		exponent++
	}
	return fmt.Sprintf("%.1f %ciB", float64(bytes)/float64(divisor), "KMGTP"[exponent])
}

func valueOrUnknown(value string) string {
	if strings.TrimSpace(value) == "" {
		return "unknown"
	}
	return value
}
