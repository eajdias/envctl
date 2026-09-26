package performance

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/eajdias/envctl/internal/domain/entity"
	"github.com/eajdias/envctl/internal/domain/repository"
)

const sysctlDropinPath = "/etc/sysctl.d/90-envctl-performance.conf"

type commandRunner func(ctx context.Context, name string, args ...string) ([]byte, error)

// effectiveValueFunc reads the value a sysctl key currently has on the host.
type effectiveValueFunc func(key string) (string, bool)

type sysctlManager struct {
	writer *dropinWriter
	read   effectiveValueFunc
}

// NewSysctlManager creates the production Linux sysctl adapter.
func NewSysctlManager() repository.SysctlManager {
	return newSysctlManager(
		sysctlDropinPath,
		func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return execCommand(ctx, name, args...)
		},
		time.Now,
		os.Geteuid() != 0,
		readSysctlValue,
	)
}

func newSysctlManager(
	destination string,
	run commandRunner,
	now func() time.Time,
	elevate bool,
	readers ...effectiveValueFunc,
) *sysctlManager {
	read := effectiveValueFunc(readSysctlValue)
	if len(readers) > 0 && readers[0] != nil {
		read = readers[0]
	}
	return &sysctlManager{
		writer: newDropinWriter(destination, run, now, elevate),
		read:   read,
	}
}

func (m *sysctlManager) Apply(ctx context.Context, settings []entity.SysctlSetting, dryRun bool) ([]entity.Diagnostic, error) {
	if len(settings) == 0 {
		return nil, nil
	}

	applicable, skipped, err := m.applyPolicy(settings)
	if err != nil {
		return skipped, err
	}
	if len(applicable) == 0 {
		return skipped, nil
	}

	content, err := renderSysctlConfig(applicable)
	if err != nil {
		return append(skipped, entity.Diagnostic{
			Category: entity.DiagError,
			System:   "Performance",
			Target:   m.writer.destination,
			Details:  err.Error(),
		}), err
	}

	if dryRun {
		return append(skipped, entity.Diagnostic{
			Category: entity.DiagInfo,
			System:   "Performance",
			Target:   m.writer.destination,
			Details:  fmt.Sprintf("would write %d sysctl setting(s) to %s", len(applicable), m.writer.destination),
		}), nil
	}

	changed, backup, err := m.writer.Install(content, 0o644, false)
	if err != nil {
		return append(skipped, entity.Diagnostic{
			Category: entity.DiagError,
			System:   "Performance",
			Target:   m.writer.destination,
			Details:  err.Error(),
		}), err
	}
	if !changed {
		return append(skipped, entity.Diagnostic{
			Category: entity.DiagOK,
			System:   "Performance",
			Target:   m.writer.destination,
			Details:  "sysctl drop-in already up to date",
		}), nil
	}

	if out, applyErr := m.writer.command(ctx, "sysctl", "-p", m.writer.destination); applyErr != nil {
		detail := fmt.Sprintf("apply sysctl drop-in failed: %v (%s)", applyErr, strings.TrimSpace(string(out)))
		if backup != "" {
			detail += fmt.Sprintf("; backup=%s", backup)
		}
		return append(skipped, entity.Diagnostic{
			Category: entity.DiagError,
			System:   "Performance",
			Target:   m.writer.destination,
			Details:  detail,
		}), applyErr
	}

	detail := "sysctl drop-in applied"
	if backup != "" {
		detail = fmt.Sprintf("sysctl drop-in applied (backup=%s)", backup)
	}
	return append(skipped, entity.Diagnostic{
		Category: entity.DiagOK,
		System:   "Performance",
		Target:   m.writer.destination,
		Details:  detail,
	}), nil
}

// applyPolicy splits the declared settings into the ones this host must be
// changed to and the ones it already satisfies. A min-policy key is never
// lowered: the fleet ships fs.file-max at the int64 ceiling and the previous
// manifest overwrote it with 2097152.
func (m *sysctlManager) applyPolicy(settings []entity.SysctlSetting) (applicable []entity.SysctlSetting, skipped []entity.Diagnostic, err error) {
	applicable = make([]entity.SysctlSetting, 0, len(settings))
	for _, setting := range settings {
		switch setting.Policy {
		case entity.SysctlPolicyMin, entity.SysctlPolicyMax:
			effective, ok := m.read(setting.Key)
			if !ok {
				detail := fmt.Sprintf("cannot read the current value of %s; refusing to apply a %q policy blind", setting.Key, setting.Policy)
				return nil, append(skipped, entity.Diagnostic{
					Category: entity.DiagError,
					System:   "Performance",
					Target:   setting.Key,
					Details:  detail,
				}), fmt.Errorf("%s", detail)
			}
			cmp := compareSysctlValues(effective, setting.Value)
			keep := (setting.Policy == entity.SysctlPolicyMin && cmp >= 0) ||
				(setting.Policy == entity.SysctlPolicyMax && cmp <= 0)
			if keep {
				skipped = append(skipped, entity.Diagnostic{
					Category: entity.DiagOK,
					System:   "Performance",
					Target:   setting.Key,
					Details: fmt.Sprintf("host value %s is at or above the declared %s; left untouched",
						effective, setting.Value),
				})
				continue
			}
		}
		applicable = append(applicable, setting)
	}
	return applicable, skipped, nil
}

func readSysctlValue(key string) (string, bool) {
	data, err := os.ReadFile(sysctlProcPath(key))
	if err != nil {
		return "", false
	}
	return strings.TrimSpace(string(data)), true
}

// sysctlProcPath maps a sysctl key to its procfs entry: every dot becomes a
// directory separator.
func sysctlProcPath(key string) string {
	return filepath.Join("/proc/sys", strings.ReplaceAll(strings.TrimSpace(key), ".", "/"))
}

// compareSysctlValues compares two sysctl values numerically. An unparseable
// side never compares equal, so a malformed manifest cannot look satisfied.
func compareSysctlValues(a, b string) int {
	av, aErr := strconv.ParseUint(strings.TrimSpace(a), 10, 64)
	bv, bErr := strconv.ParseUint(strings.TrimSpace(b), 10, 64)
	if aErr != nil || bErr != nil {
		return -1
	}
	switch {
	case av > bv:
		return 1
	case av < bv:
		return -1
	default:
		return 0
	}
}

func renderSysctlConfig(settings []entity.SysctlSetting) (string, error) {
	ordered := append([]entity.SysctlSetting(nil), settings...)
	sortSysctlSettings(ordered)

	var builder strings.Builder
	builder.WriteString("# Managed by envctl; review before editing.\n")
	for _, setting := range ordered {
		key := strings.TrimSpace(setting.Key)
		value := strings.TrimSpace(setting.Value)
		if key == "" || value == "" {
			return "", fmt.Errorf("sysctl key and value are required")
		}
		if strings.ContainsAny(key, "=\r\n#") || strings.ContainsAny(value, "\r\n#") {
			return "", fmt.Errorf("invalid sysctl setting %q=%q", key, value)
		}
		builder.WriteString(key)
		builder.WriteString(" = ")
		builder.WriteString(value)
		builder.WriteByte('\n')
	}
	return builder.String(), nil
}
