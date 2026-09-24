# Linux Performance Baseline Implementation Plan

> **For agentic workers:** implement this plan task-by-task with a fresh test checkpoint for each task. Do not mix Ubuntu and CachyOS state or mutate privileged settings outside the explicit performance command.

**Goal:** Add an OS-specific, opt-in performance baseline for Ubuntu Server 24.04+ and a separate CachyOS performance audit/provisioning path, while absorbing verified headless CLI tools from CachyOS into Ubuntu 24.04+.

**Architecture:** Extend package applicability with exact distro/version constraints instead of treating Ubuntu and all Arch-family systems as interchangeable. Add separate `performance_ubuntu.yaml` and `performance_cachyos.yaml` manifests, a `run performance` use case that selects exactly one profile, and a Linux performance adapter for idempotent sysctl application and read-only inspection. The default `run all` receives only the Ubuntu 24.04+ headless toolbox; privileged performance changes remain opt-in.

**Tech Stack:** Go 1.x, Cobra, YAML manifests, existing package-manager/repository interfaces, existing embedded asset repository, `sudo -n` for non-interactive Linux elevation, `sysctl`, systemd, procfs/sysfs.

**Spec:** This document is the design contract for the branch.

## Global Constraints

- Target Ubuntu is exactly Ubuntu `VERSION_ID >= 24.04`; Debian and older Ubuntu releases must not receive the Ubuntu 24.04 toolbox.
- CachyOS is a separate target from generic Arch; do not use an Ubuntu manifest on CachyOS or vice versa.
- `run performance` is opt-in and is not part of `run all`; `run all` may install only the verified Ubuntu 24.04+ headless CLI entries.
- All package and config provisioning is idempotent and preserves backups.
- `sysctl` and the zram generator lifecycle are the only automatically applied performance mutations in this branch; no zram sizing/priority or unrelated tuning is changed.
- Do not create, resize, remove, or prioritize swapfiles automatically. CachyOS zram and Ubuntu zram package installation are allowed; swapfile creation is a later, explicitly approved phase.
- Do not mutate journald limits, CPU governors, I/O schedulers, mitigations, kernel command lines, or unrelated systemd services. The zram generator lifecycle is the one explicit exception; inspect and report all other state read-only.
- No benchmark runs against production or a live server without a separate workload and maintenance approval.
- No hostnames, IPs, UUIDs, filesystem identifiers, or machine-specific paths enter the repository.

---

### Task 1: Add exact distro/version applicability for packages

**Files:**
- Modify: `internal/domain/entity/models.go`
- Modify: `internal/domain/entity/platform.go`
- Modify: `internal/domain/entity/platform_test.go`
- Modify: `internal/domain/entity/manifest_os_lint_test.go`
- Modify: `internal/usecase/provision_packages.go`
- Modify: `internal/usecase/doctor_audit.go`

**Interfaces:**
- Add `TargetDistro string `yaml:"target_distro,omitempty"`` and `MinDistroVersion string `yaml:"min_distro_version,omitempty"`` to `entity.Package`.
- Add a pure `entity.PackageMatchesPlatform(pkg entity.Package, goos, family, id, version string) bool` helper.
- Add pure parsing/comparison helpers for `VERSION_ID`; keep `DetectedDistro()` and existing `MatchOS()` behavior compatible for current manifests.
- `PackageMatchesPlatform` must require `MatchesOS(pkg.OS)`, then exact `TargetDistro` when set, then semantic-version minimum when `MinDistroVersion` is set.

- [x] **Step 1: Write failing tests for exact target and minimum version.**

Add table cases to `platform_test.go` covering Ubuntu 24.04, Ubuntu 22.04, Debian 12, CachyOS, generic Arch, and Windows. Add a test that a package with `TargetDistro: "ubuntu"` and `MinDistroVersion: "24.04"` is rejected on Ubuntu 22.04 and accepted on Ubuntu 24.04. Add a manifest lint assertion for the new YAML fields.

- [x] **Step 2: Run the focused tests and verify RED.**

Run:

```bash
go test ./internal/domain/entity -run 'Test(PackageMatchesPlatform|MatchOS|ManifestOSLint)' -count=1
```

Expected: FAIL because the new fields/helpers are not implemented.

- [x] **Step 3: Implement the smallest platform matcher.**

Read `ID` and `VERSION_ID` from `/etc/os-release` without changing existing family constants. Parse dotted numeric versions with a small pure helper. Do not infer Ubuntu from `ID_LIKE` when `TargetDistro` is exact.

- [x] **Step 4: Route package provisioning and doctor through the new helper.**

Replace direct `entity.MatchesOS(pkg.OS)` checks in package provisioning and package auditing with `entity.PackageMatchesPlatform(...)` using detected platform data. Keep all existing entries with empty new fields behaving exactly as before.

- [x] **Step 5: Run focused tests and verify GREEN.**

```bash
go test ./internal/domain/entity -count=1
go test ./internal/usecase -run 'Test.*Package|Test.*Doctor' -count=1
```

Expected: PASS with no changed legacy platform behavior.

---

### Task 2: Add verified Ubuntu 24.04+ headless CLI parity

**Files:**
- Modify: `manifests/packages.yaml`
- Modify: `internal/infra/embedded/manifest_repo_test.go`
- Modify: `docs/manifests.md`
- Modify: `docs/guides/linux.md`
- Modify: `docs/os-and-agent-matrix.md`
- Modify: `CHANGELOG.md` only under `[Unreleased]`

**Manifest entries:**
- Add APT siblings for `eza`, `tmux`, `sqlite3`, `restic`, `rclone`, `btop`, `duf`, `glances`, `micro`, `cmake`, `ninja-build`, `mosh`, `nvtop`, `iotop`, `sysstat`, `zstd`, and `lz4`.
- Use `type: apt`, `os: ubuntu`, `target_distro: ubuntu`, and `min_distro_version: "24.04"` for every new entry.
- Do not add packages absent from Ubuntu 24.04 repositories (`fastfetch`, `lazygit`, and `lazydocker` are not baseline entries).
- Do not duplicate the existing portable core entries (`git`, `curl`, `fzf`, `bat`, `fd-find`, `jq`, `rsync`, `unzip`, `zip`).
- Keep all CachyOS/Arch entries unchanged and separate.

- [x] **Step 1: Verify package candidates on the Ubuntu 24.04 target.**

Run on the target or a disposable 24.04 image:

```bash
apt-cache policy eza tmux sqlite3 restic rclone btop duf glances micro cmake ninja-build mosh nvtop iotop sysstat zstd lz4
```

Expected: every listed entry has a non-empty `Candidate`; unavailable candidates are removed rather than guessed.

- [x] **Step 2: Add a failing manifest test for exact Ubuntu entries.**

In `manifest_repo_test.go`, load packages and assert the new IDs have `TypeApt`, `TargetDistro == "ubuntu"`, and `MinDistroVersion == "24.04"`. Assert no new entry uses a bare `debian,ubuntu` scope.

- [x] **Step 3: Run the test and verify RED.**

```bash
go test ./internal/infra/embedded -run TestLoadManifestsFromDiskOrEmbed -count=1
```

Expected: FAIL because the APT siblings are absent.

- [x] **Step 4: Add the verified YAML entries and documentation.**

Keep the manifest ordered by OS/tool category. Document that these are server-safe headless tools, that no remotes/configuration are provisioned for `restic`/`rclone`, and that GUI-only Arch tools are not copied blindly.

- [x] **Step 5: Run focused tests and verify GREEN.**

```bash
go test ./internal/infra/embedded -run TestLoadManifestsFromDiskOrEmbed -count=1
go test ./internal/domain/entity -run TestManifestOSLint -count=1
```

Expected: PASS.

---

### Task 3: Add separate performance manifests and an opt-in use case

**Files:**
- Create: `manifests/performance_ubuntu.yaml`
- Create: `manifests/performance_cachyos.yaml`
- Create: `internal/domain/entity/performance.go`
- Create: `internal/domain/repository/performance.go`
- Create: `internal/usecase/provision_performance.go`
- Create: `internal/usecase/provision_performance_test.go`
- Modify: `internal/domain/repository/interfaces.go`
- Modify: `internal/infra/embedded/manifest_repo.go`
- Modify: `internal/infra/embedded/manifest_repo_test.go`
- Modify: `internal/ui/cli/root.go`
- Modify: `internal/ui/cli/run.go`

**Data contracts:**

```go
type PerformanceProfile string

const (
    PerformanceProfileUbuntu  PerformanceProfile = "ubuntu-24.04"
    PerformanceProfileCachyOS PerformanceProfile = "cachyos"
)

type SysctlSetting struct {
    Key       string `yaml:"key"`
    Value     string `yaml:"value"`
    Rationale string `yaml:"rationale"`
}

type PerformanceSpec struct {
    Profile  PerformanceProfile
    Packages []Package
    Sysctls  []SysctlSetting
}
```

The repository exposes `LoadPerformanceSpec(profile entity.PerformanceProfile) (entity.PerformanceSpec, error)`. The use case exposes:

```go
ExecutePerformance(ctx context.Context, profile entity.PerformanceProfile, dryRun bool, onProgress PackageProgressHandler) ([]entity.Package, []entity.Diagnostic, error)
```

**Manifest policy:**

- `performance_ubuntu.yaml`: `systemd-zram-generator` via APT; after package provisioning, the zram adapter loads the module, runs `systemctl daemon-reload`, and starts the generated `dev-zram0.swap` when `/dev/zram0` is absent, plus the conservative Ubuntu server sysctl set below.
- `performance_cachyos.yaml`: `zram-generator` via pacman; the zram adapter starts the generated `dev-zram0.swap` only when the device is absent; no sysctl mutation because the current CachyOS profile already has a tuned zram and generic scheduler/governor changes are unsafe.
- Ubuntu sysctl baseline: `vm.swappiness=10`, `vm.vfs_cache_pressure=50`, `net.core.somaxconn=65535`, `net.ipv4.tcp_max_syn_backlog=4096`, `fs.file-max=2097152`. These are server-oriented defaults, not universal performance guarantees.
- Neither manifest creates a swapfile, changes journald, changes a governor, changes an I/O scheduler, or changes any unrelated service; only the zram generator lifecycle may start.
- CachyOS `run performance` must be a no-op when zram-generator is already installed; it must never create `/dev/zram1` or a second swap device.

- [x] **Step 1: Write failing use-case tests.**

Cover: Ubuntu 24.04 selects only the Ubuntu spec; Ubuntu 22.04 is rejected before package installation; CachyOS and generic Arch do not cross-select each other's packages; installed packages are no-op; dry-run never calls `Install`; the zram adapter loads the module, reloads units, and starts the generated swap unit only when the device is absent; unavailable package manager returns a diagnostic/error; a CachyOS run with installed zram-generator does not mutate sysctls.

- [x] **Step 2: Run the tests and verify RED.**

```bash
go test ./internal/usecase -run 'Test.*Performance' -count=1
```

Expected: FAIL because the repository method, entity types, and use case do not exist.

- [x] **Step 3: Add the pure repository loader and profile selection.**

Parse each performance manifest independently. Reject an unsupported profile and reject a host whose exact ID/version does not match the profile. Do not infer CachyOS from the Arch family.

- [x] **Step 4: Add `run performance` with `--dry-run`.**

The command is registered under `run`, is excluded from `run all` and `doctor --fix`, prints the selected exact profile, provisions only the selected manifest, and returns a non-zero error for an unsupported OS/release. `run performance --dry-run` must not call package `Install`; it may call the sysctl adapter only in its explicit dry-run mode, which performs no write or apply.

- [x] **Step 5: Run focused tests and verify GREEN.**

```bash
go test ./internal/usecase -run 'Test.*Performance' -count=1
go test ./internal/infra/embedded -run TestLoadManifestsFromDiskOrEmbed -count=1
```

Expected: PASS.

---

### Task 4: Apply Ubuntu sysctls atomically and only when explicitly requested

**Files:**
- Create: `internal/infra/performance/sysctl_manager.go`
- Create: `internal/infra/performance/sysctl_manager_test.go`
- Modify: `internal/domain/repository/performance.go`
- Modify: `internal/usecase/provision_performance.go`
- Modify: `internal/ui/cli/root.go`

**Interface:**

```go
type SysctlManager interface {
    Apply(ctx context.Context, settings []entity.SysctlSetting, dryRun bool) ([]entity.Diagnostic, error)
}
```

**Behavior:**

- Only the Ubuntu profile passes a non-empty sysctl list.
- Write `/etc/sysctl.d/90-envctl-performance.conf` through a temporary file and `sudo -n install` (or direct install as root), preserving an existing file as `<path>.bak.YYYYMMDD-HHMMSS` before replacement.
- If the managed file is byte-for-byte identical, do not create a backup or rewrite it.
- Apply only the managed file with `sysctl -p`; do not run `sysctl --system` against unrelated files.
- Never invoke a shell string. Use an argv-based command and a non-interactive `sudo -n`.
- If a sysctl key is unavailable, return a diagnostic with the exact key and leave the file available for rollback; never silently claim success.
- Dry-run reports the destination and keys without writing or applying them.
- A failed apply reports the backup path and leaves the previous file recoverable.

- [x] **Step 1: Write failing infrastructure tests with an injectable command runner.**

Test exact-content no-op, timestamped backup, non-root `sudo -n` argv, failed `sysctl -p`, dry-run, and rollback metadata. Do not invoke real `/etc` in unit tests.

- [x] **Step 2: Run the tests and verify RED.**

```bash
go test ./internal/infra/performance -run TestSysctl -count=1
```

Expected: FAIL because the package and adapter do not exist.

- [x] **Step 3: Implement the minimal argv-based adapter.**

Keep the command runner injectable for tests. Use `os.CreateTemp` in a user-writable temporary directory, then install the file with the appropriate elevation path. Use mode `0644` for the sysctl drop-in.

- [x] **Step 4: Wire the adapter into the Ubuntu profile only.**

CachyOS must never receive a sysctl apply call in this branch. Wire the adapter through `AppContext` and inject a no-op/failing implementation in tests as appropriate.

- [x] **Step 5: Run focused tests and verify GREEN.**

```bash
go test ./internal/infra/performance -run TestSysctl -count=1
go test ./internal/usecase -run 'Test.*Performance' -count=1
```

Expected: PASS.

---

### Task 5: Add read-only OS-specific performance auditing

**Files:**
- Create: `internal/infra/performance/inspector.go`
- Create: `internal/infra/performance/inspector_test.go`
- Create: `internal/usecase/doctor_linux_performance.go`
- Create: `internal/usecase/doctor_linux_performance_test.go`
- Modify: `internal/domain/repository/performance.go`
- Modify: `internal/usecase/doctor_audit.go`
- Modify: `internal/usecase/doctor_audit_test.go`
- Modify: `internal/ui/cli/root.go`

**Snapshot contract:**

```go
type PerformanceSnapshot struct {
    Platform       entity.PlatformInfo
    Swap           []entity.SwapDevice
    ZRAM           entity.ZRAMState
    CPUGovernor    string
    BlockSchedulers []entity.BlockScheduler
    Journald       entity.JournaldState
    FSTRIMTimer    entity.TimerState
    Services       []entity.ServiceState
}
```

The inspector must read, when available:

- `/proc/swaps` and zram sysfs/`zramctl`;
- CPU governor paths;
- block-device scheduler paths;
- `fstrim.timer` state;
- journald disk usage/effective config;
- a small fixed service list.

Missing files/commands are reported as unavailable information, not as errors. The doctor must never demand a disk swap, a specific governor, a specific scheduler, or a specific journald size. It may warn only when an explicitly installed zram generator has no active device after a completed provisioning run; otherwise optional performance state is `INFO`/`OK`.

- [x] **Step 1: Write failing parser and doctor tests.**

Use fixture strings for `/proc/swaps`, zram state, schedulers, and journald output. Assert that missing disk swap is informational, governor/scheduler differences are not warnings, and CachyOS active zram is reported correctly.

- [x] **Step 2: Run the tests and verify RED.**

```bash
go test ./internal/infra/performance ./internal/usecase -run 'Test.*(Performance|Snapshot|Swap|ZRAM)' -count=1
```

Expected: FAIL because the inspector and doctor integration do not exist.

- [x] **Step 3: Implement read-only inspection and aggregate diagnostics.**

Keep command execution bounded with context and use absolute tool paths where needed. Do not run benchmarks or mutate state. Keep output compact: one diagnostic per category rather than one per CPU/device.

- [x] **Step 4: Run focused tests and verify GREEN.**

```bash
go test ./internal/infra/performance ./internal/usecase -run 'Test.*(Performance|Snapshot|Swap|ZRAM)' -count=1
```

Expected: PASS.

---

### Task 6: Document the operating policy and verify the full branch

**Files:**
- Create: `docs/guides/linux-performance.md`
- Create: `configs/skills/linux-performance-tuning/SKILL.md`
- Modify: `manifests/skills.yaml`
- Modify: `configs/SKILL-INDEX.md`
- Modify: `configs/commandcode/SKILL-INDEX.md`
- Modify: `docs/skills.md`
- Modify: `docs/os-and-agent-matrix.md`
- Modify: `docs/manifests.md`
- Modify: `docs/doctor-and-idempotency.md`
- Modify: `CHANGELOG.md` only under `[Unreleased]`

The skill must document:

- Ubuntu 24.04+ profile and exact command;
- CachyOS profile and why existing zram is preserved;
- what `run performance` changes and what it never changes;
- how to inspect and roll back the sysctl drop-in;
- why swapfile creation, journald caps, governors, schedulers, services, mitigations, and kernel parameters remain manual/benchmark-gated;
- a future swapfile plan requiring filesystem, size, encryption, hibernation, and rollback decisions;
- benchmark procedure restricted to staging/maintenance.

- [x] **Step 1: Add the skill and documentation.**

Use YAML block-scalar frontmatter descriptions, no machine-specific values, and no PII. Update all skill indexes and the expected skill count in manifest tests.

- [x] **Step 2: Validate YAML/frontmatter and documentation references.**

```bash
go test ./internal/infra/embedded -run 'TestLoadManifestsFromDiskOrEmbed' -count=1
go test ./internal/domain/entity -run TestManifestOSLint -count=1
```

- [x] **Step 3: Run the complete project gate.**

```bash
gofmt -w internal/domain/entity internal/domain/repository internal/infra/performance internal/infra/embedded internal/usecase internal/ui/cli
go build ./...
go vet ./...
go test ./...
golangci-lint run --new-from-rev=origin/main
```

- [x] **Step 4: Run target validation on disposable/maintenance hosts.**

Ubuntu 24.04:

```bash
envctl run performance --dry-run
envctl run performance
envctl run performance
sudo sysctl vm.swappiness vm.vfs_cache_pressure net.core.somaxconn net.ipv4.tcp_max_syn_backlog fs.file-max
envctl doctor
```

The second apply must be a no-op except for reporting the already-managed sysctl file. The existing swapfile must remain untouched.

CachyOS:

```bash
envctl run performance --dry-run
envctl run performance
cat /proc/swaps
zramctl
envctl doctor
```

No `/dev/zram1`, no second swapfile, and no changed existing zram settings may be produced.

- [x] **Step 5: Review the diff and commit atomically.**

```bash
git status --short
git diff --check
git diff --stat
git diff
git add <task-files>
git commit -m "feat(linux): add OS-specific performance baseline"
```

Do not create a tag or release manually.

## Self-review checklist

- [x] Ubuntu 24.04+ and CachyOS never load each other's performance manifest.
- [x] Ubuntu 22.04/Debian do not receive the Ubuntu 24.04 toolbox.
- [x] `run all` does not invoke sysctl or performance-only package provisioning.
- [x] Existing Arch/CachyOS gaming behavior remains unchanged.
- [x] Swapfiles are never created, resized, removed, or repurposed.
- [x] Sysctl application is explicit, argv-based, backed up, and idempotent.
- [x] Journald, governor, scheduler, unrelated service, and kernel settings are read-only in this branch; only zram lifecycle may start.
- [x] Doctor tests do not make optional performance state a warning.
- [x] No skill or manifest contains hostnames, IPs, UUIDs, secrets, or machine-specific paths.
- [x] `go build`, `go vet`, `go test`, lint, and target-host validation have fresh output.
