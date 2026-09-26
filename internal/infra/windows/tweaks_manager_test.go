package windows

import (
	"context"
	"runtime"
	"strings"
	"testing"

	"github.com/eajdias/envctl/internal/domain/entity"
	"github.com/eajdias/envctl/internal/infra/logger"
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

// A Win32_StartupCommand row is only ours to delete when it is a user startup
// entry (Run key or Startup folder). Service rows share the same class and
// deleting one breaks a service, so the predicate must reject them.
func TestStartupLocationRemovable(t *testing.T) {
	tests := []struct {
		name     string
		location string
		want     bool
	}{
		{"run key current user", `HKEY_CURRENT_USER\Software\Microsoft\Windows\CurrentVersion\Run`, true},
		{"run key local machine", `HKEY_LOCAL_MACHINE\Software\Microsoft\Windows\CurrentVersion\Run`, true},
		{"run key wow6432", `HKEY_CURRENT_USER\Software\WOW6432Node\Microsoft\Windows\CurrentVersion\Run`, true},
		{"startup folder roaming", `C:\Users\owner\AppData\Roaming\Microsoft\Windows\Start Menu\Programs\Startup\Brave.lnk`, true},
		{"startup folder programdata", `C:\ProgramData\Microsoft\Windows\Start Menu\Programs\Startup\Canva.lnk`, true},
		{"service row", `HKEY_LOCAL_MACHINE\SYSTEM\CurrentControlSet\Services\WavesSvc`, false},
		{"service row bare", "Service", false},
		{"run once is one-shot, not a startup entry", `HKEY_CURRENT_USER\Software\Microsoft\Windows\CurrentVersion\RunOnce`, false},
		{"empty location", "", false},
		{"unrelated path", `C:\Temp\evil.lnk`, false},
		{"lowercase must still match", `hkey_current_user\software\microsoft\windows\currentversion\run`, true},
		{"lookalike suffix is not a run key", `HKEY_CURRENT_USER\Software\Microsoft\Windows\CurrentVersion\RunBackup`, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := startupLocationRemovable(tt.location); got != tt.want {
				t.Errorf("startupLocationRemovable(%q) = %v, want %v", tt.location, got, tt.want)
			}
		})
	}
}

// Parse contract for the CIM readout the startup family depends on:
// `DEBLOATSTARTUP|||<name>|||<location>` rows.
func TestParseStartupRows(t *testing.T) {
	out := "DEBLOATSTARTUP|||SecurityHealth|||HKEY_CURRENT_USER\\Software\\Microsoft\\Windows\\CurrentVersion\\Run\r\n" +
		"DEBLOATSTARTUP|||WavesSvc|||HKEY_LOCAL_MACHINE\\SYSTEM\\CurrentControlSet\\Services\\WavesSvc\r\n" +
		"garbage line without separator\r\n"
	rows := parseStartupRows(out)
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows from CIM output, got %d (%v)", len(rows), rows)
	}
	if rows["securityhealth"] != `HKEY_CURRENT_USER\Software\Microsoft\Windows\CurrentVersion\Run` {
		t.Errorf("row lookup must be case-insensitive on the name, got %q", rows["securityhealth"])
	}
	if startupLocationRemovable(rows["wavessvc"]) {
		t.Error("service row must not be removable even when the name matches a manifest entry")
	}
	if !startupLocationRemovable(rows["securityhealth"]) {
		t.Error("run key row must be removable")
	}
}

// The check, the batch check and the apply path must resolve the target
// startup type identically, or the doctor audits a state nobody applies.
func TestServiceExpectedStateDefaultsToDisabled(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  string
	}{
		{"explicit Manual", "Manual", "Manual"},
		{"explicit Disabled", "Disabled", "Disabled"},
		{"empty value falls back", "", "Disabled"},
		{"nil value falls back", nil, "Disabled"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tw := entity.WindowsTweak{Name: "Spooler", Type: "Service", Value: tt.value}
			if got := serviceExpectedState(tw); got != tt.want {
				t.Errorf("serviceExpectedState(value=%v) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}

// A startup entry that exists only as a service row is NOT drift: the doctor
// must stay green and apply must be a no-op, never a service deletion.
func TestStartupConformsTreatsServiceRowAsConforming(t *testing.T) {
	rows := map[string]string{
		"wavessvc":      `HKEY_LOCAL_MACHINE\SYSTEM\CurrentControlSet\Services\WavesSvc`,
		"microsoftedge": `HKEY_CURRENT_USER\Software\Microsoft\Windows\CurrentVersion\Run`,
	}
	if ok, details := startupConforms(rows, "WavesSvc"); !ok {
		t.Errorf("service-only row must conform, got ok=false (%q)", details)
	}
	if ok, details := startupConforms(rows, "MicrosoftEdge"); ok {
		t.Errorf("run key row must be drift so it gets removed, got ok=true (%q)", details)
	}
	if ok, _ := startupConforms(rows, "NeverInstalled"); !ok {
		t.Error("absent name must conform")
	}
}

func TestWindowsTweaksManager_CheckTweak(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("skipping Windows registry tweak tests on non-windows platform")
	}

	mgr := NewWindowsTweaksManager(logger.NewNoopLogger())

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

	mgr := NewWindowsTweaksManager(logger.NewNoopLogger())

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

	mgr := NewWindowsTweaksManager(logger.NewNoopLogger())
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

	mgr := NewWindowsTweaksManager(logger.NewNoopLogger())

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
