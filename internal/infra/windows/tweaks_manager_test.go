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
