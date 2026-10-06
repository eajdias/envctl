package embedded

import (
	"testing"

	"github.com/eajdias/envctl"
	"github.com/eajdias/envctl/internal/domain/entity"
)

// TestOSValuesExpandToTheFormerPerOSEntries pins the os_values expansion to
// the exact entries the manifest used to declare as near-identical pairs:
// same names, values, paths and os filters as before the consolidation.
func TestOSValuesExpandToTheFormerPerOSEntries(t *testing.T) {
	repo := NewManifestRepository(envctl.EmbeddedFS, ".")

	vars, err := repo.LoadEnvVars()
	if err != nil {
		t.Fatalf("LoadEnvVars: %v", err)
	}
	wantVars := map[string]map[string]entity.EnvironmentVar{
		"NODE_PATH": {
			"windows":                    {Name: "NODE_PATH", Value: `%USERPROFILE%\node_modules`, Scope: "User", Target: "Windows User Environment", OS: "windows"},
			"arch,cachyos,debian,ubuntu": {Name: "NODE_PATH", Value: "$HOME/node_modules", Scope: "User", Target: "Linux user profile", OS: "arch,cachyos,debian,ubuntu"},
		},
		"ENVCTL_TEMP": {
			"windows":                    {Name: "ENVCTL_TEMP", Value: `C:\temp`, Scope: "User", Target: "Windows User Environment", OS: "windows"},
			"arch,cachyos,debian,ubuntu": {Name: "ENVCTL_TEMP", Value: "/temp", Scope: "User", Target: "Linux user profile", OS: "arch,cachyos,debian,ubuntu"},
		},
	}
	for _, v := range vars {
		byOS, ok := wantVars[v.Name]
		if !ok {
			continue // entries outside this contract
		}
		want, ok := byOS[v.OS]
		if !ok {
			t.Errorf("variable %q has unexpected os filter %q", v.Name, v.OS)
			continue
		}
		if v.Value != want.Value || v.Scope != want.Scope || v.Target != want.Target {
			t.Errorf("variable %q on %q expanded to value=%q scope=%q target=%q, want value=%q scope=%q target=%q",
				v.Name, v.OS, v.Value, v.Scope, v.Target, want.Value, want.Scope, want.Target)
		}
		if v.OSValues != nil {
			t.Errorf("variable %q must not leak os_values after expansion", v.Name)
		}
	}
	for name, byOS := range wantVars {
		for osFilter := range byOS {
			found := false
			for _, v := range vars {
				if v.Name == name && v.OS == osFilter {
					found = true
				}
			}
			if !found {
				t.Errorf("variable %q on %q was not expanded from os_values", name, osFilter)
			}
		}
	}

	dirs, err := repo.LoadDirectories()
	if err != nil {
		t.Fatalf("LoadDirectories: %v", err)
	}
	dirPaths := map[string]string{}
	for _, d := range dirs {
		if d.Description == "Global standardized temp folder for OpenCode LLM agents (ENVCTL_TEMP)" {
			dirPaths[d.OS] = d.Path
		}
	}
	if dirPaths["windows"] != "C:/temp" || dirPaths["arch,cachyos,debian,ubuntu"] != "/temp" {
		t.Errorf("temp dir expansion = %v, want C:/temp on windows and /temp on linux", dirPaths)
	}

	items, err := repo.LoadCleanupItems()
	if err != nil {
		t.Fatalf("LoadCleanupItems: %v", err)
	}
	wantItems := map[string]struct {
		path string
		os   string
	}{
		"stale_pylsp_binary":      {"~/.local/bin/pylsp", ""},
		"stale_pylsp_cmd":         {"~/.local/bin/pylsp.cmd", "windows"},
		"stale_pw_screenshot":     {"~/.local/bin/pw-screenshot", ""},
		"stale_pw_screenshot_cmd": {"~/.local/bin/pw-screenshot.cmd", "windows"},
		"stale_pw_eval":           {"~/.local/bin/pw-eval", ""},
		"stale_pw_eval_cmd":       {"~/.local/bin/pw-eval.cmd", "windows"},
	}
	gotItems := map[string]bool{}
	for _, item := range items {
		want, ok := wantItems[item.ID]
		if !ok {
			continue // entries outside this contract
		}
		gotItems[item.ID] = true
		if item.Path != want.path || item.OS != want.os {
			t.Errorf("cleanup %q = path %q os %q, want path %q os %q", item.ID, item.Path, item.OS, want.path, want.os)
		}
		if item.Category != "scripts" {
			t.Errorf("cleanup %q must keep category scripts, got %q", item.ID, item.Category)
		}
		if item.OSValues != nil {
			t.Errorf("cleanup %q must not leak os_values after expansion", item.ID)
		}
	}
	for id := range wantItems {
		if !gotItems[id] {
			t.Errorf("cleanup %q was not expanded from os_values", id)
		}
	}

	configs, err := repo.LoadConfigFiles()
	if err != nil {
		t.Fatalf("LoadConfigFiles: %v", err)
	}
	wantConfigs := map[string]entity.ConfigFile{
		"agents_manifest": {
			ID: "agents_manifest", Source: "configs/AGENTS.md",
			Destination: "~/.config/opencode/AGENTS.md", OS: "windows", Category: "opencode",
		},
		"agents_manifest_linux": {
			ID: "agents_manifest_linux", Source: "configs/AGENTS.linux.md",
			Destination: "~/.config/opencode/AGENTS.md", OS: "debian,ubuntu", Category: "opencode",
		},
		"agents_manifest_arch": {
			ID: "agents_manifest_arch", Source: "configs/AGENTS.arch.md",
			Destination: "~/.config/opencode/AGENTS.md", OS: "arch,cachyos", Category: "opencode",
		},
		"commandcode_agents_manifest": {
			ID: "commandcode_agents_manifest", Source: "configs/commandcode/AGENTS.md",
			Destination: "~/.commandcode/AGENTS.md", OS: "windows", Category: "commandcode",
			Merge: entity.MergeMarkdownSections,
		},
		"commandcode_agents_manifest_linux": {
			ID: "commandcode_agents_manifest_linux", Source: "configs/commandcode/AGENTS.linux.md",
			Destination: "~/.commandcode/AGENTS.md", OS: "debian,ubuntu", Category: "commandcode",
			Merge: entity.MergeMarkdownSections,
		},
		"commandcode_agents_manifest_arch": {
			ID: "commandcode_agents_manifest_arch", Source: "configs/commandcode/AGENTS.arch.md",
			Destination: "~/.commandcode/AGENTS.md", OS: "arch,cachyos", Category: "commandcode",
			Merge: entity.MergeMarkdownSections,
		},
		"pw_wrapper_cmd": {
			ID: "pw_wrapper_cmd", Source: "configs/bin/pw.cmd",
			Destination: "~/.local/bin/pw.cmd", OS: "windows",
		},
		"pw_wrapper_sh": {
			ID: "pw_wrapper_sh", Source: "configs/bin/pw",
			Destination: "~/.local/bin/pw", OS: "arch,cachyos,debian,ubuntu", Executable: true,
		},
	}
	gotConfigs := map[string]bool{}
	for _, cf := range configs {
		want, ok := wantConfigs[cf.ID]
		if !ok {
			continue
		}
		gotConfigs[cf.ID] = true
		if cf.Source != want.Source || cf.Destination != want.Destination || cf.OS != want.OS ||
			cf.Category != want.Category || cf.Merge != want.Merge || cf.Executable != want.Executable {
			t.Errorf("config %q = source %q dest %q os %q cat %q merge %q exec %v, want source %q dest %q os %q cat %q merge %q exec %v",
				cf.ID, cf.Source, cf.Destination, cf.OS, cf.Category, cf.Merge, cf.Executable,
				want.Source, want.Destination, want.OS, want.Category, want.Merge, want.Executable)
		}
		if cf.OSValues != nil {
			t.Errorf("config %q must not leak os_values after expansion", cf.ID)
		}
	}
	for id := range wantConfigs {
		if !gotConfigs[id] {
			t.Errorf("config %q was not expanded from os_values", id)
		}
	}
}

// TestOSValuesRejectUnknownFields fails the load on a typo'd override key
// instead of silently overriding nothing.
func TestOSValuesRejectUnknownFields(t *testing.T) {
	vars := []entity.EnvironmentVar{{
		Name: "BROKEN",
		OSValues: map[string]map[string]string{
			"windows": {"valeu": "typo"},
		},
	}}
	if _, err := expandEnvVarOSValues(vars); err == nil {
		t.Fatal("expected unknown os_values field to fail the expansion")
	}
	dirs := []entity.RestrictedDir{{
		Path: "~/x",
		OSValues: map[string]map[string]string{
			"windows": {"paths": "typo"},
		},
	}}
	if _, err := expandDirOSValues(dirs); err == nil {
		t.Fatal("expected unknown os_values field to fail the expansion")
	}
	items := []entity.CleanupItem{{
		ID: "broken",
		OSValues: map[string]map[string]string{
			"windows": {"pth": "typo"},
		},
	}}
	if _, err := expandCleanupOSValues(items); err == nil {
		t.Fatal("expected unknown os_values field to fail the expansion")
	}
	configs := []entity.ConfigFile{{
		ID: "broken",
		OSValues: map[string]map[string]string{
			"windows": {"src": "typo"},
		},
	}}
	if _, err := expandConfigFileOSValues(configs); err == nil {
		t.Fatal("expected unknown os_values field to fail the expansion")
	}
}

func TestOSValuesRejectInvalidExecutable(t *testing.T) {
	configs := []entity.ConfigFile{{
		ID: "broken",
		OSValues: map[string]map[string]string{
			"windows": {"executable": "yes"},
		},
	}}
	if _, err := expandConfigFileOSValues(configs); err == nil {
		t.Fatal("expected invalid executable os_values to fail the expansion")
	}
}
