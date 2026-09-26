package usecase

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPruneStaleSkillsQuarantinesOnlyUnlistedDirs(t *testing.T) {
	base := t.TempDir()

	for _, name := range []string{"keep-a", "keep-b", "stale-x", ".hidden"} {
		if err := os.MkdirAll(filepath.Join(base, name), 0o755); err != nil {
			t.Fatalf("setup mkdir %s: %v", name, err)
		}
	}
	if err := os.WriteFile(filepath.Join(base, "stale-x", "SKILL.md"), []byte("stale content"), 0o644); err != nil {
		t.Fatalf("setup skill file: %v", err)
	}
	// A stray regular file must be left untouched.
	if err := os.WriteFile(filepath.Join(base, "afile"), []byte("x"), 0o644); err != nil {
		t.Fatalf("setup file: %v", err)
	}

	wanted := map[string]bool{"keep-a": true, "keep-b": true}
	quarantined := pruneStaleSkills(base, wanted, &mockLogger{})

	if len(quarantined) != 1 || quarantined[0] != "stale-x" {
		t.Fatalf("expected exactly [stale-x] quarantined, got %v", quarantined)
	}
	if _, err := os.Stat(filepath.Join(base, "stale-x")); !os.IsNotExist(err) {
		t.Errorf("stale-x should have left the skills dir, stat err=%v", err)
	}
	for _, name := range []string{"keep-a", "keep-b", ".hidden", "afile"} {
		if _, err := os.Stat(filepath.Join(base, name)); err != nil {
			t.Errorf("%s should have been preserved: %v", name, err)
		}
	}

	// The skill must be recoverable from the sibling trash tree, never deleted.
	trash := filepath.Join(filepath.Dir(base), ".envctl-trash", "skills")
	entries, err := os.ReadDir(trash)
	if err != nil {
		t.Fatalf("quarantine directory not created: %v", err)
	}
	if len(entries) != 1 || !strings.HasPrefix(entries[0].Name(), "stale-x-") {
		t.Fatalf("expected one quarantined entry named stale-x-<timestamp>, got %v", entries)
	}
	content, err := os.ReadFile(filepath.Join(trash, entries[0].Name(), "SKILL.md"))
	if err != nil {
		t.Fatalf("quarantined skill content is not recoverable: %v", err)
	}
	if string(content) != "stale content" {
		t.Errorf("quarantined content = %q, want %q", content, "stale content")
	}
}

func TestPruneStaleSkillsEmptyWantedIsNoop(t *testing.T) {
	base := t.TempDir()
	if err := os.MkdirAll(filepath.Join(base, "anything"), 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}

	if removed := pruneStaleSkills(base, map[string]bool{}, &mockLogger{}); len(removed) != 0 {
		t.Fatalf("expected no removals with an empty wanted set, got %v", removed)
	}
	if _, err := os.Stat(filepath.Join(base, "anything")); err != nil {
		t.Errorf("nothing should be removed when wanted is empty: %v", err)
	}
}

func TestPruneStaleSkillsMissingDirIsNoop(t *testing.T) {
	if removed := pruneStaleSkills(filepath.Join(t.TempDir(), "does-not-exist"), map[string]bool{"a": true}, &mockLogger{}); len(removed) != 0 {
		t.Fatalf("expected no removals for a missing dir, got %v", removed)
	}
}
