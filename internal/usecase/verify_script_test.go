package usecase

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// The verifier is a shell script but it sits on the critical path of every push,
// so it is tested like code: each case builds a throwaway repository, runs the
// real script and asserts the exit code and the report. Optional tools
// (shellcheck, hadolint, tsc) are skipped when absent, which keeps the suite
// deterministic on a bare runner.

// bashBin resolves a POSIX shell on any OS. On Windows the Git for Windows bash
// is a first-class test host (the fixtures are POSIX scripts), so tests run
// there instead of being skipped — a Windows skip hides real breakage from the
// local run and leaves only CI to catch it. The WSL launcher (System32\bash.exe)
// is deliberately not a host: it sees a different filesystem namespace.
func bashBin(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		for _, candidate := range []string{
			filepath.Join(os.Getenv("ProgramFiles"), "Git", "bin", "bash.exe"),
			filepath.Join(os.Getenv("ProgramFiles(x86)"), "Git", "bin", "bash.exe"),
			filepath.Join(os.Getenv("LocalAppData"), "Programs", "Git", "bin", "bash.exe"),
		} {
			if !strings.HasSuffix(strings.ToLower(candidate), `\git\bin\bash.exe`) {
				continue
			}
			if _, err := os.Stat(candidate); err == nil {
				return candidate
			}
		}
	}
	if path, err := exec.LookPath("bash"); err == nil {
		lower := strings.ToLower(filepath.ToSlash(path))
		if !strings.Contains(lower, "/windows/system32/") {
			return path
		}
	}
	t.Skip("bash not available (install Git for Windows)")
	return ""
}

func verifyScriptPath(t *testing.T) string {
	t.Helper()
	bashBin(t) // skip when no POSIX shell is available on this OS
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

// runVerify executes the verifier with dir as the working directory. The script
// path is given to bash in slash form: backslashes are escape characters for a
// POSIX shell (C:\x\y becomes C:xy).
func runVerify(t *testing.T, dir string, env []string, args ...string) (int, string) {
	t.Helper()
	cmd := exec.Command(bashBin(t), append([]string{filepath.ToSlash(verifyScriptPath(t))}, args...)...)
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

	cmd := exec.Command(bashBin(t), filepath.ToSlash(verifyScriptPath(t)), "--git-push")
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

// The prettier advisory must stay scoped to the changed code/style files its
// selection covers: docs must never leak into it. The stub prettier prints its
// own argv, which pins the selection exactly and independent of toolchains.
func TestVerifyPrettierAdvisoryScopedToCodeFiles(t *testing.T) {
	dir := t.TempDir()
	initRepo(t, dir)
	writeTestFile(t, dir, "package.json", `{"name":"fixture","private":true}`)
	writeTestFile(t, dir, ".prettierrc", "{}\n")
	writeTestFile(t, dir, "note.md", "# unformatted   \n")
	writeTestFile(t, dir, "app.ts", "const x   =   1\n")
	bin := t.TempDir()
	writeTestFile(t, bin, "prettier", "#!/bin/sh\nprintf 'PRETTIER-ARGS:'\nfor arg in \"$@\"; do printf ' <%s>' \"$arg\"; done\nprintf '\\n'\nexit 1\n")
	pathEnv := "PATH=" + bin + string(os.PathListSeparator) + os.Getenv("PATH")

	code, out := runVerify(t, dir, []string{pathEnv}, "--hook")
	if code != 0 {
		t.Fatalf("a prettier finding is advisory only, got exit %d\n%s", code, out)
	}
	if !strings.Contains(out, "PRETTIER-ARGS:") {
		t.Fatalf("the advisory must invoke prettier on the selected files, got:\n%s", out)
	}
	if !strings.Contains(out, "<app.ts>") {
		t.Errorf("the changed code file must be selected, got:\n%s", out)
	}
	if strings.Contains(out, "<note.md>") {
		t.Errorf("docs must not leak into the prettier advisory selection, got:\n%s", out)
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

// A hung check must fail fast at the per-check timeout instead of blocking the
// push forever. .commandcode/verify.sh runs through the blocking `check` helper,
// so it exercises run_capped without depending on any toolchain.
func TestVerifyTimeoutBoundsHungCheck(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("envctl-verify is a POSIX shell script")
	}
	if _, err := exec.LookPath("timeout"); err != nil {
		t.Skip("coreutils timeout not available")
	}
	dir := t.TempDir()
	initRepo(t, dir)
	// `exec sleep 30` replaces the shell so there is no orphan child holding the
	// output pipe; without the per-check timeout this hangs for the full 30s.
	writeTestFile(t, dir, ".commandcode/verify.sh", "#!/bin/sh\nexec sleep 30\n")

	start := time.Now()
	code, out := runVerify(t, dir, []string{"ENVCTL_VERIFY_TIMEOUT=1"}, "--git-push")
	elapsed := time.Since(start)
	if elapsed > 5*time.Second {
		t.Fatalf("a hung check must be bounded by the per-check timeout, took %s", elapsed)
	}
	if code != 2 {
		t.Fatalf("a timed-out blocking check must abort the push (exit 2), got %d\n%s", code, out)
	}
	if !strings.Contains(out, "TIMED OUT") {
		t.Errorf("the failure must name the timeout, got:\n%s", out)
	}
}

// --hook is a legacy alias of --static: old callers keep working with the same
// semantics (static half only — the test suite never blocks).
func TestVerifyStaticModeAcceptsLegacyHookAlias(t *testing.T) {
	dir := goFixture(t)

	code, out := runVerify(t, dir, nil, "--hook")
	if code != 0 {
		t.Fatalf("the legacy --hook alias must behave like --static (failing tests never block), got exit %d\n%s", code, out)
	}
	if strings.TrimSpace(out) != "" {
		t.Errorf("a green static run must stay silent, got:\n%s", out)
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

func TestVerifyHookModeRunsStaticChecksOnly(t *testing.T) {
	dir := goFixture(t)
	payload := `{"hook_event_name":"Stop","stop_hook_active":false}`

	runHook := func() (int, string) {
		cmd := exec.Command(bashBin(t), filepath.ToSlash(verifyScriptPath(t)), "--hook")
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
}
