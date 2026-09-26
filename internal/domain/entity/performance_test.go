package entity

import "testing"

// TestMatchesDistroMinimum pins the release gate that moved out of the Go
// constant and into the manifest. The versionID values are the real
// VERSION_ID values reported by the fleet, not nominal sizes.
func TestMatchesDistroMinimum(t *testing.T) {
	cases := []struct {
		name      string
		versionID string
		minimum   string
		want      bool
	}{
		{"fleet Ubuntu 26.04 meets the 24.04 minimum", "26.04", "24.04", true},
		{"fleet Ubuntu 24.04 meets its own minimum", "24.04", "24.04", true},
		{"newer point release meets the minimum", "24.10", "24.04", true},
		{"legacy Ubuntu 22.04 is below the minimum", "22.04", "24.04", false},
		{"Ubuntu 20.04 is below the minimum", "20.04", "24.04", false},
		{"Debian 12 VERSION_ID is below the minimum", "12", "24.04", false},
		{"empty minimum accepts any version", "18.04", "", true},
		{"rolling version is below a numeric minimum", "rolling", "24.04", false},
		{"unknown version is below a numeric minimum", "", "24.04", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := MatchesDistroMinimum(tc.versionID, tc.minimum); got != tc.want {
				t.Fatalf("MatchesDistroMinimum(%q, %q) = %v, want %v", tc.versionID, tc.minimum, got, tc.want)
			}
		})
	}
}

// TestPerformanceProfileIdentityIsNotAVersion guards the rename: the fleet
// already runs Ubuntu 26.04, so a profile identity that encodes "24.04" is a
// lie. The minimum belongs to the manifest.
func TestPerformanceProfileIdentityIsNotAVersion(t *testing.T) {
	if PerformanceProfileUbuntuServer != PerformanceProfile("ubuntu-server") {
		t.Fatalf("Ubuntu profile identity = %q, want %q", PerformanceProfileUbuntuServer, "ubuntu-server")
	}
	if PerformanceProfileCachyOS != PerformanceProfile("cachyos") {
		t.Fatalf("CachyOS profile identity = %q, want %q", PerformanceProfileCachyOS, "cachyos")
	}
}

// TestPerformanceProfileMetaCarriesTheMinimum keeps the minimum in one struct
// that both the CLI and the use case read, so neither can hard-code a version.
func TestPerformanceProfileMetaCarriesTheMinimum(t *testing.T) {
	meta := PerformanceProfileMeta{
		Profile:          PerformanceProfileUbuntuServer,
		MinDistroVersion: "24.04",
		ManifestFile:     "performance_ubuntu.yaml",
	}
	if !MatchesDistroMinimum("26.04", meta.MinDistroVersion) {
		t.Fatalf("meta %q with minimum %q rejected the fleet's 26.04 host", meta.Profile, meta.MinDistroVersion)
	}
	if meta.ManifestFile != "performance_ubuntu.yaml" {
		t.Fatalf("meta manifest file = %q", meta.ManifestFile)
	}
}
