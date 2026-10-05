package usecase

import (
	"encoding/json"
	"strings"
	"testing"
)

const sshTemplateFixture = `# ~/.ssh/config - Optimized OpenSSH Client Configuration (Linux)
# Managed by envctl

# Global Performance & Resiliency Defaults
Host *
    ServerAliveInterval 30
    IdentityFile ~/.ssh/id_ed25519

# GitHub SSH Optimization
Host github.com
    HostName github.com
    User git
`

func TestMergeSSHHostsPreservesUserStanzas(t *testing.T) {
	existing := `# PC -> notebook (chave existente)
Host notebook
    HostName 100.104.80.117
    User eajdias-note
    IdentityFile ~/.ssh/notebook_pc

# PC -> celular
Host celular
    HostName 100.94.82.43
    Port 8022
`

	merged := string(mergeSSHHosts([]byte(sshTemplateFixture), []byte(existing)))

	for _, want := range []string{
		"Host notebook",
		"HostName 100.104.80.117",
		"IdentityFile ~/.ssh/notebook_pc",
		"Host celular",
		"Port 8022",
		"# PC -> notebook (chave existente)",
	} {
		if !strings.Contains(merged, want) {
			t.Errorf("merged config lost %q\n---\n%s", want, merged)
		}
	}
	if !strings.Contains(merged, "Host *") || !strings.Contains(merged, "Host github.com") {
		t.Errorf("merged config lost template entries\n---\n%s", merged)
	}

	// Specific hosts must precede the template wildcard so their options win.
	if strings.Index(merged, "Host notebook") > strings.Index(merged, "Host *") {
		t.Errorf("user stanza must be inserted before the template Host * block\n---\n%s", merged)
	}
}

func TestMergeSSHHostsIsIdempotent(t *testing.T) {
	existing := `# PC -> notebook
Host notebook
    HostName 100.104.80.117

Host celular
    Port 8022
`
	first := mergeSSHHosts([]byte(sshTemplateFixture), []byte(existing))
	second := mergeSSHHosts([]byte(sshTemplateFixture), first)

	if string(first) != string(second) {
		t.Errorf("merge is not idempotent\n--- first ---\n%s\n--- second ---\n%s", first, second)
	}
	if strings.Count(string(second), "# Managed by envctl") != 1 {
		t.Errorf("template header duplicated after re-merge\n---\n%s", second)
	}
	if strings.Count(string(second), "Host notebook") != 1 {
		t.Errorf("user stanza duplicated after re-merge\n---\n%s", second)
	}
}

func TestMergeSSHHostsDoesNotResurrectTemplateHosts(t *testing.T) {
	// A host the template defines must not be preserved from the destination.
	existing := `Host github.com
    User old-and-stale
`
	merged := string(mergeSSHHosts([]byte(sshTemplateFixture), []byte(existing)))

	if strings.Contains(merged, "old-and-stale") {
		t.Errorf("template-defined host must not be preserved from the destination\n---\n%s", merged)
	}
}

func TestMergeSSHHostsEmptyDestinationKeepsTemplate(t *testing.T) {
	merged := mergeSSHHosts([]byte(sshTemplateFixture), nil)
	if string(merged) != sshTemplateFixture {
		t.Errorf("empty destination must return the template unchanged\n---\n%s", merged)
	}
}

func TestMergeJSONDepsKeepsUserEntries(t *testing.T) {
	template := []byte(`{
  "name": "envctl-user-root",
  "private": true,
  "dependencies": {
    "axios": "^1.7.9",
    "cheerio": "^1.0.0"
  }
}`)
	existing := []byte(`{
  "name": "user-custom-name",
  "dependencies": {
    "axios": "^0.0.1",
    "papaparse": "^5.4.1",
    "my-own-lib": "^3.2.1"
  },
  "scripts": {
    "fetch": "node fetch.js"
  }
}`)

	merged, err := mergeJSONDeps(template, existing)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed struct {
		Name         string            `json:"name"`
		Scripts      map[string]string `json:"scripts"`
		Dependencies map[string]string `json:"dependencies"`
	}
	if err := json.Unmarshal(merged, &parsed); err != nil {
		t.Fatalf("merged output is not valid JSON: %v\n%s", err, merged)
	}

	// Template wins for managed keys, so the baseline stays current.
	if parsed.Name != "envctl-user-root" {
		t.Errorf("template key must win, got name=%q", parsed.Name)
	}
	if parsed.Dependencies["axios"] != "^1.7.9" {
		t.Errorf("template dependency version must win, got %q", parsed.Dependencies["axios"])
	}
	// User additions survive.
	for dep, version := range map[string]string{
		"papaparse":  "^5.4.1",
		"my-own-lib": "^3.2.1",
		"cheerio":    "^1.0.0",
	} {
		if parsed.Dependencies[dep] != version {
			t.Errorf("dependency %q = %q, want %q", dep, parsed.Dependencies[dep], version)
		}
	}
	if parsed.Scripts["fetch"] != "node fetch.js" {
		t.Errorf("user-only top-level keys must be preserved, got scripts=%v", parsed.Scripts)
	}
}

func TestMergeJSONDepsRejectsInvalidDestination(t *testing.T) {
	if _, err := mergeJSONDeps([]byte(`{"name":"baseline"}`), []byte(`{not json`)); err == nil {
		t.Fatalf("expected an error for an unparseable destination so the caller keeps the user file")
	}
}

const markdownTemplateFixture = `<!-- envctl:managed:start -->
# CommandCode Environment Manifest

## Ambiente
- **OS:** Windows 11

## CommandCode
- **Config:** ~/.commandcode/settings.json
<!-- envctl:managed:end -->

<!-- envctl:user:start -->
## Erros / Lições (não repetir)

## Padrões / Preferências (o que funciona)
<!-- envctl:user:end -->
`

func TestMergeMarkdownSectionsPreservesUserBlock(t *testing.T) {
	existing := `<!-- envctl:managed:start -->
# old managed content (stale)
<!-- envctl:managed:end -->

<!-- envctl:user:start -->
## Erros / Lições (não repetir)

- 2026-10-05 ❌ usar X → ✅ usar Y

## Padrões / Preferências (o que funciona)

- ✅ prefiro pnpm
<!-- envctl:user:end -->
`

	merged := string(mergeMarkdownSections([]byte(markdownTemplateFixture), []byte(existing)))

	// Managed block comes from the template (fresh), not the stale destination.
	if !strings.Contains(merged, "# CommandCode Environment Manifest") {
		t.Errorf("merged output lost the template managed header\n---\n%s", merged)
	}
	if strings.Contains(merged, "old managed content (stale)") {
		t.Errorf("stale managed content must not survive\n---\n%s", merged)
	}
	// User lessons survive.
	for _, want := range []string{
		"- 2026-10-05 ❌ usar X → ✅ usar Y",
		"- ✅ prefiro pnpm",
	} {
		if !strings.Contains(merged, want) {
			t.Errorf("merged output lost user content %q\n---\n%s", want, merged)
		}
	}
}

func TestMergeMarkdownSectionsIsIdempotent(t *testing.T) {
	existing := `<!-- envctl:managed:start -->
# stale
<!-- envctl:managed:end -->

<!-- envctl:user:start -->
## Erros / Lições (não repetir)

- lição local
<!-- envctl:user:end -->
`
	first := mergeMarkdownSections([]byte(markdownTemplateFixture), []byte(existing))
	second := mergeMarkdownSections([]byte(markdownTemplateFixture), first)

	if string(first) != string(second) {
		t.Errorf("merge is not idempotent\n--- first ---\n%s\n--- second ---\n%s", first, second)
	}
	if strings.Count(string(second), markdownManagedStart) != 1 {
		t.Errorf("managed start marker duplicated after re-merge\n---\n%s", second)
	}
	if strings.Count(string(second), "- lição local") != 1 {
		t.Errorf("user lesson duplicated after re-merge\n---\n%s", second)
	}
}

func TestMergeMarkdownSectionsWithoutMarkersKeepsEverything(t *testing.T) {
	// A legacy destination (no markers at all) is preserved whole as user
	// content: nothing the user wrote is ever lost on the first migration.
	legacy := "# CommandCode Environment Manifest\n\n## Erros / Lições (não repetir)\n\n- lição antiga sem marcador\n"

	merged := string(mergeMarkdownSections([]byte(markdownTemplateFixture), []byte(legacy)))

	if !strings.Contains(merged, "- lição antiga sem marcador") {
		t.Errorf("legacy content must be preserved\n---\n%s", merged)
	}
	if !strings.Contains(merged, "# CommandCode Environment Manifest") {
		t.Errorf("template managed content must be present\n---\n%s", merged)
	}
}

func TestMergeMarkdownSectionsEmptyDestinationSeedsUserBlock(t *testing.T) {
	merged := mergeMarkdownSections([]byte(markdownTemplateFixture), nil)
	out := string(merged)
	if !strings.Contains(out, "## Erros / Lições (não repetir)") {
		t.Errorf("fresh destination must seed the lessons header\n---\n%s", out)
	}
	if !strings.Contains(out, markdownManagedStart) || !strings.Contains(out, markdownUserStart) {
		t.Errorf("fresh destination must carry both markers\n---\n%s", out)
	}
}
