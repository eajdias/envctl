package usecase

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/eajdias/envctl/internal/domain/entity"
	"github.com/eajdias/envctl/internal/domain/repository"
)

// mockManifestRepo implements repository.ManifestRepository for testing.
type mockManifestRepo struct {
	pkgs             []entity.Package
	gamingPkgs       []entity.Package
	configFiles      []entity.ConfigFile
	skills           []entity.Skill
	lsps             []entity.LSP
	envVars          []entity.EnvironmentVar
	gitConfigs       []entity.GitConfig
	directories      []entity.RestrictedDir
	cleanupItems     []entity.CleanupItem
	tweaks           []entity.WindowsTweak
	debloat          []entity.WindowsTweak
	performanceSpecs map[entity.PerformanceProfile]entity.PerformanceSpec
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
func (m *mockManifestRepo) LoadDebloatTweaks() ([]entity.WindowsTweak, error) {
	return m.debloat, nil
}
func (m *mockManifestRepo) LoadLinuxDebloatSpec() (entity.DebloatSpec, error) {
	return entity.DebloatSpec{}, nil
}

func (m *mockManifestRepo) ListPerformanceProfiles() ([]entity.PerformanceProfileMeta, error) {
	return nil, nil
}

func (m *mockManifestRepo) LoadPerformanceSpec(profile entity.PerformanceProfile) (entity.PerformanceSpec, error) {
	return m.performanceSpecs[profile], nil
}
func (m *mockManifestRepo) SaveSkills(skills []entity.Skill) error          { return nil }
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
			// Linux candidates (used on other platforms).
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
		t.Skip("fixtures assume a linux host (windows must not match)")
	}
	uc := staleMCPUseCase(t.TempDir())
	files := []entity.ConfigFile{
		{Destination: "~/.config/opencode/AGENTS.md", OS: "windows"},
		{Destination: "~/.config/opencode/AGENTS.md", OS: "windows"},
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
	// `sh` only resolves on POSIX hosts: on Windows the audit skips the
	// fixture as "binary missing" (presence check owns it) before any
	// handshake runs — see TestDoctorAudit_LSPHandshakeSkipsMissingBinary.
	command := "sh"
	if runtime.GOOS == "windows" {
		command = "cmd"
	}
	args := []string{"-c", "echo 'Connection input stream is not set' >&2; exit 1"}
	checkBinary := "sh"
	if runtime.GOOS == "windows" {
		args = []string{"/c", "echo Connection input stream is not set >&2 & exit /b 1"}
		checkBinary = "cmd"
	}
	uc := handshakeUseCase([]entity.LSP{{
		ServerName:  "Broken Test Server",
		Command:     command,
		Args:        args,
		CheckBinary: checkBinary,
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
	if !strings.Contains(d.Details, command) {
		t.Errorf("expected diagnostic to carry the repro command, got %q", d.Details)
	}
	if d.FixHint == "" {
		t.Errorf("expected non-empty FixHint for handshake failure")
	}
}

func TestDoctorAudit_LSPHandshakeQuietExitPasses(t *testing.T) {
	command := "sh"
	args := []string{"-c", "exit 1"}
	checkBinary := "sh"
	if runtime.GOOS == "windows" {
		command = "cmd"
		args = []string{"/c", "exit /b 1"}
		checkBinary = "cmd"
	}
	uc := handshakeUseCase([]entity.LSP{{
		ServerName:  "Quiet Test Server",
		Command:     command,
		Args:        args,
		CheckBinary: checkBinary,
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

// realAMDGamingCmdline is the /proc/cmdline of the managed workstation, minus
// mitigations=off which is deliberately never required.
const realAMDGamingCmdline = "quiet nowatchdog splash rw rootflags=subvol=/@ root=UUID=556567b9-0e0a-4b8c-97fa-6d386b75bf07 " +
	"mitigations=off preempt=full split_lock_detect=off amdgpu.runpm=0 amdgpu.aspm=0 pcie_aspm=off " +
	"amdgpu.gpu_recovery=0 oops=panic panic=10 zswap.enabled=0"

func TestGamingAMDKernelParams_CoversRealCmdline(t *testing.T) {
	if missing := missingCmdlineParams(realAMDGamingCmdline, gamingAMDKernelParams); len(missing) != 0 {
		t.Errorf("expected no missing AMD params on the real cmdline, got %v", missing)
	}
	if missing := missingCmdlineParams("quiet preempt=full", gamingAMDKernelParams); len(missing) != len(gamingAMDKernelParams) {
		t.Errorf("expected all %d AMD params missing, got %v", len(gamingAMDKernelParams), missing)
	}
	partial := "quiet amdgpu.runpm=0 amdgpu.aspm=0 pcie_aspm=off"
	missing := missingCmdlineParams(partial, gamingAMDKernelParams)
	if len(missing) != 1 || missing[0] != "amdgpu.gpu_recovery=0" {
		t.Errorf("expected only [amdgpu.gpu_recovery=0] missing, got %v", missing)
	}
}

// TestGamingKernelParamsExcludeAMD guards the tiering: the universal list must
// stay hardware-agnostic, or an Intel or NVIDIA host warns forever.
func TestGamingKernelParamsExcludeAMD(t *testing.T) {
	for _, p := range gamingKernelParams {
		for _, amd := range gamingAMDKernelParams {
			if p == amd {
				t.Errorf("%q is in both the universal and the AMD list", p)
			}
		}
	}
	if strings.Contains(strings.Join(gamingKernelParams, " "), "amdgpu") {
		t.Errorf("universal kernel params must not mention amdgpu, got %v", gamingKernelParams)
	}
	if strings.Contains(strings.Join(gamingAMDKernelParams, " "), "panic") {
		t.Errorf("AMD kernel params must not include the panic-stability choice, got %v", gamingAMDKernelParams)
	}
}

func TestGamingPanicParamsAreOptional(t *testing.T) {
	if len(gamingPanicParams) == 0 {
		t.Fatal("expected the panic-stability params to be reported")
	}
	// Absent is the normal case on a stock kernel; it must never be a warning.
	if missing := missingCmdlineParams("quiet preempt=full", gamingPanicParams); len(missing) != len(gamingPanicParams) {
		t.Errorf("expected all panic params reported as absent, got %v", missing)
	}
	if missing := missingCmdlineParams(realAMDGamingCmdline, gamingPanicParams); len(missing) != 0 {
		t.Errorf("expected no missing panic params on the real cmdline, got %v", missing)
	}
}

func TestAmdgpuModulePresent(t *testing.T) {
	root := t.TempDir()
	if amdgpuModulePresent(filepath.Join(root, "module", "amdgpu")) {
		t.Error("expected amdgpu to be absent when the module dir is missing")
	}
	if err := os.MkdirAll(filepath.Join(root, "module", "amdgpu", "parameters"), 0o755); err != nil {
		t.Fatal(err)
	}
	if !amdgpuModulePresent(filepath.Join(root, "module", "amdgpu")) {
		t.Error("expected amdgpu to be detected once the module dir exists")
	}
}

func TestMissingGamingConfKeys(t *testing.T) {
	both := []byte("MESA_SHADER_CACHE_MAX_SIZE=12G\nRADV_PERFTEST=gpl\n")
	if missing := missingGamingConfKeys(both); len(missing) != 0 {
		t.Errorf("expected no missing keys, got %v", missing)
	}
	onlyCache := []byte("MESA_SHADER_CACHE_MAX_SIZE=12G\n")
	if missing := missingGamingConfKeys(onlyCache); len(missing) != 1 || missing[0] != "RADV_PERFTEST=" {
		t.Errorf("expected only RADV_PERFTEST= missing, got %v", missing)
	}
	if missing := missingGamingConfKeys(nil); len(missing) != len(gamingConfRequiredKeys) {
		t.Errorf("expected all %d keys missing for an empty file, got %v", len(gamingConfRequiredKeys), missing)
	}
	// A commented-out line is not a setting; the audit must not accept it.
	commented := []byte("# MESA_SHADER_CACHE_MAX_SIZE=12G\n#RADV_PERFTEST=gpl\n")
	if missing := missingGamingConfKeys(commented); len(missing) != len(gamingConfRequiredKeys) {
		t.Errorf("expected commented keys to count as missing, got %v", missing)
	}
}

func TestDoctorAudit_GamingTuningChecksMangoHudPreset(t *testing.T) {
	if runtime.GOOS != "linux" || !entity.MatchesOS("arch,cachyos") {
		t.Skip("gaming presence gate only resolves on Arch/CachyOS")
	}
	pkgs := []entity.Package{{ID: "steam", Type: entity.PackageTypePacman, OS: "arch,cachyos"}}
	managers := map[entity.PackageType]repository.PackageManager{
		entity.PackageTypePacman: &mockGamingPackageManager{
			available: true, installed: map[string]string{"steam": "1.0.0.87-3"},
		},
	}

	// run shell seeds both presets; MangoHud.conf missing is a real gap the
	// audit did not report at all.
	uc := gamingStackUseCaseWithFS(pkgs, managers, map[string]bool{})
	var diags []entity.Diagnostic
	uc.auditGamingStack(context.Background(), func(d entity.Diagnostic) { diags = append(diags, d) })

	got := findGamingDiag(diags, "MangoHud preset")
	if got == nil {
		t.Fatalf("expected a MangoHud preset diagnostic, got none (targets: %v)", gamingTargets(diags))
	}
	if got.Category != entity.DiagWarning {
		t.Errorf("expected WARNING when the preset is absent, got %v: %s", got.Category, got.Details)
	}
	if !strings.Contains(got.FixHint, "run shell") {
		t.Errorf("expected the FixHint to point at run shell, got %q", got.FixHint)
	}
}

func TestDoctorAudit_GamingTuningAcceptsBothShaderCacheKeys(t *testing.T) {
	if runtime.GOOS != "linux" || !entity.MatchesOS("arch,cachyos") {
		t.Skip("gaming presence gate only resolves on Arch/CachyOS")
	}
	pkgs := []entity.Package{{ID: "steam", Type: entity.PackageTypePacman, OS: "arch,cachyos"}}
	managers := map[entity.PackageType]repository.PackageManager{
		entity.PackageTypePacman: &mockGamingPackageManager{
			available: true, installed: map[string]string{"steam": "1.0.0.87-3"},
		},
	}
	// game-performance plus the MangoHud preset on disk. The mock expands no
	// path, so the read key is the literal tilde form.
	existing := map[string]bool{mangoHudPresetPath: true}
	uc := gamingStackUseCaseWithFS(pkgs, managers, existing)
	uc.fsManager.(*mockFSManager).fileContents[gamingConfPath] =
		[]byte("MESA_SHADER_CACHE_MAX_SIZE=12G\nRADV_PERFTEST=gpl\n")

	var diags []entity.Diagnostic
	uc.auditGamingStack(context.Background(), func(d entity.Diagnostic) { diags = append(diags, d) })

	cache := findGamingDiag(diags, "shader cache preset")
	if cache == nil {
		t.Fatalf("expected a shader cache diagnostic, got none (targets: %v)", gamingTargets(diags))
	}
	if cache.Category != entity.DiagOK {
		t.Errorf("expected OK with both keys present, got %v: %s", cache.Category, cache.Details)
	}
	overlay := findGamingDiag(diags, "MangoHud preset")
	if overlay == nil || overlay.Category != entity.DiagOK {
		t.Errorf("expected OK for the present MangoHud preset, got %+v", overlay)
	}
}

func TestPendingPacnewFiles(t *testing.T) {
	root := t.TempDir()
	if got := pendingPacnewFiles(root); len(got) != 0 {
		t.Errorf("expected no pending .pacnew in an empty tree, got %v", got)
	}

	if err := os.MkdirAll(filepath.Join(root, "pacman.d"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{
		filepath.Join("pacman.conf.pacnew"),
		filepath.Join("limine-snapper-sync.conf.pacnew"),
		filepath.Join("pacman.d", "cachyos-mirrorlist.pacnew"),
		filepath.Join("pacman.d", "not-a-pacnew.txt"),
		filepath.Join("pacman.conf"),
	} {
		if err := os.WriteFile(filepath.Join(root, rel), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	got := pendingPacnewFiles(root)
	if len(got) != 3 {
		t.Fatalf("expected 3 pending .pacnew files, got %d: %v", len(got), got)
	}
	want := map[string]bool{
		"pacman.conf.pacnew":                 true,
		"limine-snapper-sync.conf.pacnew":    true,
		"pacman.d/cachyos-mirrorlist.pacnew": true,
	}
	for _, p := range got {
		if !want[p] {
			t.Errorf("unexpected file reported: %q", p)
		}
	}
	// Sorted, so the diagnostic is stable between runs.
	if !sort.StringsAreSorted(got) {
		t.Errorf("expected sorted output, got %v", got)
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

func TestKwinrcCompositingEnabled(t *testing.T) {
	withBoth := []byte("[Compositing]\nAllowBlockCompositing=true\nUnredirectFullscreen=true\n")
	if !kwinrcCompositingEnabled(withBoth) {
		t.Error("expected compositing bypass to be detected when both keys are active")
	}
	partial := []byte("[Compositing]\nAllowBlockCompositing=true\n")
	if kwinrcCompositingEnabled(partial) {
		t.Error("expected partial compositing settings to be reported as not enabled")
	}
	noSection := []byte("[General]\nAllowBlockCompositing=true\nUnredirectFullscreen=true\n")
	if kwinrcCompositingEnabled(noSection) {
		t.Error("expected keys outside [Compositing] to not count")
	}
	if kwinrcCompositingEnabled(nil) {
		t.Error("expected an empty file to not count")
	}
}

func TestDoctorAudit_GamingTuningChecksLactConfig(t *testing.T) {
	if runtime.GOOS != "linux" || !entity.MatchesOS("arch,cachyos") {
		t.Skip("gaming presence gate only resolves on Arch/CachyOS")
	}
	pkgs := []entity.Package{{ID: "steam", Type: entity.PackageTypePacman, OS: "arch,cachyos"}}
	managers := map[entity.PackageType]repository.PackageManager{
		entity.PackageTypePacman: &mockGamingPackageManager{
			available: true, installed: map[string]string{"steam": "1.0.0.87-3"},
		},
	}

	// LACT fan curve is a manual, privileged decision: absent config is INFO,
	// never WARN, so a host that skipped the tuning stays 0 WARN.
	uc := gamingStackUseCaseWithFS(pkgs, managers, map[string]bool{})
	var diags []entity.Diagnostic
	uc.auditGamingStack(context.Background(), func(d entity.Diagnostic) { diags = append(diags, d) })

	got := findGamingDiag(diags, "lact-config")
	if got == nil {
		t.Fatalf("expected a lact-config diagnostic, got none (targets: %v)", gamingTargets(diags))
	}
	if got.Category != entity.DiagInfo {
		t.Errorf("expected INFO when LACT config is absent, got %v: %s", got.Category, got.Details)
	}
	ucWith := gamingStackUseCaseWithFS(pkgs, managers, map[string]bool{lactConfigPath: true})
	diags = nil
	ucWith.auditGamingStack(context.Background(), func(d entity.Diagnostic) { diags = append(diags, d) })
	if got := findGamingDiag(diags, "lact-config"); got == nil || got.Category != entity.DiagOK {
		t.Errorf("expected OK when LACT config is present, got %+v", got)
	}
}

func TestDoctorAudit_GamingTuningChecksScxLoaderConfig(t *testing.T) {
	if runtime.GOOS != "linux" || !entity.MatchesOS("arch,cachyos") {
		t.Skip("gaming presence gate only resolves on Arch/CachyOS")
	}
	pkgs := []entity.Package{{ID: "steam", Type: entity.PackageTypePacman, OS: "arch,cachyos"}}
	managers := map[entity.PackageType]repository.PackageManager{
		entity.PackageTypePacman: &mockGamingPackageManager{
			available: true, installed: map[string]string{"steam": "1.0.0.87-3"},
		},
	}

	uc := gamingStackUseCaseWithFS(pkgs, managers, map[string]bool{})
	var diags []entity.Diagnostic
	uc.auditGamingStack(context.Background(), func(d entity.Diagnostic) { diags = append(diags, d) })

	got := findGamingDiag(diags, "scx-loader-config")
	if got == nil {
		t.Fatalf("expected a scx-loader-config diagnostic, got none (targets: %v)", gamingTargets(diags))
	}
	if got.Category != entity.DiagInfo {
		t.Errorf("expected INFO when scx_loader config is absent, got %v: %s", got.Category, got.Details)
	}
	ucWith := gamingStackUseCaseWithFS(pkgs, managers, map[string]bool{scxLoaderConfigPath: true})
	diags = nil
	ucWith.auditGamingStack(context.Background(), func(d entity.Diagnostic) { diags = append(diags, d) })
	if got := findGamingDiag(diags, "scx-loader-config"); got == nil || got.Category != entity.DiagOK {
		t.Errorf("expected OK when scx_loader config is present, got %+v", got)
	}
}

func TestDoctorAudit_GamingTuningChecksKwinrcCompositing(t *testing.T) {
	if runtime.GOOS != "linux" || !entity.MatchesOS("arch,cachyos") {
		t.Skip("gaming presence gate only resolves on Arch/CachyOS")
	}
	pkgs := []entity.Package{{ID: "steam", Type: entity.PackageTypePacman, OS: "arch,cachyos"}}
	managers := map[entity.PackageType]repository.PackageManager{
		entity.PackageTypePacman: &mockGamingPackageManager{
			available: true, installed: map[string]string{"steam": "1.0.0.87-3"},
		},
	}

	// kwinrc lives in the user profile; the audit reads it through the fs
	// manager so the mock can serve a custom content.
	uc := gamingStackUseCaseWithFS(pkgs, managers, map[string]bool{})
	var diags []entity.Diagnostic
	uc.auditGamingStack(context.Background(), func(d entity.Diagnostic) { diags = append(diags, d) })

	got := findGamingDiag(diags, "kwinrc-compositing")
	if got == nil {
		t.Fatalf("expected a kwinrc-compositing diagnostic, got none (targets: %v)", gamingTargets(diags))
	}
	if got.Category != entity.DiagInfo {
		t.Errorf("expected INFO when compositing bypass is absent, got %v: %s", got.Category, got.Details)
	}

	// With the compositing bypass configured, the check becomes OK.
	uc = gamingStackUseCaseWithFS(pkgs, managers, map[string]bool{})
	uc.fsManager.(*mockFSManager).fileContents[kwinrcConfigPath] =
		[]byte("[Compositing]\nAllowBlockCompositing=true\nUnredirectFullscreen=true\n")
	diags = nil
	uc.auditGamingStack(context.Background(), func(d entity.Diagnostic) { diags = append(diags, d) })
	got = findGamingDiag(diags, "kwinrc-compositing")
	if got == nil || got.Category != entity.DiagOK {
		t.Errorf("expected OK when the compositing bypass is configured, got %+v", got)
	}
}

func TestGamingEmulatorConfigs_CoverRealMachine(t *testing.T) {
	// The validated machine runs every emulator with a Vulkan backend. This
	// test pins the source of truth (the files the owner tuned by hand) so the
	// audit table cannot silently drift from what the real host uses.
	realConfigs := map[string]string{
		"~/.config/dolphin-emu/Dolphin.ini":      "GFXBackend = Vulkan",
		"~/.config/retroarch/retroarch.cfg":      `video_driver = "vulkan"`,
		"~/.config/ppsspp/PSP/SYSTEM/ppsspp.ini": "GraphicsBackend = 3",
		"~/.config/PCSX2/inis/PCSX2.ini":         "Renderer = 14",
		"~/.config/duckstation/settings.ini":     "Renderer = Vulkan",
		"~/.config/azahar-emu/qt-config.ini":     "graphics_api=2",
		"~/.config/eden/qt-config.ini":           "backend=1",
		"~/.config/Vita3K/config.yml":            "backend-renderer: Vulkan",
		"~/.config/Cemu/settings.xml":            "<api>1</api>",
	}
	for _, check := range gamingEmulatorConfigs {
		want, ok := realConfigs[check.path]
		if !ok {
			t.Errorf("emulator config %q (%s) is not part of the tuned machine set", check.path, check.target)
			continue
		}
		if want != check.key {
			t.Errorf("emulator config %q: expected key %q, table has %q", check.path, want, check.key)
		}
	}
}

func TestConfigHasKey(t *testing.T) {
	ini := []byte("# comment\nGFXBackend = Vulkan\n[General]\nfoo=bar\n")
	if !configHasKey(ini, "GFXBackend = Vulkan") {
		t.Error("expected the real key to be found")
	}
	if configHasKey(ini, "foo = bar") {
		t.Error("expected a key with a different spacing to not match")
	}
	if configHasKey(ini, "# comment") {
		t.Error("expected commented lines to not count")
	}
	yaml := []byte("backend-renderer: Vulkan\nresolution-multiplier: 2\n")
	if !configHasKey(yaml, "backend-renderer: Vulkan") {
		t.Error("expected the YAML key to be found")
	}
	xml := []byte("<api>3</api>\n<api>1</api>\n")
	if !configHasKey(xml, "<api>1</api>") {
		t.Error("expected the XML key to be found")
	}
	if configHasKey(nil, "anything") {
		t.Error("expected an empty file to not match")
	}
}

func TestDoctorAudit_GamingTuningChecksEmulatorConfigs(t *testing.T) {
	if runtime.GOOS != "linux" || !entity.MatchesOS("arch,cachyos") {
		t.Skip("gaming presence gate only resolves on Arch/CachyOS")
	}
	pkgs := []entity.Package{{ID: "steam", Type: entity.PackageTypePacman, OS: "arch,cachyos"}}
	managers := map[entity.PackageType]repository.PackageManager{
		entity.PackageTypePacman: &mockGamingPackageManager{
			available: true, installed: map[string]string{"steam": "1.0.0.87-3"},
		},
	}

	// No emulator config present: every check is INFO, never WARN — the
	// renderer choice is the owner's manual decision.
	uc := gamingStackUseCaseWithFS(pkgs, managers, map[string]bool{})
	var diags []entity.Diagnostic
	uc.auditGamingStack(context.Background(), func(d entity.Diagnostic) { diags = append(diags, d) })
	for _, check := range gamingEmulatorConfigs {
		got := findGamingDiag(diags, check.target)
		if got == nil {
			t.Fatalf("expected a %s diagnostic, got none (targets: %v)", check.target, gamingTargets(diags))
		}
		if got.Category != entity.DiagInfo {
			t.Errorf("expected INFO for missing %s, got %v: %s", check.target, got.Category, got.Details)
		}
	}

	// All configs present with the tuned keys: every check is OK.
	uc = gamingStackUseCaseWithFS(pkgs, managers, map[string]bool{})
	mock := uc.fsManager.(*mockFSManager)
	for _, check := range gamingEmulatorConfigs {
		mock.fileContents[check.path] = []byte(check.key + "\n")
	}
	diags = nil
	uc.auditGamingStack(context.Background(), func(d entity.Diagnostic) { diags = append(diags, d) })
	for _, check := range gamingEmulatorConfigs {
		got := findGamingDiag(diags, check.target)
		if got == nil {
			t.Fatalf("expected a %s diagnostic when config is present, got none", check.target)
		}
		if got.Category != entity.DiagOK {
			t.Errorf("expected OK for tuned %s, got %v: %s", check.target, got.Category, got.Details)
		}
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
	return gamingStackUseCaseWithFS(gamingPkgs, managers, nil)
}

func gamingStackUseCaseWithFS(gamingPkgs []entity.Package, managers map[entity.PackageType]repository.PackageManager, existingPaths map[string]bool) *DoctorAuditUseCase {
	if existingPaths == nil {
		existingPaths = map[string]bool{}
	}
	return NewDoctorAuditUseCase(
		&mockManifestRepo{gamingPkgs: gamingPkgs},
		&mockFSManager{existingPaths: existingPaths, fileContents: map[string][]byte{}},
		&mockEnvManager{},
		nil,
		nil,
		managers,
		&mockLogger{},
	)
}

// findGamingDiag returns the first Gaming diagnostic for target, if any.
func findGamingDiag(diags []entity.Diagnostic, target string) *entity.Diagnostic {
	for i := range diags {
		if diags[i].System == "Gaming" && diags[i].Target == target {
			return &diags[i]
		}
	}
	return nil
}

func TestDoctorAudit_GamingTuningWarnsWhenAnanicyHasNoRuleset(t *testing.T) {
	if runtime.GOOS != "linux" || !entity.MatchesOS("arch,cachyos") {
		t.Skip("gaming presence gate only resolves on Arch/CachyOS")
	}
	pkgs := []entity.Package{
		{ID: "steam", Type: entity.PackageTypePacman, OS: "arch,cachyos"},
		{ID: "ananicy-cpp", Type: entity.PackageTypePacman, OS: "arch,cachyos"},
		{ID: "cachyos-ananicy-rules", Type: entity.PackageTypePacman, OS: "arch,cachyos"},
	}
	managers := map[entity.PackageType]repository.PackageManager{
		entity.PackageTypePacman: &mockGamingPackageManager{
			available: true,
			installed: map[string]string{
				"steam":                 "1.0.0.87-3",
				"ananicy-cpp":           "1.1.1-1",
				"cachyos-ananicy-rules": "1:1.1.49-1",
			},
		},
	}

	// The ruleset package is installed but /etc/ananicy.d holds no rules: the
	// service audit passes and the daemon stays inert. The package loop cannot
	// see this, so the tuning audit has to.
	uc := gamingStackUseCaseWithFS(pkgs, managers, map[string]bool{})
	var diags []entity.Diagnostic
	uc.auditGamingStack(context.Background(), func(d entity.Diagnostic) { diags = append(diags, d) })

	got := findGamingDiag(diags, "ananicy-rules")
	if got == nil {
		t.Fatalf("expected an ananicy-rules diagnostic, got none (targets: %v)", gamingTargets(diags))
	}
	if got.Category != entity.DiagWarning {
		t.Errorf("expected WARNING for a missing ananicy ruleset, got %v: %s", got.Category, got.Details)
	}
	if !strings.Contains(got.FixHint, "cachyos-ananicy-rules") {
		t.Errorf("expected the FixHint to name the ruleset package, got %q", got.FixHint)
	}
}

func TestDoctorAudit_GamingTuningReportsAnanicyRulesetPresent(t *testing.T) {
	if runtime.GOOS != "linux" || !entity.MatchesOS("arch,cachyos") {
		t.Skip("gaming presence gate only resolves on Arch/CachyOS")
	}
	pkgs := []entity.Package{
		{ID: "steam", Type: entity.PackageTypePacman, OS: "arch,cachyos"},
		{ID: "ananicy-cpp", Type: entity.PackageTypePacman, OS: "arch,cachyos"},
	}
	managers := map[entity.PackageType]repository.PackageManager{
		entity.PackageTypePacman: &mockGamingPackageManager{
			available: true,
			installed: map[string]string{"steam": "1.0.0.87-3", "ananicy-cpp": "1.1.1-1"},
		},
	}
	uc := gamingStackUseCaseWithFS(pkgs, managers, map[string]bool{ananicyTypesMarker: true})

	var diags []entity.Diagnostic
	uc.auditGamingStack(context.Background(), func(d entity.Diagnostic) { diags = append(diags, d) })

	got := findGamingDiag(diags, "ananicy-rules")
	if got == nil {
		t.Fatalf("expected an ananicy-rules diagnostic, got none (targets: %v)", gamingTargets(diags))
	}
	if got.Category != entity.DiagOK {
		t.Errorf("expected OK when the ruleset is present, got %v: %s", got.Category, got.Details)
	}
}

// TestDoctorAudit_GamingStackAuditsAnanicyRulesPackage guards the T1 wiring:
// declaring the ruleset in the manifest is what makes the package audit cover
// the "fresh machine has no rules at all" case.
func TestDoctorAudit_GamingStackAuditsAnanicyRulesPackage(t *testing.T) {
	if runtime.GOOS != "linux" || !entity.MatchesOS("arch,cachyos") {
		t.Skip("gaming presence gate only resolves on Arch/CachyOS")
	}
	pkgs := []entity.Package{
		{ID: "steam", Type: entity.PackageTypePacman, OS: "arch,cachyos"},
		{ID: "cachyos-ananicy-rules", Type: entity.PackageTypePacman, OS: "arch,cachyos"},
	}
	managers := map[entity.PackageType]repository.PackageManager{
		entity.PackageTypePacman: &mockGamingPackageManager{
			available: true,
			installed: map[string]string{"steam": "1.0.0.87-3"},
		},
	}
	uc := gamingStackUseCaseWithFS(pkgs, managers, map[string]bool{})

	var diags []entity.Diagnostic
	uc.auditGamingStack(context.Background(), func(d entity.Diagnostic) { diags = append(diags, d) })

	got := findGamingDiag(diags, "cachyos-ananicy-rules")
	if got == nil {
		t.Fatalf("expected the package loop to audit cachyos-ananicy-rules, got none (targets: %v)", gamingTargets(diags))
	}
	if got.Category != entity.DiagWarning {
		t.Errorf("expected WARNING for the uninstalled ruleset package, got %v", got.Category)
	}
}

func gamingTargets(diags []entity.Diagnostic) []string {
	var targets []string
	for _, d := range diags {
		if d.System == "Gaming" {
			targets = append(targets, d.Target)
		}
	}
	return targets
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

func TestAuditOpenCodeVersionSkewParsesMajor(t *testing.T) {
	cases := []struct {
		output string
		skewed bool
	}{
		{"opencode 1.18.30", true},
		{"opencode v1.18.32", true},
		{"opencode v2.0.15", false},
		{"opencode v10.0.0", false},
		{"2.0.5", false},
		{"no numbers here", false},
	}
	for _, tc := range cases {
		version := firstVersionToken(tc.output)
		skewed := version != "" && !versionMajorAtLeast(version, 2)
		if skewed != tc.skewed {
			t.Errorf("firstVersionToken(%q) = %q, skewed = %v, want %v", tc.output, version, skewed, tc.skewed)
		}
	}
}

// stubTweaksManager scripts CheckTweak results for the debloat audit tests.
type stubTweaksManager struct {
	check func(entity.WindowsTweak) (bool, string, error)
}

func (s *stubTweaksManager) CheckTweak(_ context.Context, tw entity.WindowsTweak) (bool, string, error) {
	return s.check(tw)
}

func (s *stubTweaksManager) CheckBatch(_ context.Context, tweaks []entity.WindowsTweak) []entity.TweakCheckResult {
	results := make([]entity.TweakCheckResult, len(tweaks))
	for i, tw := range tweaks {
		ok, details, err := s.check(tw)
		results[i] = entity.TweakCheckResult{Tweak: tw, OK: ok, Details: details, Err: err}
	}
	return results
}

func (s *stubTweaksManager) ApplyTweak(_ context.Context, _ entity.WindowsTweak) error {
	return nil
}

func (s *stubTweaksManager) EnsureTweaks(_ context.Context, tweaks []entity.WindowsTweak) ([]entity.Diagnostic, error) {
	return nil, nil
}

func debloatAuditUseCase(debloat []entity.WindowsTweak, mgr repository.WindowsTweaksManager) *DoctorAuditUseCase {
	return NewDoctorAuditUseCase(
		&mockManifestRepo{debloat: debloat},
		&mockFSManager{},
		&mockEnvManager{},
		nil,
		mgr,
		map[entity.PackageType]repository.PackageManager{},
		&mockLogger{},
	)
}

func TestAuditDebloatNilManagerIsSilent(t *testing.T) {
	uc := debloatAuditUseCase([]entity.WindowsTweak{
		{ID: "x", Category: "telemetry", Type: "DWord"},
	}, nil)

	var diags []entity.Diagnostic
	uc.auditDebloat(context.Background(), func(d entity.Diagnostic) { diags = append(diags, d) })

	if len(diags) != 0 {
		t.Errorf("expected no debloat diagnostics without a tweaks manager, got %d", len(diags))
	}
}

func TestAuditDebloatAggregatesPerCategoryAsInfo(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("debloat audit is Windows-only")
	}
	uc := debloatAuditUseCase([]entity.WindowsTweak{
		{ID: "t1", Category: "telemetry", Type: "DWord"},
		{ID: "t2", Category: "telemetry", Type: "DWord"},
		{ID: "a1", Category: "apps", Type: "Appx"},
	}, &stubTweaksManager{check: func(tw entity.WindowsTweak) (bool, string, error) {
		if tw.ID == "t1" {
			return true, "applied", nil
		}
		return false, "drifted", nil
	}})

	var diags []entity.Diagnostic
	uc.auditDebloat(context.Background(), func(d entity.Diagnostic) { diags = append(diags, d) })

	if len(diags) != 2 {
		t.Fatalf("expected one line per category (2), got %d: %v", len(diags), diags)
	}
	for _, d := range diags {
		if d.System != "Debloat" {
			t.Errorf("expected System Debloat, got %q", d.System)
		}
		if d.Category == entity.DiagWarning || d.Category == entity.DiagError {
			t.Errorf("debloat drift must never WARN/ERROR, got %v for %q", d.Category, d.Target)
		}
		switch d.Target {
		case "category telemetry":
			if d.Category != entity.DiagInfo {
				t.Errorf("partial telemetry must be INFO, got %v", d.Category)
			}
			if !strings.Contains(d.Details, "1/2") {
				t.Errorf("telemetry details must show 1/2 applied, got %q", d.Details)
			}
			if !strings.Contains(d.FixHint, "run debloat") {
				t.Errorf("telemetry fix hint must point at run debloat, got %q", d.FixHint)
			}
		case "category apps":
			if d.Category != entity.DiagInfo {
				t.Errorf("drifted apps must be INFO, got %v", d.Category)
			}
		default:
			t.Errorf("unexpected debloat target %q", d.Target)
		}
	}
}

func TestAuditDebloatFullyAppliedIsOK(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("debloat audit is Windows-only")
	}
	uc := debloatAuditUseCase([]entity.WindowsTweak{
		{ID: "t1", Category: "telemetry", Type: "DWord"},
	}, &stubTweaksManager{check: func(_ entity.WindowsTweak) (bool, string, error) {
		return true, "applied", nil
	}})

	var diags []entity.Diagnostic
	uc.auditDebloat(context.Background(), func(d entity.Diagnostic) { diags = append(diags, d) })

	if len(diags) != 1 || diags[0].Category != entity.DiagOK {
		t.Fatalf("expected a single OK line, got %v", diags)
	}
}
