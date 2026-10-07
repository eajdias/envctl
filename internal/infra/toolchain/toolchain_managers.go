package toolchain

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/eajdias/envctl/internal/domain/entity"
	"github.com/eajdias/envctl/internal/domain/repository"
	"github.com/eajdias/envctl/internal/infra/executil"
)

// pipPythonBin returns the correct python binary name for the current OS.
func pipPythonBin() string {
	if runtime.GOOS == "linux" {
		return "python3"
	}
	return "python"
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
	// uv is the preferred installer (isolated tool envs, PEP 668-safe) and it
	// needs no system pip — which Arch does not ship by default, so a
	// pip-only check would silently skip every Python tool there.
	if executil.ExecTool(ctx, "uv", "--version").Run() == nil {
		return true
	}
	// #nosec G204 -- fixed binary probe (python -m pip --version), no user input.
	cmd := exec.CommandContext(ctx, pipPythonBin(), "-m", "pip", "--version")
	return cmd.Run() == nil
}

func (p *PipManager) IsInstalled(ctx context.Context, pkg entity.Package) (bool, string, error) {
	if pkg.CheckCommand != "" {
		if out, ok := executil.ProbeCheckCommand(ctx, pkg.CheckCommand); ok {
			return true, out, nil
		}
	}
	// #nosec G204 -- pkg.ID comes from the embedded manifest, never from user input.
	cmd := exec.CommandContext(ctx, pipPythonBin(), "-m", "pip", "show", pkg.ID)
	out, err := cmd.CombinedOutput()
	if err == nil && strings.Contains(string(out), "Name: "+pkg.ID) {
		return true, "installed via pip", nil
	}
	return false, "", nil
}

func (p *PipManager) Install(ctx context.Context, pkg entity.Package) error {
	// uv installs into an isolated tool environment, which is the supported
	// path on PEP 668 "externally managed" hosts (Arch/CachyOS, Ubuntu 24.04+).
	// Prefer it and fall back to pip where uv is unavailable.
	if executil.ExecTool(ctx, "uv", "--version").Run() == nil {
		cmd := executil.ExecTool(ctx, "uv", "tool", "install", "--upgrade", pkg.ID)
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("uv tool install %s failed: %s (%w)", pkg.ID, string(out), err)
		}
		return nil
	}
	args := []string{"-m", "pip", "install", "--upgrade"}
	if pythonExternallyManaged(ctx) {
		args = append(args, "--break-system-packages")
	}
	args = append(args, pkg.ID)
	// #nosec G204 -- argv elements handed to pip directly (no shell); the id
	// comes from the embedded manifest, never from user input.
	cmd := exec.CommandContext(ctx, pipPythonBin(), args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("pip install %s failed: %s (%w)", pkg.ID, string(out), err)
	}
	return nil
}

// pythonExternallyManaged reports whether the system Python declares itself
// externally managed (PEP 668), in which case pip refuses installs outside a
// virtualenv unless --break-system-packages is passed.
func pythonExternallyManaged(ctx context.Context) bool {
	script := `import os, sysconfig; print(os.path.exists(os.path.join(sysconfig.get_paths()["stdlib"], "EXTERNALLY-MANAGED")))`
	// #nosec G204 -- fixed inline script, no interpolation and no shell.
	out, err := exec.CommandContext(ctx, pipPythonBin(), "-c", script).Output()
	return err == nil && strings.TrimSpace(string(out)) == "True"
}

func (p *PipManager) ListInstalled(ctx context.Context) ([]entity.Package, error) {
	// #nosec G204 -- fixed binary invocation (python -m pip list), no user input.
	cmd := exec.CommandContext(ctx, pipPythonBin(), "-m", "pip", "list", "--format=freeze")
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
