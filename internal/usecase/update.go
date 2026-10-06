package usecase

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/eajdias/envctl/internal/domain/entity"
	"github.com/eajdias/envctl/internal/infra/toolchain"
)

// UpdateGroup identifies an install mechanism family. Only the families whose
// updates are user-local, need no sudo and are reversible belong here; the OS
// package managers deliberately do not, because upgrading a subset through
// pacman is a partial upgrade and Arch forbids those.
type UpdateGroup string

const (
	GroupMise UpdateGroup = "mise"
	GroupNpm  UpdateGroup = "npm"
	GroupUV   UpdateGroup = "uv"
	GroupGo   UpdateGroup = "go"
)

// automatableGroup maps a manifest install type to the group that can update it.
// A type absent from this map is never automated.
func automatableGroup(installType entity.PackageType) (UpdateGroup, bool) {
	switch installType {
	case entity.PackageTypeMise:
		return GroupMise, true
	case entity.PackageTypeNpm:
		return GroupNpm, true
	case entity.PackageTypePip:
		return GroupUV, true
	case entity.PackageTypeGo:
		return GroupGo, true
	default:
		return "", false
	}
}

// updateCommand is the exact command a group runs to move a target to latest.
// Several install_target values in lsp.yaml already carry the "@latest" suffix,
// so appending another one would produce an unrunnable command.
func updateCommand(group UpdateGroup, target string) string {
	pin := ""
	if !strings.HasSuffix(target, "@latest") {
		pin = "@latest"
	}
	switch group {
	case GroupMise:
		return "mise install " + target + pin
	case GroupNpm:
		return "npm install -g " + target + pin
	case GroupUV:
		return "uv tool upgrade " + target
	case GroupGo:
		return "go install " + target + pin
	default:
		return ""
	}
}

// UpdateEnv is the machine behind the use case. The real implementation calls
// the providers helpers (installedVersion, npmLatest, runWithToolchain)
// directly, so there is exactly one way to resolve a version in this codebase;
// tests fake the interface.
type UpdateEnv interface {
	installedVersion(binary string) string
	latestVersion(group UpdateGroup, target string) string
	applyUpdate(ctx context.Context, group UpdateGroup, target string) (string, error)
}

// NewRealUpdateEnv wires the use case to the actual machine.
func NewRealUpdateEnv() UpdateEnv { return &realUpdateEnv{} }

// realUpdateEnv runs against the machine. It is a zero-field struct on
// purpose: every helper it needs already exists at package level (version.go,
// provision_providers.go), so injected fields would only be a second spelling
// of the same call.
type realUpdateEnv struct{}

func (e *realUpdateEnv) installedVersion(binary string) string {
	return installedVersion(context.Background(), binary)
}

func (e *realUpdateEnv) latestVersion(group UpdateGroup, target string) string {
	switch group {
	case GroupMise, GroupNpm:
		return npmLatest(context.Background(), target)
	case GroupUV:
		return uvToolVersionOf(context.Background(), target)
	case GroupGo:
		return goLatestOf(target)
	default:
		return ""
	}
}

func (e *realUpdateEnv) applyUpdate(ctx context.Context, group UpdateGroup, target string) (string, error) {
	var name string
	var args []string
	switch group {
	case GroupMise:
		name, args = "mise", []string{"install", target + "@latest"}
	case GroupNpm:
		// Same user-local prefix as NpmManager: a bare `npm install -g`
		// lands in a root-owned system directory on Arch/Debian.
		// The display command (updateCommand) shows the portable form.
		name, args = "npm", []string{"install", "-g", target + "@latest"}
		if prefix, err := toolchain.UserLocalPrefix(); err == nil {
			args = []string{"install", "-g", "--prefix", prefix, target + "@latest"}
		}
	case GroupUV:
		name, args = "uv", []string{"tool", "upgrade", target}
	case GroupGo:
		name, args = "go", []string{"install", target + "@latest"}
	default:
		return "", errUnautomatableGroup
	}
	return runWithToolchain(ctx, name, args...)
}

// errUnautomatableGroup guards a programming error rather than a machine
// condition: collect() only ever emits the three groups above.
var errUnautomatableGroup = errors.New("install group is not automatable")

// UpdateCandidate is one tool that envctl knows how to keep current. Target is
// the real install name, which can differ from the manifest id: the "typescript"
// package installs tsc, while the "typescript" LSP installs
// typescript-language-server. Label keeps the inventory readable when the two
// share an id.
type UpdateCandidate struct {
	ID      string
	Label   string
	Group   UpdateGroup
	Target  string
	Binary  string
	Command string
}

// displayName prefers the target, which is what the user would type, and falls
// back to the id.
func (c UpdateCandidate) DisplayName() string {
	if c.Label != "" {
		return c.Label
	}
	if c.Target != "" {
		return c.Target
	}
	return c.ID
}

// UpdateOutcome is the record of a single tool after the run, whether it moved,
// was already current, or could not be resolved.
type UpdateOutcome struct {
	ID     string
	Group  UpdateGroup
	From   string
	To     string
	Detail string
}

// UpdateInventory is what the manifests declare, before any OS filtering.
type UpdateInventory struct {
	Packages []entity.Package
	LSPs     []entity.LSP
}

// UpdateOptions is the caller's intent.
type UpdateOptions struct {
	// DryRun reports what would run and touches nothing.
	DryRun bool
	// List reports the inventory without asking any registry for a version.
	List bool
	// Only narrows the run to a single group.
	Only UpdateGroup
}

// UpdateResult separates what happened from what could not, so one failure never
// reads as a failed run.
type UpdateResult struct {
	Planned []UpdateCandidate
	Applied []UpdateOutcome
	Current []UpdateOutcome
	Skipped []UpdateOutcome
	Failed  []UpdateOutcome
}

// UpdateUseCase keeps the global toolchain current.
type UpdateUseCase struct {
	env UpdateEnv
}

// NewUpdateUseCase builds the use case over a machine.
func NewUpdateUseCase(env UpdateEnv) *UpdateUseCase {
	return &UpdateUseCase{env: env}
}

// collect turns manifest entries into the automatable inventory, dropping every
// OS package manager.
func (uc *UpdateUseCase) collect(packages []entity.Package, lsps []entity.LSP) []UpdateCandidate {
	var out []UpdateCandidate
	seen := map[string]bool{}

	add := func(id, label string, installType entity.PackageType, target, binary string) {
		group, ok := automatableGroup(installType)
		if !ok || target == "" || binary == "" || seen[target] {
			return
		}
		// A runtime pin (node@24.19.0) belongs to the providers phase, which
		// already keeps the agent CLIs current.
		if group == GroupMise && isProviderRuntime(id) {
			return
		}
		seen[target] = true
		out = append(out, UpdateCandidate{
			ID:      id,
			Label:   label,
			Group:   group,
			Target:  target,
			Binary:  binary,
			Command: updateCommand(group, target),
		})
	}

	for _, l := range lsps {
		add(l.ID, l.InstallTarget, l.InstallType, l.InstallTarget, l.CheckBinary)
	}
	for _, p := range packages {
		binary := p.CheckCommand
		if idx := strings.Index(binary, " "); idx > 0 {
			binary = binary[:idx]
		}
		// A package declared as an npm global has no separate target field: the
		// id is the npm package name, which is how the providers phase installs it.
		add(p.ID, p.ID, p.Type, p.ID, binary)
	}
	return out
}

// Execute reports the inventory and, unless asked not to, applies the updates.
func (uc *UpdateUseCase) Execute(ctx context.Context, inv UpdateInventory, opts UpdateOptions) (*UpdateResult, error) {
	result := &UpdateResult{}

	for _, c := range uc.collect(inv.Packages, inv.LSPs) {
		if opts.Only != "" && c.Group != opts.Only {
			continue
		}
		result.Planned = append(result.Planned, c)
	}
	sort.SliceStable(result.Planned, func(i, j int) bool {
		if result.Planned[i].Group != result.Planned[j].Group {
			return result.Planned[i].Group < result.Planned[j].Group
		}
		return result.Planned[i].DisplayName() < result.Planned[j].DisplayName()
	})

	if opts.List || opts.DryRun {
		return result, nil
	}

	for _, c := range result.Planned {
		from := uc.env.installedVersion(c.Binary)
		if from == "" {
			result.Skipped = append(result.Skipped, UpdateOutcome{
				ID: c.ID, Group: c.Group,
				Detail: "not installed; run the provisioning that owns it",
			})
			continue
		}
		latest := uc.env.latestVersion(c.Group, c.Target)
		if latest == "" {
			// Without a version to compare against, applying would be a blind
			// reinstall, so report and move on.
			result.Skipped = append(result.Skipped, UpdateOutcome{
				ID: c.ID, Group: c.Group, From: from,
				Detail: "latest version unknown (registry unreachable?)",
			})
			continue
		}
		if !versionsDiffer(from, latest) {
			result.Current = append(result.Current, UpdateOutcome{
				ID: c.ID, Group: c.Group, From: from, To: latest,
			})
			continue
		}

		out, err := uc.env.applyUpdate(ctx, c.Group, c.Target)
		if err != nil {
			result.Failed = append(result.Failed, UpdateOutcome{
				ID: c.ID, Group: c.Group, From: from,
				Detail: fmt.Sprintf("%v (%s) — run: %s", err, strings.TrimSpace(out), c.Command),
			})
			continue
		}
		to := uc.env.installedVersion(c.Binary)
		if to == "" {
			to = latest
		}
		result.Applied = append(result.Applied, UpdateOutcome{
			ID: c.ID, Group: c.Group, From: from, To: to,
		})
	}
	return result, nil
}
