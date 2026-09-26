package performance

import (
	"context"
	"fmt"
	"github.com/eajdias/envctl/internal/domain/entity"
	"github.com/eajdias/envctl/internal/domain/repository"
)

// installedFunc reports whether a package is present, reusing the package
// managers' own probe so the debloat decision and the doctor audit can never
// disagree about what is installed.
type installedFunc func(ctx context.Context, pkg entity.Package) (bool, string, error)

// aptRemover removes Debian/Ubuntu packages.
type aptRemover struct {
	run       commandRunner
	installed installedFunc
	elevate   bool
	aptPath   string
}

func newAptRemover(run commandRunner, installed installedFunc, elevate bool) *aptRemover {
	return &aptRemover{run: run, installed: installed, elevate: elevate, aptPath: "apt-get"}
}

func (r *aptRemover) Type() entity.PackageType { return entity.PackageTypeApt }

func (r *aptRemover) IsInstalled(ctx context.Context, pkg entity.Package) (bool, string, error) {
	if r.installed == nil {
		return false, "", nil
	}
	return r.installed(ctx, pkg)
}

func (r *aptRemover) Remove(ctx context.Context, pkg entity.Package) error {
	return execRemoval(ctx, r.run, r.elevate, r.aptPath, []string{"purge", "-y", pkg.ID})
}

// pacmanRemover removes Arch packages.
type pacmanRemover struct {
	run        commandRunner
	installed  installedFunc
	elevate    bool
	pacmanPath string
}

func newPacmanRemover(run commandRunner, installed installedFunc, elevate bool) *pacmanRemover {
	return &pacmanRemover{run: run, installed: installed, elevate: elevate, pacmanPath: "pacman"}
}

func (r *pacmanRemover) Type() entity.PackageType { return entity.PackageTypePacman }

func (r *pacmanRemover) IsInstalled(ctx context.Context, pkg entity.Package) (bool, string, error) {
	if r.installed == nil {
		return false, "", nil
	}
	return r.installed(ctx, pkg)
}

func (r *pacmanRemover) Remove(ctx context.Context, pkg entity.Package) error {
	return execRemoval(ctx, r.run, r.elevate, r.pacmanPath, []string{"-R", "--noconfirm", pkg.ID})
}

// execRemoval issues an argv-only removal command, prefixed with sudo -n when
// elevated so an automated run can never block on a password prompt.
func execRemoval(ctx context.Context, run commandRunner, elevate bool, name string, args []string) error {
	if elevate {
		_, err := run(ctx, "sudo", append([]string{"-n", name}, args...)...)
		return err
	}
	_, err := run(ctx, name, args...)
	return err
}

type linuxDebloatManager struct {
	guard    *dropinWriter
	removers map[entity.PackageType]repository.PackageRemover
}

// NewLinuxDebloatManager creates the production Linux removal adapter.
func NewLinuxDebloatManager(
	aptPath, pacmanPath string,
	installed installedFunc,
) repository.LinuxDebloatManager {
	run := execCommand
	guardPath := "/etc/needrestart/conf.d/99-envctl.conf"
	return newLinuxDebloatManager(
		newDropinWriter(guardPath, run, nowFunc, needsElevation()),
		map[entity.PackageType]repository.PackageRemover{
			entity.PackageTypeApt:    newAptRemover(aptRunner(aptPath, run), installed, needsElevation()),
			entity.PackageTypePacman: newPacmanRemover(pacmanRunner(pacmanPath, run), installed, needsElevation()),
		},
	)
}

func aptRunner(path string, run commandRunner) commandRunner {
	return func(ctx context.Context, name string, args ...string) ([]byte, error) {
		if name == "apt-get" && path != "" {
			name = path
		}
		return run(ctx, name, args...)
	}
}

func pacmanRunner(path string, run commandRunner) commandRunner {
	return func(ctx context.Context, name string, args ...string) ([]byte, error) {
		if name == "pacman" && path != "" {
			name = path
		}
		return run(ctx, name, args...)
	}
}

func newLinuxDebloatManager(guard *dropinWriter, removers map[entity.PackageType]repository.PackageRemover) *linuxDebloatManager {
	return &linuxDebloatManager{guard: guard, removers: removers}
}

// Apply writes the needrestart guard and then walks the removal list.
//
// The guard comes first, always. Ubuntu ships needrestart in automatic mode
// (measured 3.11-1ubuntu2 with no override) and it restarts services after a
// package change, which over SSH can drop the session performing the change.
func (m *linuxDebloatManager) Apply(ctx context.Context, spec entity.DebloatSpec, dryRun bool) ([]entity.Diagnostic, error) {
	if err := entity.ValidateDebloatSpec(spec); err != nil {
		return []entity.Diagnostic{{
			Category: entity.DiagError,
			System:   "Debloat",
			Target:   "linux",
			Details:  err.Error(),
		}}, err
	}

	installedPackages := m.collectInstalled(ctx, spec)

	if dryRun {
		diagnostics := make([]entity.Diagnostic, 0, len(spec.Removals))
		for _, removal := range spec.Removals {
			decision := entity.ResolveRemoval(removal, installedPackages, false)
			diagnostics = append(diagnostics, entity.Diagnostic{
				Category: entity.DiagInfo,
				System:   "Debloat",
				Target:   removal.Package,
				Details:  "would " + string(decision.Action) + ": " + decision.Details,
			})
		}
		return diagnostics, nil
	}

	if _, _, err := m.guard.Install(entity.RenderNeedrestartConfig(), 0o644, false); err != nil {
		return []entity.Diagnostic{{
			Category: entity.DiagError,
			System:   "Debloat",
			Target:   m.guard.destination,
			Details: fmt.Sprintf("writing the needrestart guard failed: %v; refusing to purge with automatic "+
				"needrestart enabled, because it can restart services and drop this session", err),
		}}, err
	}

	diagnostics := make([]entity.Diagnostic, 0, len(spec.Removals))
	for _, removal := range spec.Removals {
		remover, ok := m.removers[removal.Type]
		if !ok {
			diagnostics = append(diagnostics, entity.Diagnostic{
				Category: entity.DiagInfo,
				System:   "Debloat",
				Target:   removal.Package,
				Details:  fmt.Sprintf("no remover is configured for package type %q; left as it is", removal.Type),
			})
			continue
		}

		pkg := entity.Package{ID: removal.Package, Name: removal.Package, Type: removal.Type}
		installed, _, err := remover.IsInstalled(ctx, pkg)
		if err != nil {
			detail := fmt.Sprintf("cannot determine whether %s is installed: %v; left as it is", removal.Package, err)
			return append(diagnostics, entity.Diagnostic{
				Category: entity.DiagInfo,
				System:   "Debloat",
				Target:   removal.Package,
				Details:  detail,
			}), nil
		}

		decision := entity.ResolveRemoval(removal, installedPackages, installed)
		switch decision.Action {
		case entity.RemovalAlreadyAbsent, entity.RemovalSkip:
			diagnostics = append(diagnostics, entity.Diagnostic{
				Category: decision.Category,
				System:   "Debloat",
				Target:   removal.Package,
				Details:  decision.Details,
			})
			continue
		}

		if err := remover.Remove(ctx, pkg); err != nil {
			detail := fmt.Sprintf("removing %s failed: %v", removal.Package, err)
			return append(diagnostics, entity.Diagnostic{
				Category: entity.DiagError,
				System:   "Debloat",
				Target:   removal.Package,
				Details:  detail,
			}), err
		}
		diagnostics = append(diagnostics, entity.Diagnostic{
			Category: entity.DiagOK,
			System:   "Debloat",
			Target:   removal.Package,
			Details:  decision.Details,
		})
	}
	return diagnostics, nil
}

// collectInstalled builds the set of installed package names once, so the guard
// check for every entry is a map lookup rather than a probe per entry.
func (m *linuxDebloatManager) collectInstalled(ctx context.Context, spec entity.DebloatSpec) map[string]bool {
	installed := make(map[string]bool)
	for _, removal := range spec.Removals {
		remover, ok := m.removers[removal.Type]
		if !ok {
			continue
		}
		for _, name := range append([]string{removal.Package}, removal.ProtectedWhen...) {
			if _, seen := installed[name]; seen {
				continue
			}
			present, _, err := remover.IsInstalled(ctx, entity.Package{ID: name, Name: name, Type: removal.Type})
			if err != nil {
				continue
			}
			installed[name] = present
		}
	}
	return installed
}
