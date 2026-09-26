# Spec — Debloat Tier 2 expansion (absorb the rest of windows11-clean)

Date: 2026-09-26
Status: approved (3 decisions confirmed with the owner)

## Goal

Close the gap left when `windows11-clean` was absorbed (#24): the three Appx
packages and 27 services left out of `manifests/debloat.yaml`, plus the
**startup-items phase that has no representation at all** in envctl.

## Decisions (owner-confirmed)

1. **Services → `Manual`, never `Disabled`.** The legacy tool disables
   `SysMain`/`WSearch`/`NgcSvc`/`wbengine`/`OneSyncSvc`/`Dell*` and only sets
   `fb*` to Manual. envctl keeps the non-destructive state: the service still
   starts on demand, nothing the machine depends on breaks, and reverting is a
   one-word manifest edit. `ApplyTweak` already implements
   `-StartupType Manual` (`tweaks_manager.go:451`), so **no Go change** is
   needed for services.
2. **Startup items: the legacy six minus the audio ones.** `WavesSvc` and
   `RtkAuduService` are audio driver services — disabling them can break the
   audio enhancer on the owner's hardware. Enter `BraveSoftware`, `Canva`,
   `MicrosoftEdge`, `SecurityHealth`.
3. **Startup removal scoped to Run keys + Startup folder.** The legacy uses
   `Win32_StartupCommand | .Delete()`, which also enumerates **services** as
   startup commands — deleting the wrong row breaks a service. envctl filters
   by `Location` and never touches a service.

## Changes

### `manifests/debloat.yaml` (76 → 94 tweaks)

| Family | Before | After | Added |
|---|---|---|---|
| registry (telemetry/privacy/gaming) | 36 | 36 | — |
| `Appx` (category `apps`) | 31 | 34 | `MSTeams`, `Microsoft.OutlookForWindows`, `Microsoft.Windows.Ai.Copilot.Provider` |
| `Service` → `Disabled` | 9 | 9 | — |
| `Service` → `Manual` (category `services`) | 0 | 11 | `WSearch`, `SysMain`, `NgcSvc`, `wbengine`, `OneSyncSvc`, `DellCustomerConnect`, `DellSupportAssistAgent`, `DellTechHub`, `DellOptiPlex`, `fbguard`, `fbserver` |
| `StartupItem` (category `startup`) | 0 | 4 | `BraveSoftware`, `Canva`, `MicrosoftEdge`, `SecurityHealth` |

New manifest type `StartupItem` (no `path`, no `value`; `name` = the
`Win32_StartupCommand` name). Xbox suite stays out — gaming.

### `internal/infra/windows/tweaks_manager.go`

- `startupLocationRemovable(location string) bool` — pure Go predicate, the
  single source of truth for "is this a user startup entry?". Accepts
  `HKCU`/`HKLM` `...\\CurrentVersion\\Run` (plus `Wow6432Node`) and any
  `...\\Start Menu\\Programs\\Startup` path; rejects service rows and
  everything else. Lives in Go (not PowerShell) so it is unit-testable off
  Windows and so check and apply **cannot** drift.
- `CheckTweak` case `startup` — one CIM query filtered by the predicate.
  Conforming = absent (`OK`).
- `CheckBatch` — new `startupIdx` family, one spawn listing
  `NAME|||LOCATION` for all rows, answered from that map (details strings
  identical to the single path).
- `ApplyTweak` case `startup` — re-queries, filters in Go with the same
  predicate, deletes each matching row by `Name` **and** `Location`. No match
  is a no-op, so apply is idempotent.

### Tests

- `tweaks_manager_test.go`: `startupLocationRemovable` table test (happy path
  Run key, Startup folder, boundary `RunOnce`/service row/empty/Wow6432Node)
  — runs on Linux, unlike the existing Windows-only tests.
- `manifest_repo_test.go`: `expectedDebloat` 76 → 94.

### Docs

- `manifests/debloat.yaml` header: the "Deliberately OUT" list is stale.
- `docs/manifests.md` §7: counts + the new type; drop the dead
  `skill windows-debloat` reference (removed from the catalog 2026-09-26).
- `docs/os-and-agent-matrix.md` lines 36/162/203: 76 → 94, same dead ref.
- `docs/guides/windows-debloat-tier3.md` §8/§9: Teams/Outlook and the Manual
  services are no longer Tier 3; add the startup scope.
- `CHANGELOG.md` `[Unreleased]`.

## Non-goals

- Registry rollback (neither tool has it).
- Xbox suite, `Spooler`, `AnyDesk`, `Firebird*`, `Tailscale`, `sshd`,
  `StorSvc`, `gupdate*`, `EasyAntiCheat*`, `NgcRingFenceSvc`.
- Tier 3 (OneDrive, power, `.wslconfig`, Teredo, `Binary`, Defender opt-out).
- The `debloat.yaml` manifest stays opt-in in the doctor: drift is `INFO`,
  never `WARN`, never `--fix`.

## Verification

```bash
go build ./... && go vet ./... && go test ./...
golangci-lint run --new-from-rev=origin/main
```

Live audit after deploy: `envctl.exe doctor` must show
`Debloat / category startup` and `Debloat / category services` with the new
counts, 0 WARN / 0 ERROR.
