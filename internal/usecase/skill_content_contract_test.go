package usecase

import (
	"strings"
	"testing"

	"github.com/eajdias/envctl"
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

func TestVPSDispatchDescriptionUsesTriggerContract(t *testing.T) {
	data, err := envctl.EmbeddedFS.ReadFile("configs/skills/vps-agent-dispatch/SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, "description: >-") {
		t.Error("vps-agent-dispatch description must use the folded YAML form")
	}
	if !strings.Contains(content, "Triggers:") {
		t.Error("vps-agent-dispatch description must expose explicit triggers")
	}
}
