package toolchain

import (
	"testing"

	"github.com/eajdias/envctl/internal/domain/entity"
)

func TestMisePackageManager_Type(t *testing.T) {
	mm := NewMiseManager()
	if mm == nil {
		t.Fatal("Expected MiseManager to not be nil")
	}

	if mm.Type() != entity.PackageTypeMise {
		t.Errorf("Expected package manager type %s, got %s", entity.PackageTypeMise, mm.Type())
	}
}

// Fixture shape per mise official docs (`mise ls --json`: object keyed by
// tool name; `mise ls <tool> --json`: array of records with
// version/installed/active/install_path) — live-verified in M9.
func TestMiseLsRecordsInstalled(t *testing.T) {
	withNode := `[{"version":"24.19.0","installed":true,"active":true,"install_path":"/home/u/.local/share/mise/installs/node/24.19.0"}]`
	notInstalled := `[{"version":"24.19.0","installed":false,"active":true}]`
	orphanShim := `[{"version":"3.8.5","installed":true,"active":false,"install_path":"C:\\Users\\u\\AppData\\Local\\mise\\installs\\npm-mcp-ssh-manager\\3.8.5"}]`
	empty := `[]`

	tests := []struct {
		name string
		out  string
		want bool
	}{
		{"installed and active", withNode, true},
		{"requested but missing", notInstalled, false},
		{"installed but not active (orphan shim)", orphanShim, false},
		{"no records", empty, false},
		{"empty output", "", false},
		{"not JSON", "node 24.19.0", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, info := miseLsRecordsInstalled(tt.out)
			if got != tt.want {
				t.Errorf("miseLsRecordsInstalled(%q) = %v, want %v", tt.out, got, tt.want)
			}
			if got && info == "" {
				t.Errorf("miseLsRecordsInstalled(%q) = true with empty info", tt.out)
			}
		})
	}
}

func TestMiseToolName(t *testing.T) {
	tests := []struct{ id, want string }{
		{"node@24.19.0", "node"},
		{"go@latest", "go"},
		{"node", "node"},
		{"npm:prettier", "npm:prettier"},
		{"npm:typescript@5.6", "npm:typescript"},
		{"npm:typescript@latest", "npm:typescript"},
		{"npm:@playwright/cli", "npm:@playwright/cli"},
		{"@playwright/cli", "@playwright/cli"},
		{"", ""},
	}
	for _, tt := range tests {
		if got := miseToolName(tt.id); got != tt.want {
			t.Errorf("miseToolName(%q) = %q, want %q", tt.id, got, tt.want)
		}
	}
}
