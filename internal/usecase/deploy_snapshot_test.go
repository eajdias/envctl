package usecase

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/eajdias/envctl/internal/infra/filesystem"
)

func testSnapshotStore(t *testing.T, path string) *DeploySnapshotStore {
	t.Helper()
	return &DeploySnapshotStore{fs: filesystem.NewFileSystemManager(), path: path}
}

func TestDeploySnapshotStore_Roundtrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "deploys.json")
	store := testSnapshotStore(t, path)

	if got := store.Load(); len(got) != 0 {
		t.Fatalf("fresh store must load empty, got %v", got)
	}
	if err := store.Record("~/.config/opencode/opencode.json", []byte(`{"a":1}`)); err != nil {
		t.Fatalf("Record: %v", err)
	}
	if err := store.Record("~/.config/opencode/AGENTS.md", []byte("# rules")); err != nil {
		t.Fatalf("Record: %v", err)
	}

	loaded := testSnapshotStore(t, path).Load()
	if len(loaded) != 2 {
		t.Fatalf("want 2 records, got %d: %v", len(loaded), loaded)
	}
	rec := loaded["~/.config/opencode/opencode.json"]
	if rec.SHA256 != sha256Hex([]byte(`{"a":1}`)) {
		t.Errorf("wrong sha: %q", rec.SHA256)
	}
	if rec.At.IsZero() {
		t.Error("record must carry a deploy timestamp")
	}

	// Re-recording the same destination overwrites, not duplicates.
	if err := store.Record("~/.config/opencode/opencode.json", []byte(`{"a":2}`)); err != nil {
		t.Fatalf("re-Record: %v", err)
	}
	loaded = store.Load()
	if len(loaded) != 2 {
		t.Fatalf("re-record must not duplicate, got %d records", len(loaded))
	}
	if loaded["~/.config/opencode/opencode.json"].SHA256 != sha256Hex([]byte(`{"a":2}`)) {
		t.Error("re-record must replace the sha")
	}
}

func TestDeploySnapshotStore_MissingAndCorrupt(t *testing.T) {
	missing := testSnapshotStore(t, filepath.Join(t.TempDir(), "nope.json"))
	if got := missing.Load(); len(got) != 0 {
		t.Errorf("missing index must load empty, got %v", got)
	}

	corruptPath := filepath.Join(t.TempDir(), "deploys.json")
	if err := os.WriteFile(corruptPath, []byte("{not json"), 0644); err != nil {
		t.Fatal(err)
	}
	if got := testSnapshotStore(t, corruptPath).Load(); len(got) != 0 {
		t.Errorf("corrupt index must degrade to empty, got %v", got)
	}
}

func TestDeploySnapshotStore_NilSafe(t *testing.T) {
	var nilStore *DeploySnapshotStore
	if got := nilStore.Load(); len(got) != 0 {
		t.Errorf("nil store must load empty, got %v", got)
	}
	if err := nilStore.Record("x", []byte("y")); err != nil {
		t.Errorf("nil store Record must be a no-op, got %v", err)
	}
}
