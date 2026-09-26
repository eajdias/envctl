package entity

import "testing"

func TestMatchOS(t *testing.T) {
	cases := []struct {
		name   string
		filter string
		goos   string
		distro string
		want   bool
	}{
		{"empty filter is portable", "", "linux", DistroArch, true},
		{"exact platform", "linux", "linux", DistroUnknown, true},
		{"other platform", "windows", "linux", DistroArch, false},
		{"arch family matches arch host", "arch", "linux", DistroArch, true},
		{"cachyos alias matches arch host", "cachyos", "linux", DistroArch, true},
		{"arch family skipped on debian host", "arch", "linux", DistroDebian, false},
		{"debian family matches debian host", "debian", "linux", DistroDebian, true},
		{"ubuntu alias matches debian host", "ubuntu", "linux", DistroDebian, true},
		{"debian family skipped on arch host", "debian", "linux", DistroArch, false},
		{"distro family never matches windows", "arch", "windows", DistroWindows, false},
		{"comma list matches any member", "arch,cachyos", "linux", DistroArch, true},
		{"comma list with no match", "arch,debian", "linux", DistroUnknown, false},
		{"unknown distro ignores distro families", "arch", "linux", DistroUnknown, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := MatchOS(tc.filter, tc.goos, tc.distro); got != tc.want {
				t.Errorf("MatchOS(%q, %q, %q) = %v, want %v", tc.filter, tc.goos, tc.distro, got, tc.want)
			}
		})
	}
}

func TestParseOSRelease(t *testing.T) {
	id, idLike, version := parseOSRelease("ID=ubuntu\nID_LIKE=debian\nVERSION_ID=\"24.04\"\n")
	if id != "ubuntu" || idLike != "debian" || version != "24.04" {
		t.Fatalf("parseOSRelease = %q, %q, %q", id, idLike, version)
	}
}

func TestDetectedDistroIsAKnownValue(t *testing.T) {
	switch got := DetectedDistro(); got {
	case DistroArch, DistroDebian, DistroWindows, DistroUnknown:
	default:
		t.Errorf("DetectedDistro() returned unexpected value %q", got)
	}
}

func TestSkillAppliesToOS(t *testing.T) {
	portable := Skill{Name: "portable"}
	if !portable.AppliesToOS("linux") || !portable.AppliesToOS("windows") {
		t.Errorf("a skill without an os scope must apply everywhere")
	}

	windowsOnly := Skill{Name: "windows-only", OS: "windows"}
	if windowsOnly.AppliesToOS("linux") {
		t.Errorf("windows-scoped skill must not apply on linux")
	}
	if !windowsOnly.AppliesToOS("windows") {
		t.Errorf("windows-scoped skill must apply on windows")
	}
}

func TestPackageMatchesPlatform(t *testing.T) {
	ubuntuPackage := Package{
		OS:               "ubuntu",
		TargetDistro:     "ubuntu",
		MinDistroVersion: "24.04",
	}
	cases := []struct {
		name     string
		pkg      Package
		platform PlatformInfo
		want     bool
	}{
		{
			name:     "ubuntu 24.04 accepts target",
			pkg:      ubuntuPackage,
			platform: PlatformInfo{GOOS: "linux", Family: DistroDebian, ID: "ubuntu", VersionID: "24.04"},
			want:     true,
		},
		{
			name:     "ubuntu 22.04 rejects minimum",
			pkg:      ubuntuPackage,
			platform: PlatformInfo{GOOS: "linux", Family: DistroDebian, ID: "ubuntu", VersionID: "22.04"},
			want:     false,
		},
		{
			name:     "debian rejects exact ubuntu target",
			pkg:      ubuntuPackage,
			platform: PlatformInfo{GOOS: "linux", Family: DistroDebian, ID: "debian", VersionID: "12"},
			want:     false,
		},
		{
			name:     "cachyos accepts exact target",
			pkg:      Package{OS: "arch", TargetDistro: "cachyos"},
			platform: PlatformInfo{GOOS: "linux", Family: DistroArch, ID: "cachyos", VersionID: "rolling"},
			want:     true,
		},
		{
			name:     "generic arch rejects exact cachyos target",
			pkg:      Package{OS: "arch", TargetDistro: "cachyos"},
			platform: PlatformInfo{GOOS: "linux", Family: DistroArch, ID: "arch", VersionID: "rolling"},
			want:     false,
		},
		{
			name:     "windows rejects linux target",
			pkg:      ubuntuPackage,
			platform: PlatformInfo{GOOS: "windows", Family: DistroWindows, ID: "windows", VersionID: ""},
			want:     false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := PackageMatchesPlatform(tc.pkg, tc.platform); got != tc.want {
				t.Errorf("PackageMatchesPlatform() = %v, want %v", got, tc.want)
			}
		})
	}
}
