package usecase

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/eajdias/envctl"
	"gopkg.in/yaml.v3"
)

const proactiveSkillRule = "Se a descrição de uma skill casar com a tarefa, carregue-a com a tool `skill` antes de agir"

func TestAgentInstructionsRequireProactiveSkillLoading(t *testing.T) {
	paths := []string{
		"configs/AGENTS.md",
		"configs/AGENTS.linux.md",
		"configs/AGENTS.arch.md",
		"configs/commandcode/AGENTS.md",
		"configs/commandcode/AGENTS.linux.md",
		"configs/commandcode/AGENTS.arch.md",
	}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			data, err := envctl.EmbeddedFS.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", path, err)
			}
			if !strings.Contains(string(data), proactiveSkillRule) {
				t.Errorf("%s does not contain the proactive skill-loading rule", path)
			}
		})
	}
}

func TestSubagentRoutingPrefersInlineExecution(t *testing.T) {
	data, err := envctl.EmbeddedFS.ReadFile("configs/skills/subagent-routing/SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(strings.ToLower(content), "inline") {
		t.Error("subagent-routing must state that inline execution is the default")
	}
	if !strings.Contains(strings.ToLower(content), "contexto") {
		t.Error("subagent-routing must explain context isolation as the dispatch criterion")
	}
}

// markdownSection returns the body of the `## <prefix>` section, stopping at the
// next level-2 heading. Shared skills document one runtime per section, so the
// contracts below assert vocabulary per runtime instead of per file.
func markdownSection(t *testing.T, content, prefix string) string {
	t.Helper()

	var body []string
	inSection := false
	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(line, "## ") {
			if inSection {
				break
			}
			if strings.HasPrefix(line, prefix) {
				inSection = true
				continue
			}
		}
		if inSection {
			body = append(body, line)
		}
	}
	if !inSection {
		t.Fatalf("no %q section found", prefix)
	}
	return strings.Join(body, "\n")
}

func requireTerms(t *testing.T, label, section string, terms ...string) {
	t.Helper()
	for _, term := range terms {
		if !strings.Contains(section, term) {
			t.Errorf("%s is missing the term %q", label, term)
		}
	}
}

func forbidTerms(t *testing.T, label, section string, terms ...string) {
	t.Helper()
	for _, term := range terms {
		if strings.Contains(section, term) {
			t.Errorf("%s must not use %q (wrong runtime vocabulary)", label, term)
		}
	}
}

func readEmbeddedSkill(t *testing.T, name string) string {
	t.Helper()
	data, err := envctl.EmbeddedFS.ReadFile("configs/skills/" + name + "/SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// commandCodeVocabulary is the background surface that really exists in
// CommandCode (docs/background-tasks + agent_output schema in the shipped
// bundle) and must never leak into the OpenCode section.
var commandCodeVocabulary = []string{
	"agent_output",
	"agent_id",
	"run_in_background",
	"kill_shell",
	"monitor_command",
	"shell_output",
}

func TestSubagentSupervisionIsRuntimeAware(t *testing.T) {
	content := readEmbeddedSkill(t, "subagent-supervision")

	openCode := markdownSection(t, content, "## OpenCode")
	requireTerms(t, "subagent-supervision/OpenCode",
		openCode,
		"opencode api post /api/session/",
		"sessionID",
		"interrupt",
		"PID",
	)
	forbidTerms(t, "subagent-supervision/OpenCode", openCode, commandCodeVocabulary...)

	commandCode := markdownSection(t, content, "## CommandCode")
	requireTerms(t, "subagent-supervision/CommandCode",
		commandCode,
		"agent_output",
		"agent_id",
		`action: "kill"`,
		"kill_shell",
		"monitor_command",
	)
	forbidTerms(t, "subagent-supervision/CommandCode", commandCode, "opencode api")
}

func TestTaskHangWatchdogIsRuntimeAware(t *testing.T) {
	content := readEmbeddedSkill(t, "task-hang-watchdog")

	openCode := markdownSection(t, content, "## OpenCode")
	requireTerms(t, "task-hang-watchdog/OpenCode", openCode, "timeout", "PID")
	forbidTerms(t, "task-hang-watchdog/OpenCode", openCode, commandCodeVocabulary...)

	commandCode := markdownSection(t, content, "## CommandCode")
	requireTerms(t, "task-hang-watchdog/CommandCode",
		commandCode,
		"run_in_background",
		"shell_output",
		"task_output",
		"kill_shell",
		"monitor_command",
	)
	forbidTerms(t, "task-hang-watchdog/CommandCode", commandCode, "opencode api")
}

// The CommandCode catalog is built as `description + "\n\n" + when_to_use` and
// truncated to the first 249 chars, so a skill whose combined text is longer is
// advertised with a cut description and loses the trigger entirely. The
// OpenCode catalog is not truncated at all, so the same text serves both
// runtimes: the summary is what both see, and the long trigger list lives in
// the body where it is only read after the skill loads.
const commandCodeCatalogBudget = 247

// whenToUseSkills is the routing/process set where trigger matching actually
// decides whether the skill gets loaded at all.
var whenToUseSkills = map[string]bool{
	"agent-memory":            true,
	"clarify-before-acting":   true,
	"code-playbooks":          true,
	"git-workflow":            true,
	"subagent-routing":        true,
	"subagent-supervision":    true,
	"systematic-debugging":    true,
	"task-hang-watchdog":      true,
	"technical-research":      true,
	"test-driven-development": true,
	"writing-plans":           true,
}

type catalogFrontmatter struct {
	Description string `yaml:"description"`
	WhenToUse   string `yaml:"when_to_use"`
}

func TestWhenToUseFitsCommandCodeCatalog(t *testing.T) {
	entries, err := envctl.EmbeddedFS.ReadDir("configs/skills")
	if err != nil {
		t.Fatal(err)
	}

	seen := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		data, readErr := envctl.EmbeddedFS.ReadFile("configs/skills/" + name + "/SKILL.md")
		if readErr != nil {
			t.Fatalf("read skill %s: %v", name, readErr)
		}
		content := string(data)

		block, ok := skillFrontmatterBlock(data)
		if !ok {
			t.Fatalf("skill %s has no parseable frontmatter", name)
		}
		var fm catalogFrontmatter
		if err := yaml.Unmarshal(block, &fm); err != nil {
			t.Fatalf("skill %s frontmatter does not parse: %v", name, err)
		}

		if !whenToUseSkills[name] {
			if strings.TrimSpace(fm.WhenToUse) != "" {
				t.Errorf("skill %q declares when_to_use but is not part of the routing set (scope creep)", name)
			}
			continue
		}
		seen++

		if !strings.Contains(content, "when_to_use: >-") {
			t.Errorf("skill %q must declare when_to_use as a folded block (a plain scalar with \": \" breaks the YAML)", name)
		}
		combined := utf8.RuneCountInString(fm.Description) + 2 + utf8.RuneCountInString(fm.WhenToUse)
		if combined > commandCodeCatalogBudget {
			t.Errorf("skill %q catalog text is %d characters; CommandCode truncates at 249 with JS slice() semantics, so keep description+2+when_to_use <= %d (shorten the summary, move the trigger list into the body)",
				name, combined, commandCodeCatalogBudget)
		}
		if strings.TrimSpace(fm.Description) == "" {
			t.Errorf("skill %q lost its description", name)
		}
	}

	if seen != len(whenToUseSkills) {
		t.Errorf("%d skill(s) declare when_to_use, want %d (the routing set)", seen, len(whenToUseSkills))
	}
}
