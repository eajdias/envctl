package performance

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/eajdias/envctl/internal/domain/entity"
)

// fakeDropinFS emulates the argv commands the writer issues against a real
// directory, so the sequence and the resulting files are both observable.
func fakeDropinFS(t *testing.T, failOn map[string]error, calls *[][]string) commandRunner {
	t.Helper()
	return func(_ context.Context, name string, args ...string) ([]byte, error) {
		if calls != nil {
			*calls = append(*calls, append([]string{name}, args...))
		}
		if err, ok := failOn[name]; ok && err != nil {
			return nil, err
		}
		switch name {
		case "cp":
			content, err := os.ReadFile(args[1])
			if err != nil {
				return nil, err
			}
			return nil, os.WriteFile(args[2], content, 0o644)
		case "install":
			content, err := os.ReadFile(args[len(args)-2])
			if err != nil {
				return nil, err
			}
			return nil, os.WriteFile(args[len(args)-1], content, 0o644)
		case "mv":
			// mv is a rename, not a copy: the source must disappear. Emulating
			// it as a copy would hide exactly the leftover-temp bug these tests
			// exist to catch.
			return nil, os.Rename(args[len(args)-2], args[len(args)-1])
		case "rm":
			for _, path := range args {
				if strings.HasPrefix(path, "-") {
					continue
				}
				_ = os.Remove(path)
			}
			return nil, nil
		}
		return nil, nil
	}
}

func TestDropinWriterIdenticalContentIsNoop(t *testing.T) {
	destination := filepath.Join(t.TempDir(), "90-envctl.conf")
	if err := os.WriteFile(destination, []byte("payload\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var calls [][]string
	writer := newDropinWriter(destination, fakeDropinFS(t, nil, &calls), func() time.Time { return time.Unix(1, 0) }, false)

	changed, backup, err := writer.Install("payload\n", 0o644, false)
	if err != nil {
		t.Fatalf("no-op failed: %v", err)
	}
	if changed {
		t.Fatal("identical content reported a change")
	}
	if backup != "" {
		t.Fatalf("identical content created backup %q", backup)
	}
	if len(calls) != 0 {
		t.Fatalf("identical content invoked %#v", calls)
	}
	if backups, _ := filepath.Glob(destination + ".bak.*"); len(backups) != 0 {
		t.Fatalf("identical content left backups %#v", backups)
	}
}

func TestDropinWriterBacksUpReplacesAtomically(t *testing.T) {
	dir := t.TempDir()
	destination := filepath.Join(dir, "90-envctl.conf")
	if err := os.WriteFile(destination, []byte("old\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var calls [][]string
	writer := newDropinWriter(destination, fakeDropinFS(t, nil, &calls), func() time.Time { return time.Unix(1, 0) }, false)

	changed, backup, err := writer.Install("new\n", 0o644, false)
	if err != nil {
		t.Fatalf("install failed: %v", err)
	}
	if !changed {
		t.Fatal("differing content reported no change")
	}
	if backup == "" {
		t.Fatal("differing content did not report a backup")
	}
	if data, err := os.ReadFile(backup); err != nil || string(data) != "old\n" {
		t.Fatalf("backup content = %q err=%v, want the previous content", data, err)
	}
	if data, _ := os.ReadFile(destination); string(data) != "new\n" {
		t.Fatalf("destination content = %q, want the new content", data)
	}

	// The temp file must land in the destination directory so the rename is
	// atomic: a temp in /tmp would be a cross-device copy on some hosts.
	if len(calls) != 3 {
		t.Fatalf("commands = %#v, want cp, install, mv", calls)
	}
	tempPath := calls[1][len(calls[1])-1]
	if filepath.Dir(tempPath) != dir {
		t.Fatalf("temp file %q is not in the destination directory %q", tempPath, dir)
	}
	if leftover, _ := filepath.Glob(filepath.Join(dir, "*.tmp.*")); len(leftover) != 0 {
		t.Fatalf("temp file was not renamed away: %#v", leftover)
	}
}

func TestDropinWriterDisambiguatesSameSecondBackups(t *testing.T) {
	dir := t.TempDir()
	destination := filepath.Join(dir, "90-envctl.conf")
	writer := newDropinWriter(destination, fakeDropinFS(t, nil, nil), func() time.Time { return time.Unix(1, 0) }, false)

	if err := os.WriteFile(destination, []byte("v1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, first, err := writer.Install("v2\n", 0o644, false)
	if err != nil {
		t.Fatal(err)
	}
	_, second, err := writer.Install("v3\n", 0o644, false)
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatalf("two replacements in the same second reused backup %q", first)
	}
	if !strings.HasPrefix(second, destination+".bak.") || !strings.HasSuffix(second, "-1") {
		t.Fatalf("second backup = %q, want the -1 suffix", second)
	}
}

func TestDropinWriterFailedMovePreservesOriginal(t *testing.T) {
	dir := t.TempDir()
	destination := filepath.Join(dir, "90-envctl.conf")
	if err := os.WriteFile(destination, []byte("old\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var calls [][]string
	writer := newDropinWriter(destination, fakeDropinFS(t, map[string]error{"mv": errors.New("synthetic mv failure")}, &calls), func() time.Time { return time.Unix(1, 0) }, false)

	if _, _, err := writer.Install("new\n", 0o644, false); err == nil {
		t.Fatal("expected a failed move to be reported")
	}
	if data, _ := os.ReadFile(destination); string(data) != "old\n" {
		t.Fatalf("destination content = %q, want the original preserved", data)
	}
	// The abandoned temp must be cleaned, and the cleanup must be attempted.
	tempPath := calls[1][len(calls[1])-1]
	if _, err := os.Stat(tempPath); !os.IsNotExist(err) {
		t.Fatalf("abandoned temp %q was not cleaned up", tempPath)
	}
	sawCleanup := false
	for _, call := range calls {
		if call[0] == "rm" {
			sawCleanup = true
		}
	}
	if !sawCleanup {
		t.Fatalf("commands = %#v, want an rm cleanup after the failed move", calls)
	}
}

func TestDropinWriterDryRunWritesNothing(t *testing.T) {
	destination := filepath.Join(t.TempDir(), "90-envctl.conf")
	called := false
	writer := newDropinWriter(destination, func(context.Context, string, ...string) ([]byte, error) {
		called = true
		return nil, nil
	}, func() time.Time { return time.Unix(1, 0) }, false)

	changed, backup, err := writer.Install("new\n", 0o644, true)
	if err != nil {
		t.Fatalf("dry-run failed: %v", err)
	}
	if changed || backup != "" {
		t.Fatalf("dry-run reported changed=%v backup=%q, want neither", changed, backup)
	}
	if called {
		t.Fatal("dry-run invoked a command")
	}
	if _, err := os.Stat(destination); !os.IsNotExist(err) {
		t.Fatal("dry-run created the destination")
	}
}

func TestDropinWriterUsesNoninteractiveSudo(t *testing.T) {
	destination := filepath.Join(t.TempDir(), "90-envctl.conf")
	var calls [][]string
	writer := newDropinWriter(destination, fakeDropinFS(t, nil, &calls), func() time.Time { return time.Unix(1, 0) }, true)

	if _, _, err := writer.Install("new\n", 0o644, false); err != nil {
		t.Fatal(err)
	}
	for _, call := range calls {
		if call[0] != "sudo" || call[1] != "-n" {
			t.Fatalf("command %#v is not sudo -n argv", call)
		}
	}
}

// A min-policy key must never be lowered below what the host already has. The
// fleet case is fs.file-max: Oracle and AWS both ship it at the int64 ceiling,
// and the previous manifest wrote 2097152 over it.
func TestSysctlManagerMinPolicyNeverLowersTheHostValue(t *testing.T) {
	destination := filepath.Join(t.TempDir(), "90-envctl-performance.conf")
	var calls [][]string
	manager := newSysctlManager(
		destination,
		fakeDropinFS(t, nil, &calls),
		func() time.Time { return time.Unix(1, 0) },
		false,
		func(string) (string, bool) { return "9223372036854775807", true },
	)

	diags, err := manager.Apply(context.Background(), []entity.SysctlSetting{{
		Key: "fs.file-max", Value: "2097152", Policy: entity.SysctlPolicyMin,
	}}, false)
	if err != nil {
		t.Fatalf("min-policy apply failed: %v", err)
	}
	if len(diags) != 1 || diags[0].Category != entity.DiagOK {
		t.Fatalf("diagnostics = %#v, want one OK diagnostic", diags)
	}
	if !strings.Contains(diags[0].Details, "at or above") {
		t.Fatalf("diagnostic %q does not explain that the host value was kept", diags[0].Details)
	}
	if len(calls) != 0 {
		t.Fatalf("min-policy lowered a better host value: %#v", calls)
	}
}

func TestSysctlManagerMinPolicyRaisesALowerHostValue(t *testing.T) {
	destination := filepath.Join(t.TempDir(), "90-envctl-performance.conf")
	if err := os.WriteFile(destination, []byte("fs.file-max = 4096\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var calls [][]string
	manager := newSysctlManager(
		destination,
		fakeDropinFS(t, nil, &calls),
		func() time.Time { return time.Unix(1, 0) },
		false,
		func(string) (string, bool) { return "4096", true },
	)

	if _, err := manager.Apply(context.Background(), []entity.SysctlSetting{{
		Key: "fs.file-max", Value: "2097152", Policy: entity.SysctlPolicyMin,
	}}, false); err != nil {
		t.Fatalf("min-policy raise failed: %v", err)
	}
	if data, _ := os.ReadFile(destination); !strings.Contains(string(data), "2097152") {
		t.Fatalf("min-policy did not raise the value: %q", data)
	}
}

func TestSysctlManagerMinPolicyMaxPolicyAndSet(t *testing.T) {
	cases := []struct {
		name      string
		policy    entity.SysctlPolicy
		desired   string
		effective string
		wantWrite bool
	}{
		{"set always writes", entity.SysctlPolicySet, "2097152", "9223372036854775807", true},
		{"min raises a lower value", entity.SysctlPolicyMin, "2097152", "4096", true},
		{"min keeps a higher value", entity.SysctlPolicyMin, "2097152", "9223372036854775807", false},
		{"min keeps an equal value", entity.SysctlPolicyMin, "65535", "65535", false},
		{"max lowers a higher value", entity.SysctlPolicyMax, "4096", "9223372036854775807", true},
		{"max keeps a lower value", entity.SysctlPolicyMax, "4096", "1024", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			destination := filepath.Join(t.TempDir(), "90-envctl-performance.conf")
			original := "# Managed by envctl; review before editing.\nprevious.key = 1\n"
			if err := os.WriteFile(destination, []byte(original), 0o644); err != nil {
				t.Fatal(err)
			}
			var calls [][]string
			manager := newSysctlManager(
				destination,
				func(_ context.Context, name string, args ...string) ([]byte, error) {
					if name == "sysctl" {
						return nil, nil
					}
					return fakeDropinFS(t, nil, &calls)(context.Background(), name, args...)
				},
				func() time.Time { return time.Unix(1, 0) },
				false,
				func(string) (string, bool) { return tc.effective, true },
			)

			if _, err := manager.Apply(context.Background(), []entity.SysctlSetting{{
				Key: "fs.file-max", Value: tc.desired, Policy: tc.policy,
			}}, false); err != nil {
				t.Fatalf("apply failed: %v", err)
			}
			data, _ := os.ReadFile(destination)
			wrote := !strings.Contains(string(data), "previous.key")
			if wrote != tc.wantWrite {
				t.Fatalf("wrote=%v want=%v; content=%q calls=%#v", wrote, tc.wantWrite, data, calls)
			}
		})
	}
}

// An unreadable /proc/sys entry must be an error, never a silent success: the
// host may already hold a better value the tool cannot see.
func TestSysctlManagerMinPolicyUnreadableValueIsAnError(t *testing.T) {
	destination := filepath.Join(t.TempDir(), "90-envctl-performance.conf")
	manager := newSysctlManager(
		destination,
		fakeDropinFS(t, nil, nil),
		func() time.Time { return time.Unix(1, 0) },
		false,
		func(string) (string, bool) { return "", false },
	)

	diags, err := manager.Apply(context.Background(), []entity.SysctlSetting{{
		Key: "fs.file-max", Value: "2097152", Policy: entity.SysctlPolicyMin,
	}}, false)
	if err == nil {
		t.Fatal("expected an unreadable effective value to be reported")
	}
	if len(diags) != 1 || diags[0].Category != entity.DiagError || !strings.Contains(diags[0].Details, "fs.file-max") {
		t.Fatalf("diagnostics = %#v, want an error naming the key", diags)
	}
	if _, statErr := os.Stat(destination); !os.IsNotExist(statErr) {
		t.Fatal("a failed policy check must not create the drop-in")
	}
}

func TestSysctlProcPath(t *testing.T) {
	// /proc/sys is a POSIX path, so the expected form only holds where the
	// kernel interface exists. On Windows filepath.Join would produce
	// backslashes, and the performance profile is Linux-only anyway.
	if runtime.GOOS != "linux" {
		t.Skipf("sysctl procfs paths are Linux-only; this host is %s", runtime.GOOS)
	}
	if got := sysctlProcPath("vm.swappiness"); got != "/proc/sys/vm/swappiness" {
		t.Fatalf("sysctlProcPath = %q", got)
	}
	if got := sysctlProcPath("net.core.somaxconn"); got != "/proc/sys/net/core/somaxconn" {
		t.Fatalf("sysctlProcPath = %q", got)
	}
	if got := sysctlProcPath("fs.file-max"); got != "/proc/sys/fs/file-max" {
		t.Fatalf("sysctlProcPath = %q", got)
	}
}

func TestCompareSysctlValues(t *testing.T) {
	if compareSysctlValues("10", "9") <= 0 {
		t.Fatal("10 must compare greater than 9")
	}
	if compareSysctlValues("4096", "9223372036854775807") >= 0 {
		t.Fatal("4096 must compare lower than the int64 ceiling")
	}
	if compareSysctlValues("65535", "65535") != 0 {
		t.Fatal("equal values must compare equal")
	}
	if compareSysctlValues("", "1") == 0 {
		t.Fatal("an unparseable value must not silently compare equal")
	}
}

// The content scratch file must NOT be created next to the destination. This
// process is normally not root, so a direct create under /etc fails with EACCES
// and the whole write is refused. Found by running the profile on a real host:
// the sysctl drop-in failed with "open /etc/sysctl.d/.envctl-dropin-...:
// permission denied".
//
// The assertion is on the recorded argv rather than on a filesystem effect,
// because the privileged `install` and `mv` are what touch the destination
// directory and cannot be emulated by an unprivileged test process.
func TestDropinWriterKeepsTheScratchFileOutOfTheDestinationDirectory(t *testing.T) {
	destination := filepath.Join(t.TempDir(), "etc", "sysctl.d", "90-envctl-performance.conf")

	var calls [][]string
	writer := newDropinWriter(destination, func(_ context.Context, name string, args ...string) ([]byte, error) {
		calls = append(calls, append([]string{name}, args...))
		return nil, nil
	}, func() time.Time { return time.Unix(1, 0) }, false)

	// A missing destination is the interesting case: the writer must not need
	// to create anything itself before the elevated commands run.
	changed, _, err := writer.Install("vm.swappiness = 150\n", 0o644, false)
	if err != nil {
		t.Fatalf("Install failed: %v", err)
	}
	if !changed {
		t.Fatal("Install reported no change")
	}

	var installCall []string
	for _, call := range calls {
		if call[0] == "install" {
			installCall = call
		}
	}
	if installCall == nil {
		t.Fatalf("commands = %#v, want an install stage", calls)
	}
	scratch := installCall[len(installCall)-2]
	staged := installCall[len(installCall)-1]

	// The scratch file must live outside the destination tree.
	if strings.HasPrefix(scratch, filepath.Dir(destination)) {
		t.Fatalf("scratch file %q is inside the destination directory %q; an unprivileged create would fail with EACCES", scratch, filepath.Dir(destination))
	}
	// The staged copy must live inside it, or the rename is not atomic.
	if filepath.Dir(staged) != filepath.Dir(destination) {
		t.Fatalf("staged copy %q is not in the destination directory %q", staged, filepath.Dir(destination))
	}
	// The directory itself is created by an elevated mkdir, not by the process.
	sawMkdir := false
	for _, call := range calls {
		if call[0] == "mkdir" && len(call) >= 2 && call[1] == "-p" {
			sawMkdir = true
		}
	}
	if !sawMkdir {
		t.Fatalf("commands = %#v, want an elevated mkdir -p for the drop-in directory", calls)
	}
}
