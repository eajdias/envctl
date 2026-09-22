package cli

import (
	"github.com/spf13/cobra"
)

// newCommandCodeCmd provisions only the CommandCode agent. CommandCode is the
// primary agent in this environment, so it gets its own entry point: configs,
// custom agents, MCP servers, the skill index and the skill tree — without
// touching a single opencode file.
func newCommandCodeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "commandcode",
		Short: "Provision only the CommandCode agent (configs, agents, MCP, skills)",
		Long: `Provisions everything CommandCode consumes, and nothing else:

  ~/.commandcode/settings.json    permissions (allow/ask/deny)
  ~/.commandcode/AGENTS.md        global rules (auto-loaded every turn)
  ~/.commandcode/SKILL-INDEX.md   skill index, consulted on demand
  ~/.commandcode/mcp.json         user-scope MCP servers
  ~/.commandcode/agents/          custom subagents
  ~/.commandcode/skills/          skill tree

Machine-level layers (packages, toolchains, LSP binaries, shell/git config) stay
with 'envctl run all'.`,
		Run: func(cmd *cobra.Command, args []string) {
			PrintBanner()
			runAgentProvisioning("commandcode", "CommandCode", "~/.commandcode/skills")
		},
	}
}

// newOpenCodeCmd provisions only the OpenCode agent.
func newOpenCodeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "opencode",
		Short: "Provision only the OpenCode agent (configs, plugins, memory seeds, skills)",
		Long: `Provisions everything OpenCode consumes, and nothing else:

  ~/.config/opencode/opencode.json   agent/MCP/plugin config (native V2)
  ~/.config/opencode/AGENTS.md       global rules (auto-loaded every turn)
  ~/.config/opencode/SKILL-INDEX.md  skill index, consulted on demand
  ~/.config/opencode/REFERENCE.md    operating detail, consulted on demand
  ~/.config/opencode/package.json    plugin dependency declaration
  ~/.config/opencode/memory/         lessons/patterns seeds
  ~/.config/opencode/skills/         skill tree

Machine-level layers (packages, toolchains, LSP binaries, shell/git config) stay
with 'envctl run all'.`,
		Run: func(cmd *cobra.Command, args []string) {
			PrintBanner()
			runAgentProvisioning("opencode", "OpenCode", "")
		},
	}
}
