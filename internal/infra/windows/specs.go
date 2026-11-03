package windows

import (
	"context"
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"time"
)

// SpecsFields is the closed, emission-ordered set of fields the specs probe
// may report. The probe is the single source of the collector contract: any
// field outside this set is an unknown token and fails the parse instead of
// being shown as data (same fail-closed rule as the startup probe).
var SpecsFields = []string{
	"hostname",
	"manufacturer",
	"model",
	"bios",
	"os",
	"os_arch",
	"install_date",
	"last_boot",
	"uptime",
	"timezone",
	"user",
	"cpu",
	"ram_total_gb",
	"ram_free_gb",
	"ram_modules",
	"disks",
	"gpus",
	"net",
	"net_ip",
	"usb_hubs",
	"usb3",
	"activation",
	"antivirus",
}

const (
	specsTag = "SPECS"
	// specsProbeTimeout is generous because the first WMI/CIM call on a cold
	// box can take a while; the doctor caps the whole section and treats a
	// probe failure as informational context, never as a warning.
	specsProbeTimeout = 60 * time.Second
)

// specsProbeScript emits one `SPECS|||field|||value` line per field of
// SpecsFields. Multi-valued data is joined with ";" by the emitter itself; the
// Emit helper sanitizes each value so a pipe or a line break can never break
// the protocol (pipes are regex-filtered, CR/LF become ";"). Absence of data
// is an empty value, reported explicitly, so the parse can verify full
// cardinality instead of guessing what was skipped.
func specsProbeScript() string {
	return fmt.Sprintf(`
$ErrorActionPreference = 'SilentlyContinue'
function Emit { param($field, $value)
  if ($null -eq $value) { $value = '' }
  $value = ([string]$value).Trim() -replace '\|', '/' -replace '\r', ';' -replace '\n', ';'
  Write-Output ("%s|||" + $field + "|||" + $value)
}
`, specsTag) + `
$cs = Get-CimInstance Win32_ComputerSystem
$os = Get-CimInstance Win32_OperatingSystem
$bios = Get-CimInstance Win32_BIOS
Emit 'hostname' $cs.Name
Emit 'manufacturer' $cs.Manufacturer
Emit 'model' $cs.Model
Emit 'bios' ($bios.SMBIOSBIOSVersion + ' (' + $bios.Manufacturer + ')')
Emit 'os' $os.Caption
Emit 'os_arch' $os.OSArchitecture
Emit 'install_date' ('{0:yyyy-MM-dd HH:mm}' -f $os.InstallDate)
Emit 'last_boot' ('{0:yyyy-MM-dd HH:mm}' -f $os.LastBootUpTime)
$up = (Get-Date) - $os.LastBootUpTime
Emit 'uptime' ('{0}d {1}h {2}m' -f $up.Days, $up.Hours, $up.Minutes)
Emit 'timezone' ((Get-TimeZone).Id)
Emit 'user' $cs.UserName
Emit 'cpu' (@(Get-CimInstance Win32_Processor).Name -join ';')
Emit 'ram_total_gb' ([math]::Round($os.TotalVisibleMemorySize / 1MB, 2))
Emit 'ram_free_gb' ([math]::Round($os.FreePhysicalMemory / 1MB, 2))
Emit 'ram_modules' (@(Get-CimInstance Win32_PhysicalMemory | ForEach-Object { '{0}GB {1}MHz' -f ($_.Capacity / 1GB), $_.Speed }) -join ';')
$disks = @()
foreach ($d in @(Get-CimInstance Win32_LogicalDisk -Filter 'DriveType=3')) {
  $letter = $d.DeviceID.Trim(':')
  $part = @(Get-Partition | Where-Object { $_.DriveLetter -eq $letter })[0]
  $phys = $null
  if ($part) { $phys = @(Get-PhysicalDisk | Where-Object { $_.DeviceID -eq $part.DiskNumber })[0] }
  $kind = 'unknown'
  if ($phys) {
    if ($phys.MediaType -eq 'SSD') {
      if ($phys.BusType -eq 'NVMe') { $kind = 'SSD NVMe' } else { $kind = 'SSD' }
    } elseif ($phys.MediaType -eq 'HDD') { $kind = 'HDD' }
  }
  $model = if ($phys) { $phys.FriendlyName } else { '-' }
  $disks += ('{0}: {1}GB free of {2}GB ({3}, {4}, {5})' -f $letter, [math]::Round($d.FreeSpace / 1GB, 1), [math]::Round($d.Size / 1GB, 1), $d.FileSystem, $kind, $model)
}
Emit 'disks' ($disks -join ';')
Emit 'gpus' (@(Get-CimInstance Win32_VideoController | Where-Object { $_.Name -notmatch 'Microsoft Basic' } | ForEach-Object { '{0} (driver {1})' -f $_.Name, $_.DriverVersion }) -join ';')
Emit 'net' (@(Get-NetAdapter | Where-Object { $_.Status -eq 'Up' -and $_.Name -notmatch 'Bluetooth' } | ForEach-Object { '{0} ({1}) {2} {3}' -f $_.Name, $_.InterfaceDescription, $_.MacAddress, $_.LinkSpeed }) -join ';')
Emit 'net_ip' (@(Get-NetIPAddress -AddressFamily IPv4 | Where-Object { $_.IPAddress -ne '127.0.0.1' -and $_.IPAddress -notmatch '^169\.' } | Select-Object -First 1).IPAddress)
Emit 'usb_hubs' (@(Get-PnpDevice | Where-Object { $_.FriendlyName -like '*USB Hub*' -and $_.Status -eq 'OK' }).Count)
$usb3 = @(Get-PnpDevice | Where-Object { ($_.FriendlyName -like '*USB 3.*' -or $_.FriendlyName -like '*xHCI*' -or $_.FriendlyName -like '*Extensible Host Controller*') -and $_.Status -eq 'OK' })
$usb3val = 'no'
if ($usb3.Count -gt 0) { $usb3val = 'yes' }
Emit 'usb3' $usb3val
$lic = @(Get-CimInstance -ClassName SoftwareLicensingProduct -Filter "PartialProductKey IS NOT NULL AND ApplicationId='55c92734-d682-4d71-983e-d6ec3f16059f'")[0]
$licval = 'unknown'
if ($lic) { if ($lic.LicenseStatus -eq 1) { $licval = 'activated' } else { $licval = 'not activated' } }
Emit 'activation' $licval
Emit 'antivirus' (@(Get-CimInstance -Namespace root/SecurityCenter2 -ClassName AntivirusProduct).displayName -join ';')
`
}

// SpecsProbe runs the PowerShell specs probe and returns the closed field
// map. A degraded readout is an error (unknown field, duplicate field, or
// missing field), so missing data can never be rendered as "absent" — the
// doctor would then certify a convergence that never happened.
func SpecsProbe(ctx context.Context) (map[string]string, error) {
	ctx, cancel := context.WithTimeout(ctx, specsProbeTimeout)
	defer cancel()
	//nolint:gosec // G204: fixed, repo-embedded script, no user input.
	cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-Command", specsProbeScript())
	out, err := cmd.CombinedOutput()
	if err != nil {
		head := strings.TrimSpace(string(out))
		if len(head) > 400 {
			head = head[:400]
		}
		return nil, fmt.Errorf("specs probe failed: %s (%w)", head, err)
	}
	return ParseSpecsProbe(string(out))
}

// knownSpecsField reports whether field belongs to the closed set.
func knownSpecsField(field string) bool {
	for _, f := range SpecsFields {
		if f == field {
			return true
		}
	}
	return false
}

// ParseSpecsProbe indexes a probe readout by field. A line with a different
// tag is ignored (host noise may interleave), but a line the collector itself
// emitted must be fully interpretable: unknown field name, duplicate field
// name, or missing field all fail closed, mirroring validateStartupRows.
func ParseSpecsProbe(out string) (map[string]string, error) {
	vals := make(map[string]string)
	for _, line := range strings.Split(out, "\n") {
		parts := strings.SplitN(strings.TrimSpace(line), "|||", 3)
		if len(parts) != 3 || parts[0] != specsTag {
			continue
		}
		if !knownSpecsField(parts[1]) {
			return nil, fmt.Errorf("specs probe reported unknown field %q", parts[1])
		}
		if _, dup := vals[parts[1]]; dup {
			return nil, fmt.Errorf("specs probe reported field %q twice", parts[1])
		}
		vals[parts[1]] = parts[2]
	}
	var missing []string
	for _, f := range SpecsFields {
		if _, ok := vals[f]; !ok {
			missing = append(missing, f)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return nil, fmt.Errorf("specs probe answered %d of %d fields (missing: %s)", len(vals), len(SpecsFields), strings.Join(missing, ", "))
	}
	return vals, nil
}
