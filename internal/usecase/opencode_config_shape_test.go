package usecase

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eajdias/envctl"
	"github.com/eajdias/envctl/internal/domain/entity"
	"github.com/eajdias/envctl/internal/domain/repository"
)

func TestShippedOpenCodeTemplatesHaveNativeShape(t *testing.T) {
	for _, path := range []string{"configs/opencode.json", "configs/opencode.linux.json"} {
		t.Run(path, func(t *testing.T) {
			data, err := envctl.EmbeddedFS.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", path, err)
			}
			if problems := validateOpenCodeConfigShape(data); len(problems) > 0 {
				t.Errorf("shipped template %s has shape problems: %s", path, strings.Join(problems, "; "))
			}
		})
	}
}

func TestValidateOpenCodeConfigShape(t *testing.T) {
	tests := []struct {
		name        string
		config      string
		wantProblem string
	}{
		{
			name:   "valid native shape",
			config: `{"agents":{"review":{"mode":"primary","description":"review","permissions":[{"action":"shell","resource":"*","effect":"ask"}]},"planner":{"mode":"subagent","description":"plan","permissions":[{"action":"edit","resource":"*","effect":"deny"}]}}}`,
		},
		{
			name:        "legacy agent root",
			config:      `{"agent":{"build":{"prompt":"legacy"}}}`,
			wantProblem: "agent",
		},
		{
			name:        "legacy permission root",
			config:      `{"permission":{"bash":"allow"}}`,
			wantProblem: "permission",
		},
		{
			name:        "legacy permission action",
			config:      `{"agents":{"planner":{"mode":"subagent","description":"plan","permissions":[{"action":"bash","resource":"*","effect":"allow"}]}}}`,
			wantProblem: "bash",
		},
		{
			name:        "subagent without description",
			config:      `{"agents":{"planner":{"mode":"subagent","permissions":[]}}}`,
			wantProblem: "description",
		},
		{
			name:        "invalid json",
			config:      `{`,
			wantProblem: "invalid JSON",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			problems := validateOpenCodeConfigShape([]byte(tt.config))
			if tt.wantProblem == "" {
				if len(problems) != 0 {
					t.Fatalf("got problems %v, want none", problems)
				}
				return
			}
			if len(problems) == 0 {
				t.Fatalf("got no problems, want one containing %q", tt.wantProblem)
			}
			joined := strings.Join(problems, "; ")
			if !strings.Contains(joined, tt.wantProblem) {
				t.Fatalf("problems %q do not contain %q", joined, tt.wantProblem)
			}
		})
	}
}

func TestAuditOpenCodeConfigShapeReportsProblems(t *testing.T) {
	home := t.TempDir()
	configDir := filepath.Join(home, ".config", "opencode")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	config := `{"agents":{"planner":{"mode":"subagent","permissions":[{"action":"task","resource":"*","effect":"allow"}]}}}`
	if err := os.WriteFile(filepath.Join(configDir, "opencode.json"), []byte(config), 0644); err != nil {
		t.Fatal(err)
	}

	uc := NewDoctorAuditUseCase(
		&mockManifestRepo{},
		&expandingFSManager{mockFSManager: mockFSManager{existingPaths: map[string]bool{}}, home: home},
		&mockEnvManager{},
		nil,
		nil,
		map[entity.PackageType]repository.PackageManager{},
		&mockLogger{},
	)

	var diagnostics []entity.Diagnostic
	uc.auditOpenCodeConfigShape(func(d entity.Diagnostic) { diagnostics = append(diagnostics, d) })
	if len(diagnostics) != 1 {
		t.Fatalf("got %d diagnostics, want 1", len(diagnostics))
	}
	if diagnostics[0].Category != entity.DiagWarning {
		t.Errorf("category = %v, want WARNING", diagnostics[0].Category)
	}
	if !strings.Contains(diagnostics[0].Details, "task") {
		t.Errorf("details = %q, want legacy task action", diagnostics[0].Details)
	}
}
