package performance

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/eajdias/envctl/internal/domain/entity"
	"github.com/eajdias/envctl/internal/domain/repository"
)

// Timezone verification modes.
const (
	// TimezoneModeVerify reports the host's zone and never writes.
	TimezoneModeVerify = "verify"
	// TimezoneModeEnforce writes the declared zone when the host differs.
	TimezoneModeEnforce = "enforce"
)

type timezoneManager struct {
	root     string
	readZone func(context.Context, string, ...string) ([]byte, error)
	run      commandRunner
	now      func() time.Time
	elevate  bool
}

// NewTimezoneManager creates the production timezone adapter.
func NewTimezoneManager() repository.TimezoneManager {
	return newTimezoneManager(
		"/",
		func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return execCommand(ctx, name, args...)
		},
		func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return execCommand(ctx, name, args...)
		},
		time.Now,
		os.Geteuid() != 0,
	)
}

func newTimezoneManager(
	root string,
	readZone func(context.Context, string, ...string) ([]byte, error),
	run commandRunner,
	now func() time.Time,
	elevate bool,
) *timezoneManager {
	return &timezoneManager{root: root, readZone: readZone, run: run, now: now, elevate: elevate}
}

// Current reports the host's timezone, preferring timedatectl and falling back
// to /etc/timezone for images where the binary is absent.
func (m *timezoneManager) Current(ctx context.Context) (string, error) {
	if m.readZone != nil {
		if out, err := m.readZone(ctx, "timedatectl", "show", "-p", "Timezone", "--value"); err == nil {
			if zone := strings.TrimSpace(string(out)); zone != "" {
				return zone, nil
			}
		}
	}
	if data, err := os.ReadFile(filepath.Join(m.root, "etc", "timezone")); err == nil {
		if zone := strings.TrimSpace(string(data)); zone != "" {
			return zone, nil
		}
	}
	return "", fmt.Errorf(
		"could not determine the host timezone: timedatectl returned nothing and %s is unreadable",
		filepath.Join(m.root, "etc", "timezone"),
	)
}

// Apply verifies the host against the declared policy and, only in enforce
// mode, writes the declared zone.
//
// Verify is the default because the timezone is an operator's deliberate
// choice, not a tuning knob: reporting a mismatch as INFO leaves the decision
// with the operator, while writing it by default would silently change log
// timestamps and scheduled jobs on production hosts.
func (m *timezoneManager) Apply(ctx context.Context, spec entity.TimezoneSpec, dryRun bool) ([]entity.Diagnostic, error) {
	mode := spec.Mode
	if mode == "" {
		mode = TimezoneModeVerify
	}
	if mode != TimezoneModeVerify && mode != TimezoneModeEnforce {
		return []entity.Diagnostic{{
			Category: entity.DiagError,
			System:   "Timezone",
			Target:   "timezone",
			Details:  fmt.Sprintf("unknown timezone mode %q; use %q or %q", mode, TimezoneModeVerify, TimezoneModeEnforce),
		}}, fmt.Errorf("unknown timezone mode %q", mode)
	}
	if strings.TrimSpace(spec.Expected) == "" {
		return []entity.Diagnostic{{
			Category: entity.DiagError,
			System:   "Timezone",
			Target:   "timezone",
			Details:  "no expected timezone is declared",
		}}, fmt.Errorf("no expected timezone is declared")
	}

	current, err := m.Current(ctx)
	if err != nil {
		return []entity.Diagnostic{{
			Category: entity.DiagError,
			System:   "Timezone",
			Target:   "timezone",
			Details:  err.Error(),
		}}, err
	}

	if current == spec.Expected {
		return []entity.Diagnostic{{
			Category: entity.DiagOK,
			System:   "Timezone",
			Target:   current,
			Details:  "host timezone matches the declared zone",
		}}, nil
	}

	if mode == TimezoneModeVerify {
		return []entity.Diagnostic{{
			Category: entity.DiagInfo,
			System:   "Timezone",
			Target:   current,
			Details: fmt.Sprintf("host timezone is %s, declared %s; verify mode does not write (use an explicit enforce to change it)",
				current, spec.Expected),
		}}, nil
	}

	if err := m.validateZone(spec.Expected); err != nil {
		return []entity.Diagnostic{{
			Category: entity.DiagError,
			System:   "Timezone",
			Target:   spec.Expected,
			Details:  err.Error(),
		}}, err
	}
	if dryRun {
		return []entity.Diagnostic{{
			Category: entity.DiagInfo,
			System:   "Timezone",
			Target:   current,
			Details:  fmt.Sprintf("would set the timezone to %s", spec.Expected),
		}}, nil
	}

	if out, err := m.command(ctx, "timedatectl", "set-timezone", spec.Expected); err != nil {
		detail := fmt.Sprintf("setting the timezone to %s failed: %v (%s)", spec.Expected, err, strings.TrimSpace(string(out)))
		if strings.Contains(detail, "executable file not found") || strings.Contains(detail, "no such file") {
			detail = "timedatectl is not available on this host; install the tzdata/timedatectl package first"
		}
		return []entity.Diagnostic{{
			Category: entity.DiagError,
			System:   "Timezone",
			Target:   current,
			Details:  detail,
		}}, fmt.Errorf("%s", detail)
	}

	return []entity.Diagnostic{{
		Category: entity.DiagOK,
		System:   "Timezone",
		Target:   spec.Expected,
		Details:  fmt.Sprintf("timezone set from %s", current),
	}}, nil
}

// validateZone refuses a name the host does not know before the write, so a
// typo is never reported as a successful change.
func (m *timezoneManager) validateZone(zone string) error {
	// A slash is legal in an IANA name ("Etc/UTC"), so only path traversal and
	// whitespace are refused here.
	if zone != strings.TrimSpace(zone) || strings.ContainsAny(zone, " \t\r\n") || strings.Contains(zone, "..") {
		return fmt.Errorf("invalid timezone name %q", zone)
	}
	if _, err := os.Stat(filepath.Join(m.root, "usr", "share", "zoneinfo", zone)); err != nil {
		return fmt.Errorf("unknown timezone %q: not present in the host zoneinfo database", zone)
	}
	return nil
}

func (m *timezoneManager) command(ctx context.Context, name string, args ...string) ([]byte, error) {
	if m.elevate {
		return m.run(ctx, "sudo", append([]string{"-n", name}, args...)...)
	}
	return m.run(ctx, name, args...)
}
