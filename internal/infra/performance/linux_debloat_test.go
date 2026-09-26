package performance

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/eajdias/envctl/internal/domain/entity"
	"github.com/eajdias/envctl/internal/domain/repository"
)

type debloatFixture struct {
	spec      entity.DebloatSpec
	installed map[string]string
	calls     [][]string
	writer    *dropinWriter
	removers  map[entity.PackageType]repository.PackageRemover
	failOn    map[string]error
}

func newDebloatFixture(t *testing.T) *debloatFixture {
	t.Helper()
	dir := t.TempDir()
	f := &debloatFixture{
		spec: entity.DebloatSpec{
			NeedrestartDropin: "/etc/needrestart/conf.d/99-envctl.conf",
			Removals: []entity.PackageRemoval{
				{ID: "modemmanager", Package: "modemmanager", Type: entity.PackageTypeApt, Category: "desktop-daemons", Rationale: "a modem manager on a headless server"},
				{ID: "rpcbind", Package: "rpcbind", Type: entity.PackageTypeApt, Category: "network", Rationale: "portmapper with no exported share", ProtectedWhen: []string{"nfs-common"}},
				{ID: "avahi", Package: "avahi-daemon", Type: entity.PackageTypeApt, Category: "discovery", Rationale: "mDNS responder"},
			},
		},
		installed: map[string]string{},
		calls:     nil,
		failOn:    map[string]error{},
	}
	f.writer = newDropinWriter(dir+"/99-envctl.conf", func(_ context.Context, name string, args ...string) ([]byte, error) {
		f.calls = append(f.calls, append([]string{name}, args...))
		if err, ok := f.failOn[name]; ok && err != nil {
			return nil, err
		}
		if name == "install" {
			content, readErr := os.ReadFile(args[len(args)-2])
			if readErr != nil {
				return nil, readErr
			}
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return nil, err
			}
			return nil, os.WriteFile(args[len(args)-1], content, 0o644)
		}
		if name == "mv" {
			content, readErr := os.ReadFile(args[len(args)-2])
			if readErr != nil {
				return nil, readErr
			}
			return nil, os.WriteFile(args[len(args)-1], content, 0o644)
		}
		return nil, nil
	}, func() time.Time { return time.Unix(1, 0) }, false)
	f.removers = map[entity.PackageType]repository.PackageRemover{
		entity.PackageTypeApt: newAptRemover(f.run, f.isInstalled, false),
	}
	return f
}

func (f *debloatFixture) run(_ context.Context, name string, args ...string) ([]byte, error) {
	f.calls = append(f.calls, append([]string{name}, args...))
	if err, ok := f.failOn[name]; ok && err != nil {
		return nil, err
	}
	return nil, nil
}

func (f *debloatFixture) isInstalled(_ context.Context, pkg entity.Package) (bool, string, error) {
	if version, ok := f.installed[pkg.ID]; ok {
		return true, version, nil
	}
	return false, "", nil
}

func (f *debloatFixture) issued(want string) bool {
	for _, call := range f.calls {
		if call[0] == want {
			return true
		}
	}
	return false
}

func (f *debloatFixture) manager() *linuxDebloatManager {
	return newLinuxDebloatManager(f.writer, f.removers)
}

func TestDebloatPurgesAnInstalledPackage(t *testing.T) {
	f := newDebloatFixture(t)
	f.installed["modemmanager"] = "1.25.95-1ubuntu1"

	diags, err := f.manager().Apply(context.Background(), f.spec, false)
	if err != nil {
		t.Fatalf("debloat failed: %v", err)
	}
	if !f.issued("apt-get") {
		t.Fatalf("no apt-get purge was issued: %#v", f.calls)
	}
	var purge []string
	for _, call := range f.calls {
		if call[0] == "apt-get" {
			purge = call
		}
	}
	if !contains(purge, "purge") || !contains(purge, "modemmanager") {
		t.Fatalf("purge argv = %#v", purge)
	}
	sawOK := false
	for _, diag := range diags {
		if diag.Category == entity.DiagOK && strings.Contains(diag.Details, "modemmanager") {
			sawOK = true
		}
	}
	if !sawOK {
		t.Fatalf("diagnostics = %#v, want an OK for the purge", diags)
	}
}

// The needrestart drop-in must be written before the first purge. Ubuntu ships
// needrestart in automatic mode (measured: 3.11-1ubuntu2, no override) and it
// restarts services after a package change, which over SSH can drop the session
// performing the change.
func TestDebloatInstallsTheNeedrestartGuardBeforeAnyPurge(t *testing.T) {
	f := newDebloatFixture(t)
	f.installed["modemmanager"] = "1.25.95-1ubuntu1"

	if _, err := f.manager().Apply(context.Background(), f.spec, false); err != nil {
		t.Fatalf("debloat failed: %v", err)
	}
	guardIndex, purgeIndex := -1, -1
	for i, call := range f.calls {
		if call[0] == "install" && guardIndex < 0 {
			guardIndex = i
		}
		if call[0] == "apt-get" && purgeIndex < 0 {
			purgeIndex = i
		}
	}
	if guardIndex < 0 {
		t.Fatalf("the needrestart drop-in was never written: %#v", f.calls)
	}
	if purgeIndex < 0 {
		t.Fatalf("no purge was issued: %#v", f.calls)
	}
	if guardIndex > purgeIndex {
		t.Fatalf("the guard was written at %d, after the purge at %d: %#v", guardIndex, purgeIndex, f.calls)
	}
	if content := readFileString(t, f.writer.destination); !strings.Contains(content, "NEEDRESTART_MODE=l") {
		t.Fatalf("the guard does not force list-only mode: %q", content)
	}
}

// nfs-common is an installed reverse dependency of rpcbind on the Oracle hosts,
// so the purge is skipped and reported instead of cascading.
func TestDebloatSkipsAGuardedRemoval(t *testing.T) {
	f := newDebloatFixture(t)
	f.installed["modemmanager"] = "1.25.95-1ubuntu1"
	f.installed["rpcbind"] = "1.2.7-1build2"
	f.installed["nfs-common"] = "1.2.6-1ubuntu1"

	diags, err := f.manager().Apply(context.Background(), f.spec, false)
	if err != nil {
		t.Fatalf("debloat failed: %v", err)
	}
	for _, call := range f.calls {
		if call[0] == "apt-get" && contains(call, "rpcbind") {
			t.Fatalf("a guarded removal was purged anyway: %#v", f.calls)
		}
	}
	sawGuard := false
	for _, diag := range diags {
		if diag.Category == entity.DiagInfo && strings.Contains(diag.Details, "nfs-common") {
			sawGuard = true
		}
	}
	if !sawGuard {
		t.Fatalf("diagnostics = %#v, want an INFO naming the guard", diags)
	}
}

// avahi-daemon is absent from every measured cloud image, so the run must report
// it as already clean without invoking the package manager.
func TestDebloatReportsAnAbsentPackageWithoutInvokingApt(t *testing.T) {
	f := newDebloatFixture(t)

	diags, err := f.manager().Apply(context.Background(), f.spec, false)
	if err != nil {
		t.Fatalf("debloat failed: %v", err)
	}
	if f.issued("apt-get") {
		t.Fatalf("an absent package still invoked apt: %#v", f.calls)
	}
	sawOK := false
	for _, diag := range diags {
		if diag.Category == entity.DiagOK && strings.Contains(diag.Details, "already absent") {
			sawOK = true
		}
	}
	if !sawOK {
		t.Fatalf("diagnostics = %#v, want an OK reporting the absence", diags)
	}
}

func TestDebloatDryRunIsInert(t *testing.T) {
	f := newDebloatFixture(t)
	f.installed["modemmanager"] = "1.25.95-1ubuntu1"

	diags, err := f.manager().Apply(context.Background(), f.spec, true)
	if err != nil {
		t.Fatalf("dry-run failed: %v", err)
	}
	if len(f.calls) != 0 {
		t.Fatalf("dry-run issued %#v", f.calls)
	}
	for _, diag := range diags {
		if diag.Category != entity.DiagInfo {
			t.Fatalf("dry-run diagnostic is not INFO: %#v", diag)
		}
	}
}

func TestDebloatReportsAPurgeFailure(t *testing.T) {
	f := newDebloatFixture(t)
	f.installed["modemmanager"] = "1.25.95-1ubuntu1"
	f.failOn["apt-get"] = fmt.Errorf("synthetic purge failure")

	diags, err := f.manager().Apply(context.Background(), f.spec, false)
	if err == nil {
		t.Fatal("expected a purge failure to be reported")
	}
	if len(diags) != 1 || diags[0].Category != entity.DiagError {
		t.Fatalf("diagnostics = %#v, want an ERROR", diags)
	}
}

// Purging as a non-root user must go through sudo -n so the run can never block
// on a password prompt in an automated context.
func TestAptRemoverUsesNoninteractiveSudo(t *testing.T) {
	var seen [][]string
	remover := newAptRemover(func(_ context.Context, name string, args ...string) ([]byte, error) {
		seen = append(seen, append([]string{name}, args...))
		return nil, nil
	}, func(context.Context, entity.Package) (bool, string, error) { return false, "", nil }, true)

	if err := remover.Remove(context.Background(), entity.Package{ID: "modemmanager", Type: entity.PackageTypeApt}); err != nil {
		t.Fatal(err)
	}
	if len(seen) != 1 || seen[0][0] != "sudo" || seen[0][1] != "-n" || seen[0][2] != "apt-get" {
		t.Fatalf("elevated purge = %#v, want sudo -n apt-get", seen)
	}
	if !contains(seen[0], "purge") {
		t.Fatalf("purge argv = %#v", seen[0])
	}
}

func TestPacmanRemoverArgv(t *testing.T) {
	var seen [][]string
	remover := newPacmanRemover(func(_ context.Context, name string, args ...string) ([]byte, error) {
		seen = append(seen, append([]string{name}, args...))
		return nil, nil
	}, func(context.Context, entity.Package) (bool, string, error) { return false, "", nil }, false)

	if err := remover.Remove(context.Background(), entity.Package{ID: "fwupd", Type: entity.PackageTypePacman}); err != nil {
		t.Fatal(err)
	}
	if !contains(seen[0], "-R") || !contains(seen[0], "--noconfirm") {
		t.Fatalf("pacman removal argv = %#v, want -R --noconfirm", seen[0])
	}
}

func TestDebloatRejectsAnInvalidSpec(t *testing.T) {
	f := newDebloatFixture(t)
	broken := f.spec
	broken.NeedrestartDropin = ""
	if _, err := f.manager().Apply(context.Background(), broken, false); err == nil {
		t.Fatal("expected a spec without the needrestart guard to be rejected")
	}
	if len(f.calls) != 0 {
		t.Fatalf("an invalid spec still issued %#v", f.calls)
	}
}

func TestAptRemoverReportsMissingBinary(t *testing.T) {
	remover := newAptRemover(
		func(context.Context, string, ...string) ([]byte, error) {
			return nil, exec.ErrNotFound
		},
		func(context.Context, entity.Package) (bool, string, error) { return true, "1", nil },
		false,
	)
	if err := remover.Remove(context.Background(), entity.Package{ID: "x", Type: entity.PackageTypeApt}); err == nil {
		t.Fatal("expected a missing package manager to be reported")
	}
}

func contains(list []string, want string) bool {
	for _, item := range list {
		if item == want {
			return true
		}
	}
	return false
}

func readFileString(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}
