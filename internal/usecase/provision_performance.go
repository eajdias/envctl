package usecase

import (
	"context"
	"fmt"

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
) *ProvisionPerformanceUseCase {
	if platform == nil {
		platform = entity.DetectedPlatform
	}
	return &ProvisionPerformanceUseCase{
		manifestRepo: manifestRepo,
		packages:     packages,
		sysctl:       sysctl,
		zram:         zram,
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
	if len(spec.Sysctls) > 0 {
		if uc.sysctl == nil {
			return packages, diagnostics, fmt.Errorf("performance profile %q requires a sysctl manager", profile)
		}
		sysctlDiagnostics, sysctlErr := uc.sysctl.Apply(ctx, spec.Sysctls, dryRun)
		diagnostics = append(diagnostics, sysctlDiagnostics...)
		if sysctlErr != nil {
			return packages, diagnostics, sysctlErr
		}
	}
	if specContainsPackage(spec.Packages, "zram-generator", "systemd-zram-generator") {
		if uc.zram == nil {
			return packages, diagnostics, fmt.Errorf("performance profile %q requires a zram manager", profile)
		}
		zramDiagnostics, zramErr := uc.zram.Ensure(ctx, dryRun)
		diagnostics = append(diagnostics, zramDiagnostics...)
		if zramErr != nil {
			return packages, diagnostics, zramErr
		}
	}

	return packages, diagnostics, nil
}

// validatePerformanceSpec checks profile-internal consistency and package
// scoping. The platform is supplied by the caller so the check runs against the
// real host instead of a synthesized one: a synthetic VERSION_ID would hide a
// package whose min_distro_version no longer matches the fleet.
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
