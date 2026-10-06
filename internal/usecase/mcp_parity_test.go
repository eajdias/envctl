package usecase

import (
	"encoding/json"
	"sort"
	"testing"

	"github.com/eajdias/envctl"
)

// expectedMCPServers is the server set both agent runtimes must provision.
// The IDs are identical across runtimes on purpose: a server added to one
// config and forgotten in the other (ssh-manager was once missing on Linux)
// means an agent that cannot reach a tool the other agent has.
var expectedMCPServers = []string{"context7", "brave", "exa", "ssh-manager", "chrome-devtools"}

// TestMCPServerParityAcrossAgents locks the 5-server set on both runtimes.
// Schemas differ (OpenCode nests under mcp.servers, CommandCode under
// mcpServers), so the test compares ID sets, not shapes.
func TestMCPServerParityAcrossAgents(t *testing.T) {
	openCodeIDs := mcpServerIDs(t, "configs/opencode.json", func(data []byte) map[string]json.RawMessage {
		var doc struct {
			MCP struct {
				Servers map[string]json.RawMessage `json:"servers"`
			} `json:"mcp"`
		}
		if err := json.Unmarshal(data, &doc); err != nil {
			t.Fatalf("parse opencode mcp section: %v", err)
		}
		return doc.MCP.Servers
	})
	commandCodeIDs := mcpServerIDs(t, "configs/commandcode/mcp.json", func(data []byte) map[string]json.RawMessage {
		var doc struct {
			Servers map[string]json.RawMessage `json:"mcpServers"`
		}
		if err := json.Unmarshal(data, &doc); err != nil {
			t.Fatalf("parse commandcode mcp section: %v", err)
		}
		return doc.Servers
	})

	assertMCPServers(t, "opencode", openCodeIDs)
	assertMCPServers(t, "commandcode", commandCodeIDs)

	sort.Strings(openCodeIDs)
	sort.Strings(commandCodeIDs)
	if len(openCodeIDs) != len(commandCodeIDs) {
		t.Fatalf("mcp server count differs: opencode %v vs commandcode %v", openCodeIDs, commandCodeIDs)
	}
	for i := range openCodeIDs {
		if openCodeIDs[i] != commandCodeIDs[i] {
			t.Fatalf("mcp server drift: opencode %v vs commandcode %v", openCodeIDs, commandCodeIDs)
		}
	}
}

func mcpServerIDs(t *testing.T, path string, section func([]byte) map[string]json.RawMessage) []string {
	t.Helper()
	data, err := envctl.EmbeddedFS.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	servers := section(data)
	ids := make([]string, 0, len(servers))
	for id := range servers {
		ids = append(ids, id)
	}
	return ids
}

func assertMCPServers(t *testing.T, agent string, ids []string) {
	t.Helper()
	if len(ids) == 0 {
		t.Fatalf("%s declares no mcp servers", agent)
	}
	want := map[string]bool{}
	for _, id := range expectedMCPServers {
		want[id] = true
	}
	for _, id := range ids {
		if !want[id] {
			t.Errorf("%s has unexpected mcp server %q", agent, id)
		}
		delete(want, id)
	}
	for id := range want {
		t.Errorf("%s is missing mcp server %q", agent, id)
	}
}
