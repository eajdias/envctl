package repository

import (
	"context"

	"github.com/eajdias/envctl/internal/domain/entity"
)

// SysctlManager applies a reviewed sysctl drop-in. Implementations must use
// argv-based commands, preserve an atomic backup, and honor dry-run.
type SysctlManager interface {
	Apply(ctx context.Context, settings []entity.SysctlSetting, dryRun bool) ([]entity.Diagnostic, error)
}

// ZRAMManager ensures the OS zram generator has produced a device. It is
// invoked only by the explicit performance profile.
type ZRAMManager interface {
	Ensure(ctx context.Context, dryRun bool) ([]entity.Diagnostic, error)
}

// HardwareProbe returns the detected, read-only description of the host. A
// missing source is reported as a zero value, never as an error: the caller
// resolves policy against what could be measured.
type HardwareProbe interface {
	Snapshot(ctx context.Context) entity.HardwareState
}

// TimezoneManager reports the host timezone and, only when explicitly asked to,
// sets it. The default policy is verification.
type TimezoneManager interface {
	Current(ctx context.Context) (string, error)
	Apply(ctx context.Context, spec entity.TimezoneSpec, dryRun bool) ([]entity.Diagnostic, error)
}

// JournaldManager installs the journald size drop-in. It restarts the service
// rather than stopping it: man 8 systemd-journald documents that a restart
// preserves the client streams and that stopping is not recommended.
type JournaldManager interface {
	Apply(ctx context.Context, spec entity.JournaldSpec, dryRun bool) ([]entity.Diagnostic, error)
}

// ResourceLimitsManager installs the systemd and PAM limit drop-ins. It may
// re-execute the service manager, which is the only privileged PID 1 operation
// in this package and is opt-out.
type ResourceLimitsManager interface {
	Apply(ctx context.Context, spec entity.LimitsSpec, dryRun bool) ([]entity.Diagnostic, error)
}

// PerformanceInspector returns read-only Linux performance state. Missing
// probes are represented by zero values/empty slices and never mutate the host.
type PerformanceInspector interface {
	Snapshot(ctx context.Context) entity.PerformanceSnapshot
}
