package usecase

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPruneStaleSkillsRemovesOnlyUnlistedDirs(t *testing.T) {
	base := t.TempDir()

	for _, name := range []string{"keep-a", "keep-b", "stale-x", ".hidden"} {
		if err := os.MkdirAll(filepath.Join(base, name), 0o755); err != nil {
			t.Fatalf("setup mkdir %s: %v", name, err)
		}
	}
	// A stray regular file must be left untouched.
	if err := os.WriteFile(filepath.Join(base, "afile"), []byte("x"), 0o644); err != nil {
		t.Fatalf("setup file: %v", err)
	}

	wanted := map[string]bool{"keep-a": true, "keep-b": true}
	removed := pruneStaleSkills(base, wanted, nil)

	if len(removed) != 1 || removed[0] != "stale-x" {
		t.Fatalf("expected exactly [stale-x] removed, got %v", removed)
	}
	if _, err := os.Stat(filepath.Join(base, "stale-x")); !os.IsNotExist(err) {
		t.Errorf("stale-x should have been removed, stat err=%v", err)
	}
	for _, name := range []string{"keep-a", "keep-b", ".hidden", "afile"} {
		if _, err := os.Stat(filepath.Join(base, name)); err != nil {
			t.Errorf("%s should have been preserved: %v", name, err)
		}
	}
}

func TestPruneStaleSkillsEmptyWantedIsNoop(t *testing.T) {
	base := t.TempDir()
	if err := os.MkdirAll(filepath.Join(base, "anything"), 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}

	if removed := pruneStaleSkills(base, map[string]bool{}, nil); len(removed) != 0 {
		t.Fatalf("expected no removals with an empty wanted set, got %v", removed)
	}
	if _, err := os.Stat(filepath.Join(base, "anything")); err != nil {
		t.Errorf("nothing should be removed when wanted is empty: %v", err)
	}
}

func TestPruneStaleSkillsMissingDirIsNoop(t *testing.T) {
	if removed := pruneStaleSkills(filepath.Join(t.TempDir(), "does-not-exist"), map[string]bool{"a": true}, nil); len(removed) != 0 {
		t.Fatalf("expected no removals for a missing dir, got %v", removed)
	}
}
