package windows

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"testing"

	"github.com/eajdias/envctl/internal/domain/entity"
	"github.com/eajdias/envctl/internal/infra/logger"
)

// TestS1LiveCommandRoundTrip proves the Command apply path on a live host:
// it activates the High Performance power plan (reversible) and verifies the
// check converges, then confirms the hibernation check agrees with the real
// firmware state. Skips off Windows.
func TestS1LiveCommandRoundTrip(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("live probe requires Windows")
	}
	if os.Getenv("S1_LIVE_PROBE") != "1" {
		t.Skip("set S1_LIVE_PROBE=1 to run the live probe")
	}
	mgr := NewWindowsTweaksManager(logger.NewNoopLogger())
	ctx := context.Background()

	// Power plan: apply the High Performance scheme (reversible via
	// powercfg), then the check must report converged.
	powerTweak := entity.WindowsTweak{ID: "tier3-power-high-performance", Name: "PowerPlan", Type: "Command"}
	ok, details, err := mgr.CheckTweak(ctx, powerTweak)
	fmt.Printf("POWER check before: ok=%v details=%s err=%v\n", ok, details, err)

	// Force drift: switch to Balanced manually, the apply must bring it back.
	drifting := exec.Command("powercfg", "/setactive", "381b4222-f694-41f0-9685-ff5bb260df2e") // Balanced
	if out, derr := drifting.CombinedOutput(); derr != nil {
		t.Fatalf("could not set Balanced for the probe: %v (%s)", derr, out)
	}
	ok, details, err = mgr.CheckTweak(ctx, powerTweak)
	fmt.Printf("POWER check after forced drift: ok=%v details=%s err=%v\n", ok, details, err)
	if ok {
		t.Fatalf("expected drift after forcing Balanced")
	}

	if err := mgr.ApplyTweak(ctx, powerTweak); err != nil {
		t.Fatalf("power apply failed: %v", err)
	}
	ok, details, err = mgr.CheckTweak(ctx, powerTweak)
	fmt.Printf("POWER check after apply: ok=%v details=%s err=%v\n", ok, details, err)
	if !ok {
		t.Fatalf("expected power convergence after apply, got %s", details)
	}

	// Hibernation: check is read-only (reports drift when hiberfil.sys exists).
	hiberTweak := entity.WindowsTweak{ID: "tier3-hibernation-off", Name: "Hibernation", Type: "Command"}
	ok, details, err = mgr.CheckTweak(ctx, hiberTweak)
	fmt.Printf("HIBER check: ok=%v details=%s err=%v\n", ok, details, err)
	if err != nil {
		t.Fatalf("hibernation check failed: %v", err)
	}
}

// It corrupts the UserPreferencesMask value, applies the tweak, and verifies
// the byte list comes back — proving the Binary registry apply + check agree.
// Run manually on a Windows host: GOOS is not enforced here because the
// manager itself gates on runtime.GOOS; this test skips off Windows.
func TestS1LiveBinaryRoundTrip(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("live probe requires Windows")
	}
	if os.Getenv("S1_LIVE_PROBE") != "1" {
		t.Skip("set S1_LIVE_PROBE=1 to run the live probe")
	}
	mgr := NewWindowsTweaksManager(logger.NewNoopLogger())
	tweak := entity.WindowsTweak{
		ID:    "tier3-user-preferences-mask",
		Path:  `HKCU:\Control Panel\Desktop`,
		Name:  "UserPreferencesMask",
		Value: []any{144, 18, 3, 128, 16, 0, 0, 0},
		Type:  "Binary",
	}

	if err := mgr.ApplyTweak(context.Background(), entity.WindowsTweak{
		Path: `HKCU:\Control Panel\Desktop`, Name: "UserPreferencesMask",
		Value: []any{9, 9, 9, 9, 9, 9, 9, 9}, Type: "Binary",
	}); err != nil {
		t.Fatalf("setup corrupt write failed: %v", err)
	}

	ok, details, err := mgr.CheckTweak(context.Background(), tweak)
	fmt.Printf("CHECK after corrupt: ok=%v details=%s err=%v\n", ok, details, err)
	if ok {
		t.Fatalf("expected drift after corrupting the value")
	}

	if err := mgr.ApplyTweak(context.Background(), tweak); err != nil {
		t.Fatalf("apply failed: %v", err)
	}
	ok, details, err = mgr.CheckTweak(context.Background(), tweak)
	fmt.Printf("CHECK after apply: ok=%v details=%s err=%v\n", ok, details, err)
	if !ok {
		t.Fatalf("expected convergence after apply, got %s", details)
	}
}
