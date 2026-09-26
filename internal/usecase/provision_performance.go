package usecase

import (
	"context"
	"fmt"
	"strconv"

	"github.com/eajdias/envctl/internal/domain/entity"
	"github.com/eajdias/envctl/internal/domain/repository"
)

// ProvisionPerformanceUseCase applies one exact OS performance profile. It
// deliberately depends on a supplied PlatformInfo provider so release gates
// are deterministic in tests and never inferred from package-manager presence.
type ProvisionPerformanceUseCase struct {
	manifestRepo repository.ManifestRepository
	packages     *ProvisionPackagesUseCase
	sysctl       repository.SysctlManager
	zram         repository.ZRAMManager
	timezone     repository.TimezoneManager
	journald     repository.JournaldManager
	limits       repository.ResourceLimitsManager
	probe        repository.HardwareProbe
	logger       repository.Logger
	platform     func() entity.PlatformInfo
}

func NewProvisionPerformanceUseCase(
	manifestRepo repository.ManifestRepository,
	packages *ProvisionPackagesUseCase,
	sysctl repository.SysctlManager,
	zram repository.ZRAMManager,
	logger repository.Logger,
	platform func() entity.PlatformInfo,
	timezone repository.TimezoneManager,
	journald repository.JournaldManager,
	limits repository.ResourceLimitsManager,
	probe repository.HardwareProbe,
) *ProvisionPerformanceUseCase {
	if platform == nil {
		platform = entity.DetectedPlatform
	}
	return &ProvisionPerformanceUseCase{
		manifestRepo: manifestRepo,
		packages:     packages,
		sysctl:       sysctl,
		zram:         zram,
		timezone:     timezone,
		journald:     journald,
		limits:       limits,
		probe:        probe,
		logger:       logger,
		platform:     platform,
	}
}

// ExecutePerformance provisions the exact requested profile. Package and
// sysctl changes are both suppressed by dryRun; the profile is rejected before
// any manifest or package operation when the host is not supported.
func (uc *ProvisionPerformanceUseCase) ExecutePerformance(
	ctx context.Context,
	profile entity.PerformanceProfile,
	dryRun bool,
	onProgress PackageProgressHandler,
) ([]entity.Package, []entity.Diagnostic, error) {
	platform := uc.platform()
	if !entity.PerformanceProfileMatchesOS(profile, platform) {
		return nil, nil, fmt.Errorf(
			"performance profile %q targets a different OS than this host (%s %s, family=%s)",
			profile, platform.ID, platform.VersionID, platform.Family,
		)
	}

	spec, err := uc.manifestRepo.LoadPerformanceSpec(profile)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load performance profile %q: %w", profile, err)
	}
	if spec.Profile != profile {
		return nil, nil, fmt.Errorf("performance manifest returned profile %q, expected %q", spec.Profile, profile)
	}
	// The release floor is manifest data, so it is evaluated after the spec is
	// loaded and never hard-coded here.
	if !entity.MatchesDistroMinimum(platform.VersionID, spec.MinDistroVersion) {
		return nil, nil, fmt.Errorf(
			"performance profile %q requires %s >= %s; this host is %s",
			profile, platform.ID, spec.MinDistroVersion, platform.VersionID,
		)
	}
	if err := validatePerformanceSpec(spec, profile, platform); err != nil {
		return nil, nil, err
	}
	if uc.packages == nil {
		return nil, nil, fmt.Errorf("performance package provisioner is not configured")
	}

	packages, err := uc.packages.ExecuteList(ctx, spec.Packages, "", dryRun, onProgress)
	if err != nil {
		return packages, nil, err
	}

	diagnostics := packageDiagnostics(packages, dryRun)
	if !dryRun {
		for _, pkg := range packages {
			if pkg.Status == entity.StatusFailed || pkg.Status == entity.StatusSkipped {
				return packages, diagnostics, fmt.Errorf("performance package %q was not provisioned: %s", pkg.ID, pkg.Error)
			}
		}
	}
	// The host is measured, never assumed: both the memory band and the swap
	// topology come from the probe, and both feed the settings applied below.
	hardware := entity.HardwareState{}
	if uc.probe != nil {
		hardware = uc.probe.Snapshot(ctx)
	}

	// Tier resolution is optional: a profile that declares no memory bands (the
	// CachyOS profile) has no band policy to apply, and forcing one would
	// invent a memory model the profile does not have.
	var tier entity.PerformanceTier
	if len(spec.Tiers) > 0 {
		selected, tierErr := entity.SelectPerformanceTier(hardware, spec.Tiers)
		if tierErr != nil {
			return packages, append(diagnostics, entity.Diagnostic{
				Category: entity.DiagError,
				System:   "Performance",
				Target:   "tier",
				Details:  tierErr.Error(),
			}), tierErr
		}
		tier = selected
		tierCategory := entity.DiagOK
		if dryRun {
			tierCategory = entity.DiagInfo
		}
		diagnostics = append(diagnostics, entity.Diagnostic{
			Category: tierCategory,
			System:   "Performance",
			Target:   "tier",
			Details: fmt.Sprintf("tier %q %s for %d MiB of detected memory (band ceiling %d MiB)",
				tier.ID, map[bool]string{true: "would be selected", false: "selected"}[dryRun],
				hardware.MemTotalMiB(), tier.MatchMemTotalMax),
		})
	}

	if specContainsPackage(spec.Packages, "zram-generator", "systemd-zram-generator") {
		policy := entity.ZRAMPolicyTier
		switch {
		case spec.ZRAM != nil:
			policy = spec.ZRAM.Policy
		case len(spec.Tiers) == 0:
			// No bands means no band policy, so there is nothing to gate on and
			// the profile's zram package is provisioned unconditionally.
			policy = entity.ZRAMPolicyAlways
		}
		run, reason := entity.ResolvedZRAMPolicy(policy, tier, hardware)
		switch {
		case run && uc.zram == nil:
			return packages, diagnostics, fmt.Errorf("performance profile %q requires a zram manager", profile)
		case run:
			zramDiagnostics, zramErr := uc.zram.Ensure(ctx, dryRun)
			diagnostics = append(diagnostics, zramDiagnostics...)
			if zramErr != nil {
				return packages, diagnostics, zramErr
			}
			// Re-read the host: bringing zram up changes the swap topology, and
			// the derived swappiness depends on the topology that exists AFTER
			// the device is live, not on the state before it.
			if uc.probe != nil {
				hardware = uc.probe.Snapshot(ctx)
			}
		default:
			diagnostics = append(diagnostics, entity.Diagnostic{
				Category: entity.DiagInfo,
				System:   "Performance",
				Target:   "zram",
				Details:  reason,
			})
		}
	}

	// vm.swappiness is derived last and therefore always wins. It is a function
	// of the measured swap topology, so a value pinned in the manifest would be
	// a claim about a host nobody measured. This is also what removes the old
	// contradiction of shipping zram together with a pinned value of 10.
	settings := mergeTierSysctls(spec.Sysctls, tier.Sysctls)
	// Only a profile that declares memory bands opts into topology-derived
	// tuning. The CachyOS profile declares none and must never receive a
	// sysctl apply, so deriving a value for it would be an unrequested change.
	if uc.probe != nil && len(spec.Tiers) > 0 {
		value, rationale := entity.DeriveSwappiness(hardware)
		settings = upsertSysctl(settings, entity.SysctlSetting{
			Key:       "vm.swappiness",
			Value:     strconv.Itoa(value),
			Policy:    entity.SysctlPolicySet,
			Rationale: rationale,
		})
	}
	if len(settings) > 0 {
		if uc.sysctl == nil {
			return packages, diagnostics, fmt.Errorf("performance profile %q requires a sysctl manager", profile)
		}
		sysctlDiagnostics, sysctlErr := uc.sysctl.Apply(ctx, settings, dryRun)
		diagnostics = append(diagnostics, sysctlDiagnostics...)
		if sysctlErr != nil {
			return packages, diagnostics, sysctlErr
		}
	}
	if spec.Limits != nil {
		if uc.limits == nil {
			return packages, diagnostics, fmt.Errorf("performance profile %q declares a limits policy but no limits manager is configured", profile)
		}
		limitDiagnostics, limitErr := uc.limits.Apply(ctx, *spec.Limits, dryRun)
		diagnostics = append(diagnostics, limitDiagnostics...)
		if limitErr != nil {
			return packages, diagnostics, limitErr
		}
	}
	if spec.Journald != nil {
		if uc.journald == nil {
			return packages, diagnostics, fmt.Errorf("performance profile %q declares a journald policy but no journald manager is configured", profile)
		}
		journaldDiagnostics, journaldErr := uc.journald.Apply(ctx, *spec.Journald, dryRun)
		diagnostics = append(diagnostics, journaldDiagnostics...)
		if journaldErr != nil {
			return packages, diagnostics, journaldErr
		}
	}
	if spec.Timezone != nil {
		if uc.timezone == nil {
			return packages, diagnostics, fmt.Errorf("performance profile %q declares a timezone policy but no timezone manager is configured", profile)
		}
		timezoneDiagnostics, timezoneErr := uc.timezone.Apply(ctx, *spec.Timezone, dryRun)
		diagnostics = append(diagnostics, timezoneDiagnostics...)
		if timezoneErr != nil {
			return packages, diagnostics, timezoneErr
		}
	}
	return packages, diagnostics, nil
}

// validatePerformanceSpec checks profile-internal consistency and package
// scoping. The platform is supplied by the caller so the check runs against the
// real host instead of a synthesized one: a synthetic VERSION_ID would hide a
// package whose min_distro_version no longer matches the fleet.
// mergeTierSysctls layers a tier's sysctls over the profile's unconditional
// ones, so a band can override a base value without the base being repeated in
// every tier.
func mergeTierSysctls(base, tier []entity.SysctlSetting) []entity.SysctlSetting {
	merged := make(map[string]entity.SysctlSetting, len(base)+len(tier))
	order := make([]string, 0, len(base)+len(tier))
	for _, setting := range append(append([]entity.SysctlSetting(nil), base...), tier...) {
		if _, seen := merged[setting.Key]; !seen {
			order = append(order, setting.Key)
		}
		merged[setting.Key] = setting
	}
	out := make([]entity.SysctlSetting, 0, len(order))
	for _, key := range order {
		out = append(out, merged[key])
	}
	return out
}

// upsertSysctl replaces a key in place or appends it.
func upsertSysctl(settings []entity.SysctlSetting, setting entity.SysctlSetting) []entity.SysctlSetting {
	for i, existing := range settings {
		if existing.Key == setting.Key {
			settings[i] = setting
			return settings
		}
	}
	return append(settings, setting)
}

func validatePerformanceSpec(spec entity.PerformanceSpec, profile entity.PerformanceProfile, platform entity.PlatformInfo) error {
	if profile == entity.PerformanceProfileCachyOS && len(spec.Sysctls) > 0 {
		return fmt.Errorf("CachyOS performance profile cannot contain sysctl settings")
	}

	for _, pkg := range spec.Packages {
		if !entity.PackageMatchesPlatform(pkg, platform) {
			return fmt.Errorf("performance package %q is not applicable to %s %s in profile %q",
				pkg.ID, platform.ID, platform.VersionID, profile)
		}
	}
	return nil
}

func specContainsPackage(packages []entity.Package, ids ...string) bool {
	for _, pkg := range packages {
		for _, id := range ids {
			if pkg.ID == id {
				return true
			}
		}
	}
	return false
}

func packageDiagnostics(packages []entity.Package, dryRun bool) []entity.Diagnostic {
	diagnostics := make([]entity.Diagnostic, 0, len(packages))
	for _, pkg := range packages {
		category := entity.DiagOK
		details := "package ready"
		if dryRun && pkg.Status == entity.StatusMissing {
			category = entity.DiagInfo
			details = "would install (dry-run)"
		} else if pkg.Status == entity.StatusFailed {
			category = entity.DiagError
			details = pkg.Error
		} else if pkg.Status == entity.StatusSkipped {
			category = entity.DiagInfo
			details = pkg.Error
		} else if pkg.Version != "" {
			details = fmt.Sprintf("installed (%s)", pkg.Version)
		}
		diagnostics = append(diagnostics, entity.Diagnostic{
			Category: category,
			System:   "Performance",
			Target:   pkg.ID,
			Details:  details,
		})
	}
	return diagnostics
}
