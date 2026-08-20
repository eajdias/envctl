package repository

import (
	"context"
	"io/fs"
	"os"

	"github.com/eajdias/envctl/internal/domain/entity"
)

// PackageManager defines operations for installing and checking software packages.
type PackageManager interface {
	Type() entity.PackageType
	IsAvailable(ctx context.Context) bool
	IsInstalled(ctx context.Context, pkg entity.Package) (bool, string, error)
	Install(ctx context.Context, pkg entity.Package) error
	ListInstalled(ctx context.Context) ([]entity.Package, error)
}

// FileSystemManager provides file operations with atomic backup and path expansion.
type FileSystemManager interface {
	WriteWithBackup(destPath string, content []byte, perm os.FileMode) (backupCreated string, err error)
	ReadFile(path string) ([]byte, error)
	EnsureDirectory(path string, perm os.FileMode) error
	Exists(path string) bool
	ExpandUserPath(path string) (string, error)
	SetStrictWindowsACL(path string) error
	CopyEmbeddedTree(embeddedFS fs.FS, sourceDir, targetDir string) (int, error)
}

// ManifestRepository loads and saves declarative environment specifications.
type ManifestRepository interface {
	LoadPackages() ([]entity.Package, error)
	LoadConfigFiles() ([]entity.ConfigFile, error)
	LoadSkills() ([]entity.Skill, error)
	LoadLSPs() ([]entity.LSP, error)
	LoadEnvVars() ([]entity.EnvironmentVar, error)
	LoadGitConfigs() ([]entity.GitConfig, error)
	LoadDirectories() ([]entity.RestrictedDir, error)
	LoadWindowsTweaks() ([]entity.WindowsTweak, error)

	SavePackages(pkgs []entity.Package) error
	SaveSkills(skills []entity.Skill) error
	SaveLSPs(lsps []entity.LSP) error
	SaveGitConfigs(configs []entity.GitConfig) error
}

// GitManager handles global git configs, branches and GitHub PRs.
type GitManager interface {
	GetGlobalConfig(ctx context.Context, key string) (string, error)
	SetGlobalConfig(ctx context.Context, key, value string) error
	EnsureGlobalConfigs(ctx context.Context, configs []entity.GitConfig) ([]entity.Diagnostic, error)
	CreateSnapshotBranchAndPR(ctx context.Context, branchName, title, body string, changedFiles []string) (string, error)
}

// WindowsEnvManager manages Windows User and Machine environment variables.
type WindowsEnvManager interface {
	GetEnvVar(scope, name string) (string, error)
	SetEnvVar(scope, name, value string) error
	EnsureEnvVars(ctx context.Context, vars []entity.EnvironmentVar) ([]entity.Diagnostic, error)
}

// WindowsTweaksManager manages Windows 11 system registry tweaks, features and fonts.
type WindowsTweaksManager interface {
	CheckTweak(ctx context.Context, tweak entity.WindowsTweak) (bool, string, error)
	ApplyTweak(ctx context.Context, tweak entity.WindowsTweak) error
	EnsureTweaks(ctx context.Context, tweaks []entity.WindowsTweak) ([]entity.Diagnostic, error)
}

// Logger provides structured and persistent execution logging to disk.
type Logger interface {
	Info(format string, args ...any)
	Warn(format string, args ...any)
	Error(format string, args ...any)
	Debug(format string, args ...any)
	LogCommand(cmd string, args []string, exitCode int, output string, err error)
	LogIdempotency(system, target string, skipped bool, reason string)
	GetLogFilePath() string
	Close() error
}
