package entity

import "fmt"

// PerformanceTier is one band of detected memory. The manifest declares the
// bands; the host is measured; the code picks the band. Nothing here carries a
// machine-specific number that the manifest did not choose deliberately.
type PerformanceTier struct {
	ID string `yaml:"id"`
	// MatchMemTotalMax is the inclusive upper bound in MiB of the kernel's
	// MemTotal report. Zero means unbounded and must be the last entry.
	MatchMemTotalMax int `yaml:"mem_total_max_mib,omitempty"`
	// Sysctls are merged over the profile's unconditional sysctls for hosts in
	// this band.
	Sysctls []SysctlSetting `yaml:"sysctls,omitempty"`
	// EnableZRAM is a pointer so an omitted value stays distinguishable from an
	// explicit false: a manifest that forgets to declare it must not silently
	// disable zram on a memory-constrained host.
	EnableZRAM *bool `yaml:"zram,omitempty"`
	// SwapSizeMax optionally overrides the profile's swapfile ceiling.
	SwapSizeMax string `yaml:"swap_size_max,omitempty"`
	// Rationale is required and must state whether the band is validated by
	// hardware or derived from reasoning.
	Rationale string `yaml:"rationale"`
}

// ZRAMEnabled reports whether the tier asks for compressed-RAM swap.
func (t PerformanceTier) ZRAMEnabled() bool {
	return t.EnableZRAM != nil && *t.EnableZRAM
}

// Matches reports whether a measured MiB total falls in this band.
func (t PerformanceTier) Matches(memTotalMiB uint64) bool {
	if t.MatchMemTotalMax <= 0 {
		return true
	}
	return memTotalMiB <= uint64(t.MatchMemTotalMax)
}

// ValidatePerformanceTiers rejects a tier list that could resolve
// ambiguously or that omits the reason a band exists.
func ValidatePerformanceTiers(tiers []PerformanceTier) error {
	if len(tiers) == 0 {
		return fmt.Errorf("at least one performance tier is required")
	}
	var previous PerformanceTier
	for i, tier := range tiers {
		if tier.ID == "" {
			return fmt.Errorf("performance tier %d has no id", i)
		}
		if tier.Rationale == "" {
			return fmt.Errorf("performance tier %q has no rationale; state whether it is hardware-validated or derived", tier.ID)
		}
		if i > 0 {
			switch {
			case previous.MatchMemTotalMax <= 0:
				return fmt.Errorf("performance tier %q is unbounded and must be the last entry, but %q follows it", previous.ID, tier.ID)
			case tier.MatchMemTotalMax <= 0:
				// An unbounded final tier is correct.
			case tier.MatchMemTotalMax <= previous.MatchMemTotalMax:
				// The list runs from the narrowest band to the widest, so each
				// boundary must be strictly greater than the previous one. A
				// repeated or smaller boundary makes two bands overlap and the
				// first match would silently win.
				return fmt.Errorf(
					"performance tier %q (%d MiB) must be strictly above %q (%d MiB); overlapping or descending bands resolve ambiguously",
					tier.ID, tier.MatchMemTotalMax, previous.ID, previous.MatchMemTotalMax,
				)
			}
		}
		previous = tier
	}
	return nil
}

// SelectPerformanceTier resolves the band for a measured host. The bands are
// validated first, so an ambiguous manifest is reported instead of silently
// resolving to the first match.
func SelectPerformanceTier(hw HardwareState, tiers []PerformanceTier) (PerformanceTier, error) {
	if err := ValidatePerformanceTiers(tiers); err != nil {
		return PerformanceTier{}, err
	}
	if hw.MemTotalKB == 0 {
		return PerformanceTier{}, fmt.Errorf(
			"cannot select a performance tier: MemTotal is unavailable, so the host memory is unknown",
		)
	}
	memTotalMiB := hw.MemTotalMiB()
	for _, tier := range tiers {
		if tier.Matches(memTotalMiB) {
			return tier, nil
		}
	}
	// Only reachable when the final band is bounded, which Validate rejects.
	return PerformanceTier{}, fmt.Errorf(
		"no performance tier matches %d MiB of detected memory", memTotalMiB,
	)
}
