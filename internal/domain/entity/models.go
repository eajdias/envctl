package entity

import "fmt"

// PackageType defines the package manager type.
type PackageType string

const (
	PackageTypeWinget PackageType = "winget"
	PackageTypeVolta  PackageType = "volta"
	PackageTypeNpm    PackageType = "npm"
	PackageTypePip    PackageType = "pip"
	PackageTypeGo     PackageType = "go"
	PackageTypeApt    PackageType = "apt"
	PackageTypePacman PackageType = "pacman"
	PackageTypeParu   PackageType = "paru"
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
	OS           string        `yaml:"os,omitempty"` // "windows", "linux" or empty for all
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

// MergeMode defines how a managed config file combines the template with the
// user-owned content already on disk. The zero value replaces the file (with
// the usual timestamped backup), which is only safe for files envctl fully
// owns. Files the user is expected to edit must declare a merge mode so
// provisioning never silently discards their entries.
type MergeMode string

const (
	// MergeOverwrite replaces the destination with the template (default).
	MergeOverwrite MergeMode = ""
	// MergeSSHHosts keeps every `Host` block found in the destination that the
	// template does not define, inserting them before the first template
	// `Host` entry so specific hosts keep precedence over `Host *`.
	MergeSSHHosts MergeMode = "ssh_hosts"
	// MergeJSONDeps merges the destination's JSON object keys over the
	// template, unioning `dependencies`/`devDependencies` so user-added
	// entries survive while template entries stay current.
	MergeJSONDeps MergeMode = "json_deps"
)

// ConfigFile represents a system or user configuration file.
type ConfigFile struct {
	ID            string    `yaml:"id"`
	Description   string    `yaml:"description"`
	Source        string    `yaml:"source"`      // path in embedded FS or template
	Destination   string    `yaml:"destination"` // target path with env vars expanded (e.g. ~ / %USERPROFILE%)
	StrictACL     bool      `yaml:"strict_acl"`  // Restrict to current user only (for SSH/keys)
	Category      string    `yaml:"category"`
	OS            string    `yaml:"os,omitempty"`              // "windows", "linux", "darwin", distro family ("arch"/"debian") or empty for all
	SeedIfMissing bool      `yaml:"seed_if_missing,omitempty"` // write baseline only when destination does not exist (e.g. agent memory templates)
	Merge         MergeMode `yaml:"merge,omitempty"`           // non-destructive merge with the existing user content
	// RuntimeManaged marks a file the agent itself writes to while it runs
	// (CommandCode appends approved commands to settings.json). Provisioning
	// still realigns it to the template — that is the cleanup — but the audit
	// must not report the runtime's own writes as drift.
	RuntimeManaged bool `yaml:"runtime_managed,omitempty"`
	Executable     bool `yaml:"executable,omitempty"` // chmod +x after write (POSIX scripts deployed to ~/bin-style dirs)
}

// Skill represents an agent skill deployed to OpenCode and CommandCode.
type Skill struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Source      string   `yaml:"source"` // directory inside configs/skills/ or repository URL
	TargetDir   string   `yaml:"target_dir"`
	Enabled     bool     `yaml:"enabled"`
	OS          string   `yaml:"os,omitempty"` // "windows", "linux" or empty for all
	Files       []string `yaml:"files,omitempty"`
}

// AppliesToOS reports whether the skill belongs on goos. An empty OS means the
// skill is portable and is deployed everywhere; a scoped skill is skipped (and
// pruned) on every other platform, so it never pollutes that machine's
// catalog. Distro families ("arch", "debian") are honored through MatchOS.
func (s Skill) AppliesToOS(goos string) bool {
	if s.OS == "" || s.OS == goos {
		return true
	}
	return MatchOS(s.OS, goos, DetectedDistro())
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
	OS            string      `yaml:"os,omitempty"` // "windows", "linux" or empty for all
}

// EnvironmentVar represents an OS environment variable.
type EnvironmentVar struct {
	Name   string `yaml:"name"`
	Value  string `yaml:"value"`
	Scope  string `yaml:"scope"` // "User" or "Machine"
	Target string `yaml:"target"`
	OS     string `yaml:"os,omitempty"` // "windows", "linux" or empty for all
}

// WindowsTweak represents a Windows OS setting, registry key or system customization.
type WindowsTweak struct {
	ID          string `yaml:"id"`
	Description string `yaml:"description"`
	Path        string `yaml:"path"`  // Registry Path e.g. "HKCU:\Software\Microsoft\Windows\CurrentVersion\Explorer\Advanced"
	Name        string `yaml:"name"`  // Registry Value Name e.g. "HideFileExt"
	Value       any    `yaml:"value"` // Target Value e.g. 0 or 1
	Type        string `yaml:"type"`  // "DWord", "String", "Binary", "Feature", "Font"
	Category    string `yaml:"category"`
}

// TweakCheckResult is the outcome of checking one Windows tweak. Batch
// checks return one per input tweak (order-preserving) so callers can audit
// dozens of tweaks with ~3 PowerShell spawns instead of one per tweak.
type TweakCheckResult struct {
	Tweak   WindowsTweak
	OK      bool
	Details string
	Err     error
}

// GitConfig represents a global Git configuration.
type GitConfig struct {
	Key   string `yaml:"key"`
	Value string `yaml:"value"`
	OS    string `yaml:"os,omitempty"` // "windows", "linux" or empty for all
}

// RestrictedDir represents a directory to be created with optional strict permissions.
type RestrictedDir struct {
	Path        string `yaml:"path"`
	StrictACL   bool   `yaml:"strict_acl"`
	Description string `yaml:"description"`
	Category    string `yaml:"category,omitempty"` // agent subsystem ("opencode", "commandcode") or empty for machine-level
	OS          string `yaml:"os,omitempty"`       // "windows", "linux" or empty for all
}

// CleanupItem represents a stale file or directory to remove during provisioning.
type CleanupItem struct {
	ID          string `yaml:"id"`
	Description string `yaml:"description"`
	Path        string `yaml:"path"` // expanded with env vars (e.g. ~ / %USERPROFILE%)
	Category    string `yaml:"category"`
	OS          string `yaml:"os,omitempty"`        // "windows", "linux" or empty for all
	Recursive   bool   `yaml:"recursive,omitempty"` // remove directory tree (os.RemoveAll)
	// KeepNewest prunes timestamped backups (<name>.bak.YYYYMMDD-HHMMSS) inside
	// the Path directory, keeping the newest N per original file (0 = disabled).
	KeepNewest int `yaml:"keep_newest,omitempty"`
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
	System   string           `yaml:"system"` // e.g. "Winget", "Git", "Skills", "LSP"
	Target   string           `yaml:"target"`
	Details  string           `yaml:"details"`
	FixHint  string           `yaml:"fix_hint,omitempty"`
}
