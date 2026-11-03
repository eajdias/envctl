package usecase

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/eajdias/envctl"
	"github.com/eajdias/envctl/internal/domain/entity"
	"github.com/eajdias/envctl/internal/infra/embedded"
	"github.com/eajdias/envctl/internal/infra/filesystem"
)

// guardFixtureUC builds a ProvisionShellUseCase with one overwrite file
// (template "managed", live "operator edit") plus a snapshot index that
// records a previous deploy of different bytes ("deployed before").
func guardFixtureUC(t *testing.T, force bool) (*ProvisionShellUseCase, string, string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "manifests"), 0755); err != nil {
		t.Fatal(err)
	}
	tpl := filepath.Join(dir, "tpl.txt")
	dst := filepath.Join(dir, "live.txt")
	if err := os.WriteFile(tpl, []byte("managed-baseline\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, []byte("operator edit\n"), 0644); err != nil {
		t.Fatal(err)
	}
	q := func(s string) string { return "\"" + filepath.ToSlash(s) + "\"" }
	shellYAML := "config_files:\n" +
		"  - id: guard_overwrite\n    description: \"f\"\n    source: " + q(tpl) + "\n    destination: " + q(dst) + "\n    category: \"guard-fixture\"\n"
	if err := os.WriteFile(filepath.Join(dir, "manifests", "shell.yaml"), []byte(shellYAML), 0644); err != nil {
		t.Fatal(err)
	}
	snapPath := filepath.Join(dir, "deploys.json")
	uc := &ProvisionShellUseCase{
		manifestRepo:   embedded.NewManifestRepository(envctl.EmbeddedFS, dir),
		fsManager:      filesystem.NewFileSystemManager(),
		embeddedFS:     fstest.MapFS{},
		logger:         &mockLogger{},
		snapshotPath:   snapPath,
		forceOverwrite: force,
	}
	store := &DeploySnapshotStore{fs: filesystem.NewFileSystemManager(), path: snapPath}
	if err := store.Record(filepath.ToSlash(dst), []byte("deployed before\n")); err != nil {
		t.Fatal(err)
	}
	return uc, dst, snapPath
}

func TestProvisionShell_SkipsCustomizedWithoutForce(t *testing.T) {
	uc, dst, _ := guardFixtureUC(t, false)
	res, err := uc.Execute(context.Background(), "guard-fixture")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	live, _ := os.ReadFile(dst)
	if string(live) != "operator edit\n" {
		t.Fatalf("customized file must be left untouched without --force, got %q", live)
	}
	found := false
	for _, d := range res.ConfigDiagnostics {
		if d.Target == filepath.ToSlash(dst) && strings.Contains(d.Details, "Skipped") {
			found = true
		}
	}
	if !found {
		t.Errorf("skip must surface as a Skipped diagnostic, got %+v", res.ConfigDiagnostics)
	}
}

func TestProvisionShell_ForceOverwritesWithBackup(t *testing.T) {
	uc, dst, snapPath := guardFixtureUC(t, true)
	res, err := uc.Execute(context.Background(), "guard-fixture")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	live, _ := os.ReadFile(dst)
	if string(live) != "managed-baseline\n" {
		t.Fatalf("--force must converge the file, got %q", live)
	}
	updated := false
	for _, d := range res.ConfigDiagnostics {
		if d.Target == filepath.ToSlash(dst) && strings.Contains(d.Details, "Updated") {
			updated = true
		}
	}
	if !updated {
		t.Errorf("--force overwrite must report Updated with backup, got %+v", res.ConfigDiagnostics)
	}
	store := &DeploySnapshotStore{fs: filesystem.NewFileSystemManager(), path: snapPath}
	if rec, ok := store.Load()[filepath.ToSlash(dst)]; !ok || rec.SHA256 != sha256Hex(live) {
		t.Error("post-force snapshot must pin the deployed bytes")
	}
}

func TestFreshMigrationTarget(t *testing.T) {
	tpl := []byte(markdownManagedStart + "\nmanaged\n" + markdownManagedEnd + "\n" + markdownUserStart + "\nseed\n" + markdownUserEnd + "\n")
	if got := freshMigrationTarget(tpl, []byte("managed\n"), entity.MergeMarkdownSections); got != nil {
		t.Errorf("unmarked copy of managed must migrate fresh, got %q", got)
	}
	legacy := []byte("managed\noperator note\n")
	if got := freshMigrationTarget(tpl, legacy, entity.MergeMarkdownSections); string(got) != "operator note" {
		t.Errorf("appended user content must migrate as the user block, got %q", got)
	}
	edited := []byte("MANAGED EDITED\noperator note\n")
	if got := freshMigrationTarget(tpl, edited, entity.MergeMarkdownSections); string(got) != string(edited) {
		t.Error("in-place edit of managed text must pass through untouched")
	}
	marked := []byte(markdownManagedStart + "\nm\n" + markdownManagedEnd + "\n" + markdownUserStart + "\nu\n" + markdownUserEnd + "\n")
	if got := freshMigrationTarget(tpl, marked, entity.MergeMarkdownSections); string(got) != string(marked) {
		t.Error("already-marked destination must pass through untouched")
	}
}
