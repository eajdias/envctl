package winget

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/eajdias/envctl/internal/domain/entity"
	"github.com/eajdias/envctl/internal/domain/repository"
)

type wingetManager struct{}

// NewWingetManager creates a new PackageManager for Windows Package Manager (winget).
func NewWingetManager() repository.PackageManager {
	return &wingetManager{}
}

func (w *wingetManager) Type() entity.PackageType {
	return entity.PackageTypeWinget
}

func (w *wingetManager) IsAvailable(ctx context.Context) bool {
	_, err := exec.LookPath("winget.exe")
	if err != nil {
		_, err = exec.LookPath("winget")
	}
	return err == nil
}

func (w *wingetManager) IsInstalled(ctx context.Context, pkg entity.Package) (bool, string, error) {
	// If custom check_command is specified, use bash or cmd
	if pkg.CheckCommand != "" {
		parts := strings.Fields(pkg.CheckCommand)
		cmdName := parts[0]
		cmdArgs := parts[1:]
		cmd := exec.CommandContext(ctx, cmdName, cmdArgs...)
		out, err := cmd.CombinedOutput()
		if err == nil {
			return true, strings.TrimSpace(string(out)), nil
		}
	}

	// Fallback to winget list
	cmd := exec.CommandContext(ctx, "winget", "list", "--id", pkg.ID, "--exact", "--accept-source-agreements")
	out, err := cmd.CombinedOutput()
	if err == nil && strings.Contains(string(out), pkg.ID) {
		return true, "installed via winget", nil
	}

	return false, "", nil
}

func (w *wingetManager) Install(ctx context.Context, pkg entity.Package) error {
	args := []string{
		"install",
		"--id", pkg.ID,
		"--exact",
		"--accept-package-agreements",
		"--accept-source-agreements",
		"--silent",
	}

	if len(pkg.Args) > 0 {
		args = append(args, pkg.Args...)
	}

	cmd := exec.CommandContext(ctx, "winget", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check if it failed because it is already installed
		if strings.Contains(string(output), "already installed") || strings.Contains(string(output), "No newer package versions") {
			return nil
		}
		return fmt.Errorf("winget install %s failed: %s (%w)", pkg.ID, string(output), err)
	}

	return nil
}

func (w *wingetManager) ListInstalled(ctx context.Context) ([]entity.Package, error) {
	cmd := exec.CommandContext(ctx, "winget", "list", "--accept-source-agreements")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(out), "\n")
	var pkgs []entity.Package
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			pkgs = append(pkgs, entity.Package{
				ID:     fields[1],
				Name:   fields[0],
				Type:   entity.PackageTypeWinget,
				Status: entity.StatusInstalled,
			})
		}
	}
	return pkgs, nil
}
