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

// PipManager installs Python packages exclusively through uv: tools go to
// isolated tool envs (`uv tool install`), importable libraries go to the
// system environment (`uv pip install --system`, refused by uv itself on
// PEP 668 externally-managed hosts instead of overridden). There is no pip
// install path: raw pip can neither isolate nor fail closed.
type PipManager struct{}

func NewPipManager() repository.PackageManager {
	return &PipManager{}
}

func (p *PipManager) Type() entity.PackageType {
	return entity.PackageTypePip
}

func (p *PipManager) IsAvailable(ctx context.Context) bool {
	// uv-only: without it there is no PEP 668-safe install path, so report
	// unavailable (fail closed) instead of falling back to raw pip.
	return executil.ExecTool(ctx, "uv", "--version").Run() == nil
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
	// Libraries (manifest check_command `py -m pip show <id>`) have no binary
	// to isolate: install into the system environment, which uv refuses on
	// externally-managed hosts instead of overriding. Tools go to an
	// isolated `uv tool` env.
	args := []string{"tool", "install", "--upgrade", pkg.ID}
	if isPipLibrary(pkg) {
		args = []string{"pip", "install", "--system", "--upgrade", pkg.ID}
	}
	cmd := executil.ExecTool(ctx, "uv", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("uv %s failed: %s (%w)", strings.Join(args, " "), string(out), err)
	}
	return nil
}

// isPipLibrary reports whether a manifest entry is an importable library
// rather than a CLI tool: libraries are probed with `py -m pip show`.
func isPipLibrary(pkg entity.Package) bool {
	return strings.Contains(pkg.CheckCommand, "pip show")
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
