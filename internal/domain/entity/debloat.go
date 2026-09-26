package entity

import (
	"fmt"
	"sort"
	"strings"
)

// PackageRemoval is one declared package removal.
//
// Rationale is mandatory: a list that removes state nobody explained is how a
// provisioning tool becomes unreviewable.
type PackageRemoval struct {
	ID           string      `yaml:"id"`
	Package      string      `yaml:"package"`
	Type         PackageType `yaml:"type"`
	Category     string      `yaml:"category"`
	Rationale    string      `yaml:"rationale"`
	TargetDistro string      `yaml:"target_distro,omitempty"`
	// ProtectedWhen lists packages whose presence makes removal unsafe. When
	// any of them is installed the removal is skipped and reported, never
	// forced. The measured case is rpcbind, whose installed reverse dependency
	// nfs-common would be cascaded away with it.
	ProtectedWhen []string `yaml:"protected_when,omitempty"`
}

// DebloatSpec is the declared Linux removal policy.
type DebloatSpec struct {
	// NeedrestartDropin is written before the first removal. Ubuntu's
	// needrestart runs automatically and will restart services while a purge is
	// in progress, which over SSH can drop the very session running the
	// provisioning. List-only mode removes that risk.
	NeedrestartDropin string           `yaml:"needrestart_dropin"`
	Removals          []PackageRemoval `yaml:"removals"`
}

// RemovalAction is what the policy decided to do about one package.
type RemovalAction string

const (
	// RemovalPurge removes an installed, unguarded package.
	RemovalPurge RemovalAction = "purge"
	// RemovalAlreadyAbsent reports an installed-state check that found nothing.
	RemovalAlreadyAbsent RemovalAction = "already-absent"
	// RemovalSkip declines because a guard package is installed.
	RemovalSkip RemovalAction = "skip"
)

// RemovalDecision is the outcome for one package, ready to be turned into a
// diagnostic.
type RemovalDecision struct {
	Action   RemovalAction
	Category DiagnosticStatus
	Details  string
}

// ValidatePackageRemovals rejects a removal list that could not be reviewed.
func ValidatePackageRemovals(removals []PackageRemoval) error {
	if len(removals) == 0 {
		return fmt.Errorf("the debloat list is empty")
	}
	seen := make(map[string]bool, len(removals))
	for i, removal := range removals {
		switch {
		case strings.TrimSpace(removal.ID) == "":
			return fmt.Errorf("removal %d has no id", i)
		case strings.TrimSpace(removal.Package) == "":
			return fmt.Errorf("removal %q has no package name", removal.ID)
		case removal.Type == "":
			return fmt.Errorf("removal %q has no package type", removal.ID)
		case strings.TrimSpace(removal.Rationale) == "":
			return fmt.Errorf("removal %q has no rationale; state why the package is not wanted", removal.ID)
		case seen[removal.ID]:
			return fmt.Errorf("removal id %q is declared twice", removal.ID)
		}
		seen[removal.ID] = true
	}
	return nil
}

// ValidateDebloatSpec rejects a spec that could restart services under the
// operator's feet.
func ValidateDebloatSpec(spec DebloatSpec) error {
	if strings.TrimSpace(spec.NeedrestartDropin) == "" {
		return fmt.Errorf("the debloat spec must declare a needrestart drop-in; a purge on a host with automatic " +
			"needrestart can restart services and drop the session running the provisioning")
	}
	return ValidatePackageRemovals(spec.Removals)
}

// ResolveRemoval decides what to do about one package.
//
// No outcome is a warning. An absent package is what a converged host looks
// like, and a guarded package is a deliberate decision, so both stay out of the
// warning budget that the doctor holds at zero.
func ResolveRemoval(removal PackageRemoval, guards map[string]bool, installed bool) RemovalDecision {
	names := make([]string, 0, len(removal.ProtectedWhen))
	for _, guard := range removal.ProtectedWhen {
		if guards[guard] {
			names = append(names, guard)
		}
	}
	if len(names) > 0 {
		sort.Strings(names)
		return RemovalDecision{
			Action:   RemovalSkip,
			Category: DiagInfo,
			Details: fmt.Sprintf("not removing %s: %s %s installed (rationale for removal: %s)",
				removal.Package, strings.Join(names, ", "),
				map[bool]string{true: "is", false: "are"}[len(names) == 1], removal.Rationale),
		}
	}
	if !installed {
		return RemovalDecision{
			Action:   RemovalAlreadyAbsent,
			Category: DiagOK,
			Details:  removal.Package + " is already absent",
		}
	}
	return RemovalDecision{
		Action:   RemovalPurge,
		Category: DiagOK,
		Details: fmt.Sprintf("removing %s (%s): %s",
			removal.Package, removal.Category, removal.Rationale),
	}
}

// AssessRemovalState classifies a package after a completed run.
func AssessRemovalState(installed, guarded bool, guardName string) (DiagnosticStatus, string) {
	switch {
	case !installed:
		return DiagOK, "absent as declared"
	case guarded:
		return DiagInfo, fmt.Sprintf("still installed because %s is present; the removal is guarded", guardName)
	default:
		return DiagError, "still installed after a completed run; the declared policy did not hold"
	}
}

// renderNeedrestartConfig forces Ubuntu's needrestart into list-only mode.
//
// The package is installed by default on Ubuntu Server (3.11 was measured on
// vps_oracle_2) with no override, so it runs automatically and restarts services
// after a package change. Over SSH that can drop the session that is performing
// the change.
func renderNeedrestartConfig() string {
	return strings.Join([]string{
		"# Managed by envctl; review before editing.",
		"#",
		"# Ubuntu ships needrestart in automatic mode, which restarts services after",
		"# a package change. During a purge over SSH that can drop the very session",
		"# running the provisioning, so the mode is pinned to list-only here.",
		"NEEDRESTART_MODE=l",
		"NEEDRESTART_SUSPEND=1",
		"",
	}, "\n")
}

// RenderNeedrestartConfig is the exported form used by the adapter.
func RenderNeedrestartConfig() string { return renderNeedrestartConfig() }
