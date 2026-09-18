package toolchain

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/eajdias/envctl/internal/domain/entity"
	"github.com/eajdias/envctl/internal/domain/repository"
)

// execTool builds an exec.Cmd resolved against the Volta/user-local/Go
// toolchain PATH on Linux. envctl often runs from non-login shells (ssh,
// systemd) where those shims are absent from the default PATH — without this,
// every toolchain check would falsely report packages as missing.
func execTool(ctx context.Context, name string, args ...string) *exec.Cmd {
	if runtime.GOOS == "linux" {
		if home, err := os.UserHomeDir(); err == nil && home != "" {
			toolchainPath := strings.Join([]string{
				filepath.Join(home, ".volta", "bin"),
				filepath.Join(home, ".local", "bin"),
				"/usr/local/go/bin",
				filepath.Join(home, "go", "bin"),
				os.Getenv("PATH"),
			}, string(os.PathListSeparator))
			// exec.LookPath only consults the process PATH, so resolve the
			// binary explicitly against the toolchain PATH and hand the
			// absolute path to exec.Command.
			if resolved, err := lookPathWithEnv(name, toolchainPath); err == nil {
				cmd := exec.CommandContext(ctx, resolved, args...)
				env := os.Environ()
				for i, kv := range env {
					if strings.HasPrefix(kv, "PATH=") {
						env[i] = "PATH=" + toolchainPath
						break
					}
				}
				cmd.Env = env
				return cmd
			}
		}
	}
	return exec.CommandContext(ctx, name, args...)
}

// lookPathWithEnv searches for an executable in the given PATH string,
// honoring the Unix executable-bit convention.
func lookPathWithEnv(name, path string) (string, error) {
	if filepath.IsAbs(name) {
		return name, nil
	}
	for _, dir := range filepath.SplitList(path) {
		if dir == "" {
			dir = "."
		}
		candidate := filepath.Join(dir, name)
		if fi, err := os.Stat(candidate); err == nil && !fi.IsDir() {
			if fi.Mode()&0111 != 0 {
				return candidate, nil
			}
		}
	}
	return "", fmt.Errorf("executable %q not found in toolchain PATH", name)
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
	cmd := execTool(ctx, "npm", "-v")
	return cmd.Run() == nil
}

func (n *NpmManager) IsInstalled(ctx context.Context, pkg entity.Package) (bool, string, error) {
	if pkg.CheckCommand != "" {
		parts := strings.Fields(pkg.CheckCommand)
		cmd := execTool(ctx, parts[0], parts[1:]...)
		if out, err := cmd.CombinedOutput(); err == nil {
			return true, strings.TrimSpace(string(out)), nil
		}
	}
	cmd := execTool(ctx, "npm", "list", "-g", "--depth=0", pkg.ID)
	out, err := cmd.CombinedOutput()
	if err == nil && strings.Contains(string(out), pkg.ID+"@") {
		return true, "installed globally via npm", nil
	}
	return false, "", nil
}

func (n *NpmManager) Install(ctx context.Context, pkg entity.Package) error {
	args := []string{"install", "-g"}
	// Pin the global prefix to ~/.local: distro npm packages (Arch, Debian)
	// resolve the global prefix to a root-owned system directory, so an
	// unpinned `npm install -g` fails or needs sudo.
	if prefix, err := userLocalPrefix(); err == nil {
		args = append(args, "--prefix", prefix)
	}
	args = append(args, strings.Fields(pkg.ID)...)
	cmd := execTool(ctx, "npm", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("npm install -g %s failed: %s (%w)", pkg.ID, string(out), err)
	}
	return nil
}

// userLocalPrefix returns ~/.local, creating it when missing, so global
// toolchain installs land in a user-writable prefix instead of a root-owned
// system directory.
func userLocalPrefix() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "", fmt.Errorf("cannot resolve user home directory")
	}
	prefix := filepath.Join(home, ".local")
	if err := os.MkdirAll(prefix, 0o750); err != nil {
		return "", err
	}
	return prefix, nil
}

func (n *NpmManager) ListInstalled(ctx context.Context) ([]entity.Package, error) {
	cmd := execTool(ctx, "npm", "list", "-g", "--depth=0", "--json")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}
	var parsed struct {
		Dependencies map[string]struct {
			Version string `json:"version"`
		} `json:"dependencies"`
	}
	if err := json.Unmarshal(out, &parsed); err != nil {
		return nil, err
	}
	var pkgs []entity.Package
	for name, dep := range parsed.Dependencies {
		pkgs = append(pkgs, entity.Package{
			ID:      name,
			Name:    name,
			Version: dep.Version,
			Type:    entity.PackageTypeNpm,
			Status:  entity.StatusInstalled,
		})
	}
	return pkgs, nil
}

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
	cmd := exec.CommandContext(ctx, pipPythonBin(), "-m", "pip", "--version")
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
	if execTool(ctx, "uv", "--version").Run() == nil {
		cmd := execTool(ctx, "uv", "tool", "install", "--upgrade", pkg.ID)
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

// GoManager handles Go tooling via `go install`.
type GoManager struct{}

func NewGoManager() repository.PackageManager {
	return &GoManager{}
}

func (g *GoManager) Type() entity.PackageType {
	return entity.PackageTypeGo
}

func (g *GoManager) IsAvailable(ctx context.Context) bool {
	cmd := execTool(ctx, "go", "version")
	return cmd.Run() == nil
}

func (g *GoManager) IsInstalled(ctx context.Context, pkg entity.Package) (bool, string, error) {
	if pkg.CheckCommand != "" {
		parts := strings.Fields(pkg.CheckCommand)
		cmd := execTool(ctx, parts[0], parts[1:]...)
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
	if runtime.GOOS == "windows" {
		cmd := execTool(ctx, "where.exe", binName)
		if out, err := cmd.CombinedOutput(); err == nil {
			return true, strings.TrimSpace(string(out)), nil
		}
	} else {
		// Resolve against the toolchain PATH (covers ~/go/bin, /usr/local/go/bin).
		cmd := execTool(ctx, "bash", "-lc", "command -v "+binName+" >/dev/null 2>&1")
		if cmd.Run() == nil {
			return true, "in toolchain PATH", nil
		}
	}
	return false, "", nil
}

func (g *GoManager) Install(ctx context.Context, pkg entity.Package) error {
	cmd := execTool(ctx, "go", "install", pkg.ID)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("go install %s failed: %s (%w)", pkg.ID, string(out), err)
	}
	return nil
}

func (g *GoManager) ListInstalled(ctx context.Context) ([]entity.Package, error) {
	return []entity.Package{{ID: "go-tools", Name: "go-tools", Status: entity.StatusInstalled}}, nil
}
