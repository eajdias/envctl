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

// withFakeFishOnPATH installs a fake fish binary so the test exercises the
// fish branch deterministically instead of depending on the ambient machine
// (a host without fish must skip conjuring a fish config; a host with fish
// manages it). The binary name is per-OS: exec.LookPath only resolves
// extensionless names on POSIX and needs .exe on Windows.
func withFakeFishOnPATH(t *testing.T) {
	t.Helper()
	binDir := t.TempDir()
	name := "fish"
	if runtime.GOOS == "windows" {
		name = "fish.exe"
	}
	if err := os.WriteFile(filepath.Join(binDir, name), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func TestPersistEnvVarTargetsFishWithFishSyntax(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	withFakeFishOnPATH(t)
	manager := &WindowsEnvManager{}

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

	manager := &WindowsEnvManager{}
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

	manager := &WindowsEnvManager{}
	got := manager.getEnvVarFromRC("ENVCTL_TEMP")
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
	withFakeFishOnPATH(t)
	// State left by an older run: the variable only reached the bash files.
	if err := os.WriteFile(filepath.Join(home, ".bashrc"), []byte("export ENVCTL_TEMP=\"/temp\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	manager := &WindowsEnvManager{}
	vars := []entity.EnvironmentVar{{Name: "ENVCTL_TEMP", Value: "/temp", Scope: "User", OS: "linux"}}
	if _, err := manager.EnsureEnvVars(context.Background(), vars); err != nil {
		t.Fatalf("EnsureEnvVars: %v", err)
	}

	if !strings.Contains(readFileOrFail(t, fishConfigPath(home)), `set -gx ENVCTL_TEMP "/temp"`) {
		t.Errorf("fish config must receive the variable even when bash already declared it")
	}
}

func TestIsStaleToolShimReference(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	line := `. "$HOME/.local/bin/env"`
	if !isStaleToolShimReference(line) {
		t.Errorf("a sourcing line for a missing shim must be dropped")
	}

	// Once the shim exists the line is legitimate again.
	binDir := filepath.Join(home, ".local", "bin")
	if err := os.MkdirAll(binDir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(binDir, "env"), []byte("#!/bin/sh\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if isStaleToolShimReference(line) {
		t.Errorf("an existing shim must keep its sourcing line")
	}

	if err := os.Remove(filepath.Join(binDir, "env")); err != nil {
		t.Fatal(err)
	}
	for _, keep := range []string{
		`export PATH="$HOME/.local/bin:$PATH"`,
		`source ~/.bashrc`,
		`# . "$HOME/.local/bin/env" (disabled)`,
		`fzf --version`,
	} {
		if isStaleToolShimReference(keep) {
			t.Errorf("unrelated line %q must be preserved", keep)
		}
	}
}

func TestEnsurePathEntrySkipsFishWithoutFish(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("PATH persistence on Windows writes the registry value, not fish rc files")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	// A PATH with no fish in it: the legacy shell installers only touched
	// fish when `command -v fish` hit, so no config.fish may be conjured.
	t.Setenv("PATH", t.TempDir())
	manager := &WindowsEnvManager{}

	if _, err := manager.EnsurePathEntry(context.Background(), filepath.Join(home, ".local", "bin")); err != nil {
		t.Fatalf("EnsurePathEntry: %v", err)
	}
	if _, err := os.Stat(fishConfigPath(home)); !os.IsNotExist(err) {
		t.Errorf("fish config must not be created on a host without fish")
	}
}

// writeRC writes a shell startup file under home, creating parent directories.
func writeRC(t *testing.T, home, rel, content string) string {
	t.Helper()
	path := filepath.Join(home, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// withoutFishOnPATH makes fish unavailable so EnsurePathEntry only touches the
// POSIX files: a fake fish would create config.fish and report a change that
// has nothing to do with the assertion under test.
func withoutFishOnPATH(t *testing.T) {
	t.Helper()
	t.Setenv("PATH", t.TempDir())
}

// TestEnsurePathEntryRecognizesPortableForms locks the detection contract: a
// PATH line already declaring the dir in any portable spelling is a hit, so no
// second line is appended. The absolute spelling used to be the only one
// recognized, and the `$HOME` one (which every installer writes) produced a
// duplicate entry on the live PATH.
func TestEnsurePathEntryRecognizesPortableForms(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("PATH persistence on Windows writes the registry value, not shell rc files")
	}
	tests := []struct {
		name string
		line func(home string) string
	}{
		{"home", func(string) string { return `export PATH="$HOME/.local/bin:$PATH"` }},
		{"braced home", func(string) string { return `export PATH="${HOME}/.local/bin:$PATH"` }},
		{"tilde", func(string) string { return `export PATH=~/.local/bin:$PATH` }},
		{"absolute", func(home string) string { return `export PATH="` + home + `/.local/bin:$PATH"` }},
		{"appended after others", func(home string) string {
			return `export PATH="/usr/bin:` + home + `/.local/bin:$PATH"`
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			withoutFishOnPATH(t)
			bashrc := writeRC(t, home, ".bashrc", tt.line(home)+"\n")

			if _, err := (&WindowsEnvManager{}).EnsurePathEntry(context.Background(), filepath.Join(home, ".local", "bin")); err != nil {
				t.Fatalf("EnsurePathEntry: %v", err)
			}

			if got := readFileOrFail(t, bashrc); strings.Count(got, "export PATH=") != 1 {
				t.Errorf("an existing declaration must not be duplicated:\n%s", got)
			}
		})
	}
}

// TestEnsurePathEntryIgnoresNonPathContext pins the other half of the
// contract: only a PATH declaration counts. A mention inside a comment or in a
// different variable used to satisfy the old substring guard, so envctl
// reported "already present" while the shell never received the entry.
func TestEnsurePathEntryIgnoresNonPathContext(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("PATH persistence on Windows writes the registry value, not shell rc files")
	}
	tests := []struct {
		name    string
		content func(home, dir string) string
	}{
		{"comment", func(_, dir string) string { return "# " + dir + " is managed elsewhere" }},
		{"different variable", func(_, dir string) string {
			return `export NODE_PATH="` + dir + `:$HOME/node_modules"`
		}},
		{"sibling with the same prefix", func(_, dir string) string {
			return `export PATH="` + dir + `-tools:$PATH"`
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			withoutFishOnPATH(t)
			dir := filepath.Join(home, ".local", "bin")
			bashrc := writeRC(t, home, ".bashrc", tt.content(home, dir)+"\n")

			if _, err := (&WindowsEnvManager{}).EnsurePathEntry(context.Background(), dir); err != nil {
				t.Fatalf("EnsurePathEntry: %v", err)
			}

			want := `export PATH="` + dir + `:$PATH"`
			if got := readFileOrFail(t, bashrc); !strings.Contains(got, want) {
				t.Errorf("missing %q after a non-PATH mention:\n%s", want, got)
			}
		})
	}
}

// TestEnsurePathEntryRecognizesFishAddPath covers the idiomatic fish spelling:
// fish_add_path is how a user declares PATH entries by hand, so appending
// `set -gx PATH` next to it would add the dir twice in every fish session.
func TestEnsurePathEntryRecognizesFishAddPath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("PATH persistence on Windows writes the registry value, not shell rc files")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	withFakeFishOnPATH(t)
	fish := writeRC(t, home, ".config/fish/config.fish", "fish_add_path ~/.local/bin\n")

	if _, err := (&WindowsEnvManager{}).EnsurePathEntry(context.Background(), filepath.Join(home, ".local", "bin")); err != nil {
		t.Fatalf("EnsurePathEntry: %v", err)
	}

	if got := readFileOrFail(t, fish); strings.Contains(got, "set -gx PATH") {
		t.Errorf("fish_add_path already declares the dir; no extra PATH line expected:\n%s", got)
	}
}

// TestPathEntryPersistedAuditsEveryShell covers the doctor side: the audit used
// to look only at ~/.profile and ~/.bashrc with a substring match, so a fish
// host (the login shell on CachyOS) was reported as missing a persisted entry
// that was in fact declared in config.fish.
func TestPathEntryPersistedAuditsEveryShell(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("PATH persistence on Windows writes the registry value, not shell rc files")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir := filepath.Join(home, ".local", "bin")

	writeRC(t, home, ".config/fish/config.fish", `set -gx PATH "`+dir+`" $PATH`+"\n")
	if !(&WindowsEnvManager{}).PathEntryPersisted(dir) {
		t.Errorf("a fish declaration must satisfy the audit")
	}

	writeRC(t, home, ".config/fish/config.fish", "# "+dir+" is managed elsewhere\n")
	if (&WindowsEnvManager{}).PathEntryPersisted(dir) {
		t.Errorf("a comment must not satisfy the audit")
	}
}

func TestEnsurePathEntryAddsFishPathOnce(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("PATH persistence on Windows writes the registry value, not fish rc files")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	withFakeFishOnPATH(t)
	manager := &WindowsEnvManager{}
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
