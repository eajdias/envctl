package performance

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/eajdias/envctl/internal/domain/entity"
	"github.com/eajdias/envctl/internal/domain/repository"
)

// selectThreshold is the soft descriptor limit above which select(2) stops
// working. man 5 systemd.exec states the rule on LimitNOFILE: "Be careful when
// raising the soft limit above 1024, since select(2) cannot function with file
// descriptors >= 1024".
const selectThreshold = 1024

type limitsManager struct {
	system *dropinWriter
	pam    *dropinWriter
	paths  entity.LimitsSpec
}

// NewResourceLimitsManager creates the production file-descriptor limit adapter.
func NewResourceLimitsManager() repository.ResourceLimitsManager {
	run := func(ctx context.Context, name string, args ...string) ([]byte, error) {
		return execCommand(ctx, name, args...)
	}
	return newLimitsManager(
		newDropinWriter(defaultSystemLimitsDropin, run, nowFunc, needsElevation()),
		newDropinWriter(defaultPAMLimitsDropin, run, nowFunc, needsElevation()),
		entity.LimitsSpec{
			SystemDropin: defaultSystemLimitsDropin,
			PAMDropin:    defaultPAMLimitsDropin,
		},
	)
}

func newLimitsManager(system, pam *dropinWriter, paths entity.LimitsSpec) *limitsManager {
	return &limitsManager{system: system, pam: pam, paths: paths}
}

// Apply installs both limit drop-ins and, when a drop-in actually changed,
// re-executes the service manager so the new defaults are read.
//
// Two destinations are required because they are different mechanisms. The
// systemd drop-in sets the defaults for units the manager starts; the PAM
// drop-in covers login sessions. Writing only the PAM file, which is what a
// naive limits.d approach does, leaves every service untouched.
//
// The re-exec is the only operation in this package that touches PID 1. It runs
// only when a file changed, and it can be turned off with the
// --no-daemon-reexec flag; man 1 systemctl documents that all sockets stay
// accessible while the manager is being reexecuted.
func (m *limitsManager) Apply(ctx context.Context, spec entity.LimitsSpec, dryRun bool) ([]entity.Diagnostic, error) {
	systemContent, err := renderSystemLimits(spec)
	if err != nil {
		return []entity.Diagnostic{{
			Category: entity.DiagError,
			System:   "Performance",
			Target:   m.system.destination,
			Details:  err.Error(),
		}}, err
	}
	pamContent, err := renderPAMLimits(spec)
	if err != nil {
		return []entity.Diagnostic{{
			Category: entity.DiagError,
			System:   "Performance",
			Target:   m.pam.destination,
			Details:  err.Error(),
		}}, err
	}

	if dryRun {
		return []entity.Diagnostic{{
			Category: entity.DiagInfo,
			System:   "Performance",
			Target:   m.system.destination,
			Details: fmt.Sprintf("would write nofile soft=%s to %s and %s",
				orUnknown(strconv.Itoa(spec.NofileSoft)), m.system.destination, m.pam.destination),
		}}, nil
	}

	systemChanged, systemBackup, err := m.system.Install(systemContent, 0o644, false)
	if err != nil {
		return []entity.Diagnostic{{
			Category: entity.DiagError,
			System:   "Performance",
			Target:   m.system.destination,
			Details:  err.Error(),
		}}, err
	}
	pamChanged, pamBackup, err := m.pam.Install(pamContent, 0o644, false)
	if err != nil {
		return []entity.Diagnostic{{
			Category: entity.DiagError,
			System:   "Performance",
			Target:   m.pam.destination,
			Details:  err.Error(),
		}}, err
	}

	diags := []entity.Diagnostic{{
		Category: entity.DiagOK,
		System:   "Performance",
		Target:   m.system.destination,
		Details:  describeInstall("systemd", systemChanged, systemBackup),
	}, {
		Category: entity.DiagOK,
		System:   "Performance",
		Target:   m.pam.destination,
		Details:  describeInstall("PAM", pamChanged, pamBackup),
	}}

	if !systemChanged && !pamChanged {
		return diags, nil
	}

	if _, err := m.system.command(ctx, "systemctl", "daemon-reload"); err != nil {
		return append(diags, entity.Diagnostic{
			Category: entity.DiagError,
			System:   "Performance",
			Target:   "systemctl",
			Details:  fmt.Sprintf("daemon-reload failed: %v", err),
		}), err
	}

	if !spec.Reexec {
		return append(diags, entity.Diagnostic{
			Category: entity.DiagInfo,
			System:   "Performance",
			Target:   "systemd",
			Details: "the drop-ins were written but PID 1 was not re-executed (opt-out); " +
				"existing units keep their current limits until a reboot or a manual `systemctl daemon-reexec`",
		}), nil
	}

	if _, err := m.system.command(ctx, "systemctl", "daemon-reexec"); err != nil {
		return append(diags, entity.Diagnostic{
			Category: entity.DiagError,
			System:   "Performance",
			Target:   "systemd",
			Details:  fmt.Sprintf("daemon-reexec failed: %v; the drop-ins are written and apply on the next reboot", err),
		}), err
	}

	diags = append(diags, entity.Diagnostic{
		Category: entity.DiagOK,
		System:   "Performance",
		Target:   "systemd",
		Details:  "PID 1 re-executed; the new defaults apply to units started from now on",
	})

	if spec.NofileSoft > selectThreshold {
		diags = append(diags, entity.Diagnostic{
			Category: entity.DiagInfo,
			System:   "Performance",
			Target:   "DefaultLimitNOFILE",
			Details: fmt.Sprintf(
				"soft limit raised to %d, above the %d threshold: man 5 systemd.exec warns that select(2) "+
					"cannot use file descriptors >= 1024, so software still calling select(2) instead of poll/epoll "+
					"may misbehave. The hard limit was left at the host's own value.",
				spec.NofileSoft, selectThreshold,
			),
		})
	}
	return diags, nil
}

func describeInstall(kind string, changed bool, backup string) string {
	if !changed {
		return fmt.Sprintf("%s limit drop-in already up to date", kind)
	}
	if backup != "" {
		return fmt.Sprintf("%s limit drop-in applied (backup=%s)", kind, backup)
	}
	return fmt.Sprintf("%s limit drop-in applied", kind)
}

// renderSystemLimits writes the systemd manager drop-in.
//
// DefaultLimitNOFILE is emitted with an EMPTY hard side when the manifest does
// not declare one. That is deliberate: the fleet reports a hard limit of 524288
// on Oracle and 1048576 on AWS, so pinning a hard value would lower the AWS
// hosts. The "soft:" form was verified to parse cleanly with
// systemd-analyze cat-config and to leave the host's hard limit untouched.
func renderSystemLimits(spec entity.LimitsSpec) (string, error) {
	if spec.NofileSoft <= 0 {
		return "", fmt.Errorf("nofile_soft must be positive, got %d", spec.NofileSoft)
	}
	soft := strconv.Itoa(spec.NofileSoft)
	hard := strings.TrimSpace(spec.NofileHard)
	if hard != "" {
		if _, err := strconv.ParseUint(hard, 10, 64); err != nil {
			return "", fmt.Errorf("nofile_hard %q is not a number", hard)
		}
		if strings.ContainsAny(hard, "\r\n#") {
			return "", fmt.Errorf("invalid nofile_hard %q", hard)
		}
	}

	var builder strings.Builder
	builder.WriteString("# Managed by envctl; review before editing.\n")
	builder.WriteString("[Manager]\n")
	builder.WriteString("DefaultLimitNOFILE=")
	builder.WriteString(soft)
	builder.WriteString(":")
	builder.WriteString(hard)
	builder.WriteByte('\n')
	if spec.Nproc > 0 {
		builder.WriteString("DefaultLimitNPROC=")
		builder.WriteString(strconv.Itoa(spec.Nproc))
		builder.WriteByte('\n')
	}
	return builder.String(), nil
}

// renderPAMLimits writes the login-session drop-in. It never declares a hard
// limit: a PAM hard would replace the value the session already inherits, which
// is the same regression the systemd side avoids by leaving it empty.
func renderPAMLimits(spec entity.LimitsSpec) (string, error) {
	if spec.NofileSoft <= 0 {
		return "", fmt.Errorf("nofile_soft must be positive, got %d", spec.NofileSoft)
	}
	var builder strings.Builder
	builder.WriteString("# Managed by envctl; review before editing.\n")
	builder.WriteString("* soft nofile ")
	builder.WriteString(strconv.Itoa(spec.NofileSoft))
	builder.WriteByte('\n')
	if spec.Nproc > 0 {
		builder.WriteString("* soft nproc ")
		builder.WriteString(strconv.Itoa(spec.Nproc))
		builder.WriteByte('\n')
	}
	return builder.String(), nil
}
