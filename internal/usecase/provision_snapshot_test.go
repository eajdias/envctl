package usecase

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/eajdias/envctl"
	"github.com/eajdias/envctl/internal/infra/embedded"
	"github.com/eajdias/envctl/internal/infra/filesystem"
)

// provisionSnapshotFixtureUC builds a ProvisionShellUseCase whose shell
// manifest declares one file per regime (overwrite, json_deps merge, seed),
// all under category "snapshot-fixture" so Execute can run scoped without
// touching machine-level state (env, git, npm sections stay skipped).
func provisionSnapshotFixtureUC(t *testing.T) (*ProvisionShellUseCase, string, map[string]string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "manifests"), 0755); err != nil {
		t.Fatal(err)
	}

	tplOverwrite := filepath.Join(dir, "tpl-overwrite.txt")
	tplMerge := filepath.Join(dir, "tpl-merge.json")
	tplSeed := filepath.Join(dir, "tpl-seed.txt")
	dstOverwrite := filepath.Join(dir, "live-overwrite.txt")
	dstMerge := filepath.Join(dir, "live-merge.json")
	dstSeed := filepath.Join(dir, "live-seed.txt")
	write := func(p, c string) {
		t.Helper()
		if err := os.WriteFile(p, []byte(c), 0644); err != nil {
			t.Fatal(err)
		}
	}
	write(tplOverwrite, "managed-baseline\n")
	write(dstOverwrite, "stale-operator-copy\n")
	write(tplMerge, `{"managed": true}`)
	write(dstMerge, `{"mine": true}`)
	write(tplSeed, "seed-baseline\n")
	// dstSeed intentionally absent: the deploy creates it, and even then no
	// snapshot may be recorded for seed files.

	q := func(s string) string { return "\"" + filepath.ToSlash(s) + "\"" }
	shellYAML := "config_files:\n" +
		"  - id: snap_overwrite\n    description: \"f\"\n    source: " + q(tplOverwrite) + "\n    destination: " + q(dstOverwrite) + "\n    category: \"snapshot-fixture\"\n" +
		"  - id: snap_merge\n    description: \"f\"\n    source: " + q(tplMerge) + "\n    destination: " + q(dstMerge) + "\n    category: \"snapshot-fixture\"\n    merge: json_deps\n" +
		"  - id: snap_seed\n    description: \"f\"\n    source: " + q(tplSeed) + "\n    destination: " + q(dstSeed) + "\n    category: \"snapshot-fixture\"\n    seed_if_missing: true\n"
	if err := os.WriteFile(filepath.Join(dir, "manifests", "shell.yaml"), []byte(shellYAML), 0644); err != nil {
		t.Fatal(err)
	}

	snapPath := filepath.Join(dir, "deploys.json")
	uc := &ProvisionShellUseCase{
		manifestRepo: embedded.NewManifestRepository(envctl.EmbeddedFS, dir),
		fsManager:    filesystem.NewFileSystemManager(),
		embeddedFS:   fstest.MapFS{},
		logger:       &mockLogger{},
		snapshotPath: snapPath,
	}
	dests := map[string]string{
		"overwrite": filepath.ToSlash(dstOverwrite),
		"merge":     filepath.ToSlash(dstMerge),
		"seed":      filepath.ToSlash(dstSeed),
	}
	return uc, snapPath, dests
}

func TestProvisionShell_RecordsSnapshotOnlyForOverwrite(t *testing.T) {
	uc, snapPath, dests := provisionSnapshotFixtureUC(t)

	// First run: overwrite file is Updated, merge is merged, seed is Created.
	if _, err := uc.Execute(context.Background(), "snapshot-fixture"); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	store := &DeploySnapshotStore{fs: filesystem.NewFileSystemManager(), path: snapPath}
	records := store.Load()

	live, err := os.ReadFile(dests["overwrite"])
	if err != nil {
		t.Fatal(err)
	}
	if string(live) != "managed-baseline\n" {
		t.Fatalf("overwrite deploy must replace content, got %q", live)
	}
	rec, ok := records[dests["overwrite"]]
	if !ok {
		t.Fatalf("overwrite file must be snapshotted, got keys %v", keysOf(records))
	}
	if rec.SHA256 != sha256Hex(live) {
		t.Errorf("snapshot must pin the deployed bytes, got %q", rec.SHA256)
	}
	for _, regime := range []string{"merge", "seed"} {
		if _, ok := records[dests[regime]]; ok {
			t.Errorf("%s file must never be snapshotted (own semantics), but has a record", regime)
		}
	}

	// Second run: everything is Already up to date — the snapshot must be
	// (re)recorded anyway (backfill for pre-store deploys), still only for
	// the overwrite file.
	if _, err := uc.Execute(context.Background(), "snapshot-fixture"); err != nil {
		t.Fatalf("second Execute: %v", err)
	}
	records = store.Load()
	if len(records) != 1 {
		t.Fatalf("only the overwrite file may ever be snapshotted, got %v", keysOf(records))
	}
	if records[dests["overwrite"]].SHA256 != sha256Hex(live) {
		t.Error("backfill record must pin the same bytes")
	}
}

func keysOf(m map[string]DeployRecord) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
