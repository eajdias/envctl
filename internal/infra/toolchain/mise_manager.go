package toolchain

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/eajdias/envctl/internal/domain/entity"
	"github.com/eajdias/envctl/internal/domain/repository"
	"github.com/eajdias/envctl/internal/infra/executil"
)

// MiseManager installs language runtimes (node, go) through mise. npm
// globals stay with NpmManager: mise only owns runtimes here, never packages.
type MiseManager struct{}

// NewMiseManager creates a new PackageManager for the mise runtime manager.
func NewMiseManager() repository.PackageManager {
	return &MiseManager{}
}

func (m *MiseManager) Type() entity.PackageType {
	return entity.PackageTypeMise
}

func (m *MiseManager) IsAvailable(ctx context.Context) bool {
	cmd := executil.ExecTool(ctx, "mise", "--version")
	return cmd.Run() == nil
}

func (m *MiseManager) IsInstalled(ctx context.Context, pkg entity.Package) (bool, string, error) {
	// Backend tools (npm:<pkg>) are owned by mise: the `mise ls` record is
	// the truth. A PATH binary without a record is a legacy (volta,
	// npm-prefix) copy that the sweep archives after install — probing it
	// here would report "installed" forever and skip the migration.
	if hasMiseBackend(pkg.ID) {
		return m.lsInstalled(ctx, miseToolName(pkg.ID))
	}
	// If custom check_command is specified, verify execution
	if pkg.CheckCommand != "" {
		if out, ok := executil.ProbeCheckCommand(ctx, pkg.CheckCommand); ok {
			return true, out, nil
		}
	}

	return m.lsInstalled(ctx, miseToolName(pkg.ID))
}

// hasMiseBackend reports whether an id carries a backend prefix
// ("npm:prettier"). Runtimes ("node@24.19.0") have none.
func hasMiseBackend(pkgID string) bool {
	return strings.Index(pkgID, ":") > 0
}

// lsInstalled inspects `mise ls <tool> --json` (array of version records).
func (m *MiseManager) lsInstalled(ctx context.Context, tool string) (bool, string, error) {
	if tool == "" {
		return false, "", fmt.Errorf("mise: cannot derive tool name from %q", tool)
	}
	cmd := executil.ExecTool(ctx, "mise", "ls", tool, "--json")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return false, "", err
	}

	found, info := miseLsRecordsInstalled(string(out))
	return found, info, nil
}

// miseToolName derives the `mise ls` query from a manifest ID, keeping a
// backend prefix ("npm:") and never stripping a scope ("@scope/pkg" has no
// version). A "@suffix" is cut only when the head is a real tool name.
func miseToolName(pkgID string) string {
	name := pkgID
	if i := strings.LastIndex(name, "@"); i > 0 {
		head := name[:i]
		if head != "" && !strings.HasSuffix(head, ":") {
			name = head
		}
	}
	if name == "" || name == "@" {
		return ""
	}
	return name
}

// miseLsRecord mirrors one entry of `mise ls --json` (object keyed by tool
// name; `mise ls <tool> --json` is the array directly). Unknown fields are
// ignored so registry additions never break the parse.
type miseLsRecord struct {
	Version     string `json:"version"`
	Installed   bool   `json:"installed"`
	Active      bool   `json:"active"`
	InstallPath string `json:"install_path"`
}

// miseLsRecordsInstalled reports whether any record in a `mise ls --json`
// array is installed AND active, returning its version (or install path when
// the version is empty). Installed-but-inactive is an orphan shim: `mise
// install` without `mise use -g` leaves the tool off PATH (mise's own help:
// "Installing alone does not add the tool to your config"), so it must not
// count — otherwise the doctor stays green over unreachable tools.
func miseLsRecordsInstalled(jsonOut string) (bool, string) {
	var records []miseLsRecord
	if err := json.Unmarshal([]byte(jsonOut), &records); err != nil {
		return false, ""
	}
	for _, r := range records {
		if r.Installed && r.Active {
			if r.Version != "" {
				return true, r.Version
			}
			return true, r.InstallPath
		}
	}
	return false, ""
}

func (m *MiseManager) Install(ctx context.Context, pkg entity.Package) error {
	// --yes reaches the embedded aube reputation gate (e.g. low weekly
	// downloads): the manifest is the trust decision, and a provisioner
	// cannot answer an interactive prompt. Without it, curated niche tools
	// fail closed in non-interactive runs.
	cmd := executil.ExecTool(ctx, "mise", "install", "--yes", pkg.ID)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("mise install %s failed: %s (%w)", pkg.ID, string(out), err)
	}
	// Installing alone does not add the tool to the mise config, so it stays
	// inactive (off PATH) until pinned. `mise use -g` writes the same ID to
	// the global config.toml, which is what flips `active` in `mise ls` and
	// exposes the shim — the same call phase 0 already makes for the Node
	// runtime. Same ID string, never an invented version.
	pin := executil.ExecTool(ctx, "mise", "use", "-g", pkg.ID)
	if out, err := pin.CombinedOutput(); err != nil {
		return fmt.Errorf("mise use -g %s failed: %s (%w)", pkg.ID, string(out), err)
	}
	return nil
}

func (m *MiseManager) ListInstalled(ctx context.Context) ([]entity.Package, error) {
	cmd := executil.ExecTool(ctx, "mise", "ls", "--json", "--installed")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	var byTool map[string][]miseLsRecord
	if err := json.Unmarshal(out, &byTool); err != nil {
		return nil, err
	}

	var pkgs []entity.Package
	for tool, records := range byTool {
		for _, r := range records {
			if !r.Installed {
				continue
			}
			version := r.Version
			if version == "" {
				version = r.InstallPath
			}
			pkgs = append(pkgs, entity.Package{
				ID:      tool,
				Name:    tool,
				Type:    entity.PackageTypeMise,
				Status:  entity.StatusInstalled,
				Version: version,
			})
		}
	}

	return pkgs, nil
}
