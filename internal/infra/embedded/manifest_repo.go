package embedded

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/eajdias/envctl/internal/domain/entity"
)

type ManifestRepository struct {
	embeddedFS  fs.FS
	localDir    string
	shellCache  *shellManifest
	shellErr    error
	shellLoaded bool
}

// NewManifestRepository creates a ManifestRepository backed by embedded assets and optional local directory.
func NewManifestRepository(embeddedFS fs.FS, localDir string) *ManifestRepository {
	return &ManifestRepository{
		embeddedFS: embeddedFS,
		localDir:   localDir,
	}
}

func (m *ManifestRepository) readManifestFile(filename string) ([]byte, error) {
	// A local manifest override is authoritative when the caller provides a
	// localDir (development workflow or tests). Only a missing file falls
	// through to the embedded asset; permission/I/O errors must not silently
	// activate a different profile.
	if m.localDir != "" {
		path := filepath.Join(m.localDir, "manifests", filename)
		data, err := os.ReadFile(path)
		if err == nil {
			return data, nil
		}
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to read local manifest %s: %w", path, err)
		}
		// file not found in localDir — fall through to embedded
	}

	// Always prefer the embedded FS. A CWD-relative disk read was removed
	// here: it caused silent stale-manifest reads when the binary was run
	// from outside the repo root (lessons.md:17-18).
	embeddedPath := filepath.ToSlash(filepath.Join("manifests", filename))
	data, err := fs.ReadFile(m.embeddedFS, embeddedPath)
	if err != nil {
		return nil, fmt.Errorf("manifest %q not found in embedded FS: %w", filename, err)
	}
	return data, nil
}

type packagesManifest struct {
	Packages []entity.Package `yaml:"packages"`
}

func loadManifestFile[T any](m *ManifestRepository, filename string, newManifest func() T) (T, error) {
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

func (m *ManifestRepository) loadShell() (*shellManifest, error) {
	if m.shellLoaded {
		return m.shellCache, m.shellErr
	}
	m.shellLoaded = true
	manifest, err := loadManifestFile(m, "shell.yaml", func() *shellManifest { return &shellManifest{} })
	m.shellCache, m.shellErr = manifest, err
	return m.shellCache, m.shellErr
}

func (m *ManifestRepository) LoadPackages() ([]entity.Package, error) {
	manifest, err := loadManifestFile(m, "packages.yaml", func() *packagesManifest { return &packagesManifest{} })
	if err != nil {
		return nil, err
	}
	return manifest.Packages, nil
}

func (m *ManifestRepository) LoadGamingPackages() ([]entity.Package, error) {
	manifest, err := loadManifestFile(m, "gaming.yaml", func() *packagesManifest { return &packagesManifest{} })
	if err != nil {
		return nil, err
	}
	return manifest.Packages, nil
}

func (m *ManifestRepository) LoadExtrasPackages() ([]entity.Package, error) {
	manifest, err := loadManifestFile(m, "extras.yaml", func() *packagesManifest { return &packagesManifest{} })
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

func (m *ManifestRepository) LoadConfigFiles() ([]entity.ConfigFile, error) {
	manifest, err := m.loadShell()
	if err != nil {
		return nil, err
	}
	return expandConfigFileInstances(manifest.ConfigFiles), nil
}

// expandConfigFileInstances turns entries declaring instances: into one
// deployment per name, substituting {{name}} in id, source and destination.
// Entries without instances pass through untouched, so the expansion is a
// no-op for the rest of the manifest.
func expandConfigFileInstances(configs []entity.ConfigFile) []entity.ConfigFile {
	expanded := make([]entity.ConfigFile, 0, len(configs))
	for _, cf := range configs {
		if len(cf.Instances) == 0 {
			expanded = append(expanded, cf)
			continue
		}
		for _, name := range cf.Instances {
			instance := cf
			instance.Instances = nil
			instance.ID = strings.ReplaceAll(cf.ID, "{{name}}", name)
			instance.Source = strings.ReplaceAll(cf.Source, "{{name}}", name)
			instance.Destination = strings.ReplaceAll(cf.Destination, "{{name}}", name)
			expanded = append(expanded, instance)
		}
	}
	return expanded
}

func (m *ManifestRepository) LoadEnvVars() ([]entity.EnvironmentVar, error) {
	manifest, err := m.loadShell()
	if err != nil {
		return nil, err
	}
	return manifest.EnvVars, nil
}

func (m *ManifestRepository) LoadDirectories() ([]entity.RestrictedDir, error) {
	manifest, err := m.loadShell()
	if err != nil {
		return nil, err
	}
	return manifest.Directories, nil
}

func (m *ManifestRepository) LoadCleanupItems() ([]entity.CleanupItem, error) {
	manifest, err := m.loadShell()
	if err != nil {
		return nil, err
	}
	return manifest.Cleanup, nil
}

type skillsManifest struct {
	Skills []entity.Skill `yaml:"skills"`
}

func (m *ManifestRepository) LoadSkills() ([]entity.Skill, error) {
	manifest, err := loadManifestFile(m, "skills.yaml", func() *skillsManifest { return &skillsManifest{} })
	if err != nil {
		return nil, err
	}
	return manifest.Skills, nil
}

type lspManifest struct {
	LSPs []entity.LSP `yaml:"lsps"`
}

func (m *ManifestRepository) LoadLSPs() ([]entity.LSP, error) {
	manifest, err := loadManifestFile(m, "lsp.yaml", func() *lspManifest { return &lspManifest{} })
	if err != nil {
		return nil, err
	}
	return manifest.LSPs, nil
}

type gitManifest struct {
	Configs []entity.GitConfig `yaml:"configs"`
}

func (m *ManifestRepository) LoadGitConfigs() ([]entity.GitConfig, error) {
	manifest, err := loadManifestFile(m, "git.yaml", func() *gitManifest { return &gitManifest{} })
	if err != nil {
		return nil, err
	}
	return manifest.Configs, nil
}

type windowsManifest struct {
	Tweaks []entity.WindowsTweak `yaml:"tweaks"`
}

func (m *ManifestRepository) LoadWindowsTweaks() ([]entity.WindowsTweak, error) {
	manifest, err := loadManifestFile(m, "windows.yaml", func() *windowsManifest { return &windowsManifest{} })
	if err != nil {
		return nil, err
	}
	return manifest.Tweaks, nil
}

func (m *ManifestRepository) LoadDebloatTweaks() ([]entity.WindowsTweak, error) {
	manifest, err := loadManifestFile(m, "debloat_windows.yaml", func() *windowsManifest { return &windowsManifest{} })
	if err != nil {
		return nil, err
	}
	return manifest.Tweaks, nil
}

const linuxDebloatManifest = "debloat_linux.yaml"

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
	Swap             *entity.SwapSpec         `yaml:"swap,omitempty"`
	Debloat          *entity.DebloatSpec      `yaml:"debloat,omitempty"`
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

func (m *ManifestRepository) parsePerformanceManifest(filename string, expected entity.PerformanceProfile) (entity.PerformanceSpec, error) {
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
		Swap:             manifest.Swap,
		Debloat:          manifest.Debloat,
	}, nil
}

func (m *ManifestRepository) LoadPerformanceSpec(profile entity.PerformanceProfile) (entity.PerformanceSpec, error) {
	filename, ok := performanceManifestFile(profile)
	if !ok {
		return entity.PerformanceSpec{}, fmt.Errorf("unsupported performance profile %q", profile)
	}
	return m.parsePerformanceManifest(filename, profile)
}

// LoadLinuxDebloatSpec reads the standalone Linux removal manifest.
func (m *ManifestRepository) LoadLinuxDebloatSpec() (entity.DebloatSpec, error) {
	data, err := m.readManifestFile(linuxDebloatManifest)
	if err != nil {
		return entity.DebloatSpec{}, err
	}
	var manifest struct {
		NeedrestartDropin string                  `yaml:"needrestart_dropin"`
		Removals          []entity.PackageRemoval `yaml:"removals"`
		Version           string                  `yaml:"version"`
	}
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return entity.DebloatSpec{}, fmt.Errorf("failed to parse %s: %w", linuxDebloatManifest, err)
	}
	spec := entity.DebloatSpec{
		NeedrestartDropin: manifest.NeedrestartDropin,
		Removals:          manifest.Removals,
	}
	if err := entity.ValidateDebloatSpec(spec); err != nil {
		return entity.DebloatSpec{}, fmt.Errorf("%s: %w", linuxDebloatManifest, err)
	}
	return spec, nil
}

// ListPerformanceProfiles reports every shipped profile with the release floor
// its manifest declares, so the CLI can select one without knowing a version.
func (m *ManifestRepository) ListPerformanceProfiles() ([]entity.PerformanceProfileMeta, error) {
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
	//nolint:gosec // G301: manifests are repo content (shared, not secrets).
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return fmt.Errorf("failed to create manifest dir for %s: %w", dest, err)
	}
	//nolint:gosec // G306: manifests are repo content (world-readable by design), never secrets.
	return os.WriteFile(dest, data, 0644)
}

func (m *ManifestRepository) SaveSkills(skills []entity.Skill) error {
	return saveManifestFile(m.localDir, "skills.yaml", skillsManifest{Skills: skills})
}

func (m *ManifestRepository) SaveGitConfigs(configs []entity.GitConfig) error {
	return saveManifestFile(m.localDir, "git.yaml", gitManifest{Configs: configs})
}
