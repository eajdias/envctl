package performance

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestParseSwapTable(t *testing.T) {
	got := parseSwapTable("Filename\tType\tSize\tUsed\tPriority\n/swapfile\tfile\t4194300\t729316\t-2\n/dev/zram0\tpartition\t23518204\t2176840\t100\n")
	if len(got) != 2 {
		t.Fatalf("swap entries = %d, want 2", len(got))
	}
	if got[0].Name != "/swapfile" || got[0].SizeKB != 4194300 || got[0].UsedKB != 729316 || got[0].Priority != -2 {
		t.Errorf("first swap entry = %#v", got[0])
	}
	if got[1].Name != "/dev/zram0" || got[1].Type != "partition" || got[1].Priority != 100 {
		t.Errorf("second swap entry = %#v", got[1])
	}
}

func TestParseZramctl(t *testing.T) {
	got := parseZramctl("NAME       ALGORITHM DISKSIZE DATA COMPR TOTAL STREAMS MOUNTPOINT\n/dev/zram0 zstd         22,4G   2G   1G   1G         1 [SWAP]\n")
	if !got.Present || got.Algorithm != "zstd" || got.SizeBytes != 22_400_000_000 {
		t.Errorf("zram state = %#v", got)
	}
}

func TestParseScheduler(t *testing.T) {
	selected, available := parseScheduler("none mq-deadline [kyber] adios bfq")
	if selected != "kyber" || available != "none mq-deadline kyber adios bfq" {
		t.Errorf("scheduler = %q, available = %q", selected, available)
	}
}

func TestParseJournaldDiskUsage(t *testing.T) {
	usage := parseJournaldDiskUsage("Archived and active journals take up 46.9M in the file system.")
	if usage != "46.9M" {
		t.Errorf("journald usage = %q, want 46.9M", usage)
	}
}

func TestPerformanceInspectorSnapshotUsesReadOnlyFixtures(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, filepath.Join(root, "proc/swaps"), "Filename\tType\tSize\tUsed\tPriority\n/swapfile\tfile\t4194300\t729316\t-2\n")
	writeFixture(t, filepath.Join(root, "sys/block/zram0/comp_algorithm"), "lzo lz4 [zstd] deflate\n")
	writeFixture(t, filepath.Join(root, "sys/block/zram0/disksize"), "22400000000\n")
	writeFixture(t, filepath.Join(root, "sys/block/zram0/mem_used_total"), "2000000000\n")
	writeFixture(t, filepath.Join(root, "sys/block/zram0/mem_total"), "22400000000\n")
	writeFixture(t, filepath.Join(root, "sys/block/nvme0n1/queue/scheduler"), "none mq-deadline [kyber] adios bfq\n")
	writeFixture(t, filepath.Join(root, "sys/devices/system/cpu/cpu0/cpufreq/scaling_governor"), "schedutil\n")

	inspector := newPerformanceInspector(root, func(context.Context, string, ...string) ([]byte, error) {
		return nil, nil
	})
	snapshot := inspector.Snapshot(context.Background())
	if len(snapshot.Swap) != 1 || snapshot.Swap[0].Name != "/swapfile" {
		t.Fatalf("snapshot swap = %#v", snapshot.Swap)
	}
	if !snapshot.ZRAM.Present || snapshot.ZRAM.Algorithm != "zstd" || snapshot.ZRAM.SizeBytes == 0 {
		t.Fatalf("snapshot zram = %#v", snapshot.ZRAM)
	}
	if len(snapshot.CPUGovernors) != 1 || snapshot.CPUGovernors[0] != "schedutil" {
		t.Fatalf("snapshot governors = %#v", snapshot.CPUGovernors)
	}
	if len(snapshot.BlockSchedulers) != 1 || snapshot.BlockSchedulers[0].Selected != "kyber" {
		t.Fatalf("snapshot schedulers = %#v", snapshot.BlockSchedulers)
	}
}

func writeFixture(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
