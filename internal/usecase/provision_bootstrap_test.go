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

// TestGoPathConfigStepReportsWorkOnlyOnce locks the contract configStep relies
// on: goPathDoneCheck must fail before the write and succeed after it, and a
// second write must not duplicate the entry. Without this, the step reports
// "installed" on every run while the write is a no-op, and the run log — the
// evidence an idempotency review reads — stops being trustworthy.
func TestGoPathConfigStepReportsWorkOnlyOnce(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX profile persistence is Linux-only")
	}

	// Both scripts branch on `command -v fish`, so PATH is built explicitly here.
	// Inheriting the host PATH made the first assertion true on a machine with
	// fish and false on one without, which is how the first version of this test
	// passed locally and then failed on the runner: a check that exited 0 on a
	// profile with no Go PATH at all was reported as "already present".
	cases := []struct {
		name     string
		withFish bool
	}{
		{name: "fish installed", withFish: true},
		{name: "fish absent", withFish: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			for _, name := range []string{".bashrc", ".profile"} {
				if err := os.WriteFile(filepath.Join(home, name), nil, 0600); err != nil {
					t.Fatalf("WriteFile(%s): %v", name, err)
				}
			}

			// A bin dir holding only what the scripts shell out to, so
			// `command -v fish` answers from this dir and not from the host.
			bin := t.TempDir()
			for _, tool := range []string{"grep", "mkdir", "dirname"} {
				resolved, err := exec.LookPath(tool)
				if err != nil {
					t.Fatalf("LookPath(%s): %v", tool, err)
				}
				if err := os.Symlink(resolved, filepath.Join(bin, tool)); err != nil {
					t.Fatalf("Symlink(%s): %v", tool, err)
				}
			}
			if tc.withFish {
				// `command -v` only needs the file to be executable: neither
				// script runs fish, it only writes fish's config.
				if err := os.WriteFile(filepath.Join(bin, "fish"), []byte("#!/bin/sh\nexit 0\n"), 0700); err != nil {
					t.Fatalf("WriteFile(fish stub): %v", err)
				}
			}

			env := make([]string, 0, len(os.Environ())+2)
			for _, entry := range os.Environ() {
				if !strings.HasPrefix(entry, "HOME=") && !strings.HasPrefix(entry, "PATH=") {
					env = append(env, entry)
				}
			}
			env = append(env, "HOME="+home, "PATH="+bin)

			run := func(script string) error {
				// #nosec G204 -- the scripts are package-level constants, not input.
				cmd := exec.Command("bash", "-c", script)
				cmd.Env = env
				out, err := cmd.CombinedOutput()
				if err != nil {
					t.Logf("script output: %s", strings.TrimSpace(string(out)))
				}
				return err
			}
			read := func(name string) string {
				data, err := os.ReadFile(filepath.Join(home, name))
				if err != nil {
					t.Fatalf("ReadFile(%s): %v", name, err)
				}
				return string(data)
			}
			fishRC := ".config/fish/config.fish"

			// Before the write the check must report "not done" so the step runs.
			if err := run(goPathDoneCheck); err == nil {
				t.Error("goPathDoneCheck succeeded on a fresh profile, want failure so the step applies the write")
			}
			if err := run(goPathInstaller); err != nil {
				t.Fatalf("goPathInstaller: %v", err)
			}
			if err := run(goPathDoneCheck); err != nil {
				t.Errorf("goPathDoneCheck failed after the write, want success so the step reports already present: %v", err)
			}

			// Running the step again must neither fail nor change a byte: the
			// check reporting "already present" and the installer being a no-op
			// are the same fact seen from two sides.
			before := map[string]string{}
			for _, name := range []string{".bashrc", ".profile", fishRC} {
				if _, err := os.Stat(filepath.Join(home, name)); err == nil {
					before[name] = read(name)
				}
			}
			if err := run(goPathInstaller); err != nil {
				t.Fatalf("goPathInstaller rerun: %v", err)
			}
			for name, want := range before {
				if got := read(name); got != want {
					t.Errorf("rerun changed %s:\n--- first run ---\n%s\n--- rerun ---\n%s", name, want, got)
				}
			}

			for _, name := range []string{".bashrc", ".profile"} {
				if got := strings.Count(read(name), "/usr/local/go/bin"); got != 1 {
					t.Errorf("%s contains %d Go PATH entries, want 1", name, got)
				}
			}
			if tc.withFish {
				got := read(fishRC)
				if n := strings.Count(got, "/usr/local/go/bin"); n != 1 {
					t.Errorf("%s contains %d Go PATH entries, want 1", fishRC, n)
				}
				// The bash export is a syntax error in fish, so the write uses
				// fish's own syntax; see the comment on goPathInstaller.
				if !strings.Contains(got, "set -gx PATH") || strings.Contains(got, "export PATH=") {
					t.Errorf("%s is not fish syntax: %q", fishRC, got)
				}

				// A partial state — the POSIX profiles done, the fish config
				// gone — is still work to do. Reporting "already present" there
				// would leave the config.fish entry missing for good, since the
				// installer only ever runs when the check says there is work.
				if err := os.Remove(filepath.Join(home, fishRC)); err != nil {
					t.Fatalf("Remove(%s): %v", fishRC, err)
				}
				if err := run(goPathDoneCheck); err == nil {
					t.Errorf("goPathDoneCheck succeeded with %s missing, want failure so the step rewrites it", fishRC)
				}
				if err := run(goPathInstaller); err != nil {
					t.Fatalf("goPathInstaller after partial state: %v", err)
				}
				if got := read(fishRC); !strings.Contains(got, "set -gx PATH") {
					t.Errorf("%s not restored after the partial state: %q", fishRC, got)
				}
			} else if _, err := os.Stat(filepath.Join(home, fishRC)); err == nil {
				t.Errorf("%s written on a host without fish, want no fish config", fishRC)
			}
		})
	}
}
