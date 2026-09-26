package usecase

import (
	"strings"
	"testing"

	"github.com/eajdias/envctl/internal/domain/entity"
)

func intentKey(diags []entity.Diagnostic, key string) (entity.Diagnostic, bool) {
	for _, diag := range diags {
		if diag.Target == key {
			return diag, true
		}
	}
	return entity.Diagnostic{}, false
}

// TestAssessSysctlIntentReportsAHostDropInThatWins is the regression test for the
// defect a reboot exposed on a real host: the fleet ships
// /etc/sysctl.d/99-swappiness.conf, written by the cloud agent, which sorts
// after the profile's 90-envctl-performance.conf. The profile derives
// vm.swappiness=150 from the measured zram topology, the run writes 150 and sets
// it live, and the next boot silently restores 10 with nothing reported
// anywhere.
//
// The audit has to compare the profile's intent against what the host will
// actually apply, not against what the profile wrote.
func TestAssessSysctlIntentReportsAHostDropInThatWins(t *testing.T) {
	diags := assessSysctlIntent(
		[]entity.SysctlSetting{
			{Key: "vm.swappiness", Value: "150", Policy: entity.SysctlPolicySet},
		},
		[]entity.SysctlAssignment{{
			Key:     "vm.swappiness",
			File:    "/etc/sysctl.d/99-swappiness.conf",
			Boot:    "10",
			Live:    "10",
			Managed: false,
		}},
	)

	diag, ok := intentKey(diags, "vm.swappiness")
	if !ok {
		t.Fatal("a key the host overrides produced no diagnostic")
	}
	if diag.Category != entity.DiagWarning {
		t.Fatalf("category %v, want a warning: the owner must see the disagreement", diag.Category)
	}
	for _, want := range []string{"99-swappiness.conf", "10", "150", "boot"} {
		if !strings.Contains(diag.Details, want) {
			t.Fatalf("the diagnostic is missing %q: %q", want, diag.Details)
		}
	}
	if !strings.Contains(diag.FixHint, "99-swappiness.conf") {
		t.Fatalf("the fix hint does not name the file to deal with: %q", diag.FixHint)
	}
}

// A key the profile owns and that is in effect is a plain OK.
func TestAssessSysctlIntentAcceptsTheProfilesOwnDropIn(t *testing.T) {
	diags := assessSysctlIntent(
		[]entity.SysctlSetting{
			{Key: "net.core.somaxconn", Value: "65535", Policy: entity.SysctlPolicySet},
		},
		[]entity.SysctlAssignment{{
			Key:     "net.core.somaxconn",
			File:    "/etc/sysctl.d/90-envctl-performance.conf",
			Boot:    "65535",
			Live:    "65535",
			Managed: true,
		}},
	)

	diag, ok := intentKey(diags, "net.core.somaxconn")
	if !ok {
		t.Fatal("a converged key produced no diagnostic")
	}
	if diag.Category != entity.DiagOK {
		t.Fatalf("category %v, want OK: %q", diag.Category, diag.Details)
	}
}

// The profile's file says one thing and the kernel holds another: something
// changed the value after the run, and the next boot would put it back. That
// drift must be visible.
func TestAssessSysctlIntentReportsDriftFromTheProfilesOwnValue(t *testing.T) {
	diags := assessSysctlIntent(
		[]entity.SysctlSetting{
			{Key: "vm.vfs_cache_pressure", Value: "50", Policy: entity.SysctlPolicySet},
		},
		[]entity.SysctlAssignment{{
			Key:     "vm.vfs_cache_pressure",
			File:    "/etc/sysctl.d/90-envctl-performance.conf",
			Boot:    "50",
			Live:    "100",
			Managed: true,
		}},
	)

	diag, ok := intentKey(diags, "vm.vfs_cache_pressure")
	if !ok {
		t.Fatal("drift from the profile's own value produced no diagnostic")
	}
	if diag.Category != entity.DiagWarning {
		t.Fatalf("category %v, want a warning: %q", diag.Category, diag.Details)
	}
	if !strings.Contains(diag.Details, "100") || !strings.Contains(diag.Details, "50") {
		t.Fatalf("the diagnostic must carry the live and declared values, got %q", diag.Details)
	}
}

// A min-policy key whose host value is already better is satisfied, so there is
// nothing for the owner to decide and the report stays silent. Saying OK instead
// would be noise: fs.file-max ships at the int64 ceiling on both clouds, and an
// audit line per run for a key the profile will never touch trains the reader to
// skip the report.
func TestAssessSysctlIntentHonoursTheMinPolicy(t *testing.T) {
	diags := assessSysctlIntent(
		[]entity.SysctlSetting{
			{Key: "fs.file-max", Value: "2097152", Policy: entity.SysctlPolicyMin},
		},
		[]entity.SysctlAssignment{{
			Key:     "fs.file-max",
			File:    "/usr/lib/sysctl.d/50-default.conf",
			Boot:    "9223372036854775807",
			Live:    "9223372036854775807",
			Managed: false,
		}},
	)

	if diag, ok := intentKey(diags, "fs.file-max"); ok {
		t.Fatalf("a satisfied min-policy key was reported: %+v", diag)
	}
}

// A min-policy key the host holds *below* the declared floor is a real gap even
// though the winning file is somebody else's.
func TestAssessSysctlIntentReportsAMinPolicyBelowTheFloor(t *testing.T) {
	diags := assessSysctlIntent(
		[]entity.SysctlSetting{
			{Key: "fs.file-max", Value: "2097152", Policy: entity.SysctlPolicyMin},
		},
		[]entity.SysctlAssignment{{
			Key:     "fs.file-max",
			File:    "/etc/sysctl.d/99-host.conf",
			Boot:    "1024",
			Live:    "1024",
			Managed: false,
		}},
	)

	diag, ok := intentKey(diags, "fs.file-max")
	if !ok {
		t.Fatal("a min-policy key below its floor produced no diagnostic")
	}
	if diag.Category != entity.DiagWarning {
		t.Fatalf("category %v, want a warning: %q", diag.Category, diag.Details)
	}
	if !strings.Contains(diag.Details, "1024") {
		t.Fatalf("the diagnostic does not carry the host's value: %q", diag.Details)
	}
}

// A min-policy key with no drop-in is not a gap. The policy exists so the host's
// own value stands, so demanding a file that pins a floor the host already
// clears would report a warning on every fleet host that never had one — noise
// that trains the reader to ignore the line that matters.
func TestAssessSysctlIntentDoesNotDemandADropInForAMinPolicyTheHostClears(t *testing.T) {
	diags := assessSysctlIntent(
		[]entity.SysctlSetting{
			{Key: "fs.file-max", Value: "2097152", Policy: entity.SysctlPolicyMin},
		},
		[]entity.SysctlAssignment{
			{Key: "net.core.somaxconn", File: "/etc/sysctl.d/90-envctl-performance.conf", Boot: "65535", Managed: true},
		},
	)

	if diag, ok := intentKey(diags, "fs.file-max"); ok {
		t.Fatalf("a satisfied min-policy key with no drop-in was reported: %+v", diag)
	}
}

// No drop-in at all means nothing will restore the value after a reboot, which
// is drift even when the running kernel currently holds the right number.
func TestAssessSysctlIntentReportsAKeyNoFileRestores(t *testing.T) {
	diags := assessSysctlIntent(
		[]entity.SysctlSetting{
			{Key: "net.ipv4.tcp_max_syn_backlog", Value: "4096", Policy: entity.SysctlPolicySet},
		},
		nil,
	)

	diag, ok := intentKey(diags, "net.ipv4.tcp_max_syn_backlog")
	if !ok {
		t.Fatal("a key no file declares produced no diagnostic")
	}
	if diag.Category != entity.DiagWarning {
		t.Fatalf("category %v, want a warning: %q", diag.Category, diag.Details)
	}
	if !strings.Contains(diag.Details, "reboot") {
		t.Fatalf("the diagnostic must say the value does not survive a reboot: %q", diag.Details)
	}
}

// An unreadable /proc/sys entry must not be reported as drift: the audit
// reports what it could not see as unknown, not as wrong.
func TestAssessSysctlIntentDoesNotInventADriftForAnUnreadableKey(t *testing.T) {
	diags := assessSysctlIntent(
		[]entity.SysctlSetting{
			{Key: "vm.swappiness", Value: "150", Policy: entity.SysctlPolicySet},
		},
		[]entity.SysctlAssignment{{
			Key:     "vm.swappiness",
			File:    "/etc/sysctl.d/90-envctl-performance.conf",
			Boot:    "150",
			Managed: true,
		}},
	)

	diag, ok := intentKey(diags, "vm.swappiness")
	if !ok {
		t.Fatal("the key produced no diagnostic")
	}
	if diag.Category == entity.DiagWarning {
		t.Fatalf("an unreadable live value was reported as drift: %q", diag.Details)
	}
}

// A host file that happens to agree with the profile is not a decision the owner
// has to make, so it stays out of the report entirely.
func TestAssessSysctlIntentIgnoresAnAgreeingHostDropIn(t *testing.T) {
	diags := assessSysctlIntent(
		[]entity.SysctlSetting{
			{Key: "vm.swappiness", Value: "150", Policy: entity.SysctlPolicySet},
		},
		[]entity.SysctlAssignment{{
			Key:     "vm.swappiness",
			File:    "/etc/sysctl.d/99-swappiness.conf",
			Boot:    "150",
			Live:    "150",
			Managed: false,
		}},
	)

	if _, ok := intentKey(diags, "vm.swappiness"); ok {
		t.Fatal("an agreeing host file was reported; there is nothing to decide")
	}
}

// A profile with no sysctl intent at all (CachyOS) must produce nothing rather
// than a line per key the audit happened to resolve.
func TestAssessSysctlIntentOnAnEmptyIntent(t *testing.T) {
	if diags := assessSysctlIntent(nil, []entity.SysctlAssignment{
		{Key: "vm.swappiness", File: "/usr/lib/sysctl.d/70-x.conf", Boot: "100"},
	}); len(diags) != 0 {
		t.Fatalf("an empty intent produced %d diagnostics: %+v", len(diags), diags)
	}
}
