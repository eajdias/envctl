package entity

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

// SysctlSetting is one explicit, reviewable sysctl value in a performance
// profile. The rationale is carried into CLI/doctor output and documentation.
type SysctlSetting struct {
	Key       string `yaml:"key"`
	Value     string `yaml:"value"`
	Rationale string `yaml:"rationale"`
}

// PerformanceSpec is the complete declarative payload for one OS profile.
type PerformanceSpec struct {
	Profile PerformanceProfile `yaml:"profile"`
	// MinDistroVersion is the release floor declared by the manifest. It is
	// data, not code, so raising the floor is a manifest edit rather than a
	// code change.
	MinDistroVersion string    `yaml:"min_distro_version,omitempty"`
	Packages         []Package `yaml:"packages"`
	Sysctls          []SysctlSetting
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
