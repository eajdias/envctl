package executil

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// ToolchainDirs is the single source of truth for the user toolchain bin
// directories, highest precedence first. Every place that used to rebuild
// this list by hand (bootstrap env, provider probes, manager execution)
// derives from here, so a new toolchain directory is added once and can no
// longer drift between probe and execution.
func ToolchainDirs(home string) []string {
	return []string{
		filepath.Join(home, ".opencode", "bin"),
		filepath.Join(home, ".local", "bin"),
		filepath.Join(home, ".local", "share", "mise", "shims"),
		filepath.Join(home, "go", "bin"),
	}
}

// MiseShimDir is where `mise install` links shims: %LOCALAPPDATA% on
// Windows, ~/.local/share/mise on POSIX.
func MiseShimDir(home string) string {
	if runtime.GOOS == "windows" {
		if localApp := os.Getenv("LOCALAPPDATA"); localApp != "" {
			return filepath.Join(localApp, "mise", "shims")
		}
	}
	return filepath.Join(home, ".local", "share", "mise", "shims")
}

// VoltaBinDir locates the legacy Volta shim directory from the pre-mise
// toolchain era. Volta shims shadow mise shims on PATH, so the legacy sweep
// archives them once the mise shim for the same binary exists.
func VoltaBinDir(home string) string {
	if runtime.GOOS == "windows" {
		if localApp := os.Getenv("LOCALAPPDATA"); localApp != "" {
			return filepath.Join(localApp, "Volta", "bin")
		}
		return ""
	}
	return filepath.Join(home, ".volta", "bin")
}

// ToolchainPath joins ToolchainDirs with the process PATH.
func ToolchainPath(home string) string {
	return strings.Join(append(ToolchainDirs(home), os.Getenv("PATH")), string(os.PathListSeparator))
}

// ToolchainEnv builds an environment that resolves mise shims, user-local
// binaries and Go, shared by the bootstrap and doctor use cases, without
// mutating the process environment.
func ToolchainEnv(home string) []string {
	env := []string{
		"PATH=" + ToolchainPath(home),
		"GOPATH=" + filepath.Join(home, "go"),
	}
	for _, kv := range os.Environ() {
		key := kv[:strings.IndexByte(kv, '=')]
		if key == "PATH" {
			continue
		}
		env = append(env, kv)
	}
	return env
}

// ExecTool builds an exec.Cmd resolved against the toolchain PATH on Linux.
// envctl often runs from non-login shells (ssh, systemd) where those shims
// are absent from the default PATH — without this, every toolchain check
// would falsely report packages as missing.
func ExecTool(ctx context.Context, name string, args ...string) *exec.Cmd {
	if runtime.GOOS == "linux" {
		if home, err := os.UserHomeDir(); err == nil && home != "" {
			toolchainPath := ToolchainPath(home)
			// exec.LookPath only consults the process PATH, so resolve the
			// binary explicitly against the toolchain PATH and hand the
			// absolute path to exec.Command.
			if resolved, err := LookPathIn(toolchainPath, name); err == nil {
				cmd := exec.CommandContext(ctx, resolved, args...)
				env := os.Environ()
				for i, kv := range env {
					if strings.HasPrefix(kv, "PATH=") {
						env[i] = "PATH=" + toolchainPath
						break
					}
				}
				cmd.Env = env
				return cmd
			}
		}
	}
	return exec.CommandContext(ctx, name, args...)
}

// LookPathIn resolves a bare command name against an explicit PATH value,
// mirroring exec.LookPath. An empty PATH is an explicit miss, not a reason to
// fall back to the process PATH.
func LookPathIn(pathValue, name string) (string, error) {
	if pathValue == "" {
		return "", fmt.Errorf("environment declares no PATH")
	}
	if filepath.IsAbs(name) {
		return name, nil
	}
	for _, dir := range filepath.SplitList(pathValue) {
		if dir == "" {
			dir = "."
		}
		candidate := filepath.Join(dir, name)
		if IsExecutableFile(candidate) {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("%s not found in PATH", name)
}

// IsExecutableFile reports whether path exists, is a regular file and carries
// an executable bit. Deployed helper scripts (the local verifier and the git
// hooks) are inert without it, so the audit treats a missing bit as drift.
// On Windows the bit is synthesized from file attributes and is unreliable
// for provisioned scripts, so existence of a regular file is enough there;
// POSIX hosts keep the strict bit check.
func IsExecutableFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}
	if runtime.GOOS == "windows" {
		return true
	}
	return info.Mode()&0o111 != 0
}
