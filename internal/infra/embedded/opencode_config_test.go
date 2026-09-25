package embedded

import (
	"encoding/json"
	"testing"

	"github.com/eajdias/envctl"
)

type openCodeTemplateAgent struct {
	Mode        string                       `json:"mode"`
	Description string                       `json:"description"`
	System      string                       `json:"system"`
	Steps       int                          `json:"steps"`
	Permissions []openCodeTemplatePermission `json:"permissions"`
}

type openCodeTemplatePermission struct {
	Action   string `json:"action"`
	Resource string `json:"resource"`
	Effect   string `json:"effect"`
}

type openCodeTemplateConfig struct {
	Agents map[string]openCodeTemplateAgent `json:"agents"`
}

func hasPermission(permissions []openCodeTemplatePermission, action, resource, effect string) bool {
	for _, permission := range permissions {
		if permission.Action == action && permission.Resource == resource && permission.Effect == effect {
			return true
		}
	}
	return false
}

func TestOpenCodeConfigTemplates(t *testing.T) {
	for _, path := range []string{
		"configs/opencode.json",
		"configs/opencode.linux.json",
	} {
		t.Run(path, func(t *testing.T) {
			data, err := envctl.EmbeddedFS.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", path, err)
			}

			var config openCodeTemplateConfig
			if err := json.Unmarshal(data, &config); err != nil {
				t.Fatalf("parse %s: %v", path, err)
			}

			if _, ok := config.Agents["review"]; !ok {
				t.Fatal("expected review agent")
			}
			if _, ok := config.Agents["plan"]; !ok {
				t.Fatal("expected plan agent")
			}

			planner, ok := config.Agents["planner"]
			if !ok {
				t.Fatal("expected planner subagent")
			}
			if planner.Mode != "subagent" {
				t.Errorf("planner mode = %q, want subagent", planner.Mode)
			}
			if planner.Description == "" {
				t.Error("planner description must be non-empty so the parent model can select it")
			}
			if planner.System == "" {
				t.Error("planner system prompt must be non-empty")
			}
			if planner.Steps <= 0 {
				t.Errorf("planner steps = %d, want a positive cap", planner.Steps)
			}

			if got := config.Agents["plan"].Mode; got != "" {
				t.Errorf("built-in plan override mode = %q, want empty to preserve primary", got)
			}
			if !hasPermission(planner.Permissions, "edit", "*", "deny") {
				t.Error("planner must deny edits outside its plan artifact")
			}
			if !hasPermission(planner.Permissions, "edit", "spec-agent/**", "allow") {
				t.Error("planner must allow spec-agent plan artifacts")
			}
			if !hasPermission(planner.Permissions, "subagent", "*", "deny") {
				t.Error("planner must not dispatch nested subagents")
			}
			if !hasPermission(planner.Permissions, "shell", "*", "ask") {
				t.Error("planner shell commands must require approval by default")
			}

			for agentName, agent := range config.Agents {
				for _, permission := range agent.Permissions {
					if permission.Action == "bash" || permission.Action == "task" {
						t.Errorf("agent %q uses legacy permission action %q", agentName, permission.Action)
					}
				}
			}
		})
	}
}
