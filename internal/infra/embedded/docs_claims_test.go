package embedded

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// stalePerformanceClaims are sentences the old documentation used and that this
// branch makes false: the profile now creates a swapfile and caps journald.
var stalePerformanceClaims = []string{
	"não cria swapfile",
	"Nenhum dos dois cria swapfile",
	"não altera journald",
}

// currentBehaviourDocs are the files that assert what the profile does today.
// The list is explicit rather than a repository-wide scan on purpose: a spec
// quotes the sentence it is correcting and a memory lesson names the identity
// that was current when it was written, and neither is a claim about present
// behaviour. A blanket scan produced false positives on both.
var currentBehaviourDocs = []string{
	"docs/manifests.md",
	"docs/guides/linux.md",
	"docs/architecture.md",
	"docs/doctor-and-idempotency.md",
}

// TestDocumentationClaimsMatchManifests fails when a document that describes
// current behaviour asserts something the profile no longer does.
func TestDocumentationClaimsMatchManifests(t *testing.T) {
	root := repoRootForDocs(t)
	for _, rel := range currentBehaviourDocs {
		path := filepath.Join(root, rel)
		data, err := os.ReadFile(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			t.Fatalf("reading %s: %v", rel, err)
		}
		content := string(data)
		for _, claim := range stalePerformanceClaims {
			if strings.Contains(content, claim) {
				t.Errorf("%s still asserts %q; the ubuntu-server profile now creates a swapfile and caps journald", rel, claim)
			}
		}
	}
}

// retiredProfileIdentities must not appear in a document that describes the
// current tool: the identity moved from "ubuntu-24.04" to "ubuntu-server" with
// the release floor in the manifest, because the fleet already runs 26.04.
var retiredProfileIdentities = []string{"ubuntu-24.04"}

// TestProfileIdentityIsNotAVersionedString keeps the profile identity honest in
// the docs too.
func TestProfileIdentityIsNotAVersionedString(t *testing.T) {
	root := repoRootForDocs(t)
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			// spec-agent/ and .opencode/memory/ record a decision rather than
			// describing current behaviour, so they may name the old identity.
			if info.Name() == ".git" || info.Name() == ".worktrees" ||
				info.Name() == "spec-agent" || (info.Name() == "memory" && strings.Contains(path, ".opencode")) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".md") {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		if rel == "CHANGELOG.md" {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		for _, retired := range retiredProfileIdentities {
			if strings.Contains(string(data), retired) {
				t.Errorf("%s names the retired profile identity %q; the profile is ubuntu-server with min_distro_version in the manifest", rel, retired)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking the repository failed: %v", err)
	}
}

func repoRootForDocs(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	// internal/infra/embedded -> repository root
	return filepath.Join(dir, "..", "..", "..")
}
