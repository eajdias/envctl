package filesystem

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/eajdias/envctl/internal/domain/repository"
)

type fsManager struct{}

// NewFileSystemManager creates a new FileSystemManager instance.
func NewFileSystemManager() repository.FileSystemManager {
	return &fsManager{}
}

// ExpandUserPath resolves paths with ~, %USERPROFILE%, %APPDATA%, and forward slashes.
func (f *fsManager) ExpandUserPath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("empty path provided")
	}

	userHome, err := os.UserHomeDir()
	if err != nil {
		if runtime.GOOS == "windows" {
			userHome = os.Getenv("USERPROFILE")
			if userHome == "" {
				userHome = "C:\\Users\\Default"
			}
		} else {
			userHome = os.Getenv("HOME")
			if userHome == "" {
				userHome = "/root"
			}
		}
	}

	normalized := path
	if runtime.GOOS == "windows" {
		normalized = strings.ReplaceAll(path, "/", "\\")
		if strings.HasPrefix(normalized, "~\\") || normalized == "~" {
			normalized = filepath.Join(userHome, strings.TrimPrefix(normalized, "~"))
		}

		// Expand Windows %VAR% syntax (e.g. %LOCALAPPDATA%, %APPDATA%, %USERPROFILE%).
		// Undefined variables are kept literal so they are never silently dropped.
		var sb strings.Builder
		cursor := 0
		for {
			start := strings.Index(normalized[cursor:], "%")
			if start == -1 {
				sb.WriteString(normalized[cursor:])
				break
			}
			start += cursor
			endRel := strings.Index(normalized[start+1:], "%")
			if endRel == -1 {
				sb.WriteString(normalized[cursor:])
				break
			}
			end := start + 1 + endRel
			varName := normalized[start+1 : end]
			sb.WriteString(normalized[cursor:start])
			if val := os.Getenv(varName); val != "" {
				sb.WriteString(val)
			} else {
				sb.WriteString(normalized[start : end+1])
			}
			cursor = end + 1
		}
		normalized = sb.String()
	} else {
		if strings.HasPrefix(normalized, "~/") || normalized == "~" {
			normalized = filepath.Join(userHome, strings.TrimPrefix(normalized, "~"))
		}
	}

	// Expand POSIX $VAR or ${VAR} syntax
	normalized = os.ExpandEnv(normalized)

	return filepath.Clean(normalized), nil
}

// Exists checks if a file or directory exists.
func (f *fsManager) Exists(path string) bool {
	expanded, err := f.ExpandUserPath(path)
	if err != nil {
		return false
	}
	_, err = os.Stat(expanded)
	return err == nil
}

// EnsureDirectory creates directory hierarchy if not present.
func (f *fsManager) EnsureDirectory(path string, perm os.FileMode) error {
	expanded, err := f.ExpandUserPath(path)
	if err != nil {
		return err
	}
	return os.MkdirAll(expanded, perm)
}

// ReadFile reads the full contents of a file.
func (f *fsManager) ReadFile(path string) ([]byte, error) {
	expanded, err := f.ExpandUserPath(path)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(expanded)
}

// WriteWithBackup writes content to target file. If destination exists and differs,
// an atomic timestamped backup (.bak.YYYYMMDD-HHMMSS) is created first.
func (f *fsManager) WriteWithBackup(destPath string, content []byte, perm os.FileMode) (string, error) {
	expanded, err := f.ExpandUserPath(destPath)
	if err != nil {
		return "", fmt.Errorf("failed to expand path %s: %w", destPath, err)
	}

	dir := filepath.Dir(expanded)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	var backupPath string
	if f.Exists(expanded) {
		existingData, err := os.ReadFile(expanded)
		if err == nil {
			// If content is already identical, no-op
			if string(existingData) == string(content) {
				return "", nil
			}

			// Content changed: create timestamped backup (same permission as target file)
			timestamp := time.Now().Format("20060102-150405")
			backupPath = fmt.Sprintf("%s.bak.%s", expanded, timestamp)
			if err := os.WriteFile(backupPath, existingData, perm); err != nil {
				return "", fmt.Errorf("failed to create backup file %s: %w", backupPath, err)
			}
		}
	}

	// Write the new content
	if err := os.WriteFile(expanded, content, perm); err != nil {
		return "", fmt.Errorf("failed to write file %s: %w", expanded, err)
	}

	return backupPath, nil
}

// SetStrictWindowsACL restricts file/directory permissions to the current user only.
// On Windows, it uses icacls to remove inheritance and grant full control to the user.
// On Linux/macOS, it sets POSIX mode 0700 for directories or 0600 for files.
func (f *fsManager) SetStrictWindowsACL(path string) error {
	expanded, err := f.ExpandUserPath(path)
	if err != nil {
		return err
	}

	if runtime.GOOS != "windows" {
		fi, err := os.Stat(expanded)
		if err != nil {
			return nil
		}
		if fi.IsDir() {
			return os.Chmod(expanded, 0700)
		}
		return os.Chmod(expanded, 0600)
	}

	currentUser := os.Getenv("USERNAME")
	if currentUser == "" {
		currentUser = os.Getenv("USER")
	}
	if currentUser == "" {
		return fmt.Errorf("could not determine current username for ACLs")
	}

	// icacls command: disable inheritance and grant full control to current user
	cmd := exec.Command("icacls.exe", expanded, "/inheritance:r", "/grant:r", fmt.Sprintf("%s:(OI)(CI)F", currentUser))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("icacls failed for %s: %s (%w)", expanded, string(output), err)
	}

	return nil
}

// CopyEmbeddedTree copies all files from an embedded fs.FS folder into a target directory.
func (f *fsManager) CopyEmbeddedTree(embeddedFS fs.FS, sourceDir, targetDir string) (int, error) {
	expandedTarget, err := f.ExpandUserPath(targetDir)
	if err != nil {
		return 0, err
	}

	count := 0
	err = fs.WalkDir(embeddedFS, sourceDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Calculate relative path
		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}

		targetPath := filepath.Join(expandedTarget, relPath)

		if d.IsDir() {
			return os.MkdirAll(targetPath, 0755)
		}

		// Read embedded file
		data, err := fs.ReadFile(embeddedFS, path)
		if err != nil {
			return fmt.Errorf("failed to read embedded file %s: %w", path, err)
		}

		// Ensure parent directory exists
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return err
		}

		// Write to disk
		if err := os.WriteFile(targetPath, data, 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", targetPath, err)
		}

		count++
		return nil
	})

	return count, err
}
