package windows

import (
	"context"
	"fmt"
	"os/exec"
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

// The probe reports a target by TOKEN, never by path: a path crossing the
// process boundary is mangled by the console code page (a non-ASCII %APPDATA%),
// and no separator can truncate it. Go maps the token back to the target it
// generated, so the closed set is closed by construction.
func TestStartupTokenIsClosedSet(t *testing.T) {
	seen := map[string]bool{}
	for _, tgt := range startupTargets {
		if tgt.token == "" || tgt.kind == "" {
			t.Errorf("target %+v must declare a token and a kind", tgt)
		}
		if seen[tgt.token] {
			t.Errorf("duplicate startup token %q", tgt.token)
		}
		seen[tgt.token] = true
	}
	if len(seen) < 4 {
		t.Errorf("expected at least the 2 Run keys and 2 Startup folders, got %d", len(seen))
	}
}

// Unknown tokens are not removable: a name probing into a target envctl did
// not generate must never reach a deletion.
func TestStartupTargetForTokenRejectsUnknown(t *testing.T) {
	for _, tgt := range startupTargets {
		got, ok := startupTargetForToken(tgt.token)
		if !ok || got.token != tgt.token || got.kind != tgt.kind {
			t.Errorf("startupTargetForToken(%q) = (%+v, %v), want the declared target", tgt.token, got, ok)
		}
	}
	for _, bad := range []string{"", "RUN_UNKNOWN", "Services\\WavesSvc", "run_hkcu:"} {
		if _, ok := startupTargetForToken(bad); ok {
			t.Errorf("startupTargetForToken(%q) must be rejected", bad)
		}
	}
	// Token matching is case-sensitive on purpose: the probe always emits the
	// declared casing. A case-insensitive lookup would let a hand-written probe
	// reach a deletion.
	if _, ok := startupTargetForToken("run_hkcu"); ok {
		t.Error("token lookup must stay case-sensitive")
	}
}

// The rendered script must carry the literal registry paths, so a typo in the
// table cannot silently make every name probe as absent.
func TestStartupScriptsEmbedLiteralRegistryPaths(t *testing.T) {
	for _, script := range []string{startupProbeScript(nil), startupRemovalScript("X", startupTargets)} {
		for _, want := range []string{
			`HKCU:\Software\Microsoft\Windows\CurrentVersion\Run`,
			`HKLM:\Software\Microsoft\Windows\CurrentVersion\Run`,
		} {
			if !strings.Contains(script, want) {
				t.Errorf("script must embed the literal Run key %q: %s", want, script)
			}
		}
		if !strings.Contains(script, "ApplicationData") || !strings.Contains(script, "CommonApplicationData") {
			t.Errorf("Startup folders must resolve via GetFolderPath, not an unset env var: %s", script)
		}
		if strings.Contains(script, "$env:APPDATA") || strings.Contains(script, "$env:ProgramData") {
			t.Errorf("Startup folders must not depend on env vars (unset in non-interactive contexts): %s", script)
		}
	}
}

// Removal goes through the registry provider and the filesystem, never a WMI
// method: Win32_StartupCommand is a CIM_Setting whose published MOF lists
// properties only, so a .Delete() call cannot work.
func TestStartupRemovalScriptAvoidsMissingWmiMethod(t *testing.T) {
	reg := startupTargetForKind(t, startupKindRegistry)
	dir := startupTargetForKind(t, startupKindDir)
	registry := startupRemovalScript("SecurityHealth", []startupTarget{reg})
	if strings.Contains(registry, ".Delete()") || strings.Contains(registry, "Invoke-CimMethod") {
		t.Errorf("removal must not call a WMI method: %s", registry)
	}
	if !strings.Contains(registry, "Remove-ItemProperty") {
		t.Errorf("registry target must be removed with Remove-ItemProperty: %s", registry)
	}
	if strings.Contains(registry, "Get-ChildItem") {
		t.Errorf("registry target must not enumerate the Startup folders: %s", registry)
	}
	folder := startupRemovalScript("Brave", []startupTarget{dir})
	if strings.Contains(folder, "Remove-ItemProperty") {
		t.Errorf("folder target must not hit the registry: %s", folder)
	}
	if !strings.Contains(folder, "Get-ChildItem") || !strings.Contains(folder, "Remove-Item ") {
		t.Errorf("folder target must be removed with Remove-Item: %s", folder)
	}
	// -LiteralPath is what keeps a `[` in a manifest name from becoming a path
	// wildcard; the registry branch relies on Escape() for the value name.
	if !strings.Contains(folder, "-LiteralPath") {
		t.Errorf("folder enumeration must use -LiteralPath: %s", folder)
	}
	// A same-named directory in the Startup folder must never be a removal
	// target, and must not make the check report drift either.
	if !strings.Contains(folder, "PSIsContainer") {
		t.Errorf("folder enumeration must skip directories: %s", folder)
	}
}

// -Name on the registry provider is always a WildcardPattern, so a manifest
// name carrying wildcard characters must be escaped or it would match and
// delete several Run values. The probe compares with -contains instead.
func TestStartupScriptsNeutralizeWildcardNames(t *testing.T) {
	name := `Steam*Game[x]?`
	probe := startupProbeScript([]string{name})
	removal := startupRemovalScript(name, []startupTarget{startupTargetForKind(t, startupKindRegistry)})
	// Two independent assertions: an `&&` here would be inert, since the probe
	// does use -contains and would short-circuit the check away.
	if !strings.Contains(probe, "-contains") {
		t.Errorf("probe must compare with -contains: %s", probe)
	}
	if strings.Contains(probe, "Get-ItemProperty") {
		t.Errorf("probe must not read a value by -Name (always a WildcardPattern in the registry provider): %s", probe)
	}
	if !strings.Contains(removal, "[WildcardPattern]::Escape") {
		t.Errorf("removal must escape the name for -Name (WildcardPattern): %s", removal)
	}
	for _, script := range []string{probe, removal} {
		if strings.Contains(script, "-Filter") {
			t.Errorf("name must never reach -Filter (wildcard injection): %s", script)
		}
	}
}

// Adversarial manifest names stay inside single-quoted PowerShell strings.
func TestStartupScriptsQuoteAdversarialNames(t *testing.T) {
	name := `a'b"c$d`
	probe := startupProbeScript([]string{name})
	removal := startupRemovalScript(name, []startupTarget{startupTargetForKind(t, startupKindRegistry)})
	for _, script := range []string{probe, removal} {
		if strings.Contains(script, `"`+name) {
			t.Errorf("payload leaked into double-quoted interpolation: %s", script)
		}
		if !strings.Contains(script, psQuote(name)) {
			t.Errorf("name must be single-quoted via psQuote: %s", script)
		}
	}
}

// The fail-closed guards are the behaviour this whole rewrite exists for, and
// they are only reachable with a real spawn. Pin them as pure Go so Linux CI
// covers the decision, not just the parsing.
func TestValidateStartupRows(t *testing.T) {
	names := []string{"BraveSoftware", "Canva", "MicrosoftEdge"}

	t.Run("complete readout is accepted", func(t *testing.T) {
		rows := map[string]string{
			"bravesoftware": "RUN_HKCU",
			"canva":         "DIR_ROAMING",
			"microsoftedge": "",
		}
		if err := validateStartupRows(rows, names); err != nil {
			t.Errorf("a complete readout of known tokens must be accepted, got %v", err)
		}
	})

	t.Run("partial readout fails instead of reading as absent", func(t *testing.T) {
		rows := map[string]string{"bravesoftware": "RUN_HKCU"}
		err := validateStartupRows(rows, names)
		if err == nil {
			t.Fatal("a short readout must fail, not certify the missing names as absent")
		}
		if !strings.Contains(err.Error(), "answered 1 of 3") {
			t.Errorf("error must report the cardinality gap, got %q", err)
		}
	})

	t.Run("unknown target fails instead of reading as absent", func(t *testing.T) {
		rows := map[string]string{
			"bravesoftware": "RUN_HKCU",
			"canva":         "RUN_NOT_A_TARGET",
			"microsoftedge": "",
		}
		err := validateStartupRows(rows, names)
		if err == nil {
			t.Fatal("a token envctl cannot resolve must fail, not read as absent")
		}
		if !strings.Contains(err.Error(), "RUN_NOT_A_TARGET") {
			t.Errorf("error must name the offending token, got %q", err)
		}
		if !strings.Contains(err.Error(), "canva") {
			t.Errorf("error must name the offending entry, got %q", err)
		}
	})

	t.Run("error message is stable across map iteration order", func(t *testing.T) {
		rows := map[string]string{
			"bravesoftware": "BAD_ONE",
			"canva":         "BAD_TWO",
			"microsoftedge": "BAD_THREE",
		}
		first := validateStartupRows(rows, names)
		if first == nil {
			t.Fatal("expected an error")
		}
		for i := 0; i < 20; i++ {
			if got := validateStartupRows(rows, names); got == nil || got.Error() != first.Error() {
				t.Fatalf("error message is not stable: %v vs %v", got, first)
			}
		}
	})
}

func TestStartupConforms(t *testing.T) {
	if ok, details := startupConforms(""); !ok {
		t.Errorf("absent entry must conform, got ok=false (%q)", details)
	}
	reg := startupTargetForKind(t, startupKindRegistry)
	if ok, details := startupConforms(reg.token); ok {
		t.Errorf("an entry found in a target must be drift, got ok=true (%q)", details)
	}
	if ok, _ := startupConforms("RUN_HKCU,RUN_HKLM"); ok {
		t.Error("several targets must still be drift")
	}
}

func TestStartupUnknownTokens(t *testing.T) {
	if got := startupUnknownTokens("RUN_HKCU,DIR_PROGRAMDATA"); len(got) != 0 {
		t.Errorf("known tokens must not be reported unknown, got %v", got)
	}
	if got := startupUnknownTokens("RUN_HKCU,RUN_NOPE"); len(got) != 1 || got[0] != "RUN_NOPE" {
		t.Errorf("startupUnknownTokens = %v, want [RUN_NOPE]", got)
	}
	if got := startupUnknownTokens(""); len(got) != 0 {
		t.Errorf("absent entry has no unknown tokens, got %v", got)
	}
}

// The parser decides present/absent for the whole family, and the cardinality
// guard depends on it skipping malformed rows, so pin both.
func TestParseStartupProbe(t *testing.T) {
	out := "DEBLOATSTARTUP|||SecurityHealth|||RUN_HKCU\r\n" +
		"DEBLOATSTARTUP|||Canva|||DIR_ROAMING\r\n" +
		"DEBLOATSTARTUP|||BraveSoftware|||\r\n" +
		"garbage line without separator\r\n" +
		"DEBLOATSTARTUP||||RUN_HKCU\r\n"
	rows := parseStartupProbe(out)
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows (empty-name and malformed lines skipped), got %d (%v)", len(rows), rows)
	}
	if got := rows["securityhealth"]; got != "RUN_HKCU" {
		t.Errorf("lookup must be case-insensitive on the name, got %q", got)
	}
	if got := rows["canva"]; got != "DIR_ROAMING" {
		t.Errorf("folder token = %q, want DIR_ROAMING", got)
	}
	// An absent entry must probe as an empty token list, not as missing data:
	// that is what startupConforms reads to report conformance.
	if got, ok := rows["bravesoftware"]; !ok || got != "" {
		t.Errorf("absent entry must probe as empty, got %q (present=%v)", got, ok)
	}
}

// The check, the batch check and the apply path must resolve the target
// startup type identically, or the doctor audits a state nobody applies. The
// fallback is the guard against silently disabling a service the owner wanted
// left on Manual.
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

// Round-trip on disposable startup entries: the only coverage of the
// destructive path, for both target kinds. Creates an entry, proves the check
// reports drift, applies, proves convergence, and re-applies for idempotency —
// exercising the probe, the escaped -Name, the removal script and the Startup
// folder branch together. Runs in the windows-latest CI job.
func TestWindowsTweaksManager_StartupItemRoundTrip(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("skipping startup apply round-trip on non-windows platform")
	}
	runPS := func(script string) ([]byte, error) {
		return exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", script).CombinedOutput()
	}

	mgr := NewWindowsTweaksManager(logger.NewNoopLogger())
	ctx := context.Background()

	// A startup entry is a Run-key value or a Startup-folder file; the two take
	// different code paths in both the probe and the removal.
	cases := []struct {
		label  string
		name   string
		create string
		// present reports whether the entry is still on disk: exit 0 = present.
		// It must test THIS name, not the container: GetValueNames() returns
		// every value in the Run key, so a bare non-null check would report
		// "present" for entries the apply never touched.
		present string
		remove  string
	}{
		{
			label: "Run key value",
			name:  "EnvctlStartupProbeRunKey",
			create: fmt.Sprintf(
				`New-Item -Path 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Run' -Force | Out-Null; `+
					`New-ItemProperty -Path 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Run' -Name %s -Value 'envctl-test' -PropertyType String -Force | Out-Null`,
				psQuote("EnvctlStartupProbeRunKey")),
			present: fmt.Sprintf(
				`$k = 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Run'; `+
					`if (Test-Path -LiteralPath $k) { if (@((Get-Item -LiteralPath $k).GetValueNames()) -contains %s) { exit 0 } }; exit 1`,
				psQuote("EnvctlStartupProbeRunKey")),
			remove: `Remove-ItemProperty -LiteralPath 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Run' -Name %s -Force -ErrorAction SilentlyContinue`,
		},
		{
			label: "Startup folder file",
			name:  "EnvctlStartupProbeFolder",
			create: fmt.Sprintf(
				`$d = Join-Path ([Environment]::GetFolderPath('ApplicationData')) '%s'; `+
					`New-Item -ItemType Directory -Path $d -Force | Out-Null; `+
					`New-Item -ItemType File -Path (Join-Path $d '%s.lnk') -Force | Out-Null`,
				startupFolderSuffix, "EnvctlStartupProbeFolder"),
			present: fmt.Sprintf(
				`$d = Join-Path ([Environment]::GetFolderPath('ApplicationData')) '%s'; `+
					`if (Test-Path -LiteralPath (Join-Path $d '%s.lnk')) { exit 0 }; exit 1`,
				startupFolderSuffix, "EnvctlStartupProbeFolder"),
			remove: fmt.Sprintf(
				`$d = Join-Path ([Environment]::GetFolderPath('ApplicationData')) '%s'; `+
					`Remove-Item -LiteralPath (Join-Path $d '%s.lnk') -Force -ErrorAction SilentlyContinue`,
				startupFolderSuffix, "EnvctlStartupProbeFolder"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.label, func(t *testing.T) {
			// A failed setup must fail the build, not skip it: a skip here is
			// the CI staying green without ever executing the removal.
			if out, err := runPS(tc.create); err != nil {
				t.Fatalf("cannot create the disposable %s: %v: %s", tc.label, err, out)
			}
			t.Cleanup(func() {
				_, _ = runPS(fmt.Sprintf(tc.remove, psQuote(tc.name)))
			})
			if out, err := runPS(tc.present); err != nil {
				t.Fatalf("the disposable %s was not created: %v: %s", tc.label, err, out)
			}

			tweak := entity.WindowsTweak{
				ID: "test-startup-" + tc.name, Name: tc.name, Type: "StartupItem", Category: "startup",
			}

			ok, details, err := mgr.CheckTweak(ctx, tweak)
			if err != nil {
				t.Fatalf("check before apply: %v", err)
			}
			if ok {
				t.Fatalf("an existing %s must be drift, got conforming (%q)", tc.label, details)
			}

			if err := mgr.ApplyTweak(ctx, tweak); err != nil {
				t.Fatalf("apply: %v", err)
			}

			ok, details, err = mgr.CheckTweak(ctx, tweak)
			if err != nil {
				t.Fatalf("check after apply: %v", err)
			}
			if !ok {
				t.Errorf("%s must be gone after apply, still reporting drift (%q)", tc.label, details)
			}
			if out, err := runPS(tc.present); err == nil {
				t.Errorf("the %s still exists on disk after apply: %s", tc.label, out)
			}

			// Idempotent: a second apply on an absent entry is a no-op, not an
			// error.
			if err := mgr.ApplyTweak(ctx, tweak); err != nil {
				t.Errorf("second apply on an absent entry must be a no-op, got %v", err)
			}
		})
	}
}

func startupTargetForKind(t *testing.T, kind string) startupTarget {
	t.Helper()
	for _, tgt := range startupTargets {
		if tgt.kind == kind {
			return tgt
		}
	}
	t.Fatalf("no startup target of kind %q", kind)
	return startupTarget{}
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
