package usecase

import (
	"context"
	"fmt"

	"github.com/eajdias/win11-new/internal/domain/entity"
	"github.com/eajdias/win11-new/internal/domain/repository"
)

// ProvisionWindowsUseCase coordinates the application of Windows 11 system tweaks, registry settings, and fonts.
type ProvisionWindowsUseCase struct {
	manifestRepo  repository.ManifestRepository
	tweaksManager repository.WindowsTweaksManager
	logger        repository.Logger
}

func NewProvisionWindowsUseCase(
	manifestRepo repository.ManifestRepository,
	tweaksManager repository.WindowsTweaksManager,
	logger repository.Logger,
) *ProvisionWindowsUseCase {
	return &ProvisionWindowsUseCase{
		manifestRepo:  manifestRepo,
		tweaksManager: tweaksManager,
		logger:        logger,
	}
}

func (u *ProvisionWindowsUseCase) Execute(
	ctx context.Context,
	onProgress func(tweak entity.WindowsTweak, status string, details string),
) ([]entity.Diagnostic, error) {
	if u.logger != nil {
		u.logger.Info("Starting ProvisionWindowsUseCase execution")
	}

	tweaks, err := u.manifestRepo.LoadWindowsTweaks()
	if err != nil {
		if u.logger != nil {
			u.logger.Error("Failed to load windows.yaml manifest: %v", err)
		}
		return nil, fmt.Errorf("failed to load windows tweaks manifest: %w", err)
	}

	var results []entity.Diagnostic
	for _, tweak := range tweaks {
		targetName := fmt.Sprintf("%s\\%s", tweak.Path, tweak.Name)
		if tweak.Path == "" {
			targetName = fmt.Sprintf("[%s] %s", tweak.Type, tweak.Name)
		}

		if onProgress != nil {
			onProgress(tweak, "checking", "Verifying current system state")
		}

		ok, details, err := u.tweaksManager.CheckTweak(ctx, tweak)
		if err != nil {
			if u.logger != nil {
				u.logger.Error("Error checking Windows tweak %s: %v", targetName, err)
			}
			if onProgress != nil {
				onProgress(tweak, "failed", err.Error())
			}
			results = append(results, entity.Diagnostic{
				Category: entity.DiagError,
				System:   "Windows11",
				Target:   targetName,
				Details:  fmt.Sprintf("Check failed: %v", err),
			})
			continue
		}

		if ok {
			if u.logger != nil {
				u.logger.LogIdempotency("Windows11", targetName, true, details)
			}
			if onProgress != nil {
				onProgress(tweak, "skipped", details)
			}
			results = append(results, entity.Diagnostic{
				Category: entity.DiagOK,
				System:   "Windows11",
				Target:   targetName,
				Details:  details,
			})
			continue
		}

		// Apply tweak
		if onProgress != nil {
			onProgress(tweak, "applying", "Applying Windows configuration")
		}
		if u.logger != nil {
			u.logger.Info("Applying Windows tweak %s", targetName)
		}

		if err := u.tweaksManager.ApplyTweak(ctx, tweak); err != nil {
			if u.logger != nil {
				u.logger.Error("Failed to apply Windows tweak %s: %v", targetName, err)
			}
			if onProgress != nil {
				onProgress(tweak, "failed", err.Error())
			}
			results = append(results, entity.Diagnostic{
				Category: entity.DiagError,
				System:   "Windows11",
				Target:   targetName,
				Details:  fmt.Sprintf("Apply failed: %v", err),
				FixHint:  "Run terminal as Administrator if required for HKLM settings",
			})
		} else {
			if u.logger != nil {
				u.logger.LogIdempotency("Windows11", targetName, false, "Applied successfully")
			}
			if onProgress != nil {
				onProgress(tweak, "applied", "Applied successfully")
			}
			results = append(results, entity.Diagnostic{
				Category: entity.DiagOK,
				System:   "Windows11",
				Target:   targetName,
				Details:  "Applied successfully",
			})
		}
	}

	return results, nil
}
