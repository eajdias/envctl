package embedded

import (
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

	if len(skills) != 74 {
		t.Errorf("expected exactly 74 skills in manifest, got %d", len(skills))
	}

	if len(lsps) != 18 {
		t.Errorf("expected exactly 18 LSPs in manifest, got %d", len(lsps))
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
