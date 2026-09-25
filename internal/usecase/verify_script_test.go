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

func mergeEnv(overrides []string) []string {
	keys := make(map[string]struct{}, len(overrides))
	for _, override := range overrides {
		key, _, ok := strings.Cut(override, "=")
		if ok {
			keys[key] = struct{}{}
		}
	}
	merged := make([]string, 0, len(os.Environ())+len(overrides))
	for _, entry := range os.Environ() {
		key, _, ok := strings.Cut(entry, "=")
		if ok {
			if _, overridden := keys[key]; overridden {
				continue
			}
		}
		merged = append(merged, entry)
	}
	return append(merged, overrides...)
}

func stripGitRepoEnv(env []string) []string {
	blocked := map[string]struct{}{
		"GIT_DIR": {}, "GIT_WORK_TREE": {}, "GIT_COMMON_DIR": {},
		"GIT_INDEX_FILE": {}, "GIT_OBJECT_DIRECTORY": {},
		"GIT_ALTERNATE_OBJECT_DIRECTORIES": {}, "GIT_PREFIX": {},
		"GIT_CONFIG": {}, "GIT_CONFIG_PARAMETERS": {},
	}
	filtered := make([]string, 0, len(env))
	for _, entry := range env {
		key, _, ok := strings.Cut(entry, "=")
		if ok {
			if _, remove := blocked[key]; remove {
				continue
			}
		}
		filtered = append(filtered, entry)
	}
	return filtered
}

func testEnv(overrides []string) []string {
	return stripGitRepoEnv(mergeEnv(overrides))
}

// runVerify executes the verifier with dir as the working directory.
func runVerify(t *testing.T, dir string, env []string, args ...string) (int, string) {
	t.Helper()
	cmd := exec.Command("bash", append([]string{verifyScriptPath(t)}, args...)...)
	cmd.Dir = dir
	cmd.Env = testEnv(env)
	out, err := cmd.CombinedOutput()
	code := 0
	if exitErr, ok := err.(*exec.ExitError); ok {
		code = exitErr.ExitCode()
	} else if err != nil {
		t.Fatalf("running verifier: %v", err)
	}
	return code, string(out)
}

func runVerifyHook(t *testing.T, dir string, env []string) (int, string) {
	t.Helper()
	cmd := exec.Command("bash", verifyScriptPath(t), "--hook")
	cmd.Dir = dir
	cmd.Env = testEnv(env)
	cmd.Stdin = strings.NewReader(`{"hook_event_name":"Stop","stop_hook_active":false}`)
	out, err := cmd.CombinedOutput()
	code := 0
	if exitErr, ok := err.(*exec.ExitError); ok {
		code = exitErr.ExitCode()
	} else if err != nil {
		t.Fatalf("running verifier hook: %v", err)
	}
	return code, string(out)
}

func prePushHookPath(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("pre-push fixture requires a POSIX shell")
	}
	path, err := filepath.Abs(filepath.Join("..", "..", "configs", "git", "hooks", "pre-push"))
	if err != nil {
		t.Fatalf("resolve pre-push hook path: %v", err)
	}
	return path
}

func runPrePushHook(t *testing.T, dir string, env []string) (int, string) {
	t.Helper()
	cmd := exec.Command("sh", prePushHookPath(t), "origin", "https://example.invalid/repo.git")
	cmd.Dir = dir
	cmd.Env = testEnv(env)
	out, err := cmd.CombinedOutput()
	code := 0
	if exitErr, ok := err.(*exec.ExitError); ok {
		code = exitErr.ExitCode()
	} else if err != nil {
		t.Fatalf("running pre-push hook: %v", err)
	}
	return code, string(out)
}

func initRepo(t *testing.T, dir string) {
	t.Helper()
	cmd := exec.Command("git", "init", "-q")
	cmd.Dir = dir
	cmd.Env = testEnv(nil)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v (%s)", err, out)
	}
}

func gitUpdateRemote(t *testing.T, dir string) {
	t.Helper()
	commit := exec.Command("git", "-c", "user.name=Verifier Test", "-c", "user.email=verifier@example.invalid", "commit", "--allow-empty", "-qm", "fixture")
	commit.Dir = dir
	commit.Env = testEnv(nil)
	if out, err := commit.CombinedOutput(); err != nil {
		t.Fatalf("git commit: %v (%s)", err, out)
	}
	ref := exec.Command("git", "update-ref", "refs/remotes/origin/main", "HEAD")
	ref.Dir = dir
	ref.Env = testEnv(nil)
	if out, err := ref.CombinedOutput(); err != nil {
		t.Fatalf("git update-ref: %v (%s)", err, out)
	}
}

func moveRemoteBase(t *testing.T, dir string) {
	t.Helper()
	oldRefCmd := exec.Command("git", "rev-parse", "refs/remotes/origin/main")
	oldRefCmd.Dir = dir
	oldRefCmd.Env = testEnv(nil)
	oldRef, err := oldRefCmd.Output()
	if err != nil {
		t.Fatalf("read comparison base: %v", err)
	}
	treeCmd := exec.Command("git", "rev-parse", "HEAD^{tree}")
	treeCmd.Dir = dir
	treeCmd.Env = testEnv(nil)
	tree, err := treeCmd.Output()
	if err != nil {
		t.Fatalf("read fixture tree: %v", err)
	}
	commit := exec.Command("git", "commit-tree", strings.TrimSpace(string(tree)), "-m", "base moved")
	commit.Dir = dir
	commit.Env = testEnv([]string{
		"GIT_AUTHOR_NAME=Verifier Test",
		"GIT_AUTHOR_EMAIL=verifier@example.invalid",
		"GIT_COMMITTER_NAME=Verifier Test",
		"GIT_COMMITTER_EMAIL=verifier@example.invalid",
	})
	newRef, err := commit.Output()
	if err != nil {
		t.Fatalf("create moved base: %v", err)
	}
	update := exec.Command("git", "update-ref", "refs/remotes/origin/main", strings.TrimSpace(string(newRef)), strings.TrimSpace(string(oldRef)))
	update.Dir = dir
	update.Env = testEnv(nil)
	if out, err := update.CombinedOutput(); err != nil {
		t.Fatalf("move comparison base: %v (%s)", err, out)
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

func nodeLintFixture(t *testing.T, includeSource bool) string {
	t.Helper()
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node not available")
	}
	dir := t.TempDir()
	initRepo(t, dir)
	writeTestFile(t, dir, "package.json", `{"name":"fixture","private":true}`)
	writeTestFile(t, dir, "eslint.config.mjs", "export default []\n")
	if includeSource {
		writeTestFile(t, dir, "index.js", "const value = 1;\n")
	}
	writeTestFile(t, dir, "node_modules/.bin/eslint", "#!/usr/bin/env node\nconsole.error('advisory eslint finding')\nprocess.exit(1)\n")
	return dir
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

func TestVerifyScriptWithoutHomeUsesFallbackLogPath(t *testing.T) {
	dir := t.TempDir()
	initRepo(t, dir)
	writeTestFile(t, dir, "notes.txt", "nothing to check here\n")

	cmd := exec.Command("env", "-u", "HOME", "bash", verifyScriptPath(t), "--git-push")
	cmd.Dir = dir
	cmd.Env = testEnv([]string{"HOME="})
	out, err := cmd.CombinedOutput()
	code := 0
	if exitErr, ok := err.(*exec.ExitError); ok {
		code = exitErr.ExitCode()
	} else if err != nil {
		t.Fatalf("running verifier without HOME: %v", err)
	}
	if code != 0 {
		t.Fatalf("a missing HOME must not abort the verifier, got %d\n%s", code, out)
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

func TestVerifyScriptPushModeReportsInferredESLintAsAdvisory(t *testing.T) {
	dir := nodeLintFixture(t, true)
	logPath := filepath.Join(t.TempDir(), "verify.log")

	code, out := runVerify(t, dir, []string{"ENVCTL_VERIFY_LOG=" + logPath}, "--git-push")
	if code != 0 {
		t.Fatalf("inferred ESLint findings must be advisory, got exit %d\n%s", code, out)
	}
	for _, want := range []string{"advisory", "eslint (changed files)", "advisory eslint finding"} {
		if !strings.Contains(out, want) {
			t.Errorf("advisory report must mention %q, got:\n%s", want, out)
		}
	}
	logBytes, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read verifier trail: %v", err)
	}
	if !strings.Contains(string(logBytes), "ADVISORY") {
		t.Errorf("advisory result must be recorded in the trail, got:\n%s", logBytes)
	}
}

func TestVerifyScriptDoesNotLintESLintConfigByDefault(t *testing.T) {
	dir := nodeLintFixture(t, false)

	code, out := runVerify(t, dir, nil, "--git-push")
	if code != 0 {
		t.Fatalf("a config-only change must not fail the inferred check, got exit %d\n%s", code, out)
	}
	if strings.Contains(out, "eslint (changed files)") {
		t.Errorf("eslint.config.mjs must not be passed to the inferred check, got:\n%s", out)
	}
}

func TestVerifyScriptRunsExplicitPackageLintAsBlocking(t *testing.T) {
	dir := nodeLintFixture(t, true)
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node not available")
	}
	writeTestFile(t, dir, "package.json", `{"name":"fixture","packageManager":"pnpm@11.0.0","scripts":{"lint":"true"}}`)
	fakeBin := filepath.Join(dir, "fake-bin")
	writeTestFile(t, filepath.Join(dir, "fake-bin"), "pnpm", "#!/usr/bin/env node\nconsole.error('project lint via pnpm')\nprocess.exit(1)\n")
	pathEnv := "PATH=" + fakeBin + string(os.PathListSeparator) + os.Getenv("PATH")

	code, out := runVerify(t, dir, []string{pathEnv}, "--git-push")
	if code != 2 {
		t.Fatalf("an explicit project lint script must block the push, got exit %d\n%s", code, out)
	}
	for _, want := range []string{"package.json lint (pnpm)", "project lint via pnpm"} {
		if !strings.Contains(out, want) {
			t.Errorf("explicit lint report must mention %q, got:\n%s", want, out)
		}
	}
	if strings.Contains(out, "eslint (changed files)") {
		t.Errorf("the explicit project script must replace inferred ESLint, got:\n%s", out)
	}
}

func TestVerifyScriptDryRunLabelsAdvisoryChecks(t *testing.T) {
	dir := nodeLintFixture(t, true)

	code, out := runVerify(t, dir, nil, "--dry-run")
	if code != 0 {
		t.Fatalf("dry run must not fail, got %d\n%s", code, out)
	}
	if !strings.Contains(out, "[advisory] eslint (changed files)") {
		t.Errorf("dry run must identify inferred ESLint as advisory, got:\n%s", out)
	}
}

func nodeOnlyPath(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("POSIX PATH fixture is not applicable on Windows")
	}
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node not available")
	}
	bin := t.TempDir()
	if err := os.Symlink(node, filepath.Join(bin, "node")); err != nil {
		t.Fatalf("create node PATH shim: %v", err)
	}
	return bin + ":/usr/bin:/bin"
}

func TestVerifyScriptDoesNotCacheUnavailablePackageLint(t *testing.T) {
	dir := t.TempDir()
	initRepo(t, dir)
	path := nodeOnlyPath(t)
	writeTestFile(t, dir, "package.json", `{"name":"fixture","packageManager":"pnpm@11.0.0","scripts":{"lint":"true"}}`)
	logPath := filepath.Join(t.TempDir(), "verify.log")

	bin := strings.Split(path, string(os.PathListSeparator))[0]
	writeTestFile(t, bin, "pnpm", "#!/bin/sh\nexit 0\n")
	code, out := runVerifyHook(t, dir, []string{"PATH=" + path, "ENVCTL_VERIFY_LOG=" + logPath})
	if code != 0 {
		t.Fatalf("the initial explicit lint must be green, got exit %d\n%s", code, out)
	}
	if _, err := os.Stat(filepath.Join(dir, ".git", "envctl-verify.stamp")); err != nil {
		t.Fatalf("the green hook must create the cache stamp: %v", err)
	}

	if err := os.Remove(filepath.Join(bin, "pnpm")); err != nil {
		t.Fatal(err)
	}
	code, out = runVerifyHook(t, dir, []string{"PATH=" + path, "ENVCTL_VERIFY_LOG=" + logPath})
	if code != 0 {
		t.Fatalf("removing the package manager must be a skip, got exit %d\n%s", code, out)
	}
	if _, err := os.Stat(filepath.Join(dir, ".git", "envctl-verify.stamp")); !os.IsNotExist(err) {
		t.Fatalf("a skipped hook must remove the stale cache stamp, stat error: %v", err)
	}

	writeTestFile(t, bin, "pnpm", "#!/bin/sh\nprintf 'project lint ran\\n' >&2\nexit 1\n")
	code, out = runVerifyHook(t, dir, []string{"PATH=" + path, "ENVCTL_VERIFY_LOG=" + logPath})
	if code != 2 {
		t.Fatalf("a newly available explicit lint must run and block, got exit %d\n%s", code, out)
	}
	if !strings.Contains(out, "project lint ran") {
		t.Errorf("the explicit lint must run after the tool becomes available, got:\n%s", out)
	}
}

func noNodePath(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("POSIX PATH fixture is not applicable on Windows")
	}
	return t.TempDir() + ":/usr/bin:/bin"
}

func TestVerifyScriptKeepsExplicitLintAuthorityWithoutNode(t *testing.T) {
	dir := t.TempDir()
	initRepo(t, dir)
	path := noNodePath(t)
	bin := strings.Split(path, string(os.PathListSeparator))[0]
	writeTestFile(t, bin, "node", "#!/bin/sh\nexit 127\n")
	writeTestFile(t, dir, "package.json", `{"name":"fixture","packageManager":"pnpm@11.0.0","scripts":{"lint":"true"}}`)
	writeTestFile(t, dir, "eslint.config.mjs", "export default []\n")
	writeTestFile(t, dir, "index.js", "const value = 1;\n")
	writeTestFile(t, dir, "node_modules/.bin/eslint", "#!/bin/sh\nprintf 'fallback eslint ran\\n' >&2\nexit 1\n")

	code, out := runVerify(t, dir, []string{"PATH=" + path}, "--git-push")
	if code != 0 {
		t.Fatalf("a missing Node must not turn an explicit lint into a failure, got exit %d\n%s", code, out)
	}
	if strings.Contains(out, "fallback eslint ran") {
		t.Errorf("an explicit lint script must suppress inferred ESLint without Node, got:\n%s", out)
	}
}

func TestVerifyScriptDoesNotFallBackFromPnpmLockToNpm(t *testing.T) {
	dir := t.TempDir()
	initRepo(t, dir)
	path := nodeOnlyPath(t)
	bin := strings.Split(path, string(os.PathListSeparator))[0]
	writeTestFile(t, dir, "package.json", `{"name":"fixture","scripts":{"lint":"true"}}`)
	writeTestFile(t, dir, "pnpm-lock.yaml", "lockfileVersion: '9.0'\n")
	writeTestFile(t, dir, "eslint.config.mjs", "export default []\n")
	writeTestFile(t, dir, "index.js", "const value = 1;\n")
	writeTestFile(t, dir, "node_modules/.bin/eslint", "#!/bin/sh\nprintf 'fallback eslint ran\\n' >&2\nexit 1\n")
	writeTestFile(t, bin, "npm", "#!/bin/sh\nprintf 'npm fallback ran\\n' >&2\nexit 1\n")

	code, out := runVerify(t, dir, []string{"PATH=" + path}, "--git-push")
	if code != 0 {
		t.Fatalf("an unavailable pnpm must be a skip, got exit %d\n%s", code, out)
	}
	if strings.Contains(out, "npm fallback ran") || strings.Contains(out, "fallback eslint ran") {
		t.Errorf("a pnpm lockfile must not fall back to npm or inferred ESLint, got:\n%s", out)
	}
}

func TestVerifyScriptCacheIncludesBaseRef(t *testing.T) {
	dir := t.TempDir()
	initRepo(t, dir)
	gitUpdateRemote(t, dir)
	path := os.Getenv("PATH")
	bin := t.TempDir()
	path = bin + string(os.PathListSeparator) + path
	marker := filepath.Join(t.TempDir(), "allow-lint")
	writeTestFile(t, dir, "package.json", `{"name":"fixture","packageManager":"pnpm@11.0.0","scripts":{"lint":"true"}}`)
	writeTestFile(t, bin, "pnpm", "#!/bin/sh\nif [ -f \"$MARKER\" ]; then exit 0; fi\nprintf 'base moved\\n' >&2\nexit 1\n")
	if err := os.WriteFile(marker, []byte("ok\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	env := []string{"PATH=" + path, "MARKER=" + marker}

	code, out := runVerifyHook(t, dir, env)
	if code != 0 {
		t.Fatalf("the initial base must be green, got exit %d\n%s", code, out)
	}
	if _, err := os.Stat(filepath.Join(dir, ".git", "envctl-verify.stamp")); err != nil {
		t.Fatalf("the initial base must create the cache stamp: %v", err)
	}
	if err := os.Remove(marker); err != nil {
		t.Fatal(err)
	}
	moveRemoteBase(t, dir)

	code, out = runVerifyHook(t, dir, env)
	if code != 2 {
		t.Fatalf("moving the comparison base must invalidate the cache, got exit %d\n%s", code, out)
	}
	if !strings.Contains(out, "base moved") {
		t.Errorf("the check must run against the moved base, got:\n%s", out)
	}
}

func TestVerifyScriptDoesNotCacheRemovedLocalTool(t *testing.T) {
	dir := t.TempDir()
	initRepo(t, dir)
	writeTestFile(t, dir, "package.json", `{"name":"fixture","private":true}`)
	writeTestFile(t, dir, ".gitignore", "node_modules/\n")
	writeTestFile(t, dir, "tsconfig.json", "{}\n")
	writeTestFile(t, dir, "index.ts", "const value = 1;\n")
	writeTestFile(t, dir, ".prettierrc", "{}\n")
	writeTestFile(t, dir, "node_modules/.bin/tsc", "#!/bin/sh\nexit 0\n")
	writeTestFile(t, dir, "node_modules/.bin/prettier", "#!/bin/sh\nif [ -x node_modules/.bin/tsc ]; then exit 0; fi\nprintf 'prettier reran\\n' >&2\nexit 1\n")

	code, out := runVerifyHook(t, dir, nil)
	if code != 0 {
		t.Fatalf("the initial local-tool fixture must be green, got exit %d\n%s", code, out)
	}
	if err := os.Remove(filepath.Join(dir, "node_modules", ".bin", "tsc")); err != nil {
		t.Fatal(err)
	}
	code, out = runVerifyHook(t, dir, nil)
	if code != 0 {
		t.Fatalf("removing an advisory local tool must remain non-blocking, got exit %d\n%s", code, out)
	}
	if !strings.Contains(out, "prettier reran") {
		t.Errorf("removing a local tool must invalidate the hook cache, got:\n%s", out)
	}
}

func TestVerifyScriptUsesNpmWhenPackageManagerMetadataIsAbsent(t *testing.T) {
	dir := t.TempDir()
	initRepo(t, dir)
	path := t.TempDir()
	writeTestFile(t, dir, "package.json", `{"name":"fixture","scripts":{"lint":"true"}}`)
	writeTestFile(t, path, "npm", "#!/bin/sh\nprintf 'project lint via npm\\n' >&2\nexit 1\n")
	pathEnv := "PATH=" + path + string(os.PathListSeparator) + os.Getenv("PATH")

	code, out := runVerify(t, dir, []string{pathEnv}, "--git-push")
	if code != 2 {
		t.Fatalf("npm must be the default explicit lint manager, got exit %d\n%s", code, out)
	}
	if !strings.Contains(out, "project lint via npm") {
		t.Errorf("the npm lint script must run, got:\n%s", out)
	}
}

func TestVerifyScriptDoesNotCacheIgnoredOverride(t *testing.T) {
	dir := t.TempDir()
	initRepo(t, dir)
	writeTestFile(t, dir, ".gitignore", ".commandcode/\n")
	writeTestFile(t, dir, "go.mod", "module example.com/fixture\n\ngo 1.21\n")
	writeTestFile(t, dir, "main.go", "package fixture\n")

	code, out := runVerifyHook(t, dir, nil)
	if code != 0 {
		t.Fatalf("the initial hook must be green, got exit %d\n%s", code, out)
	}
	if _, err := os.Stat(filepath.Join(dir, ".git", "envctl-verify.stamp")); err != nil {
		t.Fatalf("the initial hook must create the cache stamp: %v", err)
	}

	writeTestFile(t, dir, ".commandcode/verify.sh", "#!/bin/sh\nexit 1\n")
	code, out = runVerifyHook(t, dir, nil)
	if code != 2 {
		t.Fatalf("an ignored executable override must invalidate the cache, got exit %d\n%s", code, out)
	}
	if !strings.Contains(out, ".commandcode/verify.sh") {
		t.Errorf("the override failure must be reported, got:\n%s", out)
	}
}

func TestVerifyScriptDoesNotCacheOversizedUntrackedEdit(t *testing.T) {
	dir := t.TempDir()
	initRepo(t, dir)
	writeTestFile(t, dir, "go.mod", "module example.com/large\n\ngo 1.21\n")
	padding := strings.Repeat("// "+strings.Repeat("x", 120)+"\n", 9000)
	writeTestFile(t, dir, "large.go", "package large\n"+padding)

	code, out := runVerifyHook(t, dir, nil)
	if code != 0 {
		t.Fatalf("the initial oversized fixture must be non-blocking, got exit %d\n%s", code, out)
	}
	if _, err := os.Stat(filepath.Join(dir, ".git", "envctl-verify.stamp")); !os.IsNotExist(err) {
		t.Fatalf("oversized untracked content must disable the cache, stat error: %v", err)
	}

	writeTestFile(t, dir, "large.go", "package large\nfunc broken( {\n"+padding)
	code, out = runVerifyHook(t, dir, nil)
	if code != 2 {
		t.Fatalf("an edit to an oversized untracked file must invalidate the cache, got exit %d\n%s", code, out)
	}
	if !strings.Contains(out, "go build") {
		t.Errorf("the second run must report the build failure, got:\n%s", out)
	}
}

func TestVerifyScriptClassifiesUnavailablePackageManagerAsSkip(t *testing.T) {
	dir := t.TempDir()
	initRepo(t, dir)
	path := nodeOnlyPath(t)
	bin := strings.Split(path, string(os.PathListSeparator))[0]
	writeTestFile(t, dir, "package.json", `{"name":"fixture","packageManager":"pnpm@11.0.0","scripts":{"lint":"true"}}`)
	writeTestFile(t, dir, "eslint.config.mjs", "export default []\n")
	writeTestFile(t, dir, "index.js", "const value = 1;\n")
	writeTestFile(t, dir, "node_modules/.bin/eslint", "#!/usr/bin/env node\nconsole.error('fallback eslint ran')\nprocess.exit(1)\n")
	writeTestFile(t, bin, "pnpm", "#!/bin/sh\nprintf 'Volta error: missing project dependency\\n' >&2\nexit 1\n")

	code, out := runVerify(t, dir, []string{"PATH=" + path}, "--git-push")
	if code != 0 {
		t.Fatalf("an unavailable package manager must not block, got exit %d\n%s", code, out)
	}
	if strings.Contains(out, "fallback eslint ran") {
		t.Errorf("scripts.lint must suppress the ESLint fallback even when its manager is unavailable, got:\n%s", out)
	}
	if strings.Contains(out, "checks failed") {
		t.Errorf("tool unavailability must be a skip, not a failure, got:\n%s", out)
	}
}

func TestVerifyScriptDoesNotTreatCommandNotFoundOutputAsSkip(t *testing.T) {
	dir := t.TempDir()
	initRepo(t, dir)
	path := nodeOnlyPath(t)
	bin := strings.Split(path, string(os.PathListSeparator))[0]
	writeTestFile(t, dir, "package.json", `{"name":"fixture","packageManager":"pnpm@11.0.0","scripts":{"lint":"true"}}`)
	writeTestFile(t, bin, "pnpm", "#!/bin/sh\nprintf 'test assertion: command not found\\n' >&2\nexit 1\n")

	code, out := runVerify(t, dir, []string{"PATH=" + path}, "--git-push")
	if code != 2 {
		t.Fatalf("a real project failure must remain blocking, got exit %d\n%s", code, out)
	}
	if !strings.Contains(out, "test assertion: command not found") {
		t.Errorf("the real failure output must be preserved, got:\n%s", out)
	}
}

func TestVerifyScriptTreatsStartupExitAsSkip(t *testing.T) {
	dir := t.TempDir()
	initRepo(t, dir)
	path := nodeOnlyPath(t)
	bin := strings.Split(path, string(os.PathListSeparator))[0]
	writeTestFile(t, dir, "package.json", `{"name":"fixture","packageManager":"pnpm@11.0.0","scripts":{"lint":"true"}}`)
	writeTestFile(t, bin, "pnpm", "#!/bin/sh\nexit 127\n")

	code, out := runVerify(t, dir, []string{"PATH=" + path}, "--git-push")
	if code != 0 {
		t.Fatalf("a command startup failure must be a skip, got exit %d\n%s", code, out)
	}
	if strings.Contains(out, "checks failed") {
		t.Errorf("startup unavailability must not be a finding, got:\n%s", out)
	}
}

func TestVerifyScriptClassifiesUnavailableGlobalToolAsSkip(t *testing.T) {
	dir := t.TempDir()
	initRepo(t, dir)
	writeTestFile(t, dir, "go.mod", "module example.com/tool\n\ngo 1.21\n")
	writeTestFile(t, dir, "main.go", "package tool\n")
	gitUpdateRemote(t, dir)
	path := t.TempDir()
	writeTestFile(t, path, "golangci-lint", "#!/bin/sh\nprintf 'Volta error: missing project dependency\\n' >&2\nexit 1\n")
	pathEnv := "PATH=" + path + string(os.PathListSeparator) + os.Getenv("PATH")

	code, out := runVerify(t, dir, []string{pathEnv}, "--git-push")
	if code != 0 {
		t.Fatalf("an unavailable global linter must be a skip, got exit %d\n%s", code, out)
	}
	if strings.Contains(out, "advisory finding") || !strings.Contains(out, "golangci-lint") {
		t.Errorf("the unavailable global linter must be reported as a skip, got:\n%s", out)
	}
}

func TestVerifyScriptSupportsTypeScriptESLintConfig(t *testing.T) {
	dir := nodeLintFixture(t, true)
	if err := os.Remove(filepath.Join(dir, "eslint.config.mjs")); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, dir, "eslint.config.ts", "export default []\n")

	code, out := runVerify(t, dir, nil, "--git-push")
	if code != 0 {
		t.Fatalf("TypeScript flat config must remain advisory, got exit %d\n%s", code, out)
	}
	if !strings.Contains(out, "eslint (changed files)") || !strings.Contains(out, "advisory eslint finding") {
		t.Errorf("eslint.config.ts must be recognized by the advisory fallback, got:\n%s", out)
	}
}

func TestVerifyScriptDryRunLabelsUnavailableExplicitLint(t *testing.T) {
	dir := t.TempDir()
	initRepo(t, dir)
	path := nodeOnlyPath(t)
	writeTestFile(t, dir, "package.json", `{"name":"fixture","packageManager":"pnpm@11.0.0","scripts":{"lint":"true"}}`)

	code, out := runVerify(t, dir, []string{"PATH=" + path}, "--dry-run")
	if code != 0 {
		t.Fatalf("dry run must not fail, got %d\n%s", code, out)
	}
	if !strings.Contains(out, "[skip] package.json lint") {
		t.Errorf("dry run must show an unavailable explicit lint as skipped, got:\n%s", out)
	}
}

func TestVerifyScriptClassifiesUnavailableESLintAsSkip(t *testing.T) {
	dir := nodeLintFixture(t, true)
	writeTestFile(t, dir, "node_modules/.bin/eslint", "#!/usr/bin/env node\nconsole.error('Volta error: missing project dependency')\nprocess.exit(1)\n")

	code, out := runVerify(t, dir, nil, "--git-push")
	if code != 0 {
		t.Fatalf("an unavailable ESLint binary must be a skip, got exit %d\n%s", code, out)
	}
	if strings.Contains(out, "advisory eslint finding") || strings.Contains(out, "checks failed") {
		t.Errorf("tool unavailability must not be reported as a finding, got:\n%s", out)
	}
	if !strings.Contains(out, "eslint (changed files)") {
		t.Errorf("the skipped ESLint check must be named, got:\n%s", out)
	}
}

func TestVerifyScriptReportsGofmtFailureForIgnoredInvalidFile(t *testing.T) {
	dir := t.TempDir()
	initRepo(t, dir)
	writeTestFile(t, dir, "go.mod", "module example.com/ignored\n\ngo 1.21\n")
	writeTestFile(t, dir, "main.go", "package ignored\n")
	writeTestFile(t, dir, "ignored.go", "//go:build ignore\n\npackage ignored\nfunc broken( {\n")

	code, out := runVerify(t, dir, nil, "--git-push")
	if code != 0 {
		t.Fatalf("gofmt findings remain advisory, got exit %d\n%s", code, out)
	}
	if !strings.Contains(out, "advisory: gofmt -l .") || !strings.Contains(out, "expected ')'") {
		t.Errorf("gofmt stderr must be reported as an advisory, got:\n%s", out)
	}
}

func TestVerifyScriptPreservesPythonToolPathWithSpaces(t *testing.T) {
	dir := t.TempDir()
	initRepo(t, dir)
	toolDir := filepath.Join(t.TempDir(), "python tools")
	writeTestFile(t, toolDir, "ruff", "#!/bin/sh\nprintf 'ruff path ok\\n' >&2\nexit 1\n")
	writeTestFile(t, dir, "setup.py", "from setuptools import setup\n")
	writeTestFile(t, dir, "main.py", "value = 1\n")
	pathEnv := "PATH=" + toolDir + string(os.PathListSeparator) + os.Getenv("PATH")

	code, out := runVerify(t, dir, []string{pathEnv}, "--git-push")
	if code != 0 {
		t.Fatalf("a Python lint finding must remain advisory, got exit %d\n%s", code, out)
	}
	if !strings.Contains(out, "ruff path ok") {
		t.Errorf("the Python tool path must remain a single argument, got:\n%s", out)
	}
}

func TestVerifyScriptFindsWindowsPythonToolFromScripts(t *testing.T) {
	dir := t.TempDir()
	initRepo(t, dir)
	writeTestFile(t, dir, "pyproject.toml", "[tool.sqlfluff]\n")
	writeTestFile(t, dir, "query.sql", "select 1;\n")
	writeTestFile(t, dir, ".venv/Scripts/sqlfluff.exe", "#!/bin/sh\nprintf 'sqlfluff windows path\\n' >&2\nexit 1\n")

	code, out := runVerify(t, dir, nil, "--git-push")
	if code != 0 {
		t.Fatalf("a local Windows Python tool finding must remain advisory, got exit %d\n%s", code, out)
	}
	if !strings.Contains(out, "sqlfluff windows path") {
		t.Errorf(".venv/Scripts tools must be resolved, got:\n%s", out)
	}
}

func TestPrePushHookFailsClosedWithoutVerifier(t *testing.T) {
	dir := t.TempDir()
	initRepo(t, dir)

	code, out := runPrePushHook(t, dir, []string{"HOME=", "PATH=/usr/bin:/bin"})
	if code != 2 {
		t.Fatalf("a missing verifier must not allow an ungated push, got %d\n%s", code, out)
	}
	if !strings.Contains(out, "refusing an ungated push") && !strings.Contains(out, "HOME is not set") {
		t.Errorf("the hook must explain why it failed closed, got:\n%s", out)
	}
}

func TestPrePushHookFindsVerifierOnPathWithoutHome(t *testing.T) {
	dir := t.TempDir()
	initRepo(t, dir)
	bin := t.TempDir()
	writeTestFile(t, bin, "envctl-verify", "#!/bin/sh\nexit 7\n")
	pathEnv := "PATH=" + bin + string(os.PathListSeparator) + os.Getenv("PATH")

	code, out := runPrePushHook(t, dir, []string{"HOME=", pathEnv})
	if code != 7 {
		t.Fatalf("the PATH verifier must remain authoritative, got %d\n%s", code, out)
	}
}

func TestVerifyScriptHookModeIgnoresRetry(t *testing.T) {
	dir := t.TempDir()
	initRepo(t, dir)
	writeTestFile(t, dir, ".commandcode/verify.sh", "#!/bin/sh\nexit 1\n")

	payload := `{"hook_event_name":"Stop","stop_hook_active":true}`
	cmd := exec.Command("bash", verifyScriptPath(t), "--hook")
	cmd.Dir = dir
	cmd.Env = testEnv(nil)
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
		cmd.Env = testEnv(nil)
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
		cmd.Env = testEnv(nil)
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
