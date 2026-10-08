package usecase

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/eajdias/envctl/internal/domain/entity"
)

// fakeUpdateEnv stands in for the machine: it answers version probes from maps
// and records every apply, so the tests assert on decisions rather than on the
// network or on real package managers.
type fakeUpdateEnv struct {
	installed map[string]string // binary -> version
	latest    map[string]string // target -> version ("" means unknown)
	applied   []string          // targets, in call order
	failOn    map[string]error  // target -> error the apply should return
	appliedTo map[string]string // target -> version after the apply
	binaryOf  map[string]string // target -> binary, so a fake apply can move the
	// installed version the way a real toolchain would
}

func (f *fakeUpdateEnv) installedVersion(binary string) string { return f.installed[binary] }

func (f *fakeUpdateEnv) latestVersion(_ UpdateGroup, target string) string {
	return f.latest[target]
}

func (f *fakeUpdateEnv) applyUpdate(_ context.Context, group UpdateGroup, target string) (string, error) {
	f.applied = append(f.applied, target)
	if err, ok := f.failOn[target]; ok {
		return "", err
	}
	version := f.appliedTo[target]
	if binary, ok := f.binaryOf[target]; ok {
		f.installed[binary] = version
	}
	return version, nil
}

func (f *fakeUpdateEnv) wasApplied(target string) bool {
	for _, a := range f.applied {
		if a == target {
			return true
		}
	}
	return false
}

// The manifest already carries "@latest" in some install_target values, so the
// command must not append a second one. Both halves pin: `install` resolves
// the version, `use -g` activates it (an install without the pin leaves an
// orphan shim — installed, not active, off PATH).
func TestUpdateCommandIsWellFormed(t *testing.T) {
	if got := updateCommand(GroupMise, "npm:typescript"); got != "mise install --yes npm:typescript@latest && mise use -g npm:typescript@latest" {
		t.Errorf("mise npm-backend command = %q", got)
	}
	if got := updateCommand(GroupMise, "npm:typescript@latest"); got != "mise install --yes npm:typescript@latest && mise use -g npm:typescript@latest" {
		t.Errorf("a target already ending in @latest must not double it: %q", got)
	}
	if got := updateCommand(GroupMise, "node"); got != "mise install --yes node@latest && mise use -g node@latest" {
		t.Errorf("mise command = %q", got)
	}
	if got := updateCommand(GroupUV, "pytest"); got != "uv tool upgrade pytest" {
		t.Errorf("uv command = %q", got)
	}
}

// node@24.19.0 is a providers entry pinning the Node runtime, not a tool with a
// newer version to move to.
func TestUpdateExcludesProvidersRuntimeEntries(t *testing.T) {
	uc := NewUpdateUseCase(&fakeUpdateEnv{
		installed: map[string]string{}, latest: map[string]string{},
		failOn: map[string]error{}, binaryOf: map[string]string{},
	})

	candidates := uc.collect([]entity.Package{
		{ID: "node@24.19.0", Type: entity.PackageTypeMise, CheckCommand: "node --version"},
		{ID: "npm:typescript", Type: entity.PackageTypeMise, CheckCommand: "tsc --version"},
	}, []entity.LSP{})

	for _, c := range candidates {
		if c.ID == "node@24.19.0" {
			t.Errorf("providers runtime entry leaked into the inventory: %+v", c)
		}
	}
	if len(candidates) != 1 {
		t.Errorf("candidates = %+v, want only npm:typescript", candidates)
	}
}

// Package and LSP entries name different mise-backed targets (the
// "npm:typescript" package and the "npm:typescript-language-server" LSP). Both are real
// tools, so the inventory keeps both and labels them by target.
func TestUpdateKeepsDistinctTargetsThatShareAnID(t *testing.T) {
	uc := NewUpdateUseCase(&fakeUpdateEnv{
		installed: map[string]string{}, latest: map[string]string{},
		failOn: map[string]error{}, binaryOf: map[string]string{},
	})

	candidates := uc.collect(
		[]entity.Package{{ID: "npm:typescript", Type: entity.PackageTypeMise, CheckCommand: "tsc --version"}},
		[]entity.LSP{{
			ID: "typescript", InstallType: entity.PackageTypeMise,
			InstallTarget: "npm:typescript-language-server", CheckBinary: "typescript-language-server",
		}},
	)

	if len(candidates) != 2 {
		t.Fatalf("candidates = %+v, want both targets", candidates)
	}
	seen := map[string]bool{}
	for _, c := range candidates {
		if seen[c.Target] {
			t.Errorf("target %q appears twice", c.Target)
		}
		seen[c.Target] = true
	}
}

func TestUpdateCollectsOnlyAutomatableGroups(t *testing.T) {
	uc := NewUpdateUseCase(&fakeUpdateEnv{installed: map[string]string{}, latest: map[string]string{}, failOn: map[string]error{}, binaryOf: map[string]string{}})

	candidates := uc.collect([]entity.Package{
		{ID: "npm:typescript", Type: entity.PackageTypeMise, OS: "arch,cachyos", CheckCommand: "tsc --version"},
		{ID: "pytest", Type: entity.PackageTypePip, OS: "arch,cachyos", CheckCommand: "pytest --version"},
		// Every OS package manager must stay out: upgrading a subset through
		// pacman is a partial upgrade, which Arch forbids outright.
		{ID: "ruff", Type: entity.PackageTypePacman, OS: "arch,cachyos", CheckCommand: "ruff --version"},
		{ID: "git", Type: entity.PackageTypeApt, OS: "debian,ubuntu", CheckCommand: "git --version"},
		{ID: "7zip", Type: entity.PackageTypeWinget, OS: "windows", CheckCommand: "7z --version"},
		{ID: "cachyos-settings", Type: entity.PackageTypeParu, OS: "arch,cachyos", CheckCommand: "cachyos-settings --version"},
	}, []entity.LSP{})

	got := map[string]UpdateGroup{}
	for _, c := range candidates {
		got[c.ID] = c.Group
	}
	want := map[string]UpdateGroup{"npm:typescript": GroupMise, "pytest": GroupUV}
	if len(got) != len(want) {
		t.Fatalf("candidates = %v, want exactly %v", got, want)
	}
	for id, group := range want {
		if got[id] != group {
			t.Errorf("candidate %q group = %q, want %q", id, got[id], group)
		}
	}
}

func TestUpdateAppliesBehindVersionsAndSkipsCurrent(t *testing.T) {
	env := &fakeUpdateEnv{
		installed: map[string]string{"tsc": "5.4.2", "prettier": "3.1.0", "pytest": "9.0.0"},
		latest:    map[string]string{"npm:typescript": "5.9.2", "npm:prettier": "3.1.0", "pytest": "9.1.1"},
		appliedTo: map[string]string{"npm:typescript": "5.9.2", "pytest": "9.1.1"},
		failOn:    map[string]error{},
		binaryOf:  map[string]string{"npm:typescript": "tsc", "pytest": "pytest"},
	}
	uc := NewUpdateUseCase(env)

	result, err := uc.Execute(context.Background(), UpdateInventory{
		Packages: []entity.Package{{ID: "npm:typescript", Type: entity.PackageTypeMise, CheckCommand: "tsc --version"}},
		LSPs:     []entity.LSP{{ID: "prettier", InstallType: entity.PackageTypeMise, InstallTarget: "npm:prettier", CheckBinary: "prettier"}},
	}, UpdateOptions{})

	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(result.Applied) != 1 || result.Applied[0].ID != "npm:typescript" {
		t.Fatalf("applied = %+v, want only npm:typescript", result.Applied)
	}
	if env.wasApplied("npm:prettier") {
		t.Error("prettier is already current and must not be touched")
	}
	if result.Applied[0].From != "5.4.2" || result.Applied[0].To != "5.9.2" {
		t.Errorf("applied outcome = %+v, want 5.4.2 -> 5.9.2", result.Applied[0])
	}
}

func TestUpdateDryRunNeverApplies(t *testing.T) {
	env := &fakeUpdateEnv{
		installed: map[string]string{"tsc": "5.4.2"},
		latest:    map[string]string{"npm:typescript": "5.9.2"},
		appliedTo: map[string]string{"npm:typescript": "5.9.2"},
		failOn:    map[string]error{},
		binaryOf:  map[string]string{"npm:typescript": "tsc", "pytest": "pytest"},
	}
	uc := NewUpdateUseCase(env)

	result, err := uc.Execute(context.Background(), UpdateInventory{
		Packages: []entity.Package{{ID: "npm:typescript", Type: entity.PackageTypeMise, CheckCommand: "tsc --version"}},
	}, UpdateOptions{DryRun: true})

	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(env.applied) != 0 {
		t.Errorf("dry run applied %v, want nothing", env.applied)
	}
	if len(result.Planned) != 1 || !strings.Contains(result.Planned[0].Command, "mise install --yes npm:") || !strings.Contains(result.Planned[0].Command, "mise use -g npm:") {
		t.Fatalf("planned = %+v, want one mise install+pin command", result.Planned)
	}
}

// An unknown latest must never be treated as "behind": without a version to
// compare, applying would be a blind reinstall of everything.
func TestUpdateUnknownLatestUpdatesNothing(t *testing.T) {
	env := &fakeUpdateEnv{
		installed: map[string]string{"tsc": "5.4.2"},
		latest:    map[string]string{}, // registry unreachable
		appliedTo: map[string]string{},
		failOn:    map[string]error{},
		binaryOf:  map[string]string{"npm:typescript": "tsc", "pytest": "pytest"},
	}
	uc := NewUpdateUseCase(env)

	result, err := uc.Execute(context.Background(), UpdateInventory{
		Packages: []entity.Package{{ID: "npm:typescript", Type: entity.PackageTypeMise, CheckCommand: "tsc --version"}},
	}, UpdateOptions{})

	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(env.applied) != 0 {
		t.Errorf("applied %v with an unknown latest, want nothing", env.applied)
	}
	if len(result.Skipped) == 0 {
		t.Error("a tool with an unknown latest must be reported as skipped")
	}
}

// One failure must not take the rest of the list down: each update is unit-level.
func TestUpdateFailureIsIsolatedPerTool(t *testing.T) {
	env := &fakeUpdateEnv{
		installed: map[string]string{"pytest": "9.0.0", "tsc": "5.4.2"},
		latest:    map[string]string{"pytest": "9.1.1", "npm:typescript": "5.9.2"},
		appliedTo: map[string]string{"npm:typescript": "5.9.2"},
		failOn:    map[string]error{"pytest": errors.New("network down")},
		binaryOf:  map[string]string{"npm:typescript": "tsc", "pytest": "pytest"},
	}
	uc := NewUpdateUseCase(env)

	result, err := uc.Execute(context.Background(), UpdateInventory{
		Packages: []entity.Package{
			{ID: "pytest", Type: entity.PackageTypePip, CheckCommand: "pytest --version"},
			{ID: "npm:typescript", Type: entity.PackageTypeMise, CheckCommand: "tsc --version"},
		},
	}, UpdateOptions{})

	if err != nil {
		t.Fatalf("a per-tool failure must not fail the run: %v", err)
	}
	if len(result.Applied) != 1 || result.Applied[0].ID != "npm:typescript" {
		t.Errorf("applied = %+v, want npm:typescript to still go through", result.Applied)
	}
	if len(result.Failed) != 1 || result.Failed[0].ID != "pytest" {
		t.Errorf("failed = %+v, want pytest", result.Failed)
	}
	if !strings.Contains(result.Failed[0].Detail, "network down") {
		t.Errorf("failure detail = %q, want the underlying error surfaced", result.Failed[0].Detail)
	}
}

func TestUpdateFilterNarrowsToOneGroup(t *testing.T) {
	env := &fakeUpdateEnv{
		installed: map[string]string{"tsc": "5.4.2", "pytest": "9.0.0"},
		latest:    map[string]string{"npm:typescript": "5.9.2", "pytest": "9.1.1"},
		appliedTo: map[string]string{"npm:typescript": "5.9.2", "pytest": "9.1.1"},
		failOn:    map[string]error{},
		binaryOf:  map[string]string{"npm:typescript": "tsc", "pytest": "pytest"},
	}
	uc := NewUpdateUseCase(env)

	_, err := uc.Execute(context.Background(), UpdateInventory{
		Packages: []entity.Package{
			{ID: "npm:typescript", Type: entity.PackageTypeMise, CheckCommand: "tsc --version"},
			{ID: "pytest", Type: entity.PackageTypePip, CheckCommand: "pytest --version"},
		},
	}, UpdateOptions{Only: GroupUV})

	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !env.wasApplied("pytest") {
		t.Error("--only uv must still update pytest")
	}
	if env.wasApplied("npm:typescript") {
		t.Error("--only uv must not touch mise tools")
	}
}

func TestUpdateListDoesNotQueryLatest(t *testing.T) {
	queried := map[string]bool{}
	env := &countingUpdateEnv{fakeUpdateEnv: fakeUpdateEnv{installed: map[string]string{"tsc": "5.4.2"}}, queried: queried}
	uc := NewUpdateUseCase(env)

	if _, err := uc.Execute(context.Background(), UpdateInventory{
		Packages: []entity.Package{{ID: "npm:typescript", Type: entity.PackageTypeMise, CheckCommand: "tsc --version"}},
	}, UpdateOptions{List: true}); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(queried) != 0 {
		t.Errorf("--list queried latest for %v, want no network at all", queried)
	}
}

type countingUpdateEnv struct {
	fakeUpdateEnv
	queried map[string]bool
}

func (c *countingUpdateEnv) latestVersion(_ UpdateGroup, target string) string {
	c.queried[target] = true
	return ""
}
