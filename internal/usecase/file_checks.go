package usecase

import "os"

// isExecutableFile reports whether path exists, is a regular file and carries
// an executable bit. Deployed helper scripts (the local verifier and the git
// hooks) are inert without it, so the audit treats a missing bit as drift.
func isExecutableFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}
	return info.Mode()&0o111 != 0
}
