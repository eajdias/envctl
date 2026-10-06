package usecase

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/eajdias/envctl/internal/domain/entity"
)

const (
	// Limits from the Agent Skills specification. A skill that breaks one is
	// skipped by the loader, so envctl reports it instead of calling it valid.
	skillNameMaxLen        = 64
	skillDescriptionMaxLen = 1024
)

// skillNamePattern mirrors the Agent Skills open standard honored by CommandCode
// and OpenCode: lowercase alphanumerics separated by single hyphens.
var skillNamePattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// skillFrontmatter is the subset of SKILL.md YAML frontmatter envctl inspects.
type skillFrontmatter struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

// skillFrontmatterBlock returns the raw YAML inside the leading `---` block of a
// SKILL.md file. ok is false when the block is missing, unterminated or not
// valid YAML. The closing delimiter must be a line of its own, so a `---` inside
// the body (or a stray `----`) never truncates the block early.
func skillFrontmatterBlock(content []byte) ([]byte, bool) {
	text := strings.ReplaceAll(string(content), "\r\n", "\n")
	text = strings.TrimPrefix(text, "\ufeff")
	if !strings.HasPrefix(text, "---\n") {
		return nil, false
	}

	lines := strings.Split(text[len("---\n"):], "\n")
	for i, line := range lines {
		if strings.TrimRight(line, " \t") == "---" {
			block := []byte(strings.Join(lines[:i], "\n"))
			var probe any
			if err := yaml.Unmarshal(block, &probe); err != nil {
				return nil, false
			}
			return block, true
		}
	}
	return nil, false
}

// parseSkillFrontmatter decodes the leading `---` YAML block of a SKILL.md file.
func parseSkillFrontmatter(content []byte) (skillFrontmatter, bool) {
	var fm skillFrontmatter

	block, ok := skillFrontmatterBlock(content)
	if !ok {
		return fm, false
	}
	if err := yaml.Unmarshal(block, &fm); err != nil {
		return fm, false
	}
	return fm, true
}

// validateSkillFrontmatter reports why an Agent Skills loader would reject a
// skill: `name` and a non-empty `description` are mandatory, `name` must be
// lowercase-hyphenated, at most 64 characters and equal to the skill's
// directory name, and `description` must fit in 1024 characters.
func validateSkillFrontmatter(dirName string, fm skillFrontmatter) error {
	name := strings.TrimSpace(fm.Name)
	if name == "" {
		return fmt.Errorf("frontmatter 'name' is missing")
	}
	if !skillNamePattern.MatchString(name) {
		return fmt.Errorf("frontmatter 'name' (%s) is not lowercase-hyphenated", name)
	}
	if len([]rune(name)) > skillNameMaxLen {
		return fmt.Errorf("frontmatter 'name' is %d characters (max %d)", len([]rune(name)), skillNameMaxLen)
	}
	if name != dirName {
		return fmt.Errorf("frontmatter 'name' (%s) does not match directory '%s'", name, dirName)
	}

	description := strings.TrimSpace(fm.Description)
	if description == "" {
		return fmt.Errorf("frontmatter 'description' is missing or empty (the skill will not load)")
	}
	if len([]rune(description)) > skillDescriptionMaxLen {
		return fmt.Errorf("frontmatter 'description' is %d characters (max %d)", len([]rune(description)), skillDescriptionMaxLen)
	}
	return nil
}

// skillDescription returns the frontmatter description of the SKILL.md found in
// skillDir, or "" when the file is unreadable or the description is absent.
func skillDescription(skillDir string) string {
	content, err := os.ReadFile(filepath.Join(skillDir, "SKILL.md"))
	if err != nil {
		return ""
	}
	fm, ok := parseSkillFrontmatter(content)
	if !ok {
		return ""
	}
	return strings.TrimSpace(fm.Description)
}

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
		return []entity.Diagnostic{entity.OK(
			"Skills",
			"Catalog budget",
			details,
		)}
	}
	return []entity.Diagnostic{entity.Warn(
		"Skills",
		"Catalog budget",
		details+" — above the budget the runtime degrades the catalog to names-only and skill auto-activation stops",
		fmt.Sprintf("shrink the catalog (merge into code-playbooks/references, promote to AGENTS.md, or %s >= %d)",
			catalogBudgetEnvVar, projected),
	)}
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
