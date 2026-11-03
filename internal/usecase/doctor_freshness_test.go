package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/eajdias/envctl/internal/domain/entity"
)

func collectFreshnessDiags(uc *DoctorAuditUseCase) []entity.Diagnostic {
	var diags []entity.Diagnostic
	uc.auditEnvctlFreshness(context.Background(), func(d entity.Diagnostic) { diags = append(diags, d) })
	return diags
}

func withNoRepo(t *testing.T) {
	t.Helper()
	old := repoRootFinder
	t.Cleanup(func() { repoRootFinder = old })
	repoRootFinder = func() (string, error) { return "", errors.New("no .git above") }
}

// A dev binary outside any checkout has nothing to compare against: the
// audit records the unknown version as INFO instead of staying silent.
func TestAuditEnvctlFreshnessInfoForDevOutsideCheckout(t *testing.T) {
	withNoRepo(t)
	uc := &DoctorAuditUseCase{
		envctlVersion: "dev",
		gitDescribe: func(_ context.Context, _ string) (string, error) {
			return "v9.9.9", nil
		},
	}
	diags := collectFreshnessDiags(uc)
	if len(diags) != 1 {
		t.Fatalf("want exactly the dev INFO, got %+v", diags)
	}
	d := diags[0]
	if d.Category != entity.DiagInfo || d.System != "Envctl" || d.Target != "Binary freshness" {
		t.Errorf("unexpected diagnostic: %+v", d)
	}
}

// The same silence, but for a versioned binary: without a checkout there is
// nothing to judge, and without --check-updates there is no release lookup.
func TestAuditEnvctlFreshnessSilentForReleaseOutsideCheckout(t *testing.T) {
	withNoRepo(t)
	uc := &DoctorAuditUseCase{
		envctlVersion: "v2.1.5",
		gitDescribe: func(_ context.Context, _ string) (string, error) {
			return "v9.9.9", nil
		},
	}
	if diags := collectFreshnessDiags(uc); len(diags) != 0 {
		t.Errorf("versioned binary outside checkout stays silent, got %+v", diags)
	}
}

// --check-updates with a fake release behind the binary: WARN naming
// self-update. The repo seam fails so only the release check speaks.
func TestAuditEnvctlFreshnessWarnsBehindLatest(t *testing.T) {
	withNoRepo(t)
	uc := &DoctorAuditUseCase{
		envctlVersion: "v2.1.5",
		gitDescribe: func(_ context.Context, _ string) (string, error) {
			return "", errors.New("no checkout")
		},
		checkUpdates: true,
		latestRelease: func(_ context.Context) (string, error) {
			return "v9.9.9", nil
		},
	}
	diags := collectFreshnessDiags(uc)
	if len(diags) != 1 {
		t.Fatalf("want exactly the release WARN, got %+v", diags)
	}
	d := diags[0]
	if d.Category != entity.DiagWarning || d.Target != "Release freshness" {
		t.Errorf("unexpected diagnostic: %+v", d)
	}
	if got := d.FixHint; got != "run 'envctl self-update'" {
		t.Errorf("fix hint = %q, want the self-update command", got)
	}
}

// --check-updates at the latest release: OK, proving the lookup ran.
func TestAuditEnvctlFreshnessOKAtLatest(t *testing.T) {
	withNoRepo(t)
	uc := &DoctorAuditUseCase{
		envctlVersion: "v9.9.9",
		gitDescribe: func(_ context.Context, _ string) (string, error) {
			return "", errors.New("no checkout")
		},
		checkUpdates: true,
		latestRelease: func(_ context.Context) (string, error) {
			return "v9.9.9", nil
		},
	}
	diags := collectFreshnessDiags(uc)
	if len(diags) != 1 || diags[0].Category != entity.DiagOK {
		t.Fatalf("at latest must be a single OK, got %+v", diags)
	}
}

// --check-updates with a dead network: silence, never a warning. The
// doctor stays usable offline.
func TestAuditEnvctlFreshnessSilentOnNetworkError(t *testing.T) {
	withNoRepo(t)
	uc := &DoctorAuditUseCase{
		envctlVersion: "v2.1.5",
		gitDescribe: func(_ context.Context, _ string) (string, error) {
			return "", errors.New("no checkout")
		},
		checkUpdates: true,
		latestRelease: func(_ context.Context) (string, error) {
			return "", errors.New("dial tcp: connectex")
		},
	}
	if diags := collectFreshnessDiags(uc); len(diags) != 0 {
		t.Errorf("network failure must stay silent, got %+v", diags)
	}
}
