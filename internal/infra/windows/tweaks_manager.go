package windows

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"github.com/eajdias/envctl/internal/domain/entity"
	"github.com/eajdias/envctl/internal/domain/repository"
)

// TweaksManager implements repository.WindowsTweaksManager.
type TweaksManager struct {
	logger repository.Logger
}

func NewWindowsTweaksManager(logger repository.Logger) repository.WindowsTweaksManager {
	return &TweaksManager{
		logger: logger,
	}
}

// psQuote mirrors environment.psQuote (single-quote escape for PowerShell
// string literals); duplicated to avoid an infra→infra import for 3 lines.
func psQuote(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

// psValue renders a tweak value as a PowerShell literal: strings are
// single-quoted (inert to expansion), bools become $true/$false, and
// ints/floats stay bare (DWord-compatible).
func psValue(v any) string {
	switch t := v.(type) {
	case string:
		return "'" + psQuote(t) + "'"
	case bool:
		if t {
			return "$true"
		}
		return "$false"
	default:
		return fmt.Sprintf("%v", v)
	}
}

// Startup entries are probed and removed against a closed set: the two Run keys
// and the two Startup folders. NOT via Win32_StartupCommand, for two reasons
// confirmed against its published MOF and docs:
//   - it is a CIM_Setting whose MOF lists properties only, so the legacy tool's
//     `$_.Delete()` does not exist;
//   - its `Location` is inconsistent (registry key, the bare literals
//     "Startup"/"Common Startup", HKU\<SID>\...), which is what forced a
//     fragile classifier that matched nothing and left the doctor green.
//
// Reading the exact locations makes "never touches a service" structural: a
// service is neither a Run-key value nor a Startup-folder file.
type startupTarget struct {
	token string // what the probe reports; a path never crosses the boundary
	kind  string
	path  string // registry only: ASCII literal, safe to embed
	env   string // folder only: [Environment]::GetFolderPath argument
}

const (
	startupKindRegistry = "registry"
	startupKindDir      = "dir"
)

var startupTargets = []startupTarget{
	{token: "RUN_HKCU", kind: startupKindRegistry, path: `HKCU:\Software\Microsoft\Windows\CurrentVersion\Run`},
	{token: "RUN_HKLM", kind: startupKindRegistry, path: `HKLM:\Software\Microsoft\Windows\CurrentVersion\Run`},
	{token: "DIR_ROAMING", kind: startupKindDir, env: "ApplicationData"},
	{token: "DIR_PROGRAMDATA", kind: startupKindDir, env: "CommonApplicationData"},
}

const startupFolderSuffix = `Microsoft\Windows\Start Menu\Programs\Startup`

// startupTargetForToken resolves a probe token. An unknown token is refused, so
// a name that somehow probes outside the closed set can never reach a deletion.
func startupTargetForToken(token string) (startupTarget, bool) {
	for _, tgt := range startupTargets {
		if tgt.token == token {
			return tgt, true
		}
	}
	return startupTarget{}, false
}

// startupTargetsTable renders the PowerShell hashtable a script needs, so
// probe and apply cannot disagree about what a target is. Only the requested
// targets are rendered: a registry-only removal must not resolve a Startup
// folder it will never touch. The Startup folders resolve through GetFolderPath
// rather than %APPDATA%/%ProgramData%: those are absent in non-interactive
// contexts (service, scheduled task), and a folder path never travels back
// through the console code page.
func startupTargetsTable(targets []startupTarget) string {
	rows := make([]string, 0, len(targets))
	for _, tgt := range targets {
		var path string
		if tgt.kind == startupKindDir {
			path = fmt.Sprintf("(Join-Path ([Environment]::GetFolderPath('%s')) '%s')", tgt.env, startupFolderSuffix)
		} else {
			path = "'" + psQuote(tgt.path) + "'"
		}
		rows = append(rows, fmt.Sprintf("  '%s' = @{ Kind = '%s'; Path = %s }", tgt.token, tgt.kind, path))
	}
	// No trailing comma: `@( @{...}, )` is a parse error in powershell 5.1.
	return "@{\n" + strings.Join(rows, "\n") + "\n}"
}

// startupProbeScript reports, for each wanted name, the `,`-joined tokens of
// the targets it was found in (empty when absent). Registry keys are read with
// GetValueNames() and compared with -contains, and folders with BaseName, so a
// name carrying wildcard characters is never handed to a wildcard matcher.
func startupProbeScript(names []string) string {
	quoted := make([]string, 0, len(names))
	for _, n := range names {
		quoted = append(quoted, "'"+psQuote(n)+"'")
	}
	return fmt.Sprintf(`
$targets = %s
$keyNames = @{}
$dirNames = @{}
foreach ($tk in $targets.Keys) {
  $p = $targets[$tk].Path
  if ($targets[$tk].Kind -eq '%s') {
    if (Test-Path -LiteralPath $p) { $keyNames[$tk] = @((Get-Item -LiteralPath $p).GetValueNames()) } else { $keyNames[$tk] = @() }
  } else {
    if (Test-Path -LiteralPath $p) {
      $dirNames[$tk] = @(Get-ChildItem -LiteralPath $p -ErrorAction SilentlyContinue |
        Where-Object { -not $_.PSIsContainer } | ForEach-Object { $_.BaseName })
    } else { $dirNames[$tk] = @() }
  }
}
foreach ($n in @(%s)) {
  $found = @()
  foreach ($tk in $keyNames.Keys) { if (@($keyNames[$tk]) -contains $n) { $found += $tk } }
  foreach ($tk in $dirNames.Keys) { if (@($dirNames[$tk]) -contains $n) { $found += $tk } }
  Write-Output ("DEBLOATSTARTUP|||" + $n + "|||" + ($found -join ','))
}
`, startupTargetsTable(startupTargets), startupKindRegistry, strings.Join(quoted, ","))
}

// startupRemovalScript deletes one name from the given targets. Registry
// targets go through Remove-ItemProperty — the System Registry Provider route
// the WMI docs point at — with the name escaped because -Name is always a
// WildcardPattern there. Folder targets go through the enumerated file, since
// a Startup-folder entry is a file with an extension.
func startupRemovalScript(name string, targets []startupTarget) string {
	var b strings.Builder
	b.WriteString("\n$targets = " + startupTargetsTable(targets) + "\n")
	for _, tgt := range targets {
		if tgt.kind == startupKindRegistry {
			fmt.Fprintf(&b, `$p = $targets['%s'].Path
if ((Test-Path -LiteralPath $p) -and (@((Get-Item -LiteralPath $p).GetValueNames()) -contains '%s')) {
  Remove-ItemProperty -LiteralPath $p -Name ([WildcardPattern]::Escape('%s')) -Force -ErrorAction Stop
}
`, tgt.token, psQuote(name), psQuote(name))
		} else {
			fmt.Fprintf(&b, `$p = $targets['%s'].Path
if (Test-Path -LiteralPath $p) {
  Get-ChildItem -LiteralPath $p -ErrorAction Stop |
    Where-Object { -not $_.PSIsContainer -and $_.BaseName -eq '%s' } |
    Remove-Item -Force -ErrorAction Stop
}
`, tgt.token, psQuote(name))
		}
	}
	return b.String()
}

// parseStartupProbe indexes the probe readout by lowercased name. A name
// reported with no token is absent, not missing data.
func parseStartupProbe(out string) map[string]string {
	rows := make(map[string]string)
	for _, line := range strings.Split(out, "\n") {
		parts := strings.SplitN(strings.TrimSpace(line), "|||", 3)
		if len(parts) != 3 || parts[0] != "DEBLOATSTARTUP" || parts[1] == "" {
			continue
		}
		rows[strings.ToLower(parts[1])] = parts[2]
	}
	return rows
}

// startupConforms maps a probe token list to the tweak verdict. Conforming =
// nothing envctl is able to remove, so the doctor never claims drift it cannot
// resolve. probeStartup has already rejected unknown tokens, so an empty list
// here really means "not present in any target".
func startupConforms(tokens string) (bool, string) {
	if len(startupTargetsForTokens(tokens)) == 0 {
		return true, "No removable startup entry"
	}
	return false, "Startup entry present (will be removed on apply)"
}

// startupUnknownTokens returns the tokens the probe emitted that envctl did not
// generate. Anything here is a probe/table disagreement, never "absent".
func startupUnknownTokens(tokens string) []string {
	var out []string
	for _, tok := range strings.Split(tokens, ",") {
		if tok = strings.TrimSpace(tok); tok != "" {
			if _, ok := startupTargetForToken(tok); !ok {
				out = append(out, tok)
			}
		}
	}
	return out
}

// startupTargetsForTokens resolves the probe tokens, dropping any unknown one.
// Callers reach it only through probeStartup, which already rejected unknowns.
func startupTargetsForTokens(tokens string) []startupTarget {
	var out []startupTarget
	for _, tok := range strings.Split(tokens, ",") {
		if tok = strings.TrimSpace(tok); tok != "" {
			if tgt, ok := startupTargetForToken(tok); ok {
				out = append(out, tgt)
			}
		}
	}
	return out
}

// validateStartupRows fails closed on a probe readout envctl cannot fully
// interpret. Both checks exist so a degraded probe can never be reported as
// "absent", which is how the doctor would certify a convergence that never
// happened:
//
//   - cardinality: a partial readout leaves names missing, and a missing name
//     resolves to "" and then to "conforming". Same guard as the registry batch.
//   - unknown target: the removal path refuses a token it cannot resolve, so
//     approving one in the audit would have the two halves of the invariant
//     disagreeing.
func validateStartupRows(rows map[string]string, names []string) error {
	if len(rows) != len(names) {
		return fmt.Errorf("startup probe answered %d of %d names", len(rows), len(names))
	}
	// Sorted so the message is stable: map iteration order is randomized in Go
	// and a flaky error message is a flaky CI log.
	offenders := make([]string, 0, len(rows))
	for name, tokens := range rows {
		if unknown := startupUnknownTokens(tokens); len(unknown) > 0 {
			offenders = append(offenders, fmt.Sprintf("%q reported %v", name, unknown))
		}
	}
	if len(offenders) > 0 {
		sort.Strings(offenders)
		return fmt.Errorf("startup probe reported unknown target(s): %s", strings.Join(offenders, "; "))
	}
	return nil
}

func (m *TweaksManager) probeStartup(ctx context.Context, names []string) (map[string]string, error) {
	if len(names) == 0 {
		return map[string]string{}, nil
	}
	//nolint:gosec // G204: names come from the embedded manifest (same trust level as the package tables) and reach PowerShell only as single-quoted literals via psQuote; no shell, no expansion. Asserted by TestStartupScriptsQuoteAdversarialNames.
	cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-Command", startupProbeScript(names))
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to probe startup entries: %s (%w)", strings.TrimSpace(string(out)), err)
	}
	rows := parseStartupProbe(string(out))
	if err := validateStartupRows(rows, names); err != nil {
		return nil, err
	}
	return rows, nil
}

func (m *TweaksManager) probeStartupTweak(ctx context.Context, name string) (string, error) {
	rows, err := m.probeStartup(ctx, []string{name})
	if err != nil {
		return "", err
	}
	return rows[strings.ToLower(name)], nil
}

// serviceExpectedState reads the declared target startup type, defaulting to
// Disabled when the manifest omits it. Shared by the check, batch-check and
// apply paths so all three agree on the target.
func serviceExpectedState(tweak entity.WindowsTweak) string {
	if s, ok := tweak.Value.(string); ok && s != "" {
		return s
	}
	return "Disabled"
}

func (m *TweaksManager) CheckTweak(ctx context.Context, tweak entity.WindowsTweak) (bool, string, error) {
	if runtime.GOOS != "windows" {
		return true, "Skipped on non-Windows platform", nil
	}

	switch strings.ToLower(tweak.Type) {
	case "feature":
		psScript := fmt.Sprintf(`
$f = Get-WindowsOptionalFeature -Online -FeatureName '%s' -ErrorAction SilentlyContinue
if ($f -and $f.State -eq 'Enabled') { Write-Output "ENABLED" } else { Write-Output "DISABLED" }
`, psQuote(tweak.Name))
		cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-Command", psScript)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return false, "", fmt.Errorf("failed to check feature %s: %w", tweak.Name, err)
		}
		if strings.Contains(string(out), "ENABLED") {
			return true, "Windows optional feature is enabled", nil
		}
		return false, "Feature is disabled or not present", nil

	case "psmodule":
		// Availability = importable by powershell.exe (5.1): existence in
		// the tree is not enough (a Save-PSResource copy without a valid
		// manifest resolves in Get-Module -ListAvailable but fails at
		// import). powershell.exe is the strictest probe — a module that
		// imports there also satisfies pwsh, whose PSModulePath is a
		// superset on this machine.
		// CurrentUser scope on purpose: all-users needs elevation, and envctl
		// runs unprivileged.
		checkPsScript := fmt.Sprintf(
			`try { Import-Module -Name '%s' -Force -ErrorAction Stop; Write-Output "INSTALLED" } catch { Write-Output "MISSING" }`,
			psQuote(tweak.Name))
		cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-Command", checkPsScript)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return false, "", fmt.Errorf("failed to check PowerShell module %s: %w", tweak.Name, err)
		}
		if strings.Contains(string(out), "INSTALLED") {
			return true, "PowerShell module importable (powershell + pwsh)", nil
		}
		return false, "PowerShell module not importable by powershell.exe", nil

	case "appx":
		// Conforming = absent: the package was removed (or never installed).
		appxScript := fmt.Sprintf(
			`if (Get-AppxPackage -Name '%s' -AllUsers -ErrorAction SilentlyContinue) { Write-Output "INSTALLED" } else { Write-Output "ABSENT" }`,
			psQuote(tweak.Name))
		appxCmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-Command", appxScript)
		appxOut, appErr := appxCmd.CombinedOutput()
		if appErr != nil {
			return false, "", fmt.Errorf("failed to check Appx package %s: %w", tweak.Name, appErr)
		}
		if strings.Contains(string(appxOut), "INSTALLED") {
			return false, "Appx package installed (will be removed on apply)", nil
		}
		return true, "Appx package not installed", nil

	case "service":
		// Conforming = StartType matches the declared state (default
		// Disabled). A service missing from the machine is conforming too:
		// there is nothing to disable.
		expectedState := serviceExpectedState(tweak)
		svcScript := fmt.Sprintf(`
$s = Get-Service -Name '%s' -ErrorAction SilentlyContinue
if (-not $s) { Write-Output "NOT_PRESENT" } else { Write-Output ("STATE:" + $s.StartType.ToString()) }
`, psQuote(tweak.Name))
		svcCmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-Command", svcScript)
		svcOut, svcErr := svcCmd.CombinedOutput()
		if svcErr != nil {
			return false, "", fmt.Errorf("failed to check service %s: %w", tweak.Name, svcErr)
		}
		svcStr := strings.TrimSpace(string(svcOut))
		if svcStr == "NOT_PRESENT" {
			return true, "Service not present on system", nil
		}
		actualState := strings.TrimPrefix(svcStr, "STATE:")
		if strings.EqualFold(actualState, expectedState) {
			return true, fmt.Sprintf("Service startup type is %s", actualState), nil
		}
		return false, fmt.Sprintf("Service startup type is %s, expected %s", actualState, expectedState), nil

	case "startupitem":
		tokens, err := m.probeStartupTweak(ctx, tweak.Name)
		if err != nil {
			return false, "", err
		}
		ok, details := startupConforms(tokens)
		return ok, details, nil

	default: // Registry DWord, String, Binary, etc.
		psScript := fmt.Sprintf(`
$path = '%s'
$name = '%s'
if (Test-Path $path) {
    $val = Get-ItemPropertyValue -Path $path -Name $name -ErrorAction SilentlyContinue
    if ($val -ne $null) {
        Write-Output "VALUE:$val"
    } else {
        Write-Output "VALUE_NOT_SET"
    }
} else {
    Write-Output "PATH_NOT_FOUND"
}
`, psQuote(tweak.Path), psQuote(tweak.Name))
		cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-Command", psScript)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return false, "", fmt.Errorf("failed to read registry %s\\%s: %w", tweak.Path, tweak.Name, err)
		}
		outStr := strings.TrimSpace(string(out))
		expectedStr := fmt.Sprintf("%v", tweak.Value)
		if strings.HasPrefix(outStr, "VALUE:") {
			ok, details := matchRegistryValue(strings.TrimPrefix(outStr, "VALUE:"), expectedStr)
			return ok, details, nil
		}
		return false, outStr, nil
	}
}

// matchRegistryValue compares a raw registry readout against the manifest
// expectation. Numeric values normalize (YAML int/float vs registry DWORD
// int32) so 1 == 1 and 1.0 == 1 regardless of the parsed YAML type.
func matchRegistryValue(actualVal, expectedStr string) (bool, string) {
	if av, errA := strconv.ParseInt(actualVal, 10, 64); errA == nil {
		if ev, errE := strconv.ParseInt(expectedStr, 10, 64); errE == nil {
			actualVal = strconv.FormatInt(av, 10)
			expectedStr = strconv.FormatInt(ev, 10)
		}
	}
	if actualVal == expectedStr {
		return true, fmt.Sprintf("Registry value matches (%s)", actualVal)
	}
	return false, fmt.Sprintf("Registry value mismatch (actual: %s, expected: %s)", actualVal, expectedStr)
}

// CheckBatch checks many tweaks with one PowerShell spawn per family
// (registry, Appx, services, startup entries) instead of one per tweak.
// Feature/PSModule tweaks keep the single-check path. Results are
// order-preserving.
func (m *TweaksManager) CheckBatch(ctx context.Context, tweaks []entity.WindowsTweak) []entity.TweakCheckResult {
	results := make([]entity.TweakCheckResult, len(tweaks))
	for i, tw := range tweaks {
		results[i].Tweak = tw
	}
	if runtime.GOOS != "windows" {
		for i := range results {
			results[i].OK = true
			results[i].Details = "Skipped on non-Windows platform"
		}
		return results
	}

	var regIdx, appxIdx, svcIdx, startupIdx, singleIdx []int
	for i, tw := range tweaks {
		switch strings.ToLower(tw.Type) {
		case "appx":
			appxIdx = append(appxIdx, i)
		case "service":
			svcIdx = append(svcIdx, i)
		case "startupitem":
			startupIdx = append(startupIdx, i)
		case "feature", "psmodule":
			singleIdx = append(singleIdx, i)
		default: // registry DWord, String, QWord, Binary
			regIdx = append(regIdx, i)
		}
	}

	m.checkRegistryBatch(ctx, tweaks, regIdx, results)
	m.checkAppxBatch(ctx, tweaks, appxIdx, results)
	m.checkServiceBatch(ctx, tweaks, svcIdx, results)
	m.checkStartupBatch(ctx, tweaks, startupIdx, results)
	for _, i := range singleIdx {
		ok, details, err := m.CheckTweak(ctx, tweaks[i])
		results[i].OK = ok
		results[i].Details = details
		results[i].Err = err
	}
	return results
}

// checkRegistryBatch reads every registry tweak in one PowerShell spawn.
// Lines look like `DEBLOAT|||<id>|||VALUE:<val>` (or VALUE_NOT_SET /
// PATH_NOT_FOUND); ids come from the embedded manifest, quoted defensively.
func (m *TweaksManager) checkRegistryBatch(ctx context.Context, tweaks []entity.WindowsTweak, idx []int, results []entity.TweakCheckResult) {
	if len(idx) == 0 {
		return
	}
	var entryLines []string
	for _, i := range idx {
		entryLines = append(entryLines, fmt.Sprintf("  @{ Id = '%s'; Path = '%s'; Name = '%s' }",
			psQuote(tweaks[i].ID), psQuote(tweaks[i].Path), psQuote(tweaks[i].Name)))
	}
	// No trailing comma: `@( @{...}, )` is a parse error in powershell 5.1.
	script := fmt.Sprintf(`
$entries = @(
%s)
foreach ($e in $entries) {
  if (Test-Path $e.Path) {
    $val = Get-ItemPropertyValue -Path $e.Path -Name $e.Name -ErrorAction SilentlyContinue
    if ($val -ne $null) { Write-Output ("DEBLOAT|||" + $e.Id + "|||VALUE:" + $val) }
    else { Write-Output ("DEBLOAT|||" + $e.Id + "|||VALUE_NOT_SET") }
  } else {
    Write-Output ("DEBLOAT|||" + $e.Id + "|||PATH_NOT_FOUND")
  }
}
`, strings.Join(entryLines, ",\n"))
	cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-Command", script)
	out, err := cmd.CombinedOutput()
	if err != nil {
		for _, i := range idx {
			results[i].Err = fmt.Errorf("batch registry check failed: %w", err)
		}
		return
	}
	seen := map[string]bool{}
	for _, line := range strings.Split(string(out), "\n") {
		parts := strings.SplitN(strings.TrimSpace(line), "|||", 3)
		if len(parts) != 3 || parts[0] != "DEBLOAT" {
			continue
		}
		seen[parts[1]] = true
		for _, i := range idx {
			if tweaks[i].ID != parts[1] {
				continue
			}
			outcome := parts[2]
			if strings.HasPrefix(outcome, "VALUE:") {
				ok, details := matchRegistryValue(strings.TrimPrefix(outcome, "VALUE:"), fmt.Sprintf("%v", tweaks[i].Value))
				results[i].OK = ok
				results[i].Details = details
			} else {
				results[i].Details = outcome
			}
		}
	}
	for _, i := range idx {
		if !seen[tweaks[i].ID] {
			results[i].Err = fmt.Errorf("batch registry check returned no row for %q", tweaks[i].ID)
		}
	}
}

// checkAppxBatch lists installed Appx packages once and answers every Appx
// tweak from that set (conforming = absent).
func (m *TweaksManager) checkAppxBatch(ctx context.Context, tweaks []entity.WindowsTweak, idx []int, results []entity.TweakCheckResult) {
	if len(idx) == 0 {
		return
	}
	cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-Command",
		`Get-AppxPackage -AllUsers -ErrorAction SilentlyContinue | ForEach-Object { Write-Output ("DEBLOATAPPX|||" + $_.Name) }`)
	out, err := cmd.CombinedOutput()
	if err != nil {
		for _, i := range idx {
			results[i].Err = fmt.Errorf("batch Appx check failed: %w", err)
		}
		return
	}
	installed := map[string]bool{}
	for _, line := range strings.Split(string(out), "\n") {
		if name, ok := strings.CutPrefix(strings.TrimSpace(line), "DEBLOATAPPX|||"); ok && name != "" {
			installed[strings.ToLower(name)] = true
		}
	}
	for _, i := range idx {
		if installed[strings.ToLower(tweaks[i].Name)] {
			results[i].Details = "Appx package installed (will be removed on apply)"
		} else {
			results[i].OK = true
			results[i].Details = "Appx package not installed"
		}
	}
}

// checkServiceBatch queries the wanted services in one spawn. Services
// missing from the output are not present on the machine (conforming).
func (m *TweaksManager) checkServiceBatch(ctx context.Context, tweaks []entity.WindowsTweak, idx []int, results []entity.TweakCheckResult) {
	if len(idx) == 0 {
		return
	}
	quoted := make([]string, 0, len(idx))
	for _, i := range idx {
		quoted = append(quoted, "'"+psQuote(tweaks[i].Name)+"'")
	}
	// Per-name loop: a single Get-Service -Name a,b,c exits 1 when ANY name
	// is missing (SilentlyContinue only hides the message), which would fail
	// the whole batch. Missing services report NOT_PRESENT explicitly.
	script := fmt.Sprintf(`foreach ($n in @(%s)) {
  $s = Get-Service -Name $n -ErrorAction SilentlyContinue
  if ($s) { Write-Output ("DEBLOATSVC|||" + $s.Name + "|||" + $s.StartType.ToString()) }
  else { Write-Output ("DEBLOATSVC|||" + $n + "|||NOT_PRESENT") }
}`, strings.Join(quoted, ","))
	cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-Command", script)
	out, err := cmd.CombinedOutput()
	if err != nil {
		for _, i := range idx {
			results[i].Err = fmt.Errorf("batch service check failed: %w", err)
		}
		return
	}
	states := map[string]string{}
	for _, line := range strings.Split(string(out), "\n") {
		parts := strings.SplitN(strings.TrimSpace(line), "|||", 3)
		if len(parts) != 3 || parts[0] != "DEBLOATSVC" {
			continue
		}
		states[strings.ToLower(parts[1])] = parts[2]
	}
	for _, i := range idx {
		expectedState := serviceExpectedState(tweaks[i])
		actualState, present := states[strings.ToLower(tweaks[i].Name)]
		switch {
		case !present || actualState == "NOT_PRESENT":
			results[i].OK = true
			results[i].Details = "Service not present on system"
		case strings.EqualFold(actualState, expectedState):
			results[i].OK = true
			results[i].Details = fmt.Sprintf("Service startup type is %s", actualState)
		default:
			results[i].Details = fmt.Sprintf("Service startup type is %s, expected %s", actualState, expectedState)
		}
	}
}

// checkStartupBatch probes every wanted name in one spawn and answers from the
// resulting map (conforming = the probe found nothing to remove).
func (m *TweaksManager) checkStartupBatch(ctx context.Context, tweaks []entity.WindowsTweak, idx []int, results []entity.TweakCheckResult) {
	if len(idx) == 0 {
		return
	}
	names := make([]string, 0, len(idx))
	for _, i := range idx {
		names = append(names, tweaks[i].Name)
	}
	rows, err := m.probeStartup(ctx, names)
	if err != nil {
		for _, i := range idx {
			results[i].Err = err
		}
		return
	}
	for _, i := range idx {
		results[i].OK, results[i].Details = startupConforms(rows[strings.ToLower(tweaks[i].Name)])
	}
}

func (m *TweaksManager) ApplyTweak(ctx context.Context, tweak entity.WindowsTweak) error {
	if runtime.GOOS != "windows" {
		return nil
	}

	switch strings.ToLower(tweak.Type) {
	case "feature":
		psScript := fmt.Sprintf(`Enable-WindowsOptionalFeature -Online -FeatureName '%s' -NoRestart -ErrorAction Stop`, psQuote(tweak.Name))
		cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-Command", psScript)
		out, err := cmd.CombinedOutput()
		exitCode := 0
		if cmd.ProcessState != nil {
			exitCode = cmd.ProcessState.ExitCode()
		}
		m.logger.LogCommand("powershell.exe", []string{"-Command", psScript}, exitCode, string(out), err)
		if err != nil {
			return fmt.Errorf("failed to enable feature %s: %s (%w)", tweak.Name, string(out), err)
		}
		return nil

	case "psmodule":
		// CurrentUser scope on purpose: installing for all users needs
		// elevation, and envctl runs unprivileged.
		// Single install via Install-PSResource into the shared Documents
		// tree: both powershell.exe (5.1) and pwsh (7.x) list
		// Documents\PowerShell\Modules on this machine, and Import-Module
		// succeeds in both (validated live). A second Save-PSResource copy
		// is NOT created — it resolves in Get-Module -ListAvailable but
		// carries no valid manifest and fails at import.
		// -Repository PSGallery pins the source explicitly: with several
		// registered repositories, Install-PSResource would otherwise prompt
		// for one and hang the non-interactive provisioning shell.
		psScript := fmt.Sprintf(`
if (Get-Command Install-PSResource -ErrorAction SilentlyContinue) {
    $r = Get-PSResource -Name '%s' -Scope CurrentUser -ErrorAction SilentlyContinue
    if (-not $r) { Install-PSResource -Name '%s' -Scope CurrentUser -TrustRepository -Repository PSGallery -ErrorAction Stop }
} else {
    if (-not (Get-PackageProvider -Name NuGet -ErrorAction SilentlyContinue)) {
        Install-PackageProvider -Name NuGet -MinimumVersion 2.8.5.201 -Force | Out-Null
    }
    Set-PSRepository -Name PSGallery -InstallationPolicy Trusted -ErrorAction SilentlyContinue
    Install-Module -Name '%s' -Scope CurrentUser -Force -AllowClobber -ErrorAction Stop
}
Import-Module -Name '%s' -Force -ErrorAction Stop`, psQuote(tweak.Name), psQuote(tweak.Name), psQuote(tweak.Name), psQuote(tweak.Name))
		cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-Command", psScript)
		out, err := cmd.CombinedOutput()
		exitCode := 0
		if cmd.ProcessState != nil {
			exitCode = cmd.ProcessState.ExitCode()
		}
		m.logger.LogCommand("powershell.exe", []string{"-Command", psScript}, exitCode, string(out), err)
		if err != nil {
			return fmt.Errorf("failed to install PowerShell module %s: %s (%w)", tweak.Name, string(out), err)
		}
		return nil

	case "appx":
		// -AllUsers needs elevation; without it the error surfaces with the
		// admin hint added by the caller.
		appxScript := fmt.Sprintf(`Get-AppxPackage -Name '%s' -AllUsers -ErrorAction SilentlyContinue | Remove-AppxPackage -AllUsers -ErrorAction Stop`, psQuote(tweak.Name))
		appxCmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-Command", appxScript)
		appxOut, appErr := appxCmd.CombinedOutput()
		exitCode := 0
		if appxCmd.ProcessState != nil {
			exitCode = appxCmd.ProcessState.ExitCode()
		}
		m.logger.LogCommand("powershell.exe", []string{"-Command", appxScript}, exitCode, string(appxOut), appErr)
		if appErr != nil {
			return fmt.Errorf("failed to remove Appx package %s: %s (%w)", tweak.Name, string(appxOut), appErr)
		}
		return nil

	case "service":
		// Stop is best-effort (already stopped is fine); the startup-type
		// change is strict so elevation problems surface.
		expectedState := serviceExpectedState(tweak)
		var svcScript string
		switch strings.ToLower(expectedState) {
		case "disabled":
			svcScript = fmt.Sprintf(`Stop-Service -Name '%s' -Force -ErrorAction SilentlyContinue; Set-Service -Name '%s' -StartupType Disabled -ErrorAction Stop`, psQuote(tweak.Name), psQuote(tweak.Name))
		case "manual":
			svcScript = fmt.Sprintf(`Set-Service -Name '%s' -StartupType Manual -ErrorAction Stop`, psQuote(tweak.Name))
		default:
			return fmt.Errorf("unsupported service state %q for %s (want Disabled or Manual)", expectedState, tweak.Name)
		}
		svcCmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-Command", svcScript)
		svcOut, svcErr := svcCmd.CombinedOutput()
		exitCode := 0
		if svcCmd.ProcessState != nil {
			exitCode = svcCmd.ProcessState.ExitCode()
		}
		m.logger.LogCommand("powershell.exe", []string{"-Command", svcScript}, exitCode, string(svcOut), svcErr)
		if svcErr != nil {
			return fmt.Errorf("failed to set service %s to %s: %s (%w)", tweak.Name, expectedState, string(svcOut), svcErr)
		}
		return nil

	case "startupitem":
		// Re-probe with the same script the check uses, then delete only from
		// the targets it reported. The closed set is what keeps a service out
		// of reach, not a classifier over a free-form Location.
		tokens, err := m.probeStartupTweak(ctx, tweak.Name)
		if err != nil {
			return err
		}
		targets := startupTargetsForTokens(tokens)
		if len(targets) == 0 {
			// Converged by someone else between the two probes. Not an error:
			// the desired end state holds, so returning nil keeps `run debloat`
			// idempotent. Logged so the trace is not read as "we removed it".
			m.logger.Info("Startup entry %s no longer present at apply time; nothing to remove", tweak.Name)
			return nil
		}
		startupScript := startupRemovalScript(tweak.Name, targets)
		//nolint:gosec // G204: the name comes from the embedded manifest (same trust level as the package tables) and reaches PowerShell only as a single-quoted literal via psQuote, then through [WildcardPattern]::Escape; paths are the fixed Run keys or GetFolderPath results. No shell, no expansion. Asserted by TestStartupScriptsQuoteAdversarialNames and TestStartupScriptsNeutralizeWildcardNames.
		startupCmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-Command", startupScript)
		startupOut, startupErr := startupCmd.CombinedOutput()
		exitCode := 0
		if startupCmd.ProcessState != nil {
			exitCode = startupCmd.ProcessState.ExitCode()
		}
		m.logger.LogCommand("powershell.exe", []string{"-Command", startupScript}, exitCode, string(startupOut), startupErr)
		if startupErr != nil {
			return fmt.Errorf("failed to remove startup entry %s: %s (%w)", tweak.Name, string(startupOut), startupErr)
		}
		return nil

	default: // Registry
		valType := tweak.Type
		if valType == "" {
			valType = "DWord"
		}
		psScript := fmt.Sprintf(`
$path = '%s'
$name = '%s'
$val = %s
$type = '%s'
if (-not (Test-Path $path)) {
    New-Item -Path $path -Force | Out-Null
}
Set-ItemProperty -Path $path -Name $name -Value $val -Type $type -Force | Out-Null
`, psQuote(tweak.Path), psQuote(tweak.Name), psValue(tweak.Value), psQuote(valType))
		cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-Command", psScript)
		out, err := cmd.CombinedOutput()
		exitCode := 0
		if cmd.ProcessState != nil {
			exitCode = cmd.ProcessState.ExitCode()
		}
		m.logger.LogCommand("powershell.exe", []string{"-Command", psScript}, exitCode, string(out), err)
		if err != nil {
			return fmt.Errorf("failed to set registry %s\\%s: %s (%w)", tweak.Path, tweak.Name, string(out), err)
		}
		return nil
	}
}

func (m *TweaksManager) EnsureTweaks(ctx context.Context, tweaks []entity.WindowsTweak) ([]entity.Diagnostic, error) {
	var diags []entity.Diagnostic
	for _, tw := range tweaks {
		targetName := fmt.Sprintf("%s\\%s", tw.Path, tw.Name)
		if tw.Path == "" {
			targetName = fmt.Sprintf("[%s] %s", tw.Type, tw.Name)
		}

		ok, details, err := m.CheckTweak(ctx, tw)
		if err != nil {
			m.logger.Error("Windows tweak check failed for %s: %v", targetName, err)
			diags = append(diags, entity.Diagnostic{
				Category: entity.DiagError,
				System:   "Windows11",
				Target:   targetName,
				Details:  fmt.Sprintf("Failed to check tweak: %v", err),
			})
			continue
		}

		if ok {
			m.logger.LogIdempotency("Windows11", targetName, true, "Already configured correctly: "+details)
			diags = append(diags, entity.Diagnostic{
				Category: entity.DiagOK,
				System:   "Windows11",
				Target:   targetName,
				Details:  details,
			})
			continue
		}

		// Needs application
		m.logger.Info("Applying Windows tweak %s (current: %s)", targetName, details)
		if err := m.ApplyTweak(ctx, tw); err != nil {
			m.logger.Error("Failed to apply Windows tweak %s: %v", targetName, err)
			diags = append(diags, entity.Diagnostic{
				Category: entity.DiagError,
				System:   "Windows11",
				Target:   targetName,
				Details:  fmt.Sprintf("Failed to apply: %v", err),
				FixHint:  "Run terminal as Administrator if required for HKLM settings",
			})
		} else {
			m.logger.LogIdempotency("Windows11", targetName, false, "Applied successfully")
			diags = append(diags, entity.Diagnostic{
				Category: entity.DiagOK,
				System:   "Windows11",
				Target:   targetName,
				Details:  "Applied successfully",
			})
		}
	}
	return diags, nil
}
