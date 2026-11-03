package windows

import (
	"context"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"
)

// specsFieldsContract pins the closed set and the emission order of the specs
// collector. Any change here is a conscious contract change: the live probe,
// the doctor table, and the real host parity all key on this list.
func TestSpecsFieldsContractExpectations(t *testing.T) {
	want := []string{
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
	if len(SpecsFields) != len(want) {
		t.Fatalf("SpecsFields has %d entries, contract expects %d: %v", len(SpecsFields), len(want), SpecsFields)
	}
	for i, f := range want {
		if SpecsFields[i] != f {
			t.Errorf("SpecsFields[%d] = %q, contract expects %q (order is part of the contract)", i, SpecsFields[i], f)
		}
	}
}

func TestParseSpecsProbeFullReadout(t *testing.T) {
	values := map[string]string{}
	for _, f := range SpecsFields {
		values[f] = f + "-value"
	}
	var b strings.Builder
	for _, f := range SpecsFields {
		b.WriteString("SPECS|||" + f + "|||" + values[f] + "\n")
	}
	b.WriteString("host noise: an unrelated line that must be ignored\n")

	got, err := ParseSpecsProbe(b.String())
	if err != nil {
		t.Fatalf("ParseSpecsProbe failed on a full readout: %v", err)
	}
	if len(got) != len(SpecsFields) {
		t.Fatalf("parsed %d fields, want %d", len(got), len(SpecsFields))
	}
	for f, want := range values {
		if got[f] != want {
			t.Errorf("field %q = %q, want %q", f, got[f], want)
		}
	}
}

// TestParseSpecsProbeFailsClosed covers the three readout corruptions that
// must never pass as a healthy map: an unknown field, a duplicate field, and
// a missing field (the doctor would dismiss a convergence that never happened).
func TestParseSpecsProbeFailsClosed(t *testing.T) {
	full := func() string {
		var b strings.Builder
		for _, f := range SpecsFields {
			b.WriteString("SPECS|||" + f + "|||" + f + "-value\n")
		}
		return b.String()
	}

	t.Run("unknown field fails", func(t *testing.T) {
		_, err := ParseSpecsProbe(full() + "SPECS|||junkfield|||x\n")
		if err == nil || !strings.Contains(err.Error(), `unknown field "junkfield"`) {
			t.Errorf("want unknown-field error, got %v", err)
		}
	})

	t.Run("duplicate field fails", func(t *testing.T) {
		_, err := ParseSpecsProbe(full() + "SPECS|||cpu|||again\n")
		if err == nil || !strings.Contains(err.Error(), `field "cpu" twice`) {
			t.Errorf("want duplicate-field error, got %v", err)
		}
	})

	t.Run("missing fields fail", func(t *testing.T) {
		readout := full()
		readout = strings.ReplaceAll(readout, "SPECS|||cpu|||cpu-value\n", "")
		readout = strings.ReplaceAll(readout, "SPECS|||disks|||disks-value\n", "")
		_, err := ParseSpecsProbe(readout)
		if err == nil {
			t.Fatal("want missing-fields error, got nil")
		}
		for _, field := range []string{"cpu", "disks"} {
			if !strings.Contains(err.Error(), field) {
				t.Errorf("error %q must name the missing field %q", err, field)
			}
		}
	})
}

// TestSpecsProbeScript_EmitsEveryField makes the script and SpecsFields drift
// proof: the script must contain an emit line for every field of the set.
func TestSpecsProbeScript_EmitsEveryField(t *testing.T) {
	script := specsProbeScript()
	for _, f := range SpecsFields {
		if !strings.Contains(script, "Emit '"+f+"'") {
			t.Errorf("specs probe script does not emit field %q", f)
		}
	}
}

// TestSpecsProbeScript_DataSources checks the table of WMI/CIM sources the
// collector must query, so the absorbed capability cannot silently shrink.
func TestSpecsProbeScript_DataSources(t *testing.T) {
	sources := []string{
		"Win32_ComputerSystem",
		"Win32_OperatingSystem",
		"Win32_BIOS",
		"Win32_Processor",
		"Win32_PhysicalMemory",
		"Win32_LogicalDisk",
		"Win32_VideoController",
		"Get-Partition",
		"Get-PhysicalDisk",
		"Get-NetAdapter",
		"Get-NetIPAddress",
		"Get-PnpDevice",
		"SoftwareLicensingProduct",
		"root/SecurityCenter2",
		"Get-TimeZone",
	}
	script := strings.ToLower(specsProbeScript())
	for _, s := range sources {
		if !strings.Contains(script, strings.ToLower(s)) {
			t.Errorf("specs probe script does not query source %q", s)
		}
	}
}

// TestSpecsProbeScript_ZeroPII mirrors the extras-manifest hygiene rule: this
// collector was absorbed from a private utility repo and never carries
// business apps or personal data.
func TestSpecsProbeScript_ZeroPII(t *testing.T) {
	forbidden := []string{"zscan", "zscansoftware", "whatsapp", "discord", "spotify", "path of exile", "npcap"}
	script := strings.ToLower(specsProbeScript())
	for _, f := range forbidden {
		if strings.Contains(script, f) {
			t.Errorf("specs probe script mentions forbidden business term %q (public repo, zero PII)", f)
		}
	}
}

// TestSpecsLiveProbe runs the real probe on Windows when the operator opts in
// (SPECS_LIVE_PROBE=1): a converged readout must fail the parse for no field.
func TestSpecsLiveProbe(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("live probe requires Windows")
	}
	if os.Getenv("SPECS_LIVE_PROBE") != "1" {
		t.Skip("set SPECS_LIVE_PROBE=1 to run the live probe")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	rows, err := SpecsProbe(ctx)
	if err != nil {
		t.Fatalf("SpecsProbe failed on the live host: %v", err)
	}
	for _, f := range SpecsFields {
		if _, ok := rows[f]; !ok {
			t.Errorf("live readout missed field %q (parse must guarantee cardinality)", f)
		}
	}
	for _, field := range []string{"hostname", "cpu", "ram_total_gb", "disks", "os"} {
		if strings.TrimSpace(rows[field]) == "" {
			t.Errorf("live readout: field %q is empty on a converged host", field)
		}
	}
	if diag := rows["disks"]; !strings.Contains(diag, ":") {
		t.Errorf("live readout: disks %q must carry a drive letter and sizes", diag)
	}
}
