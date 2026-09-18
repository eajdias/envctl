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
		// The rc files are the source of truth for vars persisted by envctl:
		// they keep the portable $HOME form, while the process environment
		// may hold the shell-expanded copy inherited from the login shell.
		if val, _ := e.getEnvVarFromRC(name); val != "" {
			return val, nil
		}
		if val := os.Getenv(name); val != "" {
			return val, nil
		}
		return "", nil
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
		if err := e.persistEnvVar(name, value); err != nil {
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

// shellRC is one shell startup file plus the syntax used to persist variables
// and PATH entries in it.
type shellRC struct {
	path string
	fish bool
}

// rcFiles lists the startup files envctl keeps aligned. Fish is a first-class
// target: on hosts where fish is the login shell (CachyOS and many Arch
// installs) ~/.profile and ~/.bashrc are never read, so variables written only
// there would silently never reach an interactive session.
func (e *envManager) rcFiles() []shellRC {
	home := os.Getenv("HOME")
	if home == "" {
		if h, err := os.UserHomeDir(); err == nil {
			home = h
		}
	}
	if home == "" {
		return nil
	}
	return []shellRC{
		{path: filepath.Join(home, ".profile")},
		{path: filepath.Join(home, ".bashrc")},
		{path: filepath.Join(home, ".config", "fish", "config.fish"), fish: true},
	}
}

// exportLine renders the declaration for this shell's syntax.
func (rc shellRC) exportLine(name, value string) string {
	if rc.fish {
		return fmt.Sprintf("set -gx %s %q", name, value)
	}
	return fmt.Sprintf("export %s=%q", name, value)
}

// declarationPrefix is what precedes the value in an already-present line.
func (rc shellRC) declarationPrefix(name string) string {
	if rc.fish {
		return "set -gx " + name + " "
	}
	return "export " + name + "="
}

// persistEnvVar writes the export into every supported shell startup file,
// replacing any existing declaration so the variable survives shell restarts.
func (e *envManager) persistEnvVar(name, value string) error {
	for _, rc := range e.rcFiles() {
		if rc.path == "" {
			continue
		}
		data, err := os.ReadFile(rc.path)
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to read %s: %w", rc.path, err)
		}
		existing := strings.TrimSpace(string(data))
		var lines []string
		if existing != "" {
			lines = strings.Split(existing, "\n")
		}
		prefix := rc.declarationPrefix(name)
		exportLine := rc.exportLine(name, value)
		var out []string
		replaced := false
		for _, line := range lines {
			if strings.HasPrefix(line, prefix) {
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
		if err := os.MkdirAll(filepath.Dir(rc.path), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(rc.path, []byte(content), 0600); err != nil {
			return fmt.Errorf("failed to write %s: %w", rc.path, err)
		}
	}
	return nil
}

// getEnvVarFromRC reads the current value of a variable from the shell rc files.
func (e *envManager) getEnvVarFromRC(name string) (string, error) {
	for _, rc := range e.rcFiles() {
		if rc.path == "" {
			continue
		}
		data, err := os.ReadFile(rc.path)
		if err != nil {
			continue
		}
		prefix := rc.declarationPrefix(name)
		for _, line := range strings.Split(string(data), "\n") {
			trimmed := strings.TrimSpace(line)
			if !strings.HasPrefix(trimmed, prefix) {
				continue
			}
			val := strings.Trim(strings.TrimPrefix(trimmed, prefix), "\"'")
			if val != "" {
				return val, nil
			}
		}
	}
	return "", nil
}

// declarationsAligned reports whether every supported shell startup file
// already declares name with exactly the given value. A value present in a
// single rc file is not enough: a host whose login shell is fish would keep
// working with bash while the interactive shell never sees the variable.
func (e *envManager) declarationsAligned(name, value string) bool {
	files := e.rcFiles()
	if len(files) == 0 {
		return false
	}
	for _, rc := range files {
		data, err := os.ReadFile(rc.path)
		if err != nil {
			return false
		}
		if !strings.Contains(string(data), rc.exportLine(name, value)) {
			return false
		}
	}
	return true
}

func (e *envManager) EnsureEnvVars(ctx context.Context, vars []entity.EnvironmentVar) ([]entity.Diagnostic, error) {
	var diagnostics []entity.Diagnostic

	for _, v := range vars {
		if !entity.MatchesOS(v.OS) {
			continue
		}

		currentVal, _ := e.GetEnvVar(v.Scope, v.Name)
		aligned := currentVal == v.Value
		if aligned && runtime.GOOS != "windows" {
			aligned = e.declarationsAligned(v.Name, v.Value)
		}
		if !aligned {
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

// EnsurePathEntry guarantees that dir is present in the user PATH,
// prepending it when missing. Windows: User-scope Path registry value.
// POSIX: `export PATH="<dir>:$PATH"` in ~/.profile and ~/.bashrc; fish:
// `set -gx PATH "<dir>" $PATH` in ~/.config/fish/config.fish.
// Returns changed=true when the PATH was modified.
func (e *envManager) EnsurePathEntry(ctx context.Context, dir string) (bool, error) {
	if dir == "" {
		return false, fmt.Errorf("empty dir provided")
	}
	if runtime.GOOS == "windows" {
		current, err := e.GetEnvVar("User", "Path")
		if err != nil {
			return false, err
		}
		for _, p := range strings.Split(current, ";") {
			if strings.EqualFold(strings.TrimSpace(p), dir) {
				return false, nil
			}
		}
		updated := dir + ";" + current
		if strings.Trim(current, "; ") == "" {
			updated = dir
		}
		if err := e.SetEnvVar("User", "Path", updated); err != nil {
			return false, err
		}
		return true, nil
	}
	changed := false
	for _, rc := range e.rcFiles() {
		if rc.path == "" {
			continue
		}
		line := fmt.Sprintf(`export PATH="%s:$PATH"`, dir)
		if rc.fish {
			line = fmt.Sprintf("set -gx PATH %q $PATH", dir)
		}
		data, err := os.ReadFile(rc.path)
		if err != nil && !os.IsNotExist(err) {
			return changed, fmt.Errorf("failed to read %s: %w", rc.path, err)
		}
		if strings.Contains(string(data), dir) {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(rc.path), 0755); err != nil {
			return changed, err
		}
		f, err := os.OpenFile(rc.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			return changed, fmt.Errorf("failed to open %s: %w", rc.path, err)
		}
		if _, err := f.WriteString(line + "\n"); err != nil {
			f.Close()
			return changed, fmt.Errorf("failed to write %s: %w", rc.path, err)
		}
		f.Close()
		changed = true
	}
	return changed, nil
}
