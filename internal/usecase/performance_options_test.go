package usecase

import (
	"testing"

	"github.com/eajdias/envctl/internal/domain/entity"
)

func TestProbeRebootPending(t *testing.T) {
	cases := []struct {
		name    string
		present bool
		want    bool
	}{
		{"clean host", false, false},
		{"host waiting for a reboot", true, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			state := ProbeRebootPending(func(string) bool { return tc.present })
			if state.Pending != tc.want {
				t.Fatalf("Pending = %v, want %v", state.Pending, tc.want)
			}
			if state.Path != rebootRequiredPath {
				t.Fatalf("Path = %q, want %q", state.Path, rebootRequiredPath)
			}
			if tc.want && state.Detail == "" {
				t.Fatal("a pending state must carry a detail the operator can act on")
			}
		})
	}
}

func TestPerformanceOptionsMapping(t *testing.T) {
	t.Run("no-daemon-reexec disables the re-exec", func(t *testing.T) {
		limits := entity.LimitsSpec{Reexec: true}
		tz := entity.TimezoneSpec{Mode: "verify", Expected: "Etc/UTC"}
		PerformanceOptions{NoDaemonReexec: true}.applyTo(&limits, &tz)
		if limits.Reexec {
			t.Fatal("--no-daemon-reexec did not disable the re-exec")
		}
		if tz.Mode != "verify" {
			t.Fatal("--no-daemon-reexec must not touch the timezone spec")
		}
	})

	t.Run("an explicit timezone switches to enforce", func(t *testing.T) {
		limits := entity.LimitsSpec{Reexec: true}
		tz := entity.TimezoneSpec{Mode: "verify", Expected: "Etc/UTC"}
		PerformanceOptions{Timezone: "America/Sao_Paulo"}.applyTo(&limits, &tz)
		if tz.Mode != "enforce" || tz.Expected != "America/Sao_Paulo" {
			t.Fatalf("timezone spec = %#v, want enforce America/Sao_Paulo", tz)
		}
		if !limits.Reexec {
			t.Fatal("a timezone flag must not change the limits spec")
		}
	})

	t.Run("an empty timezone keeps the default verify mode", func(t *testing.T) {
		tz := entity.TimezoneSpec{Mode: "verify", Expected: "Etc/UTC"}
		PerformanceOptions{}.applyTo(nil, &tz)
		if tz.Mode != "verify" {
			t.Fatal("the default must stay verify-only: writing a timezone is never implicit")
		}
	})
}

func TestPerformanceOptionsValidate(t *testing.T) {
	if err := (PerformanceOptions{DebloatOnly: true, AllowDebloat: true}).Validate(); err == nil {
		t.Fatal("expected the redundant debloat flags to be rejected")
	}
	if err := (PerformanceOptions{DebloatOnly: true}).Validate(); err != nil {
		t.Fatalf("a debloat-only run is valid on its own: %v", err)
	}
	if err := (PerformanceOptions{AllowDebloat: true}).Validate(); err != nil {
		t.Fatalf("an allow-debloat run is valid: %v", err)
	}
	if err := (PerformanceOptions{}).Validate(); err != nil {
		t.Fatalf("the default is valid: %v", err)
	}
}

// The doctor must judge the declared policy, not merely report the host. An
// uncapped journal is worth reporting but is not a warning: a host nobody has
// provisioned is not a failure.
func TestDoctorJournaldUsesThePolicyAssessment(t *testing.T) {
	capped, category, detail := assessJournald(entity.JournaldState{})
	if capped {
		t.Fatal("an empty journald state is not capped")
	}
	if category != entity.DiagInfo {
		t.Fatalf("category = %q, want INFO so a fresh host stays at 0 WARN", category)
	}
	if detail == "" {
		t.Fatal("the detail must explain the risk")
	}
}
