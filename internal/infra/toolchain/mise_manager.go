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
	// If custom check_command is specified, verify execution
	if pkg.CheckCommand != "" {
		if out, ok := executil.ProbeCheckCommand(ctx, pkg.CheckCommand); ok {
			return true, out, nil
		}
	}

	// Inspect `mise ls <tool> --json` (array of version records)
	tool := miseToolName(pkg.ID)
	if tool == "" {
		return false, "", fmt.Errorf("mise: cannot derive tool name from %q", pkg.ID)
	}
	cmd := executil.ExecTool(ctx, "mise", "ls", tool, "--json")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return false, "", err
	}

	found, info := miseLsRecordsInstalled(string(out))
	return found, info, nil
}

// miseToolName strips a version qualifier (node@24.19.0 -> node). A leading
// "@" with nothing before it is not a tool name.
func miseToolName(pkgID string) string {
	name := pkgID
	if i := strings.LastIndex(name, "@"); i > 0 {
		name = name[:i]
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
	InstallPath string `json:"install_path"`
}

// miseLsRecordsInstalled reports whether any record in a `mise ls --json`
// array is installed, returning its version (or install path when the
// version is empty).
func miseLsRecordsInstalled(jsonOut string) (bool, string) {
	var records []miseLsRecord
	if err := json.Unmarshal([]byte(jsonOut), &records); err != nil {
		return false, ""
	}
	for _, r := range records {
		if r.Installed {
			if r.Version != "" {
				return true, r.Version
			}
			return true, r.InstallPath
		}
	}
	return false, ""
}

func (m *MiseManager) Install(ctx context.Context, pkg entity.Package) error {
	cmd := executil.ExecTool(ctx, "mise", "install", pkg.ID)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("mise install %s failed: %s (%w)", pkg.ID, string(out), err)
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
