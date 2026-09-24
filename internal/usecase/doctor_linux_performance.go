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
