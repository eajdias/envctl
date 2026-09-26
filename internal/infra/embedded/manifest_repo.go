package embedded

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/eajdias/envctl/internal/domain/entity"
	"github.com/eajdias/envctl/internal/domain/repository"
)

type manifestRepository struct {
	embeddedFS  fs.FS
	localDir    string
	shellCache  *shellManifest
	shellErr    error
	shellLoaded bool
}

// NewManifestRepository creates a ManifestRepository backed by embedded assets and optional local directory.
func NewManifestRepository(embeddedFS fs.FS, localDir string) repository.ManifestRepository {
	return &manifestRepository{
		embeddedFS: embeddedFS,
		localDir:   localDir,
	}
}

func (m *manifestRepository) readManifestFile(filename string) ([]byte, error) {
	readLocal := func(path string) ([]byte, error) {
		data, err := os.ReadFile(path)
		if err == nil {
			return data, nil
		}
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to read local manifest %s: %w", path, err)
		}
		return nil, nil
	}

	// A local manifest is authoritative when present. Only a missing file may
	// fall through to the embedded asset; permission/I/O errors must not
	// silently activate a different profile.
	if m.localDir != "" {
		if data, err := readLocal(filepath.Join(m.localDir, "manifests", filename)); err != nil {
			return nil, err
		} else if data != nil {
			return data, nil
		}
	}
	if data, err := readLocal(filepath.Join("manifests", filename)); err != nil {
		return nil, err
	} else if data != nil {
		return data, nil
	}

	// Fallback to embedded filesystem
	embeddedPath := filepath.ToSlash(filepath.Join("manifests", filename))
	data, err := fs.ReadFile(m.embeddedFS, embeddedPath)
	if err != nil {
		return nil, fmt.Errorf("manifest file not found in disk or embedded FS (%s): %w", filename, err)
	}
	return data, nil
}

type packagesManifest struct {
	Packages []entity.Package `yaml:"packages"`
}

func loadManifestFile[T any](m *manifestRepository, filename string, newManifest func() T) (T, error) {
	var zero T
	data, err := m.readManifestFile(filename)
	if err != nil {
		return zero, err
	}
	manifest := newManifest()
	if err := yaml.Unmarshal(data, manifest); err != nil {
		return zero, fmt.Errorf("failed to parse %s: %w", filename, err)
	}
	return manifest, nil
}

func (m *manifestRepository) loadShell() (*shellManifest, error) {
	if m.shellLoaded {
		return m.shellCache, m.shellErr
	}
	m.shellLoaded = true
	manifest, err := loadManifestFile(m, "shell.yaml", func() *shellManifest { return &shellManifest{} })
	m.shellCache, m.shellErr = manifest, err
	return m.shellCache, m.shellErr
}

func (m *manifestRepository) LoadPackages() ([]entity.Package, error) {
	manifest, err := loadManifestFile(m, "packages.yaml", func() *packagesManifest { return &packagesManifest{} })
	if err != nil {
		return nil, err
	}
	return manifest.Packages, nil
}

func (m *manifestRepository) LoadGamingPackages() ([]entity.Package, error) {
	manifest, err := loadManifestFile(m, "gaming.yaml", func() *packagesManifest { return &packagesManifest{} })
	if err != nil {
		return nil, err
	}
	return manifest.Packages, nil
}

type shellManifest struct {
	EnvVars     []entity.EnvironmentVar `yaml:"environment_variables"`
	ConfigFiles []entity.ConfigFile     `yaml:"config_files"`
	Directories []entity.RestrictedDir  `yaml:"directories"`
	Cleanup     []entity.CleanupItem    `yaml:"cleanup"`
}

func (m *manifestRepository) LoadConfigFiles() ([]entity.ConfigFile, error) {
	manifest, err := m.loadShell()
	if err != nil {
		return nil, err
	}
	return manifest.ConfigFiles, nil
}

func (m *manifestRepository) LoadEnvVars() ([]entity.EnvironmentVar, error) {
	manifest, err := m.loadShell()
	if err != nil {
		return nil, err
	}
	return manifest.EnvVars, nil
}

func (m *manifestRepository) LoadDirectories() ([]entity.RestrictedDir, error) {
	manifest, err := m.loadShell()
	if err != nil {
		return nil, err
	}
	return manifest.Directories, nil
}

func (m *manifestRepository) LoadCleanupItems() ([]entity.CleanupItem, error) {
	manifest, err := m.loadShell()
	if err != nil {
		return nil, err
	}
	return manifest.Cleanup, nil
}

type skillsManifest struct {
	Skills []entity.Skill `yaml:"skills"`
}

func (m *manifestRepository) LoadSkills() ([]entity.Skill, error) {
	manifest, err := loadManifestFile(m, "skills.yaml", func() *skillsManifest { return &skillsManifest{} })
	if err != nil {
		return nil, err
	}
	return manifest.Skills, nil
}

type lspManifest struct {
	LSPs []entity.LSP `yaml:"lsps"`
}

func (m *manifestRepository) LoadLSPs() ([]entity.LSP, error) {
	manifest, err := loadManifestFile(m, "lsp.yaml", func() *lspManifest { return &lspManifest{} })
	if err != nil {
		return nil, err
	}
	return manifest.LSPs, nil
}

type gitManifest struct {
	Configs []entity.GitConfig `yaml:"configs"`
}

func (m *manifestRepository) LoadGitConfigs() ([]entity.GitConfig, error) {
	manifest, err := loadManifestFile(m, "git.yaml", func() *gitManifest { return &gitManifest{} })
	if err != nil {
		return nil, err
	}
	return manifest.Configs, nil
}

type windowsManifest struct {
	Tweaks []entity.WindowsTweak `yaml:"tweaks"`
}

func (m *manifestRepository) LoadWindowsTweaks() ([]entity.WindowsTweak, error) {
	manifest, err := loadManifestFile(m, "windows.yaml", func() *windowsManifest { return &windowsManifest{} })
	if err != nil {
		return nil, err
	}
	return manifest.Tweaks, nil
}

func (m *manifestRepository) LoadDebloatTweaks() ([]entity.WindowsTweak, error) {
	manifest, err := loadManifestFile(m, "debloat.yaml", func() *windowsManifest { return &windowsManifest{} })
	if err != nil {
		return nil, err
	}
	return manifest.Tweaks, nil
}

type performanceManifest struct {
	Profile  entity.PerformanceProfile `yaml:"profile"`
	Packages []entity.Package          `yaml:"packages"`
	Sysctls  []entity.SysctlSetting    `yaml:"sysctls"`
}

func (m *manifestRepository) LoadPerformanceSpec(profile entity.PerformanceProfile) (entity.PerformanceSpec, error) {
	filename := ""
	switch profile {
	case entity.PerformanceProfileUbuntu:
		filename = "performance_ubuntu.yaml"
	case entity.PerformanceProfileCachyOS:
		filename = "performance_cachyos.yaml"
	default:
		return entity.PerformanceSpec{}, fmt.Errorf("unsupported performance profile %q", profile)
	}

	data, err := m.readManifestFile(filename)
	if err != nil {
		return entity.PerformanceSpec{}, err
	}
	var manifest performanceManifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return entity.PerformanceSpec{}, fmt.Errorf("failed to parse %s: %w", filename, err)
	}
	if manifest.Profile != profile {
		return entity.PerformanceSpec{}, fmt.Errorf("%s declares profile %q, expected %q", filename, manifest.Profile, profile)
	}
	if profile == entity.PerformanceProfileCachyOS && len(manifest.Sysctls) > 0 {
		return entity.PerformanceSpec{}, fmt.Errorf("%s cannot declare sysctls for the CachyOS profile", filename)
	}
	return entity.PerformanceSpec{
		Profile:  manifest.Profile,
		Packages: manifest.Packages,
		Sysctls:  manifest.Sysctls,
	}, nil
}

func saveManifestFile(localDir, filename string, manifest any) error {
	data, err := yaml.Marshal(manifest)
	if err != nil {
		return err
	}
	dest := filepath.Join("manifests", filename)
	if localDir != "" {
		dest = filepath.Join(localDir, "manifests", filename)
	}
	_ = os.MkdirAll(filepath.Dir(dest), 0755)
	return os.WriteFile(dest, data, 0644)
}

func (m *manifestRepository) SaveSkills(skills []entity.Skill) error {
	return saveManifestFile(m.localDir, "skills.yaml", skillsManifest{Skills: skills})
}

func (m *manifestRepository) SaveGitConfigs(configs []entity.GitConfig) error {
	return saveManifestFile(m.localDir, "git.yaml", gitManifest{Configs: configs})
}
