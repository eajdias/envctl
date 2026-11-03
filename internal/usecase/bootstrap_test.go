package usecase

import (
	"os"
	"path/filepath"
	"testing"
)

// repoRootDir resolves the repository root by searching upward for go.mod.
// Test working dir is the package dir, so CWD assumptions would break.
func repoRootDir(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd failed: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("go.mod not found searching upward from package dir")
		}
		dir = parent
	}
}

// The Windows bootstrap is piped from raw.githubusercontent.com into
// Invoke-Expression, which parses the downloaded text as a script block. A
// UTF-8 BOM at the start leaves the <# #> help comment unparsed, so the
// comment CONTRACT lines (containing PowerShell keywords like `from`) are
// parsed as code and `irm ... | iex` fails (observed 2026-10-29: "The 'from'
// keyword is not supported in this version of the language"). The file must
// stay BOM-free.
func TestBootstrapPs1HasNoBOM(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(repoRootDir(t), "bootstrap.ps1"))
	if err != nil {
		t.Fatalf("reading bootstrap.ps1: %v", err)
	}
	if len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
		t.Fatal("bootstrap.ps1 starts with a UTF-8 BOM; irm | iex parses it as code and fails")
	}
}
