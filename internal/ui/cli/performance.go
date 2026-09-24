package cli

import (
	"context"
	"fmt"

	"github.com/pterm/pterm"

	"github.com/eajdias/envctl/internal/domain/entity"
)

func runPerformanceProvisioning(ctx context.Context, dryRun bool) error {
	platform := entity.DetectedPlatform()
	profile, err := selectPerformanceProfile(platform)
	if err != nil {
		return err
	}

	pterm.DefaultHeader.WithFullWidth().Println(
		fmt.Sprintf("Performance profile: %s (%s %s)", profile, platform.ID, platform.VersionID),
	)
	packages, diagnostics, err := appCtx.ProvisionPerformanceUC.ExecutePerformance(
		ctx,
		profile,
		dryRun,
		func(pkg entity.Package, status string, err error) {
			if err != nil {
				pterm.Warning.Printf("  • %s: %s (%v)\n", pkg.ID, status, err)
				return
			}
			pterm.Info.Printf("  • %s: %s\n", pkg.ID, status)
		},
	)
	for _, diagnostic := range diagnostics {
		switch diagnostic.Category {
		case entity.DiagOK:
			pterm.Success.Printf("  • [%s] %s: %s\n", diagnostic.System, diagnostic.Target, diagnostic.Details)
		case entity.DiagWarning:
			pterm.Warning.Printf("  • [%s] %s: %s\n", diagnostic.System, diagnostic.Target, diagnostic.Details)
		case entity.DiagError:
			pterm.Error.Printf("  • [%s] %s: %s\n", diagnostic.System, diagnostic.Target, diagnostic.Details)
		default:
			pterm.Info.Printf("  • [%s] %s: %s\n", diagnostic.System, diagnostic.Target, diagnostic.Details)
		}
	}
	if err != nil {
		return err
	}
	if len(packages) == 0 {
		pterm.Info.Println("  • no performance packages declared for this profile")
	}
	if dryRun {
		pterm.Info.Println("Dry-run complete; no packages, sysctls, or services were changed.")
	}
	return nil
}

func selectPerformanceProfile(platform entity.PlatformInfo) (entity.PerformanceProfile, error) {
	if entity.PerformanceProfileMatchesPlatform(entity.PerformanceProfileUbuntu, platform) {
		return entity.PerformanceProfileUbuntu, nil
	}
	if entity.PerformanceProfileMatchesPlatform(entity.PerformanceProfileCachyOS, platform) {
		return entity.PerformanceProfileCachyOS, nil
	}
	return "", fmt.Errorf(
		"no performance profile is available for %s %s (family=%s); supported profiles are Ubuntu >= 24.04 and CachyOS",
		platform.ID, platform.VersionID, platform.Family,
	)
}
