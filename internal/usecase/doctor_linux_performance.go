package usecase

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/eajdias/envctl/internal/domain/entity"
)

// linuxPerfSection renders one read-only snapshot area into its diagnostic.
// The table keeps the eight areas in audit order; each render preserves the
// original empty-vs-present wording exactly (pacnew keeps its own block: it
// is the only area whose present branch carries a fix hint).
type linuxPerfSection struct {
	target string
	render func(entity.PerformanceSnapshot) entity.Diagnostic
}

// linuxPerformanceSections covers every snapshot area except pacnew, whose
// present branch carries a fix hint and stays inline above.
var linuxPerformanceSections = []linuxPerfSection{
	{
		target: "swap",
		render: func(snapshot entity.PerformanceSnapshot) entity.Diagnostic {
			if len(snapshot.Swap) == 0 {
				return entity.Info(
					"Performance",
					"swap",
					"no active swap device detected (informational; disk swap is not created automatically)",
				)
			}
			names := make([]string, 0, len(snapshot.Swap))
			for _, device := range snapshot.Swap {
				names = append(names, fmt.Sprintf("%s (%s, priority %d)", device.Name, formatPerformanceBytes(device.SizeKB*1024), device.Priority))
			}
			return entity.OK(
				"Performance",
				"swap",
				"active devices: "+strings.Join(names, ", "),
			)
		},
	},
	{
		target: "zram",
		render: func(snapshot entity.PerformanceSnapshot) entity.Diagnostic {
			if snapshot.ZRAM.Present {
				return entity.OK(
					"Performance",
					"zram",
					fmt.Sprintf("%s active (%s, %s, priority %d)",
						snapshot.ZRAM.Name, snapshot.ZRAM.Algorithm, formatPerformanceBytes(snapshot.ZRAM.SizeBytes), snapshot.ZRAM.Priority),
				)
			}
			return entity.Info(
				"Performance",
				"zram",
				"zram device not detected (run the OS-specific performance profile if desired)",
			)
		},
	},
	{
		target: "cpu-governor",
		render: func(snapshot entity.PerformanceSnapshot) entity.Diagnostic {
			if len(snapshot.CPUGovernors) == 0 {
				return entity.Info(
					"Performance",
					"cpu-governor",
					"CPU frequency governor unavailable",
				)
			}
			return entity.Info(
				"Performance",
				"cpu-governor",
				"current governor(s): "+strings.Join(snapshot.CPUGovernors, ", "),
			)
		},
	},
	{
		target: "io-scheduler",
		render: func(snapshot entity.PerformanceSnapshot) entity.Diagnostic {
			if len(snapshot.BlockSchedulers) == 0 {
				return entity.Info(
					"Performance",
					"io-scheduler",
					"block scheduler information unavailable",
				)
			}
			details := make([]string, 0, len(snapshot.BlockSchedulers))
			for _, scheduler := range snapshot.BlockSchedulers {
				details = append(details, fmt.Sprintf("%s=%s", scheduler.Device, scheduler.Selected))
			}
			return entity.Info(
				"Performance",
				"io-scheduler",
				"current scheduler(s): "+strings.Join(details, ", "),
			)
		},
	},
	{
		target: "journald",
		render: func(snapshot entity.PerformanceSnapshot) entity.Diagnostic {
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
			return entity.Info(
				"Performance",
				"journald",
				strings.Join(journaldDetails, ", "),
			)
		},
	},
	{
		target: "fstrim.timer",
		render: func(snapshot entity.PerformanceSnapshot) entity.Diagnostic {
			fstrim := snapshot.FSTRIMTimer
			if fstrim.Name == "" {
				return entity.Info(
					"Performance",
					"fstrim.timer",
					"timer information unavailable",
				)
			}
			return entity.Info(
				"Performance",
				"fstrim.timer",
				fmt.Sprintf("enabled=%s active=%s", valueOrUnknown(fstrim.Enabled), valueOrUnknown(fstrim.Active)),
			)
		},
	},
	{
		target: "services",
		render: func(snapshot entity.PerformanceSnapshot) entity.Diagnostic {
			if len(snapshot.Services) == 0 {
				return entity.Info(
					"Performance",
					"services",
					"service state unavailable",
				)
			}
			details := make([]string, 0, len(snapshot.Services))
			for _, service := range snapshot.Services {
				details = append(details, fmt.Sprintf("%s(active=%s,enabled=%s)", service.Name, valueOrUnknown(service.Active), valueOrUnknown(service.Enabled)))
			}
			return entity.Info(
				"Performance",
				"services",
				strings.Join(details, ", "),
			)
		},
	},
}

func (uc *DoctorAuditUseCase) auditLinuxPerformance(snapshot entity.PerformanceSnapshot, addDiag func(entity.Diagnostic)) {
	if runtime.GOOS != "linux" {
		return
	}

	// A pending .pacnew means a packaged config differs from the one on disk.
	// Informational on purpose: it is not a performance defect and not every
	// pending file deserves action, so warning would be noise.
	if pending := pendingPacnewFiles("/etc"); len(pending) > 0 {
		addDiag(entity.Info(
			"Performance",
			"pacnew",
			fmt.Sprintf("%d pending .pacnew file(s): %s", len(pending), strings.Join(pending, ", ")),
			"review with 'pacdiff', then merge or delete each file; leaving them is safe but the packaged change never lands",
		))
	} else {
		addDiag(entity.OK(
			"Performance",
			"pacnew",
			"no pending .pacnew files",
		))
	}

	for _, section := range linuxPerformanceSections {
		addDiag(section.render(snapshot))
	}

	// What the profile wants, against what the host will actually apply at the
	// next boot. The profile's own file being correct proves nothing about the
	// running host, which is how a vendor file with a later filename kept
	// vm.swappiness at 10 across a reboot with a green audit.
	uc.auditSysctlIntent(snapshot, addDiag)

	// A pending reboot is the one performance line allowed to warn: the host
	// genuinely is not in the state the profile would converge it to.
	if state := ProbeRebootPending(nil); state.Pending {
		addDiag(entity.Warn(
			"Performance",
			"reboot",
			state.Detail,
			"sudo reboot, then re-run envctl run performance",
		))
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
