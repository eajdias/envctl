package entity

import (
	"strings"
	"testing"
)

func swapSpecFixture() SwapSpec {
	return SwapSpec{
		Policy:      "auto",
		File:        "/swapfile.envctl",
		Priority:    -2,
		SizeOf:      "mem_total",
		SizeMin:     "1G",
		SizeMax:     "8G",
		DiskReserve: "5G",
		FSAllow:     []string{"ext4", "xfs"},
		FSBtrfs:     "mkswapfile",
		FSDeny:      []string{"zfs", "overlay", "tmpfs"},
	}
}

func TestParseByteSize(t *testing.T) {
	cases := map[string]uint64{
		"1G":    1_000_000_000,
		"8G":    8_000_000_000,
		"5G":    5_000_000_000,
		"512M":  512_000_000,
		"1024":  1024,
		"1.5G":  1_500_000_000,
		"":      0,
		"junk":  0,
		"12XiB": 0,
	}
	for input, want := range cases {
		got, err := ParseByteSize(input)
		if input == "junk" || input == "12XiB" {
			if err == nil {
				t.Fatalf("ParseByteSize(%q) should have failed", input)
			}
			continue
		}
		if err != nil {
			t.Fatalf("ParseByteSize(%q) failed: %v", input, err)
		}
		if got != want {
			t.Fatalf("ParseByteSize(%q) = %d, want %d", input, got, want)
		}
	}
}

// The two measured hosts pin the clamps: the 951 MiB Oracle box is pushed up to
// the 1 GB floor, and the 15783 MiB AWS box is pulled down to the 8 GB ceiling.
func TestResolveSwapSizeBytes(t *testing.T) {
	cases := []struct {
		name    string
		host    string
		spec    SwapSpec
		hw      HardwareState
		want    uint64
		wantErr bool
	}{
		{
			name: "oracle 1 GB is raised to the floor",
			host: "vps_oracle_2 / vps_oracle_1",
			spec: swapSpecFixture(),
			hw:   NewHardwareState(974092, 2, "ext4", 34_000_000_000, nil),
			want: 1_000_000_000,
		},
		{
			// MemTotal is in KiB, so a 4 GiB host reports 4194304 kB. Writing
			// 4194304*1024 here would describe a 4 TiB machine.
			name: "a 4 GiB host uses its own memory",
			host: "synthetic 4 GB, no host in the fleet",
			spec: swapSpecFixture(),
			hw:   NewHardwareState(4194304, 2, "ext4", 34_000_000_000, nil),
			want: 4_294_967_296,
		},
		{
			name: "aws 16 GB is pulled down to the ceiling",
			host: "zscan_chatbot",
			spec: swapSpecFixture(),
			hw:   NewHardwareState(16162396, 4, "ext4", 295_000_000_000, nil),
			want: 8_000_000_000,
		},
		{
			name:    "a disk too small for the reserve is refused",
			host:    "hypothetical 8 GB disk",
			spec:    swapSpecFixture(),
			hw:      NewHardwareState(974092, 2, "ext4", 3_000_000_000, nil),
			wantErr: true,
		},
		{
			name:    "an unmeasured host is refused",
			host:    "fixture",
			spec:    swapSpecFixture(),
			hw:      NewHardwareState(0, 2, "ext4", 34_000_000_000, nil),
			wantErr: true,
		},
		{
			name:    "an unmeasurable free space is refused",
			host:    "fixture",
			spec:    swapSpecFixture(),
			hw:      NewHardwareState(974092, 2, "ext4", 0, nil),
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ResolveSwapSizeBytes(tc.spec, tc.hw)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected a refusal, got %d (host %s)", got, tc.host)
				}
				return
			}
			if err != nil {
				t.Fatalf("ResolveSwapSizeBytes failed: %v (host %s)", err, tc.host)
			}
			if got != tc.want {
				t.Fatalf("size = %d, want %d (host %s)", got, tc.want, tc.host)
			}
		})
	}
}

// The filesystem policy is the safety gate for swapfile creation. btrfs needs
// its own command sequence; anything not on the allowlist is refused outright
// rather than assumed to work.
func TestSwapFilesystemPolicy(t *testing.T) {
	spec := swapSpecFixture()
	cases := []struct {
		fs       string
		decision SwapFilesystemDecision
	}{
		{"ext4", SwapFilesystemCreate},
		{"xfs", SwapFilesystemCreate},
		{"btrfs", SwapFilesystemCreateBtrfs},
		{"zfs", SwapFilesystemRefuse},
		{"overlay", SwapFilesystemRefuse},
		{"tmpfs", SwapFilesystemRefuse},
		{"unknown-deadbeef", SwapFilesystemRefuse},
	}
	for _, tc := range cases {
		if got := ResolveSwapFilesystem(spec, tc.fs); got != tc.decision {
			t.Fatalf("ResolveSwapFilesystem(%q) = %v, want %v", tc.fs, got, tc.decision)
		}
	}

	// btrfs with fs_btrfs: refuse must not create a swapfile, because the
	// btrfs docs call a root-filesystem swapfile actively discouraged.
	refusing := spec
	refusing.FSBtrfs = "refuse"
	if got := ResolveSwapFilesystem(refusing, "btrfs"); got != SwapFilesystemRefuse {
		t.Fatalf("btrfs with fs_btrfs=refuse = %v, want refuse", got)
	}
}

// Adoption is the common case on this fleet: both Oracle hosts already ship an
// 8 GB swapfile that the operator created. The tool must not touch it.
func TestSwapAdoptionIsDetectedNotCreated(t *testing.T) {
	hw := NewHardwareState(974092, 2, "ext4", 34_000_000_000, []SwapDevice{
		{Name: "/swapfile", Type: "file", SizeKB: 8388604, Priority: -1},
	})
	adopted, name, size := hw.AdoptedDiskSwap()
	if !adopted {
		t.Fatal("an existing disk swapfile was not detected")
	}
	if name != "/swapfile" {
		t.Fatalf("adopted name = %q", name)
	}
	if size != 8388604*1024 {
		t.Fatalf("adopted size = %d", size)
	}
}

func TestSwapAdoptionIsAbsentWhenOnlyZRAMExists(t *testing.T) {
	hw := NewHardwareState(974092, 2, "ext4", 34_000_000_000, []SwapDevice{
		{Name: "/dev/zram0", Type: "partition", SizeKB: 23518204, Priority: 100},
	})
	if adopted, _, _ := hw.AdoptedDiskSwap(); adopted {
		t.Fatal("a zram device must not count as an adoptable disk swap")
	}
}

func TestSwapSpecValidation(t *testing.T) {
	valid := swapSpecFixture()
	if err := ValidateSwapSpec(valid); err != nil {
		t.Fatalf("the declared spec is invalid: %v", err)
	}
	for name, broken := range map[string]SwapSpec{
		"no file":        {Policy: "auto", SizeOf: "mem_total", SizeMin: "1G", SizeMax: "8G", DiskReserve: "1G"},
		"bad size base":  {Policy: "auto", File: "/swapfile.envctl", SizeOf: "disk", SizeMin: "1G", SizeMax: "8G", DiskReserve: "1G"},
		"min above max":  {Policy: "auto", File: "/swapfile.envctl", SizeOf: "mem_total", SizeMin: "9G", SizeMax: "8G", DiskReserve: "1G"},
		"unknown policy": {Policy: "maybe", File: "/swapfile.envctl", SizeOf: "mem_total", SizeMin: "1G", SizeMax: "8G", DiskReserve: "1G"},
		"traversal":      {Policy: "auto", File: "../escape", SizeOf: "mem_total", SizeMin: "1G", SizeMax: "8G", DiskReserve: "1G"},
	} {
		if err := ValidateSwapSpec(broken); err == nil {
			t.Fatalf("expected %s to be rejected", name)
		}
	}
}

func TestSwapSpecPolicyDisabled(t *testing.T) {
	spec := swapSpecFixture()
	spec.Policy = SwapPolicyDisabled
	if spec.Enabled() {
		t.Fatal("a disabled policy must not report as enabled")
	}
	spec.Policy = SwapPolicyAuto
	if !spec.Enabled() {
		t.Fatal("the auto policy must report as enabled")
	}
}

func TestFormatBytesForDiagnostics(t *testing.T) {
	if got := FormatBytes(1_000_000_000); !strings.Contains(got, "G") {
		t.Fatalf("FormatBytes(1G) = %q", got)
	}
	if got := FormatBytes(974092 * 1024); got == "" {
		t.Fatal("FormatBytes returned an empty string")
	}
}
