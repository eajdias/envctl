package entity

import (
	"path/filepath"
	"strconv"
	"strings"
)

// PerformanceProfile identifies an OS-specific performance manifest. Profiles
// are intentionally separate so Ubuntu server tuning can never be applied to
// CachyOS, or vice versa. The identity never encodes a version: the fleet
// already runs Ubuntu 26.04, so the release floor lives in the manifest as
// min_distro_version and is evaluated with MatchesDistroMinimum.
type PerformanceProfile string

const (
	PerformanceProfileUbuntuServer PerformanceProfile = "ubuntu-server"
	PerformanceProfileCachyOS      PerformanceProfile = "cachyos"
)

// PerformanceProfileMeta lets a caller discover a profile and its release
// floor without loading the whole spec, so no caller hard-codes a version.
type PerformanceProfileMeta struct {
	Profile          PerformanceProfile `yaml:"profile"`
	MinDistroVersion string             `yaml:"min_distro_version,omitempty"`
	ManifestFile     string             `yaml:"-"`
}

// SysctlPolicy decides how a declared value interacts with the host's current
// value. It exists because a manifest states a direction, not a replacement:
// writing over a value the host already sets better is a regression.
type SysctlPolicy string

const (
	// SysctlPolicySet always writes the declared value.
	SysctlPolicySet SysctlPolicy = ""
	// SysctlPolicyMin writes only when the host's effective value is lower, so
	// the tool can raise a ceiling but never lower one.
	SysctlPolicyMin SysctlPolicy = "min"
	// SysctlPolicyMax writes only when the host's effective value is higher.
	SysctlPolicyMax SysctlPolicy = "max"
)

// SysctlSetting is one explicit, reviewable sysctl value in a performance
// profile. The rationale is carried into CLI/doctor output and documentation.
type SysctlSetting struct {
	Key       string `yaml:"key"`
	Value     string `yaml:"value"`
	Rationale string `yaml:"rationale"`
	// Policy defaults to SysctlPolicySet. A min or max policy makes the
	// declared value a bound rather than a replacement.
	Policy SysctlPolicy `yaml:"policy,omitempty"`
}

// PerformanceSpec is the complete declarative payload for one OS profile.
type PerformanceSpec struct {
	Profile PerformanceProfile `yaml:"profile"`
	// MinDistroVersion is the release floor declared by the manifest. It is
	// data, not code, so raising the floor is a manifest edit rather than a
	// code change.
	MinDistroVersion string          `yaml:"min_distro_version,omitempty"`
	Packages         []Package       `yaml:"packages"`
	Sysctls          []SysctlSetting `yaml:"sysctls"`
	// Tiers are the memory bands the manifest declares. A host is measured
	// and matched against them; nothing here is inferred from the machine.
	Tiers []PerformanceTier `yaml:"tiers,omitempty"`
	// Timezone is the declared zone policy.
	Timezone *TimezoneSpec `yaml:"timezone,omitempty"`
	// Journald is the declared journal size policy.
	Journald *JournaldSpec `yaml:"journald,omitempty"`
	// Limits is the declared descriptor/process limit policy.
	Limits *LimitsSpec `yaml:"limits,omitempty"`
	// ZRAM is the declared compressed-RAM swap policy.
	ZRAM *ZRAMSpec `yaml:"zram,omitempty"`
	// Swap is the declared disk swapfile policy.
	Swap *SwapSpec `yaml:"swap,omitempty"`
	// Debloat is the declared package removal policy.
	Debloat *DebloatSpec `yaml:"debloat,omitempty"`
}

type SwapDevice struct {
	Name     string
	Type     string
	SizeKB   uint64
	UsedKB   uint64
	Priority int
}

type ZRAMState struct {
	Present    bool
	Name       string
	Algorithm  string
	SizeBytes  uint64
	DataBytes  uint64
	TotalBytes uint64
	Priority   int
}

// NoDiskSwapPriority is the sentinel for "this host has no disk swap". It sits
// below every value swapon(8) can assign, including the -2 default and the -1
// the Oracle images ship, so a caller can compare against it directly.
const NoDiskSwapPriority = -1 << 31

// HardwareState is the detected, read-only description of the host that the
// declarative performance policy resolves against. Nothing here is a declared
// value: everything is measured, so a manifest never has to carry a machine
// number.
type HardwareState struct {
	MemTotalKB    uint64
	CPUCount      int
	RootFSType    string
	DiskFreeBytes uint64
	Swap          []SwapDevice
	HasZRAM       bool
	ZRAMPriority  int
	// DiskSwapTopPri is the highest priority among non-zram swap devices, or
	// NoDiskSwapPriority when there is none.
	DiskSwapTopPri int
}

// NewHardwareState derives the swap topology from a raw swap table. It is pure
// so the zram-first and disk-fallback decisions are unit testable without a
// procfs fixture.
func NewHardwareState(memTotalKB uint64, cpuCount int, rootFSType string, diskFreeBytes uint64, devices []SwapDevice) HardwareState {
	state := HardwareState{
		MemTotalKB:     memTotalKB,
		CPUCount:       cpuCount,
		RootFSType:     rootFSType,
		DiskFreeBytes:  diskFreeBytes,
		Swap:           append([]SwapDevice(nil), devices...),
		DiskSwapTopPri: NoDiskSwapPriority,
	}
	for _, device := range devices {
		if filepath.Base(device.Name) == "zram0" {
			state.HasZRAM = true
			state.ZRAMPriority = device.Priority
			continue
		}
		if !IsDiskSwapDevice(device.Name) {
			continue
		}
		if device.Priority > state.DiskSwapTopPri {
			state.DiskSwapTopPri = device.Priority
		}
	}
	return state
}

// MemTotalMiB converts the kernel's kilobyte report to MiB. Callers must size
// tiers against this and never against a nominal instance size: the fleet
// reports 974092 kB (951 MiB) for a "1 GB" shape and 16162396 kB (15783 MiB)
// for a "16 GB" one.
func (h HardwareState) MemTotalMiB() uint64 {
	return h.MemTotalKB / 1024
}

// HasDiskSwap reports whether a real disk-backed swap device is active, which
// is the signal to adopt rather than create.
func (h HardwareState) HasDiskSwap() bool {
	return h.DiskSwapTopPri != NoDiskSwapPriority
}

// IsDiskSwapDevice reports whether a swap path is disk-backed. Compressed RAM
// devices are not: they are the fast tier, not the fallback.
func IsDiskSwapDevice(name string) bool {
	return !strings.HasPrefix(filepath.Base(name), "zram")
}

type BlockScheduler struct {
	Device    string
	Selected  string
	Available string
}

// LimitsSpec is the declared file-descriptor and process limit policy.
//
// NofileHard empty means "keep the host's own hard limit". The fleet reports
// 524288 on Oracle and 1048576 on AWS, so a pinned hard value would lower the
// AWS hosts.
type LimitsSpec struct {
	SystemDropin string `yaml:"system_dropin"`
	PAMDropin    string `yaml:"pam_dropin"`
	NofileSoft   int    `yaml:"nofile_soft"`
	NofileHard   string `yaml:"nofile_hard"`
	Nproc        int    `yaml:"nproc"`
	// Reexec controls the PID 1 re-exec that makes the systemd drop-in
	// effective. It is the only operation in the profile that touches PID 1, so
	// it must be possible to turn it off from the command line.
	Reexec bool `yaml:"daemon_reexec"`
}

// JournaldSetting is one size or retention key in the journald drop-in.
type JournaldSetting struct {
	Key   string `yaml:"key"`
	Value string `yaml:"value"`
}

// JournaldSpec is the declared journald size policy. SystemKeepFree is part of
// the contract, not an extra: journald honours the smaller of SystemMaxUse and
// SystemKeepFree, so a cap without a floor still lets the journal grow to the
// filesystem default.
type JournaldSpec struct {
	Dropin string            `yaml:"dropin"`
	Values []JournaldSetting `yaml:"values"`
}

// TimezoneSpec declares the timezone policy. Mode defaults to "verify", which
// only reports; writing a timezone changes log timestamps and scheduled jobs,
// so it is never implicit.
type TimezoneSpec struct {
	Mode     string `yaml:"mode"`
	Expected string `yaml:"expected"`
}

type JournaldState struct {
	DiskUsage string
	Storage   string
	// SystemMaxUse and SystemKeepFree are both honored by journald, which
	// applies the smaller of the two. Reading only the first would let a host
	// look capped when it has no floor at all.
	SystemMaxUse   string
	SystemKeepFree string
}

type TimerState struct {
	Name    string
	Enabled string
	Active  string
}

type ServiceState struct {
	Name    string
	Enabled string
	Active  string
}

type PerformanceSnapshot struct {
	Swap            []SwapDevice
	ZRAM            ZRAMState
	CPUGovernors    []string
	BlockSchedulers []BlockScheduler
	Journald        JournaldState
	FSTRIMTimer     TimerState
	Services        []ServiceState
	// Sysctls is the resolved sysctl.d state: one entry per key, naming the
	// file that systemd-sysctl will apply last at the next boot, the value that
	// file sets, and the value the kernel currently holds.
	Sysctls []SysctlAssignment
	// MemoryKB is MemTotal from /proc/meminfo. It is what selects the memory
	// tier, so an audit can resolve the same tier a run would.
	MemoryKB uint64
}

// HardwareState projects a snapshot onto the measured host the tier and the
// derived sysctls are computed from. The inspector observes with procfs and
// sysfs while the provisioning pipeline observes with the hardware probe; both
// end up here, so the decision functions see one shape.
func (s PerformanceSnapshot) HardwareState() HardwareState {
	return NewHardwareState(s.MemoryKB, 0, "", 0, s.Swap)
}

// CompareSysctlValues compares two sysctl values numerically: 1 when a is
// greater, -1 when it is smaller, 0 when they are equal. An unparseable side
// never compares equal, so a malformed manifest cannot look satisfied.
func CompareSysctlValues(a, b string) int {
	av, aErr := strconv.ParseUint(strings.TrimSpace(a), 10, 64)
	bv, bErr := strconv.ParseUint(strings.TrimSpace(b), 10, 64)
	if aErr != nil || bErr != nil {
		return -1
	}
	switch {
	case av > bv:
		return 1
	case av < bv:
		return -1
	default:
		return 0
	}
}

// SysctlSettingSatisfied reports whether a live value already meets a setting's
// policy. A min-policy key is satisfied by anything at or above the declared
// value, a max-policy key by anything at or below, and a set-policy key only by
// an exact match.
func SysctlSettingSatisfied(setting SysctlSetting, live string) bool {
	cmp := CompareSysctlValues(live, setting.Value)
	switch setting.Policy {
	case SysctlPolicyMin:
		return cmp >= 0
	case SysctlPolicyMax:
		return cmp <= 0
	default:
		return strings.TrimSpace(live) == strings.TrimSpace(setting.Value)
	}
}

// SysctlAssignment is one key as systemd-sysctl will resolve it at boot.
//
// A live host can disagree with its own configuration: the profile writes a
// value, another drop-in with a later filename overrides it, and the next boot
// silently restores the other value. Reporting the resolved file, the boot
// value and the live value together is what makes that visible.
type SysctlAssignment struct {
	Key string
	// File is the path of the drop-in that wins the precedence order.
	File string
	// Boot is the value the winning file sets.
	Boot string
	// Live is what /proc/sys currently reports, empty when unreadable.
	Live string
	// Managed reports that File is the drop-in this project writes.
	Managed bool
}

// ResolvePerformanceProfile returns the profile this host is provisioned with.
// The CachyOS profile is matched first because it is the narrower claim: a
// CachyOS host is an Arch-family Linux host, and matching the family alone would
// hand a gaming desktop the server profile.
//
// The default is the Ubuntu server profile, which is the only remaining Linux
// target. A caller that is not on Linux must not reach here: the Windows path
// has no performance profile.
func ResolvePerformanceProfile(platform PlatformInfo) PerformanceProfile {
	if PerformanceProfileMatchesOS(PerformanceProfileCachyOS, platform) {
		return PerformanceProfileCachyOS
	}
	return PerformanceProfileUbuntuServer
}

// PerformanceProfileMatchesOS reports whether the supplied platform is the OS
// this profile targets, ignoring any release floor. The Ubuntu server profile
// requires the exact ubuntu ID inside the Debian family; the CachyOS profile
// requires the exact cachyos ID, never merely the Arch family.
//
// Use PerformanceProfileSatisfiedBy to also apply the release floor. Calling
// this function alone on the Ubuntu profile also accepts 22.04, which is
// exactly the tolerance the manifest floor removed.
func PerformanceProfileMatchesOS(profile PerformanceProfile, platform PlatformInfo) bool {
	switch profile {
	case PerformanceProfileUbuntuServer:
		return platform.GOOS == "linux" && platform.Family == DistroDebian && platform.ID == "ubuntu"
	case PerformanceProfileCachyOS:
		return platform.GOOS == "linux" && platform.Family == DistroArch && platform.ID == "cachyos"
	default:
		return false
	}
}

// PerformanceProfileSatisfiedBy composes the OS identity check with the
// manifest's declared release floor, so no caller has to know the version.
func PerformanceProfileSatisfiedBy(profile PerformanceProfile, platform PlatformInfo, minVersion string) bool {
	return PerformanceProfileMatchesOS(profile, platform) && MatchesDistroMinimum(platform.VersionID, minVersion)
}
