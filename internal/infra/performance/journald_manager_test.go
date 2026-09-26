package performance

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/eajdias/envctl/internal/domain/entity"
)

func journaldSpec() entity.JournaldSpec {
	return entity.JournaldSpec{
		Dropin: "/etc/systemd/journald.conf.d/90-envctl-journald.conf",
		Values: []entity.JournaldSetting{
			{Key: "SystemMaxUse", Value: "200M"},
			{Key: "SystemKeepFree", Value: "1G"},
			{Key: "RuntimeMaxUse", Value: "50M"},
			{Key: "MaxRetentionSec", Value: "2week"},
		},
	}
}

func TestRenderJournaldConfigSortsByKey(t *testing.T) {
	content, err := renderJournaldConfig(journaldSpec())
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}
	// The managed-by header comes first, matching the sysctl drop-in, and the
	// Journal section follows it.
	if !strings.HasPrefix(content, "# Managed by envctl") {
		t.Fatalf("rendered config is missing the managed-by header: %q", content)
	}
	if !strings.Contains(content, "\n[Journal]\n") {
		t.Fatalf("rendered config does not declare the Journal section: %q", content)
	}
	// Sorted lexicographically so the rendered file has a stable diff across
	// runs regardless of manifest key order.
	order := []string{"MaxRetentionSec", "RuntimeMaxUse", "SystemKeepFree", "SystemMaxUse"}
	previous := -1
	for _, key := range order {
		index := strings.Index(content, key+"=")
		if index < 0 {
			t.Fatalf("rendered config is missing %s: %q", key, content)
		}
		if index < previous {
			t.Fatalf("rendered config is not sorted: %q", content)
		}
		previous = index
	}
}

func TestRenderJournaldConfigRejectsUnknownKey(t *testing.T) {
	spec := journaldSpec()
	spec.Values = append(spec.Values, entity.JournaldSetting{Key: "Compress", Value: "yes"})
	if _, err := renderJournaldConfig(spec); err == nil {
		t.Fatal("expected an unknown journald key to be rejected")
	}
}

func TestRenderJournaldConfigRejectsInjectionAndEmptyValues(t *testing.T) {
	for _, bad := range []entity.JournaldSetting{
		{Key: "SystemMaxUse\nForwardToSyslog", Value: "1"},
		{Key: "SystemMaxUse", Value: "200M\nForwardToSyslog=yes"},
		{Key: "", Value: "200M"},
		{Key: "SystemMaxUse", Value: ""},
	} {
		spec := entity.JournaldSpec{
			Dropin: "/tmp/90-journald.conf",
			Values: []entity.JournaldSetting{bad},
		}
		if _, err := renderJournaldConfig(spec); err == nil {
			t.Fatalf("expected %q to be rejected", bad)
		}
	}
}

// man 8 systemd-journald: restarting preserves the client stream connections,
// while stopping the service terminates them. The apply sequence must therefore
// be a single restart and must never stop.
func TestJournaldManagerRestartsAndNeverStops(t *testing.T) {
	dir := t.TempDir()
	dropin := filepath.Join(dir, "90-envctl-journald.conf")
	var calls [][]string
	manager := newJournaldManager(newDropinWriter(dropin, fakeDropinFS(t, nil, &calls),
		func() time.Time { return time.Unix(1, 0) }, false))

	if _, err := manager.Apply(context.Background(), journaldSpec(), false); err != nil {
		t.Fatalf("apply failed: %v", err)
	}
	sawRestart := false
	for _, call := range calls {
		joined := strings.Join(call, " ")
		if strings.Contains(joined, "stop") {
			t.Fatalf("journald apply issued a stop: %#v", calls)
		}
		if len(call) >= 2 && call[0] == "systemctl" && call[1] == "restart" && call[2] == "systemd-journald" {
			sawRestart = true
		}
	}
	if !sawRestart {
		t.Fatalf("commands = %#v, want systemctl restart systemd-journald", calls)
	}
}

func TestJournaldManagerIdenticalContentIsNoop(t *testing.T) {
	dir := t.TempDir()
	dropin := filepath.Join(dir, "90-envctl-journald.conf")
	content, err := renderJournaldConfig(journaldSpec())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dropin, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	var calls [][]string
	manager := newJournaldManager(newDropinWriter(dropin, fakeDropinFS(t, nil, &calls),
		func() time.Time { return time.Unix(1, 0) }, false))

	diags, err := manager.Apply(context.Background(), journaldSpec(), false)
	if err != nil {
		t.Fatalf("no-op failed: %v", err)
	}
	if len(calls) != 0 {
		t.Fatalf("identical content invoked %#v", calls)
	}
	if len(diags) != 1 || !strings.Contains(diags[0].Details, "already up to date") {
		t.Fatalf("diagnostics = %#v, want a no-op diagnostic", diags)
	}
}

func TestJournaldManagerReportsRestartFailure(t *testing.T) {
	dir := t.TempDir()
	dropin := filepath.Join(dir, "90-envctl-journald.conf")
	// A pre-existing drop-in is what produces a backup, which is the case an
	// operator needs to be able to roll back to after a failed restart.
	if err := os.WriteFile(dropin, []byte("[Journal]\nSystemMaxUse=999M\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var calls [][]string
	manager := newJournaldManager(newDropinWriter(dropin,
		fakeDropinFS(t, map[string]error{"systemctl": errors.New("synthetic restart failure")}, &calls),
		func() time.Time { return time.Unix(1, 0) }, false))

	diags, err := manager.Apply(context.Background(), journaldSpec(), false)
	if err == nil {
		t.Fatal("expected a restart failure to be reported")
	}
	if len(diags) != 1 || diags[0].Category != entity.DiagError {
		t.Fatalf("diagnostics = %#v, want an error", diags)
	}
	if !strings.Contains(diags[0].Details, "backup=") {
		t.Fatalf("diagnostic %q does not point at the recoverable backup", diags[0].Details)
	}
}

func TestJournaldManagerDryRunWritesNothing(t *testing.T) {
	dir := t.TempDir()
	dropin := filepath.Join(dir, "90-envctl-journald.conf")
	called := false
	manager := newJournaldManager(newDropinWriter(dropin,
		func(context.Context, string, ...string) ([]byte, error) { called = true; return nil, nil },
		func() time.Time { return time.Unix(1, 0) }, false))

	diags, err := manager.Apply(context.Background(), journaldSpec(), true)
	if err != nil {
		t.Fatalf("dry-run failed: %v", err)
	}
	if called {
		t.Fatal("dry-run invoked a command")
	}
	if len(diags) != 1 || diags[0].Category != entity.DiagInfo {
		t.Fatalf("diagnostics = %#v, want one INFO", diags)
	}
	if _, err := os.Stat(dropin); !os.IsNotExist(err) {
		t.Fatal("dry-run created the drop-in")
	}
}

// The doctor must judge the host against the effective floor. journald honours
// the smaller of SystemMaxUse and SystemKeepFree, so a host that declares only
// one of them is not necessarily unbounded, and the check must not turn that
// into a permanent warning.
func TestJournaldPolicyAssessment(t *testing.T) {
	cases := []struct {
		name         string
		state        entity.JournaldState
		wantCapped   bool
		wantCategory entity.DiagnosticStatus
	}{
		{
			name:         "capped with a keep-free floor",
			state:        entity.JournaldState{SystemMaxUse: "200M", SystemKeepFree: "1G"},
			wantCapped:   true,
			wantCategory: entity.DiagOK,
		},
		{
			name:         "capped without a keep-free floor is still capped",
			state:        entity.JournaldState{SystemMaxUse: "200M"},
			wantCapped:   true,
			wantCategory: entity.DiagOK,
		},
		{
			name:         "uncapped host is information, not a warning",
			state:        entity.JournaldState{},
			wantCapped:   false,
			wantCategory: entity.DiagInfo,
		},
		{
			name:         "only keep-free without a max use does not bound the journal",
			state:        entity.JournaldState{SystemKeepFree: "1G"},
			wantCapped:   false,
			wantCategory: entity.DiagInfo,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			capped, category, detail := AssessJournaldPolicy(tc.state)
			if capped != tc.wantCapped {
				t.Fatalf("capped = %v, want %v (state %#v)", capped, tc.wantCapped, tc.state)
			}
			if category != tc.wantCategory {
				t.Fatalf("category = %q, want %q", category, tc.wantCategory)
			}
			if !strings.Contains(detail, "10%") && category == entity.DiagInfo {
				t.Fatalf("uncapped detail %q does not state the 10%%-of-filesystem default", detail)
			}
		})
	}
}
