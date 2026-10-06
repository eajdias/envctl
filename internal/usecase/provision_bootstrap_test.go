package usecase

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/eajdias/envctl/internal/infra/environment"
	"github.com/eajdias/envctl/internal/infra/executil"
	"github.com/eajdias/envctl/internal/infra/filesystem"
)

func TestPathStepPersistsEachDirOnce(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX profile persistence is Linux-only")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	for _, name := range []string{".bashrc", ".profile"} {
		if err := os.WriteFile(filepath.Join(home, name), nil, 0600); err != nil {
			t.Fatalf("WriteFile(%s): %v", name, err)
		}
	}

	uc := NewProvisionBootstrapUseCase(filesystem.NewFileSystemManager(), nil, environment.NewWindowsEnvManager(), nil, &mockLogger{})
	first := &BootstrapResult{}
	uc.pathStep(context.Background(), first, "Persist Go PATH in shell profiles", "/usr/local/go/bin", "$HOME/go/bin")
	if len(first.Diagnostics) != 1 || first.Diagnostics[0].Details != "Written to the shell profiles" {
		t.Fatalf("first run = %+v, want one Written diagnostic", first.Diagnostics)
	}

	second := &BootstrapResult{}
	uc.pathStep(context.Background(), second, "Persist Go PATH in shell profiles", "/usr/local/go/bin", "$HOME/go/bin")
	if len(second.Diagnostics) != 1 || second.Diagnostics[0].Details != "Already present in the shell profiles" {
		t.Fatalf("second run = %+v, want one Already-present diagnostic", second.Diagnostics)
	}

	for _, name := range []string{".bashrc", ".profile"} {
		data, err := os.ReadFile(filepath.Join(home, name))
		if err != nil {
			t.Fatalf("ReadFile(%s): %v", name, err)
		}
		if got := strings.Count(string(data), "/usr/local/go/bin"); got != 1 {
			t.Errorf("%s contains %d Go PATH entries, want 1", name, got)
		}
	}

	// Lines written by the legacy shell installers (same $HOME form) count as
	// present: the step must not duplicate them.
	legacyHome := t.TempDir()
	t.Setenv("HOME", legacyHome)
	for _, name := range []string{".bashrc", ".profile"} {
		line := "export PATH=\"$HOME/.opencode/bin:$PATH\"\n"
		if err := os.WriteFile(filepath.Join(legacyHome, name), []byte(line), 0600); err != nil {
			t.Fatalf("WriteFile(%s): %v", name, err)
		}
	}
	legacy := &BootstrapResult{}
	uc.pathStep(context.Background(), legacy, "OpenCode CLI PATH", "$HOME/.opencode/bin")
	if len(legacy.Diagnostics) != 1 || legacy.Diagnostics[0].Details != "Already present in the shell profiles" {
		t.Fatalf("legacy run = %+v, want Already-present without duplicating the line", legacy.Diagnostics)
	}
	for _, name := range []string{".bashrc", ".profile"} {
		data, err := os.ReadFile(filepath.Join(legacyHome, name))
		if err != nil {
			t.Fatalf("ReadFile(%s): %v", name, err)
		}
		if got := strings.Count(string(data), ".opencode/bin"); got != 1 {
			t.Errorf("%s contains %d OpenCode PATH entries, want 1", name, got)
		}
	}
}

func TestLinuxToolchainEnvIncludesOpenCodePath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Linux toolchain PATH is POSIX-only")
	}
	home := t.TempDir()
	want := filepath.Join(home, ".opencode", "bin")
	for _, entry := range executil.ToolchainEnv(home) {
		if !strings.HasPrefix(entry, "PATH=") {
			continue
		}
		path := strings.TrimPrefix(entry, "PATH=")
		if !strings.HasPrefix(path, want+string(filepath.ListSeparator)) {
			t.Fatalf("toolchain PATH = %q, want %q first", path, want)
		}
		return
	}
	t.Fatal("toolchain environment has no PATH entry")
}

func TestFzfHasWalker(t *testing.T) {
	cases := []struct {
		version string
		want    bool
	}{
		// Ubuntu 24.04 ships 0.44.1: no walker, so fzf still falls back to find.
		{"0.44.1", false},
		{"0.46.1", false},
		// 0.47 replaced the `find` fallback with the built-in walker.
		{"0.47.0", true},
		{"0.74.4", true},
		{"1.0.0", true},
		{"0.47", true},
		{"", false},
		{"not-a-version", false},
		{"0", false},
		// `fzf --version` prints "0.74.4 (sha)"; only major/minor are read, so
		// the suffix is harmless.
		{"0.74.4 (a140afeb)", true},
	}
	for _, tc := range cases {
		if got := fzfHasWalker(tc.version); got != tc.want {
			t.Errorf("fzfHasWalker(%q) = %v, want %v", tc.version, got, tc.want)
		}
	}
}

// TestGoPathConfigStepReportsWorkOnlyOnce is covered by
// TestPathStepPersistsEachDirOnce above: pathStep reports the write honestly
// from EnsurePathEntry's changed signal instead of from shell check scripts.
