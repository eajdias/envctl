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
}

func NewProvisionSkillsUseCase(
	manifestRepo repository.ManifestRepository,
	fsManager repository.FileSystemManager,
	embeddedFS fs.FS,
) *ProvisionSkillsUseCase {
	return &ProvisionSkillsUseCase{
		manifestRepo: manifestRepo,
		fsManager:    fsManager,
		embeddedFS:   embeddedFS,
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
		return nil, fmt.Errorf("failed to load skills manifest: %w", err)
	}

	var results []SkillDeployResult

	for _, skill := range skills {
		if !skill.Enabled {
			continue
		}

		skillTargetDir := filepath.Join(targetBaseDir, skill.Name)
		skillSourceDir := filepath.ToSlash(filepath.Join("configs", "skills", skill.Name))

		filesCopied, copyErr := uc.fsManager.CopyEmbeddedTree(uc.embeddedFS, skillSourceDir, skillTargetDir)
		if copyErr != nil {
			results = append(results, SkillDeployResult{
				SkillName:    skill.Name,
				TargetDir:    skillTargetDir,
				Status:       entity.DiagError,
				ErrorMessage: copyErr.Error(),
			})
		} else {
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
