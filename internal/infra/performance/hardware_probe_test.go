package performance

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/eajdias/envctl/internal/domain/entity"
)

// The two meminfo fixtures below are the real /proc/meminfo MemTotal lines
// reported by the fleet, not nominal instance sizes:
//
//	vps_oracle_2 / vps_oracle_1 (Oracle, "1 GB" shape) ->  974092 kB =  951 MiB
//	zscan_chatbot              (AWS,   "16 GB" shape) -> 16162396 kB = 15783 MiB
//
// MemTotal is always short of the nominal size, so a tier boundary written
// against "1 GB" or "16 GB" would misclassify both. These assertions are the
// guard against that.
func TestParseMemTotalConvertsKilobytesToMebibytes(t *testing.T) {
	cases := []struct {
		name    string
		host    string
		meminfo string
		wantKB  uint64
		wantMiB uint64
	}{
		{
			name:    "oracle 1 GB shape reports 951 MiB",
			host:    "vps_oracle_2",
			meminfo: "MemTotal:       974092 kB\nMemFree:         124916 kB\n",
			wantKB:  974092,
			wantMiB: 951,
		},
		{
			name:    "aws 16 GB shape reports 15783 MiB",
			host:    "zscan_chatbot",
			meminfo: "MemTotal:       16162396 kB\nMemFree:        2100388 kB\n",
			wantKB:  16162396,
			wantMiB: 15783,
		},
		{
			name:    "absent meminfo is zero",
			host:    "fixture",
			meminfo: "HugePages_Total:       0\n",
			wantKB:  0,
			wantMiB: 0,
		},
		{
			name:    "unparseable meminfo is zero",
			host:    "fixture",
			meminfo: "MemTotal:       not-a-number kB\n",
			wantKB:  0,
			wantMiB: 0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			kb := parseMemTotalKB(tc.meminfo)
			if kb != tc.wantKB {
				t.Fatalf("parseMemTotalKB = %d, want %d (host %s)", kb, tc.wantKB, tc.host)
			}
			if got := (entity.HardwareState{MemTotalKB: kb}).MemTotalMiB(); got != tc.wantMiB {
				t.Fatalf("MemTotalMiB = %d, want %d (host %s)", got, tc.wantMiB, tc.host)
			}
		})
	}
}

// swapWithZRAM is a host that has both: compressed RAM swap above a disk
// fallback. vps_oracle_2 today has only the disk row; this is the shape the
// profile produces once the zram lifecycle runs.
const swapWithZRAM = "Filename\t\t\t\tType\t\tSize\t\tUsed\t\tPriority\n" +
	"/dev/zram0                              partition\t23518204\t4127236\t\t100\n" +
	"/swapfile                               file\t\t8388604\t\t106060\t\t-1\n"

// swapDiskOnly is the measured vps_oracle_2 / vps_oracle_1 shape: an Oracle
// swapfile with no zram at all.
const swapDiskOnly = "Filename\t\t\t\tType\t\tSize\t\tUsed\t\tPriority\n" +
	"/swapfile                               file\t\t8388604\t\t106060\t\t-1\n"

// swapNone is the measured zscan_chatbot shape: no swap whatsoever.
const swapNone = "Filename\t\t\t\tType\t\tSize\t\tUsed\t\tPriority\n"

func TestNewHardwareStateDerivesSwapTopology(t *testing.T) {
	cases := []struct {
		name             string
		host             string
		table            string
		wantHasZRAM      bool
		wantZRAMPriority int
		wantHasDisk      bool
		wantDiskTopPri   int
	}{
		{
			name:             "zram above a disk fallback",
			host:             "post-Task-8 vps_oracle_2",
			table:            swapWithZRAM,
			wantHasZRAM:      true,
			wantZRAMPriority: 100,
			wantHasDisk:      true,
			wantDiskTopPri:   -1,
		},
		{
			name:             "disk only, as measured today",
			host:             "vps_oracle_2",
			table:            swapDiskOnly,
			wantHasZRAM:      false,
			wantZRAMPriority: 0,
			wantHasDisk:      true,
			wantDiskTopPri:   -1,
		},
		{
			name:             "no swap at all",
			host:             "zscan_chatbot",
			table:            swapNone,
			wantHasZRAM:      false,
			wantZRAMPriority: 0,
			wantHasDisk:      false,
			wantDiskTopPri:   entity.NoDiskSwapPriority,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			state := entity.NewHardwareState(0, 0, "", 0, parseSwapTable(tc.table))
			if state.HasZRAM != tc.wantHasZRAM {
				t.Fatalf("HasZRAM = %v, want %v (host %s)", state.HasZRAM, tc.wantHasZRAM, tc.host)
			}
			if state.ZRAMPriority != tc.wantZRAMPriority {
				t.Fatalf("ZRAMPriority = %d, want %d (host %s)", state.ZRAMPriority, tc.wantZRAMPriority, tc.host)
			}
			if state.HasDiskSwap() != tc.wantHasDisk {
				t.Fatalf("HasDiskSwap = %v, want %v (host %s)", state.HasDiskSwap(), tc.wantHasDisk, tc.host)
			}
			if state.DiskSwapTopPri != tc.wantDiskTopPri {
				t.Fatalf("DiskSwapTopPri = %d, want %d (host %s)", state.DiskSwapTopPri, tc.wantDiskTopPri, tc.host)
			}
			if len(state.Swap) != len(parseSwapTable(tc.table)) {
				t.Fatalf("swap table was not carried through: %#v", state.Swap)
			}
		})
	}
}

// IsDiskSwapDevice separates a compressed-RAM device from a real disk file or
// partition, so adoption and swappiness decisions key off the right thing.
func TestIsDiskSwapDevice(t *testing.T) {
	cases := map[string]bool{
		"/dev/zram0":       false,
		"/dev/zram1":       false,
		"/swapfile":        true,
		"/swapfile.envctl": true,
		"/dev/sda2":        true,
	}
	for name, want := range cases {
		if got := entity.IsDiskSwapDevice(name); got != want {
			t.Fatalf("IsDiskSwapDevice(%q) = %v, want %v", name, got, want)
		}
	}
}

func TestHardwareProbeSnapshot(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "proc"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "proc", "meminfo"),
		[]byte("MemTotal:       974092 kB\n"), 0o644); err != nil {
		t.Fatalf("write meminfo: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "proc", "swaps"), []byte(swapDiskOnly), 0o644); err != nil {
		t.Fatalf("write swaps: %v", err)
	}

	probe := newHardwareProbe(root, func(string) (fsStat, error) {
		return fsStat{Type: "ext4", FreeBytes: 34_000_000_000}, nil
	}, nil)

	state := probe.Snapshot(context.Background())
	if state.MemTotalKB != 974092 {
		t.Fatalf("MemTotalKB = %d, want 974092", state.MemTotalKB)
	}
	if state.MemTotalMiB() != 951 {
		t.Fatalf("MemTotalMiB = %d, want 951", state.MemTotalMiB())
	}
	if state.RootFSType != "ext4" {
		t.Fatalf("RootFSType = %q, want ext4", state.RootFSType)
	}
	if state.DiskFreeBytes != 34_000_000_000 {
		t.Fatalf("DiskFreeBytes = %d", state.DiskFreeBytes)
	}
	if !state.HasDiskSwap() || state.HasZRAM {
		t.Fatalf("swap topology = zram:%v disk:%v, want zram:false disk:true", state.HasZRAM, state.HasDiskSwap())
	}
	if state.CPUCount < 1 {
		t.Fatalf("CPUCount = %d, want the affinity mask to report at least one CPU", state.CPUCount)
	}
}

// A missing procfs or a statfs failure must degrade to zero values, never to an
// error and never to a panic: the probe is read-only and best effort.
func TestHardwareProbeDegradesWhenSourcesAreMissing(t *testing.T) {
	probe := newHardwareProbe(filepath.Join(t.TempDir(), "absent"), func(string) (fsStat, error) {
		return fsStat{}, os.ErrNotExist
	}, nil)

	state := probe.Snapshot(context.Background())
	if state.MemTotalKB != 0 || state.RootFSType != "" || state.DiskFreeBytes != 0 {
		t.Fatalf("degraded snapshot is not zero-valued: %#v", state)
	}
	if state.HasDiskSwap() || state.HasZRAM {
		t.Fatalf("degraded snapshot invented swap: %#v", state)
	}
	if got := state.MemTotalMiB(); got != 0 {
		t.Fatalf("MemTotalMiB on a degraded snapshot = %d, want 0", got)
	}
}

func TestFilesystemTypeNames(t *testing.T) {
	cases := map[int64]string{
		0xEF53:     "ext4",
		0x58465342: "xfs",
		0x9123683E: "btrfs",
		0x01021994: "tmpfs",
		0x794c7630: "overlay",
		0x2FC12FC1: "zfs",
		0x6969:     "nfs",
	}
	for magic, want := range cases {
		if got := filesystemTypeName(magic); got != want {
			t.Fatalf("filesystemTypeName(%#x) = %q, want %q", magic, got, want)
		}
	}
	// An unknown magic must still yield a name so the swap policy can refuse it
	// explicitly instead of silently assuming a supported filesystem.
	if got := filesystemTypeName(0xDEADBEEF); got == "" {
		t.Fatal("unknown filesystem magic must still return a name")
	}
}
