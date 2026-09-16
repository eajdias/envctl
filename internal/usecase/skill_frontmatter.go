package usecase

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
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

// parseSkillFrontmatter decodes the leading `---` YAML block of a SKILL.md file.
// ok is false when the block is missing, unterminated or not valid YAML. The
// closing delimiter must be a line of its own, so a `---` inside the body (or a
// stray `----`) never truncates the block early.
func parseSkillFrontmatter(content []byte) (skillFrontmatter, bool) {
	var fm skillFrontmatter

	text := strings.ReplaceAll(string(content), "\r\n", "\n")
	text = strings.TrimPrefix(text, "\ufeff")
	if !strings.HasPrefix(text, "---\n") {
		return fm, false
	}

	lines := strings.Split(text[len("---\n"):], "\n")
	closing := -1
	for i, line := range lines {
		if strings.TrimRight(line, " \t") == "---" {
			closing = i
			break
		}
	}
	if closing < 0 {
		return fm, false
	}

	if err := yaml.Unmarshal([]byte(strings.Join(lines[:closing], "\n")), &fm); err != nil {
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
