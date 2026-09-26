package embedded

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/eajdias/envctl"
)

func TestLoadManifestsFromDiskOrEmbed(t *testing.T) {
	repo := NewManifestRepository(envctl.EmbeddedFS, ".")

	pkgs, err := repo.LoadPackages()
	if err != nil {
		t.Fatalf("failed to load packages manifest: %v", err)
	}

	if len(pkgs) == 0 {
		t.Errorf("expected packages to be non-empty")
	}

	gitConfigs, err := repo.LoadGitConfigs()
	if err != nil {
		t.Fatalf("failed to load git configs: %v", err)
	}

	if len(gitConfigs) == 0 {
		t.Errorf("expected global git configs to be non-empty")
	}

	configFiles, err := repo.LoadConfigFiles()
	if err != nil {
		t.Fatalf("failed to load config files: %v", err)
	}

	if len(configFiles) == 0 {
		t.Errorf("expected config files to be non-empty")
	}

	lsps, err := repo.LoadLSPs()
	if err != nil {
		t.Fatalf("failed to load lsp manifest: %v", err)
	}

	if len(lsps) == 0 {
		t.Errorf("expected lsp servers to be non-empty")
	}

	skills, err := repo.LoadSkills()
	if err != nil {
		t.Fatalf("failed to load skills manifest: %v", err)
	}

	if len(skills) == 0 {
		t.Errorf("expected skills to be non-empty")
	}

	const expectedSkills = 12
	if len(skills) != expectedSkills {
		t.Errorf("expected exactly %d skills in manifest, got %d", expectedSkills, len(skills))
	}

	// The manifest and the embedded skill directories must agree: a skill that
	// is embedded but undeclared never gets deployed, and one that is declared
	// without a directory breaks provisioning.
	entries, readErr := fs.ReadDir(envctl.EmbeddedFS, "configs/skills")
	if readErr != nil {
		t.Fatalf("failed to read embedded skill directories: %v", readErr)
	}
	declared := map[string]bool{}
	for _, s := range skills {
		declared[s.Name] = true
	}
	shipped := map[string]bool{}
	for _, entry := range entries {
		if entry.IsDir() {
			shipped[entry.Name()] = true
		}
	}
	for name := range shipped {
		if !declared[name] {
			t.Errorf("skill directory %q is embedded but missing from manifests/skills.yaml", name)
		}
	}
	for name := range declared {
		if !shipped[name] {
			t.Errorf("skill %q is declared in manifests/skills.yaml but has no embedded directory", name)
		}
	}
	if len(shipped) != expectedSkills {
		t.Errorf("expected exactly %d embedded skill directories, got %d", expectedSkills, len(shipped))
	}

	const expectedLSPs = 15
	if len(lsps) != expectedLSPs {
		t.Errorf("expected exactly %d LSPs in manifest, got %d", expectedLSPs, len(lsps))
	}

	// Verify Google Chrome is present and Brave Nightly is absent
	chromeFound := false
	braveFound := false
	for _, p := range pkgs {
		if p.ID == "Google.Chrome" {
			chromeFound = true
		}
		if p.ID == "Brave.Brave.Nightly" {
			braveFound = true
		}
	}
	if !chromeFound {
		t.Errorf("expected Google.Chrome to be in packages manifest")
	}
	if braveFound {
		t.Errorf("expected Brave.Brave.Nightly to NOT be in packages manifest")
	}

	// Verify BRAVE_NIGHTLY_PATH is absent from env vars
	envVars, err := repo.LoadEnvVars()
	if err != nil {
		t.Fatalf("failed to load env vars: %v", err)
	}
	for _, ev := range envVars {
		if ev.Name == "BRAVE_NIGHTLY_PATH" {
			t.Errorf("expected BRAVE_NIGHTLY_PATH to NOT be in env vars")
		}
	}

	tweaks, err := repo.LoadWindowsTweaks()
	if err != nil {
		t.Fatalf("failed to load windows tweaks manifest: %v", err)
	}

	if len(tweaks) == 0 {
		t.Errorf("expected windows tweaks to be non-empty")
	}

	debloat, err := repo.LoadDebloatTweaks()
	if err != nil {
		t.Fatalf("failed to load debloat manifest: %v", err)
	}

	const expectedDebloat = 76
	if len(debloat) != expectedDebloat {
		t.Errorf("expected exactly %d debloat tweaks, got %d", expectedDebloat, len(debloat))
	}

	validTypes := map[string]bool{"DWord": true, "String": true, "Appx": true, "Service": true}
	validCats := map[string]bool{"telemetry": true, "privacy": true, "gaming": true, "apps": true, "services": true}
	seen := map[string]bool{}
	for _, tw := range debloat {
		if seen[tw.ID] {
			t.Errorf("duplicate debloat tweak id %q", tw.ID)
		}
		seen[tw.ID] = true
		if !validTypes[tw.Type] {
			t.Errorf("debloat tweak %q has unsupported type %q (want DWord/String/Appx/Service)", tw.ID, tw.Type)
		}
		if !validCats[tw.Category] {
			t.Errorf("debloat tweak %q has unknown category %q", tw.ID, tw.Category)
		}
		switch tw.Type {
		case "Appx":
			if tw.Path != "" {
				t.Errorf("debloat Appx tweak %q must have empty path, got %q", tw.ID, tw.Path)
			}
			if tw.Name == "" {
				t.Errorf("debloat Appx tweak %q must name a package", tw.ID)
			}
		case "Service":
			if s, ok := tw.Value.(string); !ok || (s != "Disabled" && s != "Manual") {
				t.Errorf("debloat Service tweak %q must declare Disabled/Manual, got %v", tw.ID, tw.Value)
			}
		default: // registry
			if tw.Path == "" || tw.Name == "" {
				t.Errorf("debloat registry tweak %q needs path and name", tw.ID)
			}
		}
	}

	// Regression: windows11-clean declared KeyboardDelay as DWord, but the
	// value is REG_SZ upstream — a DWord write would corrupt the type.
	for _, tw := range debloat {
		if tw.ID == "gaming-win-keyboard-delay" && tw.Type != "String" {
			t.Errorf("gaming-win-keyboard-delay must be String (REG_SZ), got %q", tw.Type)
		}
	}
}

func TestPerformanceManifestsAreSeparateByProfile(t *testing.T) {
	repo := NewManifestRepository(envctl.EmbeddedFS, ".")

	ubuntu, err := repo.LoadPerformanceSpec("ubuntu-24.04")
	if err != nil {
		t.Fatalf("failed to load Ubuntu performance manifest: %v", err)
	}
	if ubuntu.Profile != "ubuntu-24.04" || len(ubuntu.Packages) != 1 || ubuntu.Packages[0].ID != "systemd-zram-generator" {
		t.Fatalf("unexpected Ubuntu performance spec: %#v", ubuntu)
	}
	if len(ubuntu.Sysctls) == 0 {
		t.Fatal("Ubuntu performance spec must contain sysctl settings")
	}

	cachyos, err := repo.LoadPerformanceSpec("cachyos")
	if err != nil {
		t.Fatalf("failed to load CachyOS performance manifest: %v", err)
	}
	if cachyos.Profile != "cachyos" || len(cachyos.Packages) != 1 || cachyos.Packages[0].ID != "zram-generator" {
		t.Fatalf("unexpected CachyOS performance spec: %#v", cachyos)
	}
	if len(cachyos.Sysctls) != 0 {
		t.Fatal("CachyOS performance spec must not mutate sysctls in this branch")
	}

	if _, err := repo.LoadPerformanceSpec("debian"); err == nil {
		t.Fatal("expected unsupported performance profile to be rejected")
	}
}

func TestUnreadableLocalPerformanceManifestDoesNotFallBack(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "manifests", "performance_cachyos.yaml"), 0755); err != nil {
		t.Fatal(err)
	}

	repo := NewManifestRepository(envctl.EmbeddedFS, dir)
	if _, err := repo.LoadPerformanceSpec("cachyos"); err == nil {
		t.Fatal("expected unreadable local manifest to fail closed")
	}
}

func TestLocalCachyPerformanceManifestCannotInjectSysctls(t *testing.T) {
	dir := t.TempDir()
	manifestDir := filepath.Join(dir, "manifests")
	if err := os.MkdirAll(manifestDir, 0755); err != nil {
		t.Fatal(err)
	}
	manifest := []byte("profile: cachyos\npackages: []\nsysctls:\n  - key: vm.swappiness\n    value: \"10\"\n")
	if err := os.WriteFile(filepath.Join(manifestDir, "performance_cachyos.yaml"), manifest, 0644); err != nil {
		t.Fatal(err)
	}

	repo := NewManifestRepository(envctl.EmbeddedFS, dir)
	if _, err := repo.LoadPerformanceSpec("cachyos"); err == nil {
		t.Fatal("expected local CachyOS sysctl injection to be rejected")
	}
}

func TestUbuntuPerformanceToolboxManifest(t *testing.T) {
	repo := NewManifestRepository(envctl.EmbeddedFS, ".")
	packages, err := repo.LoadPackages()
	if err != nil {
		t.Fatalf("failed to load packages manifest: %v", err)
	}

	required := map[string]bool{
		"eza":         false,
		"tmux":        false,
		"sqlite3":     false,
		"restic":      false,
		"rclone":      false,
		"btop":        false,
		"duf":         false,
		"glances":     false,
		"micro":       false,
		"cmake":       false,
		"ninja-build": false,
		"mosh":        false,
		"nvtop":       false,
		"iotop":       false,
		"sysstat":     false,
		"zstd":        false,
		"lz4":         false,
	}

	for _, pkg := range packages {
		if _, ok := required[pkg.ID]; !ok {
			continue
		}
		if pkg.Type != "apt" || pkg.TargetDistro != "ubuntu" {
			continue
		}
		if pkg.OS != "ubuntu" || pkg.MinDistroVersion != "24.04" {
			t.Errorf("Ubuntu performance package %q has wrong scope: os=%q target=%q min=%q", pkg.ID, pkg.OS, pkg.TargetDistro, pkg.MinDistroVersion)
		}
		required[pkg.ID] = true
	}

	for id, found := range required {
		if !found {
			t.Errorf("missing Ubuntu 24.04 performance package %q", id)
		}
	}
}
