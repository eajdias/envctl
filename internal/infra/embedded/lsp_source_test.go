package embedded

import (
	"testing"

	"github.com/eajdias/envctl"
)

// TestLSPSingleSource pins manifests/lsp.yaml as the only language-server
// source: no packages.yaml entry may carry the lsp category (a server added
// to one manifest and forgotten in the other used to mean a green doctor
// with a broken runtime), and every lsp.yaml entry must be installable on
// its own (install target + check binary present).
func TestLSPSingleSource(t *testing.T) {
	repo := NewManifestRepository(envctl.EmbeddedFS, ".")

	pkgs, err := repo.LoadPackages()
	if err != nil {
		t.Fatalf("failed to load packages manifest: %v", err)
	}
	for _, p := range pkgs {
		if p.Category == "lsp" {
			t.Errorf("packages.yaml entry %q carries category lsp: language servers live in manifests/lsp.yaml only", p.ID)
		}
	}

	lsps, err := repo.LoadLSPs()
	if err != nil {
		t.Fatalf("failed to load lsp manifest: %v", err)
	}
	if len(lsps) == 0 {
		t.Fatal("lsp.yaml declares no language servers")
	}
	byTarget := map[string]bool{}
	for _, l := range lsps {
		if l.InstallTarget == "" {
			t.Errorf("lsp %q has no install_target", l.ID)
		}
		if l.CheckBinary == "" {
			t.Errorf("lsp %q has no check_binary", l.ID)
		}
		byTarget[l.InstallTarget] = true
	}

	// The six mise-backed (npm:) servers formerly duplicated in packages.yaml
	// must resolve through `run lsp` alone.
	for _, target := range []string{
		"npm:typescript-language-server",
		"npm:pyright",
		"npm:bash-language-server",
		"npm:yaml-language-server",
		"npm:dockerfile-language-server-nodejs",
		"npm:vscode-langservers-extracted",
	} {
		if !byTarget[target] {
			t.Errorf("install_target %q missing from lsp.yaml: `run lsp` would not install it", target)
		}
	}
}
