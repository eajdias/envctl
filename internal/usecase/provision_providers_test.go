package usecase

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestFirstVersionToken(t *testing.T) {
	cases := []struct {
		output string
		want   string
	}{
		{"1.55.1", "1.55.1"},
		{"opencode v2.0.5", "2.0.5"},
		{"volta 2.0.2", "2.0.2"},
		{"CommandCode CLI v1.56.0 (linux)", "1.56.0"},
		{"v0.0.55", "0.0.55"},
		{"no numbers here", ""},
		{"", ""},
	}
	for _, tc := range cases {
		if got := firstVersionToken(tc.output); got != tc.want {
			t.Errorf("firstVersionToken(%q) = %q, want %q", tc.output, got, tc.want)
		}
	}
}

func TestVersionsDiffer(t *testing.T) {
	cases := []struct {
		installed string
		latest    string
		want      bool
	}{
		{"1.55.1", "1.56.0", true},
		{"1.55.1", "v1.55.1", false},
		{"2.0.5", "2.0.5", false},
		{" 1.0.0 ", "1.0.0", false},
	}
	for _, tc := range cases {
		if got := versionsDiffer(tc.installed, tc.latest); got != tc.want {
			t.Errorf("versionsDiffer(%q, %q) = %v, want %v", tc.installed, tc.latest, got, tc.want)
		}
	}
}

func TestClassifyInstallSource(t *testing.T) {
	home := "/home/user"
	cases := []struct {
		path string
		home string
		want string
	}{
		{filepath.Join(home, ".volta", "bin", "cmdc"), home, sourceVolta},
		{filepath.Join(home, ".local", "bin", "opencode"), home, sourceEnvctl},
		{"/usr/bin/opencode", home, sourceSystem},
		{"/usr/bin/opencode", "", sourceSystem},
	}
	for _, tc := range cases {
		if got := classifyInstallSource(tc.path, tc.home); got != tc.want {
			t.Errorf("classifyInstallSource(%q, %q) = %q, want %q", tc.path, tc.home, got, tc.want)
		}
	}
}

// The update path is chosen by comparing this token inside a switch; a display
// string that differs only by case silently sent Volta tools down the "leave it
// alone" branch.
func TestSourceLabelIsDisplayOnly(t *testing.T) {
	home := "/home/user"
	voltaBin := filepath.Join(home, ".volta", "bin", "cmdc")
	if got := classifyInstallSource(voltaBin, home); got != sourceVolta {
		t.Fatalf("Volta path classified as %q, want %q", got, sourceVolta)
	}
	if label := sourceLabel(sourceVolta); label != "Volta" {
		t.Errorf("sourceLabel(%q) = %q, want %q", sourceVolta, label, "Volta")
	}
	if label := sourceLabel(sourceSystem); label != "system package" {
		t.Errorf("sourceLabel(%q) = %q, want %q", sourceSystem, label, "system package")
	}
	for _, source := range []string{sourceVolta, sourceEnvctl, sourceSystem} {
		if label := sourceLabel(source); label == "" {
			t.Errorf("sourceLabel(%q) is empty", source)
		}
	}
}

// Phase 0 probes must resolve on the toolchain PATH, not the process PATH:
// under ssh/systemd/agent non-login shells ~/.volta/bin is absent from the
// process PATH, and the old exec.LookPath probe reported Volta tools as
// missing (reinstalling on every run).
func TestInstalledVersionResolvesVoltaShim(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("volta shim layout under $HOME/.volta is POSIX-only")
	}
	tmp := t.TempDir()
	voltaBin := filepath.Join(tmp, ".volta", "bin")
	if err := os.MkdirAll(voltaBin, 0755); err != nil {
		t.Fatalf("MkdirAll volta bin failed: %v", err)
	}
	shim := filepath.Join(voltaBin, "fakecli")
	if err := os.WriteFile(shim, []byte("#!/bin/sh\necho 'fakecli v9.9.9'\n"), 0755); err != nil {
		t.Fatalf("WriteFile shim failed: %v", err)
	}
	t.Setenv("HOME", tmp)
	t.Setenv("PATH", "/usr/bin:/bin")

	uc := &ProvisionProvidersUseCase{}
	if got := uc.installedVersion(context.Background(), "fakecli"); got != "9.9.9" {
		t.Errorf("installedVersion(fakecli) = %q, want %q (shim invisible on process PATH, visible on toolchain PATH)", got, "9.9.9")
	}
	if got := installSource("fakecli"); got != sourceVolta {
		t.Errorf("installSource(fakecli) = %q, want %q", got, sourceVolta)
	}
}

// Phase 0 must know how to reach every provider on every platform it claims to
// support: a missing install path would leave a fresh machine stuck.
func TestProviderCLIsAreInstallable(t *testing.T) {
	onWindows := runtime.GOOS == "windows"
	for _, tool := range providerCLIs() {
		if tool.name == "" || tool.binary == "" {
			t.Errorf("provider entry is incomplete: %+v", tool)
		}
		if tool.voltaPkg == "" && tool.wingetID == "" && tool.installer == "" {
			t.Errorf("%s has no install path at all", tool.name)
		}
		if onWindows && tool.voltaPkg == "" && tool.wingetID == "" {
			t.Errorf("%s has no Windows install path", tool.name)
		}
		if !onWindows && tool.voltaPkg == "" && tool.installer == "" {
			t.Errorf("%s has no Linux install path", tool.name)
		}
	}
}
