package entity

// PerformanceProfile identifies an OS-specific performance manifest. Profiles
// are intentionally separate so Ubuntu server tuning can never be applied to
// CachyOS, or vice versa.
type PerformanceProfile string

const (
	PerformanceProfileUbuntu  PerformanceProfile = "ubuntu-24.04"
	PerformanceProfileCachyOS PerformanceProfile = "cachyos"
)

// SysctlSetting is one explicit, reviewable sysctl value in a performance
// profile. The rationale is carried into CLI/doctor output and documentation.
type SysctlSetting struct {
	Key       string `yaml:"key"`
	Value     string `yaml:"value"`
	Rationale string `yaml:"rationale"`
}

// PerformanceSpec is the complete declarative payload for one OS profile.
type PerformanceSpec struct {
	Profile  PerformanceProfile
	Packages []Package
	Sysctls  []SysctlSetting
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

type BlockScheduler struct {
	Device    string
	Selected  string
	Available string
}

type JournaldState struct {
	DiskUsage    string
	Storage      string
	SystemMaxUse string
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
}

// PerformanceProfileMatchesPlatform reports whether an exact profile is
// supported by the supplied platform. Ubuntu requires VERSION_ID >= 24.04;
// CachyOS requires the exact cachyos ID, not merely the Arch family.
func PerformanceProfileMatchesPlatform(profile PerformanceProfile, platform PlatformInfo) bool {
	switch profile {
	case PerformanceProfileUbuntu:
		return platform.GOOS == "linux" && platform.Family == DistroDebian && platform.ID == "ubuntu" && compareDistroVersions(platform.VersionID, "24.04") >= 0
	case PerformanceProfileCachyOS:
		return platform.GOOS == "linux" && platform.Family == DistroArch && platform.ID == "cachyos"
	default:
		return false
	}
}
