package usecase

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/eajdias/envctl/internal/domain/entity"
)

// openCodeConfigShapeAgent mirrors one entry of the V2 "agents" map. Only the
// fields the contract reasons about are declared; anything else is ignored on
// purpose, so unknown-but-valid V2 keys never produce a finding.
type openCodeConfigShapeAgent struct {
	Mode        string          `json:"mode"`
	Description string          `json:"description"`
	Prompt      json.RawMessage `json:"prompt"`
	Permission  json.RawMessage `json:"permission"`
	Tools       json.RawMessage `json:"tools"`
	Temperature json.RawMessage `json:"temperature"`
	TopP        json.RawMessage `json:"top_p"`
	Disable     json.RawMessage `json:"disable"`
	MaxSteps    json.RawMessage `json:"maxSteps"`
	Permissions []struct {
		Action string `json:"action"`
	} `json:"permissions"`
}

// legacyField returns the raw value of a V1 field so the caller can report it
// only when it is actually present.
func (a openCodeConfigShapeAgent) legacyField(name string) json.RawMessage {
	switch name {
	case "prompt":
		return a.Prompt
	case "permission":
		return a.Permission
	case "tools":
		return a.Tools
	case "temperature":
		return a.Temperature
	case "top_p":
		return a.TopP
	case "disable":
		return a.Disable
	case "maxSteps":
		return a.MaxSteps
	default:
		return nil
	}
}

type openCodeConfigShapeRoot struct {
	Agent      json.RawMessage                     `json:"agent"`
	Permission json.RawMessage                     `json:"permission"`
	Agents     map[string]openCodeConfigShapeAgent `json:"agents"`
}

// legacyAgentFields are the V1 agent keys the V2 runtime ignores, per the
// official "Agents" docs ("Do not use legacy top-level fields such as
// temperature, top_p, prompt, permission, tools, disable, or maxSteps"). They
// fail silently: the agent still loads, but with the provider base prompt and
// the default permissions, so a v1-style "permission": {"edit": "deny"} grants
// full edit access to an agent its author believed was read-only. Each entry
// names the V2 replacement to use instead.
var legacyAgentFields = map[string]string{
	"prompt":      `use "system"`,
	"permission":  `use the native "permissions[]" list`,
	"tools":       `use "permissions[]"`,
	"temperature": `use "request.body.temperature"`,
	"top_p":       `use "request.body.top_p"`,
	"disable":     `use "disabled"`,
	"maxSteps":    `use "steps"`,
}

// legacyAgentFieldNames keeps the report order stable across runs.
var legacyAgentFieldNames = func() []string {
	names := make([]string, 0, len(legacyAgentFields))
	for name := range legacyAgentFields {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}()

// validateOpenCodeConfigShape checks the small set of V2 shape mistakes that
// otherwise leave doctor green while OpenCode silently ignores configuration.
func validateOpenCodeConfigShape(data []byte) []string {
	var root openCodeConfigShapeRoot
	if err := json.Unmarshal(data, &root); err != nil {
		return []string{fmt.Sprintf("invalid JSON: %v", err)}
	}

	var problems []string
	if presentJSONField(root.Agent) {
		problems = append(problems, `legacy top-level "agent" is not the native V2 shape`)
	}
	if presentJSONField(root.Permission) {
		problems = append(problems, `legacy top-level "permission" is not the native V2 shape`)
	}

	for name, agent := range root.Agents {
		if agent.Mode != "" && agent.Mode != "primary" && agent.Mode != "subagent" && agent.Mode != "all" {
			problems = append(problems, fmt.Sprintf("agent %q has unsupported mode %q", name, agent.Mode))
		}
		if (agent.Mode == "subagent" || agent.Mode == "all") && strings.TrimSpace(agent.Description) == "" {
			problems = append(problems, fmt.Sprintf("dispatchable agent %q needs a description", name))
		}
		for _, field := range legacyAgentFieldNames {
			if presentJSONField(agent.legacyField(field)) {
				problems = append(problems, fmt.Sprintf("agent %q uses legacy V1 field %q (%s)", name, field, legacyAgentFields[field]))
			}
		}
		for _, permission := range agent.Permissions {
			if permission.Action == "bash" || permission.Action == "task" {
				problems = append(problems, fmt.Sprintf("agent %q uses legacy permission action %q", name, permission.Action))
			}
		}
	}

	sort.Strings(problems)
	return problems
}

func presentJSONField(raw json.RawMessage) bool {
	return len(raw) > 0 && string(raw) != "null"
}

func (uc *DoctorAuditUseCase) auditOpenCodeConfigShape(addDiag func(entity.Diagnostic)) {
	expanded, err := uc.fsManager.ExpandUserPath("~/.config/opencode/opencode.json")
	if err != nil {
		return
	}
	data, err := os.ReadFile(expanded)
	if err != nil {
		return
	}

	problems := validateOpenCodeConfigShape(data)
	if len(problems) == 0 {
		addDiag(entity.Diagnostic{
			Category: entity.DiagOK,
			System:   "OpenCode",
			Target:   "Config shape",
			Details:  "native V2 agents/permissions shape",
		})
		return
	}

	addDiag(entity.Diagnostic{
		Category: entity.DiagWarning,
		System:   "OpenCode",
		Target:   "Config shape",
		Details:  strings.Join(problems, "; "),
		FixHint:  "run 'envctl run shell' to re-provision the native V2 config, or migrate the reported fields",
	})
}
