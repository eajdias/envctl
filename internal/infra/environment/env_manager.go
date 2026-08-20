package environment

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/eajdias/envctl/internal/domain/entity"
	"github.com/eajdias/envctl/internal/domain/repository"
)

type envManager struct{}

// NewWindowsEnvManager creates an environment variable manager for Windows and POSIX.
func NewWindowsEnvManager() repository.WindowsEnvManager {
	return &envManager{}
}

// psQuote escapes single quotes for safe embedding in a PowerShell string literal.
func psQuote(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

func (e *envManager) GetEnvVar(scope, name string) (string, error) {
	if runtime.GOOS != "windows" {
		if val := os.Getenv(name); val != "" {
			return val, nil
		}
		return e.getEnvVarFromRC(name)
	}
	psCmd := fmt.Sprintf("[System.Environment]::GetEnvironmentVariable('%s', '%s')", psQuote(name), psQuote(scope))
	cmd := exec.Command("powershell.exe", "-NoProfile", "-Command", psCmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func (e *envManager) SetEnvVar(scope, name, value string) error {
	if runtime.GOOS != "windows" {
		// Persist for future shells and set for the current process.
		if err := e.persistEnvVarPOSIX(name, value); err != nil {
			return err
		}
		return os.Setenv(name, value)
	}
	psCmd := fmt.Sprintf("[System.Environment]::SetEnvironmentVariable('%s', '%s', '%s')", psQuote(name), psQuote(value), psQuote(scope))
	cmd := exec.Command("powershell.exe", "-NoProfile", "-Command", psCmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to set env var %s=%s (%s): %s (%w)", name, value, scope, string(out), err)
	}
	return nil
}

// posixRCFiles returns the shell rc files where POSIX environment variables are persisted.
func (e *envManager) posixRCFiles() []string {
	home := os.Getenv("HOME")
	if home == "" {
		if h, err := os.UserHomeDir(); err == nil {
			home = h
		}
	}
	if home == "" {
		return nil
	}
	return []string{
		filepath.Join(home, ".profile"),
		filepath.Join(home, ".bashrc"),
	}
}

// persistEnvVarPOSIX writes `export NAME="value"` into ~/.profile and ~/.bashrc,
// replacing any existing declaration so the variable survives shell restarts.
func (e *envManager) persistEnvVarPOSIX(name, value string) error {
	exportLine := fmt.Sprintf("export %s=%q", name, value)
	for _, rc := range e.posixRCFiles() {
		if rc == "" {
			continue
		}
		data, err := os.ReadFile(rc)
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to read %s: %w", rc, err)
		}
		existing := strings.TrimSpace(string(data))
		var lines []string
		if existing != "" {
			lines = strings.Split(existing, "\n")
		}
		var out []string
		replaced := false
		for _, line := range lines {
			if strings.HasPrefix(line, "export "+name+"=") {
				if !replaced {
					out = append(out, exportLine)
					replaced = true
				}
				continue
			}
			out = append(out, line)
		}
		if !replaced {
			out = append(out, exportLine)
		}
		content := strings.Join(out, "\n") + "\n"
		if err := os.MkdirAll(filepath.Dir(rc), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(rc, []byte(content), 0600); err != nil {
			return fmt.Errorf("failed to write %s: %w", rc, err)
		}
	}
	return nil
}

// getEnvVarFromRC reads the current value of a variable from the shell rc files.
func (e *envManager) getEnvVarFromRC(name string) (string, error) {
	prefix := "export " + name + "="
	for _, rc := range e.posixRCFiles() {
		if rc == "" {
			continue
		}
		data, err := os.ReadFile(rc)
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(data), "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, prefix) {
				val := strings.TrimPrefix(trimmed, prefix)
				val = strings.Trim(val, "\"'")
				if val != "" {
					return val, nil
				}
			}
		}
	}
	return "", nil
}

func (e *envManager) EnsureEnvVars(ctx context.Context, vars []entity.EnvironmentVar) ([]entity.Diagnostic, error) {
	var diagnostics []entity.Diagnostic

	for _, v := range vars {
		if v.OS != "" && v.OS != runtime.GOOS {
			continue
		}

		currentVal, _ := e.GetEnvVar(v.Scope, v.Name)
		if currentVal != v.Value {
			if err := e.SetEnvVar(v.Scope, v.Name, v.Value); err != nil {
				diagnostics = append(diagnostics, entity.Diagnostic{
					Category: entity.DiagError,
					System:   "Environment",
					Target:   v.Name,
					Details:  fmt.Sprintf("Failed to set %s=%s: %v", v.Name, v.Value, err),
				})
			} else {
				diagnostics = append(diagnostics, entity.Diagnostic{
					Category: entity.DiagOK,
					System:   "Environment",
					Target:   v.Name,
					Details:  fmt.Sprintf("Configured %s=%s (Scope: %s)", v.Name, v.Value, v.Scope),
				})
			}
		} else {
			diagnostics = append(diagnostics, entity.Diagnostic{
				Category: entity.DiagOK,
				System:   "Environment",
				Target:   v.Name,
				Details:  fmt.Sprintf("Already set %s=%s (Scope: %s)", v.Name, v.Value, v.Scope),
			})
		}
	}

	return diagnostics, nil
}
