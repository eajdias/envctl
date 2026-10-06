package usecase

import (
	"fmt"
	"os"
	"strings"

	"github.com/eajdias/envctl/internal/domain/entity"
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

// assessSysctlIntent projects what a profile wants onto what the host will
// actually apply, and reports only the keys that need a decision.
//
// It compares two things, because either alone hides a defect. The live value
// says whether the kernel is right now; the resolved drop-in says whether it
// will still be right after the next reboot. A key can satisfy the first and
// fail the second — that is exactly what a host-owned file with a later
// filename does, and it is why the profile's own file being correct proves
// nothing about the running host.
//
// It is pure so the rule is testable without a host, and it deliberately emits
// nothing for a key a foreign file already sets to the declared value: there is
// nothing for the owner to decide.
func assessSysctlIntent(intent []entity.SysctlSetting, resolved []entity.SysctlAssignment) []entity.Diagnostic {
	if len(intent) == 0 {
		return nil
	}
	byKey := make(map[string]entity.SysctlAssignment, len(resolved))
	for _, assignment := range resolved {
		byKey[assignment.Key] = assignment
	}

	var diags []entity.Diagnostic
	for _, setting := range intent {
		assignment, found := byKey[setting.Key]
		if !found {
			// A key with no drop-in has no live reading either: the resolver
			// only reports keys it found in a file. For a set-policy key the
			// missing declaration is itself the defect, since nothing would
			// restore the declared value after a reboot. For a min or max key
			// there is nothing to claim: the policy asks the host's value to
			// stand, and a warning without a reading would be a guess.
			if setting.Policy != entity.SysctlPolicySet {
				continue
			}
			diags = append(diags, entity.Warn(
				"Performance",
				setting.Key,
				fmt.Sprintf(
					"declared %s, but no sysctl drop-in declares it: the running value will not survive a reboot",
					setting.Value),
				fmt.Sprintf("run 'envctl run performance' to write the %s drop-in", setting.Key),
			))
			continue
		}

		if !assignment.Managed && !entity.SysctlSettingSatisfied(setting, assignment.Boot) &&
			entity.SysctlSettingSatisfied(setting, assignment.Live) {
			// The kernel holds what the profile wants, but the file that will
			// decide the next boot belongs to somebody else and says otherwise.
			// Reporting OK here would be a lie that a reboot collects.
			diags = append(diags, entity.Warn(
				"Performance",
				setting.Key,
				fmt.Sprintf(
					"this profile declares %s and the kernel currently holds it, but the next boot applies %s from %s, which sorts after this profile's drop-in",
					setting.Value, assignment.Boot, assignment.File),
				fmt.Sprintf(
					"remove or rename %s to let this profile own %s, or keep it to hold the host's %s",
					assignment.File, setting.Key, assignment.Boot),
			))
			continue
		}

		if !assignment.Managed {
			// Somebody else decides the key and the decision disagrees with the
			// profile's policy, so the profile yields and says so.
			if entity.SysctlSettingSatisfied(setting, assignment.Boot) {
				continue
			}
			diags = append(diags, entity.Warn(
				"Performance",
				setting.Key,
				fmt.Sprintf(
					"left at the host's %s, declared %s: %s decides this key at boot because it sorts after this profile's drop-in",
					assignment.Boot, setting.Value, assignment.File),
				fmt.Sprintf(
					"remove or rename %s to let this profile own %s, or keep it to hold the host's %s",
					assignment.File, setting.Key, assignment.Boot),
			))
			continue
		}

		// The profile owns the key: it only has to be in effect.
		if assignment.Live == "" {
			diags = append(diags, entity.OK(
				"Performance",
				setting.Key,
				fmt.Sprintf("%s (declared %s; the running value is unreadable)",
					setting.Value, setting.Value),
			))
			continue
		}
		if entity.SysctlSettingSatisfied(setting, assignment.Live) {
			diags = append(diags, entity.OK(
				"Performance",
				setting.Key,
				fmt.Sprintf("%s as declared", assignment.Live),
			))
			continue
		}
		diags = append(diags, entity.Warn(
			"Performance",
			setting.Key,
			fmt.Sprintf(
				"the kernel holds %s but the drop-in declares %s: something changed it after the run, and the next boot restores the declared value",
				assignment.Live, assignment.Boot),
			"run 'envctl run performance' to re-apply the declared value",
		))
	}
	return diags
}

// auditSysctlIntent resolves the profile this host would be provisioned with and
// audits the sysctl keys that profile wants against the resolved drop-in state.
//
// It reuses performanceSysctlIntent, the same function the provisioning pipeline
// calls, so the audit can never describe a set of keys a run would not apply. The
// difference is the observer: a run probes the hardware, the audit reads the
// snapshot the inspector already took, and both land on the same HardwareState.
func (uc *DoctorAuditUseCase) auditSysctlIntent(snapshot entity.PerformanceSnapshot, addDiag func(entity.Diagnostic)) {
	platform := entity.DetectedPlatform()
	if uc.platform != nil {
		platform = uc.platform()
	}
	profile := entity.ResolvePerformanceProfile(platform)
	if !entity.PerformanceProfileMatchesOS(profile, platform) {
		return
	}
	if uc.manifestRepo == nil {
		return
	}
	spec, err := uc.manifestRepo.LoadPerformanceSpec(profile)
	if err != nil || spec.Profile != profile {
		return
	}

	hardware := snapshot.HardwareState()
	tier := entity.PerformanceTier{}
	if len(spec.Tiers) > 0 {
		selected, tierErr := entity.SelectPerformanceTier(hardware, spec.Tiers)
		if tierErr != nil {
			// Reported rather than skipped. An audit that silently gives up is
			// indistinguishable from an audit that found nothing, which is the
			// failure mode this check exists to remove.
			addDiag(entity.Warn(
				"Performance",
				"sysctl",
				fmt.Sprintf(
					"the memory tier could not be resolved, so the sysctl intent was not audited: %v", tierErr),
				"check the profile's tiers in manifests/performance_ubuntu.yaml",
			))
			return
		}
		tier = selected
	}

	for _, diagnostic := range assessSysctlIntent(
		performanceSysctlIntent(spec, tier, hardware, snapshot.MemoryKB > 0),
		snapshot.Sysctls,
	) {
		addDiag(diagnostic)
	}
}
