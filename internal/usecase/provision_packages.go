package usecase

import (
	"context"
	"fmt"
	"runtime"

	"github.com/eajdias/envctl/internal/domain/entity"
	"github.com/eajdias/envctl/internal/domain/repository"
)

type PackageProgressHandler func(pkg entity.Package, status string, err error)

type ProvisionPackagesUseCase struct {
	manifestRepo repository.ManifestRepository
	managers     map[entity.PackageType]repository.PackageManager
	logger       repository.Logger
	platform     func() entity.PlatformInfo
}

func NewProvisionPackagesUseCase(
	manifestRepo repository.ManifestRepository,
	managers map[entity.PackageType]repository.PackageManager,
	logger repository.Logger,
) *ProvisionPackagesUseCase {
	return &ProvisionPackagesUseCase{
		manifestRepo: manifestRepo,
		managers:     managers,
		logger:       logger,
		platform:     entity.DetectedPlatform,
	}
}

func (uc *ProvisionPackagesUseCase) Execute(ctx context.Context, filterType entity.PackageType, onProgress PackageProgressHandler) ([]entity.Package, error) {
	allPkgs, err := uc.manifestRepo.LoadPackages()
	if err != nil {
		uc.logger.Error("Failed to load package manifests: %v", err)
		return nil, fmt.Errorf("failed to load package manifests: %w", err)
	}

	uc.logger.Info("Starting package provisioning (Total: %d manifests, Filter: '%s')", len(allPkgs), filterType)

	return uc.provisionList(ctx, allPkgs, filterType, onProgress)
}

// ExecuteGaming provisions the opt-in gaming manifest (gaming.yaml): no type
// filter is applied, so pacman and paru entries install side by side.
func (uc *ProvisionPackagesUseCase) ExecuteGaming(ctx context.Context, onProgress PackageProgressHandler) ([]entity.Package, error) {
	gamingPkgs, err := uc.manifestRepo.LoadGamingPackages()
	if err != nil {
		uc.logger.Error("Failed to load gaming manifests: %v", err)
		return nil, fmt.Errorf("failed to load gaming manifests: %w", err)
	}

	uc.logger.Info("Starting gaming provisioning (Total: %d manifests)", len(gamingPkgs))

	if entity.DetectedDistro() != entity.DistroArch && runtime.GOOS == "linux" {
		return nil, fmt.Errorf("gaming stack is Arch/CachyOS-only (this host: %q); refusing to install Steam/GUI packages on a server", entity.DetectedDistro())
	}

	return uc.provisionList(ctx, gamingPkgs, "", onProgress)
}

// packageOwnershipProbe removes a check command only where a generic command
// probe would confuse a user-local binary with a distro package. All other
// package types retain their manifest check behavior.
func packageOwnershipProbe(pkg entity.Package) entity.Package {
	if pkg.Type == entity.PackageTypePacman && pkg.ID == "opencode" {
		pkg.CheckCommand = ""
	}
	return pkg
}

// ExecuteList provisions an explicitly supplied package list. It is used by
// opt-in profiles so they cannot accidentally load the general packages.yaml.
func (uc *ProvisionPackagesUseCase) ExecuteList(ctx context.Context, allPkgs []entity.Package, filterType entity.PackageType, dryRun bool, onProgress PackageProgressHandler) ([]entity.Package, error) {
	return uc.provisionListMode(ctx, allPkgs, filterType, dryRun, onProgress)
}

func (uc *ProvisionPackagesUseCase) provisionList(ctx context.Context, allPkgs []entity.Package, filterType entity.PackageType, onProgress PackageProgressHandler) ([]entity.Package, error) {
	return uc.provisionListMode(ctx, allPkgs, filterType, false, onProgress)
}

func (uc *ProvisionPackagesUseCase) provisionListMode(ctx context.Context, allPkgs []entity.Package, filterType entity.PackageType, dryRun bool, onProgress PackageProgressHandler) ([]entity.Package, error) {

	var results []entity.Package
	platform := uc.platform
	if platform == nil {
		platform = entity.DetectedPlatform
	}
	currentPlatform := platform()
	if currentPlatform.GOOS == "" {
		currentPlatform = entity.DetectedPlatform()
	}

	for _, pkg := range allPkgs {
		if !entity.PackageMatchesPlatform(pkg, currentPlatform) {
			continue
		}

		if filterType != "" && pkg.Type != filterType {
			continue
		}

		mgr, ok := uc.managers[pkg.Type]
		if !ok {
			pkg.Status = entity.StatusSkipped
			pkg.Error = fmt.Sprintf("unsupported package manager: %s", pkg.Type)
			results = append(results, pkg)
			uc.logger.Warn("Unsupported package manager '%s' for package '%s'", pkg.Type, pkg.ID)
			if onProgress != nil {
				onProgress(pkg, "unsupported manager", nil)
			}
			continue
		}

		if !mgr.IsAvailable(ctx) {
			pkg.Status = entity.StatusSkipped
			pkg.Error = fmt.Sprintf("package manager %s is not available on this system", pkg.Type)
			results = append(results, pkg)
			uc.logger.Warn("Package manager '%s' is not available for package '%s'", pkg.Type, pkg.ID)
			if onProgress != nil {
				onProgress(pkg, "manager not available", nil)
			}
			continue
		}

		// A user-local opencode can satisfy the manifest's check_command even
		// when no pacman package exists. Probe the package database for this
		// one entry so run all cannot skip the authoritative Arch install.
		probePkg := packageOwnershipProbe(pkg)
		isInstalled, info, probeErr := mgr.IsInstalled(ctx, probePkg)
		if probeErr != nil {
			uc.logger.Warn("Package manager '%s' could not query '%s': %v", pkg.Type, pkg.ID, probeErr)
			isInstalled = false
		}
		if isInstalled {
			pkg.Status = entity.StatusInstalled
			pkg.Version = info
			results = append(results, pkg)
			uc.logger.LogIdempotency("Package", pkg.ID, true, fmt.Sprintf("already installed (%s)", info))
			if onProgress != nil {
				onProgress(pkg, fmt.Sprintf("already installed (%s)", info), nil)
			}
			continue
		}

		if dryRun {
			pkg.Status = entity.StatusMissing
			results = append(results, pkg)
			if onProgress != nil {
				onProgress(pkg, "would install (dry-run)", nil)
			}
			continue
		}

		// Install package
		uc.logger.LogIdempotency("Package", pkg.ID, false, fmt.Sprintf("triggering installation via %s", pkg.Type))
		if onProgress != nil {
			onProgress(pkg, "installing...", nil)
		}

		if err := mgr.Install(ctx, pkg); err != nil {
			pkg.Status = entity.StatusFailed
			pkg.Error = err.Error()
			results = append(results, pkg)
			uc.logger.Error("Failed to install package '%s' (%s): %v", pkg.ID, pkg.Type, err)
			if onProgress != nil {
				onProgress(pkg, "failed", err)
			}
		} else {
			pkg.Status = entity.StatusInstalled
			results = append(results, pkg)
			uc.logger.Info("Successfully installed package '%s' (%s)", pkg.ID, pkg.Type)
			if onProgress != nil {
				onProgress(pkg, "installed successfully", nil)
			}
		}
	}

	return results, nil
}
