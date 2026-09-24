package cli

import (
	"testing"

	"github.com/eajdias/envctl/internal/domain/entity"
)

func TestSelectPerformanceProfileRequiresExactOS(t *testing.T) {
	cases := []struct {
		name     string
		platform entity.PlatformInfo
		want     entity.PerformanceProfile
		wantErr  bool
	}{
		{
			name:     "ubuntu 24.04",
			platform: entity.PlatformInfo{GOOS: "linux", Family: entity.DistroDebian, ID: "ubuntu", VersionID: "24.04"},
			want:     entity.PerformanceProfileUbuntu,
		},
		{
			name:     "ubuntu 26.04",
			platform: entity.PlatformInfo{GOOS: "linux", Family: entity.DistroDebian, ID: "ubuntu", VersionID: "26.04"},
			want:     entity.PerformanceProfileUbuntu,
		},
		{
			name:     "cachyos",
			platform: entity.PlatformInfo{GOOS: "linux", Family: entity.DistroArch, ID: "cachyos", VersionID: "rolling"},
			want:     entity.PerformanceProfileCachyOS,
		},
		{
			name:     "ubuntu 22.04",
			platform: entity.PlatformInfo{GOOS: "linux", Family: entity.DistroDebian, ID: "ubuntu", VersionID: "22.04"},
			wantErr:  true,
		},
		{
			name:     "generic arch",
			platform: entity.PlatformInfo{GOOS: "linux", Family: entity.DistroArch, ID: "arch", VersionID: "rolling"},
			wantErr:  true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := selectPerformanceProfile(tc.platform)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected profile selection to fail")
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
