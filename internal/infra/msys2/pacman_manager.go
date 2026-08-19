package msys2

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/eajdias/win11-new/internal/domain/entity"
	"github.com/eajdias/win11-new/internal/domain/repository"
)

type pacmanManager struct {
	pacmanPath string
}

// NewPacmanManager creates a PackageManager for MSYS2 pacman.
func NewPacmanManager() repository.PackageManager {
	// Look for pacman in PATH or default C:\msys64\usr\bin\pacman.exe
	pPath := "pacman"
	if _, err := exec.LookPath("pacman"); err != nil {
		defaultPath := `C:\msys64\usr\bin\pacman.exe`
		if _, err := os.Stat(defaultPath); err == nil {
			pPath = defaultPath
		}
	}
	return &pacmanManager{pacmanPath: pPath}
}

func (p *pacmanManager) Type() entity.PackageType {
	return entity.PackageTypePacman
}

func (p *pacmanManager) IsAvailable(ctx context.Context) bool {
	cmd := exec.CommandContext(ctx, p.pacmanPath, "-V")
	return cmd.Run() == nil
}

func (p *pacmanManager) IsInstalled(ctx context.Context, pkg entity.Package) (bool, string, error) {
	cmd := exec.CommandContext(ctx, p.pacmanPath, "-Q", pkg.ID)
	out, err := cmd.CombinedOutput()
	if err == nil {
		return true, strings.TrimSpace(string(out)), nil
	}
	return false, "", nil
}

func (p *pacmanManager) Install(ctx context.Context, pkg entity.Package) error {
	cmd := exec.CommandContext(ctx, p.pacmanPath, "-S", "--noconfirm", "--needed", pkg.ID)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("pacman install %s failed: %s (%w)", pkg.ID, string(out), err)
	}
	return nil
}

func (p *pacmanManager) ListInstalled(ctx context.Context) ([]entity.Package, error) {
	cmd := exec.CommandContext(ctx, p.pacmanPath, "-Qe")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(out), "\n")
	var pkgs []entity.Package
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 1 {
			version := ""
			if len(fields) >= 2 {
				version = fields[1]
			}
			pkgs = append(pkgs, entity.Package{
				ID:      fields[0],
				Name:    fields[0],
				Type:    entity.PackageTypePacman,
				Version: version,
				Status:  entity.StatusInstalled,
			})
		}
	}
	return pkgs, nil
}
