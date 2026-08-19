package logger

import (
	"os"
	"strings"
	"testing"
)

func TestFileLogger_WriteAndClose(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "envctl-log-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	fl, err := NewFileLogger(tempDir)
	if err != nil {
		t.Fatalf("Failed to initialize file logger: %v", err)
	}

	fl.Info("Testing info message")
	fl.Warn("Testing warn message: %s", "attention required")
	fl.Error("Testing error message: code %d", 500)
	fl.Debug("Testing debug message")
	fl.LogCommand("test_cmd", []string{"arg1", "arg2"}, 0, "sample stdout output", nil)
	fl.LogIdempotency("winget", "Git.Git", true, "package Git.Git already installed")

	logPath := fl.GetLogFilePath()
	if !strings.HasPrefix(logPath, tempDir) {
		t.Errorf("Expected log path to be in %s, got %s", tempDir, logPath)
	}

	if err := fl.Close(); err != nil {
		t.Errorf("Failed to close file logger: %v", err)
	}

	contentBytes, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	content := string(contentBytes)
	expectedStrings := []string{
		"[INFO ] Testing info message",
		"[WARN ] Testing warn message: attention required",
		"[ERROR] Testing error message: code 500",
		"[DEBUG] Testing debug message",
		"Command: test_cmd arg1 arg2",
		"Result: SUCCESS",
		"sample stdout output",
		"[IDEMPOTENT-SKIP]",
		"package Git.Git already installed",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(content, expected) {
			t.Errorf("Expected log to contain '%s', but it didn't. Full log:\n%s", expected, content)
		}
	}
}
