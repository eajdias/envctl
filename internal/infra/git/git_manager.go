package git

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/eajdias/envctl/internal/domain/entity"
)

type GitManager struct{}

// NewGitManager creates a new GitManager instance.
func NewGitManager() *GitManager {
	return &GitManager{}
}

func (g *GitManager) GetGlobalConfig(ctx context.Context, key string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "config", "--global", "--get", key)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func (g *GitManager) SetGlobalConfig(ctx context.Context, key, value string) error {
	cmd := exec.CommandContext(ctx, "git", "config", "--global", key, value)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to set git config %s=%s: %s (%w)", key, value, string(out), err)
	}
	return nil
}

func (g *GitManager) EnsureGlobalConfigs(ctx context.Context, configs []entity.GitConfig) ([]entity.Diagnostic, error) {
	var diagnostics []entity.Diagnostic

	for _, cfg := range configs {
		currentVal, err := g.GetGlobalConfig(ctx, cfg.Key)
		if err != nil || currentVal != cfg.Value {
			if err := g.SetGlobalConfig(ctx, cfg.Key, cfg.Value); err != nil {
				diagnostics = append(diagnostics, entity.Error(
					"Git",
					cfg.Key,
					fmt.Sprintf("Failed to set config %s: %v", cfg.Key, err),
				))
			} else {
				diagnostics = append(diagnostics, entity.OK(
					"Git",
					cfg.Key,
					fmt.Sprintf("Configured %s = %s", cfg.Key, cfg.Value),
				))
			}
		} else {
			diagnostics = append(diagnostics, entity.OK(
				"Git",
				cfg.Key,
				fmt.Sprintf("Already aligned: %s = %s", cfg.Key, cfg.Value),
			))
		}
	}

	return diagnostics, nil
}
