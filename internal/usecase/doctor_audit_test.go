package usecase

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"testing"

	"github.com/eajdias/envctl/internal/domain/entity"
	"github.com/eajdias/envctl/internal/domain/repository"
)

// mockManifestRepo implements repository.ManifestRepository for testing.
type mockManifestRepo struct {
	pkgs         []entity.Package
	configFiles  []entity.ConfigFile
	skills       []entity.Skill
	lsps         []entity.LSP
	envVars      []entity.EnvironmentVar
	gitConfigs   []entity.GitConfig
	directories  []entity.RestrictedDir
	cleanupItems []entity.CleanupItem
	tweaks       []entity.WindowsTweak
}

func (m *mockManifestRepo) LoadPackages() ([]entity.Package, error)         { return m.pkgs, nil }
func (m *mockManifestRepo) LoadConfigFiles() ([]entity.ConfigFile, error)   { return m.configFiles, nil }
func (m *mockManifestRepo) LoadSkills() ([]entity.Skill, error)             { return m.skills, nil }
func (m *mockManifestRepo) LoadLSPs() ([]entity.LSP, error)                 { return m.lsps, nil }
func (m *mockManifestRepo) LoadEnvVars() ([]entity.EnvironmentVar, error)   { return m.envVars, nil }
func (m *mockManifestRepo) LoadGitConfigs() ([]entity.GitConfig, error)     { return m.gitConfigs, nil }
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
			`C:\Program Files\Google\Chrome\Application\chrome.exe`: true,
			`node_modules/playwright`:                               true,
		},
		fileContents: map[string][]byte{},
	}
	logger := &mockLogger{}

	uc := NewDoctorAuditUseCase(
		manifestRepo,
		fsManager,
		nil,
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
		nil,
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
