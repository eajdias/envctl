package usecase

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestNpmShortName(t *testing.T) {
	cases := map[string]string{
		"npm:command-code":    "command-code",
		"npm:@playwright/cli": "cli",
		"npm:prettier":        "prettier",
		"node@24.19.0":        "",
		"prettier":            "",
		"":                    "",
	}
	for in, want := range cases {
		if got := npmShortName(in); got != want {
			t.Errorf("npmShortName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestToolBinName(t *testing.T) {
	cases := map[string]string{
		"prettier --version":       "prettier",
		"tsc --version":            "tsc",
		"cmdc --version":           "cmdc",
		"playwright-cli --version": "playwright-cli",
		"":                         "",
	}
	for in, want := range cases {
		if got := toolBinName(in); got != want {
			t.Errorf("toolBinName(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestSweepLegacyToolShims builds a fake home plus a fake LOCALAPPDATA so
// both the legacy dir and the shim dir resolve under temp: a legacy copy
// with a mise shim beside it must be archived, one without must be kept.
func TestSweepLegacyToolShims(t *testing.T) {
	home := t.TempDir()
	localApp := t.TempDir()
	t.Setenv("LOCALAPPDATA", localApp)

	var legacy, shimDir string
	if runtime.GOOS == "windows" {
		legacy = filepath.Join(home, ".local", "bin")
		shimDir = filepath.Join(localApp, "mise", "shims")
	} else {
		legacy = filepath.Join(home, ".local", "bin")
		shimDir = filepath.Join(home, ".local", "share", "mise", "shims")
	}
	for _, d := range []string{legacy, shimDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	shimName := "prettier"
	if runtime.GOOS == "windows" {
		shimName = "prettier.exe"
	}
	mustWriteFile(t, filepath.Join(shimDir, shimName), "shim")
	mustWriteFile(t, filepath.Join(legacy, "prettier"), "stale")
	mustWriteFile(t, filepath.Join(legacy, "kept"), "stale-no-shim")

	swept := sweepLegacyToolShims(&mockLogger{}, home, []string{"prettier", "kept"})
	if len(swept) != 1 {
		t.Fatalf("swept = %v, want exactly the archived prettier copy", swept)
	}
	if _, err := os.Stat(filepath.Join(legacy, "prettier")); !os.IsNotExist(err) {
		t.Error("stale prettier copy still in place although the mise shim exists")
	}
	if _, err := os.Stat(filepath.Join(legacy, "kept")); err != nil {
		t.Error("copy without a mise shim must be left untouched")
	}
	if _, err := os.Stat(swept[0]); err != nil {
		t.Errorf("archived copy %q not found: %v", swept[0], err)
	}
}

func mustWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
