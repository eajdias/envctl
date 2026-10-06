// Package arch implements the Arch Linux package managers (pacman and paru)
// as one parameterized manager. The two differ only in data — install
// binary, review skip, privilege escalation, list query and result tags —
// never in control flow, so a single struct with two constructors replaces
// two near-identical files.
package arch

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/eajdias/envctl/internal/domain/entity"
	"github.com/eajdias/envctl/internal/domain/repository"
	"github.com/eajdias/envctl/internal/infra/executil"
)

type archManager struct {
	name       string // "pacman" or "paru", for error messages
	bin        string // install binary path
	queryBin   string // pacman path: AUR and repo packages share its database
	pkgType    entity.PackageType
	listArgs   []string // pacman: -Q (all); paru: -Qm (foreign only)
	listType   entity.PackageType
	skipReview bool
	useSudo    bool
}

func resolveBin(name string) string {
	if p, err := exec.LookPath(name); err == nil {
		return p
	}
	return name
}

// NewPacmanManager creates a PackageManager for Arch Linux pacman.
func NewPacmanManager() repository.PackageManager {
	return &archManager{
		name:       "pacman",
		bin:        resolveBin("pacman"),
		queryBin:   resolveBin("pacman"),
		pkgType:    entity.PackageTypePacman,
		listArgs:   []string{"-Q"},
		listType:   entity.PackageTypePacman,
		skipReview: false,
		useSudo:    true,
	}
}

// NewParuManager creates a PackageManager for Arch User Repository via paru.
// Installed-state checks use the shared pacman database; installs delegate
// to paru, which handles sudo itself (requires passwordless sudo headless).
func NewParuManager() repository.PackageManager {
	return &archManager{
		name:       "paru",
		bin:        resolveBin("paru"),
		queryBin:   resolveBin("pacman"),
		pkgType:    entity.PackageTypeParu,
		listArgs:   []string{"-Qm"},
		listType:   entity.PackageTypeParu,
		skipReview: true,
		useSudo:    false,
	}
}

func (m *archManager) Type() entity.PackageType {
	return m.pkgType
}

func (m *archManager) IsAvailable(ctx context.Context) bool {
	//nolint:gosec // G204: fixed binary --version probe, no user input.
	cmd := exec.CommandContext(ctx, m.bin, "--version")
	return cmd.Run() == nil
}

func (m *archManager) IsInstalled(ctx context.Context, pkg entity.Package) (bool, string, error) {
	if pkg.CheckCommand != "" {
		if out, ok := executil.ProbeCheckCommand(ctx, pkg.CheckCommand); ok {
			return true, out, nil
		}
	}

	// Query package status via pacman -Q (exit 0 means installed)
	//nolint:gosec // G204: pkg.ID comes from the embedded manifest, never from user input.
	cmd := exec.CommandContext(ctx, m.queryBin, "-Q", pkg.ID)
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

func (m *archManager) Install(ctx context.Context, pkg entity.Package) error {
	args := []string{"-S", "--noconfirm", "--needed"}
	if m.skipReview {
		args = append(args, "--skipreview")
	}
	if len(pkg.Args) > 0 {
		args = append(args, pkg.Args...)
	}
	args = append(args, pkg.ID)

	// Elevated privileges are required when running as a non-root user.
	// paru handles escalation itself; pacman needs an explicit sudo -n.
	var cmd *exec.Cmd
	if m.useSudo && executil.IsNonRoot() {
		//nolint:gosec // G204: pkg.ID/args come from the embedded manifest, never from user input.
		cmd = exec.CommandContext(ctx, "sudo", append([]string{"-n", m.bin}, args...)...)
	} else {
		//nolint:gosec // G204: pkg.ID/args come from the embedded manifest, never from user input.
		cmd = exec.CommandContext(ctx, m.bin, args...)
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s -S %s failed: %s (%w)", m.name, pkg.ID, string(out), err)
	}
	return nil
}

func (m *archManager) ListInstalled(ctx context.Context) ([]entity.Package, error) {
	//nolint:gosec // G204: fixed pacman query (-Q/-Qm), no user input.
	cmd := exec.CommandContext(ctx, m.queryBin, m.listArgs...)
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
				Type:    m.listType,
				Version: fields[1],
				Status:  entity.StatusInstalled,
			})
		}
	}
	return pkgs, nil
}
