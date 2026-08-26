package usecase

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/eajdias/envctl/internal/domain/repository"
)

const (
	cleanupToolOutputMinBytes = 10 * 1024 * 1024
	cleanupTempMaxAge         = 24 * time.Hour
)

// CleanupOpenCodeUseCase prunes OpenCode storage accumulation: legacy configs,
// oversized tool-output files and stale scratch in the standardized agent temp
// folder (ENVCTL_TEMP). Plugin cache entries are NOT pruned — opencode
// recreates the full referenced set on every start (verified empirically), so
// pruning them only causes re-download churn.
type CleanupOpenCodeUseCase struct {
	fsManager repository.FileSystemManager
	logger    repository.Logger
}

func NewCleanupOpenCodeUseCase(
	fsManager repository.FileSystemManager,
	logger repository.Logger,
) *CleanupOpenCodeUseCase {
	return &CleanupOpenCodeUseCase{
		fsManager: fsManager,
		logger:    logger,
	}
}

// CleanupResult summarizes a cleanup pass.
type CleanupResult struct {
	RemovedFiles []string
	FreedBytes   int64
}

func (uc *CleanupOpenCodeUseCase) Execute(ctx context.Context) (*CleanupResult, error) {
	result := &CleanupResult{}

	homeDir, _ := uc.fsManager.ExpandUserPath("~")

	// 1. Remove legacy opencode config files (standardized on opencode.json).
	for _, stale := range []string{
		filepath.Join(homeDir, ".config", "opencode", "opencode.jsonc"),
		filepath.Join(homeDir, ".config", "opencode", "opencode.linux.jsonc"),
	} {
		if uc.fsManager.Exists(stale) {
			if info, err := os.Stat(stale); err == nil {
				os.Remove(stale)
				result.RemovedFiles = append(result.RemovedFiles, stale)
				result.FreedBytes += info.Size()
				if uc.logger != nil {
					uc.logger.Info("[CLEANUP] removed legacy config %s", stale)
				}
			}
		}
	}

	// 2. Remove oversized tool-output files.
	toolOutputDir := filepath.Join(homeDir, ".local", "share", "opencode", "tool-output")
	if entries, err := os.ReadDir(toolOutputDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			fPath := filepath.Join(toolOutputDir, entry.Name())
			if info, err := os.Stat(fPath); err == nil && info.Size() > cleanupToolOutputMinBytes {
				os.Remove(fPath)
				result.RemovedFiles = append(result.RemovedFiles, fPath)
				result.FreedBytes += info.Size()
				if uc.logger != nil {
					uc.logger.Info("[CLEANUP] removed oversized tool-output %s (%.1f MB)", fPath, float64(info.Size())/(1024*1024))
				}
			}
		}
	}

	// 4. Prune stale scratch in the standardized agent temp folder (ENVCTL_TEMP).
	var tempDir string
	if runtime.GOOS == "windows" {
		tempDir = `C:\temp`
	} else {
		tempDir = "/temp"
	}
	if entries, err := os.ReadDir(tempDir); err == nil {
		now := time.Now()
		for _, entry := range entries {
			if ctx.Err() != nil {
				return result, ctx.Err()
			}
			info, err := entry.Info()
			if err != nil {
				continue
			}
			if now.Sub(info.ModTime()) <= cleanupTempMaxAge {
				continue
			}
			path := filepath.Join(tempDir, entry.Name())
			size, _ := dirSize(path)
			if err := os.RemoveAll(path); err != nil {
				continue
			}
			result.RemovedFiles = append(result.RemovedFiles, path)
			result.FreedBytes += size
			if uc.logger != nil {
				uc.logger.Info("[CLEANUP] removed stale scratch %s", path)
			}
		}
	}

	return result, nil
}