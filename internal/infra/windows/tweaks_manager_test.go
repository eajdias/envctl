package windows

import (
	"context"
	"runtime"
	"strings"
	"testing"

	"github.com/eajdias/envctl/internal/domain/entity"
)

func TestPSQuoteDoublesSingleQuotes(t *testing.T) {
	if got := psQuote("a'b"); got != "a''b" {
		t.Errorf("psQuote(%q) = %q, want %q", "a'b", got, "a''b")
	}
	if got := psValue("x'y"); got != "'x''y'" {
		t.Errorf("psValue(%q) = %q, want %q", "x'y", got, "'x''y'")
	}
	if got := psValue(true); got != "$true" {
		t.Errorf("psValue(true) = %q, want %q", got, "$true")
	}
	if got := psValue(false); got != "$false" {
		t.Errorf("psValue(false) = %q, want %q", got, "$false")
	}
	if got := psValue(0); got != "0" {
		t.Errorf("psValue(0) = %q, want %q", got, "0")
	}
}

// Adversarial payloads must stay inside single-quoted PowerShell strings:
// single quotes double, double quotes and $ stay harmless inside them.
func TestPSScriptEscapesAdversarialTweak(t *testing.T) {
	name := `a'b"c$d`
	quoted := psQuote(name)
	if quoted != `a''b"c$d` {
		t.Errorf("psQuote(%q) = %q, want %q", name, quoted, `a''b"c$d`)
	}
	for _, script := range []string{
		`Enable-WindowsOptionalFeature -Online -FeatureName '` + quoted + `' -NoRestart -ErrorAction Stop`,
		`$path = '` + psQuote(`HKCU:\a'b`) + `'`,
		`$val = ` + psValue("x'y"),
	} {
		if strings.Contains(script, `"a'b`) || strings.Contains(script, `"HKCU`) {
			t.Errorf("payload leaked into double-quoted interpolation: %q", script)
		}
	}
	if !strings.Contains(psValue("x'y"), `'x''y'`) {
		t.Errorf("string value not single-quoted: %q", psValue("x'y"))
	}
}

func TestWindowsTweaksManager_CheckTweak(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("skipping Windows registry tweak tests on non-windows platform")
	}

	mgr := NewWindowsTweaksManager(nil)

	// Check a well-known Windows registry key (e.g. CurrentVersion or Explorer)
	tweak := entity.WindowsTweak{
		Name:        "HideFileExt",
		Description: "Explorer Show File Extensions",
		Path:        "HKCU:\\Software\\Microsoft\\Windows\\CurrentVersion\\Explorer\\Advanced",
		Type:        "DWord",
		Value:       0,
	}

	ok, details, err := mgr.CheckTweak(context.Background(), tweak)
	if err != nil {
		t.Fatalf("unexpected error checking tweak: %v", err)
	}

	t.Logf("HideFileExt check result: %v (details: %s)", ok, details)
}

func TestWindowsTweaksManager_CheckAppxAbsent(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("skipping Appx check tests on non-windows platform")
	}

	mgr := NewWindowsTweaksManager(nil)

	// Read-only: a package name that cannot exist is conforming (absent).
	ok, details, err := mgr.CheckTweak(context.Background(), entity.WindowsTweak{
		Name: "Envctl.DefinitelyNotInstalled123",
		Type: "Appx",
	})
	if err != nil {
		t.Fatalf("unexpected error checking absent Appx: %v", err)
	}
	if !ok {
		t.Errorf("absent Appx should be conforming, got details %q", details)
	}
}

func TestWindowsTweaksManager_CheckBatchMatchesSingle(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("skipping batch consistency tests on non-windows platform")
	}

	mgr := NewWindowsTweaksManager(nil)
	ctx := context.Background()

	// One tweak per family: real registry value, absent Appx, present
	// service, missing service. Batch must agree with the single path.
	tweaks := []entity.WindowsTweak{
		{
			ID:    "sample-hide-ext",
			Path:  "HKCU:\\Software\\Microsoft\\Windows\\CurrentVersion\\Explorer\\Advanced",
			Name:  "HideFileExt",
			Value: 0,
			Type:  "DWord",
		},
		{ID: "sample-appx-absent", Name: "Envctl.DefinitelyNotInstalled123", Type: "Appx"},
		{ID: "sample-svc", Name: "Spooler", Value: "Disabled", Type: "Service"},
		{ID: "sample-svc-missing", Name: "EnvctlNoSuchService123", Value: "Disabled", Type: "Service"},
	}

	batch := mgr.CheckBatch(ctx, tweaks)
	if len(batch) != len(tweaks) {
		t.Fatalf("CheckBatch returned %d results for %d tweaks", len(batch), len(tweaks))
	}
	for i, tw := range tweaks {
		if batch[i].Tweak.ID != tw.ID {
			t.Errorf("result %d answers %q, want %q (order-preserving)", i, batch[i].Tweak.ID, tw.ID)
		}
		ok, details, err := mgr.CheckTweak(ctx, tw)
		if (err != nil) != (batch[i].Err != nil) {
			t.Errorf("%s: single err=%v, batch err=%v", tw.ID, err, batch[i].Err)
		}
		if err == nil && (ok != batch[i].OK || details != batch[i].Details) {
			t.Errorf("%s: single (%v, %q) != batch (%v, %q)",
				tw.ID, ok, details, batch[i].OK, batch[i].Details)
		}
	}
}

func TestWindowsTweaksManager_CheckServiceReadOnly(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("skipping service check tests on non-windows platform")
	}

	mgr := NewWindowsTweaksManager(nil)

	// Read-only: StartType probe only, no state change. Spooler ships with
	// Windows 11 and is not Disabled, so the check must report drift.
	ok, details, err := mgr.CheckTweak(context.Background(), entity.WindowsTweak{
		Name:  "Spooler",
		Type:  "Service",
		Value: "Disabled",
	})
	if err != nil {
		t.Fatalf("unexpected error checking service: %v", err)
	}
	if ok {
		t.Logf("Spooler already disabled (details: %s)", details)
	} else if details == "" {
		t.Errorf("drifted service check must explain actual vs expected state")
	}

	// A service missing from the machine is conforming: nothing to disable.
	ok, details, err = mgr.CheckTweak(context.Background(), entity.WindowsTweak{
		Name:  "EnvctlNoSuchService123",
		Type:  "Service",
		Value: "Disabled",
	})
	if err != nil {
		t.Fatalf("unexpected error checking missing service: %v", err)
	}
	if !ok {
		t.Errorf("missing service should be conforming, got details %q", details)
	}
}
