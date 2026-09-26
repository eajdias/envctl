package cli

import (
	"context"
	"fmt"

	"github.com/pterm/pterm"

	"github.com/eajdias/envctl/internal/domain/entity"
)

func runPerformanceProvisioning(ctx context.Context, dryRun bool) error {
	platform := entity.DetectedPlatform()
	metas, err := appCtx.ManifestRepo.ListPerformanceProfiles()
	if err != nil {
		return fmt.Errorf("failed to discover performance profiles: %w", err)
	}
	profile, err := selectPerformanceProfile(platform, metas)
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

// selectPerformanceProfile picks the profile this host satisfies. The release
// floor comes from each manifest, so no version is hard-coded here and raising
// the floor is a manifest edit.
func selectPerformanceProfile(
	platform entity.PlatformInfo,
	metas []entity.PerformanceProfileMeta,
) (entity.PerformanceProfile, error) {
	required := ""
	for _, meta := range metas {
		if entity.PerformanceProfileSatisfiedBy(meta.Profile, platform, meta.MinDistroVersion) {
			return meta.Profile, nil
		}
		if meta.MinDistroVersion != "" && (required == "" || meta.MinDistroVersion > required) {
			required = meta.MinDistroVersion
		}
	}
	if required != "" {
		return "", fmt.Errorf(
			"this host (%s %s, family=%s) does not satisfy any performance profile; the declared minimum release is %s",
			platform.ID, platform.VersionID, platform.Family, required,
		)
	}
	return "", fmt.Errorf(
		"no performance profile is available for %s %s (family=%s); no manifest declared a release minimum",
		platform.ID, platform.VersionID, platform.Family,
	)
}
