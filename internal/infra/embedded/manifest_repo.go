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
	embeddedFS fs.FS
	localDir   string
}

// NewManifestRepository creates a ManifestRepository backed by embedded assets and optional local directory.
func NewManifestRepository(embeddedFS fs.FS, localDir string) repository.ManifestRepository {
	return &manifestRepository{
		embeddedFS: embeddedFS,
		localDir:   localDir,
	}
}

func (m *manifestRepository) readManifestFile(filename string) ([]byte, error) {
	// Try local directory first if specified and file exists
	if m.localDir != "" {
		localPath := filepath.Join(m.localDir, "manifests", filename)
		if data, err := os.ReadFile(localPath); err == nil {
			return data, nil
		}
		// Also check direct manifests folder relative to current working directory
		if data, err := os.ReadFile(filepath.Join("manifests", filename)); err == nil {
			return data, nil
		}
	} else {
		if data, err := os.ReadFile(filepath.Join("manifests", filename)); err == nil {
			return data, nil
		}
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

func (m *manifestRepository) LoadPackages() ([]entity.Package, error) {
	data, err := m.readManifestFile("packages.yaml")
	if err != nil {
		return nil, err
	}
	var manifest packagesManifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse packages.yaml: %w", err)
	}
	return manifest.Packages, nil
}

type shellManifest struct {
	EnvVars     []entity.EnvironmentVar `yaml:"environment_variables"`
	ConfigFiles []entity.ConfigFile     `yaml:"config_files"`
}

func (m *manifestRepository) LoadConfigFiles() ([]entity.ConfigFile, error) {
	data, err := m.readManifestFile("shell.yaml")
	if err != nil {
		return nil, err
	}
	var manifest shellManifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse shell.yaml config_files: %w", err)
	}
	return manifest.ConfigFiles, nil
}

func (m *manifestRepository) LoadEnvVars() ([]entity.EnvironmentVar, error) {
	data, err := m.readManifestFile("shell.yaml")
	if err != nil {
		return nil, err
	}
	var manifest shellManifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse shell.yaml env_vars: %w", err)
	}
	return manifest.EnvVars, nil
}

type skillsManifest struct {
	Skills []entity.Skill `yaml:"skills"`
}

func (m *manifestRepository) LoadSkills() ([]entity.Skill, error) {
	data, err := m.readManifestFile("skills.yaml")
	if err != nil {
		return nil, err
	}
	var manifest skillsManifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse skills.yaml: %w", err)
	}
	return manifest.Skills, nil
}

type lspManifest struct {
	LSPs []entity.LSP `yaml:"lsps"`
}

func (m *manifestRepository) LoadLSPs() ([]entity.LSP, error) {
	data, err := m.readManifestFile("lsp.yaml")
	if err != nil {
		return nil, err
	}
	var manifest lspManifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse lsp.yaml: %w", err)
	}
	return manifest.LSPs, nil
}

type gitManifest struct {
	Configs []entity.GitConfig `yaml:"configs"`
}

func (m *manifestRepository) LoadGitConfigs() ([]entity.GitConfig, error) {
	data, err := m.readManifestFile("git.yaml")
	if err != nil {
		return nil, err
	}
	var manifest gitManifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse git.yaml: %w", err)
	}
	return manifest.Configs, nil
}

type windowsManifest struct {
	Tweaks []entity.WindowsTweak `yaml:"tweaks"`
}

func (m *manifestRepository) LoadWindowsTweaks() ([]entity.WindowsTweak, error) {
	data, err := m.readManifestFile("windows.yaml")
	if err != nil {
		return nil, err
	}
	var manifest windowsManifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse windows.yaml: %w", err)
	}
	return manifest.Tweaks, nil
}

func (m *manifestRepository) SavePackages(pkgs []entity.Package) error {
	manifest := packagesManifest{Packages: pkgs}
	data, err := yaml.Marshal(manifest)
	if err != nil {
		return err
	}
	dest := filepath.Join("manifests", "packages.yaml")
	if m.localDir != "" {
		dest = filepath.Join(m.localDir, "manifests", "packages.yaml")
	}
	_ = os.MkdirAll(filepath.Dir(dest), 0755)
	return os.WriteFile(dest, data, 0644)
}

func (m *manifestRepository) SaveSkills(skills []entity.Skill) error {
	manifest := skillsManifest{Skills: skills}
	data, err := yaml.Marshal(manifest)
	if err != nil {
		return err
	}
	dest := filepath.Join("manifests", "skills.yaml")
	if m.localDir != "" {
		dest = filepath.Join(m.localDir, "manifests", "skills.yaml")
	}
	_ = os.MkdirAll(filepath.Dir(dest), 0755)
	return os.WriteFile(dest, data, 0644)
}

func (m *manifestRepository) SaveLSPs(lsps []entity.LSP) error {
	manifest := lspManifest{LSPs: lsps}
	data, err := yaml.Marshal(manifest)
	if err != nil {
		return err
	}
	dest := filepath.Join("manifests", "lsp.yaml")
	if m.localDir != "" {
		dest = filepath.Join(m.localDir, "manifests", "lsp.yaml")
	}
	_ = os.MkdirAll(filepath.Dir(dest), 0755)
	return os.WriteFile(dest, data, 0644)
}

func (m *manifestRepository) SaveGitConfigs(configs []entity.GitConfig) error {
	manifest := gitManifest{Configs: configs}
	data, err := yaml.Marshal(manifest)
	if err != nil {
		return err
	}
	dest := filepath.Join("manifests", "git.yaml")
	if m.localDir != "" {
		dest = filepath.Join(m.localDir, "manifests", "git.yaml")
	}
	_ = os.MkdirAll(filepath.Dir(dest), 0755)
	return os.WriteFile(dest, data, 0644)
}
