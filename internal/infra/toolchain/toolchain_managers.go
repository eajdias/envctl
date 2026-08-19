package toolchain

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/eajdias/win11-new/internal/domain/entity"
	"github.com/eajdias/win11-new/internal/domain/repository"
)

// DotnetToolManager handles .NET global tools.
type DotnetToolManager struct{}

func NewDotnetToolManager() repository.PackageManager {
	return &DotnetToolManager{}
}

func (d *DotnetToolManager) Type() entity.PackageType {
	return entity.PackageTypeDotnetTool
}

func (d *DotnetToolManager) IsAvailable(ctx context.Context) bool {
	cmd := exec.CommandContext(ctx, "dotnet", "--version")
	return cmd.Run() == nil
}

func (d *DotnetToolManager) IsInstalled(ctx context.Context, pkg entity.Package) (bool, string, error) {
	cmd := exec.CommandContext(ctx, "dotnet", "tool", "list", "-g")
	out, err := cmd.CombinedOutput()
	if err == nil && strings.Contains(strings.ToLower(string(out)), strings.ToLower(pkg.ID)) {
		return true, "installed globally via dotnet tool", nil
	}
	return false, "", nil
}

func (d *DotnetToolManager) Install(ctx context.Context, pkg entity.Package) error {
	cmd := exec.CommandContext(ctx, "dotnet", "tool", "install", "-g", pkg.ID)
	out, err := cmd.CombinedOutput()
	if err != nil {
		if strings.Contains(string(out), "is already installed") {
			return nil
		}
		return fmt.Errorf("dotnet tool install %s failed: %s (%w)", pkg.ID, string(out), err)
	}
	return nil
}

func (d *DotnetToolManager) ListInstalled(ctx context.Context) ([]entity.Package, error) {
	cmd := exec.CommandContext(ctx, "dotnet", "tool", "list", "-g")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(out), "\n")
	var pkgs []entity.Package
	for i, line := range lines {
		if i < 2 || strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			pkgs = append(pkgs, entity.Package{
				ID:      fields[0],
				Name:    fields[0],
				Version: fields[1],
				Type:    entity.PackageTypeDotnetTool,
				Status:  entity.StatusInstalled,
			})
		}
	}
	return pkgs, nil
}

// NpmManager handles global npm packages.
type NpmManager struct{}

func NewNpmManager() repository.PackageManager {
	return &NpmManager{}
}

func (n *NpmManager) Type() entity.PackageType {
	return entity.PackageTypeNpm
}

func (n *NpmManager) IsAvailable(ctx context.Context) bool {
	cmd := exec.CommandContext(ctx, "npm", "-v")
	return cmd.Run() == nil
}

func (n *NpmManager) IsInstalled(ctx context.Context, pkg entity.Package) (bool, string, error) {
	if pkg.CheckCommand != "" {
		parts := strings.Fields(pkg.CheckCommand)
		cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
		if out, err := cmd.CombinedOutput(); err == nil {
			return true, strings.TrimSpace(string(out)), nil
		}
	}
	cmd := exec.CommandContext(ctx, "npm", "list", "-g", "--depth=0", pkg.ID)
	out, err := cmd.CombinedOutput()
	if err == nil && strings.Contains(string(out), pkg.ID+"@") {
		return true, "installed globally via npm", nil
	}
	return false, "", nil
}

func (n *NpmManager) Install(ctx context.Context, pkg entity.Package) error {
	args := []string{"install", "-g"}
	args = append(args, strings.Fields(pkg.ID)...)
	cmd := exec.CommandContext(ctx, "npm", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("npm install -g %s failed: %s (%w)", pkg.ID, string(out), err)
	}
	return nil
}

func (n *NpmManager) ListInstalled(ctx context.Context) ([]entity.Package, error) {
	cmd := exec.CommandContext(ctx, "npm", "list", "-g", "--depth=0", "--json")
	_, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}
	return []entity.Package{{ID: "npm-packages", Name: "npm-packages", Status: entity.StatusInstalled}}, nil
}

// PipManager handles global/user python packages.
type PipManager struct{}

func NewPipManager() repository.PackageManager {
	return &PipManager{}
}

func (p *PipManager) Type() entity.PackageType {
	return entity.PackageTypePip
}

func (p *PipManager) IsAvailable(ctx context.Context) bool {
	cmd := exec.CommandContext(ctx, "python", "-m", "pip", "--version")
	return cmd.Run() == nil
}

func (p *PipManager) IsInstalled(ctx context.Context, pkg entity.Package) (bool, string, error) {
	if pkg.CheckCommand != "" {
		parts := strings.Fields(pkg.CheckCommand)
		cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
		if out, err := cmd.CombinedOutput(); err == nil {
			return true, strings.TrimSpace(string(out)), nil
		}
	}
	cmd := exec.CommandContext(ctx, "python", "-m", "pip", "show", pkg.ID)
	out, err := cmd.CombinedOutput()
	if err == nil && strings.Contains(string(out), "Name: "+pkg.ID) {
		return true, "installed via pip", nil
	}
	return false, "", nil
}

func (p *PipManager) Install(ctx context.Context, pkg entity.Package) error {
	cmd := exec.CommandContext(ctx, "python", "-m", "pip", "install", "--upgrade", pkg.ID)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("pip install %s failed: %s (%w)", pkg.ID, string(out), err)
	}
	return nil
}

func (p *PipManager) ListInstalled(ctx context.Context) ([]entity.Package, error) {
	cmd := exec.CommandContext(ctx, "python", "-m", "pip", "list", "--format=freeze")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(out), "\n")
	var pkgs []entity.Package
	for _, line := range lines {
		parts := strings.Split(line, "==")
		if len(parts) == 2 {
			pkgs = append(pkgs, entity.Package{
				ID:      parts[0],
				Name:    parts[0],
				Version: parts[1],
				Type:    entity.PackageTypePip,
				Status:  entity.StatusInstalled,
			})
		}
	}
	return pkgs, nil
}

// GoManager handles Go tooling via `go install`.
type GoManager struct{}

func NewGoManager() repository.PackageManager {
	return &GoManager{}
}

func (g *GoManager) Type() entity.PackageType {
	return entity.PackageTypeGo
}

func (g *GoManager) IsAvailable(ctx context.Context) bool {
	cmd := exec.CommandContext(ctx, "go", "version")
	return cmd.Run() == nil
}

func (g *GoManager) IsInstalled(ctx context.Context, pkg entity.Package) (bool, string, error) {
	if pkg.CheckCommand != "" {
		parts := strings.Fields(pkg.CheckCommand)
		cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
		if out, err := cmd.CombinedOutput(); err == nil {
			return true, strings.TrimSpace(string(out)), nil
		}
	}
	binName := pkg.Name
	if binName == "" {
		parts := strings.Split(pkg.ID, "/")
		last := parts[len(parts)-1]
		binName = strings.Split(last, "@")[0]
	}
	cmd := exec.CommandContext(ctx, "where.exe", binName)
	if out, err := cmd.CombinedOutput(); err == nil {
		return true, strings.TrimSpace(string(out)), nil
	}
	return false, "", nil
}

func (g *GoManager) Install(ctx context.Context, pkg entity.Package) error {
	cmd := exec.CommandContext(ctx, "go", "install", pkg.ID)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("go install %s failed: %s (%w)", pkg.ID, string(out), err)
	}
	return nil
}

func (g *GoManager) ListInstalled(ctx context.Context) ([]entity.Package, error) {
	return []entity.Package{{ID: "go-tools", Name: "go-tools", Status: entity.StatusInstalled}}, nil
}

// RustupManager handles Rust toolchains and components via `rustup`.
type RustupManager struct{}

func NewRustupManager() repository.PackageManager {
	return &RustupManager{}
}

func (r *RustupManager) Type() entity.PackageType {
	return entity.PackageTypeRustup
}

func (r *RustupManager) IsAvailable(ctx context.Context) bool {
	cmd := exec.CommandContext(ctx, "rustup", "--version")
	return cmd.Run() == nil
}

func (r *RustupManager) IsInstalled(ctx context.Context, pkg entity.Package) (bool, string, error) {
	if pkg.CheckCommand != "" {
		parts := strings.Fields(pkg.CheckCommand)
		cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
		if out, err := cmd.CombinedOutput(); err == nil {
			return true, strings.TrimSpace(string(out)), nil
		}
	}
	cmd := exec.CommandContext(ctx, "rustup", "component", "list", "--installed")
	out, err := cmd.CombinedOutput()
	if err == nil && strings.Contains(string(out), pkg.ID) {
		return true, "installed via rustup component", nil
	}
	return false, "", nil
}

func (r *RustupManager) Install(ctx context.Context, pkg entity.Package) error {
	cmd := exec.CommandContext(ctx, "rustup", "component", "add", pkg.ID)
	out, err := cmd.CombinedOutput()
	if err != nil {
		// Try toolchain install if component add failed
		cmdFallback := exec.CommandContext(ctx, "rustup", "toolchain", "install", pkg.ID)
		outFallback, errFallback := cmdFallback.CombinedOutput()
		if errFallback != nil {
			return fmt.Errorf("rustup add/install %s failed: %s | fallback: %s (%w)", pkg.ID, string(out), string(outFallback), err)
		}
	}
	return nil
}

func (r *RustupManager) ListInstalled(ctx context.Context) ([]entity.Package, error) {
	cmd := exec.CommandContext(ctx, "rustup", "component", "list", "--installed")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(out), "\n")
	var pkgs []entity.Package
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			pkgs = append(pkgs, entity.Package{
				ID:     trimmed,
				Name:   trimmed,
				Type:   entity.PackageTypeRustup,
				Status: entity.StatusInstalled,
			})
		}
	}
	return pkgs, nil
}
