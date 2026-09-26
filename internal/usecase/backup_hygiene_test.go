package usecase

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A redeploy writes <file>.bak.<stamp> before overwriting, and the agent skill
// trees are nested (~/.config/opencode/skills/<skill>/SKILL.md). A prune that only
// looks at the top level therefore never sees the skill backups, and they pile up
// one per execution forever.
func TestPruneTimestampedBackupsReachesNestedDirs(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "skills", "git-workflow")
	if err := os.MkdirAll(filepath.Join(nested, "references"), 0755); err != nil {
		t.Fatal(err)
	}
	files := []string{
		filepath.Join(root, "opencode.json"),
		filepath.Join(root, "opencode.json.bak.20260826-141200"),
		filepath.Join(root, "opencode.json.bak.20260916-140849"),
		filepath.Join(nested, "SKILL.md"),
		filepath.Join(nested, "SKILL.md.bak.20260826-141200"),
		filepath.Join(nested, "SKILL.md.bak.20260903-124442"),
		filepath.Join(nested, "SKILL.md.bak.20260916-140849"),
		filepath.Join(nested, "references", "go.md"),
	}
	for _, f := range files {
		if err := os.WriteFile(f, []byte("x"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	removed, err := pruneTimestampedBackups(root, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The two stale skill backups must be reported as removed, not silently kept.
	var sawNested bool
	for _, name := range removed {
		if strings.Contains(filepath.ToSlash(name), "skills/git-workflow") {
			sawNested = true
		}
	}
	if !sawNested {
		t.Errorf("expected nested skill backups in the removed list, got %v", removed)
	}

	for _, gone := range []string{
		filepath.Join(nested, "SKILL.md.bak.20260826-141200"),
		filepath.Join(nested, "SKILL.md.bak.20260903-124442"),
		filepath.Join(root, "opencode.json.bak.20260826-141200"),
	} {
		if _, err := os.Stat(gone); !os.IsNotExist(err) {
			t.Errorf("expected %q to be pruned", gone)
		}
	}
	for _, keep := range []string{
		filepath.Join(root, "opencode.json"),
		filepath.Join(root, "opencode.json.bak.20260916-140849"),
		filepath.Join(nested, "SKILL.md"),
		filepath.Join(nested, "SKILL.md.bak.20260916-140849"),
		filepath.Join(nested, "references", "go.md"),
	} {
		if _, err := os.Stat(keep); err != nil {
			t.Errorf("expected %q to survive pruning: %v", keep, err)
		}
	}
}

// The reverse snapshot copies a deployed skill tree into configs/skills/. Without a
// filter, every accumulated backup lands in the curated repo, and the next deploy
// ships that stale content to every machine.
func TestCopyDirSkipsProvisioningBackups(t *testing.T) {
	src := filepath.Join(t.TempDir(), "skill")
	dst := filepath.Join(t.TempDir(), "out")
	if err := os.MkdirAll(filepath.Join(src, "references"), 0755); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{
		filepath.Join(src, "SKILL.md"),
		filepath.Join(src, "SKILL.md.bak.20260826-141200"),
		filepath.Join(src, "references", "go.md"),
		filepath.Join(src, "references", "go.md.bak.20260903-124442"),
	} {
		if err := os.WriteFile(f, []byte("x"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	uc := &SnapshotSyncUseCase{}
	if err := uc.copyDir(src, dst); err != nil {
		t.Fatalf("copyDir: %v", err)
	}

	for _, want := range []string{
		filepath.Join(dst, "SKILL.md"),
		filepath.Join(dst, "references", "go.md"),
	} {
		if _, err := os.Stat(want); err != nil {
			t.Errorf("expected %q to be synced: %v", want, err)
		}
	}
	for _, unwanted := range []string{
		filepath.Join(dst, "SKILL.md.bak.20260826-141200"),
		filepath.Join(dst, "references", "go.md.bak.20260903-124442"),
	} {
		if _, err := os.Stat(unwanted); !os.IsNotExist(err) {
			t.Errorf("expected provisioning backup %q to stay out of the repo", unwanted)
		}
	}
}
