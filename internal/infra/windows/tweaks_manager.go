package windows

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
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
		expectedState := "Disabled"
		if s, ok := tweak.Value.(string); ok && s != "" {
			expectedState = s
		}
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
// (registry, Appx, services) instead of one per tweak. Feature/PSModule
// tweaks keep the single-check path. Results are order-preserving.
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

	var regIdx, appxIdx, svcIdx, singleIdx []int
	for i, tw := range tweaks {
		switch strings.ToLower(tw.Type) {
		case "appx":
			appxIdx = append(appxIdx, i)
		case "service":
			svcIdx = append(svcIdx, i)
		case "feature", "psmodule":
			singleIdx = append(singleIdx, i)
		default: // registry DWord, String, QWord, Binary
			regIdx = append(regIdx, i)
		}
	}

	m.checkRegistryBatch(ctx, tweaks, regIdx, results)
	m.checkAppxBatch(ctx, tweaks, appxIdx, results)
	m.checkServiceBatch(ctx, tweaks, svcIdx, results)
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
		expectedState := "Disabled"
		if s, ok := tweaks[i].Value.(string); ok && s != "" {
			expectedState = s
		}
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
		expectedState := "Disabled"
		if s, ok := tweak.Value.(string); ok && s != "" {
			expectedState = s
		}
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
