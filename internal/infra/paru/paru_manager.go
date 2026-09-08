package paru

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/eajdias/envctl/internal/domain/entity"
	"github.com/eajdias/envctl/internal/domain/repository"
)

type paruManager struct {
	paruPath   string
	pacmanPath string
}

// NewParuManager creates a PackageManager for Arch User Repository via paru.
// Installed-state checks use the shared pacman database; installs delegate
// to paru, which handles sudo itself (requires passwordless sudo headless).
func NewParuManager() repository.PackageManager {
	paruPath := "paru"
	if p, err := exec.LookPath("paru"); err == nil {
		paruPath = p
	}
	pacmanPath := "pacman"
	if p, err := exec.LookPath("pacman"); err == nil {
		pacmanPath = p
	}
	return &paruManager{
		paruPath:   paruPath,
		pacmanPath: pacmanPath,
	}
}

func (m *paruManager) Type() entity.PackageType {
	return entity.PackageTypeParu
}

func (m *paruManager) IsAvailable(ctx context.Context) bool {
	cmd := exec.CommandContext(ctx, m.paruPath, "--version")
	return cmd.Run() == nil
}

func (m *paruManager) IsInstalled(ctx context.Context, pkg entity.Package) (bool, string, error) {
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

	// AUR and repo packages share the pacman database
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

func (m *paruManager) Install(ctx context.Context, pkg entity.Package) error {
	args := []string{"-S", "--noconfirm", "--needed", "--skipreview"}
	if len(pkg.Args) > 0 {
		args = append(args, pkg.Args...)
	}
	args = append(args, pkg.ID)

	// paru handles privilege escalation itself; it needs passwordless
	// sudo (or root) to run non-interactively.
	cmd := exec.CommandContext(ctx, m.paruPath, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("paru -S %s failed: %s (%w)", pkg.ID, string(out), err)
	}
	return nil
}

func (m *paruManager) ListInstalled(ctx context.Context) ([]entity.Package, error) {
	// Same database as pacman; tag results with the paru type.
	cmd := exec.CommandContext(ctx, m.pacmanPath, "-Qm")
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
				Type:    entity.PackageTypeParu,
				Version: fields[1],
				Status:  entity.StatusInstalled,
			})
		}
	}
	return pkgs, nil
}
