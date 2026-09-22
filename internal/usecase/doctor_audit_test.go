package usecase

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/eajdias/envctl/internal/domain/entity"
	"github.com/eajdias/envctl/internal/domain/repository"
)

// mockManifestRepo implements repository.ManifestRepository for testing.
type mockManifestRepo struct {
	pkgs         []entity.Package
	gamingPkgs   []entity.Package
	configFiles  []entity.ConfigFile
	skills       []entity.Skill
	lsps         []entity.LSP
	envVars      []entity.EnvironmentVar
	gitConfigs   []entity.GitConfig
	directories  []entity.RestrictedDir
	cleanupItems []entity.CleanupItem
	tweaks       []entity.WindowsTweak
}

func (m *mockManifestRepo) LoadPackages() ([]entity.Package, error) { return m.pkgs, nil }
func (m *mockManifestRepo) LoadGamingPackages() ([]entity.Package, error) {
	return m.gamingPkgs, nil
}
func (m *mockManifestRepo) LoadConfigFiles() ([]entity.ConfigFile, error) { return m.configFiles, nil }
func (m *mockManifestRepo) LoadSkills() ([]entity.Skill, error)           { return m.skills, nil }
func (m *mockManifestRepo) LoadLSPs() ([]entity.LSP, error)               { return m.lsps, nil }
func (m *mockManifestRepo) LoadEnvVars() ([]entity.EnvironmentVar, error) { return m.envVars, nil }
func (m *mockManifestRepo) LoadGitConfigs() ([]entity.GitConfig, error)   { return m.gitConfigs, nil }
func (m *mockManifestRepo) LoadDirectories() ([]entity.RestrictedDir, error) {
	return m.directories, nil
}
func (m *mockManifestRepo) LoadCleanupItems() ([]entity.CleanupItem, error) {
	return m.cleanupItems, nil
}
func (m *mockManifestRepo) LoadWindowsTweaks() ([]entity.WindowsTweak, error) {
	return m.tweaks, nil
}
func (m *mockManifestRepo) SavePackages(pkgs []entity.Package) error        { return nil }
func (m *mockManifestRepo) SaveSkills(skills []entity.Skill) error          { return nil }
func (m *mockManifestRepo) SaveLSPs(lsps []entity.LSP) error                { return nil }
func (m *mockManifestRepo) SaveGitConfigs(configs []entity.GitConfig) error { return nil }

// mockFSManager implements repository.FileSystemManager for testing.
type mockFSManager struct {
	existingPaths map[string]bool
	fileContents  map[string][]byte
}

func (m *mockFSManager) WriteWithBackup(destPath string, content []byte, perm os.FileMode) (string, error) {
	return "", nil
}
func (m *mockFSManager) ReadFile(path string) ([]byte, error) {
	if content, ok := m.fileContents[path]; ok {
		return content, nil
	}
	return nil, fmt.Errorf("file not found: %s", path)
}
func (m *mockFSManager) EnsureDirectory(path string, perm os.FileMode) error { return nil }
func (m *mockFSManager) Exists(path string) bool                             { return m.existingPaths[path] }
func (m *mockFSManager) ExpandUserPath(path string) (string, error)          { return path, nil }
func (m *mockFSManager) SetStrictWindowsACL(path string) error               { return nil }
func (m *mockFSManager) CopyEmbeddedTree(embeddedFS fs.FS, sourceDir, targetDir string) (int, error) {
	return 0, nil
}

// mockEnvManager implements repository.WindowsEnvManager for testing.
type mockEnvManager struct {
	vars map[string]string
}

func (m *mockEnvManager) GetEnvVar(scope, name string) (string, error) {
	if v, ok := m.vars[scope+"/"+name]; ok {
		return v, nil
	}
	return "", nil
}
func (m *mockEnvManager) SetEnvVar(scope, name, value string) error {
	if m.vars == nil {
		m.vars = map[string]string{}
	}
	m.vars[scope+"/"+name] = value
	return nil
}
func (m *mockEnvManager) EnsureEnvVars(ctx context.Context, vars []entity.EnvironmentVar) ([]entity.Diagnostic, error) {
	return nil, nil
}
func (m *mockEnvManager) EnsurePathEntry(ctx context.Context, dir string) (bool, error) {
	return false, nil
}

// mockLogger implements repository.Logger for testing.
type mockLogger struct{}

func (m *mockLogger) Info(format string, args ...any)  {}
func (m *mockLogger) Warn(format string, args ...any)  {}
func (m *mockLogger) Error(format string, args ...any) {}
func (m *mockLogger) Debug(format string, args ...any) {}
func (m *mockLogger) LogCommand(cmd string, args []string, exitCode int, output string, err error) {
}
func (m *mockLogger) LogIdempotency(system, target string, skipped bool, reason string) {}
func (m *mockLogger) GetLogFilePath() string                                            { return "" }
func (m *mockLogger) Close() error                                                      { return nil }

func TestDoctorAudit_GoogleChromeDetection(t *testing.T) {
	manifestRepo := &mockManifestRepo{}
	fsManager := &mockFSManager{
		existingPaths: map[string]bool{
			// Windows candidates (used when runtime.GOOS == "windows").
			`C:\Program Files\Google\Chrome\Application\chrome.exe`: true,
			// Linux/macOS candidates (used on other platforms).
			`/usr/bin/google-chrome`:  true,
			`node_modules/playwright`: true,
		},
		fileContents: map[string][]byte{},
	}
	logger := &mockLogger{}

	uc := NewDoctorAuditUseCase(
		manifestRepo,
		fsManager,
		&mockEnvManager{},
		nil,
		nil,
		map[entity.PackageType]repository.PackageManager{},
		logger,
	)

	report, err := uc.Execute(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	chromeDiagFound := false
	for _, d := range report.Diagnostics {
		if d.System == "Browser" && d.Target == "Google Chrome" {
			chromeDiagFound = true
			if d.Category != entity.DiagOK {
				t.Errorf("expected Chrome diag to be OK, got %v: %s", d.Category, d.Details)
			}
		}
	}

	if !chromeDiagFound {
		t.Errorf("expected Google Chrome diagnostic in results")
	}
}

func TestDoctorAudit_GoogleChromeMissing(t *testing.T) {
	manifestRepo := &mockManifestRepo{}
	fsManager := &mockFSManager{
		existingPaths: map[string]bool{},
		fileContents:  map[string][]byte{},
	}
	logger := &mockLogger{}

	uc := NewDoctorAuditUseCase(
		manifestRepo,
		fsManager,
		&mockEnvManager{},
		nil,
		nil,
		map[entity.PackageType]repository.PackageManager{},
		logger,
	)

	report, err := uc.Execute(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	chromeDiagFound := false
	for _, d := range report.Diagnostics {
		if d.System == "Browser" && d.Target == "Google Chrome" {
			chromeDiagFound = true
			// If not in PATH or candidate paths, should be a warning
			// (Note: on machines where google-chrome is in PATH, LookPath might find it)
			if d.Category == entity.DiagWarning && d.FixHint == "" {
				t.Errorf("expected non-empty FixHint when Chrome is missing")
			}
		}
	}

	if !chromeDiagFound {
		t.Errorf("expected Google Chrome diagnostic in results")
	}
}

// expandingFSManager is a mockFSManager whose ExpandUserPath maps "~" to a
// temp dir, so auditOpenCodeFileRefs can be exercised against real files.
type expandingFSManager struct {
	mockFSManager
	home string
}

func (m *expandingFSManager) ExpandUserPath(path string) (string, error) {
	if path == "~" || path == "~/" {
		return m.home, nil
	}
	if len(path) > 2 && path[:2] == "~/" {
		return filepath.Join(m.home, filepath.FromSlash(path[2:])), nil
	}
	return path, nil
}

func TestDoctorAudit_OpenCodeFileRefsMissing(t *testing.T) {
	home := t.TempDir()
	deployedDir := filepath.Join(home, ".config", "opencode")
	if err := os.MkdirAll(deployedDir, 0755); err != nil {
		t.Fatal(err)
	}
	missingRef := "~/.config/opencode/secrets/context7.key"
	content := `{"mcp":{"context7":{"headers":{"CONTEXT7_API_KEY":"{file:` + missingRef + `}"}}}}`
	if err := os.WriteFile(filepath.Join(deployedDir, "opencode.json"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	uc := NewDoctorAuditUseCase(
		&mockManifestRepo{},
		&expandingFSManager{mockFSManager: mockFSManager{existingPaths: map[string]bool{}, fileContents: map[string][]byte{}}, home: home},
		&mockEnvManager{},
		nil,
		nil,
		map[entity.PackageType]repository.PackageManager{},
		&mockLogger{},
	)

	var diags []entity.Diagnostic
	uc.auditOpenCodeFileRefs(func(d entity.Diagnostic) { diags = append(diags, d) })

	if len(diags) != 1 {
		t.Fatalf("expected 1 file-refs diagnostic, got %d", len(diags))
	}
	d := diags[0]
	if d.Category != entity.DiagError {
		t.Errorf("expected ERROR for missing file ref, got %v: %s", d.Category, d.Details)
	}
	if d.System != "OpenCode" || d.Target != "Config file references" {
		t.Errorf("unexpected diagnostic identity: %s/%s", d.System, d.Target)
	}
	if d.FixHint == "" {
		t.Errorf("expected non-empty FixHint for missing file ref")
	}
}

func TestDoctorAudit_OpenCodeFileRefsOK(t *testing.T) {
	home := t.TempDir()
	deployedDir := filepath.Join(home, ".config", "opencode")
	secretsDir := filepath.Join(deployedDir, "secrets")
	if err := os.MkdirAll(secretsDir, 0755); err != nil {
		t.Fatal(err)
	}
	ref := "~/.config/opencode/secrets/context7.key"
	content := `{"mcp":{"context7":{"headers":{"CONTEXT7_API_KEY":"{file:` + ref + `}"}}}}`
	if err := os.WriteFile(filepath.Join(deployedDir, "opencode.json"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(secretsDir, "context7.key"), []byte("dummy"), 0600); err != nil {
		t.Fatal(err)
	}

	uc := NewDoctorAuditUseCase(
		&mockManifestRepo{},
		&expandingFSManager{mockFSManager: mockFSManager{existingPaths: map[string]bool{}, fileContents: map[string][]byte{}}, home: home},
		&mockEnvManager{},
		nil,
		nil,
		map[entity.PackageType]repository.PackageManager{},
		&mockLogger{},
	)

	var diags []entity.Diagnostic
	uc.auditOpenCodeFileRefs(func(d entity.Diagnostic) { diags = append(diags, d) })

	if len(diags) != 1 {
		t.Fatalf("expected 1 file-refs diagnostic, got %d", len(diags))
	}
	if diags[0].Category != entity.DiagOK {
		t.Errorf("expected OK when file ref resolves, got %v: %s", diags[0].Category, diags[0].Details)
	}
}

func TestDoctorAudit_OpenCodeFileRefsNone(t *testing.T) {
	home := t.TempDir()
	deployedDir := filepath.Join(home, ".config", "opencode")
	if err := os.MkdirAll(deployedDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(deployedDir, "opencode.json"), []byte(`{"mcp":{}}`), 0644); err != nil {
		t.Fatal(err)
	}

	uc := NewDoctorAuditUseCase(
		&mockManifestRepo{},
		&expandingFSManager{mockFSManager: mockFSManager{existingPaths: map[string]bool{}, fileContents: map[string][]byte{}}, home: home},
		&mockEnvManager{},
		nil,
		nil,
		map[entity.PackageType]repository.PackageManager{},
		&mockLogger{},
	)

	var diags []entity.Diagnostic
	uc.auditOpenCodeFileRefs(func(d entity.Diagnostic) { diags = append(diags, d) })

	if len(diags) != 0 {
		t.Errorf("expected no diagnostic when opencode.json has no file refs, got %d", len(diags))
	}
}

// writeSkill materialises <base>/<name>/SKILL.md for the skill-tree audit tests.
func writeSkill(t *testing.T, base, name, content string) {
	t.Helper()
	dir := filepath.Join(base, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("setup skill %s: %v", name, err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("setup SKILL.md for %s: %v", name, err)
	}
}

func skillTreeUseCase(skillsDir string, skills []entity.Skill) *DoctorAuditUseCase {
	return NewDoctorAuditUseCase(
		&mockManifestRepo{skills: skills},
		&mockFSManager{existingPaths: map[string]bool{skillsDir: true}, fileContents: map[string][]byte{}},
		&mockEnvManager{},
		nil,
		nil,
		map[entity.PackageType]repository.PackageManager{},
		&mockLogger{},
	)
}

func TestAuditSkillTreeRejectsUnloadableSkills(t *testing.T) {
	base := t.TempDir()
	writeSkill(t, base, "boa", "---\nname: boa\ndescription: Skill valida\n---\n\n# boa\n")
	// Name does not match the directory: the loader rejects it.
	writeSkill(t, base, "quebrada", "---\nname: outro-nome\ndescription: Nome difere do diretorio\n---\n")
	// No frontmatter at all.
	writeSkill(t, base, "sem-fm", "# sem frontmatter\n")

	uc := skillTreeUseCase(base, []entity.Skill{
		{Name: "boa", Enabled: true},
		{Name: "quebrada", Enabled: true},
		{Name: "sem-fm", Enabled: true},
	})

	var diags []entity.Diagnostic
	uc.auditSkillTree("Skills", base, func(d entity.Diagnostic) { diags = append(diags, d) })

	if len(diags) != 1 {
		t.Fatalf("expected a single aggregated warning, got %d: %v", len(diags), diags)
	}
	if diags[0].Category != entity.DiagWarning {
		t.Errorf("expected WARNING, got %v", diags[0].Category)
	}
	for _, want := range []string{"quebrada", "sem-fm", "2 of 3"} {
		if !strings.Contains(diags[0].Details, want) {
			t.Errorf("details %q must mention %q", diags[0].Details, want)
		}
	}
}

func TestAuditSkillTreeAcceptsAValidTree(t *testing.T) {
	base := t.TempDir()
	writeSkill(t, base, "uma", "---\nname: uma\ndescription: Primeira skill\n---\n")
	writeSkill(t, base, "duas", "---\nname: duas\ndescription: Segunda skill\n---\n")

	uc := skillTreeUseCase(base, []entity.Skill{{Name: "uma", Enabled: true}, {Name: "duas", Enabled: true}})

	var diags []entity.Diagnostic
	uc.auditSkillTree("Skills", base, func(d entity.Diagnostic) { diags = append(diags, d) })

	if len(diags) != 1 || diags[0].Category != entity.DiagOK {
		t.Fatalf("expected a single OK diagnostic, got %v", diags)
	}
	if !strings.Contains(diags[0].Details, "2 skills deployed") {
		t.Errorf("details %q must report the deployed count", diags[0].Details)
	}
}

func TestAuditSkillTreeFlagsCountDrift(t *testing.T) {
	base := t.TempDir()
	writeSkill(t, base, "uma", "---\nname: uma\ndescription: Primeira skill\n---\n")

	uc := skillTreeUseCase(base, []entity.Skill{
		{Name: "uma", Enabled: true},
		{Name: "duas", Enabled: true},
		{Name: "tres", Enabled: true},
	})

	var diags []entity.Diagnostic
	uc.auditSkillTree("Skills", base, func(d entity.Diagnostic) { diags = append(diags, d) })

	if len(diags) != 2 {
		t.Fatalf("expected OK + count drift, got %v", diags)
	}
	drift := diags[1]
	if drift.Category != entity.DiagWarning || !strings.Contains(drift.Details, "manifest declares 3") {
		t.Errorf("unexpected drift diagnostic: %+v", drift)
	}
}

func TestAuditSkillTreeReportsMissingDirectory(t *testing.T) {
	base := filepath.Join(t.TempDir(), "nao-existe")
	uc := &DoctorAuditUseCase{fsManager: &mockFSManager{existingPaths: map[string]bool{}, fileContents: map[string][]byte{}}, logger: &mockLogger{}}

	var diags []entity.Diagnostic
	uc.auditSkillTree("CommandCode", base, func(d entity.Diagnostic) { diags = append(diags, d) })

	if len(diags) != 1 || diags[0].Category != entity.DiagWarning {
		t.Fatalf("expected a single WARNING for a missing directory, got %v", diags)
	}
	if !strings.Contains(diags[0].Details, "not found") {
		t.Errorf("unexpected details: %q", diags[0].Details)
	}
}

func staleMCPUseCase(home string) *DoctorAuditUseCase {
	return NewDoctorAuditUseCase(
		&mockManifestRepo{},
		&expandingFSManager{mockFSManager: mockFSManager{existingPaths: map[string]bool{}, fileContents: map[string][]byte{}}, home: home},
		&mockEnvManager{},
		nil,
		nil,
		map[entity.PackageType]repository.PackageManager{},
		&mockLogger{},
	)
}

func TestDoctorAudit_RemovedMCPEntriesFlagged(t *testing.T) {
	home := t.TempDir()
	deployedDir := filepath.Join(home, ".config", "opencode")
	if err := os.MkdirAll(deployedDir, 0755); err != nil {
		t.Fatal(err)
	}
	content := `{"mcp":{"context7":{},"zscan":{"command":["npx"]}}}`
	if err := os.WriteFile(filepath.Join(deployedDir, "opencode.json"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	uc := staleMCPUseCase(home)

	var diags []entity.Diagnostic
	uc.auditRemovedMCPEntries(func(d entity.Diagnostic) { diags = append(diags, d) })

	if len(diags) != 1 {
		t.Fatalf("expected 1 stale-MCP diagnostic, got %d", len(diags))
	}
	d := diags[0]
	if d.Category != entity.DiagWarning {
		t.Errorf("expected WARNING for removed MCP entry, got %v: %s", d.Category, d.Details)
	}
	if !strings.Contains(d.Details, "zscan") {
		t.Errorf("expected diagnostic to name the stale entry, got %q", d.Details)
	}
	if d.FixHint == "" {
		t.Errorf("expected non-empty FixHint for stale MCP entry")
	}
}

func TestDoctorAudit_RemovedMCPEntriesClean(t *testing.T) {
	home := t.TempDir()
	deployedDir := filepath.Join(home, ".config", "opencode")
	if err := os.MkdirAll(deployedDir, 0755); err != nil {
		t.Fatal(err)
	}
	content := `{"mcp":{"context7":{},"ssh-manager":{"command":["mcp-ssh-manager"]}}}`
	if err := os.WriteFile(filepath.Join(deployedDir, "opencode.json"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	uc := staleMCPUseCase(home)

	var diags []entity.Diagnostic
	uc.auditRemovedMCPEntries(func(d entity.Diagnostic) { diags = append(diags, d) })

	if len(diags) != 0 {
		t.Errorf("expected no diagnostic for a clean config, got %d", len(diags))
	}
}

func TestDoctorAudit_AgentsIdentityCoverageGap(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fixtures assume a linux host (windows/darwin must not match)")
	}
	uc := staleMCPUseCase(t.TempDir())
	files := []entity.ConfigFile{
		{Destination: "~/.config/opencode/AGENTS.md", OS: "windows"},
		{Destination: "~/.config/opencode/AGENTS.md", OS: "darwin"},
	}

	var diags []entity.Diagnostic
	uc.auditAgentsIdentityCoverage(func(d entity.Diagnostic) { diags = append(diags, d) }, files)

	if len(diags) != 1 {
		t.Fatalf("expected 1 identity-coverage diagnostic on linux, got %d", len(diags))
	}
	if diags[0].Category != entity.DiagWarning {
		t.Errorf("expected WARNING for identity gap, got %v: %s", diags[0].Category, diags[0].Details)
	}
}

func TestDoctorAudit_AgentsIdentityCoverageMatched(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fixtures assume a linux host (bare linux must match)")
	}
	uc := staleMCPUseCase(t.TempDir())
	files := []entity.ConfigFile{
		{Destination: "~/.config/opencode/AGENTS.md", OS: "linux"},
	}

	var diags []entity.Diagnostic
	uc.auditAgentsIdentityCoverage(func(d entity.Diagnostic) { diags = append(diags, d) }, files)

	if len(diags) != 0 {
		t.Errorf("expected no diagnostic when a variant matches, got %d", len(diags))
	}
}

func handshakeUseCase(lsps []entity.LSP) *DoctorAuditUseCase {
	return NewDoctorAuditUseCase(
		&mockManifestRepo{lsps: lsps},
		&mockFSManager{existingPaths: map[string]bool{}, fileContents: map[string][]byte{}},
		&mockEnvManager{},
		nil,
		nil,
		map[entity.PackageType]repository.PackageManager{},
		&mockLogger{},
	)
}

func TestDoctorAudit_LSPHandshakeFlagsConnectionError(t *testing.T) {
	uc := handshakeUseCase([]entity.LSP{{
		ServerName:  "Broken Test Server",
		Command:     "sh",
		Args:        []string{"-c", "echo 'Connection input stream is not set' >&2; exit 1"},
		CheckBinary: "sh",
	}})

	var diags []entity.Diagnostic
	uc.auditLSPHandshake(context.Background(), func(d entity.Diagnostic) { diags = append(diags, d) })

	if len(diags) != 1 {
		t.Fatalf("expected 1 handshake diagnostic, got %d", len(diags))
	}
	d := diags[0]
	if d.Category != entity.DiagWarning {
		t.Errorf("expected WARNING for connection error, got %v: %s", d.Category, d.Details)
	}
	if !strings.Contains(d.Details, "sh -c") {
		t.Errorf("expected diagnostic to carry the repro command, got %q", d.Details)
	}
	if d.FixHint == "" {
		t.Errorf("expected non-empty FixHint for handshake failure")
	}
}

func TestDoctorAudit_LSPHandshakeQuietExitPasses(t *testing.T) {
	uc := handshakeUseCase([]entity.LSP{{
		ServerName:  "Quiet Test Server",
		Command:     "sh",
		Args:        []string{"-c", "exit 1"},
		CheckBinary: "sh",
	}})

	var diags []entity.Diagnostic
	uc.auditLSPHandshake(context.Background(), func(d entity.Diagnostic) { diags = append(diags, d) })

	if len(diags) != 0 {
		t.Errorf("expected no diagnostic for quiet EOF exit (healthy shape), got %d", len(diags))
	}
}

func TestDoctorAudit_LSPHandshakeSkipsMissingBinary(t *testing.T) {
	uc := handshakeUseCase([]entity.LSP{{
		ServerName:  "Missing Test Server",
		Command:     "definitely-not-a-real-binary-xyz",
		Args:        []string{"--stdio"},
		CheckBinary: "definitely-not-a-real-binary-xyz",
	}})

	var diags []entity.Diagnostic
	uc.auditLSPHandshake(context.Background(), func(d entity.Diagnostic) { diags = append(diags, d) })

	if len(diags) != 0 {
		t.Errorf("expected no handshake diagnostic when the binary is missing (presence check owns it), got %d", len(diags))
	}
}

func TestMissingCmdlineParams(t *testing.T) {
	full := "quiet rw preempt=full split_lock_detect=off amdgpu.ppfeaturemask=0xffffffff zswap.enabled=0 mitigations=off"
	if missing := missingCmdlineParams(full, gamingKernelParams); len(missing) != 0 {
		t.Errorf("expected no missing params, got %v", missing)
	}
	// mitigations=off must never be required: absent is fine.
	withoutMitigations := "quiet rw preempt=full split_lock_detect=off zswap.enabled=0"
	if missing := missingCmdlineParams(withoutMitigations, gamingKernelParams); len(missing) != 0 {
		t.Errorf("expected no missing params without mitigations, got %v", missing)
	}
	partial := "quiet rw preempt=full"
	missing := missingCmdlineParams(partial, gamingKernelParams)
	if len(missing) != 2 || missing[0] != "split_lock_detect=off" || missing[1] != "zswap.enabled=0" {
		t.Errorf("expected [split_lock_detect=off zswap.enabled=0], got %v", missing)
	}
}

func TestMultilibEnabled(t *testing.T) {
	active := "[core]\nInclude = /etc/pacman.d/mirrorlist\n\n[multilib]\nInclude = /etc/pacman.d/mirrorlist\n"
	if !multilibEnabled(active) {
		t.Errorf("expected active [multilib] to be detected")
	}
	commented := "#[multilib]\n#Include = /etc/pacman.d/mirrorlist\n"
	if multilibEnabled(commented) {
		t.Errorf("expected commented #[multilib] to be reported as disabled")
	}
	if multilibEnabled("[core]\nInclude = /etc/pacman.d/mirrorlist\n") {
		t.Errorf("expected missing [multilib] to be reported as disabled")
	}
}

// mockGamingPackageManager is a minimal repository.PackageManager for the
// gaming audit tests.
type mockGamingPackageManager struct {
	available bool
	installed map[string]string
}

func (m *mockGamingPackageManager) Type() entity.PackageType { return entity.PackageTypePacman }
func (m *mockGamingPackageManager) IsAvailable(ctx context.Context) bool {
	return m.available
}
func (m *mockGamingPackageManager) IsInstalled(ctx context.Context, pkg entity.Package) (bool, string, error) {
	if v, ok := m.installed[pkg.ID]; ok {
		return true, v, nil
	}
	return false, "", nil
}
func (m *mockGamingPackageManager) Install(ctx context.Context, pkg entity.Package) error {
	return nil
}
func (m *mockGamingPackageManager) ListInstalled(ctx context.Context) ([]entity.Package, error) {
	return nil, nil
}

func gamingStackUseCase(gamingPkgs []entity.Package, managers map[entity.PackageType]repository.PackageManager) *DoctorAuditUseCase {
	return NewDoctorAuditUseCase(
		&mockManifestRepo{gamingPkgs: gamingPkgs},
		&mockFSManager{existingPaths: map[string]bool{}, fileContents: map[string][]byte{}},
		&mockEnvManager{},
		nil,
		nil,
		managers,
		&mockLogger{},
	)
}

func TestDoctorAudit_GamingStackSkippedWithoutManager(t *testing.T) {
	pkgs := []entity.Package{
		{ID: "steam", Type: entity.PackageTypePacman, OS: "arch,cachyos"},
	}
	uc := gamingStackUseCase(pkgs, map[entity.PackageType]repository.PackageManager{})

	var diags []entity.Diagnostic
	uc.auditGamingStack(context.Background(), func(d entity.Diagnostic) { diags = append(diags, d) })

	for _, d := range diags {
		if d.System == "Gaming" {
			t.Errorf("expected no Gaming diagnostics without a package manager, got %+v", d)
		}
	}
}

func TestDoctorAudit_GamingStackSilentWhenSteamAbsent(t *testing.T) {
	if runtime.GOOS != "linux" || !entity.MatchesOS("arch,cachyos") {
		t.Skip("gaming presence gate only resolves on Arch/CachyOS")
	}
	pkgs := []entity.Package{
		{ID: "steam", Type: entity.PackageTypePacman, OS: "arch,cachyos"},
		{ID: "lact", Type: entity.PackageTypePacman, OS: "arch,cachyos"},
	}
	uc := gamingStackUseCase(pkgs, map[entity.PackageType]repository.PackageManager{
		entity.PackageTypePacman: &mockGamingPackageManager{available: true, installed: map[string]string{}},
	})

	var diags []entity.Diagnostic
	uc.auditGamingStack(context.Background(), func(d entity.Diagnostic) { diags = append(diags, d) })

	for _, d := range diags {
		if d.System == "Gaming" {
			t.Errorf("expected silence when Steam is absent (not opted in), got %+v", d)
		}
	}
}

func TestDoctorAudit_GamingStackAuditsPackagesWhenOptedIn(t *testing.T) {
	if runtime.GOOS != "linux" || !entity.MatchesOS("arch,cachyos") {
		t.Skip("gaming presence gate only resolves on Arch/CachyOS")
	}
	pkgs := []entity.Package{
		{ID: "steam", Type: entity.PackageTypePacman, OS: "arch,cachyos"},
		{ID: "definitely-not-installed-xyz", Type: entity.PackageTypePacman, OS: "arch,cachyos"},
	}
	uc := gamingStackUseCase(pkgs, map[entity.PackageType]repository.PackageManager{
		entity.PackageTypePacman: &mockGamingPackageManager{
			available: true,
			installed: map[string]string{"steam": "1.0.0.87-3"},
		},
	})

	var diags []entity.Diagnostic
	uc.auditGamingStack(context.Background(), func(d entity.Diagnostic) { diags = append(diags, d) })

	var steamOK, missingWarn bool
	for _, d := range diags {
		if d.System != "Gaming" {
			continue
		}
		if d.Target == "steam" && d.Category == entity.DiagOK {
			steamOK = true
		}
		if d.Target == "definitely-not-installed-xyz" && d.Category == entity.DiagWarning {
			missingWarn = true
			if d.FixHint == "" {
				t.Errorf("expected non-empty FixHint for missing gaming package")
			}
		}
	}
	if !steamOK {
		t.Errorf("expected OK diagnostic for installed steam")
	}
	if !missingWarn {
		t.Errorf("expected WARNING diagnostic for missing gaming package")
	}
}
