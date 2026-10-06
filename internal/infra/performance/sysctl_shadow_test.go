package performance

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/eajdias/envctl/internal/domain/entity"
)

// shadowFixture builds a manager over a fake sysctl.d tree. The profile's own
// drop-in lands in the tree, so the resolver sees exactly what it sees on a host.
func shadowFixture(t *testing.T, live map[string]string, hostFiles map[string]string) (*SysctlManager, string) {
	t.Helper()
	root := t.TempDir()
	etc := filepath.Join(root, "etc", "sysctl.d")
	if err := os.MkdirAll(etc, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, content := range hostFiles {
		if err := os.WriteFile(filepath.Join(etc, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	destination := filepath.Join(etc, "90-envctl-performance.conf")

	manager := newSysctlManager(
		destination,
		// Emulate the two commands dropinWriter shells out to, so the test
		// observes the file the writer actually produced.
		func(_ context.Context, name string, args ...string) ([]byte, error) {
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
				return nil, os.Rename(args[1], args[2])
			}
			return nil, nil
		},
		func() time.Time { return time.Unix(0, 0) },
		false,
		func(key string) (string, bool) {
			value, ok := live[key]
			return value, ok
		},
	)
	manager.sysctlDirs = []string{etc}
	return manager, destination
}

func findDiag(diags []entity.Diagnostic, target string) (entity.Diagnostic, bool) {
	for _, diag := range diags {
		if diag.Target == target {
			return diag, true
		}
	}
	return entity.Diagnostic{}, false
}

// TestSysctlApplyDoesNotFightAHostDropIn is the regression test for the defect
// found by rebooting a real host: the fleet ships
// /etc/sysctl.d/99-swappiness.conf, written by the cloud agent, which sorts
// after the profile's 90-envctl-performance.conf. envctl wrote
// vm.swappiness=150 and set it live; the next boot silently restored 10.
//
// Writing it again would not fix anything — the host would revert it at every
// boot, and the audit would flap forever. The profile owns no state the host
// does not, so it must yield and say so.
func TestSysctlApplyDoesNotFightAHostDropIn(t *testing.T) {
	manager, destination := shadowFixture(t,
		map[string]string{"vm.swappiness": "10", "net.core.somaxconn": "65535"},
		map[string]string{"99-swappiness.conf": "vm.swappiness=10\n"},
	)

	diags, err := manager.Apply(context.Background(), []entity.SysctlSetting{
		{Key: "vm.swappiness", Value: "150", Policy: entity.SysctlPolicySet,
			Rationale: "derived from the measured swap topology"},
		{Key: "net.core.somaxconn", Value: "65535", Policy: entity.SysctlPolicySet},
	}, false)
	if err != nil {
		t.Fatalf("a host-owned value must not fail the run: %v", err)
	}

	shadow, ok := findDiag(diags, "vm.swappiness")
	if !ok {
		t.Fatal("the shadowed key produced no diagnostic at all")
	}
	if shadow.Category != entity.DiagWarning {
		t.Fatalf("category %v, want a warning: the owner must see the disagreement", shadow.Category)
	}
	if !strings.Contains(shadow.Details, "99-swappiness.conf") {
		t.Fatalf("the diagnostic does not name the file that wins: %q", shadow.Details)
	}
	if !strings.Contains(shadow.Details, "10") || !strings.Contains(shadow.Details, "150") {
		t.Fatalf("the diagnostic must carry both values, got %q", shadow.Details)
	}
	if shadow.FixHint == "" {
		t.Fatal("the diagnostic carries no fix hint")
	}

	// The profile's own file must stop claiming a value that never takes effect.
	written, readErr := os.ReadFile(destination)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if strings.Contains(string(written), "vm.swappiness") {
		t.Fatalf("the profile drop-in still claims vm.swappiness:\n%s", written)
	}
	if !strings.Contains(string(written), "net.core.somaxconn = 65535") {
		t.Fatalf("the unshadowed key was not written:\n%s", written)
	}
}

// TestSysctlApplyReclaimsTheKeyWhenTheHostFileIsGone is the self-healing half:
// once the host file is removed, the profile takes the key back with no other
// change.
func TestSysctlApplyReclaimsTheKeyWhenTheHostFileIsGone(t *testing.T) {
	manager, destination := shadowFixture(t,
		map[string]string{"vm.swappiness": "10"},
		map[string]string{"95-unrelated.conf": "net.core.somaxconn = 4096\n"},
	)

	if _, err := manager.Apply(context.Background(), []entity.SysctlSetting{
		{Key: "vm.swappiness", Value: "150", Policy: entity.SysctlPolicySet},
	}, false); err != nil {
		t.Fatal(err)
	}

	written, err := os.ReadFile(destination)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(written), "vm.swappiness = 150") {
		t.Fatalf("the key was not reclaimed after the host file went away:\n%s", written)
	}
}

// A host file that agrees with the profile is not a conflict and must stay
// silent: reporting it would bury the real disagreement in noise.
func TestSysctlApplyIsSilentWhenTheHostAgrees(t *testing.T) {
	manager, _ := shadowFixture(t,
		map[string]string{"vm.swappiness": "150"},
		map[string]string{"99-swappiness.conf": "vm.swappiness=150\n"},
	)

	diags, err := manager.Apply(context.Background(), []entity.SysctlSetting{
		{Key: "vm.swappiness", Value: "150", Policy: entity.SysctlPolicySet},
	}, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, diag := range diags {
		if diag.Target == "vm.swappiness" && diag.Category == entity.DiagWarning {
			t.Fatalf("an agreeing host file produced a warning: %q", diag.Details)
		}
	}
}

// A dry run must stay read-only while still reporting the shadow, or the owner
// would only discover the conflict after writing.
func TestSysctlApplyReportsTheShadowOnADryRun(t *testing.T) {
	manager, destination := shadowFixture(t,
		map[string]string{"vm.swappiness": "10"},
		map[string]string{"99-swappiness.conf": "vm.swappiness=10\n"},
	)
	before, err := os.ReadFile(destination)
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}

	diags, err := manager.Apply(context.Background(), []entity.SysctlSetting{
		{Key: "vm.swappiness", Value: "150", Policy: entity.SysctlPolicySet},
	}, true)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := findDiag(diags, "vm.swappiness"); !ok {
		t.Fatal("the dry run did not report the shadow")
	}
	after, err := os.ReadFile(destination)
	if err != nil {
		if os.IsNotExist(err) && len(before) == 0 {
			return
		}
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatal("a dry run wrote to the drop-in")
	}
}
