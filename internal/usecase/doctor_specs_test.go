package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/eajdias/envctl/internal/domain/entity"
	"github.com/eajdias/envctl/internal/infra/windows"
)

func fakeSpecsReadout() map[string]string {
	readout := make(map[string]string)
	for _, f := range windows.SpecsFields {
		readout[f] = f + "-value"
	}
	return readout
}

func TestAuditSystemSpecs_SkipsNonWindows(t *testing.T) {
	var diags []entity.Diagnostic
	uc := &DoctorAuditUseCase{
		platform: func() entity.PlatformInfo {
			return entity.PlatformInfo{GOOS: "linux", Family: "debian", ID: "ubuntu", VersionID: "26.04"}
		},
		logger: &mockLogger{},
	}
	uc.auditSystemSpecs(context.Background(), func(d entity.Diagnostic) {
		diags = append(diags, d)
	})
	if len(diags) != 0 {
		t.Fatalf("non-Windows host must be silent, got %d diagnostics: %+v", len(diags), diags)
	}
}

func TestAuditSystemSpecs_EmitsHardwareInfos(t *testing.T) {
	uc := &DoctorAuditUseCase{
		platform: func() entity.PlatformInfo {
			return entity.PlatformInfo{GOOS: "windows"}
		},
		specsProbe: func(ctx context.Context) (map[string]string, error) {
			return fakeSpecsReadout(), nil
		},
		logger: &mockLogger{},
	}
	var diags []entity.Diagnostic
	uc.auditSystemSpecs(context.Background(), func(d entity.Diagnostic) {
		diags = append(diags, d)
	})

	if len(diags) != len(windows.SpecsFields) {
		t.Fatalf("expected one Info per field (%d), got %d", len(windows.SpecsFields), len(diags))
	}
	for i, d := range diags {
		if d.Category != entity.DiagInfo {
			t.Errorf("diag[%d] (%s) must be INFO, got %q", i, d.Target, d.Category)
		}
		if d.System != "Hardware" {
			t.Errorf("diag[%d] belongs to System %q, want \"Hardware\"", i, d.System)
		}
		target := windows.SpecsFields[i]
		if d.Target != target {
			t.Errorf("diag[%d].Target = %q, want %q (emission order follows SpecsFields)", i, d.Target, target)
		}
		if d.Details != target+"-value" {
			t.Errorf("diag[%d] details %q, want %q", i, d.Details, target+"-value")
		}
	}
}

// TestAuditSystemSpecs_EmptyValueIsNotDetected locks the display rules: an
// empty field renders "not detected" and ";"-joined values get spacing, so
// the doctor table stays readable for multi-valued data.
func TestAuditSystemSpecs_EmptyValueIsNotDetected(t *testing.T) {
	readout := fakeSpecsReadout()
	readout["antivirus"] = ""
	readout["disks"] = "C: 100GB free of 250GB (NTFS, SSD NVMe, Samsung 980);D: 40GB free of 200GB (FAT32, HDD, WDC)"
	uc := &DoctorAuditUseCase{
		platform: func() entity.PlatformInfo { return entity.PlatformInfo{GOOS: "windows"} },
		specsProbe: func(ctx context.Context) (map[string]string, error) {
			return readout, nil
		},
		logger: &mockLogger{},
	}
	var diags []entity.Diagnostic
	uc.auditSystemSpecs(context.Background(), func(d entity.Diagnostic) {
		diags = append(diags, d)
	})
	byTarget := map[string]entity.Diagnostic{}
	for _, d := range diags {
		byTarget[d.Target] = d
	}
	if got := byTarget["antivirus"].Details; got != "not detected" {
		t.Errorf("empty antivirus must render \"not detected\", got %q", got)
	}
	want := "C: 100GB free of 250GB (NTFS, SSD NVMe, Samsung 980); D: 40GB free of 200GB (FAT32, HDD, WDC)"
	if got := byTarget["disks"].Details; got != want {
		t.Errorf("disks details: want %q, got %q", want, got)
	}
}

// TestAuditSystemSpecs_ProbeFailureIsInfo locks the "without judgment"
// contract: a collector failure is context for the operator, never a WARN
// that would poison the 0-WARN goal for an optional inventory.
func TestAuditSystemSpecs_ProbeFailureIsInfo(t *testing.T) {
	uc := &DoctorAuditUseCase{
		platform: func() entity.PlatformInfo { return entity.PlatformInfo{GOOS: "windows"} },
		specsProbe: func(ctx context.Context) (map[string]string, error) {
			return nil, errors.New("powershell.exe exited with status 1")
		},
		logger: &mockLogger{},
	}
	var diags []entity.Diagnostic
	uc.auditSystemSpecs(context.Background(), func(d entity.Diagnostic) {
		diags = append(diags, d)
	})
	if len(diags) != 1 {
		t.Fatalf("a probe failure must yield exactly one diagnostic, got %d", len(diags))
	}
	d := diags[0]
	if d.Category != entity.DiagInfo {
		t.Errorf("probe failure must be INFO, got %q", d.Category)
	}
	if d.Target != "specs-probe" {
		t.Errorf("probe failure target = %q, want %q", d.Target, "specs-probe")
	}
}
