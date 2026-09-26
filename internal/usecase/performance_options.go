package usecase

import (
	"fmt"
	"os"
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
	// Umbrella marks a whole-profile run (`run all`, `run vps`, `run windows`,
	// `run cachyos`) rather than the dedicated `run performance`. An umbrella
	// run reports a pending reboot and continues, because the phases after the
	// performance one are valid regardless and aborting would make a server with
	// pending kernel updates impossible to bootstrap.
	Umbrella bool
}

// ValidateRebootPolicy decides whether a pending reboot blocks the run. The
// state is supplied rather than probed so the rule is a pure function of the
// options.
func ValidateRebootPolicy(opts PerformanceOptions, state RebootPendingState) error {
	if opts.Umbrella || opts.ForceRebootPending || !state.Pending {
		return nil
	}
	return fmt.Errorf("a reboot is pending: %s", state.Detail)
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

// orUnknownString keeps a diagnostic readable when a probe returned nothing.
func orUnknownString(value string) string {
	if strings.TrimSpace(value) == "" {
		return "unknown"
	}
	return value
}
