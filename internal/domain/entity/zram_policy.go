package entity

import "fmt"

// ZRAMPriority is the priority systemd-zram-generator assigns to the
// compressed-RAM device. It only has to outrank the disk fallback, which the
// swap policy installs at -2. It is exported so the swap policy can detect a
// device that already outranks the compressed tier.
const ZRAMPriority = 100

// ZRAM policy modes.
const (
	// ZRAMPolicyTier runs the lifecycle only when the resolved memory tier asks
	// for it.
	ZRAMPolicyTier = "tier"
	// ZRAMPolicyAlways runs the lifecycle regardless of tier.
	ZRAMPolicyAlways = "always"
	// ZRAMPolicyDisabled never runs it.
	ZRAMPolicyDisabled = "disabled"
)

// ZRAMSpec declares the compressed-RAM swap policy. Unlike swappiness, this is
// a profile-level decision with no runtime input: whether zram belongs on a
// host is a function of how scarce RAM is, which the tier already encodes.
type ZRAMSpec struct {
	Policy string `yaml:"policy"`
}

// DeriveSwappiness returns the vm.swappiness value for a measured host together
// with the reasoning that goes into the diagnostic.
//
// It is a function of the swap topology, never of the RAM tier. The kernel
// documentation (admin-guide/sysctl/vm.html) says: "For in-memory swap, like
// zram or zswap, as well as hybrid setups that have swap on faster devices than
// the filesystem, values beyond 100 can be considered." A host whose swap is
// compressed RAM therefore wants a high value, and a host whose swap is a disk
// wants a low one so the kernel prefers evicting page cache over thrashing.
//
// Pinning this in the manifest is what produced the original defect: the
// profile installed zram-generator and declared 10, telling the kernel not to
// use the device it had just created.
func DeriveSwappiness(hw HardwareState) (int, string) {
	switch {
	case hw.HasZRAM:
		return 150, "zram is active: this is in-memory swap, and the kernel documentation " +
			"recommends values beyond 100 when swap is faster than the filesystem, " +
			"so the compressed tier is actually used"
	case hw.HasDiskSwap():
		return 10, "only disk-backed swap is active: a low value keeps the kernel evicting " +
			"page cache rather than thrashing the disk. This is the configuration the " +
			"Oracle fleet already ships and runs"
	default:
		return 10, "no swap is active yet: the policy installs disk-backed swap, and a low " +
			"value keeps the kernel preferring page-cache eviction over disk thrashing"
	}
}

// DetectZRAMPriorityConflict reports whether a disk-backed swap device already
// outranks the compressed-RAM device. Adding zram exists to be preferred; if a
// disk device already has the higher priority, the ordering is inverted and
// silently rewriting it would mask a hand-tuned decision.
func DetectZRAMPriorityConflict(hw HardwareState, zramPriority int) (bool, string) {
	if !hw.HasDiskSwap() {
		return false, ""
	}
	if hw.DiskSwapTopPri < zramPriority {
		return false, ""
	}
	detail := fmt.Sprintf(
		"disk swap at priority %d already outranks the zram priority %d; zram would never be used",
		hw.DiskSwapTopPri, zramPriority,
	)
	return true, detail
}

// ResolvedZRAMPolicy decides whether the compressed-RAM lifecycle runs for a
// host, and says why.
func ResolvedZRAMPolicy(policy string, tier PerformanceTier, hw HardwareState) (run bool, detail string) {
	switch policy {
	case ZRAMPolicyDisabled:
		return false, "zram is disabled by the profile"
	case ZRAMPolicyAlways:
		return true, "zram is enabled for every tier by the profile"
	case ZRAMPolicyTier, "":
		if !tier.ZRAMEnabled() {
			return false, fmt.Sprintf(
				"tier %q does not use zram: at %d MiB of detected memory, disk swap is cheap relative "+
					"to the RAM a compressor would hold, and the page cache is worth more",
				tier.ID, hw.MemTotalMiB(),
			)
		}
		return true, fmt.Sprintf("tier %q enables zram at %d MiB of detected memory", tier.ID, hw.MemTotalMiB())
	default:
		return false, fmt.Sprintf("unknown zram policy %q", policy)
	}
}
