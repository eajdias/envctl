package environment

import (
	"context"
	"fmt"
	"os"
	"os/exec"
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

func (e *envManager) GetEnvVar(scope, name string) (string, error) {
	if runtime.GOOS != "windows" {
		return os.Getenv(name), nil
	}
	psCmd := fmt.Sprintf("[System.Environment]::GetEnvironmentVariable('%s', '%s')", name, scope)
	cmd := exec.Command("powershell.exe", "-NoProfile", "-Command", psCmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func (e *envManager) SetEnvVar(scope, name, value string) error {
	if runtime.GOOS != "windows" {
		return os.Setenv(name, value)
	}
	psCmd := fmt.Sprintf("[System.Environment]::SetEnvironmentVariable('%s', '%s', '%s')", name, value, scope)
	cmd := exec.Command("powershell.exe", "-NoProfile", "-Command", psCmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to set env var %s=%s (%s): %s (%w)", name, value, scope, string(out), err)
	}
	return nil
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

