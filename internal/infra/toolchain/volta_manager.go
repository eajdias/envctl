package toolchain

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/eajdias/win11-new/internal/domain/entity"
	"github.com/eajdias/win11-new/internal/domain/repository"
)

type VoltaManager struct{}

// NewVoltaManager creates a new PackageManager for Volta JS toolchain.
func NewVoltaManager() repository.PackageManager {
	return &VoltaManager{}
}

func (v *VoltaManager) Type() entity.PackageType {
	return entity.PackageTypeVolta
}

func (v *VoltaManager) IsAvailable(ctx context.Context) bool {
	cmd := exec.CommandContext(ctx, "volta", "--version")
	return cmd.Run() == nil
}

func (v *VoltaManager) IsInstalled(ctx context.Context, pkg entity.Package) (bool, string, error) {
	// If custom check_command is specified, verify execution
	if pkg.CheckCommand != "" {
		parts := strings.Fields(pkg.CheckCommand)
		cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
		if out, err := cmd.CombinedOutput(); err == nil {
			return true, strings.TrimSpace(string(out)), nil
		}
	}

	// Inspect volta list
	cmd := exec.CommandContext(ctx, "volta", "list")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return false, "", err
	}

	cleanPkgID := strings.Split(pkg.ID, "@")[0] // strip version if specified (e.g. node@24 -> node)
	for _, line := range strings.Split(string(out), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(strings.ToLower(trimmed), strings.ToLower(cleanPkgID)) ||
			strings.Contains(strings.ToLower(trimmed), strings.ToLower(pkg.ID)) {
			return true, trimmed, nil
		}
	}

	return false, "", nil
}

func (v *VoltaManager) Install(ctx context.Context, pkg entity.Package) error {
	cmd := exec.CommandContext(ctx, "volta", "install", pkg.ID)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("volta install %s failed: %s (%w)", pkg.ID, string(out), err)
	}
	return nil
}

func (v *VoltaManager) ListInstalled(ctx context.Context) ([]entity.Package, error) {
	cmd := exec.CommandContext(ctx, "volta", "list")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	var pkgs []entity.Package
	for _, line := range strings.Split(string(out), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		fields := strings.Fields(trimmed)
		if len(fields) >= 2 {
			pkgID := fields[1]
			pkgs = append(pkgs, entity.Package{
				ID:      pkgID,
				Name:    pkgID,
				Type:    entity.PackageTypeVolta,
				Status:  entity.StatusInstalled,
				Version: trimmed,
			})
		}
	}

	return pkgs, nil
}
