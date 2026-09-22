package filesystem

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

func TestExpandUserPath(t *testing.T) {
	mgr := NewFileSystemManager()

	// Test relative / home expansion
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("failed to get home dir: %v", err)
	}

	expanded, err := mgr.ExpandUserPath("~/test-file.txt")
	if err != nil {
		t.Fatalf("ExpandUserPath failed: %v", err)
	}

	expected := filepath.Join(home, "test-file.txt")
	if expanded != expected {
		t.Errorf("expected %q, got %q", expected, expanded)
	}
}

func TestWriteWithBackup(t *testing.T) {
	mgr := NewFileSystemManager()

	tempDir, err := os.MkdirTemp("", "fsmanager-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testFile := filepath.Join(tempDir, "test.txt")
	initialContent := []byte("hello world")
	if err := os.WriteFile(testFile, initialContent, 0644); err != nil {
		t.Fatalf("failed to write initial test file: %v", err)
	}

	updatedContent := []byte("hello updated world")
	backupPath, err := mgr.WriteWithBackup(testFile, updatedContent, 0644)
	if err != nil {
		t.Fatalf("WriteWithBackup failed: %v", err)
	}

	if backupPath == "" {
		t.Errorf("expected non-empty backup path for existing file")
	}

	if !mgr.Exists(backupPath) {
		t.Errorf("backup file %s does not exist", backupPath)
	}

	readBackup, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatalf("failed to read backup file: %v", err)
	}
	if string(readBackup) != "hello world" {
		t.Errorf("expected backup content 'hello world', got '%s'", string(readBackup))
	}

	readNew, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("failed to read updated file: %v", err)
	}
	if string(readNew) != "hello updated world" {
		t.Errorf("expected updated content 'hello updated world', got '%s'", string(readNew))
	}
}

func TestWriteWithBackupSameSecondKeepsBothBackups(t *testing.T) {
	mgr := NewFileSystemManager()

	tempDir, err := os.MkdirTemp("", "fsmanager-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("v1"), 0644); err != nil {
		t.Fatalf("failed to write initial test file: %v", err)
	}

	first, err := mgr.WriteWithBackup(testFile, []byte("v2"), 0644)
	if err != nil {
		t.Fatalf("first WriteWithBackup failed: %v", err)
	}
	second, err := mgr.WriteWithBackup(testFile, []byte("v3"), 0644)
	if err != nil {
		t.Fatalf("second WriteWithBackup failed: %v", err)
	}
	if first == "" || second == "" {
		t.Fatalf("expected two backup paths, got %q and %q", first, second)
	}
	if first == second {
		t.Fatalf("same-second writes must not share a backup path: %q", first)
	}
	for _, p := range []string{first, second} {
		if !mgr.Exists(p) {
			t.Errorf("backup file %s does not exist", p)
		}
	}
	if data, _ := os.ReadFile(first); string(data) != "v1" {
		t.Errorf("first backup should hold v1, got %q", string(data))
	}
	if data, _ := os.ReadFile(second); string(data) != "v2" {
		t.Errorf("second backup should hold v2, got %q", string(data))
	}
	if data, _ := os.ReadFile(testFile); string(data) != "v3" {
		t.Errorf("live file should hold v3, got %q", string(data))
	}
}

func TestWriteWithBackupIdenticalIsNoop(t *testing.T) {
	mgr := NewFileSystemManager()

	tempDir, err := os.MkdirTemp("", "fsmanager-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("same"), 0644); err != nil {
		t.Fatalf("failed to write initial test file: %v", err)
	}
	before, err := os.Stat(testFile)
	if err != nil {
		t.Fatalf("stat failed: %v", err)
	}

	backupPath, err := mgr.WriteWithBackup(testFile, []byte("same"), 0644)
	if err != nil {
		t.Fatalf("WriteWithBackup failed: %v", err)
	}
	if backupPath != "" {
		t.Errorf("identical rewrite must be a no-op (no backup), got %q", backupPath)
	}
	after, err := os.Stat(testFile)
	if err != nil {
		t.Fatalf("stat failed: %v", err)
	}
	if !after.ModTime().Equal(before.ModTime()) {
		t.Errorf("identical rewrite must not touch mtime")
	}
}

func TestCopyEmbeddedTreeBacksUpUserEdits(t *testing.T) {
	mgr := NewFileSystemManager()

	tempDir, err := os.MkdirTemp("", "fsmanager-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	embedded := fstest.MapFS{
		"skill/SKILL.md": {Data: []byte("# template\n")},
	}
	target := filepath.Join(tempDir, "skills")

	if n, err := mgr.CopyEmbeddedTree(embedded, "skill", target); err != nil || n != 1 {
		t.Fatalf("first copy = (%d, %v), want (1, nil)", n, err)
	}
	// Identical second copy is a no-op and counts nothing.
	if n, err := mgr.CopyEmbeddedTree(embedded, "skill", target); err != nil || n != 0 {
		t.Fatalf("identical copy = (%d, %v), want (0, nil)", n, err)
	}

	// User customizes the file; next deploy must back it up, not destroy it.
	userEdited := "# template\n\nMy notes\n"
	if err := os.WriteFile(filepath.Join(target, "SKILL.md"), []byte(userEdited), 0644); err != nil {
		t.Fatalf("failed to seed user edit: %v", err)
	}
	if n, err := mgr.CopyEmbeddedTree(embedded, "skill", target); err != nil || n != 1 {
		t.Fatalf("update copy = (%d, %v), want (1, nil)", n, err)
	}
	entries, err := os.ReadDir(target)
	if err != nil {
		t.Fatalf("ReadDir failed: %v", err)
	}
	backedUp := false
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "SKILL.md.bak.") {
			backedUp = true
			data, _ := os.ReadFile(filepath.Join(target, e.Name()))
			if string(data) != userEdited {
				t.Errorf("backup should hold the user edit, got %q", string(data))
			}
		}
	}
	if !backedUp {
		t.Errorf("user edit was overwritten without a .bak backup")
	}
	if data, _ := os.ReadFile(filepath.Join(target, "SKILL.md")); string(data) != "# template\n" {
		t.Errorf("live file should hold the template after deploy, got %q", string(data))
	}
}

func TestCopyEmbeddedTreePreservesExecBit(t *testing.T) {
	mgr := NewFileSystemManager()

	tempDir, err := os.MkdirTemp("", "fsmanager-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	embedded := fstest.MapFS{
		"bin/tool": {Data: []byte("#!/bin/sh\necho hi\n"), Mode: 0444},
	}
	target := filepath.Join(tempDir, "bin")

	if _, err := mgr.CopyEmbeddedTree(embedded, "bin", target); err != nil {
		t.Fatalf("first copy failed: %v", err)
	}
	// User makes it executable; redeploy of identical content must keep +x.
	if err := os.Chmod(filepath.Join(target, "tool"), 0755); err != nil {
		t.Fatalf("chmod failed: %v", err)
	}
	if _, err := mgr.CopyEmbeddedTree(embedded, "bin", target); err != nil {
		t.Fatalf("second copy failed: %v", err)
	}
	fi, err := os.Stat(filepath.Join(target, "tool"))
	if err != nil {
		t.Fatalf("stat failed: %v", err)
	}
	if fi.Mode()&0111 == 0 {
		t.Errorf("exec bit was dropped by redeploy (mode %v)", fi.Mode())
	}
}
