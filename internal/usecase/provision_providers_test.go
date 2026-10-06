package usecase

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/eajdias/envctl/internal/domain/entity"
	"github.com/eajdias/envctl/internal/domain/repository"
	"github.com/eajdias/envctl/internal/infra/filesystem"
)

func TestFirstVersionToken(t *testing.T) {
	cases := []struct {
		output string
		want   string
	}{
		{"1.55.1", "1.55.1"},
		{"opencode v2.0.5", "2.0.5"},
		{"mise 2026.9.9", "2026.9.9"},
		{"CommandCode CLI v1.56.0 (linux)", "1.56.0"},
		{"v0.0.55", "0.0.55"},
		{"no numbers here", ""},
		{"", ""},
	}
	for _, tc := range cases {
		if got := firstVersionToken(tc.output); got != tc.want {
			t.Errorf("firstVersionToken(%q) = %q, want %q", tc.output, got, tc.want)
		}
	}
}

func TestVersionsDiffer(t *testing.T) {
	// A tool can report the same release with different padding depending on
	// which command is asked: yt-dlp --version says 2026.08.19 while
	// uv tool list says v2026.8.19. Treating that as a difference runs an
	// update that changes nothing.
	cases := []struct {
		installed string
		latest    string
		want      bool
	}{
		{"1.55.1", "1.56.0", true},
		{"1.55.1", "v1.55.1", false},
		{"2.0.5", "2.0.5", false},
		{" 1.0.0 ", "1.0.0", false},
		{"2026.08.19", "2026.8.19", false},
		{"v1.2.3", "1.2.3", false},
		{"1.2.3", "1.2.3+build.5", false},
		{"1.2.3", "1.2.4", true},
		{"5.9.2", "5.4.2", true},
		{"1.2.3", "1.2.3.1", true},
		{"0.0.1", "0.1.0", true},
		{"1.10.0", "1.9.0", true},
	}
	for _, tc := range cases {
		if got := versionsDiffer(tc.installed, tc.latest); got != tc.want {
			t.Errorf("versionsDiffer(%q, %q) = %v, want %v", tc.installed, tc.latest, got, tc.want)
		}
	}
}

func TestVersionMajorAtLeast(t *testing.T) {
	cases := []struct {
		version string
		major   int
		want    bool
	}{
		{"2.0.16", 2, true},
		{"v2.0.16", 2, true},
		{"3.0.0", 2, true},
		{"1.18.32", 2, false},
		{"1.18.32", 1, true},
		{"", 2, false},
		{"not-a-version", 2, false},
	}
	for _, tc := range cases {
		if got := versionMajorAtLeast(tc.version, tc.major); got != tc.want {
			t.Errorf("versionMajorAtLeast(%q, %d) = %v, want %v", tc.version, tc.major, got, tc.want)
		}
	}
}

func TestClassifyInstallSource(t *testing.T) {
	home := "/home/user"
	cases := []struct {
		path string
		home string
		want string
	}{
		{filepath.Join(home, ".local", "share", "mise", "shims", "cmdc"), home, sourceMise},
		{filepath.Join(home, ".local", "bin", "opencode"), home, sourceEnvctl},
		{filepath.Join(home, ".opencode", "bin", "opencode"), home, sourceEnvctl},
		{"/usr/bin/opencode", home, sourceSystem},
		{"/usr/bin/opencode", "", sourceSystem},
	}
	for _, tc := range cases {
		if got := classifyInstallSource(tc.path, tc.home); got != tc.want {
			t.Errorf("classifyInstallSource(%q, %q) = %q, want %q", tc.path, tc.home, got, tc.want)
		}
	}
}

// The update path is chosen by comparing this token inside a switch; a display
// string that differs only by case silently sent npm tools down the "leave it
// alone" branch.
func TestSourceLabelIsDisplayOnly(t *testing.T) {
	home := "/home/user"
	miseShim := filepath.Join(home, ".local", "share", "mise", "shims", "cmdc")
	if got := classifyInstallSource(miseShim, home); got != sourceMise {
		t.Fatalf("mise path classified as %q, want %q", got, sourceMise)
	}
	if label := sourceLabel(sourceMise); label != "mise" {
		t.Errorf("sourceLabel(%q) = %q, want %q", sourceMise, label, "mise")
	}
	if label := sourceLabel(sourceSystem); label != "system package" {
		t.Errorf("sourceLabel(%q) = %q, want %q", sourceSystem, label, "system package")
	}
	for _, source := range []string{sourceMise, sourceEnvctl, sourceSystem} {
		if label := sourceLabel(source); label == "" {
			t.Errorf("sourceLabel(%q) is empty", source)
		}
	}
}

// mise shims nest under ~/.local, so the classifier must check the mise
// prefix before the envctl one or every shim reads as envctl-owned.
func TestClassifyInstallSourceMiseBeforeEnvctl(t *testing.T) {
	home := "/home/user"
	miseShim := filepath.Join(home, ".local", "share", "mise", "shims", "node")
	if got := classifyInstallSource(miseShim, home); got != sourceMise {
		t.Errorf("classifyInstallSource(%q) = %q, want %q (mise prefix must win over .local)", miseShim, got, sourceMise)
	}
	localBin := filepath.Join(home, ".local", "bin", "cmdc")
	if got := classifyInstallSource(localBin, home); got != sourceEnvctl {
		t.Errorf("classifyInstallSource(%q) = %q, want %q", localBin, got, sourceEnvctl)
	}
}

// Phase 0 probes must resolve on the toolchain PATH, not the process PATH:
// under ssh/systemd/agent non-login shells the mise shims dir is absent from
// the process PATH, and the old exec.LookPath probe reported npm tools as
// missing (reinstalling on every run).
func TestInstalledVersionResolvesMiseShim(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("mise shim layout under $HOME/.local/share/mise is POSIX-only")
	}
	tmp := t.TempDir()
	miseShims := filepath.Join(tmp, ".local", "share", "mise", "shims")
	if err := os.MkdirAll(miseShims, 0755); err != nil {
		t.Fatalf("MkdirAll mise shims failed: %v", err)
	}
	shim := filepath.Join(miseShims, "fakecli")
	if err := os.WriteFile(shim, []byte("#!/bin/sh\necho 'fakecli v9.9.9'\n"), 0755); err != nil {
		t.Fatalf("WriteFile shim failed: %v", err)
	}
	t.Setenv("HOME", tmp)
	t.Setenv("PATH", "/usr/bin:/bin")

	if got := installedVersion(context.Background(), "fakecli"); got != "9.9.9" {
		t.Errorf("installedVersion(fakecli) = %q, want %q (shim invisible on process PATH, visible on toolchain PATH)", got, "9.9.9")
	}
	if got := installSource("fakecli"); got != sourceMise {
		t.Errorf("installSource(fakecli) = %q, want %q", got, sourceMise)
	}
}

// Phase 0 must know how to reach every provider on every platform it claims to
// support: a missing install path would leave a fresh machine stuck.
func TestProviderCLIsAreInstallable(t *testing.T) {
	onWindows := runtime.GOOS == "windows"
	for _, tool := range providerCLIs() {
		if tool.name == "" || tool.binary == "" {
			t.Errorf("provider entry is incomplete: %+v", tool)
		}
		if tool.npmPkg == "" && tool.windowsInstaller == "" && tool.installer == "" {
			t.Errorf("%s has no install path at all", tool.name)
		}
		if onWindows && tool.npmPkg == "" && tool.windowsInstaller == "" {
			t.Errorf("%s has no Windows install path", tool.name)
		}
		if !onWindows && tool.npmPkg == "" && tool.installer == "" {
			t.Errorf("%s has no Linux install path", tool.name)
		}
	}
}

func TestOpenCodeProviderUsesV2Installer(t *testing.T) {
	for _, tool := range providerCLIs() {
		if tool.binary != "opencode" {
			continue
		}
		if !strings.Contains(tool.installer, "https://opencode.ai/v2/install") {
			t.Fatalf("OpenCode Linux installer = %q, want official v2 endpoint", tool.installer)
		}
		if strings.Contains(tool.installer, "https://opencode.ai/install") {
			t.Fatalf("OpenCode Linux installer still uses the v1 endpoint: %q", tool.installer)
		}
		if tool.requiredMajor != 2 {
			t.Fatalf("OpenCode requiredMajor = %d, want 2", tool.requiredMajor)
		}
		return
	}
	t.Fatal("OpenCode provider is missing from providerCLIs()")
}

func TestPacmanOwnsOpenCodeUsesPackageDatabase(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("pacman ownership is Linux-only")
	}
	uc := &ProvisionProvidersUseCase{managers: map[entity.PackageType]repository.PackageManager{
		entity.PackageTypePacman: &mockGamingPackageManager{
			available: true,
			installed: map[string]string{"opencode": "2.0.16-1"},
		},
	}}
	if !uc.pacmanOwnsOpenCode(context.Background()) {
		t.Fatal("pacmanOwnsOpenCode = false, want true for an installed package")
	}
	uc.managers[entity.PackageTypePacman] = &mockGamingPackageManager{available: true, installed: map[string]string{}}
	if uc.pacmanOwnsOpenCode(context.Background()) {
		t.Fatal("pacmanOwnsOpenCode = true, want false when the package is absent")
	}
}

func TestFilesystemArchivePathMovesLiveAside(t *testing.T) {
	path := filepath.Join(t.TempDir(), "opencode")
	if err := os.WriteFile(path, []byte("v1"), 0755); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	backup, err := filesystem.ArchivePath(path)
	if err != nil {
		t.Fatalf("filesystem.ArchivePath: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("live path still exists after archive: %v", err)
	}
	data, err := os.ReadFile(backup)
	if err != nil || string(data) != "v1" {
		t.Fatalf("backup = %q, err = %v, want original content", data, err)
	}
}

func TestStandaloneProviderCanReplace(t *testing.T) {
	cases := []struct {
		name       string
		source     string
		pacmanOwns bool
		want       bool
	}{
		{"user-local V1 on Ubuntu", sourceEnvctl, false, true},
		{"system V1 on Ubuntu", sourceSystem, false, true},
		{"system V1 owned by pacman", sourceSystem, true, false},
		{"user-local V1 while pacman owns the package", sourceEnvctl, true, false},
	}
	for _, tc := range cases {
		if got := standaloneProviderCanReplace(tc.source, tc.pacmanOwns); got != tc.want {
			t.Errorf("%s: standaloneProviderCanReplace(%q, %v) = %v, want %v", tc.name, tc.source, tc.pacmanOwns, got, tc.want)
		}
	}
}

func TestOpenCodeWithinMajorUpdateNeeded(t *testing.T) {
	cases := []struct {
		name      string
		installed string
		latest    string
		major     int
		want      bool
	}{
		{"same minor", "2.0.23", "2.0.23", 2, false},
		{"older minor needs update", "2.0.15", "2.0.23", 2, true},
		{"older patch, same minor", "2.0.15", "2.0.20", 2, true},
		{"installed newer minor", "2.1.0", "2.0.23", 2, false},
		{"different major never downgrades", "3.0.0", "2.0.23", 2, false},
		{"installed below required major", "1.18.32", "2.0.23", 2, false},
		{"leading v tolerated", "v2.0.15", "v2.0.23", 2, true},
		{"unparseable installed", "not-a-version", "2.0.23", 2, false},
		{"unparseable latest", "2.0.15", "latest", 2, false},
	}
	for _, tc := range cases {
		if got := openCodeWithinMajorUpdateNeeded(tc.installed, tc.latest, tc.major); got != tc.want {
			t.Errorf("%s: openCodeWithinMajorUpdateNeeded(%q, %q, %d) = %v, want %v",
				tc.name, tc.installed, tc.latest, tc.major, got, tc.want)
		}
	}
}

func TestParseSemver(t *testing.T) {
	cases := []struct {
		version string
		major   int
		minor   int
		patch   int
		ok      bool
	}{
		{"2.0.23", 2, 0, 23, true},
		{"v2.15.0", 2, 15, 0, true},
		{" 1.18.32 ", 1, 18, 32, true},
		{"2.0.23-rc1", 2, 0, 23, true},
		{"3", 0, 0, 0, false},
		{"2.0", 0, 0, 0, false},
		{"", 0, 0, 0, false},
		{"no.version", 0, 0, 0, false},
	}
	for _, tc := range cases {
		major, minor, patch, ok := parseSemver(tc.version)
		if major != tc.major || minor != tc.minor || patch != tc.patch || ok != tc.ok {
			t.Errorf("parseSemver(%q) = (%d, %d, %d, %v), want (%d, %d, %d, %v)",
				tc.version, major, minor, patch, ok, tc.major, tc.minor, tc.patch, tc.ok)
		}
	}
}
