package performance

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/eajdias/envctl/internal/domain/entity"
	"github.com/eajdias/envctl/internal/domain/repository"
)

type journaldManager struct {
	writer *dropinWriter
}

// NewJournaldManager creates the production journald drop-in adapter.
func NewJournaldManager() repository.JournaldManager {
	return newJournaldManager(newDropinWriter(
		defaultJournaldDropin,
		func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return execCommand(ctx, name, args...)
		},
		nowFunc,
		needsElevation(),
	))
}

func newJournaldManager(writer *dropinWriter) *journaldManager {
	return &journaldManager{writer: writer}
}

// Apply installs the journald drop-in and restarts the service.
//
// The verb is restart, never stop: man 8 systemd-journald documents that a
// restart preserves the stream connections the service manager holds, and that
// stopping the service is explicitly not recommended. An SSH session that is
// mid-command would otherwise lose its output stream.
func (m *journaldManager) Apply(ctx context.Context, spec entity.JournaldSpec, dryRun bool) ([]entity.Diagnostic, error) {
	content, err := renderJournaldConfig(spec)
	if err != nil {
		return []entity.Diagnostic{{
			Category: entity.DiagError,
			System:   "Performance",
			Target:   m.writer.destination,
			Details:  err.Error(),
		}}, err
	}

	if dryRun {
		return []entity.Diagnostic{{
			Category: entity.DiagInfo,
			System:   "Performance",
			Target:   m.writer.destination,
			Details:  fmt.Sprintf("would write %d journald setting(s) to %s", len(spec.Values), m.writer.destination),
		}}, nil
	}

	changed, backup, err := m.writer.Install(content, 0o644, false)
	if err != nil {
		return []entity.Diagnostic{{
			Category: entity.DiagError,
			System:   "Performance",
			Target:   m.writer.destination,
			Details:  err.Error(),
		}}, err
	}
	if !changed {
		return []entity.Diagnostic{{
			Category: entity.DiagOK,
			System:   "Performance",
			Target:   m.writer.destination,
			Details:  "journald drop-in already up to date",
		}}, nil
	}

	if out, err := m.writer.command(ctx, "systemctl", "restart", journaldService); err != nil {
		detail := fmt.Sprintf("restarting %s failed: %v (%s)", journaldService, err, strings.TrimSpace(string(out)))
		if backup != "" {
			detail += "; backup=" + backup
		}
		return []entity.Diagnostic{{
			Category: entity.DiagError,
			System:   "Performance",
			Target:   m.writer.destination,
			Details:  detail,
		}}, err
	}

	detail := "journald drop-in applied"
	if backup != "" {
		detail = fmt.Sprintf("journald drop-in applied (backup=%s)", backup)
	}
	return []entity.Diagnostic{{
		Category: entity.DiagOK,
		System:   "Performance",
		Target:   m.writer.destination,
		Details:  detail,
	}}, nil
}

// allowedJournaldKeys is the set of size/retention knobs this tool manages.
// Anything else is refused so a manifest edit cannot reconfigure forwarding or
// storage, which are operator decisions with different blast radius.
var allowedJournaldKeys = map[string]bool{
	"SystemMaxUse":       true,
	"SystemKeepFree":     true,
	"SystemMaxFileSize":  true,
	"SystemMaxFiles":     true,
	"RuntimeMaxUse":      true,
	"RuntimeKeepFree":    true,
	"RuntimeMaxFileSize": true,
	"RuntimeMaxFiles":    true,
	"MaxRetentionSec":    true,
}

func renderJournaldConfig(spec entity.JournaldSpec) (string, error) {
	if strings.TrimSpace(spec.Dropin) == "" {
		return "", fmt.Errorf("journald drop-in path is required")
	}
	ordered := append([]entity.JournaldSetting(nil), spec.Values...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Key < ordered[j].Key })

	var builder strings.Builder
	builder.WriteString("# Managed by envctl; review before editing.\n")
	builder.WriteString("[Journal]\n")
	for _, setting := range ordered {
		key := strings.TrimSpace(setting.Key)
		value := strings.TrimSpace(setting.Value)
		if !allowedJournaldKeys[key] {
			return "", fmt.Errorf("journald key %q is not managed by envctl", key)
		}
		if value == "" {
			return "", fmt.Errorf("journald key %q has no value", key)
		}
		if strings.ContainsAny(key, "=\r\n#[]") || strings.ContainsAny(value, "\r\n#") {
			return "", fmt.Errorf("invalid journald setting %q=%q", key, value)
		}
		builder.WriteString(key)
		builder.WriteString("=")
		builder.WriteString(value)
		builder.WriteByte('\n')
	}
	return builder.String(), nil
}

// AssessJournaldPolicy judges a host's effective journald configuration.
//
// man 5 journald.conf: journald honours both SystemMaxUse and SystemKeepFree and
// uses the smaller of the two. So a host that declares either one is bounded,
// and a host that declares neither falls back to the built-in default of 10% of
// the filesystem, which on the fleet's 45 GB disks is roughly 4.5 GB of journal
// on a 1 GB machine. That is worth reporting, but as information rather than a
// warning: the tool reports the effective state so a converged host stays at
// zero warnings.
func AssessJournaldPolicy(state entity.JournaldState) (capped bool, category entity.DiagnosticStatus, detail string) {
	hasMaxUse := strings.TrimSpace(state.SystemMaxUse) != ""
	hasKeepFree := strings.TrimSpace(state.SystemKeepFree) != ""

	switch {
	case hasMaxUse && hasKeepFree:
		return true, entity.DiagOK, fmt.Sprintf(
			"journald is bounded by SystemMaxUse=%s and SystemKeepFree=%s; the smaller of the two applies",
			state.SystemMaxUse, state.SystemKeepFree,
		)
	case hasMaxUse:
		return true, entity.DiagOK, fmt.Sprintf(
			"journald is bounded by SystemMaxUse=%s, but no SystemKeepFree floor is declared",
			state.SystemMaxUse,
		)
	case hasKeepFree:
		return false, entity.DiagInfo, fmt.Sprintf(
			"only SystemKeepFree=%s is declared, which reserves space without capping the journal; "+
				"the default cap is 10%% of the filesystem", state.SystemKeepFree,
		)
	default:
		return false, entity.DiagInfo, fmt.Sprintf(
			"journald has no explicit cap; the built-in default is 10%% of the filesystem (current usage: %s)",
			orUnknown(state.DiskUsage),
		)
	}
}

// orUnknown keeps a diagnostic readable when a probe returned nothing.
func orUnknown(value string) string {
	if strings.TrimSpace(value) == "" {
		return "unknown"
	}
	return value
}
