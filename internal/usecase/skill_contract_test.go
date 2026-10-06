package usecase

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eajdias/envctl/internal/domain/entity"
)

func TestParseSkillFrontmatter(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantOK  bool
		wantNm  string
		wantDsc string
	}{
		{
			name:    "plain scalars",
			content: "---\nname: git-workflow\ndescription: Git and GitHub CLI workflow\n---\n\n# Body\n",
			wantOK:  true,
			wantNm:  "git-workflow",
			wantDsc: "Git and GitHub CLI workflow",
		},
		{
			name:    "folded description and extra fields",
			content: "---\nname: agent-memory\ndescription: >-\n  Agent memory description.\nlicense: MIT\nmetadata:\n  author: envctl\n---\n# Body\n",
			wantOK:  true,
			wantNm:  "agent-memory",
			wantDsc: "Agent memory description.",
		},
		{
			name:    "crlf line endings",
			content: "---\r\nname: docker\r\ndescription: Docker local\r\n---\r\n# Body\r\n",
			wantOK:  true,
			wantNm:  "docker",
			wantDsc: "Docker local",
		},
		{
			name:    "leading utf-8 bom is ignored",
			content: "\ufeff---\nname: docker\ndescription: Docker local\n---\n# Body\n",
			wantOK:  true,
			wantNm:  "docker",
			wantDsc: "Docker local",
		},
		{
			name:    "trailing whitespace on the closing delimiter",
			content: "---\nname: docker\ndescription: Docker local\n---  \n# Body\n",
			wantOK:  true,
			wantNm:  "docker",
			wantDsc: "Docker local",
		},
		{
			name:    "body containing a bare delimiter",
			content: "---\nname: docker\ndescription: Docker local\n---\n\nintro\n\n---\n\nmore\n",
			wantOK:  true,
			wantNm:  "docker",
			wantDsc: "Docker local",
		},
		{
			name:    "empty frontmatter block parses but carries no fields",
			content: "---\n---\n# Body\n",
			wantOK:  true,
		},
		{
			name:    "no frontmatter block",
			content: "# Just a body\n",
			wantOK:  false,
		},
		{
			name:    "unterminated frontmatter",
			content: "---\nname: docker\ndescription: Docker local\n",
			wantOK:  false,
		},
		{
			name:    "longer delimiter is not a closing delimiter",
			content: "---\nname: docker\ndescription: Docker local\n----\n# Body\n",
			wantOK:  false,
		},
		{
			name:    "closing line with trailing text is not a closing delimiter",
			content: "---\nname: docker\ndescription: Docker local\n--- trailing\n# Body\n",
			wantOK:  false,
		},
		{
			name:    "invalid yaml",
			content: "---\nname: docker\ndescription: Docker local: broken\n---\n# Body\n",
			wantOK:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fm, ok := parseSkillFrontmatter([]byte(tt.content))
			if ok != tt.wantOK {
				t.Fatalf("parseSkillFrontmatter() ok = %v, want %v", ok, tt.wantOK)
			}
			if !tt.wantOK {
				return
			}
			if fm.Name != tt.wantNm {
				t.Errorf("Name = %q, want %q", fm.Name, tt.wantNm)
			}
			if fm.Description != tt.wantDsc {
				t.Errorf("Description = %q, want %q", fm.Description, tt.wantDsc)
			}
		})
	}
}

func TestValidateSkillFrontmatter(t *testing.T) {
	tests := []struct {
		name    string
		dirName string
		fm      skillFrontmatter
		wantErr bool
	}{
		{
			name:    "valid",
			dirName: "git-workflow",
			fm:      skillFrontmatter{Name: "git-workflow", Description: "Git workflow"},
		},
		{
			name:    "missing name",
			dirName: "git-workflow",
			fm:      skillFrontmatter{Description: "Git workflow"},
			wantErr: true,
		},
		{
			name:    "name differs from directory",
			dirName: "git-workflow",
			fm:      skillFrontmatter{Name: "git_workflow", Description: "Git workflow"},
			wantErr: true,
		},
		{
			name:    "uppercase name",
			dirName: "GitWorkflow",
			fm:      skillFrontmatter{Name: "GitWorkflow", Description: "Git workflow"},
			wantErr: true,
		},
		{
			name:    "consecutive hyphens",
			dirName: "git--workflow",
			fm:      skillFrontmatter{Name: "git--workflow", Description: "Git workflow"},
			wantErr: true,
		},
		{
			name:    "blank description",
			dirName: "git-workflow",
			fm:      skillFrontmatter{Name: "git-workflow", Description: "   "},
			wantErr: true,
		},
		{
			name:    "name longer than 64 characters",
			dirName: strings.Repeat("a", skillNameMaxLen+1),
			fm:      skillFrontmatter{Name: strings.Repeat("a", skillNameMaxLen+1), Description: "Too long"},
			wantErr: true,
		},
		{
			name:    "description longer than 1024 characters",
			dirName: "git-workflow",
			fm:      skillFrontmatter{Name: "git-workflow", Description: strings.Repeat("x", skillDescriptionMaxLen+1)},
			wantErr: true,
		},
		{
			name:    "limits are inclusive",
			dirName: strings.Repeat("a", skillNameMaxLen),
			fm:      skillFrontmatter{Name: strings.Repeat("a", skillNameMaxLen), Description: strings.Repeat("x", skillDescriptionMaxLen)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSkillFrontmatter(tt.dirName, tt.fm)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateSkillFrontmatter() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSkillDescription(t *testing.T) {
	dir := t.TempDir()

	valid := filepath.Join(dir, "with-description")
	if err := os.MkdirAll(valid, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(valid, "SKILL.md"), []byte("---\nname: with-description\ndescription: A real description\n---\n"), 0o644); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}
	if got := skillDescription(valid); got != "A real description" {
		t.Errorf("skillDescription() = %q, want %q", got, "A real description")
	}

	missing := filepath.Join(dir, "no-skill-file")
	if err := os.MkdirAll(missing, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if got := skillDescription(missing); got != "" {
		t.Errorf("skillDescription() on a directory without SKILL.md = %q, want empty", got)
	}
}

// The CommandCode catalog is `25 + N*391` chars; above COMMANDCODE_SKILL_CATALOG_CHAR_BUDGET
// it silently falls back to names-only, which turns auto-activation off. The catalog must be
// reported before that happens, not discovered later.
func TestSkillCatalogBudgetFindings(t *testing.T) {
	diags := skillCatalogBudgetFindings(12, "")
	if len(diags) != 1 || diags[0].Category != entity.DiagOK {
		t.Fatalf("12 skills with no budget set: got %+v, want a single OK", diags)
	}

	diags = skillCatalogBudgetFindings(48, "")
	if len(diags) != 1 || diags[0].Category != entity.DiagWarning {
		t.Fatalf("48 skills: got %+v, want a single warning", diags)
	}
	if !strings.Contains(diags[0].Details, "names-only") {
		t.Errorf("warning should name the failure mode: %q", diags[0].Details)
	}
	if !strings.Contains(diags[0].FixHint, "code-playbooks") {
		t.Errorf("fix hint should point at the lever: %q", diags[0].FixHint)
	}

	diags = skillCatalogBudgetFindings(48, "30000")
	if len(diags) != 1 || diags[0].Category != entity.DiagOK {
		t.Fatalf("48 skills with 30000 budget: got %+v, want a single OK", diags)
	}

	diags = skillCatalogBudgetFindings(12, "100")
	if len(diags) != 1 || diags[0].Category != entity.DiagWarning {
		t.Fatalf("explicit budget below the catalog: got %+v, want a warning", diags)
	}
}

func TestCatalogCharsEstimate(t *testing.T) {
	if got := catalogCharsEstimate(0); got != catalogWrapperChars {
		t.Errorf("empty catalog = %d, want %d", got, catalogWrapperChars)
	}
	if got := catalogCharsEstimate(12); got <= catalogWrapperChars {
		t.Errorf("12 skills should exceed the wrapper overhead, got %d", got)
	}
}
