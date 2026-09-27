//go:build !linux

package performance

import "fmt"

// statfsRoot is unavailable outside Linux. The performance profile is Linux-only
// and gated on Ubuntu Server, so this path exists to keep the package compiling
// for the Windows release rather than to serve a real caller.
func statfsRoot(path string) (fsStat, error) {
	return fsStat{}, fmt.Errorf("filesystem statistics are not available on this platform")
}
