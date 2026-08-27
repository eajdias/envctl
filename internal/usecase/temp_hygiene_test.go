package usecase

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestClassifyTempEntry(t *testing.T) {
	fresh := time.Minute
	stale := 25 * time.Hour

	tests := []struct {
		name  string
		isDir bool
		age   time.Duration
		want  bool
	}{
		{name: ".bdef99ebee6f3ffc-00000000.dll", want: true},
		{name: ".feefafc39f67efee-00000001.node", want: true},
		{name: "node-compile-cache", isDir: true, want: true},
		{name: "tsx-someuser", isDir: true, want: true},
		{name: "zscan-assist-LBbuFs", isDir: true, want: true},
		{name: "zscan-assist-6.45.0.tar.gz", want: true},
		{name: "Meslo.zip", want: true},
		{name: "playwright_chromiumdev_profile-25WBSL", isDir: true, want: true},
		{name: "system-commandline-sentinel-files", isDir: true, want: true},
		{name: "WinGet", isDir: true, want: true},
		{name: "NuGetScratch", isDir: true, want: true},
		{name: "chocolatey", isDir: true, want: true},
		{name: "vscode-stable-user-x64", isDir: true, want: true},
		{name: "dd_vcredist_amd64_20260818144336.log", want: true},
		{name: "Microsoft.NET.Workload_26996_20260818_202054_140.log", want: true},
		{name: "vscode-inno-updater-1787083801.log", want: true},
		{name: "DEL96A4.tmp", want: true},
		{name: "doctor_out.txt", want: true},
		{name: "jrdfiles.txt", want: true},
		{name: "tree.json", want: true},
		{name: "ods.h", want: true},
		{name: "validation.cpp", want: true},
		{name: "backup25.epp", want: true},
		{name: "repair1_copy.FDB", want: true},
		// active-session scratch must be protected
		{name: "opencode", isDir: true, age: fresh, want: false},
		{name: "opencode", isDir: true, age: stale, want: true},
		{name: "00ed2dbb51c07b9d2cc253e3c9b07eef", isDir: true, age: stale, want: true},
		{name: "00ed2dbb51c07b9d2cc253e3c9b07eef", isDir: true, age: fresh, want: false},
		{name: "nsu7A63.tmp", isDir: true, age: stale, want: true},
		{name: "0hgs12nh.yvw", isDir: true, age: stale, want: true},
		{name: "vnaleqdf.pyr", isDir: true, age: stale, want: true},
		// unknown files must never be touched
		{name: "random-user-file.txt", want: false},
		{name: "project-backup.zip", want: false},
		{name: "my-script.py", want: false},
		{name: "README.md", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifyTempEntry(tt.name, tt.isDir, tt.age)
			if got.remove != tt.want {
				t.Errorf("classifyTempEntry(%q, dir=%v, age=%v).remove = %v, want %v", tt.name, tt.isDir, tt.age, got.remove, tt.want)
			}
		})
	}
}

func TestTempRoots(t *testing.T) {
	dir := t.TempDir()
	origTMP, origTEMP, origTMPDIR := os.Getenv("TMP"), os.Getenv("TEMP"), os.Getenv("TMPDIR")
	t.Cleanup(func() {
		os.Setenv("TMP", origTMP)
		os.Setenv("TEMP", origTEMP)
		os.Setenv("TMPDIR", origTMPDIR)
	})

	os.Setenv("TMP", dir)
	os.Setenv("TEMP", dir)
	os.Setenv("TMPDIR", "")

	roots := tempRoots()
	found := false
	for _, r := range roots {
		if filepath.Clean(r) == filepath.Clean(dir) {
			found = true
		}
	}
	if !found {
		t.Errorf("tempRoots() = %v, expected to include %v", roots, dir)
	}

	// Deduplication: same dir must appear only once.
	count := 0
	for _, r := range roots {
		if filepath.Clean(r) == filepath.Clean(dir) {
			count++
		}
	}
	if count != 1 {
		t.Errorf("tempRoots() duplicated entry for %v: %v", dir, roots)
	}
}

func TestIsRandomTempDir(t *testing.T) {
	cases := []struct {
		name string
		want bool
	}{
		{"00ed2dbb51c07b9d2cc253e3c9b07eef", true},
		{"0hgs12nh.yvw", true},
		{"o1nsukaz.vte", true},
		{"vnaleqdf.pyr", true},
		{"nsu7A63.tmp", true},
		{"normal-dir", false},
		{"docs", false},
		{"repair1_copy.FDB", false},
	}
	for _, c := range cases {
		if got := isRandomTempDir(c.name); got != c.want {
			t.Errorf("isRandomTempDir(%q) = %v, want %v", c.name, got, c.want)
		}
	}
}
