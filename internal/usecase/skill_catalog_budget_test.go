package usecase

import (
	"strings"
	"testing"

	"github.com/eajdias/envctl/internal/domain/entity"
	"github.com/eajdias/envctl/internal/domain/repository"
)

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

var _ = repository.GitManager(nil)
