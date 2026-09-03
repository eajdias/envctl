package usecase

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/eajdias/envctl/internal/domain/entity"
	"github.com/eajdias/envctl/internal/domain/repository"
)

type SnapshotSyncUseCase struct {
	manifestRepo repository.ManifestRepository
	fsManager    repository.FileSystemManager
	gitManager   repository.GitManager
	logger       repository.Logger
}

func NewSnapshotSyncUseCase(
	manifestRepo repository.ManifestRepository,
	fsManager repository.FileSystemManager,
	gitManager repository.GitManager,
	logger repository.Logger,
) *SnapshotSyncUseCase {
	return &SnapshotSyncUseCase{
		manifestRepo: manifestRepo,
		fsManager:    fsManager,
		gitManager:   gitManager,
		logger:       logger,
	}
}

type SnapshotResult struct {
	UpdatedFiles     []string
	DiscoveredSkills int
}

func (uc *SnapshotSyncUseCase) Execute(ctx context.Context) (*SnapshotResult, error) {
	if uc.logger != nil {
		uc.logger.Info("Starting reverse snapshot sync")
	}

	result := &SnapshotResult{}

	// 1. Sync Config files from system to configs/ (only when content differs)
	configFiles, _ := uc.manifestRepo.LoadConfigFiles()
	for _, cf := range configFiles {
		if cf.OS != "" && cf.OS != runtime.GOOS {
			continue
		}
		// Never reverse-sync sensitive or per-machine local files into the repo:
		// ~/.ssh/*, agent memory (~/.config/opencode/memory/*), and local extras
		// (~/.config/opencode/extras/*) are individual per PC/VPS.
		destLower := strings.ToLower(filepath.ToSlash(cf.Destination))
		if strings.Contains(destLower, ".ssh/") ||
			strings.Contains(destLower, ".config/opencode/memory") ||
			strings.Contains(destLower, ".config/opencode/extras") {
			continue
		}
		if uc.fsManager.Exists(cf.Destination) {
			data, err := uc.fsManager.ReadFile(cf.Destination)
			if err != nil {
				if uc.logger != nil {
					uc.logger.Warn("Snapshot: failed to read config '%s': %v", cf.Destination, err)
				}
				continue
			}
			if current, err := os.ReadFile(cf.Source); err == nil && string(current) == string(data) {
				continue // no drift: leave the curated file untouched
			}
			if err := os.MkdirAll(filepath.Dir(cf.Source), 0755); err != nil {
				if uc.logger != nil {
					uc.logger.Warn("Snapshot: failed to create dir for '%s': %v", cf.Source, err)
				}
				continue
			}
			if err := os.WriteFile(cf.Source, data, 0644); err != nil {
				if uc.logger != nil {
					uc.logger.Warn("Snapshot: failed to write '%s': %v", cf.Source, err)
				}
				continue
			}
			result.UpdatedFiles = append(result.UpdatedFiles, cf.Source)
			if uc.logger != nil {
				uc.logger.Info("Snapshot synced config '%s' -> '%s'", cf.Destination, cf.Source)
			}
		}
	}

	// 2. Discover skills in ~/.config/opencode/skills/ and sync to configs/skills/ & manifests/skills.yaml
	skillsDir, _ := uc.fsManager.ExpandUserPath("~/.config/opencode/skills")
	entries, err := os.ReadDir(skillsDir)
	if err == nil {
		var discoveredSkills []entity.Skill
		for _, entry := range entries {
			if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
				skillName := entry.Name()
				skillSrc := filepath.Join(skillsDir, skillName)
				skillDest := filepath.Join("configs", "skills", skillName)

				// Copy skill files
				if err := uc.copyDir(skillSrc, skillDest); err != nil && uc.logger != nil {
					uc.logger.Warn("Snapshot: failed to copy skill '%s': %v", skillName, err)
				}

				discoveredSkills = append(discoveredSkills, entity.Skill{
					Name:        skillName,
					Description: fmt.Sprintf("OpenCode agent skill %s", skillName),
					Source:      fmt.Sprintf("configs/skills/%s", skillName),
					Enabled:     true,
				})
			}
		}

		existingSkills, _ := uc.manifestRepo.LoadSkills()
		existingByName := map[string]entity.Skill{}
		for _, s := range existingSkills {
			existingByName[s.Name] = s
		}
		var merged []entity.Skill
		seen := map[string]bool{}
		for _, disc := range discoveredSkills {
			seen[disc.Name] = true
			if ex, ok := existingByName[disc.Name]; ok {
				// Preserve curated metadata (description, source, target_dir, enabled, files).
				merged = append(merged, entity.Skill{
					Name:        ex.Name,
					Description: ex.Description,
					Source:      ex.Source,
					TargetDir:   ex.TargetDir,
					Enabled:     ex.Enabled,
					Files:       ex.Files,
				})
				continue
			}
			merged = append(merged, disc)
		}
		// Keep curated entries that are no longer present on disk.
		for _, ex := range existingSkills {
			if !seen[ex.Name] {
				merged = append(merged, ex)
			}
		}

		if len(merged) > 0 {
			result.DiscoveredSkills = len(discoveredSkills)
			_ = uc.manifestRepo.SaveSkills(merged)
			result.UpdatedFiles = append(result.UpdatedFiles, "manifests/skills.yaml")
			if uc.logger != nil {
				uc.logger.Info("Snapshot discovered and cataloged %d agent skills", len(discoveredSkills))
			}
		}
	}

	// 3. Sync Git Configs
	gitKeys := []string{
		"core.fscache", "core.preloadindex", "core.longpaths", "core.autocrlf",
		"init.defaultBranch", "pull.rebase", "core.pager", "interactive.diffFilter",
		"delta.navigate", "delta.light", "delta.side-by-side", "delta.line-numbers",
	}
	// Preserve the OS constraint from the existing manifest so it is not lost on snapshot.
	existingGitConfigs, _ := uc.manifestRepo.LoadGitConfigs()
	osByKey := map[string]string{}
	for _, gc := range existingGitConfigs {
		osByKey[gc.Key] = gc.OS
	}
	knownWindowsOnly := map[string]bool{"core.fscache": true, "core.longpaths": true}
	var currentGitConfigs []entity.GitConfig
	for _, key := range gitKeys {
		val, err := uc.gitManager.GetGlobalConfig(ctx, key)
		if err == nil && val != "" {
			osVal := osByKey[key]
			if osVal == "" && knownWindowsOnly[key] {
				osVal = "windows"
			}
			currentGitConfigs = append(currentGitConfigs, entity.GitConfig{
				Key:   key,
				Value: val,
				OS:    osVal,
			})
		}
	}
	if len(currentGitConfigs) > 0 {
		// Preserve curated manifest keys not present on this machine so a
		// snapshot never drops versioned configuration.
		seen := map[string]bool{}
		for _, gc := range currentGitConfigs {
			seen[gc.Key] = true
		}
		for _, ex := range existingGitConfigs {
			if !seen[ex.Key] {
				currentGitConfigs = append(currentGitConfigs, ex)
			}
		}
		_ = uc.manifestRepo.SaveGitConfigs(currentGitConfigs)
		result.UpdatedFiles = append(result.UpdatedFiles, "manifests/git.yaml")
		if uc.logger != nil {
			uc.logger.Info("Snapshot captured %d global Git configurations", len(currentGitConfigs))
		}
	}

	return result, nil
}

func (uc *SnapshotSyncUseCase) copyDir(src, dst string) error {
	// Follow a top-level symlink (e.g. skill dir linked elsewhere) so the
	// target content is copied instead of the link itself.
	if resolved, err := filepath.EvalSymlinks(src); err == nil {
		src = resolved
	}
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		_ = os.MkdirAll(filepath.Dir(target), 0755)
		return os.WriteFile(target, data, 0644)
	})
}
