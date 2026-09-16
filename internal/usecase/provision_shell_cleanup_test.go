package usecase

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidBackupSuffix(t *testing.T) {
	valid := []string{"20260916-140849", "20260826-141200"}
	for _, s := range valid {
		if !validBackupSuffix(s) {
			t.Errorf("expected %q to be a valid backup suffix", s)
		}
	}
	invalid := []string{"", "20260916", "20260916-14084", "20260916-1408499", "2026091a-140849", "20260916_140849", "latest"}
	for _, s := range invalid {
		if validBackupSuffix(s) {
			t.Errorf("expected %q to be rejected as backup suffix", s)
		}
	}
}

func TestPruneTimestampedBackups(t *testing.T) {
	dir := t.TempDir()
	files := []string{
		"opencode.json",
		"opencode.json.bak.20260826-141200",
		"opencode.json.bak.20260903-124442",
		"opencode.json.bak.20260916-140849",
		"AGENTS.md.bak.20260916-140849",
		"notes.txt",
	}
	for _, f := range files {
		if err := os.WriteFile(filepath.Join(dir, f), []byte("x"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	removed, err := pruneTimestampedBackups(dir, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(removed) != 2 {
		t.Errorf("expected 2 pruned files, got %d (%v)", len(removed), removed)
	}
	// Newest backup per file must survive; live files and non-backups untouched.
	for _, keep := range []string{"opencode.json", "opencode.json.bak.20260916-140849", "AGENTS.md.bak.20260916-140849", "notes.txt"} {
		if _, err := os.Stat(filepath.Join(dir, keep)); err != nil {
			t.Errorf("expected %q to survive pruning: %v", keep, err)
		}
	}
	for _, gone := range []string{"opencode.json.bak.20260826-141200", "opencode.json.bak.20260903-124442"} {
		if _, err := os.Stat(filepath.Join(dir, gone)); !os.IsNotExist(err) {
			t.Errorf("expected %q to be pruned", gone)
		}
	}
}
