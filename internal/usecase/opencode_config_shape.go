package usecase

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/eajdias/envctl/internal/domain/entity"
)

type openCodeConfigShapeRoot struct {
	Agent      json.RawMessage `json:"agent"`
	Permission json.RawMessage `json:"permission"`
	Agents     map[string]struct {
		Mode        string `json:"mode"`
		Description string `json:"description"`
		Permissions []struct {
			Action string `json:"action"`
		} `json:"permissions"`
	} `json:"agents"`
}

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
