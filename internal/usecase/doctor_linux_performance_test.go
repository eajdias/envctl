package usecase

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/eajdias/envctl"
	"github.com/eajdias/envctl/internal/domain/entity"
	"github.com/eajdias/envctl/internal/infra/embedded"
)

func TestAuditLinuxPerformanceDoesNotWarnOnOptionalState(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux performance audit is Linux-only")
	}

	uc := &DoctorAuditUseCase{}
	snapshot := entity.PerformanceSnapshot{
		Swap:            nil,
		ZRAM:            entity.ZRAMState{},
		CPUGovernors:    []string{"schedutil"},
		BlockSchedulers: []entity.BlockScheduler{{Device: "nvme0n1", Selected: "kyber"}},
	}
	var diagnostics []entity.Diagnostic
	uc.auditLinuxPerformance(snapshot, func(diagnostic entity.Diagnostic) {
		diagnostics = append(diagnostics, diagnostic)
	})

	if len(diagnostics) == 0 {
		t.Fatal("expected performance diagnostics")
	}
	for _, diagnostic := range diagnostics {
		if diagnostic.Category == entity.DiagWarning || diagnostic.Category == entity.DiagError {
			t.Fatalf("optional performance state produced %s: %s", diagnostic.Category, diagnostic.Details)
		}
	}
}

// TestAuditLinuxPerformancePacnewIsNeverWarning pins the severity: a pending
// .pacnew is advisory. This workstation has one, and a warning here would
// break the 0 WARN/0 ERROR contract for a non-defect.
func TestAuditLinuxPerformancePacnewIsNeverWarning(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux performance audit is Linux-only")
	}

	uc := &DoctorAuditUseCase{}
	snapshot := entity.PerformanceSnapshot{
		ZRAM: entity.ZRAMState{Present: true, Name: "/dev/zram0", Algorithm: "zstd", SizeBytes: 22_400_000_000},
	}
	var diagnostics []entity.Diagnostic
	uc.auditLinuxPerformance(snapshot, func(diagnostic entity.Diagnostic) {
		diagnostics = append(diagnostics, diagnostic)
	})

	found := false
	for _, diagnostic := range diagnostics {
		if diagnostic.Target != "pacnew" {
			continue
		}
		found = true
		if diagnostic.Category == entity.DiagWarning || diagnostic.Category == entity.DiagError {
			t.Errorf("pacnew must never be %s: %s", diagnostic.Category, diagnostic.Details)
		}
	}
	if !found {
		t.Fatalf("expected a pacnew diagnostic, got %d diagnostics in total", len(diagnostics))
	}
}

func TestAuditLinuxPerformanceReportsActiveZRAM(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux performance audit is Linux-only")
	}

	uc := &DoctorAuditUseCase{}
	snapshot := entity.PerformanceSnapshot{
		ZRAM: entity.ZRAMState{Present: true, Name: "/dev/zram0", Algorithm: "zstd", SizeBytes: 22_400_000_000},
	}
	var diagnostics []entity.Diagnostic
	uc.auditLinuxPerformance(snapshot, func(diagnostic entity.Diagnostic) {
		diagnostics = append(diagnostics, diagnostic)
	})

	found := false
	for _, diagnostic := range diagnostics {
		if diagnostic.Target == "zram" && diagnostic.Category == entity.DiagOK {
			found = true
		}
	}
	if !found {
		t.Fatalf("active zram was not reported as OK: %#v", diagnostics)
	}
}

// TestAuditLinuxPerformanceReportsAHostDropInThatWins is the end-to-end shape of
// the defect a reboot exposed: the profile derives vm.swappiness from the
// measured topology, the host's own file wins at boot, and the audit used to say
// nothing because it never compared the intent against the resolved drop-ins.
func TestAuditLinuxPerformanceReportsAHostDropInThatWins(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux performance audit is Linux-only")
	}

	dir := t.TempDir()
	manifestsDir := filepath.Join(dir, "manifests")
	if err := os.MkdirAll(manifestsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	fixture := `profile: ubuntu-server
sysctls:
  - key: net.core.somaxconn
    value: "65535"
tiers:
  - id: small
    mem_total_max_mib: 1024
    rationale: "derived: the test host measures 951 MiB"
    sysctls:
      - key: vm.vfs_cache_pressure
        value: "50"
`
	if err := os.WriteFile(filepath.Join(manifestsDir, "performance_ubuntu.yaml"), []byte(fixture), 0o644); err != nil {
		t.Fatal(err)
	}

	uc := &DoctorAuditUseCase{
		manifestRepo: embedded.NewManifestRepository(envctl.EmbeddedFS, dir),
		platform: func() entity.PlatformInfo {
			return entity.PlatformInfo{GOOS: "linux", Family: "debian", ID: "ubuntu", VersionID: "26.04"}
		},
	}
	snapshot := entity.PerformanceSnapshot{
		MemoryKB: 974092,
		Swap: []entity.SwapDevice{
			{Name: "/swapfile", Type: "file", SizeKB: 8388608, Priority: -1},
			{Name: "/dev/zram0", Type: "partition", SizeKB: 486912, Priority: 100},
		},
		ZRAM: entity.ZRAMState{Present: true, Name: "/dev/zram0", Priority: 100},
		Sysctls: []entity.SysctlAssignment{
			{Key: "vm.swappiness", File: "/etc/sysctl.d/99-swappiness.conf", Boot: "10", Live: "10"},
			{
				Key: "vm.vfs_cache_pressure", File: "/etc/sysctl.d/90-envctl-performance.conf",
				Boot: "50", Live: "50", Managed: true,
			},
			{
				Key: "net.core.somaxconn", File: "/etc/sysctl.d/90-envctl-performance.conf",
				Boot: "65535", Live: "65535", Managed: true,
			},
		},
	}

	var diagnostics []entity.Diagnostic
	uc.auditLinuxPerformance(snapshot, func(diagnostic entity.Diagnostic) {
		diagnostics = append(diagnostics, diagnostic)
	})

	shadow, ok := intentKey(diagnostics, "vm.swappiness")
	if !ok {
		t.Fatal("the shadowed key was not reported by the audit")
	}
	if shadow.Category != entity.DiagWarning {
		t.Fatalf("category %v, want a warning: %q", shadow.Category, shadow.Details)
	}
	if !strings.Contains(shadow.Details, "99-swappiness.conf") {
		t.Fatalf("the audit does not name the winning file: %q", shadow.Details)
	}
	// The profile's own keys must be reported as satisfied, not as warnings.
	for _, key := range []string{"vm.vfs_cache_pressure", "net.core.somaxconn"} {
		diag, found := intentKey(diagnostics, key)
		if !found {
			t.Fatalf("the profile's own key %s was not audited", key)
		}
		if diag.Category == entity.DiagWarning {
			t.Fatalf("the profile's own key %s was reported as a warning: %q", key, diag.Details)
		}
	}
}

// A host that is not this profile's OS has no sysctl intent to audit, and the
// audit must stay silent rather than load a manifest it does not target.
func TestAuditLinuxPerformanceSkipsTheIntentOnAnotherOS(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux performance audit is Linux-only")
	}

	uc := &DoctorAuditUseCase{
		manifestRepo: embedded.NewManifestRepository(envctl.EmbeddedFS, ""),
		platform: func() entity.PlatformInfo {
			return entity.PlatformInfo{GOOS: "linux", Family: "arch", ID: "cachyos"}
		},
	}
	snapshot := entity.PerformanceSnapshot{
		Sysctls: []entity.SysctlAssignment{{Key: "vm.swappiness", File: "/x.conf", Boot: "100"}},
	}

	var diagnostics []entity.Diagnostic
	uc.auditLinuxPerformance(snapshot, func(diagnostic entity.Diagnostic) {
		diagnostics = append(diagnostics, diagnostic)
	})
	if _, ok := intentKey(diagnostics, "vm.swappiness"); ok {
		t.Fatal("the sysctl intent was audited on a host the profile does not target")
	}
}
