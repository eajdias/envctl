package usecase

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/eajdias/win11-new/internal/domain/entity"
	"github.com/eajdias/win11-new/internal/domain/repository"
)

type ProvisionLSPsUseCase struct {
	manifestRepo repository.ManifestRepository
	managers     map[entity.PackageType]repository.PackageManager
}

func NewProvisionLSPsUseCase(
	manifestRepo repository.ManifestRepository,
	managers map[entity.PackageType]repository.PackageManager,
) *ProvisionLSPsUseCase {
	return &ProvisionLSPsUseCase{
		manifestRepo: manifestRepo,
		managers:     managers,
	}
}

type LSPResult struct {
	LSP          entity.LSP
	Status       entity.DiagnosticStatus
	Details      string
	ErrorMessage string
}

func (uc *ProvisionLSPsUseCase) Execute(ctx context.Context) ([]LSPResult, error) {
	lsps, err := uc.manifestRepo.LoadLSPs()
	if err != nil {
		return nil, fmt.Errorf("failed to load LSP manifests: %w", err)
	}

	var results []LSPResult

	for _, lsp := range lsps {
		// Check if binary is already in PATH
		if lsp.CheckBinary != "" {
			if _, lookErr := exec.LookPath(lsp.CheckBinary); lookErr == nil {
				results = append(results, LSPResult{
					LSP:     lsp,
					Status:  entity.DiagOK,
					Details: fmt.Sprintf("Binary %s found in PATH", lsp.CheckBinary),
				})
				continue
			}
		}

		mgr, ok := uc.managers[lsp.InstallType]
		if !ok || !mgr.IsAvailable(ctx) {
			results = append(results, LSPResult{
				LSP:          lsp,
				Status:       entity.DiagWarning,
				ErrorMessage: fmt.Sprintf("Installer %s not available to install %s", lsp.InstallType, lsp.InstallTarget),
			})
			continue
		}

		pkg := entity.Package{
			ID:   lsp.InstallTarget,
			Type: lsp.InstallType,
		}

		if err := mgr.Install(ctx, pkg); err != nil {
			results = append(results, LSPResult{
				LSP:          lsp,
				Status:       entity.DiagError,
				ErrorMessage: fmt.Sprintf("Failed to install %s: %v", lsp.InstallTarget, err),
			})
		} else {
			results = append(results, LSPResult{
				LSP:     lsp,
				Status:  entity.DiagOK,
				Details: fmt.Sprintf("Successfully installed %s via %s", lsp.InstallTarget, lsp.InstallType),
			})
		}
	}

	return results, nil
}
