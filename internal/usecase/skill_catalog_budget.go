package usecase

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/eajdias/envctl/internal/domain/entity"
)

// Measured from the CommandCode 1.65.0 bundle: the model-facing catalog is
// `<skill><name>…<location>…` per entry (142 chars measured) plus the description
// capped at 249 chars (`Vs=250`, `slice(0, 248) + "…"`), wrapped in 25 chars of
// `<available_skills>`. When the rendered catalog does not fit in
// COMMANDCODE_SKILL_CATALOG_CHAR_BUDGET, the runtime falls back to names-only and
// skill auto-activation stops happening. That failure is silent, so doctor projects
// the catalog and reports it.
const (
	catalogWrapperChars   = 25
	catalogEntryChars     = 142
	catalogDescriptionCap = 249
	catalogBudgetEnvVar   = "COMMANDCODE_SKILL_CATALOG_CHAR_BUDGET"
	// The shipped CommandCode default (bundle constant `Gs=8e3`).
	catalogDefaultBudget = 8000
)

func catalogCharsEstimate(skillCount int) int {
	return catalogWrapperChars + skillCount*(catalogEntryChars+catalogDescriptionCap)
}

func skillCatalogBudgetFindings(skillCount int, budget string) []entity.Diagnostic {
	projected := catalogCharsEstimate(skillCount)
	effective := catalogDefaultBudget
	if b, err := strconv.Atoi(strings.TrimSpace(budget)); err == nil && b > 0 {
		effective = b
	}
	source := fmt.Sprintf("default %d", catalogDefaultBudget)
	if strings.TrimSpace(budget) != "" {
		source = fmt.Sprintf("%s=%d", catalogBudgetEnvVar, effective)
	}

	details := fmt.Sprintf("%d skill(s) project to ~%d chars of model catalog; effective budget is %s",
		skillCount, projected, source)
	if projected <= effective {
		return []entity.Diagnostic{{
			Category: entity.DiagOK,
			System:   "Skills",
			Target:   "Catalog budget",
			Details:  details,
		}}
	}
	return []entity.Diagnostic{{
		Category: entity.DiagWarning,
		System:   "Skills",
		Target:   "Catalog budget",
		Details:  details + " — above the budget the runtime degrades the catalog to names-only and skill auto-activation stops",
		FixHint: fmt.Sprintf("shrink the catalog (merge into code-playbooks/references, promote to AGENTS.md, or %s >= %d)",
			catalogBudgetEnvVar, projected),
	}}
}

// auditSkillCatalogBudget reports whether the shipped skill catalog still fits the
// model-facing budget of the CommandCode runtime.
func (uc *DoctorAuditUseCase) auditSkillCatalogBudget(skills []entity.Skill, addDiag func(entity.Diagnostic)) {
	if len(skills) == 0 {
		return
	}
	for _, diagnostic := range skillCatalogBudgetFindings(len(skills), os.Getenv(catalogBudgetEnvVar)) {
		addDiag(diagnostic)
	}
}
