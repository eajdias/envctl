package usecase

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestOpenCodePathInstallerPersistsPOSIXProfiles(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX profile persistence is Linux-only")
	}
	home := t.TempDir()
	for _, name := range []string{".bashrc", ".profile"} {
		if err := os.WriteFile(filepath.Join(home, name), nil, 0600); err != nil {
			t.Fatalf("WriteFile(%s): %v", name, err)
		}
	}
	env := make([]string, 0, len(os.Environ())+1)
	for _, entry := range os.Environ() {
		if !strings.HasPrefix(entry, "HOME=") {
			env = append(env, entry)
		}
	}
	env = append(env, "HOME="+home)
	for i := 0; i < 2; i++ {
		cmd := exec.Command("bash", "-c", openCodePathInstaller)
		cmd.Env = env
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("path installer run %d: %v (%s)", i+1, err, out)
		}
	}
	for _, name := range []string{".bashrc", ".profile"} {
		data, err := os.ReadFile(filepath.Join(home, name))
		if err != nil {
			t.Fatalf("ReadFile(%s): %v", name, err)
		}
		if got := strings.Count(string(data), "$HOME/.opencode/bin"); got != 1 {
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
	for _, entry := range linuxToolchainEnv(home) {
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
