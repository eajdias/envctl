package embedded

import (
	"path"
	"testing"

	"github.com/eajdias/envctl"
)

// TestConfigFileInstancesExpansion pins the instances: expansion that keeps
// the six delegating git hooks on a single shim script: one manifest entry
// must load as one deployment per hook name, with {{name}} substituted in
// id, source and destination — and every other entry untouched.
func TestConfigFileInstancesExpansion(t *testing.T) {
	repo := NewManifestRepository(envctl.EmbeddedFS, ".")

	configs, err := repo.LoadConfigFiles()
	if err != nil {
		t.Fatalf("failed to load config files: %v", err)
	}

	wantHooks := []string{"pre-commit", "prepare-commit-msg", "commit-msg", "post-commit", "post-checkout", "pre-rebase"}
	got := map[string]bool{}
	for _, cf := range configs {
		if cf.Source == "configs/git/hooks/{{name}}" || path.Base(cf.Destination) == "{{name}}" {
			t.Fatalf("unexpanded {{name}} placeholder survived loading: %+v", cf)
		}
		if cf.Instances != nil {
			t.Fatalf("instances must be nil after expansion: %+v", cf)
		}
		for _, hook := range wantHooks {
			if cf.Destination == "~/.config/git/hooks/"+hook {
				got[hook] = true
				if cf.Source != "configs/git/hooks/_envctl-shim" {
					t.Errorf("hook %q must deploy the shared shim, got source %q", hook, cf.Source)
				}
				if cf.ID != "git_hook_"+hook {
					t.Errorf("hook %q has id %q, want git_hook_%s", hook, cf.ID, hook)
				}
				if !cf.Executable {
					t.Errorf("hook %q must stay executable", hook)
				}
			}
		}
	}
	for _, hook := range wantHooks {
		if !got[hook] {
			t.Errorf("hook %q was not expanded from the instances entry", hook)
		}
	}
	if len(got) != len(wantHooks) {
		t.Errorf("expanded hook deployments = %d, want %d", len(got), len(wantHooks))
	}
}
