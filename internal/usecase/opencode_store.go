package usecase

import (
	"context"
	"encoding/binary"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// OpenCode session store (SQLite) hygiene.
//
// The store grows with session history and its size is audited by the doctor.
// Only free pages can be reclaimed safely: a store that reached its size with
// live rows must never be rewritten behind the user's back, so this file
// measures what is reclaimable and vacuums solely when there is something to
// reclaim. A store in use by a running OpenCode cannot be locked and is
// reported instead of retried.

// openCodeStoreWarnBytes is the size above which the store is worth reporting
// (shared by the doctor audit and `run cleanup`).
const openCodeStoreWarnBytes = 500 * 1024 * 1024

// OpenCodeStore describes the on-disk session store.
type OpenCodeStore struct {
	Path             string
	SizeBytes        int64
	PageSize         int64
	PageCount        int64
	FreelistPages    int64
	ReclaimableBytes int64
}

// ExceedsThreshold reports whether the store is larger than the audit
// threshold and therefore worth acting on.
func (s OpenCodeStore) ExceedsThreshold() bool {
	return s.SizeBytes > openCodeStoreWarnBytes
}

// openCodeDataDir returns the OpenCode data directory, honoring XDG_DATA_HOME.
func openCodeDataDir(homeDir string) string {
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return filepath.Join(xdg, "opencode")
	}
	return filepath.Join(homeDir, ".local", "share", "opencode")
}

// openCodeStorePath resolves the SQLite session store inside the data dir.
func openCodeStorePath(homeDir string) string {
	return filepath.Join(openCodeDataDir(homeDir), "opencode.db")
}

// InspectOpenCodeStore reads the SQLite header to report the store size and how
// many bytes a VACUUM would reclaim. Parsing the 100-byte header directly keeps
// the project dependency-free: no SQLite driver is pulled in just to read
// three integers.
func InspectOpenCodeStore(path string) (OpenCodeStore, error) {
	store := OpenCodeStore{Path: path}

	info, err := os.Stat(path)
	if err != nil {
		return store, err
	}
	store.SizeBytes = info.Size()

	header := make([]byte, 100)
	f, err := os.Open(path)
	if err != nil {
		return store, err
	}
	defer f.Close()
	if _, err := f.Read(header); err != nil {
		return store, fmt.Errorf("failed to read sqlite header of %s: %w", path, err)
	}
	if string(header[0:16]) != "SQLite format 3\x00" {
		return store, fmt.Errorf("%s is not a SQLite database", path)
	}

	// Header fields are big-endian; a page size of 1 encodes 65536.
	pageSize := int64(binary.BigEndian.Uint16(header[16:18]))
	if pageSize == 1 {
		pageSize = 65536
	}
	store.PageSize = pageSize
	store.PageCount = int64(binary.BigEndian.Uint32(header[28:32]))
	store.FreelistPages = int64(binary.BigEndian.Uint32(header[36:40]))
	store.ReclaimableBytes = store.FreelistPages * pageSize

	return store, nil
}

// vacuumOpenCodeStore reclaims free pages and returns the bytes freed. It
// delegates to the sqlite3 CLI or a python3 interpreter, whichever exists,
// because the project intentionally carries no SQLite driver.
func vacuumOpenCodeStore(ctx context.Context, path string) (int64, error) {
	before, err := InspectOpenCodeStore(path)
	if err != nil {
		return 0, err
	}

	var cmd *exec.Cmd
	if sqlite3, lookErr := exec.LookPath("sqlite3"); lookErr == nil {
		cmd = exec.CommandContext(ctx, sqlite3, path, "VACUUM;")
	} else if python, lookErr := exec.LookPath(sqliteInterpreter()); lookErr == nil {
		script := "import sqlite3, sys; sqlite3.connect(sys.argv[1]).execute('VACUUM')"
		cmd = exec.CommandContext(ctx, python, "-c", script, path)
	} else {
		return 0, fmt.Errorf("neither sqlite3 nor %s is available to run VACUUM", sqliteInterpreter())
	}

	if out, err := cmd.CombinedOutput(); err != nil {
		return 0, fmt.Errorf("VACUUM failed: %s (%w)", strings.TrimSpace(string(out)), err)
	}

	after, err := InspectOpenCodeStore(path)
	if err != nil {
		return 0, err
	}
	freed := before.SizeBytes - after.SizeBytes
	if freed < 0 {
		freed = 0
	}
	return freed, nil
}

// sqliteInterpreter returns the python binary name used on this platform.
func sqliteInterpreter() string {
	if runtime.GOOS == "windows" {
		return "python"
	}
	return "python3"
}
