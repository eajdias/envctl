package windows

import (
	"fmt"
	"strings"
)

// Tier 3 command names, the closed set of idempotent script pairs the
// debloat manifest can reference via `type: "Command"` + `name`. Like the
// startup-entry tokens, a name outside this set is rejected: nothing unknown
// ever reaches PowerShell.
const (
	tier3Teredo      = "Teredo"
	tier3PowerPlan   = "PowerPlan"
	tier3Hibernation = "Hibernation"
)

// highPerformanceGUID is the built-in Windows power scheme "High performance".
const highPerformanceGUID = "8c5e7fda-e8bf-4a96-9a85-a6e23a8c635c"

// commandScripts returns the idempotent check/apply PowerShell pair for a
// Tier 3 command tweak. The check prints CONFORMING when the desired state
// already holds; the apply transitions to it. Both run through powershell.exe
// as single commands, so no state is shared between them.
func commandScripts(name string) (check, apply string, err error) {
	switch name {
	case tier3Teredo:
		// Check probes the effective state; "disabled" appears in the
		// localized output as the state literal on every locale.
		return `
$out = netsh interface teredo show state 2>&1 | Out-String
if ($out -match 'disabled') { Write-Output "CONFORMING" } else { Write-Output "DRIFT" }
`, `netsh interface teredo set state disabled`, nil

	case tier3PowerPlan:
		// Check compares the active scheme GUID; apply activates it.
		return fmt.Sprintf(`
$out = powercfg /getactivescheme 2>&1 | Out-String
if ($out -match '%s') { Write-Output "CONFORMING" } else { Write-Output "DRIFT" }
`, highPerformanceGUID), fmt.Sprintf(`powercfg /setactive %s`, highPerformanceGUID), nil

	case tier3Hibernation:
		// Check: hibernation off removes hiberfil.sys. Applying it again is a
		// no-op (the file stays gone), so the check is both idempotent and
		// locale-independent.
		return `
if (Test-Path "$env:SystemRoot\hiberfil.sys") { Write-Output "DRIFT" } else { Write-Output "CONFORMING" }
`, `powercfg.exe /hibernate off`, nil

	default:
		return "", "", fmt.Errorf("unknown Tier 3 command %q (closed set: %s, %s, %s)", name, tier3Teredo, tier3PowerPlan, tier3Hibernation)
	}
}

// onedriveScripts returns the check/apply pair for the OneDrive removal.
// The check is conforming when no OneDrive process runs and the explorer
// namespace is gone — i.e. after the apply, or on a machine that never had
// OneDrive. The apply kills the process, uninstalls the setup component,
// removes the Run key and the two explorer namespace entries. It NEVER
// touches $env:USERPROFILE\OneDrive: the user's files stay.
func onedriveScripts() (check, apply string) {
	check = `
$proc = Get-Process -Name 'OneDrive' -ErrorAction SilentlyContinue
$clsid = Test-Path 'HKCR:\CLSID\{018D5C66-4533-4307-9B53-224DE2ED1FE6}'
if (-not $proc -and -not $clsid) { Write-Output "CONFORMING" } else { Write-Output "DRIFT" }
`
	apply = `
Stop-Process -Name 'OneDrive' -Force -ErrorAction SilentlyContinue
try { & "$env:SystemRoot\System32\OneDriveSetup.exe" /uninstall } catch {}
$run = 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Run'
if (Get-ItemProperty -Path $run -Name 'OneDrive' -ErrorAction SilentlyContinue) {
    Remove-ItemProperty -Path $run -Name 'OneDrive' -ErrorAction SilentlyContinue
}
Remove-Item -Path 'HKCR:\CLSID\{018D5C66-4533-4307-9B53-224DE2ED1FE6}' -Recurse -Force -ErrorAction SilentlyContinue
Remove-Item -Path 'HKCR:\Wow6432Node\CLSID\{018D5C66-4533-4307-9B53-224DE2ED1FE6}' -Recurse -Force -ErrorAction SilentlyContinue
`
	return check, apply
}

// registryExpectedStr renders a manifest registry value as the string the
// PowerShell readout is compared against. Binary values are declared as byte
// lists and PowerShell stringifies a byte[] as space-joined decimals
// ("144 18 3 128 16 0 0 0"); every other type keeps its %v form.
func registryExpectedStr(v any) string {
	switch t := v.(type) {
	case []any:
		return byteListString(t)
	case []int:
		return byteListString(t)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// byteListString joins a manifest byte list with spaces, matching the
// PowerShell stringification of a byte[].
func byteListString(v any) string {
	var parts []string
	switch t := v.(type) {
	case []any:
		for _, e := range t {
			parts = append(parts, fmt.Sprintf("%v", e))
		}
	case []int:
		for _, e := range t {
			parts = append(parts, fmt.Sprintf("%d", e))
		}
	}
	return strings.Join(parts, " ")
}

var _ = strings.Join // keep strings imported even if the switch above shrinks
