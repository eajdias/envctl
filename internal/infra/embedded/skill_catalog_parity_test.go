package embedded

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/eajdias/envctl"
)

// TestSkillCatalogParityAcrossSources pins manifests/skills.yaml as the skill
// catalog source: the same name set must appear in configs/SKILL-INDEX.md
// (the situation→skill table) and in docs/skills.md (the numbered table).
// Three wordings of the same 12 names drift silently; this fails on the
// first divergence. Full generation from skills.yaml stays future work.
func TestSkillCatalogParityAcrossSources(t *testing.T) {
	repo := NewManifestRepository(envctl.EmbeddedFS, ".")

	skills, err := repo.LoadSkills()
	if err != nil {
		t.Fatalf("failed to load skills manifest: %v", err)
	}
	want := map[string]bool{}
	for _, s := range skills {
		if s.Name == "" {
			t.Error("skills.yaml holds a nameless entry")
			continue
		}
		want[s.Name] = true
	}
	if len(want) == 0 {
		t.Fatal("skills.yaml declares no skills")
	}

	indexData, err := envctl.EmbeddedFS.ReadFile("configs/SKILL-INDEX.md")
	if err != nil {
		t.Fatalf("read embedded SKILL-INDEX.md: %v", err)
	}
	assertNameSet(t, "configs/SKILL-INDEX.md", skillIndexNames(string(indexData)), want)

	docsData, err := os.ReadFile(filepath.Join(repoRootForDocs(t), "docs", "skills.md"))
	if err != nil {
		t.Fatalf("read docs/skills.md: %v", err)
	}
	assertNameSet(t, "docs/skills.md", numberedTableNames(string(docsData)), want)
}

// skillIndexNames extracts the backticked skill of each situation→skill
// table row. Header, separator and prose lines carry no bare backticked
// single token in the last cell and are skipped.
func skillIndexNames(index string) map[string]bool {
	names := map[string]bool{}
	row := regexp.MustCompile(`^\|.*\|\s*` + "`" + `([a-z0-9-]+)` + "`" + `\s*\|?\s*$`)
	for _, line := range strings.Split(index, "\n") {
		if m := row.FindStringSubmatch(line); m != nil {
			names[m[1]] = true
		}
	}
	return names
}

// numberedTableNames extracts the backticked skill of each "| N | `skill` |"
// row of the docs catalog table.
func numberedTableNames(docs string) map[string]bool {
	names := map[string]bool{}
	row := regexp.MustCompile(`^\|\s*\d+\s*\|\s*` + "`" + `([a-z0-9-]+)` + "`" + `\s*\|`)
	for _, line := range strings.Split(docs, "\n") {
		if m := row.FindStringSubmatch(line); m != nil {
			names[m[1]] = true
		}
	}
	return names
}

func assertNameSet(t *testing.T, source string, got, want map[string]bool) {
	t.Helper()
	for name := range got {
		if !want[name] {
			t.Errorf("%s names unknown skill %q (not in skills.yaml)", source, name)
		}
	}
	for name := range want {
		if !got[name] {
			t.Errorf("%s is missing skill %q from skills.yaml", source, name)
		}
	}
}
