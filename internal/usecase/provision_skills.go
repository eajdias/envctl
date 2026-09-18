package usecase

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/eajdias/envctl/internal/domain/entity"
	"github.com/eajdias/envctl/internal/domain/repository"
)

type ProvisionSkillsUseCase struct {
	manifestRepo repository.ManifestRepository
	fsManager    repository.FileSystemManager
	embeddedFS   fs.FS
	logger       repository.Logger
}

func NewProvisionSkillsUseCase(
	manifestRepo repository.ManifestRepository,
	fsManager repository.FileSystemManager,
	embeddedFS fs.FS,
	logger repository.Logger,
) *ProvisionSkillsUseCase {
	return &ProvisionSkillsUseCase{
		manifestRepo: manifestRepo,
		fsManager:    fsManager,
		embeddedFS:   embeddedFS,
		logger:       logger,
	}
}

type SkillDeployResult struct {
	SkillName    string
	TargetDir    string
	FilesCopied  int
	Status       entity.DiagnosticStatus
	ErrorMessage string
}

// Execute deploys the manifest skills into targetBaseDir and removes any skill
// directory that is no longer in the manifest, so the target always mirrors the
// manifest. It returns the per-skill deploy results and the names of the pruned
// (stale) skill directories.
func (uc *ProvisionSkillsUseCase) Execute(ctx context.Context, targetBaseDir string) ([]SkillDeployResult, []string, error) {
	if targetBaseDir == "" {
		targetBaseDir = "~/.config/opencode/skills"
	}

	skills, err := uc.manifestRepo.LoadSkills()
	if err != nil {
		if uc.logger != nil {
			uc.logger.Error("Failed to load skills manifest: %v", err)
		}
		return nil, nil, fmt.Errorf("failed to load skills manifest: %w", err)
	}

	if uc.logger != nil {
		uc.logger.Info("Starting agent skills provisioning (Total: %d skills, Target: '%s')", len(skills), targetBaseDir)
	}

	wanted := make(map[string]bool, len(skills))
	goos := runtime.GOOS
	for _, skill := range skills {
		if skill.Enabled && skill.AppliesToOS(goos) {
			wanted[skill.Name] = true
		}
	}

	if uc.logger != nil {
		uc.logger.Info("Agent skills applicable to %s: %d of %d", goos, len(wanted), len(skills))
	}

	var results []SkillDeployResult

	for _, skill := range skills {
		if !skill.Enabled || !skill.AppliesToOS(goos) {
			continue
		}

		skillTargetDir := filepath.Join(targetBaseDir, skill.Name)
		if skill.TargetDir != "" {
			skillTargetDir = skill.TargetDir
		}
		skillSourceDir := filepath.ToSlash(filepath.Join("configs", "skills", skill.Name))

		filesCopied, copyErr := uc.fsManager.CopyEmbeddedTree(uc.embeddedFS, skillSourceDir, skillTargetDir)
		if copyErr != nil {
			if uc.logger != nil {
				uc.logger.Error("Failed to deploy skill '%s' to '%s': %v", skill.Name, skillTargetDir, copyErr)
			}
			results = append(results, SkillDeployResult{
				SkillName:    skill.Name,
				TargetDir:    skillTargetDir,
				Status:       entity.DiagError,
				ErrorMessage: copyErr.Error(),
			})
		} else {
			if uc.logger != nil {
				uc.logger.Info("Deployed skill '%s' (%d files) to '%s'", skill.Name, filesCopied, skillTargetDir)
			}
			results = append(results, SkillDeployResult{
				SkillName:   skill.Name,
				TargetDir:   skillTargetDir,
				FilesCopied: filesCopied,
				Status:      entity.DiagOK,
			})
		}
	}

	var pruned []string
	if base, expandErr := uc.fsManager.ExpandUserPath(targetBaseDir); expandErr == nil {
		pruned = pruneStaleSkills(base, wanted, uc.logger)
	} else if uc.logger != nil {
		uc.logger.Warn("Could not expand skills target '%s' for pruning: %v", targetBaseDir, expandErr)
	}

	return results, pruned, nil
}

// pruneStaleSkills removes directories directly under baseDir whose names are
// not in wanted, keeping the deployed skills mirroring the manifest. Stale
// directories are quarantined rather than destroyed, so a skill installed by
// hand or by another tool stays recoverable. It is a no-op when wanted is empty
// (safety against an empty/failed manifest) and skips dot-directories.
// Quarantined names are returned.
func pruneStaleSkills(baseDir string, wanted map[string]bool, logger repository.Logger) []string {
	if baseDir == "" || len(wanted) == 0 {
		return nil
	}

	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return nil
	}

	var quarantined []string
	for _, entry := range entries {
		name := entry.Name()
		if !entry.IsDir() || strings.HasPrefix(name, ".") || wanted[name] {
			continue
		}
		dest, err := quarantineSkill(baseDir, name)
		if err != nil {
			if logger != nil {
				logger.Warn("Failed to quarantine stale skill '%s': %v", name, err)
			}
			continue
		}
		quarantined = append(quarantined, name)
		if logger != nil {
			logger.Info("[SKILLS-PRUNE] quarantined stale skill '%s' to '%s'", name, dest)
		}
	}
	return quarantined
}

// quarantineSkill moves a stale skill directory out of the deployment tree.
// The trash tree is a sibling of the skills directory, never inside it, so
// quarantined skills are not rescanned, re-pruned, or counted by the doctor.
func quarantineSkill(baseDir, name string) (string, error) {
	dest := filepath.Join(filepath.Dir(baseDir), ".envctl-trash", "skills",
		fmt.Sprintf("%s-%s", name, time.Now().Format("20060102-150405")))
	if err := os.MkdirAll(filepath.Dir(dest), 0700); err != nil {
		return "", err
	}
	return dest, os.Rename(filepath.Join(baseDir, name), dest)
}
