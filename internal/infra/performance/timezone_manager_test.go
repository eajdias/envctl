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

// newTimezoneManagerForTest builds a manager over a temporary root that carries
// the IANA zones the tests write, so the enforce path can reach its validation.
func newTimezoneManagerForTest(t *testing.T, current string, failOn map[string]error, calls *[][]string) *timezoneManager {
	t.Helper()
	root := t.TempDir()
	for _, zone := range []string{"Etc/UTC", "America/Sao_Paulo"} {
		if err := writeFile(root, filepath.Join("usr/share/zoneinfo", zone), ""); err != nil {
			t.Fatal(err)
		}
	}
	return newTimezoneManager(
		root,
		func(context.Context, string, ...string) ([]byte, error) { return []byte(current), nil },
		func(ctx context.Context, name string, args ...string) ([]byte, error) {
			if calls != nil {
				*calls = append(*calls, append([]string{name}, args...))
			}
			if err, ok := failOn[name]; ok && err != nil {
				return nil, err
			}
			return nil, nil
		},
		func() time.Time { return time.Unix(1, 0) },
		false,
	)
}

func wroteTimezone(calls [][]string) bool {
	for _, call := range calls {
		if len(call) >= 2 && call[len(call)-2] == "set-timezone" {
			return true
		}
	}
	return false
}

// All three reachable hosts in the fleet report Etc/UTC with NTP active, so
// verify mode against Etc/UTC must be a silent pass.
func TestTimezoneManagerVerifyMatchingZoneIsOKAndWritesNothing(t *testing.T) {
	var calls [][]string
	manager := newTimezoneManagerForTest(t, "Etc/UTC", nil, &calls)

	diags, err := manager.Apply(context.Background(), entity.TimezoneSpec{Mode: "verify", Expected: "Etc/UTC"}, false)
	if err != nil {
		t.Fatalf("verify failed: %v", err)
	}
	if len(diags) != 1 || diags[0].Category != entity.DiagOK {
		t.Fatalf("diagnostics = %#v, want one OK", diags)
	}
	if wroteTimezone(calls) {
		t.Fatalf("verify mode wrote the timezone: %#v", calls)
	}
}

// A mismatch under verify is information, not a warning and not a write: the
// operator decides whether their zone is intentional.
func TestTimezoneManagerVerifyMismatchIsInfoAndDoesNotWrite(t *testing.T) {
	var calls [][]string
	manager := newTimezoneManagerForTest(t, "America/Sao_Paulo", nil, &calls)

	diags, err := manager.Apply(context.Background(), entity.TimezoneSpec{Mode: "verify", Expected: "Etc/UTC"}, false)
	if err != nil {
		t.Fatalf("verify failed: %v", err)
	}
	if len(diags) != 1 || diags[0].Category != entity.DiagInfo {
		t.Fatalf("diagnostics = %#v, want one INFO so a converged host stays 0 WARN", diags)
	}
	if !strings.Contains(diags[0].Details, "America/Sao_Paulo") || !strings.Contains(diags[0].Details, "Etc/UTC") {
		t.Fatalf("diagnostic %q does not name both zones", diags[0].Details)
	}
	for _, call := range calls {
		if call[0] == "set-timezone" {
			t.Fatalf("verify mode wrote the timezone: %#v", calls)
		}
	}
}

func TestTimezoneManagerEnforceMismatchWrites(t *testing.T) {
	var calls [][]string
	manager := newTimezoneManagerForTest(t, "America/Sao_Paulo", nil, &calls)

	diags, err := manager.Apply(context.Background(), entity.TimezoneSpec{Mode: "enforce", Expected: "Etc/UTC"}, false)
	if err != nil {
		t.Fatalf("enforce failed: %v", err)
	}
	if len(diags) != 1 || diags[0].Category != entity.DiagOK {
		t.Fatalf("diagnostics = %#v, want one OK", diags)
	}
	found := false
	for _, call := range calls {
		if len(call) >= 3 && call[len(call)-2] == "set-timezone" && call[len(call)-1] == "Etc/UTC" {
			found = true
		}
	}
	if !found {
		t.Fatalf("commands = %#v, want timedatectl set-timezone Etc/UTC", calls)
	}
}

func TestTimezoneManagerEnforceMatchingZoneDoesNotWrite(t *testing.T) {
	var calls [][]string
	manager := newTimezoneManagerForTest(t, "Etc/UTC", nil, &calls)

	diags, err := manager.Apply(context.Background(), entity.TimezoneSpec{Mode: "enforce", Expected: "Etc/UTC"}, false)
	if err != nil {
		t.Fatalf("enforce failed: %v", err)
	}
	if len(diags) != 1 || diags[0].Category != entity.DiagOK {
		t.Fatalf("diagnostics = %#v, want one OK", diags)
	}
	if wroteTimezone(calls) {
		t.Fatalf("enforce wrote an already-correct timezone: %#v", calls)
	}
}

// An unknown IANA name must be refused before the write, never passed to
// timedatectl and never reported as success.
func TestTimezoneManagerRejectsUnknownZoneName(t *testing.T) {
	var calls [][]string
	manager := newTimezoneManager("/nonexistent-zoneinfo",
		func(context.Context, string, ...string) ([]byte, error) { return []byte("America/Sao_Paulo"), nil },
		func(ctx context.Context, name string, args ...string) ([]byte, error) {
			calls = append(calls, append([]string{name}, args...))
			return nil, nil
		}, func() time.Time { return time.Unix(1, 0) }, false)

	diags, err := manager.Apply(context.Background(), entity.TimezoneSpec{Mode: "enforce", Expected: "Mars/Olympus"}, false)
	if err == nil {
		t.Fatal("expected an unknown zone name to be rejected")
	}
	if len(diags) != 1 || diags[0].Category != entity.DiagError || !strings.Contains(diags[0].Details, "Mars/Olympus") {
		t.Fatalf("diagnostics = %#v, want an error naming the zone", diags)
	}
	if wroteTimezone(calls) {
		t.Fatalf("an invalid zone reached timedatectl: %#v", calls)
	}
}

func TestTimezoneManagerReportsMissingTimedatectl(t *testing.T) {
	// A host where timedatectl is absent: the read fails, the /etc/timezone
	// fallback is absent too, and the error must name the binary so the
	// operator knows what to install.
	manager := newTimezoneManager("/nonexistent-root",
		func(context.Context, string, ...string) ([]byte, error) {
			return nil, errors.New("exec: \"timedatectl\": executable file not found in $PATH")
		}, func(context.Context, string, ...string) ([]byte, error) { return nil, nil },
		func() time.Time { return time.Unix(1, 0) }, false)

	diags, err := manager.Apply(context.Background(), entity.TimezoneSpec{Mode: "enforce", Expected: "Etc/UTC"}, false)
	if err == nil {
		t.Fatal("expected a missing timedatectl to be reported")
	}
	if len(diags) != 1 || diags[0].Category != entity.DiagError || !strings.Contains(diags[0].Details, "timedatectl") {
		t.Fatalf("diagnostics = %#v, want an error naming the binary", diags)
	}
}

func TestTimezoneManagerDryRunWritesNothing(t *testing.T) {
	var calls [][]string
	manager := newTimezoneManagerForTest(t, "America/Sao_Paulo", nil, &calls)

	diags, err := manager.Apply(context.Background(), entity.TimezoneSpec{Mode: "enforce", Expected: "Etc/UTC"}, true)
	if err != nil {
		t.Fatalf("dry-run failed: %v", err)
	}
	if len(diags) != 1 || diags[0].Category != entity.DiagInfo {
		t.Fatalf("diagnostics = %#v, want one INFO", diags)
	}
	if wroteTimezone(calls) {
		t.Fatalf("dry-run wrote the timezone: %#v", calls)
	}
}

func TestTimezoneManagerRejectsUnknownMode(t *testing.T) {
	manager := newTimezoneManagerForTest(t, "Etc/UTC", nil, nil)
	if _, err := manager.Apply(context.Background(), entity.TimezoneSpec{Mode: "rewrite", Expected: "Etc/UTC"}, false); err == nil {
		t.Fatal("expected an unknown mode to be rejected")
	}
}

func TestTimezoneManagerFallsBackToEtcTimezoneFile(t *testing.T) {
	root := t.TempDir()
	if err := writeFile(root, "etc/timezone", "Etc/UTC\n"); err != nil {
		t.Fatal(err)
	}
	manager := newTimezoneManager(root, nil, nil, func() time.Time { return time.Unix(1, 0) }, false)

	current, err := manager.Current(context.Background())
	if err != nil {
		t.Fatalf("Current failed: %v", err)
	}
	if current != "Etc/UTC" {
		t.Fatalf("Current = %q, want the /etc/timezone fallback", current)
	}
}

func writeFile(root, rel, content string) error {
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}
