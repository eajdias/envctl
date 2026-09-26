package entity

import (
	"fmt"
	"strconv"
	"strings"
)

// Swap policy modes.
const (
	// SwapPolicyAuto adopts an existing swapfile or creates one.
	SwapPolicyAuto = "auto"
	// SwapPolicyDisabled leaves the host's swap exactly as it is.
	SwapPolicyDisabled = "disabled"
)

// Swap size bases.
const (
	// SwapSizeOfMemTotal sizes the swapfile from the measured MemTotal.
	SwapSizeOfMemTotal = "mem_total"
)

// SwapFilesystemDecision is what the policy decided to do about the root
// filesystem before anything is written.
type SwapFilesystemDecision string

const (
	// SwapFilesystemCreate allows the standard fallocate/mkswap sequence.
	SwapFilesystemCreate SwapFilesystemDecision = "create"
	// SwapFilesystemCreateBtrfs allows the btrfs-specific sequence.
	SwapFilesystemCreateBtrfs SwapFilesystemDecision = "create-btrfs"
	// SwapFilesystemRefuse writes nothing.
	SwapFilesystemRefuse SwapFilesystemDecision = "refuse"
)

// SwapSpec is the declared swapfile policy.
//
// The file path is deliberately NOT the conventional "/swapfile": both Oracle
// hosts already ship a hand-created swapfile at that path, and reusing it would
// make the tool's own state indistinguishable from the operator's.
type SwapSpec struct {
	Policy   string `yaml:"policy"`
	File     string `yaml:"file"`
	Priority int    `yaml:"priority"`
	SizeOf   string `yaml:"size_of"`
	SizeMin  string `yaml:"size_min"`
	SizeMax  string `yaml:"size_max"`
	// DiskReserve is the free space left untouched on the root filesystem, so a
	// swapfile can never consume the disk the host needs.
	DiskReserve string   `yaml:"disk_reserve"`
	FSAllow     []string `yaml:"fs_allow"`
	FSBtrfs     string   `yaml:"fs_btrfs"`
	FSDeny      []string `yaml:"fs_deny"`
}

// Enabled reports whether the policy acts at all.
func (s SwapSpec) Enabled() bool {
	return s.Policy != SwapPolicyDisabled
}

// ValidateSwapSpec rejects a policy that could write outside the declared
// contract.
func ValidateSwapSpec(spec SwapSpec) error {
	switch spec.Policy {
	case SwapPolicyAuto, SwapPolicyDisabled:
	case "":
		return fmt.Errorf("swap policy is required")
	default:
		return fmt.Errorf("unknown swap policy %q; use %q or %q", spec.Policy, SwapPolicyAuto, SwapPolicyDisabled)
	}
	if spec.Policy == SwapPolicyDisabled {
		return nil
	}
	if strings.TrimSpace(spec.File) == "" {
		return fmt.Errorf("swap file path is required")
	}
	if !strings.HasPrefix(spec.File, "/") || strings.Contains(spec.File, "..") {
		return fmt.Errorf("swap file %q must be an absolute path without traversal", spec.File)
	}
	if spec.SizeOf != SwapSizeOfMemTotal {
		return fmt.Errorf("unknown swap size base %q; use %q", spec.SizeOf, SwapSizeOfMemTotal)
	}
	minSize, err := ParseByteSize(spec.SizeMin)
	if err != nil {
		return fmt.Errorf("size_min: %w", err)
	}
	maxSize, err := ParseByteSize(spec.SizeMax)
	if err != nil {
		return fmt.Errorf("size_max: %w", err)
	}
	if minSize == 0 || maxSize == 0 {
		return fmt.Errorf("size_min and size_max must both be positive")
	}
	if minSize > maxSize {
		return fmt.Errorf("size_min (%s) must not exceed size_max (%s)", spec.SizeMin, spec.SizeMax)
	}
	if _, err := ParseByteSize(spec.DiskReserve); err != nil {
		return fmt.Errorf("disk_reserve: %w", err)
	}
	return nil
}

// ResolveSwapFilesystem decides whether a swapfile may be created on the given
// root filesystem.
//
// btrfs gets its own decision because the btrfs documentation lists real
// constraints (single device, single data profile, NODATACOW, preallocated) and
// calls a root-filesystem swapfile "especially discouraged" because it makes
// balance and scrub skip block groups. Anything outside the allowlist is refused
// rather than assumed to work.
func ResolveSwapFilesystem(spec SwapSpec, fsType string) SwapFilesystemDecision {
	fsType = strings.ToLower(strings.TrimSpace(fsType))
	for _, denied := range spec.FSDeny {
		if strings.EqualFold(denied, fsType) {
			return SwapFilesystemRefuse
		}
	}
	if fsType == "btrfs" {
		if strings.EqualFold(spec.FSBtrfs, "refuse") {
			return SwapFilesystemRefuse
		}
		return SwapFilesystemCreateBtrfs
	}
	for _, allowed := range spec.FSAllow {
		if strings.EqualFold(allowed, fsType) {
			return SwapFilesystemCreate
		}
	}
	return SwapFilesystemRefuse
}

// ResolveSwapSizeBytes computes the swapfile size for a measured host.
//
// MemTotal is the base, not the nominal instance size, and the clamps are the
// actual safety: the 951 MiB Oracle host is raised to the 1 GB floor while the
// 15783 MiB AWS host is pulled down to the 8 GB ceiling. A host whose free space
// cannot cover the file plus the declared reserve is refused rather than
// truncated into something surprising.
func ResolveSwapSizeBytes(spec SwapSpec, hw HardwareState) (uint64, error) {
	if hw.MemTotalKB == 0 {
		return 0, fmt.Errorf("cannot size a swapfile: MemTotal is unavailable")
	}
	if hw.DiskFreeBytes == 0 {
		return 0, fmt.Errorf("cannot size a swapfile: free space on the root filesystem is unavailable")
	}
	minSize, err := ParseByteSize(spec.SizeMin)
	if err != nil {
		return 0, fmt.Errorf("size_min: %w", err)
	}
	maxSize, err := ParseByteSize(spec.SizeMax)
	if err != nil {
		return 0, fmt.Errorf("size_max: %w", err)
	}
	reserve, err := ParseByteSize(spec.DiskReserve)
	if err != nil {
		return 0, fmt.Errorf("disk_reserve: %w", err)
	}

	usable := hw.MemTotalKB * 1024
	if usable < minSize {
		usable = minSize
	}
	if usable > maxSize {
		usable = maxSize
	}
	if hw.DiskFreeBytes < usable+reserve {
		return 0, fmt.Errorf(
			"declining to create a %s swapfile: the root filesystem has %s free, which does not cover the file plus the %s reserve",
			FormatBytes(usable), FormatBytes(hw.DiskFreeBytes), FormatBytes(reserve),
		)
	}
	return usable, nil
}

// AdoptedDiskSwap reports an existing disk-backed swap device so it can be left
// untouched. A zram device is not adoptable: it is the fast tier, not a
// fallback, and the tool does not own it.
func (h HardwareState) AdoptedDiskSwap() (adopted bool, name string, sizeBytes uint64) {
	for _, device := range h.Swap {
		if !IsDiskSwapDevice(device.Name) {
			continue
		}
		return true, device.Name, device.SizeKB * 1024
	}
	return false, "", 0
}

// ParseByteSize parses a decimal size with an optional K/M/G/T suffix. Decimal
// units are deliberate: the manifest is read by operators, and swapfile sizes
// and disk reserves are conventionally written in decimal.
func ParseByteSize(value string) (uint64, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0, nil
	}
	multiplier := uint64(1)
	switch trimmed[len(trimmed)-1] {
	case 'K', 'k':
		multiplier = 1_000
	case 'M', 'm':
		multiplier = 1_000_000
	case 'G', 'g':
		multiplier = 1_000_000_000
	case 'T', 't':
		multiplier = 1_000_000_000_000
	}
	number := trimmed
	if multiplier > 1 {
		number = trimmed[:len(trimmed)-1]
	}
	parsed, err := strconv.ParseFloat(strings.TrimSpace(number), 64)
	if err != nil {
		return 0, fmt.Errorf("%q is not a byte size", value)
	}
	if parsed < 0 {
		return 0, fmt.Errorf("%q is negative", value)
	}
	return uint64(parsed * float64(multiplier)), nil
}

// FormatBytes renders a byte count for a diagnostic.
func FormatBytes(bytes uint64) string {
	switch {
	case bytes >= 1_000_000_000_000:
		return fmt.Sprintf("%.1fT", float64(bytes)/1_000_000_000_000)
	case bytes >= 1_000_000_000:
		return fmt.Sprintf("%.1fG", float64(bytes)/1_000_000_000)
	case bytes >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(bytes)/1_000_000)
	case bytes >= 1_000:
		return fmt.Sprintf("%.1fK", float64(bytes)/1_000)
	default:
		return fmt.Sprintf("%dB", bytes)
	}
}
