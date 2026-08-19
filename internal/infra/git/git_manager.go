package git

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/eajdias/envctl/internal/domain/entity"
	"github.com/eajdias/envctl/internal/domain/repository"
)

type gitManager struct{}

// NewGitManager creates a new GitManager instance.
func NewGitManager() repository.GitManager {
	return &gitManager{}
}

func (g *gitManager) GetGlobalConfig(ctx context.Context, key string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "config", "--global", "--get", key)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func (g *gitManager) SetGlobalConfig(ctx context.Context, key, value string) error {
	cmd := exec.CommandContext(ctx, "git", "config", "--global", key, value)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to set git config %s=%s: %s (%w)", key, value, string(out), err)
	}
	return nil
}

func (g *gitManager) EnsureGlobalConfigs(ctx context.Context, configs []entity.GitConfig) ([]entity.Diagnostic, error) {
	var diagnostics []entity.Diagnostic

	for _, cfg := range configs {
		currentVal, err := g.GetGlobalConfig(ctx, cfg.Key)
		if err != nil || currentVal != cfg.Value {
			if err := g.SetGlobalConfig(ctx, cfg.Key, cfg.Value); err != nil {
				diagnostics = append(diagnostics, entity.Diagnostic{
					Category: entity.DiagError,
					System:   "Git",
					Target:   cfg.Key,
					Details:  fmt.Sprintf("Failed to set config %s: %v", cfg.Key, err),
				})
			} else {
				diagnostics = append(diagnostics, entity.Diagnostic{
					Category: entity.DiagOK,
					System:   "Git",
					Target:   cfg.Key,
					Details:  fmt.Sprintf("Configured %s = %s", cfg.Key, cfg.Value),
				})
			}
		} else {
			diagnostics = append(diagnostics, entity.Diagnostic{
				Category: entity.DiagOK,
				System:   "Git",
				Target:   cfg.Key,
				Details:  fmt.Sprintf("Already aligned: %s = %s", cfg.Key, cfg.Value),
			})
		}
	}

	return diagnostics, nil
}

func (g *gitManager) CreateSnapshotBranchAndPR(ctx context.Context, branchName, title, body string, changedFiles []string) (string, error) {
	// Create & switch to branch
	cmdBranch := exec.CommandContext(ctx, "git", "checkout", "-b", branchName)
	if out, err := cmdBranch.CombinedOutput(); err != nil {
		return "", fmt.Errorf("failed to create branch %s: %s (%w)", branchName, string(out), err)
	}

	// Stage files
	args := append([]string{"add"}, changedFiles...)
	cmdAdd := exec.CommandContext(ctx, "git", args...)
	if out, err := cmdAdd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("failed to stage files: %s (%w)", string(out), err)
	}

	// Commit
	cmdCommit := exec.CommandContext(ctx, "git", "commit", "-m", title)
	if out, err := cmdCommit.CombinedOutput(); err != nil {
		return "", fmt.Errorf("failed to commit changes: %s (%w)", string(out), err)
	}

	// Push
	cmdPush := exec.CommandContext(ctx, "git", "push", "-u", "origin", branchName)
	if out, err := cmdPush.CombinedOutput(); err != nil {
		return "", fmt.Errorf("failed to push branch %s: %s (%w)", branchName, string(out), err)
	}

	// Create PR via gh CLI
	cmdPR := exec.CommandContext(ctx, "gh", "pr", "create", "--title", title, "--body", body)
	outPR, err := cmdPR.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to create PR via gh CLI: %s (%w)", string(outPR), err)
	}

	return strings.TrimSpace(string(outPR)), nil
}
