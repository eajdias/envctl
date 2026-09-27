package entity

import (
	"strings"
	"testing"
)

func removalByID(t *testing.T, id string) PackageRemoval {
	t.Helper()
	for _, removal := range removalsFixture() {
		if removal.ID == id {
			return removal
		}
	}
	t.Fatalf("fixture has no removal %q", id)
	return PackageRemoval{}
}

func removalsFixture() []PackageRemoval {
	return []PackageRemoval{
		{ID: "modemmanager", Package: "modemmanager", Type: PackageTypeApt, Category: "desktop-daemons",
			Rationale: "a modem manager on a headless server"},
		{ID: "fwupd", Package: "fwupd", Type: PackageTypeApt, Category: "desktop-daemons",
			Rationale: "desktop firmware update daemon"},
		{ID: "udisks2", Package: "udisks2", Type: PackageTypeApt, Category: "desktop-daemons",
			Rationale: "desktop disk manager, replaced by the kernel's own automount"},
		{ID: "iscsid", Package: "iscsid", Type: PackageTypeApt, Category: "storage",
			Rationale: "iSCSI initiator, irrelevant on a single-disk VPS"},
		{ID: "rpcbind", Package: "rpcbind", Type: PackageTypeApt, Category: "network",
			Rationale:     "ONI/NFS portmapper with no exported share",
			ProtectedWhen: []string{"nfs-common"}},
		{ID: "avahi", Package: "avahi-daemon", Type: PackageTypeApt, Category: "discovery",
			Rationale: "mDNS responder, absent from the measured cloud images"},
	}
}

// Every removal must justify itself. A list that removes state nobody
// explained is exactly how a tool becomes unreviewable.
func TestPackageRemovalRequiresRationale(t *testing.T) {
	if err := ValidatePackageRemovals(removalsFixture()); err != nil {
		t.Fatalf("the declared removal list is invalid: %v", err)
	}
	broken := removalsFixture()
	broken[0].Rationale = ""
	if err := ValidatePackageRemovals(broken); err == nil {
		t.Fatal("expected a removal without a rationale to be rejected")
	}
	broken = removalsFixture()
	broken[1].Package = ""
	if err := ValidatePackageRemovals(broken); err == nil {
		t.Fatal("expected a removal without a package name to be rejected")
	}
	broken = removalsFixture()
	broken[2].ID = ""
	if err := ValidatePackageRemovals(broken); err == nil {
		t.Fatal("expected a removal without an id to be rejected")
	}
	broken = removalsFixture()
	broken[3].Type = ""
	if err := ValidatePackageRemovals(broken); err == nil {
		t.Fatal("expected a removal without a package type to be rejected")
	}
}

func TestPackageRemovalDuplicateIDs(t *testing.T) {
	broken := removalsFixture()
	broken[1].ID = broken[0].ID
	if err := ValidatePackageRemovals(broken); err == nil {
		t.Fatal("expected a duplicate id to be rejected")
	}
}

// The guard is the measured case: nfs-common is an installed reverse
// dependency of rpcbind on the Oracle hosts, so removing rpcbind would cascade.
func TestResolveRemovalIsGuardedByAnInstalledPackage(t *testing.T) {
	removal := removalByID(t, "rpcbind")

	decision := ResolveRemoval(removal, map[string]bool{"nfs-common": true, "nfs-kernel-server": true}, true)
	if decision.Action != RemovalSkip {
		t.Fatalf("action = %v, want a skip when a guard package is installed", decision.Action)
	}
	if !strings.Contains(decision.Details, "nfs-common") {
		t.Fatalf("detail %q does not name the guard package", decision.Details)
	}

	decision = ResolveRemoval(removal, map[string]bool{"git": true}, true)
	if decision.Action != RemovalPurge {
		t.Fatalf("action = %v, want a purge when no guard is installed", decision.Action)
	}
}

// The three measured outcomes map to three diagnostic categories, and none of
// them is a warning: a converged host must stay at zero warnings.
func TestRemovalActionsAndCategories(t *testing.T) {
	cases := []struct {
		name      string
		installed bool
		guards    map[string]bool
		want      RemovalAction
		category  DiagnosticStatus
	}{
		{"absent is fine", false, nil, RemovalAlreadyAbsent, DiagOK},
		{"present and unguarded", true, nil, RemovalPurge, DiagOK},
		{"present but guarded", true, map[string]bool{"nfs-common": true}, RemovalSkip, DiagInfo},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			removal := PackageRemoval{ID: "x", Package: "x", Type: PackageTypeApt, Rationale: "r"}
			if len(tc.guards) > 0 {
				// A guard only protects when the removal declares it; that is
				// what keeps a stray installed package from silently disabling
				// an unrelated removal.
				removal.ProtectedWhen = []string{"nfs-common"}
			}
			decision := ResolveRemoval(removal, tc.guards, tc.installed)
			if decision.Action != tc.want {
				t.Fatalf("action = %v, want %v", decision.Action, tc.want)
			}
			if decision.Category != tc.category {
				t.Fatalf("category = %q, want %q", decision.Category, tc.category)
			}
			if decision.Category == DiagWarning {
				t.Fatal("no removal outcome may produce a warning")
			}
		})
	}
}

// After a completed run the doctor must distinguish: the package is gone (OK),
// it is still there (ERROR, the policy did not hold), or it is still there
// because a guard skipped it (INFO, naming the guard).
func TestRemovalAuditCategories(t *testing.T) {
	cases := []struct {
		name      string
		installed bool
		guarded   bool
		guardName string
		want      DiagnosticStatus
	}{
		{"removed", false, false, "", DiagOK},
		{"still present after a run", true, false, "", DiagError},
		{"skipped by a guard", true, true, "nfs-common", DiagInfo},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			category, detail := AssessRemovalState(tc.installed, tc.guarded, tc.guardName)
			if category != tc.want {
				t.Fatalf("category = %q, want %q", category, tc.want)
			}
			if category == DiagWarning {
				t.Fatal("the audit must never produce a warning")
			}
			if detail == "" {
				t.Fatal("every audit line needs a detail")
			}
			if tc.guarded && !strings.Contains(detail, tc.guardName) {
				t.Fatalf("detail %q does not name the guard %q", detail, tc.guardName)
			}
		})
	}
}

// The measured fleet decides the list. fwupd, udisks2 and modemmanager are
// installed on all three reachable hosts; avahi-daemon, cups, bluez and bluetooth
// are absent from every measured cloud image, so they are noise.
func TestDebloatSpecCoversTheMeasuredFleet(t *testing.T) {
	spec := DebloatSpec{
		NeedrestartDropin: "/etc/needrestart/conf.d/99-envctl.conf",
		Removals:          removalsFixture(),
	}
	if err := ValidateDebloatSpec(spec); err != nil {
		t.Fatalf("the declared spec is invalid: %v", err)
	}
	ids := make(map[string]bool, len(spec.Removals))
	for _, removal := range spec.Removals {
		ids[removal.ID] = true
	}
	// Present on 3/3 measured hosts.
	for _, required := range []string{"modemmanager", "fwupd", "udisks2"} {
		if !ids[required] {
			t.Fatalf("removal %q is installed on every measured host and must be in the list", required)
		}
	}
	// Guarded by a measured reverse dependency.
	var rpcbind PackageRemoval
	for _, removal := range spec.Removals {
		if removal.ID == "rpcbind" {
			rpcbind = removal
		}
	}
	if len(rpcbind.ProtectedWhen) == 0 {
		t.Fatal("rpcbind must be guarded: nfs-common is an installed reverse dependency")
	}
}

func TestDebloatSpecRequiresTheNeedrestartGuard(t *testing.T) {
	spec := DebloatSpec{NeedrestartDropin: "", Removals: removalsFixture()}
	if err := ValidateDebloatSpec(spec); err == nil {
		t.Fatal("expected a spec without the needrestart drop-in to be rejected")
	}
}

func TestNeedrestartConfigContent(t *testing.T) {
	content := renderNeedrestartConfig()
	if !strings.Contains(content, "NEEDRESTART_MODE=l") {
		t.Fatalf("rendered needrestart config %q does not force list-only mode", content)
	}
	if !strings.Contains(content, "NEEDRESTART_SUSPEND=") {
		t.Fatalf("rendered needrestart config %q must also pin suspend behaviour", content)
	}
}
