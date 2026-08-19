package usecase

import (
	"context"
	"fmt"

	"github.com/eajdias/win11-new/internal/domain/entity"
	"github.com/eajdias/win11-new/internal/domain/repository"
)

type PackageProgressHandler func(pkg entity.Package, status string, err error)

type ProvisionPackagesUseCase struct {
	manifestRepo repository.ManifestRepository
	managers     map[entity.PackageType]repository.PackageManager
	logger       repository.Logger
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
	}
}

func (uc *ProvisionPackagesUseCase) Execute(ctx context.Context, filterType entity.PackageType, onProgress PackageProgressHandler) ([]entity.Package, error) {
	allPkgs, err := uc.manifestRepo.LoadPackages()
	if err != nil {
		if uc.logger != nil {
			uc.logger.Error("Failed to load package manifests: %v", err)
		}
		return nil, fmt.Errorf("failed to load package manifests: %w", err)
	}

	if uc.logger != nil {
		uc.logger.Info("Starting package provisioning (Total: %d manifests, Filter: '%s')", len(allPkgs), filterType)
	}

	var results []entity.Package

	for _, pkg := range allPkgs {
		if filterType != "" && pkg.Type != filterType {
			continue
		}

		mgr, ok := uc.managers[pkg.Type]
		if !ok {
			pkg.Status = entity.StatusSkipped
			pkg.Error = fmt.Sprintf("unsupported package manager: %s", pkg.Type)
			results = append(results, pkg)
			if uc.logger != nil {
				uc.logger.Warn("Unsupported package manager '%s' for package '%s'", pkg.Type, pkg.ID)
			}
			if onProgress != nil {
				onProgress(pkg, "unsupported manager", nil)
			}
			continue
		}

		if !mgr.IsAvailable(ctx) {
			pkg.Status = entity.StatusSkipped
			pkg.Error = fmt.Sprintf("package manager %s is not available on this system", pkg.Type)
			results = append(results, pkg)
			if uc.logger != nil {
				uc.logger.Warn("Package manager '%s' is not available for package '%s'", pkg.Type, pkg.ID)
			}
			if onProgress != nil {
				onProgress(pkg, "manager not available", nil)
			}
			continue
		}

		isInstalled, info, _ := mgr.IsInstalled(ctx, pkg)
		if isInstalled {
			pkg.Status = entity.StatusInstalled
			pkg.Version = info
			results = append(results, pkg)
			if uc.logger != nil {
				uc.logger.LogIdempotency("Package", pkg.ID, true, fmt.Sprintf("already installed (%s)", info))
			}
			if onProgress != nil {
				onProgress(pkg, fmt.Sprintf("already installed (%s)", info), nil)
			}
			continue
		}

		// Install package
		if uc.logger != nil {
			uc.logger.LogIdempotency("Package", pkg.ID, false, fmt.Sprintf("triggering installation via %s", pkg.Type))
		}
		if onProgress != nil {
			onProgress(pkg, "installing...", nil)
		}

		if err := mgr.Install(ctx, pkg); err != nil {
			pkg.Status = entity.StatusFailed
			pkg.Error = err.Error()
			results = append(results, pkg)
			if uc.logger != nil {
				uc.logger.Error("Failed to install package '%s' (%s): %v", pkg.ID, pkg.Type, err)
			}
			if onProgress != nil {
				onProgress(pkg, "failed", err)
			}
		} else {
			pkg.Status = entity.StatusInstalled
			results = append(results, pkg)
			if uc.logger != nil {
				uc.logger.Info("Successfully installed package '%s' (%s)", pkg.ID, pkg.Type)
			}
			if onProgress != nil {
				onProgress(pkg, "installed successfully", nil)
			}
		}
	}

	return results, nil
}
