package performance

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/eajdias/envctl/internal/domain/entity"
)

const testFstab = "/etc/fstab"

type swapFixture struct {
	dir      string
	spec     entity.SwapSpec
	hw       entity.HardwareState
	calls    [][]string
	existing map[string][]byte
	failOn   map[string]error
	// statfsValues simulates the file's existence and content.
	fileExists map[string]bool
}

func newSwapFixture(t *testing.T) *swapFixture {
	t.Helper()
	dir := t.TempDir()
	// The fstab path resolves to <dir>/etc/fstab; the writer creates the
	// parent, but making it here keeps the failure modes distinct.
	if err := os.MkdirAll(filepath.Join(dir, "etc"), 0o755); err != nil {
		t.Fatal(err)
	}
	spec := entity.SwapSpec{
		Policy:      entity.SwapPolicyAuto,
		File:        "/swapfile.envctl",
		Priority:    -2,
		SizeOf:      entity.SwapSizeOfMemTotal,
		SizeMin:     "1G",
		SizeMax:     "8G",
		DiskReserve: "5G",
		FSAllow:     []string{"ext4", "xfs"},
		FSBtrfs:     "mkswapfile",
		FSDeny:      []string{"zfs", "overlay", "tmpfs"},
	}
	return &swapFixture{
		dir:  dir,
		spec: spec,
		hw:   entity.NewHardwareState(974092, 2, "ext4", 34_000_000_000, nil),
		// Rewrite the declared paths onto the temp dir so the test never writes
		// to a real /etc.
		existing:   map[string][]byte{},
		failOn:     map[string]error{},
		fileExists: map[string]bool{},
	}
}

func (f *swapFixture) path(absolute string) string {
	// /swapfile.envctl -> <dir>/swapfile.envctl ; /etc/fstab -> <dir>/etc/fstab
	rel := strings.TrimPrefix(absolute, "/")
	return filepath.Join(f.dir, rel)
}

func (f *swapFixture) manager() *swapfileManager {
	spec := f.spec
	spec.File = f.path(spec.File)
	fstabWriter := newDropinWriter(f.path(testFstab), f.run, func() time.Time { return time.Unix(1, 0) }, false)
	exists := func(absolute string) bool { return f.fileExists[absolute] }
	readFile := func(absolute string) ([]byte, error) {
		if data, ok := f.existing[absolute]; ok {
			return data, nil
		}
		return nil, os.ErrNotExist
	}
	return newSwapfileManager(f.run, fstabWriter, exists, readFile, func() bool { return false }, false)
}

func (f *swapFixture) run(_ context.Context, name string, args ...string) ([]byte, error) {
	f.calls = append(f.calls, append([]string{name}, args...))
	if err, ok := f.failOn[name]; ok && err != nil {
		return nil, err
	}
	switch name {
	case "fallocate", "truncate":
		target := args[len(args)-1]
		f.existing[target] = []byte("allocated")
	case "chattr", "chmod", "mkswap", "swapon", "btrfs":
		// recorded only
	case "install":
		content, err := os.ReadFile(args[len(args)-2])
		if err != nil {
			return nil, err
		}
		if err := os.WriteFile(args[len(args)-1], content, 0o644); err != nil {
			return nil, err
		}
		f.existing[args[len(args)-1]] = content
	case "mv":
		content, err := os.ReadFile(args[len(args)-2])
		if err != nil {
			return nil, err
		}
		if err := os.WriteFile(args[len(args)-1], content, 0o644); err != nil {
			return nil, err
		}
		f.existing[args[len(args)-1]] = content
		_ = os.Remove(args[len(args)-2])
	}
	return nil, nil
}

func (f *swapFixture) issued(want string) bool {
	for _, call := range f.calls {
		if call[0] == want {
			return true
		}
	}
	return false
}

func (f *swapFixture) argOf(want string) []string {
	for _, call := range f.calls {
		if call[0] == want {
			return call
		}
	}
	return nil
}

// The measured Oracle shape: a hand-created /swapfile already exists, so the
// tool must be a complete no-op. No fallocate, no mkswap, no swapon, no fstab
// write.
func TestSwapfileManagerAdoptsAnExistingDiskSwap(t *testing.T) {
	f := newSwapFixture(t)
	f.hw = entity.NewHardwareState(974092, 2, "ext4", 34_000_000_000, []entity.SwapDevice{
		{Name: "/swapfile", Type: "file", SizeKB: 8388604, Priority: -1},
	})

	diags, err := f.manager().Ensure(context.Background(), f.spec, f.hw, false)
	if err != nil {
		t.Fatalf("adoption failed: %v", err)
	}
	if len(diags) != 1 || diags[0].Category != entity.DiagOK {
		t.Fatalf("diagnostics = %#v, want one OK", diags)
	}
	if !strings.Contains(diags[0].Details, "/swapfile") || !strings.Contains(diags[0].Details, "adopt") {
		t.Fatalf("diagnostic %q does not state that the existing swap was adopted", diags[0].Details)
	}
	for _, forbidden := range []string{"fallocate", "mkswap", "swapon", "mv", "install"} {
		if f.issued(forbidden) {
			t.Fatalf("adoption issued %q: %#v", forbidden, f.calls)
		}
	}
}

func TestSwapfileManagerCreatesOnExt4(t *testing.T) {
	f := newSwapFixture(t)

	diags, err := f.manager().Ensure(context.Background(), f.spec, f.hw, false)
	if err != nil {
		t.Fatalf("creation failed: %v", err)
	}
	if len(diags) == 0 || diags[0].Category != entity.DiagOK {
		t.Fatalf("diagnostics = %#v, want OK", diags)
	}
	for _, want := range []string{"fallocate", "chmod", "mkswap", "swapon"} {
		if !f.issued(want) {
			t.Fatalf("missing %q in %#v", want, f.calls)
		}
	}
	// Priority must be expressed in fstab, not left to the tool's default.
	written := string(f.existing[f.path(testFstab)])
	if !strings.Contains(written, "pri=-2") {
		t.Fatalf("fstab entry does not carry the declared priority: %q", written)
	}
	// The rendered entry carries the declared path exactly as the manifest
	// states it, which is what an operator reads.
	if !strings.Contains(written, "/swapfile.envctl") {
		t.Fatalf("fstab entry does not reference the declared file: %q", written)
	}
}

func TestSwapfileManagerUsesTheBtrfsSequence(t *testing.T) {
	f := newSwapFixture(t)
	f.hw = entity.NewHardwareState(8388608, 2, "btrfs", 100_000_000_000, nil)

	if _, err := f.manager().Ensure(context.Background(), f.spec, f.hw, false); err != nil {
		t.Fatalf("btrfs creation failed: %v", err)
	}
	// btrfs needs NODATACOW and no holes, which is truncate + chattr + fallocate.
	for _, want := range []string{"truncate", "chattr", "fallocate"} {
		if !f.issued(want) {
			t.Fatalf("missing %q in the btrfs sequence: %#v", want, f.calls)
		}
	}
	if f.argOf("chattr") == nil || !strings.Contains(strings.Join(f.argOf("chattr"), " "), "+C") {
		t.Fatalf("btrfs sequence did not set NODATACOW: %#v", f.argOf("chattr"))
	}
}

func TestSwapfileManagerRefusesAnUnsupportedFilesystem(t *testing.T) {
	for _, fsType := range []string{"zfs", "overlay", "unknown-deadbeef"} {
		t.Run(fsType, func(t *testing.T) {
			f := newSwapFixture(t)
			f.hw = entity.NewHardwareState(974092, 2, fsType, 34_000_000_000, nil)

			diags, err := f.manager().Ensure(context.Background(), f.spec, f.hw, false)
			if err != nil {
				t.Fatalf("refusal returned an error instead of a diagnostic: %v", err)
			}
			if len(diags) != 1 || diags[0].Category != entity.DiagInfo || !strings.Contains(diags[0].Details, fsType) {
				t.Fatalf("diagnostics = %#v, want an INFO naming the filesystem", diags)
			}
			if len(f.calls) != 0 {
				t.Fatalf("a refusal still issued %#v", f.calls)
			}
		})
	}
}

// A non-swap file sitting at the declared path must never be handed to mkswap.
func TestSwapfileManagerRefusesToOverwriteANonSwapFile(t *testing.T) {
	f := newSwapFixture(t)
	f.fileExists[f.spec.File] = true

	diags, err := f.manager().Ensure(context.Background(), f.spec, f.hw, false)
	if err == nil {
		t.Fatal("expected a refusal for a non-swap file at the declared path")
	}
	if len(diags) != 1 || diags[0].Category != entity.DiagError {
		t.Fatalf("diagnostics = %#v, want an ERROR", diags)
	}
	if f.issued("mkswap") {
		t.Fatalf("mkswap was issued over an unrelated file: %#v", f.calls)
	}
}

// An already-active swapfile at the declared path is a valid pre-existing
// state, not a conflict: it is ours from a previous run.
func TestSwapfileManagerAcceptsItsOwnExistingSwapfile(t *testing.T) {
	f := newSwapFixture(t)
	f.hw = entity.NewHardwareState(974092, 2, "ext4", 34_000_000_000, []entity.SwapDevice{
		{Name: f.path("/swapfile.envctl"), Type: "file", SizeKB: 1048576, Priority: -2},
	})

	diags, err := f.manager().Ensure(context.Background(), f.spec, f.hw, false)
	if err != nil {
		t.Fatalf("re-run failed: %v", err)
	}
	if len(diags) != 1 || diags[0].Category != entity.DiagOK {
		t.Fatalf("diagnostics = %#v, want one OK on a converged host", diags)
	}
	if f.issued("fallocate") || f.issued("mkswap") {
		t.Fatalf("a converged host was re-created: %#v", f.calls)
	}
}

// A disk swap that already outranks zram is reported, not silently reordered.
func TestSwapfileManagerReportsAPriorityConflict(t *testing.T) {
	f := newSwapFixture(t)
	f.hw = entity.NewHardwareState(974092, 2, "ext4", 34_000_000_000, []entity.SwapDevice{
		{Name: "/swapfile", Type: "file", SizeKB: 8388604, Priority: 200},
	})

	diags, err := f.manager().Ensure(context.Background(), f.spec, f.hw, false)
	if err == nil {
		t.Fatal("expected a priority conflict to be reported")
	}
	if !strings.Contains(diags[0].Details, "200") {
		t.Fatalf("diagnostic %q does not state the existing priority", diags[0].Details)
	}
}

func TestSwapfileManagerDryRunWritesNothing(t *testing.T) {
	f := newSwapFixture(t)
	diags, err := f.manager().Ensure(context.Background(), f.spec, f.hw, true)
	if err != nil {
		t.Fatalf("dry-run failed: %v", err)
	}
	if len(f.calls) != 0 {
		t.Fatalf("dry-run issued %#v", f.calls)
	}
	if len(diags) != 1 || diags[0].Category != entity.DiagInfo || !strings.Contains(diags[0].Details, "would") {
		t.Fatalf("diagnostics = %#v, want an INFO describing what it would do", diags)
	}
}

func TestSwapfileManagerReportsSizingRefusal(t *testing.T) {
	f := newSwapFixture(t)
	small := entity.NewHardwareState(974092, 2, "ext4", 3_000_000_000, nil)

	diags, err := f.manager().Ensure(context.Background(), f.spec, small, false)
	if err == nil {
		t.Fatal("expected a sizing refusal")
	}
	if len(diags) != 1 || diags[0].Category != entity.DiagError {
		t.Fatalf("diagnostics = %#v, want an ERROR", diags)
	}
	if len(f.calls) != 0 {
		t.Fatalf("a refusal still issued %#v", f.calls)
	}
}

func TestSwapfileManagerDisabledPolicyIsInert(t *testing.T) {
	f := newSwapFixture(t)
	disabled := f.spec
	disabled.Policy = entity.SwapPolicyDisabled

	diags, err := f.manager().Ensure(context.Background(), disabled, f.hw, false)
	if err != nil {
		t.Fatalf("disabled policy failed: %v", err)
	}
	if len(diags) != 1 || diags[0].Category != entity.DiagInfo {
		t.Fatalf("diagnostics = %#v, want an INFO", diags)
	}
	if len(f.calls) != 0 {
		t.Fatalf("a disabled policy still issued %#v", f.calls)
	}
}

func TestSwapfileManagerSkipsAnIdenticalFstabEntry(t *testing.T) {
	f := newSwapFixture(t)
	// Pre-populate fstab with exactly the entry the manager would render.
	first := f.manager()
	if _, err := first.Ensure(context.Background(), f.spec, f.hw, false); err != nil {
		t.Fatal(err)
	}
	f.calls = nil

	second := f.manager()
	if _, err := second.Ensure(context.Background(), f.spec, f.hw, true); err != nil {
		t.Fatal(err)
	}
	// A dry-run never writes, so assert the fstab content survived untouched.
	if !strings.Contains(string(f.existing[f.path(testFstab)]), "pri=-2") {
		t.Fatalf("fstab was damaged: %q", f.existing[f.path(testFstab)])
	}
}

func TestSwapfileManagerSurfacesCreationFailure(t *testing.T) {
	f := newSwapFixture(t)
	f.failOn["swapon"] = errors.New("synthetic swapon failure")

	diags, err := f.manager().Ensure(context.Background(), f.spec, f.hw, false)
	if err == nil {
		t.Fatal("expected a swapon failure to be reported")
	}
	if len(diags) != 1 || diags[0].Category != entity.DiagError {
		t.Fatalf("diagnostics = %#v, want an ERROR", diags)
	}
	if !strings.Contains(diags[0].Details, "swapon") {
		t.Fatalf("diagnostic %q does not name the failing step", diags[0].Details)
	}
}

func TestFstabEntryRendering(t *testing.T) {
	entry := renderFstabEntry("/swapfile.envctl", -2)
	if !strings.HasPrefix(entry, "/swapfile.envctl") {
		t.Fatalf("entry %q does not start with the path", entry)
	}
	if !strings.Contains(entry, "pri=-2") {
		t.Fatalf("entry %q does not carry the priority", entry)
	}
	if !strings.HasSuffix(entry, "\n") {
		t.Fatalf("entry %q is not newline terminated", entry)
	}
}

func TestFstabEntryIsNotDuplicated(t *testing.T) {
	existing := "UUID=abc / ext4 defaults 0 1\n/swapfile.envctl none swap sw,pri=-2 0 0\n"
	if !fstabHasSwapEntry(existing, "/swapfile.envctl") {
		t.Fatal("the existing entry was not detected")
	}
	if fstabHasSwapEntry(existing, "/other") {
		t.Fatal("an unrelated path was reported as present")
	}
}
