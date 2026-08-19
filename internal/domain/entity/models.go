package entity

import "fmt"

// PackageType defines the package manager type.
type PackageType string

const (
	PackageTypeWinget     PackageType = "winget"
	PackageTypePacman     PackageType = "pacman"
	PackageTypeVolta      PackageType = "volta"
	PackageTypeNpm        PackageType = "npm"
	PackageTypePip        PackageType = "pip"
	PackageTypeCargo      PackageType = "cargo"
	PackageTypeDotnetTool PackageType = "dotnet-tool"
)

// PackageStatus indicates the installation status of a package.
type PackageStatus string

const (
	StatusInstalled PackageStatus = "installed"
	StatusMissing   PackageStatus = "missing"
	StatusOutdated  PackageStatus = "outdated"
	StatusFailed    PackageStatus = "failed"
	StatusSkipped   PackageStatus = "skipped"
)

// Package represents a system or toolchain package to be managed.
type Package struct {
	ID           string        `yaml:"id"`
	Name         string        `yaml:"name"`
	Type         PackageType   `yaml:"type"`
	Category     string        `yaml:"category"`
	Version      string        `yaml:"version,omitempty"`
	CheckCommand string        `yaml:"check_command,omitempty"`
	Args         []string      `yaml:"args,omitempty"`
	Status       PackageStatus `yaml:"status,omitempty"`
	Error        string        `yaml:"error,omitempty"`
}

func (p Package) String() string {
	if p.Name != "" && p.Name != p.ID {
		return fmt.Sprintf("[%s] %s (%s)", p.Type, p.Name, p.ID)
	}
	return fmt.Sprintf("[%s] %s", p.Type, p.ID)
}

// ConfigFile represents a system or user configuration file.
type ConfigFile struct {
	ID          string `yaml:"id"`
	Description string `yaml:"description"`
	Source      string `yaml:"source"`      // path in embedded FS or template
	Destination string `yaml:"destination"` // target path with env vars expanded (e.g. ~ / %USERPROFILE%)
	StrictACL   bool   `yaml:"strict_acl"`  // Restrict to current user only (for SSH/keys)
	Category    string `yaml:"category"`
}

// Skill represents an OpenCode agent skill.
type Skill struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Source      string   `yaml:"source"` // directory inside configs/skills/ or repository URL
	TargetDir   string   `yaml:"target_dir"`
	Enabled     bool     `yaml:"enabled"`
	Files       []string `yaml:"files,omitempty"`
}

// LSP represents a Language Server Protocol configuration.
type LSP struct {
	ID            string      `yaml:"id"`
	Language      string      `yaml:"language"`
	ServerName    string      `yaml:"server_name"`
	Command       string      `yaml:"command"`
	Args          []string    `yaml:"args"`
	InstallType   PackageType `yaml:"install_type"`
	InstallTarget string      `yaml:"install_target"`
	CheckBinary   string      `yaml:"check_binary"`
}

// EnvironmentVar represents an OS environment variable.
type EnvironmentVar struct {
	Name   string `yaml:"name"`
	Value  string `yaml:"value"`
	Scope  string `yaml:"scope"` // "User" or "Machine"
	Target string `yaml:"target"`
}

// GitConfig represents a global Git configuration.
type GitConfig struct {
	Key   string `yaml:"key"`
	Value string `yaml:"value"`
}

// DiagnosticStatus represents health check status.
type DiagnosticStatus string

const (
	DiagOK      DiagnosticStatus = "OK"
	DiagWarning DiagnosticStatus = "WARNING"
	DiagError   DiagnosticStatus = "ERROR"
	DiagInfo    DiagnosticStatus = "INFO"
)

// Diagnostic contains the result of an audit check.
type Diagnostic struct {
	Category DiagnosticStatus `yaml:"status"` // Status (OK/WARN/ERROR)
	System   string           `yaml:"system"` // e.g. "Winget", "MSYS2", "Git", "Skills", "LSP"
	Target   string           `yaml:"target"`
	Details  string           `yaml:"details"`
	FixHint  string           `yaml:"fix_hint,omitempty"`
}
