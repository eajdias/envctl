package filesystem

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type FileSystemManager struct{}

// NewFileSystemManager creates a new FileSystemManager instance.
func NewFileSystemManager() *FileSystemManager {
	return &FileSystemManager{}
}

// ExpandUserPath resolves paths with ~, %USERPROFILE%, %APPDATA%, and forward slashes.
func (f *FileSystemManager) ExpandUserPath(path string) (string, error) {
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
func (f *FileSystemManager) Exists(path string) bool {
	expanded, err := f.ExpandUserPath(path)
	if err != nil {
		return false
	}
	_, err = os.Stat(expanded)
	return err == nil
}

// EnsureDirectory creates directory hierarchy if not present.
func (f *FileSystemManager) EnsureDirectory(path string, perm os.FileMode) error {
	expanded, err := f.ExpandUserPath(path)
	if err != nil {
		return err
	}
	return os.MkdirAll(expanded, perm)
}

// ReadFile reads the full contents of a file.
func (f *FileSystemManager) ReadFile(path string) ([]byte, error) {
	expanded, err := f.ExpandUserPath(path)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(expanded)
}

// BackupPathFor returns a unique timestamped backup path for a live file.
// Same-second writes get -1, -2, … suffixes so no backup is destroyed.
// Stat (not Lstat) decides collisions: a dangling symlink is not a live file
// worth preserving under a new stamp.
func BackupPathFor(livePath string) string {
	stamp := time.Now().Format("20060102-150405")
	candidate := fmt.Sprintf("%s.bak.%s", livePath, stamp)
	for i := 1; ; i++ {
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate
		}
		candidate = fmt.Sprintf("%s.bak.%s-%d", livePath, stamp, i)
	}
}

// ArchivePath moves a live path aside to a unique timestamped backup name
// and returns it. Lstat (not Stat) decides collisions: a symlink itself is
// archived, never followed. This is the single implementation behind every
// "move aside" in provisioning; name clashes resolve with -1, -2, … suffixes.
func ArchivePath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("empty path")
	}
	if _, err := os.Lstat(path); err != nil {
		return "", err
	}
	stamp := time.Now().Format("20060102-150405")
	candidate := fmt.Sprintf("%s.bak.%s", path, stamp)
	for i := 1; ; i++ {
		if _, err := os.Lstat(candidate); os.IsNotExist(err) {
			break
		}
		candidate = fmt.Sprintf("%s.bak.%s-%d", path, stamp, i)
	}
	if err := os.Rename(path, candidate); err != nil {
		return "", err
	}
	return candidate, nil
}

// storeWithBackup writes content to an already-expanded absolute target path,
// backing up differing live bytes first. existing carries the live bytes when
// the caller already read them (nil = target absent or unreadable, write
// straight through). The backup keeps backupMode while the live file gets
// perm; identical content is a no-op returning "".
func storeWithBackup(target string, existing, content []byte, perm, backupMode os.FileMode) (string, error) {
	var backupPath string
	if existing != nil {
		if bytes.Equal(existing, content) {
			return "", nil
		}
		backupPath = BackupPathFor(target)
		//nolint:gosec // G703: target derives from ExpandUserPath / the manifest-controlled embedded tree, not raw user input.
		if err := os.WriteFile(backupPath, existing, backupMode); err != nil {
			return "", fmt.Errorf("failed to create backup file %s: %w", target, err)
		}
	}
	if err := writeAtomic(target, content, perm); err != nil {
		return "", fmt.Errorf("failed to write file %s: %w", target, err)
	}
	return backupPath, nil
}

// writeAtomic writes content to destPath via tmp+rename in the same directory,
// so a mid-write kill never leaves a truncated live file. os.Rename is atomic
// on POSIX same-dir and on Windows for this size class.
func writeAtomic(destPath string, content []byte, perm os.FileMode) error {
	dir := filepath.Dir(destPath)
	tmp, err := os.CreateTemp(dir, ".envctl-tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(content); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Chmod(perm); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	if err := os.Rename(tmpName, destPath); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("failed to rename temp file to %s: %w", destPath, err)
	}
	return nil
}

// WriteWithBackup writes content to target file. If destination exists and differs,
// an atomic timestamped backup (.bak.YYYYMMDD-HHMMSS) is created first.
func (f *FileSystemManager) WriteWithBackup(destPath string, content []byte, perm os.FileMode) (string, error) {
	expanded, err := f.ExpandUserPath(destPath)
	if err != nil {
		return "", fmt.Errorf("failed to expand path %s: %w", destPath, err)
	}

	dir := filepath.Dir(expanded)
	//nolint:gosec // G301: config dirs (~/.config, ~/.local) are shared content, not secrets.
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	var existing []byte
	if f.Exists(expanded) {
		if data, err := os.ReadFile(expanded); err == nil {
			existing = data
		}
	}

	// Write the new content atomically, backing up differing live bytes first.
	return storeWithBackup(expanded, existing, content, perm, perm)
}

// SetStrictWindowsACL restricts file/directory permissions to the current user only.
// On Windows, it uses icacls to remove inheritance and grant full control to the user.
// On Linux it sets POSIX mode 0700 for directories or 0600 for files.
func (f *FileSystemManager) SetStrictWindowsACL(path string) error {
	expanded, err := f.ExpandUserPath(path)
	if err != nil {
		return err
	}

	if runtime.GOOS != "windows" {
		fi, err := os.Stat(expanded)
		if err != nil {
			return err
		}
		if fi.IsDir() {
			//nolint:gosec // G302: directories need the exec bit (0700) to be traversable.
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
	//nolint:gosec // G702: fixed icacls invocation; currentUser comes from the environment (USERNAME/USER), not raw input.
	cmd := exec.Command("icacls.exe", expanded, "/inheritance:r", "/grant:r", fmt.Sprintf("%s:(OI)(CI)F", currentUser))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("icacls failed for %s: %s (%w)", expanded, string(output), err)
	}

	return nil
}

// CopyEmbeddedTree copies all files from an embedded fs.FS folder into a target directory.
func (f *FileSystemManager) CopyEmbeddedTree(embeddedFS fs.FS, sourceDir, targetDir string) (int, error) {
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
			//nolint:gosec // G301: deployed skill/config dirs are shared content, not secrets.
			return os.MkdirAll(targetPath, 0755)
		}

		// Read embedded file
		data, err := fs.ReadFile(embeddedFS, path)
		if err != nil {
			return fmt.Errorf("failed to read embedded file %s: %w", path, err)
		}

		// Ensure parent directory exists
		//nolint:gosec // G301: deployed skill/config dirs are shared content, not secrets.
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return err
		}

		mode := execModeForEmbedded(path, data, targetPath, d)

		// Diff-gate: identical content is a no-op (not counted), so
		// user-customized files that match the template cost nothing and
		// differing ones always leave a timestamped backup first.
		var existing []byte
		if data, err := os.ReadFile(targetPath); err == nil {
			existing = data
		}
		if existing != nil && bytes.Equal(existing, data) {
			// Refresh the mode when the content matches but the exec
			// bit drifted (e.g. template gained a shebang).
			if fi, statErr := os.Stat(targetPath); statErr == nil {
				if fi.Mode().Perm() != mode.Perm() {
					if chmodErr := os.Chmod(targetPath, mode); chmodErr != nil {
						return fmt.Errorf("failed to set mode on %s: %w", targetPath, chmodErr)
					}
				}
			}
			return nil
		}
		// Backup preserves the live file's own mode (fallback 0600).
		backupMode := os.FileMode(0600)
		if fi, statErr := os.Stat(targetPath); statErr == nil {
			backupMode = fi.Mode().Perm()
		}
		if _, err := storeWithBackup(targetPath, existing, data, mode, backupMode); err != nil {
			return err
		}

		count++
		return nil
	})

	return count, err
}

// execModeForEmbedded decides the file mode for a deployed embedded file.
// The existing target's exec bit is the source of truth when present (never
// drop +x a user made executable); otherwise scripts ship executable:
// anything under configs/bin/, *.sh, or an extensionless file with a shebang
// gets 0755, everything else 0644.
func execModeForEmbedded(sourcePath string, data []byte, targetPath string, d fs.DirEntry) os.FileMode {
	if fi, err := os.Stat(targetPath); err == nil && !fi.IsDir() {
		if fi.Mode()&0111 != 0 {
			return os.FileMode(0755)
		}
	}
	if info, err := d.Info(); err == nil {
		if info.Mode()&0111 != 0 {
			return os.FileMode(0755)
		}
	}
	normalized := filepath.ToSlash(sourcePath)
	if strings.HasPrefix(normalized, "configs/bin/") || strings.HasSuffix(normalized, ".sh") {
		return os.FileMode(0755)
	}
	if filepath.Ext(filepath.Base(normalized)) == "" && bytes.HasPrefix(data, []byte("#!")) {
		return os.FileMode(0755)
	}
	return os.FileMode(0644)
}
