package cli

import (
	"strings"
	"testing"

	"github.com/eajdias/envctl/internal/domain/entity"
)

// testProfileMetas mirrors what ManifestRepository.ListPerformanceProfiles
// returns from the two shipped manifests.
func testProfileMetas() []entity.PerformanceProfileMeta {
	return []entity.PerformanceProfileMeta{
		{Profile: entity.PerformanceProfileUbuntuServer, MinDistroVersion: "24.04", ManifestFile: "performance_ubuntu.yaml"},
		{Profile: entity.PerformanceProfileCachyOS, ManifestFile: "performance_cachyos.yaml"},
	}
}

func TestSelectPerformanceProfileUsesManifestMinimum(t *testing.T) {
	cases := []struct {
		name     string
		platform entity.PlatformInfo
		metas    []entity.PerformanceProfileMeta
		want     entity.PerformanceProfile
		wantErr  bool
	}{
		{
			name:     "ubuntu 26.04 from the fleet",
			platform: entity.PlatformInfo{GOOS: "linux", Family: entity.DistroDebian, ID: "ubuntu", VersionID: "26.04"},
			want:     entity.PerformanceProfileUbuntuServer,
		},
		{
			name:     "ubuntu 24.04 at the declared minimum",
			platform: entity.PlatformInfo{GOOS: "linux", Family: entity.DistroDebian, ID: "ubuntu", VersionID: "24.04"},
			want:     entity.PerformanceProfileUbuntuServer,
		},
		{
			name:     "cachyos rolling has no floor and matches on ID",
			platform: entity.PlatformInfo{GOOS: "linux", Family: entity.DistroArch, ID: "cachyos", VersionID: "rolling"},
			want:     entity.PerformanceProfileCachyOS,
		},
		{
			name:     "ubuntu 22.04 is below the manifest minimum",
			platform: entity.PlatformInfo{GOOS: "linux", Family: entity.DistroDebian, ID: "ubuntu", VersionID: "22.04"},
			wantErr:  true,
		},
		{
			name:     "debian 12 is a different ID",
			platform: entity.PlatformInfo{GOOS: "linux", Family: entity.DistroDebian, ID: "debian", VersionID: "12"},
			wantErr:  true,
		},
		{
			name:     "generic arch is a different ID",
			platform: entity.PlatformInfo{GOOS: "linux", Family: entity.DistroArch, ID: "arch", VersionID: "rolling"},
			wantErr:  true,
		},
		{
			name:     "a manifest without a minimum accepts the host version",
			platform: entity.PlatformInfo{GOOS: "linux", Family: entity.DistroDebian, ID: "ubuntu", VersionID: "22.04"},
			metas:    []entity.PerformanceProfileMeta{{Profile: entity.PerformanceProfileUbuntuServer}},
			want:     entity.PerformanceProfileUbuntuServer,
		},
		{
			name:     "no manifests available",
			platform: entity.PlatformInfo{GOOS: "linux", Family: entity.DistroDebian, ID: "ubuntu", VersionID: "24.04"},
			metas:    []entity.PerformanceProfileMeta{},
			wantErr:  true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			metas := tc.metas
			if metas == nil {
				metas = testProfileMetas()
			}
			_ = metas
			got, err := selectPerformanceProfile(tc.platform, metas)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected profile selection to fail")
				}
				// The error must name the host and the floor, otherwise an
				// unsupported host is unactionable.
				if !strings.Contains(err.Error(), tc.platform.ID) {
					t.Fatalf("error %q does not name the host id %q", err, tc.platform.ID)
				}
				return
			}
			if err != nil {
				t.Fatalf("selectPerformanceProfile failed: %v", err)
			}
			if got != tc.want {
				t.Errorf("profile = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestSelectPerformanceProfileErrorNamesTheMinimum keeps the failure message
// actionable: an operator on Ubuntu 22.04 must learn the required floor.
func TestSelectPerformanceProfileErrorNamesTheMinimum(t *testing.T) {
	_, err := selectPerformanceProfile(
		entity.PlatformInfo{GOOS: "linux", Family: entity.DistroDebian, ID: "ubuntu", VersionID: "22.04"},
		testProfileMetas(),
	)
	if err == nil {
		t.Fatal("expected an error for Ubuntu 22.04")
	}
	if !strings.Contains(err.Error(), "24.04") {
		t.Fatalf("error %q does not state the required minimum 24.04", err)
	}
}
