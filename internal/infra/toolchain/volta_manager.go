package toolchain

import (
	"context"
	"fmt"
	"strings"

	"github.com/eajdias/envctl/internal/domain/entity"
	"github.com/eajdias/envctl/internal/domain/repository"
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
	cmd := execTool(ctx, "volta", "--version")
	return cmd.Run() == nil
}

func (v *VoltaManager) IsInstalled(ctx context.Context, pkg entity.Package) (bool, string, error) {
	// If custom check_command is specified, verify execution
	if pkg.CheckCommand != "" {
		parts := strings.Fields(pkg.CheckCommand)
		cmd := execTool(ctx, parts[0], parts[1:]...)
		if out, err := cmd.CombinedOutput(); err == nil {
			return true, strings.TrimSpace(string(out)), nil
		}
	}

	// Inspect volta list
	cmd := execTool(ctx, "volta", "list")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return false, "", err
	}

	found, info := voltaListContains(string(out), pkg.ID)
	return found, info, nil
}

// voltaListContains reports whether the `volta list` output references the
// given package ID. Version qualifiers are ignored, but scoped IDs keep
// their scope: "@playwright/cli" must not match "playwright" and, critically,
// must never match an empty string (strings.Split(id, "@")[0] is "" for
// scoped IDs, and a project-pinned `volta list` contains a standalone "@"
// token in "(current @ /path/package.json)" that would false-positive).
func voltaListContains(listOut, pkgID string) (bool, string) {
	cleanPkgID := strings.ToLower(pkgID)
	if i := strings.LastIndex(cleanPkgID, "@"); i > 0 {
		cleanPkgID = cleanPkgID[:i] // strip version (node@24 -> node); leading scope @ is kept
	}
	if cleanPkgID == "" || cleanPkgID == "@" {
		return false, ""
	}
	for _, line := range strings.Split(listOut, "\n") {
		trimmed := strings.TrimSpace(line)
		for _, token := range strings.Fields(trimmed) {
			t := strings.ToLower(token)
			// Match the exact package name (or a version-qualified token like "node@24")
			if t == cleanPkgID || strings.HasPrefix(t, cleanPkgID+"@") {
				return true, trimmed
			}
		}
	}

	return false, ""
}

func (v *VoltaManager) Install(ctx context.Context, pkg entity.Package) error {
	cmd := execTool(ctx, "volta", "install", pkg.ID)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("volta install %s failed: %s (%w)", pkg.ID, string(out), err)
	}
	return nil
}

func (v *VoltaManager) ListInstalled(ctx context.Context) ([]entity.Package, error) {
	cmd := execTool(ctx, "volta", "list")
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
