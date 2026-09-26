package embedded

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eajdias/envctl"
	"github.com/eajdias/envctl/internal/domain/entity"
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

	ubuntu, err := repo.LoadPerformanceSpec(entity.PerformanceProfileUbuntuServer)
	if err != nil {
		t.Fatalf("failed to load Ubuntu performance manifest: %v", err)
	}
	if ubuntu.Profile != entity.PerformanceProfileUbuntuServer {
		t.Fatalf("Ubuntu performance spec profile = %q", ubuntu.Profile)
	}
	// Assert the required entries are present rather than counting packages, so
	// adding a justified package does not break an unrelated contract.
	requiredUbuntu := []string{"systemd-zram-generator", "tzdata"}
	present := make(map[string]bool, len(ubuntu.Packages))
	for _, pkg := range ubuntu.Packages {
		present[pkg.ID] = true
	}
	for _, id := range requiredUbuntu {
		if !present[id] {
			t.Fatalf("Ubuntu performance spec is missing package %q; present: %v", id, present)
		}
	}
	if len(ubuntu.Sysctls) == 0 {
		t.Fatal("Ubuntu performance spec must contain sysctl settings")
	}
	// Memory-scoped sysctls belong to the tiers, not to the profile base: the
	// base is what every memory size gets, and vfs_cache_pressure is a memory
	// policy that each tier sets for itself.
	for _, setting := range ubuntu.Sysctls {
		if setting.Key == "vm.vfs_cache_pressure" {
			t.Fatal("vm.vfs_cache_pressure must be declared per tier, not in the profile base")
		}
	}
	if ubuntu.Timezone == nil || ubuntu.Timezone.Expected != "Etc/UTC" {
		t.Fatalf("Ubuntu performance spec timezone = %#v, want the fleet's Etc/UTC", ubuntu.Timezone)
	}
	if ubuntu.MinDistroVersion != "24.04" {
		t.Fatalf("Ubuntu performance spec minimum = %q, want the manifest-declared 24.04", ubuntu.MinDistroVersion)
	}
	if len(ubuntu.Tiers) != 4 {
		t.Fatalf("Ubuntu performance spec tiers = %d, want 4", len(ubuntu.Tiers))
	}
	if ubuntu.Tiers[0].ID != "tiny" || ubuntu.Tiers[0].MatchMemTotalMax != 1536 {
		t.Fatalf("first tier = %#v, want tiny up to 1536 MiB", ubuntu.Tiers[0])
	}
	if ubuntu.Tiers[len(ubuntu.Tiers)-1].MatchMemTotalMax != 0 {
		t.Fatalf("last tier must be unbounded, got %#v", ubuntu.Tiers[len(ubuntu.Tiers)-1])
	}
	// The fleet already ships fs.file-max at the int64 ceiling, so a plain
	// write would regress it. The manifest must declare it as a floor.
	var fileMax *entity.SysctlSetting
	for i, setting := range ubuntu.Sysctls {
		if setting.Key == "fs.file-max" {
			fileMax = &ubuntu.Sysctls[i]
		}
	}
	if fileMax == nil {
		t.Fatal("Ubuntu performance spec must declare fs.file-max")
	}
	if fileMax.Policy != entity.SysctlPolicyMin {
		t.Fatalf("fs.file-max policy = %q, want min so the host ceiling is never lowered", fileMax.Policy)
	}
	// vm.swappiness must NOT be declared: it is derived from the measured swap
	// topology at run time, and pinning it is what produced the original defect
	// of shipping zram together with a value of 10.
	for _, setting := range ubuntu.Sysctls {
		if setting.Key == "vm.swappiness" {
			t.Fatal("vm.swappiness must be derived from the swap topology, not declared in the manifest")
		}
	}
	if ubuntu.ZRAM == nil || ubuntu.ZRAM.Policy != entity.ZRAMPolicyTier {
		t.Fatalf("zram policy = %#v, want tier", ubuntu.ZRAM)
	}
	for _, tier := range ubuntu.Tiers {
		if !tier.ZRAMEnabled() && tier.ID == "tiny" {
			t.Fatal("the tiny tier must enable zram: the fleet's memory-constrained hosts need it")
		}
	}

	cachyos, err := repo.LoadPerformanceSpec(entity.PerformanceProfileCachyOS)
	if err != nil {
		t.Fatalf("failed to load CachyOS performance manifest: %v", err)
	}
	if cachyos.Profile != entity.PerformanceProfileCachyOS || len(cachyos.Packages) != 1 || cachyos.Packages[0].ID != "zram-generator" {
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
	if _, err := repo.LoadPerformanceSpec(entity.PerformanceProfileCachyOS); err == nil {
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
	if _, err := repo.LoadPerformanceSpec(entity.PerformanceProfileCachyOS); err == nil {
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

// TestListPerformanceProfilesReadsTheMinimumFromDisk keeps the release floor in
// the manifest. A Go constant that encodes "24.04" is a lie once the fleet runs
// 26.04, so discovery must report the declared minimum.
func TestListPerformanceProfilesReadsTheMinimumFromDisk(t *testing.T) {
	repo := NewManifestRepository(envctl.EmbeddedFS, ".")

	metas, err := repo.ListPerformanceProfiles()
	if err != nil {
		t.Fatalf("ListPerformanceProfiles failed: %v", err)
	}
	if len(metas) != 2 {
		t.Fatalf("performance profiles = %#v, want exactly two", metas)
	}

	byProfile := make(map[entity.PerformanceProfile]entity.PerformanceProfileMeta, len(metas))
	for _, meta := range metas {
		if meta.ManifestFile == "" {
			t.Fatalf("meta %#v does not name its manifest file", meta)
		}
		byProfile[meta.Profile] = meta
	}

	ubuntu, ok := byProfile[entity.PerformanceProfileUbuntuServer]
	if !ok {
		t.Fatalf("ubuntu-server profile is not discoverable: %#v", metas)
	}
	if ubuntu.MinDistroVersion != "24.04" {
		t.Fatalf("ubuntu-server minimum = %q, want 24.04", ubuntu.MinDistroVersion)
	}
	if ubuntu.ManifestFile != "performance_ubuntu.yaml" {
		t.Fatalf("ubuntu-server manifest file = %q", ubuntu.ManifestFile)
	}

	cachyos, ok := byProfile[entity.PerformanceProfileCachyOS]
	if !ok {
		t.Fatalf("cachyos profile is not discoverable: %#v", metas)
	}
	if cachyos.MinDistroVersion != "" {
		t.Fatalf("cachyos minimum = %q, want empty for a rolling release", cachyos.MinDistroVersion)
	}
}

// TestPerformanceManifestsDoNotDeclareAVersionedProfileName is the lint that
// keeps the old identity from coming back through a manifest edit.
func readManifestFixture(t *testing.T, filename string) (string, error) {
	t.Helper()
	data, err := fs.ReadFile(envctl.EmbeddedFS, "manifests/"+filename)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func TestPerformanceManifestsDoNotDeclareAVersionedProfileName(t *testing.T) {
	for _, filename := range []string{"performance_ubuntu.yaml", "performance_cachyos.yaml"} {
		data, err := readManifestFixture(t, filename)
		if err != nil {
			t.Fatalf("read %s: %v", filename, err)
		}
		if strings.Contains(data, "ubuntu-24.04") {
			t.Fatalf("%s still declares the versioned profile identity ubuntu-24.04", filename)
		}
	}
}
