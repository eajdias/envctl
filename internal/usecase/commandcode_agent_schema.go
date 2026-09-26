package usecase

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/eajdias/envctl/internal/domain/entity"
	"gopkg.in/yaml.v3"
)

// CommandCode agent frontmatter contract, from
// https://commandcode.ai/docs/agents#frontmatter-fields. Only these keys are
// read by the runtime; anything else is ignored. Every mistake here is silent:
// the agent still loads, but delegates with fewer capabilities than intended, so
// envctl reports the field that will not be honored.
const (
	commandCodeToolAgent      = "agent"
	commandCodeToolAgentOutut = "agent_output"
)

// commandCodeAgentTools is the tool-id catalog the runtime ships (docs/agents
// "Tools" table, cross-checked against the 1.65.0 bundle). A new upstream tool is
// reported as INFO, never as a warning, so a CommandCode upgrade cannot turn the
// doctor permanently red.
var commandCodeAgentTools = map[string]bool{
	"read_file": true, "read_directory": true, "grep": true, "glob": true,
	"edit_file": true, "write_file": true,
	"shell_command": true, "run_command": true, "kill_shell": true,
	"web_search": true, "web_fetch": true,
	"todo_write":  true,
	"task_create": true, "task_update": true, "task_list": true, "task_get": true,
	"task_output": true, "task_stop": true,
	"cron_create": true, "cron_list": true, "cron_delete": true,
	"get_diagnostics": true, "taste": true, "ask_user_question": true, "sleep": true,
	"enter_plan_mode": true, "exit_plan_mode": true,
	"enter_worktree": true, "exit_worktree": true,
}

// commandCodeReservedAgentNames are the ids CommandCode owns: a custom file
// with one of these names is ignored by the runtime, so shipping one would be
// a silent no-op.
var commandCodeReservedAgentNames = map[string]bool{
	"explore": true, "plan": true, "review": true, "general": true,
}

// commandCodePermissionModes lists the values the runtime accepts, including the
// documented aliases.
var commandCodePermissionModes = map[string]bool{
	"default": true, "accept-edits": true, "yolo": true, "plan": true, "dont-ask": true,
	"auto-accept": true, "bypass": true,
}

// agentSchemaIssue is one problem found in an agent frontmatter. Status is
// WARNING when the value changes what the agent can do and INFO when the runtime
// would merely ignore it.
type agentSchemaIssue struct {
	Field   string
	Problem string
	Status  entity.DiagnosticStatus
}

func warnIssue(field, format string, args ...any) agentSchemaIssue {
	return agentSchemaIssue{Field: field, Problem: fmt.Sprintf(format, args...), Status: entity.DiagWarning}
}

func infoIssue(field, format string, args ...any) agentSchemaIssue {
	return agentSchemaIssue{Field: field, Problem: fmt.Sprintf(format, args...), Status: entity.DiagInfo}
}

// validateCommandCodeAgentFrontmatter checks a raw frontmatter block against the
// documented CommandCode agent schema. It is pure so the contract is testable
// without a CommandCode installation.
func validateCommandCodeAgentFrontmatter(frontmatter []byte) []agentSchemaIssue {
	var doc yaml.Node
	if err := yaml.Unmarshal(frontmatter, &doc); err != nil {
		return []agentSchemaIssue{warnIssue("frontmatter", "is not valid YAML: %v", err)}
	}
	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
		return []agentSchemaIssue{warnIssue("frontmatter", "is empty")}
	}
	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return []agentSchemaIssue{warnIssue("frontmatter", "is not a key/value block")}
	}

	var issues []agentSchemaIssue
	for i := 0; i+1 < len(root.Content); i += 2 {
		key, value := root.Content[i].Value, root.Content[i+1]
		switch key {
		case "name", "description":
			if value.Kind != yaml.ScalarNode {
				issues = append(issues, warnIssue(key, "must be a single-line string"))
			}
		case "model":
			if value.Kind != yaml.ScalarNode || strings.TrimSpace(value.Value) == "" {
				issues = append(issues, warnIssue(key, "must be a non-empty model id or \"inherit\""))
			}
		case "reasoningEffort":
			// The accepted levels depend on the resolved model, which is only
			// known at launch, so only the shape is checkable here.
			if value.Kind != yaml.ScalarNode || strings.TrimSpace(value.Value) == "" {
				issues = append(issues, warnIssue(key, "must be a non-empty reasoning level"))
			}
		case "tools", "disallowedTools":
			issues = append(issues, validateCommandCodeAgentTools(key, value)...)
		case "permissionMode":
			if value.Kind != yaml.ScalarNode || !commandCodePermissionModes[strings.TrimSpace(value.Value)] {
				issues = append(issues, warnIssue(key, "%q is not a valid mode (%s)", value.Value, sortedKeys(commandCodePermissionModes)))
			}
		case "maxTurns":
			issues = append(issues, validateCommandCodeMaxTurns(value)...)
		case "background", "showOutput":
			if !isBooleanLike(value) {
				issues = append(issues, warnIssue(key, "must be a boolean, got %q", value.Value))
			}
		}
	}
	return issues
}

func validateCommandCodeAgentTools(field string, value *yaml.Node) []agentSchemaIssue {
	ids, ok := stringListFromNode(value)
	if !ok || len(ids) == 0 {
		return []agentSchemaIssue{warnIssue(field, "must be a tool list (\"a, b\" or a YAML sequence), or \"*\"")}
	}

	var issues []agentSchemaIssue
	for _, id := range ids {
		if id == "*" {
			continue
		}
		// The runtime strips both from a subagent's tool set, so granting them
		// is always a mistake and silently keeps the agent one level deep.
		if id == commandCodeToolAgent || id == commandCodeToolAgentOutut {
			issues = append(issues, warnIssue(field, "%q can never be granted (delegation is one level deep)", id))
			continue
		}
		if !commandCodeAgentTools[id] {
			issues = append(issues, infoIssue(field, "%q is not in the known tool catalog", id))
		}
	}
	return issues
}

func validateCommandCodeMaxTurns(value *yaml.Node) []agentSchemaIssue {
	var raw string
	switch value.Kind {
	case yaml.ScalarNode:
		raw = value.Value
	default:
		return []agentSchemaIssue{warnIssue("maxTurns", "must be a positive integer")}
	}
	turns, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return []agentSchemaIssue{warnIssue("maxTurns", "%q is not an integer", raw)}
	}
	if turns <= 0 {
		return []agentSchemaIssue{warnIssue("maxTurns", "%d leaves the agent without turns", turns)}
	}
	return nil
}

// isBooleanLike accepts real booleans and the quoted "true"/"false" form, which
// the runtime coerces; anything else is a real mistake worth reporting.
func isBooleanLike(value *yaml.Node) bool {
	if value.Kind != yaml.ScalarNode {
		return false
	}
	if value.Tag == "!!bool" {
		return true
	}
	switch strings.ToLower(strings.TrimSpace(value.Value)) {
	case "true", "false":
		return true
	}
	return false
}

// stringListFromNode accepts the two documented shapes for tools: a
// comma/space-separated string or a YAML sequence.
func stringListFromNode(value *yaml.Node) ([]string, bool) {
	switch value.Kind {
	case yaml.ScalarNode:
		return splitToolList(value.Value), true
	case yaml.SequenceNode:
		var ids []string
		for _, item := range value.Content {
			if item.Kind != yaml.ScalarNode {
				return nil, false
			}
			ids = append(ids, strings.TrimSpace(item.Value))
		}
		return ids, true
	default:
		return nil, false
	}
}

func splitToolList(raw string) []string {
	return strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\t' || r == '\n'
	})
}

func sortedKeys(set map[string]bool) string {
	keys := make([]string, 0, len(set))
	for key := range set {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return strings.Join(keys, ", ")
}
