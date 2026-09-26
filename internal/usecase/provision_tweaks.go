package usecase

import (
	"context"
	"fmt"
	"runtime"

	"github.com/eajdias/envctl/internal/domain/entity"
	"github.com/eajdias/envctl/internal/domain/repository"
)

type tweakCheck struct {
	Tweak   entity.WindowsTweak
	OK      bool
	Details string
	Err     error
}

type tweakStackConfig struct {
	system      string
	skipMessage string
	load        func() ([]entity.WindowsTweak, error)
	loadErr     string
	check       func(ctx context.Context, tweaks []entity.WindowsTweak) ([]tweakCheck, error)
	applyHint   string
}

type ProvisionTweaksUseCase struct {
	manifestRepo  repository.ManifestRepository
	tweaksManager repository.WindowsTweaksManager
	logger        repository.Logger
	stack         tweakStackConfig
}

func TweakDisplayName(tweak entity.WindowsTweak) string {
	if tweak.Path == "" {
		return fmt.Sprintf("[%s] %s", tweak.Type, tweak.Name)
	}
	return fmt.Sprintf("%s\\%s", tweak.Path, tweak.Name)
}

func NewProvisionWindowsUseCase(
	manifestRepo repository.ManifestRepository,
	tweaksManager repository.WindowsTweaksManager,
	logger repository.Logger,
) *ProvisionTweaksUseCase {
	return &ProvisionTweaksUseCase{
		manifestRepo:  manifestRepo,
		tweaksManager: tweaksManager,
		logger:        logger,
		stack: tweakStackConfig{
			system:      "Windows11",
			skipMessage: "Skipping Windows tweaks (running on %s)",
			load:        manifestRepo.LoadWindowsTweaks,
			loadErr:     "failed to load windows tweaks manifest: %w",
			applyHint:   "Run terminal as Administrator if required for HKLM settings",
		},
	}
}

func NewProvisionDebloatUseCase(
	manifestRepo repository.ManifestRepository,
	tweaksManager repository.WindowsTweaksManager,
	logger repository.Logger,
) *ProvisionTweaksUseCase {
	uc := NewProvisionWindowsUseCase(manifestRepo, tweaksManager, logger)
	uc.stack.system = "Debloat"
	uc.stack.skipMessage = "Skipping Windows debloat (running on %s)"
	uc.stack.load = manifestRepo.LoadDebloatTweaks
	uc.stack.loadErr = "failed to load debloat manifest: %w"
	uc.stack.applyHint = "Run the terminal as Administrator: Appx -AllUsers, HKLM keys and service changes require elevation"
	uc.stack.check = func(ctx context.Context, tweaks []entity.WindowsTweak) ([]tweakCheck, error) {
		results := tweaksManager.CheckBatch(ctx, tweaks)
		checks := make([]tweakCheck, 0, len(results))
		for _, c := range results {
			checks = append(checks, tweakCheck{Tweak: c.Tweak, OK: c.OK, Details: c.Details, Err: c.Err})
		}
		return checks, nil
	}
	return uc
}

func (u *ProvisionTweaksUseCase) Execute(
	ctx context.Context,
	onProgress func(tweak entity.WindowsTweak, status string, details string),
) ([]entity.Diagnostic, error) {
	if runtime.GOOS != "windows" {
		u.logger.Info(u.stack.skipMessage, runtime.GOOS)
		return nil, nil
	}

	u.logger.Info("Starting %s tweak provisioning", u.stack.system)

	tweaks, err := u.stack.load()
	if err != nil {
		u.logger.Error("Failed to load %s manifest: %v", u.stack.system, err)
		return nil, fmt.Errorf(u.stack.loadErr, err)
	}

	checks, err := u.checkAll(ctx, tweaks)
	if err != nil {
		return nil, err
	}

	var results []entity.Diagnostic
	for _, c := range checks {
		results = append(results, u.applyOne(ctx, c, onProgress))
	}
	return results, nil
}

func (u *ProvisionTweaksUseCase) checkAll(ctx context.Context, tweaks []entity.WindowsTweak) ([]tweakCheck, error) {
	if u.stack.check != nil {
		return u.stack.check(ctx, tweaks)
	}
	checks := make([]tweakCheck, 0, len(tweaks))
	for _, tweak := range tweaks {
		ok, details, err := u.tweaksManager.CheckTweak(ctx, tweak)
		checks = append(checks, tweakCheck{Tweak: tweak, OK: ok, Details: details, Err: err})
	}
	return checks, nil
}

func (u *ProvisionTweaksUseCase) applyOne(
	ctx context.Context,
	c tweakCheck,
	onProgress func(tweak entity.WindowsTweak, status string, details string),
) entity.Diagnostic {
	targetName := TweakDisplayName(c.Tweak)
	progress := func(status, details string) {
		if onProgress != nil {
			onProgress(c.Tweak, status, details)
		}
	}
	fail := func(format string, args ...any) entity.Diagnostic {
		return entity.Diagnostic{
			Category: entity.DiagError,
			System:   u.stack.system,
			Target:   targetName,
			Details:  fmt.Sprintf(format, args...),
		}
	}

	progress("checking", "Verifying current system state")
	if c.Err != nil {
		u.logger.Error("Error checking %s tweak %s: %v", u.stack.system, targetName, c.Err)
		progress("failed", c.Err.Error())
		return fail("Check failed: %v", c.Err)
	}
	if c.OK {
		u.logger.LogIdempotency(u.stack.system, targetName, true, c.Details)
		progress("skipped", c.Details)
		return entity.Diagnostic{
			Category: entity.DiagOK,
			System:   u.stack.system,
			Target:   targetName,
			Details:  c.Details,
		}
	}

	progress("applying", "Applying Windows configuration")
	u.logger.Info("Applying %s tweak %s", u.stack.system, targetName)
	if err := u.tweaksManager.ApplyTweak(ctx, c.Tweak); err != nil {
		u.logger.Error("Failed to apply %s tweak %s: %v", u.stack.system, targetName, err)
		progress("failed", err.Error())
		d := fail("Apply failed: %v", err)
		d.FixHint = u.stack.applyHint
		return d
	}
	u.logger.LogIdempotency(u.stack.system, targetName, false, "Applied successfully")
	progress("applied", "Applied successfully")
	return entity.Diagnostic{
		Category: entity.DiagOK,
		System:   u.stack.system,
		Target:   targetName,
		Details:  "Applied successfully",
	}
}
