package usecase

import (
	"context"
	"os"
	"path/filepath"

	"github.com/eajdias/envctl/internal/domain/repository"
)

// CleanupCommandCodeUseCase prunes CommandCode storage accumulation:
// stale config variants and oversized artifacts.
type CleanupCommandCodeUseCase struct {
	fsManager repository.FileSystemManager
	logger    repository.Logger
}

func NewCleanupCommandCodeUseCase(
	fsManager repository.FileSystemManager,
	logger repository.Logger,
) *CleanupCommandCodeUseCase {
	return &CleanupCommandCodeUseCase{
		fsManager: fsManager,
		logger:    logger,
	}
}

func (uc *CleanupCommandCodeUseCase) Execute(ctx context.Context) (*CleanupResult, error) {
	result := &CleanupResult{}

	homeDir, _ := uc.fsManager.ExpandUserPath("~")
	ccDir := filepath.Join(homeDir, ".commandcode")

	if !uc.fsManager.Exists(ccDir) {
		return result, nil
	}

	stale := filepath.Join(ccDir, "settings.jsonc")
	if uc.fsManager.Exists(stale) {
		if info, err := os.Stat(stale); err == nil {
			os.Remove(stale)
			result.RemovedFiles = append(result.RemovedFiles, stale)
			result.FreedBytes += info.Size()
			if uc.logger != nil {
				uc.logger.Info("[CLEANUP] removed stale CommandCode config %s", stale)
			}
		}
	}

	return result, nil
}
