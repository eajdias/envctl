package usecase

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"

	"github.com/eajdias/win11-new/internal/domain/entity"
	"github.com/eajdias/win11-new/internal/domain/repository"
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

func (uc *ProvisionSkillsUseCase) Execute(ctx context.Context, targetBaseDir string) ([]SkillDeployResult, error) {
	if targetBaseDir == "" {
		targetBaseDir = "~/.config/opencode/skills"
	}

	skills, err := uc.manifestRepo.LoadSkills()
	if err != nil {
		if uc.logger != nil {
			uc.logger.Error("Failed to load skills manifest: %v", err)
		}
		return nil, fmt.Errorf("failed to load skills manifest: %w", err)
	}

	if uc.logger != nil {
		uc.logger.Info("Starting agent skills provisioning (Total: %d skills, Target: '%s')", len(skills), targetBaseDir)
	}

	var results []SkillDeployResult

	for _, skill := range skills {
		if !skill.Enabled {
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

	return results, nil
}
