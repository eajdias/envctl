package usecase

import (
	"context"
	"fmt"
	"runtime"

	"github.com/eajdias/envctl/internal/domain/entity"
	"github.com/eajdias/envctl/internal/domain/repository"
)

type ProvisionLSPsUseCase struct {
	manifestRepo repository.ManifestRepository
	managers     map[entity.PackageType]repository.PackageManager
	logger       repository.Logger
}

func NewProvisionLSPsUseCase(
	manifestRepo repository.ManifestRepository,
	managers map[entity.PackageType]repository.PackageManager,
	logger repository.Logger,
) *ProvisionLSPsUseCase {
	return &ProvisionLSPsUseCase{
		manifestRepo: manifestRepo,
		managers:     managers,
		logger:       logger,
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
		if uc.logger != nil {
			uc.logger.Error("Failed to load LSP manifests: %v", err)
		}
		return nil, fmt.Errorf("failed to load LSP manifests: %w", err)
	}

	if uc.logger != nil {
		uc.logger.Info("Starting LSP servers provisioning (Total: %d LSPs)", len(lsps))
	}

	var results []LSPResult

	for _, lsp := range lsps {
		if lsp.OS != "" && lsp.OS != runtime.GOOS {
			continue
		}
		// Check if binary is already in PATH
		if lsp.CheckBinary != "" {
			if toolAvailable(ctx, lsp.CheckBinary) {
				if uc.logger != nil {
					uc.logger.LogIdempotency("LSP", lsp.ServerName, true, fmt.Sprintf("binary '%s' found in PATH", lsp.CheckBinary))
				}
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
			if uc.logger != nil {
				uc.logger.Warn("LSP '%s': Installer '%s' not available to install '%s'", lsp.ServerName, lsp.InstallType, lsp.InstallTarget)
			}
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

		if uc.logger != nil {
			uc.logger.LogIdempotency("LSP", lsp.ServerName, false, fmt.Sprintf("installing '%s' via %s", lsp.InstallTarget, lsp.InstallType))
		}

		if err := mgr.Install(ctx, pkg); err != nil {
			if uc.logger != nil {
				uc.logger.Error("Failed to install LSP '%s' (%s): %v", lsp.ServerName, lsp.InstallTarget, err)
			}
			results = append(results, LSPResult{
				LSP:          lsp,
				Status:       entity.DiagError,
				ErrorMessage: fmt.Sprintf("Failed to install %s: %v", lsp.InstallTarget, err),
			})
		} else {
			if uc.logger != nil {
				uc.logger.Info("Successfully installed LSP '%s' via %s", lsp.ServerName, lsp.InstallType)
			}
			results = append(results, LSPResult{
				LSP:     lsp,
				Status:  entity.DiagOK,
				Details: fmt.Sprintf("Successfully installed %s via %s", lsp.InstallTarget, lsp.InstallType),
			})
		}
	}

	return results, nil
}
