package usecase

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/eajdias/envctl/internal/infra/executil"
)

// TestLookPathInEnvFindsAToolchainOnlyBinary isolates the resolution rule.
func TestLookPathInEnvFindsAToolchainOnlyBinary(t *testing.T) {
	dir := t.TempDir()
	tool := filepath.Join(dir, "mise")
	if err := os.WriteFile(tool, []byte("#!/bin/sh\necho fake-mise\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := executil.LookPathIn(dir, "mise")
	if err != nil {
		t.Fatalf("a binary present in the supplied PATH was not found: %v", err)
	}
	if got != tool {
		t.Fatalf("resolved %q, want %q", got, tool)
	}
	if _, err := executil.LookPathIn(t.TempDir(), "mise"); err == nil {
		t.Fatal("expected a miss when the PATH does not contain the binary")
	}
	if _, err := executil.LookPathIn("", "mise"); err == nil {
		t.Fatal("expected a miss when the environment declares no PATH")
	}
}

// TestRunWithToolchainResolvesAgainstTheSuppliedPath is the regression test for
// the real defect: exec.Command resolves the binary against the PROCESS PATH, so
// setting cmd.Env afterwards is not enough. A non-login shell (ssh, systemd,
// agent) has a minimal process PATH, so `mise` — which the official installer
// puts in ~/.local/bin — was reported as "executable file not found in $PATH"
// even though the profile had just installed it.
func TestRunWithToolchainResolvesAgainstTheSuppliedPath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the toolchain PATH is a POSIX-shell concern")
	}
	dir := t.TempDir()
	tool := filepath.Join(dir, "fakemise")
	script := "#!/bin/sh\necho \"resolved:$0 args:$*\"\n"
	if err := os.WriteFile(tool, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	// The process PATH deliberately does not contain the tool; only the
	// supplied environment does.
	out, err := runWithToolchainEnv(context.Background(), []string{"PATH=" + dir, "HOME=" + t.TempDir()}, "fakemise", "install", "node@24.19.0")
	if err != nil {
		t.Fatalf("runWithToolchainEnv failed although the binary is in the supplied PATH: %v (%s)", err, out)
	}
	if !strings.Contains(out, "install node@24.19.0") {
		t.Fatalf("arguments were not passed through: %q", out)
	}
	if !strings.Contains(out, tool) {
		t.Fatalf("the tool did not run from the resolved absolute path: %q", out)
	}
}

func TestRunWithToolchainEnvReportsAMissingBinary(t *testing.T) {
	_, err := runWithToolchainEnv(context.Background(), []string{"PATH=" + t.TempDir()}, "definitely-not-a-real-tool")
	if err == nil {
		t.Fatal("expected a missing binary to be reported")
	}
}
