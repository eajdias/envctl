package embedded

import (
	"testing"

	win11new "github.com/eajdias/envctl"
)

func TestLoadManifestsFromDiskOrEmbed(t *testing.T) {
	repo := NewManifestRepository(win11new.EmbeddedFS, ".")

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

	tweaks, err := repo.LoadWindowsTweaks()
	if err != nil {
		t.Fatalf("failed to load windows tweaks manifest: %v", err)
	}

	if len(tweaks) == 0 {
		t.Errorf("expected windows tweaks to be non-empty")
	}
}
