package performance

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eajdias/envctl/internal/domain/entity"
)

func keysOf(assignments []entity.SysctlAssignment) map[string]entity.SysctlAssignment {
	out := make(map[string]entity.SysctlAssignment, len(assignments))
	for _, assignment := range assignments {
		out[assignment.Key] = assignment
	}
	return out
}

// TestSysctlResolutionFollowsTheDocumentedPrecedence encodes sysctl.d(5):
// "All configuration files are sorted by their filename in lexicographic order,
// regardless of which of the directories they reside in", and a file in /etc
// overrides a file with the same name in /run, /usr/local/lib and /usr/lib.
func TestSysctlResolutionFollowsTheDocumentedPrecedence(t *testing.T) {
	root := t.TempDir()
	etc := filepath.Join(root, "etc")
	run := filepath.Join(root, "run")
	usr := filepath.Join(root, "usr")
	dirs := []string{etc, run, usr}

	// 99 beats 90 even though 90 lives in the higher-precedence /etc.
	// This is the exact shape found on the fleet: the profile writes
	// 90-envctl-performance.conf and the cloud agent ships 99-swappiness.conf.
	write(t, filepath.Join(etc, "90-envctl-performance.conf"), "vm.swappiness = 150\n")
	write(t, filepath.Join(etc, "99-swappiness.conf"), "vm.swappiness=10\n")

	got := keysOf(resolveSysctlAssignments(dirs, "/etc/sysctl.d/90-envctl-performance.conf"))
	swappiness, ok := got["vm.swappiness"]
	if !ok {
		t.Fatal("vm.swappiness was not resolved")
	}
	if swappiness.Boot != "10" {
		t.Fatalf("resolved value %q, want the 99- file to win with 10", swappiness.Boot)
	}
	if !strings.HasSuffix(swappiness.File, "99-swappiness.conf") {
		t.Fatalf("resolved from %q, want the 99- file", swappiness.File)
	}
	if swappiness.Managed {
		t.Fatal("a host file must never be reported as managed by envctl")
	}
}

// TestSysctlResolutionPrefersTheSameNameInTheHigherDirectory covers the one case
// where the directory decides: identical basenames.
func TestSysctlResolutionPrefersTheSameNameInTheHigherDirectory(t *testing.T) {
	root := t.TempDir()
	etc := filepath.Join(root, "etc")
	usr := filepath.Join(root, "usr")
	dirs := []string{etc, usr}

	write(t, filepath.Join(etc, "50-tuning.conf"), "kernel.pid_max = 4194304\n")
	write(t, filepath.Join(usr, "50-tuning.conf"), "kernel.pid_max = 1024\n")

	got := keysOf(resolveSysctlAssignments(dirs, ""))
	pidMax, ok := got["kernel.pid_max"]
	if !ok {
		t.Fatal("kernel.pid_max was not resolved")
	}
	if pidMax.Boot != "4194304" {
		t.Fatalf("resolved %q, want /etc to override /usr/lib for the same name", pidMax.Boot)
	}
	if pidMax.File != filepath.Join(etc, "50-tuning.conf") {
		t.Fatalf("resolved from %q, want the /etc copy", pidMax.File)
	}
}

func TestSysctlResolutionMarksTheProfileDropIn(t *testing.T) {
	root := t.TempDir()
	dirs := []string{root}
	managed := filepath.Join(root, "90-envctl-performance.conf")
	write(t, managed, "net.core.somaxconn = 65535\n")

	got := keysOf(resolveSysctlAssignments(dirs, managed))
	if !got["net.core.somaxconn"].Managed {
		t.Fatal("the profile's own drop-in must be reported as managed")
	}
}

// TestSysctlResolutionNormalizesSlashes documents that sysctl.d(5) treats "a.b.c"
// and "a/b/c" as the same key, so a host using slashes still shadows the profile.
func TestSysctlResolutionNormalizesSlashes(t *testing.T) {
	root := t.TempDir()
	dirs := []string{root}
	write(t, filepath.Join(root, "90-envctl-performance.conf"), "vm.swappiness = 150\n")
	write(t, filepath.Join(root, "99-host.conf"), "vm/swappiness = 10\n")

	got := keysOf(resolveSysctlAssignments(dirs, filepath.Join(root, "90-envctl-performance.conf")))
	if got["vm.swappiness"].Boot != "10" {
		t.Fatalf("a slash-separated assignment did not shadow the profile: %+v", got["vm.swappiness"])
	}
}

func TestSysctlResolutionIgnoresNoise(t *testing.T) {
	root := t.TempDir()
	dirs := []string{root}
	write(t, filepath.Join(root, "10-noise.conf"), strings.Join([]string{
		"# a comment",
		"; another comment",
		"",
		"   ",
		"-net.ipv4.conf.all.forwarding",
		"no-equals-here",
		"vm.swappiness = 150   ",
	}, "\n"))
	// Only *.conf is read; the manual explicitly says so.
	write(t, filepath.Join(root, "50-ignored.txt"), "vm.swappiness = 99\n")

	got := keysOf(resolveSysctlAssignments(dirs, ""))
	if len(got) != 1 {
		t.Fatalf("expected exactly one assignment, got %+v", got)
	}
	if got["vm.swappiness"].Boot != "150" {
		t.Fatalf("value %q was not trimmed correctly", got["vm.swappiness"].Boot)
	}
}

func TestSysctlResolutionOnAMissingTreeIsEmpty(t *testing.T) {
	root := t.TempDir()
	got := resolveSysctlAssignments([]string{
		filepath.Join(root, "absent-etc"),
		filepath.Join(root, "absent-run"),
	}, "")
	if len(got) != 0 {
		t.Fatalf("a missing sysctl.d tree must resolve to nothing, got %+v", got)
	}
}

// TestSysctlResolutionOrdersResultsByKey keeps the diagnostics stable across
// runs: a map iteration would make the report order random.
func TestSysctlResolutionOrdersResultsByKey(t *testing.T) {
	root := t.TempDir()
	dirs := []string{root}
	write(t, filepath.Join(root, "90-envctl-performance.conf"),
		"vm.swappiness = 150\nnet.core.somaxconn = 65535\nvm.vfs_cache_pressure = 50\n")

	got := resolveSysctlAssignments(dirs, "")
	if len(got) != 3 {
		t.Fatalf("expected 3 assignments, got %d", len(got))
	}
	want := []string{"net.core.somaxconn", "vm.swappiness", "vm.vfs_cache_pressure"}
	for i, key := range want {
		if got[i].Key != key {
			t.Fatalf("position %d is %q, want %q (results must be key-ordered)", i, got[i].Key, key)
		}
	}
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
