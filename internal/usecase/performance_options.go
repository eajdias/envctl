package usecase

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/eajdias/envctl/internal/domain/entity"
	"github.com/eajdias/envctl/internal/infra/performance"
)

// rebootRequiredPath is the marker Ubuntu and Debian write when a package change
// needs a reboot before the running libraries match the on-disk ones.
const rebootRequiredPath = "/var/run/reboot-required"

// rebootPendingProbe is injectable so the precondition is testable without a
// real /var/run.
type rebootPendingProbe func(path string) bool

func defaultRebootPendingProbe() rebootPendingProbe {
	return func(path string) bool {
		_, err := os.Stat(path)
		return err == nil
	}
}

// RebootPendingState reports whether the host is waiting for a reboot.
type RebootPendingState struct {
	Pending bool
	Path    string
	Detail  string
}

// ProbeRebootPending reports the pending-reboot precondition.
//
// It is part of the profile contract because applying performance changes on top
// of a pending library update tunes a runtime the host is about to replace: the
// sysctls survive, but the tuning may not correspond to the code that will
// actually be running.
func ProbeRebootPending(probe rebootPendingProbe) RebootPendingState {
	if probe == nil {
		probe = defaultRebootPendingProbe()
	}
	state := RebootPendingState{Path: rebootRequiredPath}
	if !probe(rebootRequiredPath) {
		return state
	}
	state.Pending = true
	state.Detail = rebootRequiredPath + " exists: the host is waiting for a reboot before these changes describe a stable runtime"
	if packages, err := os.ReadFile(rebootRequiredPath + "-pkgs"); err == nil && len(packages) > 0 {
		state.Detail += "; pending packages: " + strings.TrimSpace(string(packages))
	}
	return state
}

// PerformanceOptions carries the command-line decisions that shape a run.
type PerformanceOptions struct {
	DryRun             bool
	NoDaemonReexec     bool
	Timezone           string
	AllowDebloat       bool
	DebloatOnly        bool
	ForceRebootPending bool
}

// applyTo folds the options into the specs the managers receive, so the mapping
// from flag to effect is testable without a host.
func (o PerformanceOptions) applyTo(limits *entity.LimitsSpec, timezone *entity.TimezoneSpec) {
	if limits != nil && o.NoDaemonReexec {
		limits.Reexec = false
	}
	if timezone != nil && o.Timezone != "" {
		timezone.Mode = "enforce"
		timezone.Expected = o.Timezone
	}
}

// Validate rejects a flag combination that cannot be honoured, so the command
// fails before anything is written.
func (o PerformanceOptions) Validate() error {
	if o.DebloatOnly && o.AllowDebloat {
		return fmt.Errorf("--debloat-only already implies --allow-debloat; pass only one")
	}
	return nil
}

// assessJournald projects the read-only journald snapshot onto the declared
// policy. It reports a host nobody has provisioned as INFO, never WARNING, so
// a fresh machine is not a failure.
func assessJournald(state entity.JournaldState) (capped bool, category entity.DiagnosticStatus, detail string) {
	return performance.AssessJournaldPolicy(state)
}

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
			Details:  "no active swap device detected (informational; a host is not required to have swap)",
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
			Details:  "active swap: " + strings.Join(names, ", "),
		})
	}

	if snapshot.ZRAM.Present {
		addDiag(entity.Diagnostic{
			Category: entity.DiagOK,
			System:   "Performance",
			Target:   "zram",
			Details: fmt.Sprintf("active zram %s (%s, algorithm %s, priority %d)",
				snapshot.ZRAM.Name, formatPerformanceBytes(snapshot.ZRAM.SizeBytes),
				orUnknownString(snapshot.ZRAM.Algorithm), snapshot.ZRAM.Priority),
		})
	} else {
		addDiag(entity.Diagnostic{
			Category: entity.DiagInfo,
			System:   "Performance",
			Target:   "zram",
			Details:  "no compressed RAM swap is active (informational; whether it belongs on this host is a memory-tier decision)",
		})
	}

	if len(snapshot.CPUGovernors) > 0 {
		addDiag(entity.Diagnostic{
			Category: entity.DiagOK,
			System:   "Performance",
			Target:   "cpu-governor",
			Details:  "governors in use: " + strings.Join(snapshot.CPUGovernors, ", ") + " (read-only; governors are never changed by this profile)",
		})
	}

	if snapshot.FSTRIMTimer.Name != "" {
		category := entity.DiagInfo
		if snapshot.FSTRIMTimer.Enabled == "enabled" && snapshot.FSTRIMTimer.Active == "active" {
			category = entity.DiagOK
		}
		addDiag(entity.Diagnostic{
			Category: category,
			System:   "Performance",
			Target:   snapshot.FSTRIMTimer.Name,
			Details:  fmt.Sprintf("enabled=%s active=%s", orUnknownString(snapshot.FSTRIMTimer.Enabled), orUnknownString(snapshot.FSTRIMTimer.Active)),
		})
	}

	for _, service := range snapshot.Services {
		addDiag(entity.Diagnostic{
			Category: entity.DiagInfo,
			System:   "Performance",
			Target:   service.Name,
			Details:  fmt.Sprintf("enabled=%s active=%s (read-only; this profile never changes unrelated services)", orUnknownString(service.Enabled), orUnknownString(service.Active)),
		})
	}

	if _, category, detail := assessJournald(snapshot.Journald); detail != "" {
		addDiag(entity.Diagnostic{
			Category: category,
			System:   "Performance",
			Target:   "journald",
			Details:  detail,
		})
	}

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

// orUnknownString keeps a diagnostic readable when a probe returned nothing.
func orUnknownString(value string) string {
	if strings.TrimSpace(value) == "" {
		return "unknown"
	}
	return value
}
