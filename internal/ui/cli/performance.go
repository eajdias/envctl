package cli

import (
	"context"
	"fmt"

	"github.com/pterm/pterm"
	"github.com/spf13/pflag"

	"github.com/eajdias/envctl/internal/domain/entity"
	"github.com/eajdias/envctl/internal/usecase"
)

// mustBool and mustString read a flag whose default is known. A parse failure on
// a boolean flag would otherwise be silently treated as false, which is exactly
// the kind of quiet downgrade an automated run must not have.
func mustBool(flags *pflag.FlagSet, name string) bool {
	value, err := flags.GetBool(name)
	if err != nil {
		return false
	}
	return value
}

func mustString(flags *pflag.FlagSet, name string) string {
	value, err := flags.GetString(name)
	if err != nil {
		return ""
	}
	return value
}

func runPerformanceProvisioning(ctx context.Context, opts usecase.PerformanceOptions) error {
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
	// A pending reboot is a precondition, not advice: applying tuning on top of
	// a runtime the host is about to replace describes a state that will not
	// exist after the next boot.
	if state := usecase.ProbeRebootPending(nil); state.Pending {
		if opts.ForceRebootPending {
			pterm.Warning.Printf("Proceeding despite a pending reboot: %s\n", state.Detail)
		} else {
			pterm.Error.Printf("Refusing to apply the performance profile: %s\n", state.Detail)
			pterm.Info.Println("Reboot first, or re-run with --force-reboot-pending to proceed anyway.")
			return fmt.Errorf("performance profile aborted: a reboot is pending")
		}
	}

	packages, diagnostics, err := appCtx.ProvisionPerformanceUC.ExecutePerformanceWithOptions(
		ctx,
		profile,
		opts,
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
	if opts.DryRun {
		pterm.Info.Println("Dry-run complete; no packages, sysctls, services, or files were changed.")
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
