package apt

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/eajdias/envctl/internal/domain/entity"
	"github.com/eajdias/envctl/internal/domain/repository"
)

type aptManager struct {
	aptPath  string
	dpkgPath string
}

// NewAptManager creates a PackageManager for Debian/Ubuntu apt.
func NewAptManager() repository.PackageManager {
	aPath := "apt-get"
	if p, err := exec.LookPath("apt-get"); err == nil {
		aPath = p
	}
	dPath := "dpkg-query"
	if p, err := exec.LookPath("dpkg-query"); err == nil {
		dPath = p
	}
	return &aptManager{
		aptPath:  aPath,
		dpkgPath: dPath,
	}
}

func (a *aptManager) Type() entity.PackageType {
	return entity.PackageTypeApt
}

func (a *aptManager) IsAvailable(ctx context.Context) bool {
	cmd := exec.CommandContext(ctx, a.aptPath, "--version")
	return cmd.Run() == nil
}

func (a *aptManager) IsInstalled(ctx context.Context, pkg entity.Package) (bool, string, error) {
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

	// Query package status via dpkg-query
	cmd := exec.CommandContext(ctx, a.dpkgPath, "-W", "-f=${Status}\t${Version}", pkg.ID)
	out, err := cmd.CombinedOutput()
	if err == nil {
		outStr := strings.TrimSpace(string(out))
		if strings.HasPrefix(outStr, "install ok installed") {
			parts := strings.Split(outStr, "\t")
			version := ""
			if len(parts) >= 2 {
				version = parts[1]
			}
			return true, version, nil
		}
	}
	return false, "", nil
}

func (a *aptManager) Install(ctx context.Context, pkg entity.Package) error {
	args := []string{"install", "-y", "--no-install-recommends"}
	if len(pkg.Args) > 0 {
		args = append(args, pkg.Args...)
	}
	args = append(args, pkg.ID)

	cmd := exec.CommandContext(ctx, a.aptPath, args...)
	cmd.Env = append(cmd.Environ(), "DEBIAN_FRONTEND=noninteractive")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("apt-get install %s failed: %s (%w)", pkg.ID, string(out), err)
	}
	return nil
}

func (a *aptManager) ListInstalled(ctx context.Context) ([]entity.Package, error) {
	cmd := exec.CommandContext(ctx, a.dpkgPath, "-W", "-f=${Package}\t${Version}\t${Status}\n")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(out), "\n")
	var pkgs []entity.Package
	for _, line := range lines {
		fields := strings.Split(line, "\t")
		if len(fields) >= 3 && fields[2] == "install ok installed" {
			pkgs = append(pkgs, entity.Package{
				ID:      fields[0],
				Name:    fields[0],
				Type:    entity.PackageTypeApt,
				Version: fields[1],
				Status:  entity.StatusInstalled,
			})
		}
	}
	return pkgs, nil
}
