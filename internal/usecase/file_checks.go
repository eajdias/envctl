package usecase

import (
	"os"
	"runtime"
)

// isExecutableFile reports whether path exists, is a regular file and carries
// an executable bit. Deployed helper scripts (the local verifier and the git
// hooks) are inert without it, so the audit treats a missing bit as drift.
// On Windows the bit is synthesized from file attributes and is unreliable
// for provisioned scripts, so existence of a regular file is enough there;
// POSIX hosts keep the strict bit check.
func isExecutableFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}
	if runtime.GOOS == "windows" {
		return true
	}
	return info.Mode()&0o111 != 0
}
