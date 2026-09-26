package usecase

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eajdias/envctl/internal/domain/entity"
	"github.com/eajdias/envctl/internal/domain/repository"
)

// The CommandCode agent schema is documented in
// https://commandcode.ai/docs/agents#frontmatter-fields. Every case below is a
// documented field, and the runtime consequence of getting it wrong is silent:
// an agent that cannot do the job still loads, and the delegation quietly stops
// working.
func TestValidateCommandCodeAgentFrontmatter(t *testing.T) {
	tests := []struct {
		name        string
		frontmatter string
		wantFields  []string
		wantStatus  entity.DiagnosticStatus
		wantNothing bool
	}{
		{
			name:        "name and description only",
			frontmatter: "name: a\ndescription: d",
			wantNothing: true,
		},
		{
			name:        "tools as comma separated string",
			frontmatter: "name: a\ndescription: d\ntools: read_file, grep glob",
			wantNothing: true,
		},
		{
			name:        "tools as yaml sequence and wildcard",
			frontmatter: "name: a\ndescription: d\ntools: [read_file, \"*\"]",
			wantNothing: true,
		},
		{
			name:        "disallowedTools is validated like tools",
			frontmatter: "name: a\ndescription: d\ndisallowedTools: write_file, edit_file",
			wantNothing: true,
		},
		{
			name:        "unknown tool id is informational, not a warning",
			frontmatter: "name: a\ndescription: d\ntools: read_file, read-file",
			wantFields:  []string{"tools"},
			wantStatus:  entity.DiagInfo,
			wantNothing: false,
		},
		{
			name:        "agent tool can never be granted",
			frontmatter: "name: a\ndescription: d\ntools: read_file, agent",
			wantFields:  []string{"tools"},
			wantStatus:  entity.DiagWarning,
		},
		{
			name:        "agent_output in disallowedTools is also rejected by the runtime",
			frontmatter: "name: a\ndescription: d\ndisallowedTools: agent_output",
			wantFields:  []string{"disallowedTools"},
			wantStatus:  entity.DiagWarning,
		},
		{
			name:        "every documented permissionMode is accepted",
			frontmatter: "name: a\ndescription: d\npermissionMode: dont-ask",
			wantNothing: true,
		},
		{
			name:        "invalid permissionMode",
			frontmatter: "name: a\ndescription: d\npermissionMode: readonly",
			wantFields:  []string{"permissionMode"},
			wantStatus:  entity.DiagWarning,
		},
		{
			name:        "maxTurns as integer and as numeric string",
			frontmatter: "name: a\ndescription: d\nmaxTurns: 40",
			wantNothing: true,
		},
		{
			name:        "maxTurns as numeric string is tolerated",
			frontmatter: "name: a\ndescription: d\nmaxTurns: \"40\"",
			wantNothing: true,
		},
		{
			name:        "maxTurns zero",
			frontmatter: "name: a\ndescription: d\nmaxTurns: 0",
			wantFields:  []string{"maxTurns"},
			wantStatus:  entity.DiagWarning,
		},
		{
			name:        "maxTurns non numeric",
			frontmatter: "name: a\ndescription: d\nmaxTurns: many",
			wantFields:  []string{"maxTurns"},
			wantStatus:  entity.DiagWarning,
		},
		{
			name:        "booleans and quoted booleans",
			frontmatter: "name: a\ndescription: d\nbackground: true\nshowOutput: \"false\"",
			wantNothing: true,
		},
		{
			name:        "non boolean background",
			frontmatter: "name: a\ndescription: d\nbackground: sometimes",
			wantFields:  []string{"background"},
			wantStatus:  entity.DiagWarning,
		},
		{
			name:        "empty model is a mistake, not a default",
			frontmatter: "name: a\ndescription: d\nmodel: \"  \"",
			wantFields:  []string{"model"},
			wantStatus:  entity.DiagWarning,
			wantNothing: false,
		},
		{
			name:        "reasoningEffort is only shape checked",
			frontmatter: "name: a\ndescription: d\nreasoningEffort: high",
			wantNothing: true,
		},
		{
			name:        "unknown keys are ignored like the runtime ignores them",
			frontmatter: "name: a\ndescription: d\nlicense: MIT\nmetadata:\n  author: envctl",
			wantNothing: true,
		},
		{
			name:        "tools of the wrong shape",
			frontmatter: "name: a\ndescription: d\ntools:\n  key: value",
			wantFields:  []string{"tools"},
			wantStatus:  entity.DiagWarning,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := validateCommandCodeAgentFrontmatter([]byte(tt.frontmatter))

			if tt.wantNothing {
				if len(issues) != 0 {
					t.Fatalf("got issues %+v, want none", issues)
				}
				return
			}

			if len(issues) != len(tt.wantFields) {
				t.Fatalf("got %d issue(s) %+v, want %d", len(issues), issues, len(tt.wantFields))
			}
			for i, field := range tt.wantFields {
				if issues[i].Field != field {
					t.Errorf("issue[%d] field = %q, want %q", i, issues[i].Field, field)
				}
				if issues[i].Status != tt.wantStatus {
					t.Errorf("issue[%d] status = %v, want %v", i, issues[i].Status, tt.wantStatus)
				}
				if issues[i].Problem == "" {
					t.Errorf("issue[%d] has no problem description", i)
				}
			}
		})
	}
}

func TestValidateCommandCodeAgentFrontmatterRejectsNonMapping(t *testing.T) {
	issues := validateCommandCodeAgentFrontmatter([]byte("- a\n- b"))
	if len(issues) != 1 || issues[0].Status != entity.DiagWarning {
		t.Fatalf("got %+v, want a single warning", issues)
	}
}

// A file that does not load at all and a file that loads with a broken schema are
// both delegation failures, so both must surface on the same check.
func TestAuditCommandCodeAgentsReportsSchemaProblems(t *testing.T) {
	home := t.TempDir()
	agentsDir := filepath.Join(home, ".commandcode", "agents")
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		t.Fatal(err)
	}
	broken := "---\nname: broken\ndescription: d\ntools: read_file, agent\npermissionMode: readonly\n---\n\nbody\n"
	if err := os.WriteFile(filepath.Join(agentsDir, "broken.md"), []byte(broken), 0644); err != nil {
		t.Fatal(err)
	}
	good := "---\nname: good\ndescription: d\ntools: read_file, grep\n---\n\nbody\n"
	if err := os.WriteFile(filepath.Join(agentsDir, "good.md"), []byte(good), 0644); err != nil {
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
	uc.auditCommandCodeAgents(filepath.Join(home, ".commandcode"), func(d entity.Diagnostic) {
		diagnostics = append(diagnostics, d)
	})

	if len(diagnostics) != 1 {
		t.Fatalf("got %d diagnostics, want 1: %+v", len(diagnostics), diagnostics)
	}
	if diagnostics[0].Category != entity.DiagWarning {
		t.Errorf("category = %v, want WARNING", diagnostics[0].Category)
	}
	for _, term := range []string{"permissionMode", "tools", "agent"} {
		if !strings.Contains(diagnostics[0].Details, term) {
			t.Errorf("details %q should name %q", diagnostics[0].Details, term)
		}
	}
	if diagnostics[0].FixHint == "" {
		t.Error("warning must carry a fix hint")
	}
}

func TestAuditCommandCodeAgentsCleanTreeIsOK(t *testing.T) {
	home := t.TempDir()
	agentsDir := filepath.Join(home, ".commandcode", "agents")
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		t.Fatal(err)
	}
	agent := "---\nname: reviewer\ndescription: d\ntools: read_file, grep, glob\nmaxTurns: 40\npermissionMode: plan\n---\n\nbody\n"
	if err := os.WriteFile(filepath.Join(agentsDir, "reviewer.md"), []byte(agent), 0644); err != nil {
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
	uc.auditCommandCodeAgents(filepath.Join(home, ".commandcode"), func(d entity.Diagnostic) {
		diagnostics = append(diagnostics, d)
	})

	if len(diagnostics) != 1 || diagnostics[0].Category != entity.DiagOK {
		t.Fatalf("got %+v, want a single OK", diagnostics)
	}
	if !strings.Contains(diagnostics[0].Details, "1 custom agent") {
		t.Errorf("details = %q, want the valid agent count", diagnostics[0].Details)
	}
}
