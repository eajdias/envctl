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
	Profile entity.PerformanceProfile `yaml:"profile"`
	// MinDistroVersion is the release floor. It is manifest data so raising
	// the floor never requires a code change, and so a profile identity never
	// has to encode a version the fleet has already moved past.
	MinDistroVersion string                   `yaml:"min_distro_version,omitempty"`
	Packages         []entity.Package         `yaml:"packages"`
	Sysctls          []entity.SysctlSetting   `yaml:"sysctls"`
	Tiers            []entity.PerformanceTier `yaml:"tiers,omitempty"`
	Timezone         *entity.TimezoneSpec     `yaml:"timezone,omitempty"`
	Journald         *entity.JournaldSpec     `yaml:"journald,omitempty"`
	Limits           *entity.LimitsSpec       `yaml:"limits,omitempty"`
	ZRAM             *entity.ZRAMSpec         `yaml:"zram,omitempty"`
}

// performanceManifests is the single profile -> file map plus a deterministic
// discovery order, so ListPerformanceProfiles never depends on map iteration.
var performanceManifests = []struct {
	Profile entity.PerformanceProfile
	File    string
}{
	{entity.PerformanceProfileUbuntuServer, "performance_ubuntu.yaml"},
	{entity.PerformanceProfileCachyOS, "performance_cachyos.yaml"},
}

func performanceManifestFile(profile entity.PerformanceProfile) (string, bool) {
	for _, entry := range performanceManifests {
		if entry.Profile == profile {
			return entry.File, true
		}
	}
	return "", false
}

func (m *manifestRepository) parsePerformanceManifest(filename string, expected entity.PerformanceProfile) (entity.PerformanceSpec, error) {
	data, err := m.readManifestFile(filename)
	if err != nil {
		return entity.PerformanceSpec{}, err
	}
	var manifest performanceManifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return entity.PerformanceSpec{}, fmt.Errorf("failed to parse %s: %w", filename, err)
	}
	if manifest.Profile != expected {
		return entity.PerformanceSpec{}, fmt.Errorf("%s declares profile %q, expected %q", filename, manifest.Profile, expected)
	}
	if expected == entity.PerformanceProfileCachyOS && len(manifest.Sysctls) > 0 {
		return entity.PerformanceSpec{}, fmt.Errorf("%s cannot declare sysctls for the CachyOS profile", filename)
	}
	// Tiers are only validated when a profile declares them. The CachyOS
	// profile deliberately has none: its zram is unconditional, so a band list
	// would imply a memory policy it does not have.
	if len(manifest.Tiers) > 0 {
		if err := entity.ValidatePerformanceTiers(manifest.Tiers); err != nil {
			return entity.PerformanceSpec{}, fmt.Errorf("%s: %w", filename, err)
		}
	}
	return entity.PerformanceSpec{
		Profile:          manifest.Profile,
		MinDistroVersion: manifest.MinDistroVersion,
		Packages:         manifest.Packages,
		Sysctls:          manifest.Sysctls,
		Tiers:            manifest.Tiers,
		Timezone:         manifest.Timezone,
		Journald:         manifest.Journald,
		Limits:           manifest.Limits,
		ZRAM:             manifest.ZRAM,
	}, nil
}

func (m *manifestRepository) LoadPerformanceSpec(profile entity.PerformanceProfile) (entity.PerformanceSpec, error) {
	filename, ok := performanceManifestFile(profile)
	if !ok {
		return entity.PerformanceSpec{}, fmt.Errorf("unsupported performance profile %q", profile)
	}
	return m.parsePerformanceManifest(filename, profile)
}

// ListPerformanceProfiles reports every shipped profile with the release floor
// its manifest declares, so the CLI can select one without knowing a version.
func (m *manifestRepository) ListPerformanceProfiles() ([]entity.PerformanceProfileMeta, error) {
	metas := make([]entity.PerformanceProfileMeta, 0, len(performanceManifests))
	for _, entry := range performanceManifests {
		spec, err := m.parsePerformanceManifest(entry.File, entry.Profile)
		if err != nil {
			return nil, err
		}
		metas = append(metas, entity.PerformanceProfileMeta{
			Profile:          spec.Profile,
			MinDistroVersion: spec.MinDistroVersion,
			ManifestFile:     entry.File,
		})
	}
	return metas, nil
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
