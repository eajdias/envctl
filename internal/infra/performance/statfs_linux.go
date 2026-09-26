//go:build linux

package performance

import "syscall"

// statfsRoot reports the filesystem type and available bytes of a path.
//
// It lives behind a build tag because statfs(2) has no portable equivalent in
// the standard library: syscall.Statfs and syscall.Statfs_t do not exist on
// Windows, and this package is compiled for every target because ui/cli wires
// the performance adapters unconditionally. Without the split, the Windows
// release build fails.
func statfsRoot(path string) (fsStat, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return fsStat{}, err
	}
	// Bsize is signed and Bavail is unsigned, with the exact widths varying by
	// architecture. Both are narrowed through positiveInt64 so a negative or
	// absurd kernel value cannot wrap into a huge unsigned one and defeat the
	// swapfile size clamp.
	return fsStat{
		Type:      filesystemTypeName(stat.Type),
		FreeBytes: stat.Bavail * positiveInt64(stat.Bsize),
	}, nil
}

// positiveInt64 narrows a signed kernel-reported value to uint64 after checking
// the sign, so a negative value becomes zero rather than wrapping.
func positiveInt64(value int64) uint64 {
	if value <= 0 {
		return 0
	}
	return uint64(value)
}
