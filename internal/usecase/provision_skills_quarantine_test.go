package usecase

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// The quarantine exists so a skill removed by mistake stays recoverable, but
// nothing ever pruned it: the tree grew by one directory per removed skill
// (39 directories per runtime on a machine where the catalog went 50 -> 12).
func TestExpireQuarantinedSkillsRemovesOnlyStaleEntries(t *testing.T) {
	trash := filepath.Join(t.TempDir(), ".envctl-trash", "skills")
	if err := os.MkdirAll(trash, 0700); err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	age := func(name string, d time.Duration) {
		t.Helper()
		dir := filepath.Join(trash, name)
		if err := os.MkdirAll(filepath.Join(dir, "references"), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("x"), 0600); err != nil {
			t.Fatal(err)
		}
		when := now.Add(-d)
		if err := os.Chtimes(dir, when, when); err != nil {
			t.Fatal(err)
		}
	}
	age("grill-me-20260801-120000", 90*24*time.Hour) // expired
	age("ssh-vps-20260815-120000", 60*24*time.Hour)  // expired
	age("docker-20260925-120000", 24*time.Hour)      // still recoverable

	// A file (not a quarantined skill) in the same tree must survive.
	stray := filepath.Join(trash, "notes.txt")
	if err := os.WriteFile(stray, []byte("keep me"), 0600); err != nil {
		t.Fatal(err)
	}

	removed, err := expireQuarantinedSkills(trash, 30*24*time.Hour)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(removed) != 2 {
		t.Errorf("expected 2 expired entries, got %d (%v)", len(removed), removed)
	}
	for _, gone := range []string{"grill-me-20260801-120000", "ssh-vps-20260815-120000"} {
		if _, err := os.Stat(filepath.Join(trash, gone)); !os.IsNotExist(err) {
			t.Errorf("expected %q to expire", gone)
		}
	}
	if _, err := os.Stat(filepath.Join(trash, "docker-20260925-120000")); err != nil {
		t.Errorf("recent quarantine must stay recoverable: %v", err)
	}
	if _, err := os.Stat(stray); err != nil {
		t.Errorf("unrelated file must not be touched: %v", err)
	}
}

// quarantineSkill used to leave the moved directory with the mtime of its last
// write, so a skill that had been untouched for longer than the TTL was deleted
// by the very deploy that quarantined it — the recovery window was zero for
// exactly the skills it existed for. The window is only real if the age is
// measured from the moment of the move, so this walks the real path.
func TestQuarantinedSkillStaysRecoverableForTheWholeWindow(t *testing.T) {
	root := t.TempDir()
	live := filepath.Join(root, "skills", "long-lived-skill")
	if err := os.MkdirAll(filepath.Join(live, "references"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(live, "SKILL.md"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	// The skill was last written 90 days ago — well past the 30-day window.
	longAgo := time.Now().Add(-90 * 24 * time.Hour)
	if err := os.Chtimes(live, longAgo, longAgo); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(filepath.Join(live, "SKILL.md"), longAgo, longAgo); err != nil {
		t.Fatal(err)
	}

	dest, err := quarantineSkill(filepath.Join(root, "skills"), "long-lived-skill")
	if err != nil {
		t.Fatalf("quarantineSkill: %v", err)
	}

	trashDir := filepath.Join(root, ".envctl-trash", "skills")
	expired, err := expireQuarantinedSkills(trashDir, staleSkillQuarantineTTL)
	if err != nil {
		t.Fatalf("expireQuarantinedSkills: %v", err)
	}
	if len(expired) != 0 {
		t.Errorf("a skill quarantined seconds ago must not expire: got %v", expired)
	}
	if _, err := os.Stat(dest); err != nil {
		t.Errorf("quarantined skill must stay recoverable for the whole window: %v", err)
	}

	// Same tree, but genuinely quarantined before the window: it must expire.
	aged := filepath.Join(trashDir, "ancient-skill-20260101-000000")
	if err := os.MkdirAll(aged, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(aged, longAgo, longAgo); err != nil {
		t.Fatal(err)
	}
	expired, err = expireQuarantinedSkills(trashDir, staleSkillQuarantineTTL)
	if err != nil {
		t.Fatalf("expireQuarantinedSkills (aged): %v", err)
	}
	if len(expired) != 1 || expired[0] != filepath.Base(aged) {
		t.Errorf("expired = %v, want only %q", expired, filepath.Base(aged))
	}
	if _, err := os.Stat(dest); err != nil {
		t.Errorf("the fresh quarantine must survive the aged one expiring: %v", err)
	}
}

func TestExpireQuarantinedSkillsMissingDirIsNoop(t *testing.T) {
	removed, err := expireQuarantinedSkills(filepath.Join(t.TempDir(), "absent"), time.Hour)
	if err != nil {
		t.Fatalf("missing trash must not error: %v", err)
	}
	if len(removed) != 0 {
		t.Errorf("expected no removals, got %v", removed)
	}
}
