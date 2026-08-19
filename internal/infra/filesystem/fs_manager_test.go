package filesystem

import (
	"os"
	"path/filepath"
	"testing"
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
