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

func TestSubagentSupervisionUsesV2SessionInterrupt(t *testing.T) {
	data, err := envctl.EmbeddedFS.ReadFile("configs/skills/subagent-supervision/SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)

	for _, forbidden := range []string{"agent_output", "agent_id", "run_in_background"} {
		if strings.Contains(content, forbidden) {
			t.Errorf("subagent-supervision still references unavailable tool vocabulary %q", forbidden)
		}
	}
	for _, required := range []string{
		"opencode api post /api/session/",
		"interrupt",
		"sessionID",
		"PID",
	} {
		if !strings.Contains(content, required) {
			t.Errorf("subagent-supervision is missing the V2 lifecycle term %q", required)
		}
	}
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
