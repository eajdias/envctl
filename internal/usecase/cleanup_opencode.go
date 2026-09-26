package usecase

import (
	"context"
	"fmt"
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
	// Store reports the OpenCode session store when it exceeded the audit
	// threshold, with StoreNote describing what was (or could not be) reclaimed.
	Store     *OpenCodeStore
	StoreNote string
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
				uc.logger.Info("[CLEANUP] removed legacy config %s", stale)
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
				uc.logger.Info("[CLEANUP] removed oversized tool-output %s (%.1f MB)", fPath, float64(info.Size())/(1024*1024))
			}
		}
	}

	// 3. OpenCode session store: reclaim free pages when there are any. Live
	// rows are never rewritten behind the user's back, so a store that grew
	// from real session history is reported rather than compacted.
	dbPath := openCodeStorePath(homeDir)
	if store, storeErr := InspectOpenCodeStore(dbPath); storeErr == nil && store.ExceedsThreshold() {
		result.Store = &store
		if store.ReclaimableBytes > 0 {
			freed, vacuumErr := vacuumOpenCodeStore(ctx, dbPath)
			if vacuumErr != nil {
				result.StoreNote = fmt.Sprintf("%.1f MB, %.1f MB reclaimable but VACUUM could not run: %v",
					float64(store.SizeBytes)/(1024*1024), float64(store.ReclaimableBytes)/(1024*1024), vacuumErr)
				uc.logger.Warn("[CLEANUP] opencode.db VACUUM skipped: %v", vacuumErr)
			} else {
				result.FreedBytes += freed
				result.StoreNote = fmt.Sprintf("%.1f MB, reclaimed %.1f MB (VACUUM)",
					float64(store.SizeBytes)/(1024*1024), float64(freed)/(1024*1024))
				uc.logger.Info("[CLEANUP] opencode.db: reclaimed %.1f MB", float64(freed)/(1024*1024))
			}
		} else {
			result.StoreNote = fmt.Sprintf("%.1f MB of live session data (0 MB reclaimable — prune sessions to shrink it)",
				float64(store.SizeBytes)/(1024*1024))
			uc.logger.Info("[CLEANUP] opencode.db holds %.1f MB of live data; nothing to reclaim", float64(store.SizeBytes)/(1024*1024))
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
			uc.logger.Info("[CLEANUP] removed stale scratch %s", path)
		}
	}

	return result, nil
}
