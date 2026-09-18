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
		{"darwin scoping", "darwin", "darwin", DistroDarwin, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := MatchOS(tc.filter, tc.goos, tc.distro); got != tc.want {
				t.Errorf("MatchOS(%q, %q, %q) = %v, want %v", tc.filter, tc.goos, tc.distro, got, tc.want)
			}
		})
	}
}

func TestDetectedDistroIsAKnownValue(t *testing.T) {
	switch got := DetectedDistro(); got {
	case DistroArch, DistroDebian, DistroWindows, DistroDarwin, DistroUnknown:
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
