package usecase

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/eajdias/envctl"
	"github.com/eajdias/envctl/internal/domain/entity"
	"github.com/eajdias/envctl/internal/infra/embedded"
	"github.com/eajdias/envctl/internal/infra/filesystem"
)

func TestClassifyOverwriteDrift_NoRecordIsInfo(t *testing.T) {
	warn, details, fixHint := classifyOverwriteDrift("dest", []byte("custom"), map[string]DeployRecord{})
	if warn {
		t.Error("divergence without a deploy record must be INFO (operator customization), not WARN")
	}
	if !strings.Contains(details, "no deploy record") {
		t.Errorf("details must name the missing record, got %q", details)
	}
	if !strings.Contains(fixHint, "--diff") {
		t.Errorf("fix hint must point at --diff, got %q", fixHint)
	}
}

func TestClassifyOverwriteDrift_TemplateChangedIsWarn(t *testing.T) {
	live := []byte("deployed-by-envctl")
	snapshots := map[string]DeployRecord{
		"dest": {SHA256: sha256Hex(live), At: time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)},
	}
	warn, details, fixHint := classifyOverwriteDrift("dest", live, snapshots)
	if !warn {
		t.Error("template changed while live matches the last deploy must be WARN (safe to re-apply)")
	}
	if !strings.Contains(details, "2026-11-01") {
		t.Errorf("details must carry the deploy date, got %q", details)
	}
	if !strings.Contains(fixHint, "run shell") {
		t.Errorf("genuine drift may suggest re-apply, got %q", fixHint)
	}
}

func TestClassifyOverwriteDrift_OperatorEditIsInfo(t *testing.T) {
	snapshots := map[string]DeployRecord{
		"dest": {SHA256: sha256Hex([]byte("deployed-by-envctl")), At: time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)},
	}
	warn, details, _ := classifyOverwriteDrift("dest", []byte("operator-edit"), snapshots)
	if warn {
		t.Error("operator edit after deploy must be INFO (hands off), not WARN")
	}
	if !strings.Contains(details, "Customized by operator") {
		t.Errorf("details must say who changed it, got %q", details)
	}
}

// configFilesFixtureUC builds a DoctorAuditUseCase whose manifest has a
// single overwrite config file: template at srcPath, destination at dstPath.
func configFilesFixtureUC(t *testing.T, srcPath, dstPath string, snapshots map[string]DeployRecord, showDiff bool) (*DoctorAuditUseCase, *AuditReport) {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "manifests"), 0755); err != nil {
		t.Fatal(err)
	}
	shellYAML := "config_files:\n" +
		"  - id: fixture_overwrite\n" +
		"    description: \"fixture overwrite file\"\n" +
		"    source: \"" + filepath.ToSlash(srcPath) + "\"\n" +
		"    destination: \"" + filepath.ToSlash(dstPath) + "\"\n" +
		"    category: \"fixture\"\n"
	if err := os.WriteFile(filepath.Join(dir, "manifests", "shell.yaml"), []byte(shellYAML), 0644); err != nil {
		t.Fatal(err)
	}
	repo := embedded.NewManifestRepository(envctl.EmbeddedFS, dir)
	uc := &DoctorAuditUseCase{
		manifestRepo: repo,
		fsManager:    filesystem.NewFileSystemManager(),
		logger:       &mockLogger{},
		showDiff:     showDiff,
	}
	if snapshots != nil {
		uc.deploySnapshots = func() map[string]DeployRecord { return snapshots }
	}
	return uc, &AuditReport{}
}

func runConfigFilesAudit(t *testing.T, uc *DoctorAuditUseCase, report *AuditReport) []entity.Diagnostic {
	t.Helper()
	var diags []entity.Diagnostic
	uc.auditConfigFiles(func(d entity.Diagnostic) { diags = append(diags, d) }, report)
	return diags
}

func TestAuditConfigFiles_IdenticalIsOK(t *testing.T) {
	dir := t.TempDir()
	content := []byte("template-bytes\n")
	src := filepath.Join(dir, "tpl.txt")
	dst := filepath.Join(dir, "live.txt")
	if err := os.WriteFile(src, content, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, content, 0644); err != nil {
		t.Fatal(err)
	}
	uc, report := configFilesFixtureUC(t, src, dst, map[string]DeployRecord{}, false)
	diags := runConfigFilesAudit(t, uc, report)
	if len(diags) != 1 || diags[0].Category != entity.DiagOK {
		t.Fatalf("identical files must be a single OK, got %+v", diags)
	}
}

func TestAuditConfigFiles_NoRecordIsInfo(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "tpl.txt")
	dst := filepath.Join(dir, "live.txt")
	if err := os.WriteFile(src, []byte("template\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, []byte("custom\n"), 0644); err != nil {
		t.Fatal(err)
	}
	uc, report := configFilesFixtureUC(t, src, dst, map[string]DeployRecord{}, true)
	diags := runConfigFilesAudit(t, uc, report)
	if len(diags) != 1 {
		t.Fatalf("want 1 diagnostic, got %+v", diags)
	}
	if diags[0].Category != entity.DiagInfo {
		t.Errorf("unrecorded divergence must be INFO, got %q", diags[0].Category)
	}
	if len(report.Diffs) != 1 || report.Diffs[0].Target != filepath.ToSlash(dst) {
		t.Errorf("showDiff must collect the preview, got %+v", report.Diffs)
	}
	if !strings.Contains(report.Diffs[0].Unified, "+custom") {
		t.Errorf("preview must show the live line, got:\n%s", report.Diffs[0].Unified)
	}
}

func TestAuditConfigFiles_TemplateChangedIsWarn(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "tpl.txt")
	dst := filepath.Join(dir, "live.txt")
	live := []byte("deployed\n")
	if err := os.WriteFile(src, []byte("new-template\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, live, 0644); err != nil {
		t.Fatal(err)
	}
	snapshots := map[string]DeployRecord{filepath.ToSlash(dst): {SHA256: sha256Hex(live), At: time.Now()}}
	uc, report := configFilesFixtureUC(t, src, dst, snapshots, false)
	diags := runConfigFilesAudit(t, uc, report)
	if len(diags) != 1 || diags[0].Category != entity.DiagWarning {
		t.Fatalf("template drift must be WARN, got %+v", diags)
	}
	if len(report.Diffs) != 0 {
		t.Errorf("showDiff off must collect no previews, got %+v", report.Diffs)
	}
}

func TestAuditConfigFiles_OperatorEditIsInfo(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "tpl.txt")
	dst := filepath.Join(dir, "live.txt")
	if err := os.WriteFile(src, []byte("template\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, []byte("mine\n"), 0644); err != nil {
		t.Fatal(err)
	}
	snapshots := map[string]DeployRecord{filepath.ToSlash(dst): {SHA256: sha256Hex([]byte("deployed\n")), At: time.Now()}}
	uc, report := configFilesFixtureUC(t, src, dst, snapshots, false)
	diags := runConfigFilesAudit(t, uc, report)
	if len(diags) != 1 || diags[0].Category != entity.DiagInfo {
		t.Fatalf("operator edit must be INFO, got %+v", diags)
	}
}
