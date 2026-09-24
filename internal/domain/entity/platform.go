package entity

import (
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

// Distro families used by the `os:` manifest filter and the provisioning code.
const (
	DistroArch    = "arch"
	DistroDebian  = "debian"
	DistroWindows = "windows"
	DistroDarwin  = "darwin"
	DistroUnknown = ""
)

// PlatformInfo contains both the broad family used by existing manifests and
// the exact distro identity/version read from /etc/os-release. The latter is
// required for profiles that must not bleed across Ubuntu, Debian, and Arch
// derivatives.
type PlatformInfo struct {
	GOOS      string
	Family    string
	ID        string
	VersionID string
	IDLike    string
}

var platformOnce sync.Once
var platformCached PlatformInfo

// DetectedPlatform resolves the running platform and caches the result.
func DetectedPlatform() PlatformInfo {
	platformOnce.Do(func() {
		platformCached = detectPlatform()
	})
	return platformCached
}

// DetectedDistro resolves the running platform to a provisioning family.
func DetectedDistro() string {
	return DetectedPlatform().Family
}

// DetectedDistroID returns the exact ID from /etc/os-release on Linux.
func DetectedDistroID() string {
	return DetectedPlatform().ID
}

// DetectedDistroVersion returns VERSION_ID from /etc/os-release on Linux.
func DetectedDistroVersion() string {
	return DetectedPlatform().VersionID
}

func detectPlatform() PlatformInfo {
	if runtime.GOOS != "linux" {
		return PlatformInfo{GOOS: runtime.GOOS, Family: runtime.GOOS}
	}

	id, idLike, version := readOSRelease()
	return PlatformInfo{
		GOOS:      "linux",
		Family:    distroFamily(id, idLike),
		ID:        id,
		VersionID: version,
		IDLike:    idLike,
	}
}

func distroFamily(id, idLike string) string {
	candidates := append([]string{id}, strings.Fields(idLike)...)
	for _, candidate := range candidates {
		switch candidate {
		case "arch", "archlinux", "cachyos", "endeavouros", "manjaro", "garuda", "artix":
			return DistroArch
		case "debian", "ubuntu", "linuxmint", "pop", "raspbian", "kali", "zorin":
			return DistroDebian
		}
	}
	return DistroUnknown
}

// readOSRelease extracts ID, ID_LIKE, and VERSION_ID from /etc/os-release.
func readOSRelease() (id, idLike, version string) {
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return "", "", ""
	}
	return parseOSRelease(string(data))
}

func parseOSRelease(data string) (id, idLike, version string) {
	for _, line := range strings.Split(data, "\n") {
		key, value, ok := strings.Cut(strings.TrimSpace(line), "=")
		if !ok {
			continue
		}
		value = strings.ToLower(strings.Trim(strings.TrimSpace(value), `"'`))
		switch key {
		case "ID":
			id = value
		case "ID_LIKE":
			idLike = value
		case "VERSION_ID":
			version = value
		}
	}
	return id, idLike, version
}

// MatchOS reports whether an `os:` manifest filter applies to a platform.
//
// Accepted values: "" (portable — every platform), the Go platform names
// ("windows", "linux", "darwin"), and the Linux distro families ("arch",
// "cachyos", "debian", "ubuntu"). A distro family only matches when the host
// actually runs that family, so `os: arch` is skipped on Debian and vice
// versa. Comma/space separated lists are accepted too ("arch,cachyos").
func MatchOS(filter, goos, distro string) bool {
	if filter == "" {
		return true
	}
	for _, token := range strings.FieldsFunc(strings.ToLower(filter), func(r rune) bool {
		return r == ',' || r == ' ' || r == '|'
	}) {
		if token == strings.ToLower(goos) {
			return true
		}
		if goos != "linux" {
			continue
		}
		switch token {
		case "arch", "archlinux", "cachyos":
			if distro == DistroArch {
				return true
			}
		case "debian", "ubuntu":
			if distro == DistroDebian {
				return true
			}
		}
	}
	return false
}

// PackageMatchesPlatform applies the legacy OS-family filter plus optional
// exact distro and minimum VERSION_ID constraints. It is pure so package
// selection can be tested without depending on the host running the tests.
func PackageMatchesPlatform(pkg Package, platform PlatformInfo) bool {
	if !MatchOS(pkg.OS, platform.GOOS, platform.Family) {
		return false
	}
	if pkg.TargetDistro != "" && !strings.EqualFold(strings.TrimSpace(pkg.TargetDistro), platform.ID) {
		return false
	}
	if pkg.MinDistroVersion != "" && compareDistroVersions(platform.VersionID, pkg.MinDistroVersion) < 0 {
		return false
	}
	return true
}

// MatchesPackage applies PackageMatchesPlatform to the current host.
func MatchesPackage(pkg Package) bool {
	return PackageMatchesPlatform(pkg, DetectedPlatform())
}

func compareDistroVersions(current, minimum string) int {
	currentParts, currentOK := parseDistroVersion(current)
	minimumParts, minimumOK := parseDistroVersion(minimum)
	if !currentOK || !minimumOK {
		return -1
	}

	max := len(currentParts)
	if len(minimumParts) > max {
		max = len(minimumParts)
	}
	for i := 0; i < max; i++ {
		var currentPart, minimumPart int
		if i < len(currentParts) {
			currentPart = currentParts[i]
		}
		if i < len(minimumParts) {
			minimumPart = minimumParts[i]
		}
		if currentPart < minimumPart {
			return -1
		}
		if currentPart > minimumPart {
			return 1
		}
	}
	return 0
}

func parseDistroVersion(version string) ([]int, bool) {
	version = strings.TrimSpace(version)
	if version == "" {
		return nil, false
	}

	parts := strings.Split(version, ".")
	values := make([]int, 0, len(parts))
	for _, part := range parts {
		end := 0
		for end < len(part) && part[end] >= '0' && part[end] <= '9' {
			end++
		}
		if end == 0 {
			break
		}
		value, err := strconv.Atoi(part[:end])
		if err != nil {
			return nil, false
		}
		values = append(values, value)
	}
	return values, len(values) > 0
}

// MatchesOS reports whether the filter applies to the running platform.
func MatchesOS(filter string) bool {
	platform := DetectedPlatform()
	return MatchOS(filter, platform.GOOS, platform.Family)
}
