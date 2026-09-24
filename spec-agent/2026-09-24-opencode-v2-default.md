# OpenCode v2 Default Provider Implementation Plan

> **For agentic workers:** implement this plan task-by-task — dispatch a fresh `general` subagent per task via the task tool, or execute inline with checkpoints. Steps use checkbox (`- [x]`) syntax for tracking.

**Goal:** Make the official OpenCode V2 channel the default on Ubuntu/Debian in both `run providers` and `run bootstrap`, while preserving pacman ownership on Arch/CachyOS and converging existing V1 user-local installs.

**Architecture:** Reuse the official `https://opencode.ai/v2/install` installer and add `~/.opencode/bin` to the toolchain PATH used by non-login provisioning, with an idempotent POSIX/fish profile writer. Represent a provider's required major version in `providerCLIs()`, validate installed versions after standalone installs, query `pacman -Q` for package ownership, archive stale envctl user copies, and update package-owned Arch binaries only through the injected package manager.

**Tech Stack:** Go 1.x, `os/exec`, Cobra, Bash, official OpenCode V2 installer.

**Spec:** This plan is the design for the requested Ubuntu V2-default correction; no separate spec exists.

## Global Constraints

- Use the official V2 endpoint `https://opencode.ai/v2/install`; do not use the legacy `https://opencode.ai/install` endpoint.
- Do not install or update OpenCode through npm/Volta.
- Do not shadow a binary owned by Arch pacman; report it for package-manager update.
- Keep provisioning idempotent: an existing V2 binary must be a no-op.
- Keep changes compatible with non-login SSH/systemd execution.
- Code, comments, tests, and documentation are in English; user-facing explanation remains PT-BR.

---

### Task 1: Define and test the V2 version/ownership policy

**Files:**
- Modify: `internal/usecase/provision_providers.go`
- Test: `internal/usecase/provision_providers_test.go`

**Interfaces:**
- Produces `versionMajorAtLeast(version string, major int) bool`.
- Adds a required-major field to `providerCLI` and a pure ownership decision helper used by both provisioning paths.

- [x] **Step 1: Write the failing tests**

Cover `2.0.16`, `v2.0.16`, `1.18.32`, malformed/empty versions, V1 system ownership on Arch, and V1 system ownership on Ubuntu. Assert that the OpenCode provider declares a V2 requirement and uses the V2 endpoint.

- [x] **Step 2: Run the focused tests and confirm the expected failures**

Run: `go test ./internal/usecase -run 'Test(VersionMajorAtLeast|ProviderV2|StandaloneProvider)' -v`
Expected: FAIL because the helpers/field and V2 policy do not exist yet.

- [x] **Step 3: Implement the minimal policy helpers**

Parse only the numeric major component, normalize a leading `v`, and classify a standalone V1 as replaceable only when no pacman-owned package exists. Add `requiredMajor: 2` to OpenCode.

- [x] **Step 4: Run the focused tests and confirm they pass**

Run: `go test ./internal/usecase -run 'Test(VersionMajorAtLeast|ProviderV2|StandaloneProvider)' -v`
Expected: PASS.

---

### Task 2: Make phase 0 converge Ubuntu OpenCode to V2

**Files:**
- Modify: `internal/usecase/provision_providers.go`
- Test: `internal/usecase/provision_providers_test.go`

**Interfaces:**
- `providerCLIs()` uses the shared V2 installer for Linux.
- `ensureProviderCLI` upgrades an existing compatible-to-replace V1 install and validates the post-install major version.

- [x] **Step 1: Add failing behavior tests**

Assert that a V1 standalone provider is eligible for the V2 installer when the source is user-local or the host is non-pacman, and ineligible when the source is system-owned on pacman. Assert the installer script contains `/v2/install` and not the legacy endpoint.

- [x] **Step 2: Run the focused tests and confirm the expected failure**

Run: `go test ./internal/usecase -run 'Test(ProviderV2|StandaloneProvider)' -v`
Expected: FAIL because phase 0 still uses the legacy Linux installer and skips existing standalone binaries.

- [x] **Step 3: Implement the phase-0 upgrade path**

Replace the Linux provider installer with the official V2 endpoint. In `ensureProviderCLI`, handle a present V1 standalone provider before the generic “present but not Volta” branch when the ownership policy permits replacement. After every standalone install, probe `--version` and emit a warning if the result is missing or below the required major.

- [x] **Step 4: Run the focused tests and confirm they pass**

Run: `go test ./internal/usecase -run 'Test(ProviderV2|StandaloneProvider)' -v`
Expected: PASS.

---

### Task 3: Make Linux bootstrap converge Ubuntu OpenCode to V2

**Files:**
- Modify: `internal/usecase/provision_bootstrap.go`
- Modify: `internal/usecase/provision_providers.go` (shared PATH/ownership helpers)
- Modify: `internal/usecase/provision_packages.go` (Arch package-database probe)
- Modify: `internal/ui/cli/root.go` (inject package managers into bootstrap)
- Modify: `internal/usecase/doctor_audit.go` and its test (numeric major comparison)
- Test: `internal/usecase/provision_bootstrap_test.go`
- Test: `internal/usecase/provision_packages_test.go`

**Interfaces:**
- `linuxToolchainEnv` and `ensureProcessToolchainPath` include `$HOME/.opencode/bin` before the existing local/Volta directories.
- `classifyInstallSource` treats the official user-local `.opencode/bin` path as envctl-owned.
- `ProvisionBootstrapUseCase` verifies the installed major after provisioning OpenCode.

- [x] **Step 1: Add failing tests**

Test that the Linux toolchain PATH contains `.opencode/bin`, that `.opencode/bin/opencode` is classified as envctl-owned, and that the bootstrap V2 decision upgrades a user-local V1 but leaves a pacman-owned system V1 alone.

- [x] **Step 2: Run the focused tests and confirm the expected failures**

Run: `go test ./internal/usecase -run 'Test(OpenCodePath|OpenCodeBootstrapV2|ClassifyInstallSource)' -v`
Expected: FAIL because `.opencode/bin` is absent and bootstrap skips any existing binary.

- [x] **Step 3: Implement the bootstrap convergence path**

Replace the legacy `run bootstrap` OpenCode step with a V2-aware helper. Treat V2 as idempotent success; install/update through the shared V2 script when ownership allows; use the distro package path/report behavior for a system-owned Arch binary; validate the resulting major before reporting success.

- [x] **Step 4: Run the focused tests and confirm they pass**

Run: `go test ./internal/usecase -run 'Test(OpenCodePath|OpenCodeBootstrapV2|ClassifyInstallSource)' -v`
Expected: PASS.

---

### Task 4: Synchronize documentation and memory

**Files:**
- Modify: `docs/os-and-agent-matrix.md`
- Modify: `configs/skills/vps-agent-dispatch/SKILL.md`
- Modify: `CHANGELOG.md`
- Modify: `.opencode/memory/lessons.md` if the previous channel note is now obsolete

- [x] **Step 1: Update documentation**

Describe the official `/v2/install` endpoint, `~/.opencode/bin`, V1 convergence on Ubuntu/Debian, and the Arch pacman exception. Remove any claim that the only V2 channel is the versioned files endpoint or that the legacy endpoint is used by default.

- [x] **Step 2: Run repository consistency checks**

Run: `grep -RIn 'opencode.ai/install\\|opencode.ai/v2/install' docs configs internal CHANGELOG.md | head -n 80`
Expected: no stale Linux default instruction remains; the legacy URL may appear only in an explicitly labeled V1 comparison.

- [x] **Step 3: Commit the implementation and docs**

Run: `git add internal/usecase/provision_providers.go internal/usecase/provision_bootstrap.go internal/usecase/*test.go docs/os-and-agent-matrix.md configs/skills/vps-agent-dispatch/SKILL.md CHANGELOG.md .opencode/memory/lessons.md && git commit -m "fix(providers): default OpenCode Linux installs to v2"`

---

## Final Verification

- [x] `gofmt -w` the changed Go files and confirm `gofmt -l .` is empty.
- [x] Run `go build ./...`.
- [x] Run `go vet ./...`.
- [x] Run `go test ./...`.
- [x] Run `golangci-lint run --new-from-rev=origin/main` and require zero issues.
- [x] Build a Linux binary and run `run providers`/`run bootstrap` against a disposable Ubuntu user or the existing homologation host; verify `opencode --version` reports major 2 and a second run performs no download.
- [x] Verify Arch behavior with the existing doctor/provider tests: `pacman -Q` decides ownership, stale user-local copies are archived, and the system package remains authoritative.
