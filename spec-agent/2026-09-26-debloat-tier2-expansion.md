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
   `Win32_StartupCommand | .Delete()`. Two defects in that approach, both
   confirmed against the published MOF and the class docs during review:
   - `Win32_StartupCommand` is a `CIM_Setting` whose MOF lists **properties
     only** — there is no `Delete` method, so the legacy call cannot work. The
     docs point at the *System Registry Provider* for changes instead.
   - Its `Location` is inconsistent (registry key path, the bare literals
     `Startup` / `Common Startup`, `HKU\<SID>\...`), and it does not enumerate
     services at all. A classifier over `Location` was therefore both
     unnecessary and fragile.

   envctl probes and removes against the two Run keys and the two Startup
   folders **directly**. "Never touches a service" becomes structural: a
   service can be neither a Run-key value nor a Startup-folder file.

## Changes

### `manifests/debloat.yaml` (76 → 94 tweaks)

| Family | Before | After | Added |
|---|---|---|---|
| registry (telemetry/privacy/gaming) | 36 | 36 | — |
| `Appx` (category `apps`) | 31 | 34 | `MSTeams`, `Microsoft.OutlookForWindows`, `Microsoft.Windows.Ai.Copilot.Provider` |
| `Service` → `Disabled` | 9 | 9 | — |
| `Service` → `Manual` (category `services`) | 0 | 11 | `WSearch`, `SysMain`, `NgcSvc`, `wbengine`, `OneSyncSvc`, `DellCustomerConnect`, `DellSupportAssistAgent`, `DellTechHub`, `DellOptiPlex`, `fbguard`, `fbserver` |
| `StartupItem` (category `startup`) | 0 | 4 | `BraveSoftware`, `Canva`, `MicrosoftEdge`, `SecurityHealth` |

New manifest type `StartupItem` (no `path`, no `value`; `name` = the Run-key
value name / Startup-folder file base name). Xbox suite stays out — gaming.

### `internal/infra/windows/tweaks_manager.go`

- `startupRunKeys` — the closed set of two Run keys; `startupKindRegistry` /
  `startupKindDir` classify a probed location.
- `startupProbeScript(names)` — one spawn answering every wanted name with its
  `;`-joined locations. Names are compared with `-eq` against enumerated
  values, never via `-Filter`, so a name carrying PowerShell wildcard
  characters stays inert.
- `startupTargetKind(location)` — pure Go classifier, unit-testable off
  Windows. Returns `""` for anything outside the closed set, and the removal
  script then skips it.
- `startupRemovalScript(name, locations)` — `Remove-ItemProperty` for Run
  keys, `Get-ChildItem | Remove-Item` for Startup folders. No WMI method.
- `CheckTweak` case `startupitem` / `CheckBatch` new `startupIdx` family /
  `ApplyTweak` case `startupitem` — all three route through `probeStartup`, so
  the audit and the mutation cannot drift. Apply is a no-op when the probe
  reports nothing.
- `serviceExpectedState(tweak)` — extracted so check, batch and apply resolve
  the target `StartType` the same way (the ternary was duplicated in all three).

### Tests

- `tweaks_manager_test.go`: `startupTargetKind` table, `parseStartupProbe`,
  `startupRemovalScript` (asserts it never emits `.Delete()`/`Invoke-CimMethod`
  and routes each kind to the right cmdlet), `startupScriptsQuoteAdversarialNames`
  (asserts `psQuote` and no `-Filter`), `startupConforms`, plus a `StartupItem`
  entry in the existing batch-vs-single parity test. All run on Linux, unlike
  the pre-existing Windows-only tests.
- `manifest_repo_test.go`: `expectedDebloat` 76 → 94, the `StartupItem` shape,
  and a guard that no `-manual` id regressed to `Disabled`.

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
