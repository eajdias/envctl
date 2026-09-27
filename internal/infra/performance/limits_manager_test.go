package performance

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/eajdias/envctl/internal/domain/entity"
)

func limitsSpec() entity.LimitsSpec {
	return entity.LimitsSpec{
		SystemDropin: "/etc/systemd/system.conf.d/90-envctl-limits.conf",
		PAMDropin:    "/etc/security/limits.d/90-envctl-limits.conf",
		NofileSoft:   65536,
		NofileHard:   "",
		Nproc:        32768,
		Reexec:       true,
	}
}

func newLimitsManagerForTest(t *testing.T, spec entity.LimitsSpec, calls *[][]string, changedOnFirstWrite bool) *limitsManager {
	t.Helper()
	dir := t.TempDir()
	systemDropin := filepath.Join(dir, "90-envctl-limits.conf")
	pamDropin := filepath.Join(dir, "90-envctl-limits.pam.conf")
	if changedOnFirstWrite {
		// Pre-create both so the writer takes the "changed" path and a backup
		// exists, which is the case that triggers the re-exec.
		if err := os.WriteFile(systemDropin, []byte("[Manager]\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(pamDropin, []byte("# previous\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return newLimitsManager(
		newDropinWriter(systemDropin, fakeDropinFS(t, nil, calls), func() time.Time { return time.Unix(1, 0) }, false),
		newDropinWriter(pamDropin, fakeDropinFS(t, nil, nil), func() time.Time { return time.Unix(1, 0) }, false),
		entity.LimitsSpec{
			SystemDropin: spec.SystemDropin,
			PAMDropin:    spec.PAMDropin,
		},
	)
}

// DefaultLimitNOFILE must be rendered as a soft value with an EMPTY hard side.
//
// Measured on the fleet: OCI reports a hard limit of 524288 and AWS reports
// 1048576. Writing "65536:524288" would lower the AWS host, so the manifest
// keeps the host's own hard limit by leaving the second field empty. The form
// "65536:" was probed with systemd-analyze cat-config and parses clean.
func TestRenderSystemLimitsKeepsAnEmptyHardSide(t *testing.T) {
	content, err := renderSystemLimits(entity.LimitsSpec{NofileSoft: 65536, Nproc: 32768})
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}
	if !strings.Contains(content, "DefaultLimitNOFILE=65536:\n") {
		t.Fatalf("DefaultLimitNOFILE must keep an empty hard side: %q", content)
	}
	if strings.Contains(content, "DefaultLimitNOFILE=65536:524288") {
		t.Fatalf("rendered a hard limit that would lower the fleet's AWS host: %q", content)
	}
	if !strings.Contains(content, "[Manager]") || !strings.Contains(content, "DefaultLimitNPROC=32768") {
		t.Fatalf("rendered config is incomplete: %q", content)
	}
}

// An explicitly declared hard value is honoured, so a host that needs a lower
// ceiling can still express it.
func TestRenderSystemLimitsHonoursAnExplicitHardValue(t *testing.T) {
	content, err := renderSystemLimits(entity.LimitsSpec{NofileSoft: 4096, NofileHard: "8192"})
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}
	if !strings.Contains(content, "DefaultLimitNOFILE=4096:8192\n") {
		t.Fatalf("explicit hard value was not rendered: %q", content)
	}
}

// The PAM drop-in must never set a hard limit: PAM's hard would override the
// value the host already inherits, which is the same regression the systemd
// side avoids by leaving it empty.
func TestRenderPAMLimitsNeverSetsAHardLimit(t *testing.T) {
	content, err := renderPAMLimits(entity.LimitsSpec{NofileSoft: 65536, Nproc: 32768})
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}
	if strings.Contains(content, "hard") {
		t.Fatalf("the PAM drop-in must not declare a hard limit: %q", content)
	}
	if !strings.Contains(content, "* soft nofile 65536") || !strings.Contains(content, "* soft nproc 32768") {
		t.Fatalf("PAM soft limits are missing: %q", content)
	}
}

// man 1 systemctl: daemon-reload only reruns generators and reloads unit files,
// so it does not re-read system.conf. daemon-reexec re-reads it, and the same
// page documents that all sockets stay accessible during the re-exec, which is
// what makes it acceptable over SSH.
func TestLimitsManagerReexecsOnlyWhenADropInChanged(t *testing.T) {
	var calls [][]string
	manager := newLimitsManagerForTest(t, limitsSpec(), &calls, true)

	if _, err := manager.Apply(context.Background(), limitsSpec(), false); err != nil {
		t.Fatalf("apply failed: %v", err)
	}
	sawReexec, sawReload := false, false
	for _, call := range calls {
		if len(call) >= 2 && call[0] == "systemctl" && call[1] == "daemon-reexec" {
			sawReexec = true
		}
		if len(call) >= 2 && call[0] == "systemctl" && call[1] == "daemon-reload" {
			sawReload = true
		}
	}
	if !sawReexec {
		t.Fatalf("commands = %#v, want a daemon-reexec so system.conf is re-read", calls)
	}
	if !sawReload {
		t.Fatalf("commands = %#v, want a daemon-reload before the re-exec", calls)
	}
}

func TestLimitsManagerSkipsReexecWhenNothingChanged(t *testing.T) {
	dir := t.TempDir()
	systemDropin := filepath.Join(dir, "90-envctl-limits.conf")
	pamDropin := filepath.Join(dir, "90-envctl-limits.pam.conf")
	spec := limitsSpec()

	// Render the exact content the manager will write, so the second run is a
	// genuine no-op.
	systemContent, err := renderSystemLimits(spec)
	if err != nil {
		t.Fatal(err)
	}
	pamContent, err := renderPAMLimits(spec)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(systemDropin, []byte(systemContent), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pamDropin, []byte(pamContent), 0o644); err != nil {
		t.Fatal(err)
	}

	var calls [][]string
	manager := newLimitsManager(
		newDropinWriter(systemDropin, fakeDropinFS(t, nil, &calls), func() time.Time { return time.Unix(1, 0) }, false),
		newDropinWriter(pamDropin, fakeDropinFS(t, nil, nil), func() time.Time { return time.Unix(1, 0) }, false),
		spec,
	)

	diags, err := manager.Apply(context.Background(), spec, false)
	if err != nil {
		t.Fatalf("no-op apply failed: %v", err)
	}
	for _, call := range calls {
		if len(call) >= 2 && call[0] == "systemctl" {
			t.Fatalf("an unchanged drop-in still triggered %v: %#v", call, calls)
		}
	}
	allOK := true
	for _, diag := range diags {
		if diag.Category != entity.DiagOK {
			allOK = false
		}
	}
	if !allOK || len(diags) == 0 {
		t.Fatalf("diagnostics = %#v, want OK on a converged host", diags)
	}
}

func TestLimitsManagerHonoursTheReexecOptOut(t *testing.T) {
	spec := limitsSpec()
	spec.Reexec = false
	var calls [][]string
	manager := newLimitsManagerForTest(t, spec, &calls, true)

	if _, err := manager.Apply(context.Background(), spec, false); err != nil {
		t.Fatalf("apply failed: %v", err)
	}
	for _, call := range calls {
		if len(call) >= 2 && call[0] == "systemctl" && call[1] == "daemon-reexec" {
			t.Fatalf("the opt-out still re-executed PID 1: %#v", calls)
		}
	}
}

// man 5 systemd.exec warns that raising the soft limit above 1024 breaks
// select(2) in software that still uses it, so the operator is told on every
// apply rather than discovering it in production.
func TestLimitsManagerDiagnosticQuotesTheSelectCaveat(t *testing.T) {
	var calls [][]string
	manager := newLimitsManagerForTest(t, limitsSpec(), &calls, true)

	diags, err := manager.Apply(context.Background(), limitsSpec(), false)
	if err != nil {
		t.Fatalf("apply failed: %v", err)
	}
	found := false
	for _, diag := range diags {
		if strings.Contains(diag.Details, "select(2)") {
			found = true
		}
	}
	if !found {
		t.Fatalf("no diagnostic mentions the select(2) caveat: %#v", diags)
	}
}

func TestLimitsManagerDoesNotWarnBelowTheSelectThreshold(t *testing.T) {
	spec := limitsSpec()
	spec.NofileSoft = 1024
	var calls [][]string
	manager := newLimitsManagerForTest(t, spec, &calls, true)

	diags, err := manager.Apply(context.Background(), spec, false)
	if err != nil {
		t.Fatalf("apply failed: %v", err)
	}
	for _, diag := range diags {
		if strings.Contains(diag.Details, "select(2)") {
			t.Fatalf("the caveat was reported at the safe threshold: %#v", diags)
		}
	}
}

func TestLimitsManagerDryRunWritesNothing(t *testing.T) {
	called := false
	dir := t.TempDir()
	manager := newLimitsManager(
		newDropinWriter(filepath.Join(dir, "a.conf"),
			func(context.Context, string, ...string) ([]byte, error) { called = true; return nil, nil },
			func() time.Time { return time.Unix(1, 0) }, false),
		newDropinWriter(filepath.Join(dir, "b.conf"),
			func(context.Context, string, ...string) ([]byte, error) { called = true; return nil, nil },
			func() time.Time { return time.Unix(1, 0) }, false),
		limitsSpec(),
	)

	diags, err := manager.Apply(context.Background(), limitsSpec(), true)
	if err != nil {
		t.Fatalf("dry-run failed: %v", err)
	}
	if called {
		t.Fatal("dry-run invoked a command")
	}
	if len(diags) != 1 || diags[0].Category != entity.DiagInfo {
		t.Fatalf("diagnostics = %#v, want one INFO", diags)
	}
}

func TestLimitsManagerRejectsInvalidValues(t *testing.T) {
	for _, spec := range []entity.LimitsSpec{
		{NofileSoft: 0},
		{NofileSoft: -1},
		{NofileSoft: 65536, NofileHard: "not-a-number"},
		{NofileSoft: 65536, NofileHard: "65536\nForwardToSyslog=yes"},
	} {
		if _, err := renderSystemLimits(spec); err == nil {
			t.Fatalf("expected %#v to be rejected", spec)
		}
	}
}
