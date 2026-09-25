# Advisory Verifier Checks

## Goal

Make `envctl-verify` report inferred lint and formatting findings without
blocking a push. A repository's explicit commands remain authoritative: a
`lint` script in `package.json` and `.commandcode/verify.sh` are blocking.
Build, vet, and test failures remain blocking because they represent
repository correctness rather than an inferred style policy.

## Current problem

`check_node` passes every changed `.js`, `.mjs`, `.cjs`, `.ts`, and `.tsx`
file to the local ESLint binary. In a repository whose flat config uses
`strictTypeChecked` without type information for `.mjs`, a changed
`eslint.config.mjs` fails before the project's own `pnpm lint` can define the
correct scope. The verifier currently turns that inferred policy mismatch into
exit 2.

## Contract

- Inferred lint/format checks are advisory: they run, report `ADVISORY`, and
  do not increment the blocking failure count.
- Inferred build, vet, and test checks remain blocking.
- `.commandcode/verify.sh` remains blocking and keeps replacing inferred stack
  checks.
- A `package.json` `lint` script is an explicit project check and runs through
  the detected package manager (`pnpm`, `npm`, `yarn`, or `bun`) as blocking;
  npm is the default when no metadata/lockfile is present.
- When no explicit lint script exists, the local ESLint fallback is advisory
  and excludes `eslint.config.js`, `eslint.config.mjs`, and
  `eslint.config.cjs` from the changed-file arguments.
- Tool-unavailable results remain skips, not advisories or failures. Explicit
  project scripts and file-based linters use the soft/unavailable classifier;
  `gofmt` captures both stdout and stderr.
- `--dry-run` labels blocking, advisory, and skipped checks so the policy is
  auditable.
- Hook caching includes the comparison ref, global/local tool availability and
  metadata, bypasses executable repository overrides, disables itself for
  oversized untracked files, and never reuses a stamp after a skip.
- Tool-startup classification uses exit codes and bounded shim signatures, so
  a real check failure containing "command not found" remains blocking.
- The global pre-push resolves the verifier through `HOME` or `PATH` and fails
  closed when neither is available.
- ESLint flat-config detection covers `eslint.config.{js,mjs,cjs,ts,mts,cts}`.
- Python tools are kept as Bash argument arrays, including Windows
  `.venv/Scripts/*.exe` locations and paths containing spaces.
- Advisory-only runs exit 0 and may be cached in hook mode; explicit blocking
  failures still exit 2.

## Files

- `configs/bin/envctl-verify`: severity-aware check helpers, package-script
  detection, ESLint config filtering, cache/tool fingerprinting, advisory
  summaries/trail.
- `configs/git/hooks/pre-push`: fail-closed verifier resolution when `HOME`
  is absent or the verifier is not installed.
- `internal/usecase/verify_script_test.go`: regression fixtures and contract
  tests.
- `docs/verification.md`: document the blocking/advisory boundary and trail.
- `README.md`, `AGENTS.md`, `docs/os-and-agent-matrix.md`: align local quality
  gate descriptions with the new contract.
- `spec-agent/2026-09-24-advisory-verification.md`: this plan.

## TDD tasks

1. Add failing tests for inferred ESLint advisory output, config-file
   exclusion (including TypeScript flat-config names), explicit `pnpm lint`
   blocking, dry-run severity labels, tool skips, cache invalidation, and
   path-safe Python tool resolution.
2. Run the focused tests and confirm they fail because the old script exits 2,
   caches incomplete states, or lacks advisory markers.
3. Implement the smallest severity-aware shell changes and package-script
   detection; preserve explicit blocking checks and harden cache/tool
   classification.
4. Run focused tests, then shell syntax/format/lint checks.
5. Update documentation and run the full Go gate plus `envctl-verify --dry-run`
   and `envctl-verify --git-push` against the worktree.

## Verification commands

```sh
go test ./internal/usecase -run '^TestVerifyScript' -count=1 -v
bash -n configs/bin/envctl-verify
shellcheck configs/bin/envctl-verify
shfmt -d configs/bin/envctl-verify
go build ./...
go vet ./...
go test ./...
golangci-lint run --new-from-rev=origin/main ./...
bash configs/bin/envctl-verify --dry-run
bash configs/bin/envctl-verify --git-push
git diff --check
```
