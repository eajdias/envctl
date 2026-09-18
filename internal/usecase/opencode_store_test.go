package usecase

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

// writeSQLiteStore writes a file whose first 100 bytes mimic a real SQLite
// header, so the parser can be exercised without depending on a SQLite driver.
func writeSQLiteStore(t *testing.T, dir string, pageSize uint16, pageCount, freelistPages uint32, totalSize int64) string {
	t.Helper()

	header := make([]byte, 100)
	copy(header[0:16], "SQLite format 3\x00")
	binary.BigEndian.PutUint16(header[16:18], pageSize)
	binary.BigEndian.PutUint32(header[28:32], pageCount)
	binary.BigEndian.PutUint32(header[36:40], freelistPages)

	content := make([]byte, totalSize)
	copy(content, header)

	path := filepath.Join(dir, "opencode.db")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("setup store: %v", err)
	}
	return path
}

func TestInspectOpenCodeStoreReportsReclaimablePages(t *testing.T) {
	path := writeSQLiteStore(t, t.TempDir(), 4096, 241945, 512, 946*1024*1024)

	store, err := InspectOpenCodeStore(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if store.PageSize != 4096 || store.PageCount != 241945 || store.FreelistPages != 512 {
		t.Errorf("unexpected header values: %+v", store)
	}
	if want := int64(512 * 4096); store.ReclaimableBytes != want {
		t.Errorf("ReclaimableBytes = %d, want %d", store.ReclaimableBytes, want)
	}
	if !store.ExceedsThreshold() {
		t.Errorf("a store above the threshold must be reported")
	}
}

func TestInspectOpenCodeStorePageSizeOneMeansSixtyFourK(t *testing.T) {
	path := writeSQLiteStore(t, t.TempDir(), 1, 100, 2, 8*1024*1024)

	store, err := InspectOpenCodeStore(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if store.PageSize != 65536 {
		t.Errorf("PageSize = %d, want 65536 for a header value of 1", store.PageSize)
	}
	if want := int64(2 * 65536); store.ReclaimableBytes != want {
		t.Errorf("ReclaimableBytes = %d, want %d", store.ReclaimableBytes, want)
	}
}

func TestInspectOpenCodeStoreRejectsNonSQLiteFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "opencode.db")
	if err := os.WriteFile(path, []byte("not a database at all, just text"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := InspectOpenCodeStore(path); err == nil {
		t.Errorf("expected an error for a file that is not a SQLite database")
	}
}

func TestOpenCodeStoreExceedsThresholdBoundary(t *testing.T) {
	if (OpenCodeStore{SizeBytes: openCodeStoreWarnBytes}).ExceedsThreshold() {
		t.Errorf("a store exactly at the threshold must not be reported")
	}
	if !(OpenCodeStore{SizeBytes: openCodeStoreWarnBytes + 1}).ExceedsThreshold() {
		t.Errorf("a store above the threshold must be reported")
	}
}

func TestOpenCodeStorePathHonorsXDGDataHome(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", "/custom/data")
	if got, want := openCodeStorePath("/home/user"), "/custom/data/opencode/opencode.db"; got != want {
		t.Errorf("openCodeStorePath = %q, want %q", got, want)
	}
	if got, want := openCodeDataDir("/home/user"), "/custom/data/opencode"; got != want {
		t.Errorf("openCodeDataDir = %q, want %q", got, want)
	}

	t.Setenv("XDG_DATA_HOME", "")
	want := filepath.Join("/home/user", ".local", "share", "opencode", "opencode.db")
	if got := openCodeStorePath("/home/user"); got != want {
		t.Errorf("openCodeStorePath = %q, want %q", got, want)
	}
	if got, want := openCodeDataDir("/home/user"), filepath.Join("/home/user", ".local", "share", "opencode"); got != want {
		t.Errorf("openCodeDataDir = %q, want %q", got, want)
	}
}
