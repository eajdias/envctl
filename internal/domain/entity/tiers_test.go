package entity

import "testing"

// fleetTiers mirrors the tier list the Ubuntu server manifest declares. The
// boundaries are deliberately ~50% wider than the nominal instance sizes
// because MemTotal is always short of them:
//
//	"1 GB"  ->  951 MiB measured, so tiny ends at 1536 MiB
//	"4 GB"  -> 4096 MiB synthetic, so small ends at 4608 MiB
//	"8 GB"  -> 8192 MiB synthetic, so medium ends at 9216 MiB
//	"16 GB" -> 15783 MiB measured, so large is unbounded
func fleetTiers() []PerformanceTier {
	zramOn, zramOff := true, false
	return []PerformanceTier{
		{ID: "tiny", MatchMemTotalMax: 1536, EnableZRAM: &zramOn, Rationale: "validated"},
		{ID: "small", MatchMemTotalMax: 4608, EnableZRAM: &zramOn, Rationale: "derived"},
		{ID: "medium", MatchMemTotalMax: 9216, EnableZRAM: &zramOff, Rationale: "derived"},
		{ID: "large", MatchMemTotalMax: 0, EnableZRAM: &zramOff, Rationale: "validated"},
	}
}

func TestSelectPerformanceTier(t *testing.T) {
	cases := []struct {
		name       string
		host       string
		memTotalKB uint64
		want       string
	}{
		{"oracle 1 GB shape", "vps_oracle_2 / vps_oracle_1", 974092, "tiny"},
		{"aws 16 GB shape", "zscan_chatbot", 16162396, "large"},
		{"synthetic 4 GB", "no host in the fleet", 4194304, "small"},
		{"synthetic 8 GB", "no host in the fleet", 8388608, "medium"},
		{"exactly at the tiny boundary is inclusive", "boundary guard", 1536 * 1024, "tiny"},
		{"one MiB above the tiny boundary", "boundary guard", 1537 * 1024, "small"},
		{"exactly at the medium boundary is inclusive", "boundary guard", 9216 * 1024, "medium"},
		{"one MiB above the medium boundary", "boundary guard", 9217 * 1024, "large"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tier, err := SelectPerformanceTier(HardwareState{MemTotalKB: tc.memTotalKB}, fleetTiers())
			if err != nil {
				t.Fatalf("SelectPerformanceTier failed: %v", err)
			}
			if tier.ID != tc.want {
				t.Fatalf("tier = %q, want %q (host %s)", tier.ID, tc.want, tc.host)
			}
		})
	}
}

// An overlapping list is a manifest bug that would silently pick the first
// match, so it must be an error rather than a silent preference.
func TestSelectPerformanceTierRejectsOverlappingBands(t *testing.T) {
	tiers := []PerformanceTier{
		{ID: "first", MatchMemTotalMax: 4096},
		{ID: "second", MatchMemTotalMax: 2048},
		{ID: "unbounded", MatchMemTotalMax: 0},
	}
	if _, err := SelectPerformanceTier(HardwareState{MemTotalKB: 974092}, tiers); err == nil {
		t.Fatal("expected overlapping bands to be rejected")
	}
}

// A tier list that is not strictly descending is rejected, including the case
// where an unbounded tier is not last.
func TestSelectPerformanceTierRejectsUnorderedBands(t *testing.T) {
	cases := map[string][]PerformanceTier{
		"descending": {
			{ID: "a", MatchMemTotalMax: 4096},
			{ID: "b", MatchMemTotalMax: 2048},
			{ID: "c", MatchMemTotalMax: 0},
		},
		"unbounded not last": {
			{ID: "a", MatchMemTotalMax: 0},
			{ID: "b", MatchMemTotalMax: 4096},
		},
		"duplicated boundary": {
			{ID: "a", MatchMemTotalMax: 2048},
			{ID: "b", MatchMemTotalMax: 2048},
			{ID: "c", MatchMemTotalMax: 0},
		},
		"a second unbounded tier": {
			{ID: "a", MatchMemTotalMax: 2048},
			{ID: "b", MatchMemTotalMax: 0},
			{ID: "c", MatchMemTotalMax: 0},
		},
	}
	for name, tiers := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := SelectPerformanceTier(HardwareState{MemTotalKB: 1024}, tiers); err == nil {
				t.Fatalf("expected %s bands to be rejected", name)
			}
		})
	}
}

// An empty list and an unmeasurable host are both errors. Silently defaulting a
// zero-memory host to the smallest tier would apply desktop-grade settings to a
// machine nobody measured.
func TestSelectPerformanceTierRejectsEmptyListAndUnknownMemory(t *testing.T) {
	if _, err := SelectPerformanceTier(HardwareState{MemTotalKB: 974092}, nil); err == nil {
		t.Fatal("expected an empty tier list to be rejected")
	}
	if _, err := SelectPerformanceTier(HardwareState{MemTotalKB: 974092}, []PerformanceTier{}); err == nil {
		t.Fatal("expected an empty tier list to be rejected")
	}
	tier, err := SelectPerformanceTier(HardwareState{MemTotalKB: 0}, fleetTiers())
	if err == nil {
		t.Fatalf("an unmeasured host selected tier %q; it must be an error", tier.ID)
	}
}

// Every tier must carry a rationale, and the two rows without hardware behind
// them must say so, so nobody later reads a derived band as a measurement.
func TestPerformanceTierRequiresRationale(t *testing.T) {
	if err := ValidatePerformanceTiers([]PerformanceTier{{ID: "tiny", MatchMemTotalMax: 1536}}); err == nil {
		t.Fatal("expected a tier without a rationale to be rejected")
	}
	if err := ValidatePerformanceTiers(fleetTiers()); err != nil {
		t.Fatalf("declared tiers are invalid: %v", err)
	}
}

// EnableZRAM is a pointer so an omitted value is distinguishable from an
// explicit false: a manifest that forgets to say must not silently disable zram.
func TestPerformanceTierZRAMIsTriState(t *testing.T) {
	var unset PerformanceTier
	if unset.ZRAMEnabled() {
		t.Fatal("an unset zram flag must not report as enabled")
	}
	off := false
	if (PerformanceTier{EnableZRAM: &off}).ZRAMEnabled() {
		t.Fatal("an explicit false must report as disabled")
	}
	on := true
	if !(PerformanceTier{EnableZRAM: &on}).ZRAMEnabled() {
		t.Fatal("an explicit true must report as enabled")
	}
}
