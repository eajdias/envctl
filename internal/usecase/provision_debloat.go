package usecase

import (
	"context"
	"fmt"
	"runtime"

	"github.com/eajdias/envctl/internal/domain/entity"
	"github.com/eajdias/envctl/internal/domain/repository"
)

// ProvisionDebloatUseCase applies the opt-in Windows 11 debloat stack
// (telemetry/privacy registry, gaming visuals, Appx removals, safe-only
// service disables) from manifests/debloat.yaml. It never runs inside
// `run all` or `run windows`: the caller must invoke `run debloat`
// explicitly, and destructive Tier 3 steps stay in the
// docs/guides/windows-debloat-tier3.md
// skill as manual guidance.
type ProvisionDebloatUseCase struct {
	manifestRepo  repository.ManifestRepository
	tweaksManager repository.WindowsTweaksManager
	logger        repository.Logger
}

func NewProvisionDebloatUseCase(
	manifestRepo repository.ManifestRepository,
	tweaksManager repository.WindowsTweaksManager,
	logger repository.Logger,
) *ProvisionDebloatUseCase {
	return &ProvisionDebloatUseCase{
		manifestRepo:  manifestRepo,
		tweaksManager: tweaksManager,
		logger:        logger,
	}
}

func debloatTargetName(tweak entity.WindowsTweak) string {
	targetName := fmt.Sprintf("%s\\%s", tweak.Path, tweak.Name)
	if tweak.Path == "" {
		targetName = fmt.Sprintf("[%s] %s", tweak.Type, tweak.Name)
	}
	return targetName
}

func (u *ProvisionDebloatUseCase) Execute(
	ctx context.Context,
	onProgress func(tweak entity.WindowsTweak, status string, details string),
) ([]entity.Diagnostic, error) {
	if runtime.GOOS != "windows" {
		if u.logger != nil {
			u.logger.Info("Skipping Windows debloat (running on %s)", runtime.GOOS)
		}
		return nil, nil
	}

	if u.logger != nil {
		u.logger.Info("Starting ProvisionDebloatUseCase execution")
	}

	tweaks, err := u.manifestRepo.LoadDebloatTweaks()
	if err != nil {
		if u.logger != nil {
			u.logger.Error("Failed to load debloat.yaml manifest: %v", err)
		}
		return nil, fmt.Errorf("failed to load debloat manifest: %w", err)
	}

	var results []entity.Diagnostic
	checks := u.tweaksManager.CheckBatch(ctx, tweaks)
	for _, c := range checks {
		tweak := c.Tweak
		targetName := debloatTargetName(tweak)

		if onProgress != nil {
			onProgress(tweak, "checking", "Verifying current system state")
		}

		err := c.Err
		ok := c.OK
		details := c.Details
		if err != nil {
			if u.logger != nil {
				u.logger.Error("Error checking debloat tweak %s: %v", targetName, err)
			}
			if onProgress != nil {
				onProgress(tweak, "failed", err.Error())
			}
			results = append(results, entity.Diagnostic{
				Category: entity.DiagError,
				System:   "Debloat",
				Target:   targetName,
				Details:  fmt.Sprintf("Check failed: %v", err),
			})
			continue
		}

		if ok {
			if u.logger != nil {
				u.logger.LogIdempotency("Debloat", targetName, true, details)
			}
			if onProgress != nil {
				onProgress(tweak, "skipped", details)
			}
			results = append(results, entity.Diagnostic{
				Category: entity.DiagOK,
				System:   "Debloat",
				Target:   targetName,
				Details:  details,
			})
			continue
		}

		// Apply tweak
		if onProgress != nil {
			onProgress(tweak, "applying", "Applying debloat change")
		}
		if u.logger != nil {
			u.logger.Info("Applying debloat tweak %s", targetName)
		}

		if err := u.tweaksManager.ApplyTweak(ctx, tweak); err != nil {
			if u.logger != nil {
				u.logger.Error("Failed to apply debloat tweak %s: %v", targetName, err)
			}
			if onProgress != nil {
				onProgress(tweak, "failed", err.Error())
			}
			results = append(results, entity.Diagnostic{
				Category: entity.DiagError,
				System:   "Debloat",
				Target:   targetName,
				Details:  fmt.Sprintf("Apply failed: %v", err),
				FixHint:  "Run the terminal as Administrator: Appx -AllUsers, HKLM keys and service changes require elevation",
			})
		} else {
			if u.logger != nil {
				u.logger.LogIdempotency("Debloat", targetName, false, "Applied successfully")
			}
			if onProgress != nil {
				onProgress(tweak, "applied", "Applied successfully")
			}
			results = append(results, entity.Diagnostic{
				Category: entity.DiagOK,
				System:   "Debloat",
				Target:   targetName,
				Details:  "Applied successfully",
			})
		}
	}

	return results, nil
}
