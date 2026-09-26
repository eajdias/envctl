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

// The probe only ever looks at the two Run keys and the two Startup folders,
// so "never touches a service" is structural: there is no Location to
// misclassify. startupTargetKind routes the removal to the right mechanism.
func TestStartupTargetKind(t *testing.T) {
	tests := []struct {
		name     string
		location string
		want     string
	}{
		{"hkcu run key", startupRunKeys[0], startupKindRegistry},
		{"hklm run key", startupRunKeys[1], startupKindRegistry},
		{"roaming startup folder", `C:\Users\owner\AppData\Roaming\Microsoft\Windows\Start Menu\Programs\Startup`, startupKindDir},
		{"programdata startup folder", `C:\ProgramData\Microsoft\Windows\Start Menu\Programs\Startup`, startupKindDir},
		{"service key is not a startup target", `HKEY_LOCAL_MACHINE\SYSTEM\CurrentControlSet\Services\WavesSvc`, ""},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := startupTargetKind(tt.location); got != tt.want {
				t.Errorf("startupTargetKind(%q) = %q, want %q", tt.location, got, tt.want)
			}
		})
	}
}

func TestParseStartupProbe(t *testing.T) {
	out := "DEBLOATSTARTUP|||SecurityHealth|||" + startupRunKeys[0] + "\r\n" +
		"DEBLOATSTARTUP|||Canva|||C:\\Users\\owner\\AppData\\Roaming\\Microsoft\\Windows\\Start Menu\\Programs\\Startup\r\n" +
		"DEBLOATSTARTUP|||BraveSoftware|||\r\n" +
		"garbage line without separator\r\n"
	rows := parseStartupProbe(out)
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d (%v)", len(rows), rows)
	}
	if got := rows["securityhealth"]; got != startupRunKeys[0] {
		t.Errorf("registry row = %q, want %q", got, startupRunKeys[0])
	}
	if got := rows["canva"]; !strings.Contains(got, "Start Menu") {
		t.Errorf("dir row = %q, want the Startup folder", got)
	}
	if got, ok := rows["bravesoftware"]; !ok || got != "" {
		t.Errorf("absent entry must probe as an empty location, got %q (present=%v)", got, ok)
	}
}

// Removal goes through the registry provider and the filesystem, never a WMI
// method: Win32_StartupCommand is a CIM_Setting whose published MOF lists
// properties only, so a .Delete() call cannot work.
func TestStartupRemovalScriptAvoidsMissingWmiMethod(t *testing.T) {
	script := startupRemovalScript("SecurityHealth", []string{startupRunKeys[0]})
	if strings.Contains(script, ".Delete()") || strings.Contains(script, "Invoke-CimMethod") {
		t.Errorf("removal must not call a WMI method (Win32_StartupCommand has none): %s", script)
	}
	if !strings.Contains(script, "Remove-ItemProperty") {
		t.Errorf("registry location must be removed with Remove-ItemProperty: %s", script)
	}
	dirScript := startupRemovalScript("Brave", []string{`C:\Users\owner\AppData\Roaming\Microsoft\Windows\Start Menu\Programs\Startup`})
	if strings.Contains(dirScript, "Remove-ItemProperty") {
		t.Errorf("folder location must not hit the registry: %s", dirScript)
	}
	if !strings.Contains(dirScript, "Remove-Item ") || !strings.Contains(dirScript, "-LiteralPath") {
		t.Errorf("folder location must be removed with Remove-Item -LiteralPath: %s", dirScript)
	}
}

// Adversarial manifest names stay inside single-quoted PowerShell strings and
// never become -Filter wildcards.
func TestStartupScriptsQuoteAdversarialNames(t *testing.T) {
	name := `a'b"c$d[e]`
	probe := startupProbeScript([]string{name})
	removal := startupRemovalScript(name, []string{startupRunKeys[0]})
	for _, script := range []string{probe, removal} {
		if strings.Contains(script, `"`+name) {
			t.Errorf("payload leaked into double-quoted interpolation: %s", script)
		}
		if !strings.Contains(script, psQuote(name)) {
			t.Errorf("name must be single-quoted via psQuote: %s", script)
		}
		if strings.Contains(script, "-Filter") {
			t.Errorf("name must never reach -Filter (wildcard injection): %s", script)
		}
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

// An entry the probe found is drift; an entry the probe did not find is
// conforming. The probe can only ever report a Run key or a Startup folder,
// so there is no "looks like a service" case left to defend.
func TestStartupConforms(t *testing.T) {
	if ok, details := startupConforms(""); !ok {
		t.Errorf("absent entry must conform, got ok=false (%q)", details)
	}
	if ok, details := startupConforms(startupRunKeys[0]); ok {
		t.Errorf("run key entry must be drift so it gets removed, got ok=true (%q)", details)
	}
	dir := `C:\Users\owner\AppData\Roaming\Microsoft\Windows\Start Menu\Programs\Startup`
	if ok, details := startupConforms(dir + ";" + startupRunKeys[1]); ok {
		t.Errorf("multiple locations must be drift, got ok=true (%q)", details)
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
		{ID: "sample-startup-absent", Name: "EnvctlNoSuchStartup123", Type: "StartupItem"},
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
