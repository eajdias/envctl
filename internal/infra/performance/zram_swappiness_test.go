package performance

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/eajdias/envctl/internal/domain/entity"
)

// DeriveSwappiness must key off the measured swap topology, not the RAM tier.
//
// The kernel doc (admin-guide/sysctl/vm.html) states: "For in-memory swap, like
// zram or zswap, as well as hybrid setups that have swap on faster devices than
// the filesystem, values beyond 100 can be considered." So a zram-first host
// wants 150 and a disk-only host wants a low value. The previous manifest pinned
// 10 unconditionally while also installing zram, which told the kernel not to
// use the device the profile had just created.
func TestDeriveSwappiness(t *testing.T) {
	cases := []struct {
		name      string
		host      string
		state     entity.HardwareState
		want      int
		wantWords []string
	}{
		{
			name:  "zram active",
			host:  "post-Task-8 vps_oracle_2",
			state: entity.NewHardwareState(974092, 2, "ext4", 0, parseSwapTable(swapWithZRAM)),
			want:  150,
			wantWords: []string{
				"zram", "in-memory", "beyond 100",
			},
		},
		{
			name:  "disk swap only",
			host:  "vps_oracle_2 as measured today",
			state: entity.NewHardwareState(974092, 2, "ext4", 0, parseSwapTable(swapDiskOnly)),
			want:  10,
			wantWords: []string{
				"disk",
			},
		},
		{
			name:  "no swap yet",
			host:  "zscan_chatbot as measured today",
			state: entity.NewHardwareState(16162396, 4, "ext4", 0, parseSwapTable(swapNone)),
			want:  10,
			wantWords: []string{
				"disk",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			value, rationale := entity.DeriveSwappiness(tc.state)
			if value != tc.want {
				t.Fatalf("DeriveSwappiness = %d, want %d (host %s)", value, tc.want, tc.host)
			}
			lowered := strings.ToLower(rationale)
			for _, word := range tc.wantWords {
				if !strings.Contains(lowered, word) {
					t.Fatalf("rationale %q does not mention %q (host %s)", rationale, word, tc.host)
				}
			}
		})
	}
}

// Deriving must not depend on the declared settings at all: the value is a
// function of what the host actually has.
func TestDeriveSwappinessIgnoresTheDeclaredValue(t *testing.T) {
	state := entity.NewHardwareState(974092, 2, "ext4", 0, parseSwapTable(swapWithZRAM))
	withZram, _ := entity.DeriveSwappiness(state)
	diskOnly, _ := entity.DeriveSwappiness(entity.NewHardwareState(974092, 2, "ext4", 0, parseSwapTable(swapDiskOnly)))
	if withZram == diskOnly {
		t.Fatalf("zram and disk-only hosts derived the same value %d", withZram)
	}
}

func newZRAMManagerForTest(t *testing.T, present bool, devices []string, calls *[][]string) *zramManager {
	t.Helper()
	manager := newZRAMManager(
		func() bool { return present },
		fakeDropinFS(t, nil, calls),
		false,
		func(context.Context, func() bool) bool { return true },
		func() []string { return devices },
	)
	manager.hasNode = func() bool { return true }
	return manager
}

// A tier that enables zram on a host without a device must run the lifecycle.
func TestZRAMManagerRunsWhenTheTierEnablesIt(t *testing.T) {
	var calls [][]string
	manager := newZRAMManagerForTest(t, false, nil, &calls)

	diags, err := manager.Ensure(context.Background(), false)
	if err != nil {
		t.Fatalf("Ensure failed: %v", err)
	}
	if len(diags) != 1 || diags[0].Category != entity.DiagOK {
		t.Fatalf("diagnostics = %#v, want one OK", diags)
	}
	sawModprobe := false
	for _, call := range calls {
		if call[0] == "modprobe" {
			sawModprobe = true
		}
	}
	if !sawModprobe {
		t.Fatalf("commands = %#v, want the zram module loaded", calls)
	}
}

// A tier that disables zram must not touch the host at all: RAM is not the
// constraint, so a compressed-RAM tier would compete with the page cache. The
// gate lives in ResolvedZRAMPolicy, which the use case consults before it ever
// calls the manager, so "not calling it" is what has to be asserted.
func TestResolvedZRAMPolicyGatesOnTheTier(t *testing.T) {
	zramOn, zramOff := true, false
	cases := []struct {
		name     string
		policy   string
		tier     entity.PerformanceTier
		memKB    uint64
		wantRun  bool
		wantWord string
	}{
		{
			name: "tiny tier enables zram", policy: "tier",
			tier: entity.PerformanceTier{ID: "tiny", EnableZRAM: &zramOn}, memKB: 974092,
			wantRun: true, wantWord: "tiny",
		},
		{
			name: "medium tier does not", policy: "tier",
			tier: entity.PerformanceTier{ID: "medium", EnableZRAM: &zramOff}, memKB: 8388608,
			wantRun: false, wantWord: "page cache",
		},
		{
			name: "large tier does not", policy: "tier",
			tier: entity.PerformanceTier{ID: "large", EnableZRAM: &zramOff}, memKB: 16162396,
			wantRun: false, wantWord: "page cache",
		},
		{
			name: "profile-wide always overrides the tier", policy: "always",
			tier: entity.PerformanceTier{ID: "large", EnableZRAM: &zramOff}, memKB: 16162396,
			wantRun: true, wantWord: "every tier",
		},
		{
			name: "profile-wide disabled overrides the tier", policy: "disabled",
			tier: entity.PerformanceTier{ID: "tiny", EnableZRAM: &zramOn}, memKB: 974092,
			wantRun: false, wantWord: "disabled",
		},
		{
			name: "an omitted policy behaves as tier", policy: "",
			tier: entity.PerformanceTier{ID: "tiny", EnableZRAM: &zramOn}, memKB: 974092,
			wantRun: true, wantWord: "tiny",
		},
		{
			name: "an unknown policy never runs", policy: "sometimes",
			tier: entity.PerformanceTier{ID: "tiny", EnableZRAM: &zramOn}, memKB: 974092,
			wantRun: false, wantWord: "unknown",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			hw := entity.NewHardwareState(tc.memKB, 2, "ext4", 0, nil)
			run, detail := entity.ResolvedZRAMPolicy(tc.policy, tc.tier, hw)
			if run != tc.wantRun {
				t.Fatalf("run = %v, want %v (detail %q)", run, tc.wantRun, detail)
			}
			if !strings.Contains(detail, tc.wantWord) {
				t.Fatalf("detail %q does not explain the decision (%q)", detail, tc.wantWord)
			}
		})
	}
}

// A second pre-existing zram device is still refused: creating another would
// hide memory the operator allocated on purpose.
func TestZRAMManagerStillRefusesASecondDevice(t *testing.T) {
	var calls [][]string
	manager := newZRAMManagerForTest(t, false, []string{"/dev/zram1"}, &calls)

	diags, err := manager.Ensure(context.Background(), false)
	if err != nil {
		t.Fatalf("Ensure failed: %v", err)
	}
	if len(diags) != 1 || diags[0].Category != entity.DiagInfo || !strings.Contains(diags[0].Details, "zram1") {
		t.Fatalf("diagnostics = %#v, want an INFO naming the existing device", diags)
	}
	if len(calls) != 0 {
		t.Fatalf("a refusal still issued %#v", calls)
	}
}

// A disk swap whose priority already outranks zram is a conflict, not a
// preference: the ordering is the whole point of adding zram, and silently
// overwriting it would mask a hand-tuned decision.
func TestZRAMManagerReportsADiskSwapThatOutranksZRAM(t *testing.T) {
	state := entity.NewHardwareState(974092, 2, "ext4", 0, []entity.SwapDevice{
		{Name: "/dev/zram0", Type: "partition", SizeKB: 100, Priority: 100},
		{Name: "/swapfile", Type: "file", SizeKB: 100, Priority: 200},
	})
	conflict, detail := entity.DetectZRAMPriorityConflict(state, 100)
	if !conflict {
		t.Fatal("a disk swap at priority 200 outranking zram at 100 must be a conflict")
	}
	if !strings.Contains(detail, "200") || !strings.Contains(detail, "100") {
		t.Fatalf("detail %q does not state both priorities", detail)
	}
}

func TestZRAMManagerReportsNoConflictWhenZRAMWins(t *testing.T) {
	state := entity.NewHardwareState(974092, 2, "ext4", 0, parseSwapTable(swapWithZRAM))
	if conflict, _ := entity.DetectZRAMPriorityConflict(state, 100); conflict {
		t.Fatal("zram at 100 above a disk at -1 is the intended ordering")
	}
}

func TestZRAMManagerUsesNoninteractiveSudoForReexec(t *testing.T) {
	// Sanity: the shared command helper must prefix sudo -n when elevated, so
	// no privileged zram step can ever block on a password prompt.
	var seen [][]string
	writer := newDropinWriter("/tmp/unused.conf",
		func(_ context.Context, name string, args ...string) ([]byte, error) {
			seen = append(seen, append([]string{name}, args...))
			return nil, nil
		}, func() time.Time { return time.Unix(1, 0) }, true)
	if _, err := writer.command(context.Background(), "modprobe", "zram"); err != nil {
		t.Fatal(err)
	}
	if len(seen) != 1 || seen[0][0] != "sudo" || seen[0][1] != "-n" {
		t.Fatalf("elevated command = %#v, want sudo -n argv", seen)
	}
}
