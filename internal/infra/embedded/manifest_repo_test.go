package embedded

import (
	"io/fs"
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

	const expectedSkills = 41
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

	const expectedLSPs = 16
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
}
