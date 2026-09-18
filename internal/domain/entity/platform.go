package entity

import (
	"os"
	"runtime"
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

var (
	distroOnce   sync.Once
	distroCached string
)

// DetectedDistro resolves the running platform to a provisioning family.
//
// Linux distros are classified through /etc/os-release (ID + ID_LIKE) so that
// Arch-based hosts (Arch, CachyOS, EndeavourOS, Manjaro, Garuda) can be told
// apart from Debian-based ones (Ubuntu, Debian, Mint) without probing for a
// package manager at every call site. Windows and macOS report themselves,
// and an unrecognized Linux host reports DistroUnknown — callers then fall
// back to plain `linux` matching.
func DetectedDistro() string {
	distroOnce.Do(func() {
		distroCached = detectDistro()
	})
	return distroCached
}

func detectDistro() string {
	if runtime.GOOS != "linux" {
		return runtime.GOOS
	}
	id, idLike := readOSRelease()
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

// readOSRelease extracts ID and ID_LIKE from /etc/os-release.
func readOSRelease() (id, idLike string) {
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return "", ""
	}
	for _, line := range strings.Split(string(data), "\n") {
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
		}
	}
	return id, idLike
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

// MatchesOS reports whether the filter applies to the running platform.
func MatchesOS(filter string) bool {
	return MatchOS(filter, runtime.GOOS, DetectedDistro())
}
