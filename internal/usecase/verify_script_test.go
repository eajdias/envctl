package usecase

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// The verifier is a shell script but it sits on the critical path of every push,
// so it is tested like code: each case builds a throwaway repository, runs the
// real script and asserts the exit code and the report. Optional tools
// (shellcheck, hadolint, tsc) are skipped when absent, which keeps the suite
// deterministic on a bare runner.

func verifyScriptPath(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("envctl-verify is a POSIX shell script")
	}
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not available")
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	script, err := filepath.Abs(filepath.Join("..", "..", "configs", "bin", "envctl-verify"))
	if err != nil {
		t.Fatalf("resolve verifier path: %v", err)
	}
	if _, err := os.Stat(script); err != nil {
		t.Fatalf("verifier not found at %s: %v", script, err)
	}
	return script
}

// runVerify executes the verifier with dir as the working directory.
func runVerify(t *testing.T, dir string, env []string, args ...string) (int, string) {
	t.Helper()
	cmd := exec.Command("bash", append([]string{verifyScriptPath(t)}, args...)...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), env...)
	out, err := cmd.CombinedOutput()
	code := 0
	if exitErr, ok := err.(*exec.ExitError); ok {
		code = exitErr.ExitCode()
	} else if err != nil {
		t.Fatalf("running verifier: %v", err)
	}
	return code, string(out)
}

func initRepo(t *testing.T, dir string) {
	t.Helper()
	cmd := exec.Command("git", "init", "-q")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v (%s)", err, out)
	}
}

func writeTestFile(t *testing.T, dir, relPath, content string) {
	t.Helper()
	full := filepath.Join(dir, relPath)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
}

func TestVerifyScriptIgnoresRepositoriesWithoutStack(t *testing.T) {
	dir := t.TempDir()
	initRepo(t, dir)
	writeTestFile(t, dir, "notes.txt", "nothing to check here\n")

	code, out := runVerify(t, dir, nil, "--git-push")
	if code != 0 {
		t.Errorf("empty repository must exit 0, got %d\n%s", code, out)
	}
	if strings.TrimSpace(out) != "" {
		t.Errorf("nothing to do must stay silent, got:\n%s", out)
	}
}

func TestVerifyScriptIgnoresNonRepository(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "notes.txt", "not a repository\n")

	if code, out := runVerify(t, dir, nil, "--git-push"); code != 0 {
		t.Errorf("outside a repository the verifier must exit 0, got %d\n%s", code, out)
	}
}

func TestVerifyScriptHonoursRepoLocalOverride(t *testing.T) {
	dir := t.TempDir()
	initRepo(t, dir)
	writeTestFile(t, dir, ".commandcode/verify.sh", "#!/bin/sh\necho 'override rodou'\nexit 1\n")

	code, out := runVerify(t, dir, nil, "--git-push")
	if code != 2 {
		t.Errorf("a failing override must abort the push (exit 2), got %d\n%s", code, out)
	}
	if !strings.Contains(out, "override rodou") {
		t.Errorf("the override output must reach the report, got:\n%s", out)
	}
}

func TestVerifyScriptSkipEscapeHatch(t *testing.T) {
	dir := t.TempDir()
	initRepo(t, dir)
	writeTestFile(t, dir, ".commandcode/verify.sh", "#!/bin/sh\nexit 1\n")

	code, _ := runVerify(t, dir, []string{"ENVCTL_SKIP_VERIFY=1"}, "--git-push")
	if code != 0 {
		t.Errorf("ENVCTL_SKIP_VERIFY=1 must bypass the gate, got exit %d", code)
	}
}

func TestVerifyScriptHookModeIgnoresRetry(t *testing.T) {
	dir := t.TempDir()
	initRepo(t, dir)
	writeTestFile(t, dir, ".commandcode/verify.sh", "#!/bin/sh\nexit 1\n")

	payload := `{"hook_event_name":"Stop","stop_hook_active":true}`
	cmd := exec.Command("bash", verifyScriptPath(t), "--hook")
	cmd.Dir = dir
	cmd.Stdin = strings.NewReader(payload)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Errorf("a retry fire must let the turn end: %v\n%s", err, out)
	}
}

func TestVerifyScriptDryRunListsChecks(t *testing.T) {
	dir := t.TempDir()
	initRepo(t, dir)
	writeTestFile(t, dir, "go.mod", "module example.com/dry\n\ngo 1.21\n")
	writeTestFile(t, dir, "main.go", "package main\n\nfunc main() {}\n")

	code, out := runVerify(t, dir, nil, "--dry-run")
	if code != 0 {
		t.Errorf("dry run must not fail, got %d\n%s", code, out)
	}
	for _, want := range []string{"stacks detected: go", "go build ./...", "go test ./..."} {
		if !strings.Contains(out, want) {
			t.Errorf("dry run output must mention %q, got:\n%s", want, out)
		}
	}
}

// goFixture writes a module whose build is clean and whose test fails, so the
// static half passes while the test half does not.
func goFixture(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go toolchain not available")
	}
	dir := t.TempDir()
	initRepo(t, dir)
	writeTestFile(t, dir, "go.mod", "module example.com/fixture\n\ngo 1.21\n")
	writeTestFile(t, dir, "main.go", "package main\n\nfunc Soma(a, b int) int { return a + b }\n\nfunc main() {}\n")
	writeTestFile(t, dir, "main_test.go", "package main\n\nimport \"testing\"\n\nfunc TestSoma(t *testing.T) {\n\tif Soma(2, 2) != 5 {\n\t\tt.Fatalf(\"intentionally broken\")\n\t}\n}\n")
	return dir
}

func TestVerifyScriptPushModeRunsTests(t *testing.T) {
	dir := goFixture(t)

	code, out := runVerify(t, dir, nil, "--git-push")
	if code != 2 {
		t.Errorf("a failing test must abort the push, got exit %d\n%s", code, out)
	}
	if !strings.Contains(out, "go test") {
		t.Errorf("the report must name the failing check, got:\n%s", out)
	}
}

func TestVerifyScriptHookModeSkipsTestsAndCachesGreenState(t *testing.T) {
	dir := goFixture(t)
	payload := `{"hook_event_name":"Stop","stop_hook_active":false}`

	runHook := func() (int, string) {
		cmd := exec.Command("bash", verifyScriptPath(t), "--hook")
		cmd.Dir = dir
		cmd.Stdin = strings.NewReader(payload)
		out, err := cmd.CombinedOutput()
		code := 0
		if exitErr, ok := err.(*exec.ExitError); ok {
			code = exitErr.ExitCode()
		} else if err != nil {
			t.Fatalf("hook mode: %v", err)
		}
		return code, string(out)
	}

	code, out := runHook()
	if code != 0 {
		t.Errorf("hook mode runs static checks only, so the failing test must not block a turn: exit %d\n%s", code, out)
	}
	if strings.Contains(out, "--- go test") {
		t.Errorf("hook mode must not run the test suite, got:\n%s", out)
	}

	stamp := filepath.Join(dir, ".git", "envctl-verify.stamp")
	if _, err := os.Stat(stamp); err != nil {
		t.Errorf("a green hook run must record the tree state for the cache: %v", err)
	}

	code, out = runHook()
	if code != 0 || strings.TrimSpace(out) != "" {
		t.Errorf("an unchanged tree must stay silent and green, got exit %d:\n%s", code, out)
	}
}

// The cache must not swallow an edit to a file that is still untracked: while a
// new file is being written it appears in `git status` by path only, so a hash
// that ignored its content would keep reporting the previous green verdict.
func TestVerifyScriptHookModeReRunsAfterUntrackedEdit(t *testing.T) {
	dir := goFixture(t)
	payload := `{"hook_event_name":"Stop","stop_hook_active":false}`

	runHook := func() (int, string) {
		cmd := exec.Command("bash", verifyScriptPath(t), "--hook")
		cmd.Dir = dir
		cmd.Stdin = strings.NewReader(payload)
		out, err := cmd.CombinedOutput()
		code := 0
		if exitErr, ok := err.(*exec.ExitError); ok {
			code = exitErr.ExitCode()
		} else if err != nil {
			t.Fatalf("hook mode: %v", err)
		}
		return code, string(out)
	}

	if code, out := runHook(); code != 0 {
		t.Fatalf("fixture must start green: exit %d\n%s", code, out)
	}

	// Break a static check (the build) in a file that is still untracked.
	writeTestFile(t, dir, "broken.go", "package main\n\nfunc Quebrado( {\n")

	code, out := runHook()
	if code == 0 {
		t.Errorf("an edit to an untracked file must re-run the checks, got exit 0 (cache hit)")
	}
	if !strings.Contains(out, "go build") {
		t.Errorf("the report must name the broken check, got:\n%s", out)
	}
}
