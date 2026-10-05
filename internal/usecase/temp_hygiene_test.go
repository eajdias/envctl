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
		// third-party installer/updater caches are safe to prune
		{name: "DockerDesktop", isDir: true, want: true},
		{name: "DockerDesktopUpdates", isDir: true, want: true},
		{name: "DockerDesktopInstallers", isDir: true, want: true},
		{name: "BraveComponentUpdater_chrome_url_fetcher_6420_123", isDir: true, want: true},
		{name: "scoped_dir3136_2118523775", isDir: true, want: true},
		{name: "WinGet", isDir: true, want: true},
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

func TestTempOwner(t *testing.T) {
	cases := []struct {
		name string
		want string
	}{
		{"opencode", tempOwnerScratch},
		{"opencode-session", tempOwnerScratch},
		{"commandcode", tempOwnerScratch},
		{"node-compile-cache", tempOwnerScratch},
		{"tsx-someuser", tempOwnerScratch},
		{"zscan-assist-LBbuFs", tempOwnerScratch},
		{"DockerDesktop", tempOwnerThirdParty},
		{"DockerDesktopUpdates", tempOwnerThirdParty},
		{"BraveComponentUpdater_chrome_url_fetcher_6420_123", tempOwnerThirdParty},
		{"scoped_dir3136_2118523775", tempOwnerThirdParty},
		{"WinGet", tempOwnerThirdParty},
		{"vscode-stable-user-x64", tempOwnerThirdParty},
		{"random-user-file.txt", tempOwnerUnknown},
	}
	for _, tc := range cases {
		if got := tempOwner(tc.name); got != tc.want {
			t.Errorf("tempOwner(%q) = %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestDominantTempOwner(t *testing.T) {
	// Scratch-dominant: the agent's own cache out-sizes third-party caches.
	scratchDir := t.TempDir()
	opencodeDir := filepath.Join(scratchDir, "opencode")
	if err := os.MkdirAll(opencodeDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(opencodeDir, "big.bin"), make([]byte, 4096), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(scratchDir, "DockerDesktop"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(scratchDir, "DockerDesktop", "small.bin"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	if got := dominantTempOwner(scratchDir); got != tempOwnerScratch {
		t.Errorf("dominantTempOwner(scratch) = %q, want %q", got, tempOwnerScratch)
	}

	// Third-party-dominant: Docker cache out-sizes any scratch.
	thirdDir := t.TempDir()
	dockerDir := filepath.Join(thirdDir, "DockerDesktop")
	if err := os.MkdirAll(dockerDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dockerDir, "big.bin"), make([]byte, 8192), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(thirdDir, "opencode"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(thirdDir, "opencode", "small.bin"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	if got := dominantTempOwner(thirdDir); got != tempOwnerThirdParty {
		t.Errorf("dominantTempOwner(third-party) = %q, want %q", got, tempOwnerThirdParty)
	}

	if got := dominantTempOwner(filepath.Join(t.TempDir(), "does-not-exist")); got != tempOwnerScratch {
		t.Errorf("dominantTempOwner(missing) = %q, want %q", got, tempOwnerScratch)
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
