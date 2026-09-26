package usecase

import (
	"io/fs"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/eajdias/envctl"
	"github.com/eajdias/envctl/internal/domain/entity"
)

// commandCodeAgentTemplatesDir is the embedded source of the custom agents
// deployed to ~/.commandcode/agents.
const commandCodeAgentTemplatesDir = "configs/commandcode/agents"

// TestShippedCommandCodeAgentTemplatesMatchSchema locks the contract for the
// agent files envctl provisions. The doctor only sees what is already deployed,
// so a template that ships broken stays broken on every machine until someone
// runs the CLI there. Both severities fail: a shipped agent should never carry
// a warning, and an advisory means the tool catalog or the schema drifted.
func TestShippedCommandCodeAgentTemplatesMatchSchema(t *testing.T) {
	entries, err := fs.ReadDir(envctl.EmbeddedFS, commandCodeAgentTemplatesDir)
	if err != nil {
		t.Fatalf("read embedded agent dir: %v", err)
	}

	var found []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		found = append(found, entry.Name())

		content, readErr := fs.ReadFile(envctl.EmbeddedFS, commandCodeAgentTemplatesDir+"/"+entry.Name())
		if readErr != nil {
			t.Fatalf("read %s: %v", entry.Name(), readErr)
		}

		block, blockOK := skillFrontmatterBlock(content)
		fm, fmOK := parseSkillFrontmatter(content)
		id := strings.TrimSuffix(entry.Name(), ".md")
		if !blockOK || !fmOK {
			t.Errorf("agent %s: frontmatter block is missing or unparseable", entry.Name())
			continue
		}
		if got := strings.TrimSpace(fm.Name); got != id {
			t.Errorf("agent %s: frontmatter name %q must equal the filename id %q", entry.Name(), got, id)
		}
		if commandCodeReservedAgentNames[id] {
			t.Errorf("agent %s: %q is a reserved CommandCode name, the runtime ignores the file", entry.Name(), id)
		}
		if strings.TrimSpace(fm.Description) == "" {
			t.Errorf("agent %s: description is what the runtime matches to delegate, it cannot be empty", entry.Name())
		}
		for _, issue := range validateCommandCodeAgentFrontmatter(block) {
			status := "WARN"
			if issue.Status == entity.DiagInfo {
				status = "INFO"
			}
			t.Errorf("agent %s: %s %s: %s", entry.Name(), status, issue.Field, issue.Problem)
		}
	}

	sort.Strings(found)
	expected := []string{"code-reviewer.md", "docs-writer.md", "memory-keeper.md", "verifier.md"}
	if !reflect.DeepEqual(found, expected) {
		t.Errorf("shipped agents = %v, want %v", found, expected)
	}
}
