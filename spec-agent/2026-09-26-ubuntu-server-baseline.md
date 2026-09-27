# Ubuntu Server Baseline & Automatic Optimization — Implementation Plan

> **For agentic workers:** implement task-by-task with a fresh test checkpoint each time. The target is exactly Ubuntu Server `VERSION_ID >= 24.04`. Do not port any of this to Debian, Arch, CachyOS, Windows, WSL or macOS.

**Goal:** turn `run performance` from "install zram + write 5 sysctls" into a fully declarative, hardware-detected server baseline that converges any Ubuntu Server 24+ host — including greenfield cloud instances — to a known-good state for the 1/4/8/16 GB VPS fleet the owner operates.

**Architecture:** one profile identity (`ubuntu-server`) with the minimum version moved from a Go constant into the manifest. A new read-only `HardwareProbe` port feeds pure tier resolution; the manifest declares unconditional sysctls, per-tier sysctl overrides, swap/zram/limits/journald/timezone/debloat policy, and the code derives the values that genuinely depend on runtime state (notably `vm.swappiness` from detected swap topology). Six new narrow ports, each with an argv-based, atomically-installed drop-in adapter that honors dry-run, keeps a timestamped backup, and never resizes or destroys pre-existing state it did not create.

**Tech Stack:** Go 1.26, Cobra, YAML manifests, `//go:embed` fallback, `sudo -n` non-interactive elevation, argv-only commands, `/proc` + `/sys` + `statfs` probes, systemd (drop-ins, `daemon-reexec`).

**Spec:** This document is the design contract for the branch.

## Evidence base

Every value and every risk below was verified against a live host or an authoritative
document, not inferred. Host of record: `vps_oracle_2` (Ubuntu 26.04, kernel
`7.0.0-1009-oracle`, 951 MiB, 2 vCPU, ext4 45G, `sudo -n` available, `envctl` absent).

| Fact | Source |
| --- | --- |
| Fleet is Ubuntu 26.04 (×2) and 24.04 (×1), all ext4, all `Etc/UTC`, all NTP-synced | `ssh vps_oracle_2`, `vps_oracle_1`, `zscan_chatbot` |
| `zscan_chatbot`: 15.4 GiB RAM, **no swap at all**, journald uncapped, `nofile` soft 1024 | idem |
| `vps_oracle_1`: **681 MiB of swap in use** on 951 MiB RAM → real swap pressure | idem |
| journald default cap is 10% of the filesystem → ~4.5 G on 45 G, ~30 G on 309 G | `man 5 journald.conf`; measured 65 M / 127 M / 105 M uncapped |
| `SystemMaxUse` and `SystemKeepFree` are both honored; journald uses the **smaller** | `man 5 journald.conf` |
| `systemctl restart systemd-journald` preserves client streams; `stop` is **not** recommended | `man 8 systemd-journald` |
| `/etc/systemd/journald.conf.d/*.conf` and `system.conf.d/` are ordered **lexicographically by filename**, last-wins for single-value keys | `man 5 journald.conf` |
| `DefaultLimitNOFILE=` defaults to `1024:524288`; only the **soft** side is low | `man 5 systemd-system.conf`; `ulimit -n`=1024 on 3/3 hosts |
| `LimitNOFILE=`: *"Do not use. Be careful when raising the soft limit above 1024, since `select(2)` cannot function with file descriptors ≥ 1024"* | `man 5 systemd.exec` |
| `DefaultLimitNOFILE=65536:` parses clean and **preserves the host hard limit**; `65536:524288` would lower the AWS host (1048576) | probe on `vps_oracle_2`, drop-in discarded, no `daemon-reexec`, host restored |
| `daemon-reload` only reruns generators and reloads units; `daemon-reexec` *"of little use except for debugging and package upgrades"*, and *"all sockets systemd listening … will stay accessible"* | `man 1 systemctl` |
| For in-memory swap (zram/zswap), `vm.swappiness` **beyond 100** is appropriate; default 60; range 0–200 | `admin-guide/sysctl/vm.html` |
| `vfs_cache_pressure`: below `vfs_cache_pressure_denom` (100) the kernel retains dentry/inode cache; **0 risks OOM**; far above 100 hurts | idem |
| btrfs swapfile needs single-device, single data profile, NODATACOW, preallocated; `btrfs filesystem mkswapfile` since 6.1; *"especially discouraged for root filesystem"* | `btrfs.readthedocs.io/en/latest/Swapfile.html` |
| `MemTotal` is never the nominal size: 974092 kB = 951.26 MiB (OCI 1 G), 16162396 kB = 15.41 GiB (AWS 16 G) | measured |
| `needrestart 3.11-1ubuntu2` installed with no `/etc/default/needrestart` override → automatic mode | `vps_oracle_2` |
| `rpcbind` has an installed reverse-dependency (`nfs-common`) | `apt-cache rdepends --installed rpcbind` |
| Of the VPS-init debloat list only `modemmanager` and `rpcbind` exist on a cloud image; `fwupd`/`udisks2` are installed on 3/3 and are absent from that list | measured on 3 hosts |
| `vps_oracle_2` has `/var/run/reboot-required` | measured |
| `zscan_proxy_prod` fails authentication → 9/10 of the fleet is verifiable | `ssh-manager` |

## Non-goals

- Debian, Arch, CachyOS, Windows, WSL, macOS, Termux. The CachyOS profile is unchanged.
- CPU governor, I/O scheduler, mitigations, kernel cmdline, `crashkernel`/`kdump`. The
  2026-09-24 spec bars these behind a workload benchmark and per-case approval; that bar stands.
- Hibernation. No VPS hibernates, so no `resume=`/`resume_offset` and no
  swapfile-larger-than-RAM rule.
- A new agent skill. Commit `d5a2db0` cut the catalog 50→12 and deliberately moved
  envctl product knowledge out of the agent tier. Operational knowledge goes to
  `docs/guides/`.
- Container orchestration, Docker, reverse proxy, backup of data, `cloudflared`. Unchanged.

## Global Constraints

- Profile identity is `ubuntu-server`; the minimum version `24.04` lives in the manifest
  as `min_distro_version`, never in a Go constant.
- `run all` on a Linux host that is not Ubuntu Server `>= 24.04` must **fail with a
  non-zero exit and the exact reason**. It must not warn and continue.
- Every value the manifest does not explicitly pin is **detected**, never guessed, and the
  detection is reported in the diagnostic detail.
- Pre-existing state the tool did not create is **adopted, never resized, never rewritten**.
  A foreign swapfile, a foreign journald limit and a foreign `DefaultLimitNOFILE` are read,
  reported, and left alone unless they violate the declared policy.
- No privileged mutation happens in `--dry-run`, and dry-run performs no writes at all.
- The sysctl adapter gains a `min` policy so envctl never lowers a value the host already
  sets better (`fs.file-max` is `9223372036854775807` on both clouds; the current manifest
  would write `2097152`).
- The only PID 1 mutation in this branch is `systemctl daemon-reexec`, gated behind a flag
  that defaults to enabled and can be disabled with `--no-daemon-reexec`.
- `needrestart` is forced to list-only **before** any package removal is attempted.
- Tier boundaries for the 4 GB and 8 GB classes have **no hardware in the fleet to validate
  them**. They are derived, not measured, and the manifest says so in `rationale`.
- No hostnames, IPs, UUIDs, filesystem identifiers or machine-specific paths enter the repo.
- The whole branch targets one profile; no other manifest, command or OS behavior changes
  except the documented breaking changes below.

---

### Task 1: Move the profile minimum out of the Go constant

**Files:**
- Modify: `internal/domain/entity/performance.go`
- Modify: `internal/domain/entity/platform.go` (reuse `compareDistroVersions`; export a wrapper)
- Create: `internal/domain/entity/performance_test.go` if absent
- Modify: `internal/domain/repository/interfaces.go`
- Modify: `internal/infra/embedded/manifest_repo.go`
- Modify: `internal/infra/embedded/manifest_repo_test.go` (2 hard-coded `"ubuntu-24.04"` literals, lines 194 and 198 on `origin/main`; the in-flight `feat/debloat-tier2-expansion` branch shifts them to 219/223)
- Modify: `manifests/performance_ubuntu.yaml:1-3` (header claims *"selected only on exact
  Ubuntu >= 24.04 hosts"*, which is false today — the gate is a minimum, not an exact match)
- Modify: `internal/usecase/provision_performance.go`
- Modify: `internal/usecase/provision_performance_test.go`
- Modify: `internal/ui/cli/performance.go`
- Modify: `internal/ui/cli/run.go`

**Interfaces:**

```go
// entity/performance.go
const PerformanceProfileUbuntuServer PerformanceProfile = "ubuntu-server"

type PerformanceProfileMeta struct {
    Profile          PerformanceProfile `yaml:"profile"`
    MinDistroVersion string             `yaml:"min_distro_version,omitempty"`
    ManifestFile     string             `yaml:"-"`
}

// entity/platform.go
func MatchesDistroMinimum(versionID, min string) bool

// repository/interfaces.go — add to ManifestRepository
ListPerformanceProfiles() ([]entity.PerformanceProfileMeta, error)

// entity/performance.go — PerformanceSpec gains
MinDistroVersion string `yaml:"min_distro_version,omitempty"`
```

- [ ] **Step 1: Write failing tests.**

Table cases for `MatchesDistroMinimum`: `("26.04","24.04")→true`, `("24.04","24.04")→true`,
`("24.10","24.04")→true`, `("22.04","24.04")→false`, `("24.04","")→true` (no minimum),
`("rolling","24.04")→false`, `("","24.04")→false`. Add a repo test asserting
`performance_ubuntu.yaml` declares `profile: ubuntu-server` and
`min_distro_version: "24.04"`, and that no manifest declares the literal `ubuntu-24.04`.

- [ ] **Step 2: Verify RED.**

```bash
go test ./internal/domain/entity -run 'TestMatchesDistroMinimum|TestPerformanceProfile' -count=1
go test ./internal/infra/embedded -run TestLoadManifestsFromDiskOrEmbed -count=1
```

Expected: FAIL — the constant is `ubuntu-24.04`, `MatchesDistroMinimum` and
`ListPerformanceProfiles` do not exist.

- [ ] **Step 3: Implement the minimum.**

Rename the constant and its value; add `MinDistroVersion` to `PerformanceSpec`;
`MatchesDistroMinimum` wraps the existing dotted comparator and treats an empty
`min` as "any version". `ListPerformanceProfiles` maps profile → filename, reads each
manifest header, and returns the metas. `PerformanceProfileMatchesPlatform` keeps working
for the CachyOS exact-ID case and is no longer the gate for Ubuntu; the new gate is
`MatchesDistroMinimum(platform.VersionID, spec.MinDistroVersion)` evaluated **after** the
spec is loaded, so the CLI and the use case share one decision.

- [ ] **Step 4: Replace the CLI gate.**

`selectPerformanceProfile` stops hard-coding the 24.04 string. It calls
`ListPerformanceProfiles`, returns the first profile whose `MinDistroVersion` matches, and
on no match returns an error naming the host and the required minimum. `runVPSProfile`
treats that error as fatal: log the reason and return non-zero instead of
`pterm.Warning.Printf("Performance profile skipped")`.

- [ ] **Step 5: Verify GREEN.**

```bash
go build ./... && go vet ./...
go test ./internal/domain/entity ./internal/infra/embedded ./internal/usecase ./internal/ui/cli -count=1
grep -rn 'ubuntu-24\.04' --include=*.go --include=*.yaml . | grep -v '\.worktrees/'
```

Expected: PASS, and `grep` prints nothing.

**Rollback:** `git revert` the commit. No host state touched.

**Result:** `envctl run performance` on Ubuntu 26.04 selects `ubuntu-server`; on Debian 12
it exits non-zero naming `min_distro_version 24.04`.

---

### Task 2: Add the read-only hardware probe

**Files:**
- Create: `internal/infra/performance/hardware_probe.go`
- Create: `internal/infra/performance/hardware_probe_test.go`
- Modify: `internal/domain/repository/performance.go`
- Modify: `internal/domain/entity/performance.go`
- Modify: `internal/infra/performance/inspector.go` (extract the shared `/proc/swaps` parser)

**Interfaces:**

```go
// entity/performance.go
type HardwareState struct {
    MemTotalKB     uint64
    CPUCount       int
    RootFSType     string
    DiskFreeBytes  uint64
    Swap           []SwapDevice
    HasZRAM        bool
    ZRAMPriority   int
    DiskSwapTopPri int // -1 when no disk swap; the highest priority among non-zram devices
}

func (h HardwareState) MemTotalMiB() uint64
func (h HardwareState) HasDiskSwap() bool

// repository/performance.go
type HardwareProbe interface {
    Snapshot(ctx context.Context) entity.HardwareState
}
```

- [ ] **Step 1: Write failing parser tests with fixtures.**

Fixture `/proc/meminfo` from the real hosts (`MemTotal:       974092 kB` and
`MemTotal:       16162396 kB`); fixture `/proc/swaps` with a zram row at priority 100, a
disk row at -1, and an empty file. Assert `MemTotalMiB()` yields **951** and **15783**
respectively — the assertion is the unit-trap guard, and the test comment must name the two
hosts and note that 15783 MiB is what a nominal "16 GB" AWS instance actually reports.

- [ ] **Step 2: Verify RED.**

```bash
go test ./internal/infra/performance -run TestHardwareProbe -count=1
```

Expected: FAIL — the package does not exist.

- [ ] **Step 3: Implement the minimal probe.**

Parse `/proc/meminfo` `MemTotal`; count CPUs from the affinity mask via
`runtime.NumCPU()`; get the root filesystem type and free bytes with
`golang.org/x/sys/unix.Statfs` on `/` plus a `statfs.f_type` → name map, falling back to
`findmnt -no FSTYPE /` and `df --output=avail -B1 /` when the constant map is unknown.
`DiskSwapTopPri` is the maximum priority among swap devices whose name is not
`/dev/zram*`. Missing files yield zero values, never errors.

If `golang.org/x/sys` is not already a dependency, do **not** add it in this task: use
`syscall.Statfs` from the standard library and record the FSType-name mapping as a small
explicit table with a `findmnt` fallback for anything unmapped.

- [ ] **Step 4: Verify GREEN.**

```bash
go build ./... && go vet ./...
go test ./internal/infra/performance -count=1
```

**Rollback:** `git revert`. Read-only; no host state touched.

**Result:** the probe returns the measured values above for the three known hosts, proven
against fixtures in unit tests.

---

### Task 3: Add declarative RAM tiers and pure resolution

**Files:**
- Create: `internal/domain/entity/tiers.go`
- Create: `internal/domain/entity/tiers_test.go`
- Modify: `internal/domain/entity/performance.go` (`PerformanceSpec.Tiers`)
- Modify: `manifests/performance_ubuntu.yaml`

**Interfaces:**

```go
type PerformanceTier struct {
    ID               string          `yaml:"id"`
    MatchMemTotalMax int             `yaml:"mem_total_max_mib,omitempty"` // 0 = unbounded
    Sysctls          []SysctlSetting `yaml:"sysctls,omitempty"`
    EnableZRAM       *bool           `yaml:"zram,omitempty"`
    SwapSizeMax      string          `yaml:"swap_size_max,omitempty"`
    Rationale        string          `yaml:"rationale"`
}

func SelectPerformanceTier(hw entity.HardwareState, tiers []entity.PerformanceTier) (entity.PerformanceTier, error)
```

- [ ] **Step 1: Write failing resolution tests.**

Cases, with the host named in each comment: `974092 kB` → `tiny`; `16162396 kB` → `large`;
a synthetic `4194304 kB` (4 GiB) → `small`; a synthetic `8388608 kB` (8 GiB) → `medium`;
a tier list where two bands overlap → error, not first-match; an empty tier list → error;
`MemTotalKB == 0` → error, never `tiny` by accident.

- [ ] **Step 2: Verify RED.**

```bash
go test ./internal/domain/entity -run TestSelectPerformanceTier -count=1
```

Expected: FAIL — `SelectPerformanceTier` does not exist.

- [ ] **Step 3: Implement and declare the tiers.**

`SelectPerformanceTier` requires strictly descending `MatchMemTotalMax`, errors on
overlap, and errors on zero memory. The manifest declares four tiers with the bands
chosen to absorb the `MemTotal` shortfall measured on both clouds:

| id | `mem_total_max_mib` | validated by | `zram` | `vfs_cache_pressure` |
| --- | --- | --- | --- | --- |
| `tiny` | 1536 | ✅ `vps_oracle_2` / `vps_oracle_1` (951 MiB) | on | 50 |
| `small` | 4608 | ⚠️ derived, **no host** | on | 75 |
| `medium` | 9216 | ⚠️ derived, **no host** | off | 100 |
| `large` | unbounded | ✅ `zscan_chatbot` (15.4 GiB) | off | 100 |

Every tier's `rationale` must state in prose whether it is hardware-validated or derived,
and the two derived rows must say so in the manifest text, not only in this spec.

- [ ] **Step 4: Verify GREEN.**

```bash
go build ./... && go test ./internal/domain/entity -count=1
```

**Rollback:** `git revert`.

**Result:** `SelectPerformanceTier` maps both measured hosts to the intended tier and
refuses ambiguous tier lists.

---

### Task 4: Extract the shared drop-in writer and add the sysctl `min` policy

**Files:**
- Create: `internal/infra/performance/dropin.go`
- Create: `internal/infra/performance/dropin_test.go`
- Modify: `internal/infra/performance/sysctl_manager.go` (delegate to the writer)
- Modify: `internal/infra/performance/sysctl_manager_test.go`
- Modify: `internal/domain/entity/performance.go` (`SysctlSetting.Policy`)
- Modify: `manifests/performance_ubuntu.yaml` (`fs.file-max` → `policy: min`)

**Interfaces:**

```go
// internal/infra/performance/dropin.go
type DropinWriter struct { /* destination, run, now, elevate */ }
func (w *DropinWriter) Install(content string, mode os.FileMode) (changed bool, backup string, err error)

type SysctlPolicy string
const (
    SysctlPolicySet  SysctlPolicy = ""    // always write the declared value
    SysctlPolicyMin  SysctlPolicy = "min" // write only when the effective value is lower
    SysctlPolicyMax  SysctlPolicy = "max" // write only when the effective value is higher
)

// entity.SysctlSetting gains
Policy SysctlPolicy `yaml:"policy,omitempty"`
```

- [ ] **Step 1: Write failing tests.**

Writer: byte-identical content is a no-op with no backup; differing content produces
`<path>.bak.YYYYMDDD-HHMMSS`; a same-second collision gets the `-1` suffix; the temp file
is created in the destination directory so the rename is atomic; a failed `mv` leaves the
original intact and cleans the temp. Policy: `min` with effective `9223372036854775807` and
desired `2097152` → no write, `DiagOK` "host value already at or above target";
`min` with effective `4096` and desired `65535` → write; `min` on an unreadable
`/proc/sys` → `DiagError` with the key, never a silent success.

- [ ] **Step 2: Verify RED.**

```bash
go test ./internal/infra/performance -run 'TestDropin|TestSysctl' -count=1
```

Expected: FAIL — the writer and the policy do not exist.

- [ ] **Step 3: Implement, then delete the duplication.**

`Install` is the exact sequence already proven in `sysctl_manager.go:90-165` — backup with
`cp -a`, `install` a temp file **in the destination directory**, `mv -f` into place — lifted
verbatim so behavior does not drift. `sysctlManager.Apply` then renders, calls `Install`,
and applies with `sysctl -p <file>`. Add the policy check by reading the current value from
`/proc/sys/<key with . → />` before deciding to write.

- [ ] **Step 4: Fix the regression in the manifest.**

`fs.file-max` becomes `policy: min` with a rationale naming the measured host value
`9223372036854775807`. `net.core.somaxconn`, `net.ipv4.tcp_max_syn_backlog`,
`net.core.netdev_max_backlog` and `net.ipv4.tcp_fin_timeout` stay `policy: set` because the
measured hosts are below the target and must be corrected.

- [ ] **Step 5: Verify GREEN.**

```bash
go build ./... && go vet ./...
go test ./internal/infra/performance ./internal/usecase -run 'Test.*(Performance|Sysctl|Dropin)' -count=1
```

**Rollback:** the previous drop-in is at `/etc/sysctl.d/90-envctl-performance.conf.bak.<stamp>`.
To revert a host: restore that file and `sysctl -p` it.

**Result:** `run performance` twice on `vps_oracle_2` leaves `fs.file-max` at
`9223372036854775807` and still raises `somaxconn` to 65535.

---

### Task 5: Add timezone verification and enforcement

**Files:**
- Create: `internal/infra/performance/timezone_manager.go`
- Create: `internal/infra/performance/timezone_manager_test.go`
- Modify: `internal/domain/repository/performance.go`
- Modify: `internal/domain/entity/performance.go`
- Modify: `internal/infra/embedded/manifest_repo.go`
- Modify: `manifests/performance_ubuntu.yaml`
- Modify: `internal/usecase/provision_performance.go`

**Interfaces:**

```go
type TimezoneSpec struct {
    Mode     string `yaml:"mode"`     // "verify" (default) | "enforce"
    Expected string `yaml:"expected"` // IANA name
}

type TimezoneManager interface {
    Current(ctx context.Context) (string, error)
    Apply(ctx context.Context, spec entity.TimezoneSpec, dryRun bool) ([]entity.Diagnostic, error)
}
```

- [ ] **Step 1: Write failing tests.**

`verify` mode with `Current() == Expected` → `DiagOK`, no command issued;
`verify` mode with a mismatch → `DiagInfo` naming both zones and stating that no change was
made; `enforce` mode with a mismatch → `timedatectl set-timezone <Expected>` issued and
`DiagOK`; an invalid IANA name rejected at manifest load, not at apply; `timedatectl`
missing → `DiagError` naming the binary, never a false success; dry-run issues no command.

- [ ] **Step 2: Verify RED.**

```bash
go test ./internal/infra/performance -run TestTimezone -count=1
```

Expected: FAIL — the package does not exist.

- [ ] **Step 3: Implement.**

`Current` reads `timedatectl show -p Timezone --value` with a
`/etc/timezone` file fallback. `Apply` in `verify` mode never writes. In `enforce` mode it
validates the name against `/usr/share/zoneinfo/`, then calls
`timedatectl set-timezone` via the shared `command` helper. Add `tzdata` to the manifest
package list.

- [ ] **Step 4: Set the manifest default from the measured fleet.**

`timezone: { mode: verify, expected: Etc/UTC }`, with the rationale stating that all three
reachable hosts are `Etc/UTC` with NTP active and that `enforce` is opt-in per host.
`enforce` is reachable only through an explicit CLI flag, never by default.

- [ ] **Step 5: Verify GREEN.**

```bash
go build ./... && go vet ./...
go test ./internal/infra/performance ./internal/usecase -run 'Test.*(Timezone|Performance)' -count=1
```

**Rollback:** `sudo timedatectl set-timezone <previous>`. Nothing is written in the default
mode, so the default path has no rollback to perform.

**Result:** `envctl run performance` on `vps_oracle_2` reports `Timezone / Etc/UTC` as
`OK` and changes nothing.

---

### Task 6: Cap journald with a keep-free floor

**Files:**
- Create: `internal/infra/performance/journald_manager.go`
- Create: `internal/infra/performance/journald_manager_test.go`
- Modify: `internal/infra/performance/inspector.go` (export the shared parse)
- Modify: `internal/domain/repository/performance.go`
- Modify: `internal/domain/entity/performance.go`
- Modify: `internal/infra/embedded/manifest_repo.go`
- Modify: `manifests/performance_ubuntu.yaml`
- Modify: `internal/usecase/provision_performance.go`
- Modify: `internal/usecase/doctor_linux_performance.go`

**Interfaces:**

```go
type JournaldSetting struct {
    Key   string `yaml:"key"`
    Value string `yaml:"value"`
}
type JournaldSpec struct {
    Dropin string            `yaml:"dropin"`
    Values []JournaldSetting `yaml:"values"`
}
type JournaldManager interface {
    Apply(ctx context.Context, spec entity.JournaldSpec, dryRun bool) ([]entity.Diagnostic, error)
}
```

- [ ] **Step 1: Write failing tests.**

Render `[Journal]` sorted by key. Assert the rendered file contains
`SystemMaxUse`, `SystemKeepFree`, `RuntimeMaxUse` and `MaxRetentionSec`. Assert
byte-identical content is a no-op. Assert the apply sequence is exactly
`install <temp> <dropin>` → `mv` → `systemctl restart systemd-journald` and that
`systemctl stop` is **never** issued. Assert an unknown key is rejected at load.
Assert the doctor check compares against the **effective** limit, so a host whose
`SystemKeepFree` is unset while `SystemMaxUse` is set is reported `OK`, and a host with
neither is reported `INFO` with the 10%-of-filesystem figure as the concrete risk.

- [ ] **Step 2: Verify RED.**

```bash
go test ./internal/infra/performance ./internal/usecase -run 'TestJournald' -count=1
```

Expected: FAIL — the package does not exist.

- [ ] **Step 3: Implement.**

Reuse `DropinWriter`. Emit a `90-` prefixed drop-in in
`/etc/systemd/journald.conf.d/`, which the man page confirms is sorted lexicographically
with last-wins for single-value keys. Restart with `systemctl restart`, never
`stop`+`start`. Reject any key outside a fixed allowlist of
`SystemMaxUse SystemKeepFree SystemKeepFree SystemMaxFileSize SystemMaxFiles
RuntimeMaxUse RuntimeKeepFree RuntimeMaxFileSize RuntimeMaxFiles MaxRetentionSec
Storage Compress`.

- [ ] **Step 4: Set the manifest values from the measured fleet.**

`SystemMaxUse: 200M`, `SystemKeepFree: 1G`, `RuntimeMaxUse: 50M`,
`MaxRetentionSec: 2week`. Rationale records: 3/3 hosts uncapped; default cap is 10% of the
filesystem = 4.5 G on the 45 G OCI disks; measured usage 65 M / 127 M / 105 M;
`SystemKeepFree` is mandatory because journald honors the **smaller** of the two limits,
and the 1 GB tier cannot afford 4.5 G of journal.

- [ ] **Step 5: Verify GREEN.**

```bash
go build ./... && go vet ./...
go test ./internal/infra/performance ./internal/usecase -run 'Test.*(Journald|Performance|Doctor)' -count=1
```

**Rollback:** restore `/etc/systemd/journald.conf.d/90-envctl-journald.conf.bak.<stamp>` (or
delete it if there was no prior drop-in) and `sudo systemctl restart systemd-journald`.
Verified safe by `man 8 systemd-journald`: the restart preserves client streams.

**Result:** on `vps_oracle_2`, `systemd-analyze cat-config systemd/journald.conf` shows
`SystemMaxUse=200M` and `SystemKeepFree=1G`, and `envctl doctor` stays at 0 WARN / 0 ERROR.

---

### Task 7: Raise the soft file-descriptor limit without touching the host hard limit

**Files:**
- Create: `internal/infra/performance/limits_manager.go`
- Create: `internal/infra/performance/limits_manager_test.go`
- Modify: `internal/domain/repository/performance.go`
- Modify: `internal/domain/entity/performance.go`
- Modify: `internal/infra/embedded/manifest_repo.go`
- Modify: `manifests/performance_ubuntu.yaml`
- Modify: `internal/usecase/provision_performance.go`

**Interfaces:**

```go
type LimitsSpec struct {
    SystemDropin string `yaml:"system_dropin"`
    PAMDropin    string `yaml:"pam_dropin"`
    NofileSoft   int    `yaml:"nofile_soft"`
    NofileHard   string `yaml:"nofile_hard"` // "" = preserve the host value
    Nproc        int    `yaml:"nproc"`
    Reexec       bool   `yaml:"daemon_reexec"`
}
type ResourceLimitsManager interface {
    Apply(ctx context.Context, spec entity.LimitsSpec, dryRun bool) ([]entity.Diagnostic, error)
}
```

- [ ] **Step 1: Write failing tests.**

Assert the rendered systemd drop-in is exactly
`[Manager]\nDefaultLimitNOFILE=65536:\nDefaultLimitNPROC=32768\n` — in particular the
**trailing colon** with an empty hard side, and assert a test comment records that
`65536:524288` was rejected because it would lower the measured AWS host's 1048576.
Assert the PAM drop-in renders `* soft nofile 65536` and `* hard nofile` is **absent**,
so PAM cannot lower the inherited hard limit. Assert the apply sequence ends in
`systemctl daemon-reexec` when `Reexec` is true and in nothing but
`systemctl daemon-reload` when false. Assert `--no-daemon-reexec` sets `Reexec: false`.
Assert the diagnostic detail quotes the `select(2)` caveat when `NofileSoft > 1024`.

- [ ] **Step 2: Verify RED.**

```bash
go test ./internal/infra/performance -run TestResourceLimits -count=1
```

Expected: FAIL — the package does not exist.

- [ ] **Step 3: Implement.**

Two `DropinWriter` installs: `/etc/systemd/system.conf.d/90-envctl-limits.conf` and
`/etc/security/limits.d/90-envctl-limits.conf`. `DefaultLimitNOFILE` is rendered as
`{soft}:` when `NofileHard` is empty, which the probe on `vps_oracle_2` proved parses clean
and preserves the host's hard limit. When `Reexec` is set, run `systemctl daemon-reexec`
only when a drop-in actually changed; `man 1 systemctl` documents that all sockets stay
accessible during the reexec, which is what makes it acceptable over SSH.

- [ ] **Step 4: Set the manifest values from the measured fleet.**

`nofile_soft: 65536`, `nofile_hard: ""`, `nproc: 32768`, `daemon_reexec: true`.
Rationale records: soft limit is 1024 on 3/3 hosts; hard is 524288 (OCI) and 1048576
(AWS), so it must not be pinned; `man 5 systemd.exec` warns that a soft limit above 1024
breaks `select(2)`, and the diagnostic says so on every apply.

- [ ] **Step 5: Verify GREEN.**

```bash
go build ./... && go vet ./...
go test ./internal/infra/performance ./internal/usecase -run 'Test.*(Limits|Performance)' -count=1
```

**Rollback:** delete both drop-ins (or restore the `.bak.<stamp>` copies) and run
`sudo systemctl daemon-reexec`. This is the only task that re-executes PID 1.

**Result:** on `vps_oracle_2`, a freshly started unit reports `LimitNOFILE` soft 65536 and
hard 524288, and a new SSH session shows `ulimit -n 65536` / `ulimit -Hn 524288`.

---

### Task 8: Make zram tier-aware and derive swappiness from swap topology

**Files:**
- Modify: `internal/infra/performance/zram_manager.go`
- Modify: `internal/infra/performance/zram_manager_test.go`
- Modify: `internal/usecase/provision_performance.go`
- Modify: `internal/usecase/provision_performance_test.go`
- Modify: `manifests/performance_ubuntu.yaml`
- Modify: `internal/usecase/doctor_linux_performance.go`

**Interfaces:**

```go
type ZRAMSpec struct {
    Policy string `yaml:"policy"` // "tier" | "always" | "disabled"
}

func DeriveSwappiness(hw entity.HardwareState) (value int, rationale string)
```

- [ ] **Step 1: Write failing tests.**

`DeriveSwappiness`: zram active → 150; disk swap only, no zram → 10; nothing →
10 with a rationale that a disk swapfile is about to be created. Assert the returned
rationale for the zram case quotes the kernel doc sentence about in-memory swap values
beyond 100. Assert `vm.swappiness` is **never** read from the manifest by loading a spec
that declares it and failing the test if the manifest value is used.

Zram gating: `tiny`/`small` with zram absent → the lifecycle runs; `medium`/`large` →
`DiagInfo` "zram not enabled at this tier" and **no** `modprobe`; a second pre-existing
`zram*` device → `DiagInfo` refusal, preserving `zram_manager.go:80-87`. Assert a host
with an active disk swap at priority ≥ the zram priority reports a conflict.

- [ ] **Step 2: Verify RED.**

```bash
go test ./internal/infra/performance ./internal/usecase -run 'Test.*(ZRAM|Swappiness|Performance)' -count=1
```

Expected: FAIL — `DeriveSwappiness` and the tier gate do not exist.

- [ ] **Step 3: Implement.**

`DeriveSwappiness` is a pure function in the performance package. The use case appends the
derived `vm.swappiness` with `policy: set` **after** the manifest and tier sysctls, so it
always wins, and the diagnostic records the topology that produced it. Zram runs when the
tier enables it and the device is absent. The existing refusal of a second zram device is
kept verbatim.

- [ ] **Step 4: Remove the contradiction from the manifest.**

`vm.swappiness` is **deleted** from `performance_ubuntu.yaml` — it is derived, and pinning
it in the manifest is what produced the original bug of shipping zram together with
`swappiness=10`. The header comment must state that the value comes from swap topology,
not from the tier.

- [ ] **Step 5: Verify GREEN.**

```bash
go build ./... && go vet ./...
go test ./internal/infra/performance ./internal/usecase -run 'Test.*(ZRAM|Swappiness|Performance)' -count=1
```

**Rollback:** `sudo swapoff /dev/zram0 && sudo systemctl disable --now dev-zram0.swap`; the
`swappiness` value reverts with the previous drop-in backup.

**Result:** on `vps_oracle_2` (tier `tiny`, no zram) the run installs and starts zram and
sets `vm.swappiness=150`; on `zscan_chatbot` (tier `large`) it reports zram as
deliberately skipped and sets `vm.swappiness=10` after the swapfile task creates a disk swap.

---

### Task 9: Adopt or create the swapfile

**Files:**
- Create: `internal/infra/performance/swapfile_manager.go`
- Create: `internal/infra/performance/swapfile_manager_test.go`
- Modify: `internal/domain/repository/performance.go`
- Modify: `internal/domain/entity/performance.go`
- Modify: `internal/infra/embedded/manifest_repo.go`
- Modify: `manifests/performance_ubuntu.yaml`
- Modify: `internal/usecase/provision_performance.go`
- Modify: `internal/usecase/doctor_linux_performance.go`

**Interfaces:**

```go
type SwapSpec struct {
    Policy      string   `yaml:"policy"`      // "auto" | "disabled"
    File        string   `yaml:"file"`
    Priority    int      `yaml:"priority"`
    SizeOf      string   `yaml:"size_of"`     // "mem_total"
    SizeMin     string   `yaml:"size_min"`
    SizeMax     string   `yaml:"size_max"`
    DiskReserve string   `yaml:"disk_reserve"`
    FSAllow     []string `yaml:"fs_allow"`    // ext4, xfs
    FSBtrfs     string   `yaml:"fs_btrfs"`    // "mkswapfile" | "refuse"
    FSDeny      []string `yaml:"fs_deny"`     // zfs, overlayfs, ...
}

type SwapManager interface {
    Ensure(ctx context.Context, spec entity.SwapSpec, hw entity.HardwareState, dryRun bool) ([]entity.Diagnostic, error)
}

func ResolveSwapSizeBytes(spec entity.SwapSpec, hw entity.HardwareState) (uint64, error)
```

- [ ] **Step 1: Write failing tests.**

`ResolveSwapSizeBytes`: 951 MiB RAM with 33 G free → clamped up to `SizeMin`; 15.4 GiB RAM
with 275 G free → clamped down to `SizeMax`; a host with `SizeMax` bytes of free space →
`DiagError`, never a value larger than `DiskReserve` allows. Each test comment names the
host the numbers came from.

Adoption: a host that already has a disk swap → `DiagOK` naming path, size and priority,
**no** `fallocate`, **no** `mkswap`, **no** `swapon`, **no** fstab write. This is the
`vps_oracle_2` case and it must be a pure no-op on the host.

Creation, ext4: `fallocate -l <size> <file>` → `chmod 600` → `mkswap` → `swapon` → fstab
entry `/swapfile.envctl none swap sw,pri=-2 0 0`, installed atomically through
`DropinWriter` against `/etc/fstab` with a `.bak.<stamp>` first. Assert the fstab write is
skipped entirely when an equivalent entry already exists.

Creation, btrfs: assert the sequence is `truncate -s 0` → `chtr +C` → `fallocate` →
`chmod 600` → `mkswap`, and that a btrfs host with `fs_btrfs: refuse` returns
`DiagInfo` without writing. Assert `FSDeny` filesystems always return `DiagInfo` with the
detected type named.

Refusals: an existing file at `File` that is **not** a swap → `DiagError`, never
`mkswap` over it. An existing swap whose priority is higher than the tier's zram priority →
`DiagError` naming both, because the ordering is wrong and silently fixing it would mask
a hand-tuned decision.

- [ ] **Step 2: Verify RED.**

```bash
go test ./internal/infra/performance -run 'TestSwap' -count=1
```

Expected: FAIL — the package does not exist.

- [ ] **Step 3: Implement.**

Read the existing swap table from the probe. If a disk swap exists, adopt and return. If
the filesystem is denied, report and return. Otherwise compute the size, create, activate
and install the fstab entry. The btrfs path uses the documented
`btrfs filesystem mkswapfile --size` when the binary exists and falls back to the manual
`truncate`+`chattr`+`fallocate` sequence otherwise.

- [ ] **Step 4: Set the manifest values from the measured fleet.**

`file: /swapfile.envctl` — deliberately **not** `/swapfile`, so a hand-created host
swapfile is never mistaken for ours. `size_of: mem_total`, `size_min: 1G`, `size_max: 8G`,
`disk_reserve: 5G`, `priority: -2`, `fs_allow: [ext4, xfs]`, `fs_btrfs: mkswapfile`,
`fs_deny: [zfs, overlayfs, tmpfs]`. Rationale records: 2/3 hosts already ship an 8 G OCI
swapfile at priority -1 and must be adopted untouched; `zscan_chatbot` has no swap at all;
`MemTotal` is never the nominal size, so the percentage is a base and the clamps are the
safety.

- [ ] **Step 5: Verify GREEN.**

```bash
go build ./... && go vet ./...
go test ./internal/infra/performance ./internal/usecase -run 'Test.*(Swap|Performance)' -count=1
```

**Rollback:** `sudo swapoff /swapfile.envctl && sudo rm /swapfile.envctl`, then restore
`/etc/fstab.bak.<stamp>`. Nothing to roll back on the two OCI hosts, which are adopted.

**Result:** `vps_oracle_2` and `vps_oracle_1` report an adopted 8 G swap and write nothing;
`zscan_chatbot` gains an 8 G swapfile at priority -2 plus one fstab line.

---

### Task 10: Add Linux debloat with a needrestart guard

**Files:**
- Create: `manifests/debloat_linux.yaml`
- Create: `internal/infra/performance/remover_apt.go`
- Create: `internal/infra/performance/remover_apt_test.go`
- Create: `internal/infra/performance/remover_pacman.go`
- Create: `internal/infra/performance/remover_pacman_test.go`
- Modify: `internal/domain/repository/interfaces.go`
- Modify: `internal/domain/entity/models.go`
- Modify: `internal/infra/embedded/manifest_repo.go`
- Modify: `internal/infra/embedded/manifest_repo_test.go`
- Modify: `internal/usecase/provision_performance.go`
- Modify: `internal/usecase/doctor_linux_performance.go`
- Modify: `manifests/performance_ubuntu.yaml`

**Interfaces:**

```go
// entity/models.go
type PackageRemoval struct {
    ID            string      `yaml:"id"`
    Package       string      `yaml:"package"`
    Type          PackageType `yaml:"type"`
    TargetDistro  string      `yaml:"target_distro,omitempty"`
    Category      string      `yaml:"category"`
    Rationale     string      `yaml:"rationale"`
    // ProtectedWhen lists packages whose presence makes removal unsafe;
    // removal is skipped with DiagInfo when any is installed.
    ProtectedWhen []string `yaml:"protected_when,omitempty"`
}
type DebloatSpec struct {
    NeedrestartDropin string           `yaml:"needrestart_dropin"`
    Removals          []PackageRemoval `yaml:"removals"`
}

// repository/interfaces.go — NEW narrow port; PackageManager is NOT extended
type PackageRemover interface {
    Type() entity.PackageType
    IsInstalled(ctx context.Context, pkg entity.Package) (bool, string, error)
    Remove(ctx context.Context, pkg entity.Package) error
}
```

- [ ] **Step 1: Write failing tests.**

Adapter: `apt-get purge -y --purge <pkg>` with `DEBIAN_FRONTEND=noninteractive`, argv-based,
`sudo -n`; a purge of an absent package is a no-op `DiagOK`, not an error; `pacman -R --noconfirm` for the Arch side.

Guard ordering: assert the needrestart drop-in is installed **before** the first removal by
recording the command sequence in a fake runner and asserting the drop-in write is index 0.
Assert `NEEDRESTART_MODE=l` is the exact content.

`ProtectedWhen`: with `nfs-common` installed, removing `rpcbind` returns `DiagInfo` naming
`nfs-common` and issues **no** purge. This is the measured `vps_oracle_2` case.

Idempotency: `fwupd`, `udisks2`, `modemmanager` are installed on 3/3 hosts, so removing
them must be attempted; `avahi-daemon`, `cups`, `bluez`, `bluetooth` are absent on all
measured cloud images, so each must be `DiagOK` "already absent" and must not invoke the
package manager.

Doctor: absent = `DiagOK`, present after a successful run = `DiagError`, present when the
removal was skipped by a guard = `DiagInfo` naming the guard. No state may produce a
`DiagWarning`, because that would break the 0 WARN contract.

- [ ] **Step 2: Verify RED.**

```bash
go test ./internal/infra/performance -run 'TestRemover|TestDebloat' -count=1
go test ./internal/infra/embedded -run TestLoadManifestsFromDiskOrEmbed -count=1
```

Expected: FAIL — the port, the adapters and the manifest do not exist.

- [ ] **Step 3: Implement.**

`PackageRemover` is a **separate** port. `PackageManager` has 8 implementations
(apt, pacman, paru, winget, volta, npm, pip, go) plus a mock in
`doctor_audit_test.go`; extending it would break 9 files and 4 of those have no removal
semantics. Only apt and pacman implement `PackageRemover`.

The use case installs the needrestart drop-in first, then walks the removals in manifest
order, checking `ProtectedWhen` before each one.

- [ ] **Step 4: Build the removal list from the measured fleet, not from the VPS-init list.**

Include, each with a rationale: `modemmanager` (installed 3/3, a desktop modem daemon on
a headless server), `fwupd` and `udisks2` (installed 3/3, desktop firmware/disk daemons,
absent from the VPS-init list), `iscsid` and `multipathd` (running on `vps_oracle_2`,
irrelevant to a single-disk VPS). **Exclude** `avahi-daemon`, `cups`, `bluez` and
`bluetooth` — absent on every measured cloud image, so they are noise, and `avahi-daemon`
carries an mDNS dependency the fleet does not need to reason about. Add
`ProtectedWhen: [nfs-common]` to `rpcbind`, which is not removed by default because the
reverse dependency was measured.

- [ ] **Step 5: Verify GREEN.**

```bash
go build ./... && go vet ./...
go test ./internal/infra/performance ./internal/usecase ./internal/infra/embedded -count=1
```

**Rollback:** reinstall with `sudo apt-get install -y <pkg>`. Purge is not reversible by
envctl itself, which is why it runs last in the branch order and why every entry is
individually revertible from the package cache.

**Result:** on `vps_oracle_2` the run removes `modemmanager`, `fwupd`, `udisks2`,
`iscsid` and `multipathd`, skips `rpcbind` with the `nfs-common` reason, and reports the
four absent packages as already clean.

---

### Task 11: Wire the CLI, the reboot precondition and the doctor sections

**Files:**
- Modify: `internal/ui/cli/performance.go`
- Modify: `internal/ui/cli/run.go`
- Modify: `internal/ui/cli/root.go`
- Modify: `internal/usecase/doctor_linux_performance.go`
- Modify: `internal/usecase/doctor_audit_test.go`

**Interfaces:**

```go
// new flags on `run performance`
--no-daemon-reexec   // skip the PID 1 re-exec
--timezone <IANA>    // enforce a timezone instead of verifying
--allow-debloat      // opt in to package removal (default: verify-only)
--debloat-only
```

- [ ] **Step 1: Write failing tests.**

Assert `--no-daemon-reexec` reaches the use case as `LimitsSpec.Reexec == false`; that
`--timezone` switches the spec `Mode` to `enforce`; that without `--allow-debloat` the
removal loop is replaced by a report-only pass producing the same diagnostic set; that a
host with `/var/run/reboot-required` emits one `DiagWarning` naming the file **before** any
performance write is attempted, and that the run aborts with a non-zero exit on that
warning unless `--force-reboot-pending` is passed.

- [ ] **Step 2: Verify RED.**

```bash
go test ./internal/usecase ./internal/ui/cli -run 'Test.*(Performance|Doctor|Reboot)' -count=1
```

Expected: FAIL — the flags and the precondition do not exist.

- [ ] **Step 3: Implement.**

Add the four flags. Order the run: providers → packages → reboot precondition →
swap → zram + derived swappiness → sysctls → limits → journald → timezone → debloat.
Each stage's diagnostics are collected and printed; a stage failure does not abort the
rest except where continuing would be unsafe, which is documented per stage in the code.

- [ ] **Step 4: Keep the doctor at 0 WARN / 0 ERROR on a converged host.**

Every new check reports `DiagOK` when the declared policy holds and `DiagInfo` when the
host is merely different (an adopted swapfile, a tier without zram, a timezone mismatch
under `verify`). `DiagWarning` is reserved for a genuine drift, and the reboot-pending
check is the one addition allowed to warn on a converged host.

- [ ] **Step 5: Verify GREEN.**

```bash
gofmt -l internal/
go build ./... && go vet ./...
go test ./... -count=1
```

**Rollback:** `git revert`.

**Result:** `envctl run performance --dry-run` prints the detected hardware, the selected
tier, every resolved value with its rationale, and writes nothing.

---

### Task 12: Correct the documentation that now states the opposite

**Files:**
- Modify: `docs/manifests.md:86-87` (currently: *"Nenhum dos dois cria swapfile ou altera journald"*)
- Modify: `docs/guides/linux.md:119-121` (same claim, same file family)
- Modify: `docs/guides/linux.md:3` (promise of Ubuntu 20.04/22.04, Debian 11/12, WSL2, OCI)
- Create: `docs/guides/ubuntu-server-baseline.md`
- Modify: `docs/os-and-agent-matrix.md`
- Modify: `docs/architecture.md`
- Modify: `CHANGELOG.md` under `[Unreleased]` only
- Modify: `AGENTS.md` if the profile identity is referenced there

- [ ] **Step 1: Write the failing documentation assertion.**

In `manifest_repo_test.go`, add an assertion that no tracked Markdown file contains the
phrases `cria swapfile` or `não cria swapfile` next to `journald` in a sentence claiming
neither happens. This is a lint, not a semantic test, and its purpose is to make the
stale claim impossible to re-introduce.

- [ ] **Step 2: Verify RED.**

```bash
go test ./internal/infra/embedded -run TestDocumentationClaimsMatchManifests -count=1
```

Expected: FAIL against the two current sentences.

- [ ] **Step 3: Write the documents.**

`docs/guides/ubuntu-server-baseline.md` documents, per item: what it sets, the measured
fleet evidence for the value, the exact revert command, and what it never touches.
`docs/guides/linux.md` is corrected to say the profile is Ubuntu Server `>= 24.04` only and
that Debian and older releases are rejected by design. `docs/manifests.md` describes the
new manifest sections and the `policy: min` semantics.

- [ ] **Step 4: Verify GREEN.**

```bash
go build ./... && go vet ./... && go test ./... -count=1
golangci-lint run --new-from-rev=origin/main
envctl-verify --dry-run
```

- [ ] **Step 5: Review and commit atomically.**

```bash
git status --short
git diff --check
git diff --stat
git add <task-files>
git commit -m "feat(linux): add Ubuntu Server baseline with hardware-detected optimization"
```

Never create a tag or a release; release-please owns those.

## Risks

| Risk | Probability | Impact | Mitigation |
| --- | --- | --- | --- |
| `daemon-reexec` over SSH disturbs the session | low | high | `man 1 systemctl` guarantees sockets stay accessible; gated by `--no-daemon-reexec`; only runs when a drop-in changed |
| Raising the `nofile` soft limit breaks `select(2)` in legacy software | low | medium | documented in `man 5 systemd.exec`; quoted in the apply diagnostic; host can set `nofile_soft` down without touching anything else |
| Package purge restarts a service and drops the session | medium | high | `NEEDRESTART_MODE=l` installed before the first purge; `ProtectedWhen` guards; runs last; individually revertible |
| Swapfile consumes a disk the fleet needs | low | high | clamped by `size_max` and `disk_reserve`; btrfs refused by default; the two OCI hosts adopt instead of create |
| Tier bands are wrong for a 4 GB or 8 GB host | medium | low | bands are 50% wider than nominal; the manifest marks the two unvalidated rows as derived; the diagnostic prints the detected `MemTotal` so drift is visible |
| `fs.file-max`-style regression on any other key | low | low | `policy: min` makes lowering impossible for any key that opts in |
| journald cap is set but the host has other consumers of `/var` | low | medium | `SystemKeepFree` is mandatory in the manifest, not optional |
| A host is neither OCI nor AWS and lands on a filesystem with no `fallocate` | low | medium | filesystem allowlist with a `findmnt` fallback; denied types report `DiagInfo` and change nothing |

## Unknowns

- **`zscan_proxy_prod` cannot authenticate**, so 9 of 10 fleet members are verified. If that
  host is a different shape or filesystem, the tier and filesystem branches are untested
  against it. Owner: the user. Next step: repair the key in `~/.ssh-manager/.env` and re-run
  the Task 1 Step 5 probe command. Not blocking — the design detects rather than assumes.
- **No 4 GB or 8 GB host exists in the fleet**, so the `small` and `medium` tier rows are
  derived rather than measured. Owner: the user, if one of those sizes is added later; the
  rows are marked in the manifest, so no one mistakes them for measurements.
- **Whether raising the soft `nofile` limit to 65536 breaks anything actually running on
  the fleet** is documented as a hazard but not empirically tested. Owner: the user, on a
  disposable host, before rolling to production. Mitigation: the flag exists and the value
  is one manifest line.
- **Whether zram at `tiny` actually improves the `vps_oracle_1` workload** (681 MiB of swap
  in use) is an untested hypothesis. Owner: the user. Next step: run the branch on
  `vps_oracle_1` and compare swap-in-rate before and after. This is the one change in the
  branch that alters workload behavior rather than headroom.

## Breaking changes

- **`PerformanceProfileUbuntu` is renamed to `PerformanceProfileUbuntuServer` and its value
  changes from `"ubuntu-24.04"` to `"ubuntu-server"`.** Consumers: the two manifest files,
  `manifest_repo_test.go`, `ui/cli/performance.go`, `provision_performance.go`. The old
  literal is not accepted, so a stale checkout fails loudly at load rather than silently
  skipping tuning. No external consumer: the profile string never appears in a public flag.
- **`run all` on a Linux host that is not Ubuntu Server `>= 24.04` now exits non-zero**
  instead of warning and continuing. Consumers: any Debian, Ubuntu 22.04, Arch-generic or
  WSL host that relied on the tolerant path. Migration: none intended — the owner declared
  Ubuntu Server 24+ as the only target. Verification: `go test ./internal/usecase -run
  TestRunVPSProfile_RejectsNonUbuntuServer`.
- **`Spec.Sysctls` and `spec.Sysctls` gain a `Policy` field, and `fs.file-max` stops being
  written.** Consumers: nothing outside the performance path; the rendered drop-in changes
  only by dropping a key the host already sets better. Verification: Task 4 Step 5.
- **The `run performance` command surface gains four flags and changes no existing one.**
  Consumers: CI and any scripted invocation, which keep working unchanged. Verification:
  Task 11 Step 5.
- **`ManifestRepository` gains `ListPerformanceProfiles`.** Consumers: the one production
  implementation in `internal/infra/embedded` and the test doubles in
  `provision_performance_test.go` and `verify_script_test.go`. Migration: implement the new
  method on each double. The compiler enumerates every affected site, so none can be missed.

## Rollback and reversibility

Every task that touches a host writes through `DropinWriter`, so each managed file has a
timestamped sibling at `<path>.bak.YYYYMMDD-HHMMSS` and an atomic rename. Host-side
reverts, in the order to apply them:

1. `sudo timedatectl set-timezone Etc/UTC` — Task 5.
2. `sudo systemctl restart systemd-journald` after restoring the drop-in — Task 6.
3. `sudo systemctl daemon-reexec` after removing the two limits drop-ins — Task 7.
4. `sudo swapoff /dev/zram0 && sudo systemctl disable --now dev-zram0.swap` — Task 8.
5. `sudo swapoff /swapfile.envctl && sudo rm /swapfile.envctl`, then restore
   `/etc/fstab.bak.<stamp>` — Task 9.
6. `sudo apt-get install -y <pkg>` per removed package — Task 10.
7. Restore `/etc/sysctl.d/90-envctl-performance.conf.bak.<stamp>` and `sudo sysctl -p` on it
   — Task 4.

Only Task 9 allocates disk and Task 10 removes packages; both are individually reversible
and neither is performed by `--dry-run`.

## Self-review checklist

- [ ] `ubuntu-24.04` appears nowhere outside `CHANGELOG.md` and `.worktrees/`.
- [ ] The `ubuntu-server` minimum is read from the manifest, and no Go constant holds a version.
- [ ] No test touches `/etc`, `/proc` or `/swap*`; every host interaction goes through an injected runner.
- [ ] A host that already has a swapfile is a pure no-op: no `fallocate`, no `mkswap`, no `swapon`, no fstab write.
- [ ] `vm.swappiness` is absent from every manifest and appears only as a derived value.
- [ ] `fs.file-max` is `policy: min` and is not written on a host at `9223372036854775807`.
- [ ] `systemctl stop` is never issued for journald; only `restart`.
- [ ] `DefaultLimitNOFILE` renders with an empty hard side.
- [ ] The needrestart drop-in is written before the first package removal.
- [ ] Every unvalidated value in the manifest says so in `rationale`.
- [ ] Every doctor check reports `DiagOK` or `DiagInfo` on a converged host, except reboot-pending.
- [ ] `go build`, `go vet`, `go test ./...`, `golangci-lint --new-from-rev` and a real
      `envctl run performance --dry-run` / `envctl run performance` / `envctl run
      performance` / `envctl doctor` cycle on `vps_oracle_2` have fresh output.
- [ ] No new agent skill was created; operational knowledge lives in `docs/guides/`.
