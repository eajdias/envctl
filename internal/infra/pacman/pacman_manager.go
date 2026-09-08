package pacman

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/eajdias/envctl/internal/domain/entity"
	"github.com/eajdias/envctl/internal/domain/repository"
)

type pacmanManager struct {
	pacmanPath string
}

// NewPacmanManager creates a PackageManager for Arch Linux pacman.
func NewPacmanManager() repository.PackageManager {
	pPath := "pacman"
	if p, err := exec.LookPath("pacman"); err == nil {
		pPath = p
	}
	return &pacmanManager{
		pacmanPath: pPath,
	}
}

func (m *pacmanManager) Type() entity.PackageType {
	return entity.PackageTypePacman
}

func (m *pacmanManager) IsAvailable(ctx context.Context) bool {
	cmd := exec.CommandContext(ctx, m.pacmanPath, "--version")
	return cmd.Run() == nil
}

func (m *pacmanManager) IsInstalled(ctx context.Context, pkg entity.Package) (bool, string, error) {
	// If custom check command is provided, try that first
	if pkg.CheckCommand != "" {
		parts := strings.Fields(pkg.CheckCommand)
		if len(parts) > 0 {
			cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
			out, err := cmd.CombinedOutput()
			if err == nil {
				return true, strings.TrimSpace(string(out)), nil
			}
		}
	}

	// Query package status via pacman -Q (exit 0 means installed)
	cmd := exec.CommandContext(ctx, m.pacmanPath, "-Q", pkg.ID)
	out, err := cmd.CombinedOutput()
	if err == nil {
		parts := strings.Fields(strings.TrimSpace(string(out)))
		version := ""
		if len(parts) >= 2 {
			version = parts[1]
		}
		return true, version, nil
	}
	return false, "", nil
}

func (m *pacmanManager) Install(ctx context.Context, pkg entity.Package) error {
	args := []string{"-S", "--noconfirm", "--needed"}
	if len(pkg.Args) > 0 {
		args = append(args, pkg.Args...)
	}
	args = append(args, pkg.ID)

	// Elevated privileges are required when running as a non-root user.
	var cmd *exec.Cmd
	if isNonRoot(ctx) {
		cmd = exec.CommandContext(ctx, "sudo", append([]string{"-n", m.pacmanPath}, args...)...)
	} else {
		cmd = exec.CommandContext(ctx, m.pacmanPath, args...)
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("pacman -S %s failed: %s (%w)", pkg.ID, string(out), err)
	}
	return nil
}

// isNonRoot reports whether the current process runs as a non-root user.
// It uses `id -u` so it is safe on Linux; on other platforms it returns false.
func isNonRoot(ctx context.Context) bool {
	out, err := exec.CommandContext(ctx, "id", "-u").Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) != "0"
}

func (m *pacmanManager) ListInstalled(ctx context.Context) ([]entity.Package, error) {
	cmd := exec.CommandContext(ctx, m.pacmanPath, "-Q")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(out), "\n")
	var pkgs []entity.Package
	for _, line := range lines {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) >= 2 {
			pkgs = append(pkgs, entity.Package{
				ID:      fields[0],
				Name:    fields[0],
				Type:    entity.PackageTypePacman,
				Version: fields[1],
				Status:  entity.StatusInstalled,
			})
		}
	}
	return pkgs, nil
}
