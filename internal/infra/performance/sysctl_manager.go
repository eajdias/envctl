package performance

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/eajdias/envctl/internal/domain/entity"
	"github.com/eajdias/envctl/internal/domain/repository"
)

const sysctlDropinPath = "/etc/sysctl.d/90-envctl-performance.conf"

type commandRunner func(ctx context.Context, name string, args ...string) ([]byte, error)

type sysctlManager struct {
	destination string
	run         commandRunner
	now         func() time.Time
	elevate     bool
}

// NewSysctlManager creates the production Linux sysctl adapter.
func NewSysctlManager() repository.SysctlManager {
	return newSysctlManager(
		sysctlDropinPath,
		func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return exec.CommandContext(ctx, name, args...).CombinedOutput()
		},
		time.Now,
		os.Geteuid() != 0,
	)
}

func newSysctlManager(destination string, run commandRunner, now func() time.Time, elevate bool) *sysctlManager {
	return &sysctlManager{
		destination: destination,
		run:         run,
		now:         now,
		elevate:     elevate,
	}
}

func (m *sysctlManager) Apply(ctx context.Context, settings []entity.SysctlSetting, dryRun bool) ([]entity.Diagnostic, error) {
	if len(settings) == 0 {
		return nil, nil
	}

	content, err := renderSysctlConfig(settings)
	if err != nil {
		return []entity.Diagnostic{{
			Category: entity.DiagError,
			System:   "Performance",
			Target:   m.destination,
			Details:  err.Error(),
		}}, err
	}

	if dryRun {
		return []entity.Diagnostic{{
			Category: entity.DiagInfo,
			System:   "Performance",
			Target:   m.destination,
			Details:  fmt.Sprintf("would write %d sysctl setting(s) to %s", len(settings), m.destination),
		}}, nil
	}

	existing, readErr := os.ReadFile(m.destination)
	if readErr == nil && string(existing) == content {
		return []entity.Diagnostic{{
			Category: entity.DiagOK,
			System:   "Performance",
			Target:   m.destination,
			Details:  "sysctl drop-in already up to date",
		}}, nil
	}
	if readErr != nil && !os.IsNotExist(readErr) {
		return []entity.Diagnostic{{
			Category: entity.DiagError,
			System:   "Performance",
			Target:   m.destination,
			Details:  fmt.Sprintf("read existing sysctl drop-in: %v", readErr),
		}}, readErr
	}

	backupPath := ""
	if readErr == nil {
		backupPath = m.nextBackupPath()
		if output, err := m.command(ctx, "cp", "-a", m.destination, backupPath); err != nil {
			return []entity.Diagnostic{{
				Category: entity.DiagError,
				System:   "Performance",
				Target:   m.destination,
				Details:  fmt.Sprintf("backup failed: %v (%s)", err, strings.TrimSpace(string(output))),
			}}, err
		}
	}

	tmp, err := os.CreateTemp("", "envctl-sysctl-")
	if err != nil {
		return []entity.Diagnostic{{
			Category: entity.DiagError,
			System:   "Performance",
			Target:   m.destination,
			Details:  fmt.Sprintf("create temporary sysctl file: %v", err),
		}}, err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if _, err := tmp.WriteString(content); err != nil {
		_ = tmp.Close()
		return []entity.Diagnostic{{
			Category: entity.DiagError,
			System:   "Performance",
			Target:   m.destination,
			Details:  fmt.Sprintf("write temporary sysctl file: %v", err),
		}}, err
	}
	if err := tmp.Chmod(0644); err != nil {
		_ = tmp.Close()
		return []entity.Diagnostic{{
			Category: entity.DiagError,
			System:   "Performance",
			Target:   m.destination,
			Details:  fmt.Sprintf("set temporary sysctl permissions: %v", err),
		}}, err
	}
	if err := tmp.Close(); err != nil {
		return []entity.Diagnostic{{
			Category: entity.DiagError,
			System:   "Performance",
			Target:   m.destination,
			Details:  fmt.Sprintf("close temporary sysctl file: %v", err),
		}}, err
	}

	tempDestination := m.nextTempPath()
	if output, err := m.command(ctx, "install", "-m", "0644", tmpName, tempDestination); err != nil {
		return []entity.Diagnostic{{
			Category: entity.DiagError,
			System:   "Performance",
			Target:   m.destination,
			Details:  fmt.Sprintf("install temporary sysctl drop-in failed: %v (%s)", err, strings.TrimSpace(string(output))),
		}}, err
	}
	// The temporary file is in the same directory as the live drop-in, so
	// rename is atomic and readers never observe a truncated sysctl file.
	if output, err := m.command(ctx, "mv", "-f", tempDestination, m.destination); err != nil {
		cleanupOutput, cleanupErr := m.command(ctx, "rm", "-f", tempDestination)
		cleanupDetail := ""
		if cleanupErr != nil {
			cleanupDetail = fmt.Sprintf("; temporary cleanup failed: %v (%s)", cleanupErr, strings.TrimSpace(string(cleanupOutput)))
		}
		return []entity.Diagnostic{{
			Category: entity.DiagError,
			System:   "Performance",
			Target:   m.destination,
			Details:  fmt.Sprintf("atomically replace sysctl drop-in failed: %v (%s)%s", err, strings.TrimSpace(string(output)), cleanupDetail),
		}}, err
	}

	if output, err := m.command(ctx, "sysctl", "-p", m.destination); err != nil {
		return []entity.Diagnostic{{
			Category: entity.DiagError,
			System:   "Performance",
			Target:   m.destination,
			Details:  fmt.Sprintf("apply sysctl drop-in failed: %v (%s); backup=%s", err, strings.TrimSpace(string(output)), backupPath),
		}}, err
	}

	detail := "sysctl drop-in applied"
	if backupPath != "" {
		detail = fmt.Sprintf("sysctl drop-in applied (backup=%s)", backupPath)
	}
	return []entity.Diagnostic{{
		Category: entity.DiagOK,
		System:   "Performance",
		Target:   m.destination,
		Details:  detail,
	}}, nil
}

func (m *sysctlManager) command(ctx context.Context, name string, args ...string) ([]byte, error) {
	if m.elevate {
		elevatedArgs := append([]string{"-n", name}, args...)
		return m.run(ctx, "sudo", elevatedArgs...)
	}
	return m.run(ctx, name, args...)
}

func (m *sysctlManager) nextBackupPath() string {
	stamp := m.now().Format("20060102-150405")
	candidate := fmt.Sprintf("%s.bak.%s", m.destination, stamp)
	for i := 1; ; i++ {
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate
		}
		candidate = fmt.Sprintf("%s.bak.%s-%d", m.destination, stamp, i)
	}
}

func (m *sysctlManager) nextTempPath() string {
	stamp := m.now().Format("20060102-150405")
	candidate := fmt.Sprintf("%s.tmp.%s", m.destination, stamp)
	for i := 1; ; i++ {
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate
		}
		candidate = fmt.Sprintf("%s.tmp.%s-%d", m.destination, stamp, i)
	}
}

func renderSysctlConfig(settings []entity.SysctlSetting) (string, error) {
	ordered := append([]entity.SysctlSetting(nil), settings...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Key < ordered[j].Key })

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
