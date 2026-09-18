package environment

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/eajdias/envctl/internal/domain/entity"
)

func readFileOrFail(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

func fishConfigPath(home string) string {
	return filepath.Join(home, ".config", "fish", "config.fish")
}

func TestPersistEnvVarTargetsFishWithFishSyntax(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	manager := &envManager{}

	if err := manager.persistEnvVar("ENVCTL_TEMP", "/temp"); err != nil {
		t.Fatalf("persistEnvVar: %v", err)
	}

	if got := readFileOrFail(t, filepath.Join(home, ".bashrc")); !strings.Contains(got, `export ENVCTL_TEMP="/temp"`) {
		t.Errorf("~/.bashrc missing POSIX export:\n%s", got)
	}
	if got := readFileOrFail(t, filepath.Join(home, ".profile")); !strings.Contains(got, `export ENVCTL_TEMP="/temp"`) {
		t.Errorf("~/.profile missing POSIX export:\n%s", got)
	}

	fish := readFileOrFail(t, fishConfigPath(home))
	if !strings.Contains(fish, `set -gx ENVCTL_TEMP "/temp"`) {
		t.Errorf("fish config missing fish syntax:\n%s", fish)
	}
	if strings.Contains(fish, "export ") {
		t.Errorf("fish config must not contain POSIX export syntax:\n%s", fish)
	}
}

func TestPersistEnvVarReplacesExistingDeclaration(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	fishPath := fishConfigPath(home)
	if err := os.MkdirAll(filepath.Dir(fishPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fishPath, []byte("set -gx EDITOR vim\nset -gx ENVCTL_TEMP \"/old\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	manager := &envManager{}
	if err := manager.persistEnvVar("ENVCTL_TEMP", "/temp"); err != nil {
		t.Fatalf("persistEnvVar: %v", err)
	}

	got := readFileOrFail(t, fishPath)
	if strings.Count(got, "ENVCTL_TEMP") != 1 {
		t.Errorf("expected a single declaration after replace, got:\n%s", got)
	}
	if !strings.Contains(got, `set -gx ENVCTL_TEMP "/temp"`) {
		t.Errorf("expected the value to be updated:\n%s", got)
	}
	if !strings.Contains(got, "set -gx EDITOR vim") {
		t.Errorf("unrelated lines must be preserved:\n%s", got)
	}
}

func TestGetEnvVarFromRCReadsFishDeclarations(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	fishPath := fishConfigPath(home)
	if err := os.MkdirAll(filepath.Dir(fishPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fishPath, []byte("set -gx ENVCTL_TEMP \"/temp\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	manager := &envManager{}
	got, err := manager.getEnvVarFromRC("ENVCTL_TEMP")
	if err != nil {
		t.Fatalf("getEnvVarFromRC: %v", err)
	}
	if got != "/temp" {
		t.Errorf("getEnvVarFromRC = %q, want %q", got, "/temp")
	}
}

func TestEnsureEnvVarsAlignsEveryShell(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("POSIX rc alignment only applies on linux")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	// State left by an older run: the variable only reached the bash files.
	if err := os.WriteFile(filepath.Join(home, ".bashrc"), []byte("export ENVCTL_TEMP=\"/temp\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	manager := &envManager{}
	vars := []entity.EnvironmentVar{{Name: "ENVCTL_TEMP", Value: "/temp", Scope: "User", OS: "linux"}}
	if _, err := manager.EnsureEnvVars(context.Background(), vars); err != nil {
		t.Fatalf("EnsureEnvVars: %v", err)
	}

	if !strings.Contains(readFileOrFail(t, fishConfigPath(home)), `set -gx ENVCTL_TEMP "/temp"`) {
		t.Errorf("fish config must receive the variable even when bash already declared it")
	}
}

func TestEnsurePathEntryAddsFishPathOnce(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("PATH persistence on Windows writes the registry value, not fish rc files")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	manager := &envManager{}
	dir := filepath.Join(home, ".local", "bin")

	changed, err := manager.EnsurePathEntry(context.Background(), dir)
	if err != nil {
		t.Fatalf("EnsurePathEntry: %v", err)
	}
	if !changed {
		t.Errorf("expected the first call to report a change")
	}

	fish := readFileOrFail(t, fishConfigPath(home))
	if !strings.Contains(fish, `set -gx PATH "`+dir+`" $PATH`) {
		t.Errorf("fish config missing PATH line:\n%s", fish)
	}

	changedAgain, err := manager.EnsurePathEntry(context.Background(), dir)
	if err != nil {
		t.Fatalf("EnsurePathEntry (second call): %v", err)
	}
	if changedAgain {
		t.Errorf("second call must be a no-op")
	}
	if count := strings.Count(readFileOrFail(t, fishConfigPath(home)), dir); count != 1 {
		t.Errorf("PATH entry must not be duplicated, found %d occurrences", count)
	}
}
