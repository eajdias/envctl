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

- `startupTargets` — the closed set, as `{token, kind, path|env}` records. The
  probe reports a **token**, never a path: a path crossing the process boundary
  is mangled by the console code page (non-ASCII `%APPDATA%`), and no separator
  can truncate it (`;` is legal in an NTFS name). `startupTargetForToken()`
  refuses anything unknown, so a target outside the set never reaches a
  deletion.
- `startupTargetsTable()` — renders the PowerShell hashtable both scripts
  share, so probe and apply cannot disagree about a target. Folders resolve via
  `[Environment]::GetFolderPath()` (absent `%APPDATA%`/`%ProgramData%` do not
  exist in non-interactive contexts).
- `startupProbeScript(names)` — one spawn answering every name with its
  `,`-joined tokens. Registry keys are read with `GetValueNames()` and folders
  with `BaseName`, compared with `-contains`/`-eq`, never through a wildcard
  matcher or `-Filter`.
- `startupRemovalScript(name, targets)` — `Remove-ItemProperty` for Run keys
  with the name escaped via `[WildcardPattern]::Escape()` (`-Name` is *always* a
  `WildcardPattern` in the registry provider, so an unescaped `*` would delete
  several values), `Get-ChildItem | Remove-Item` for Startup folders, skipping
  containers. No WMI method.
- `CheckTweak` case `startupitem` / `CheckBatch` new `startupIdx` family /
  `ApplyTweak` case `startupitem` — all three route through `probeStartup`, so
  the audit and the mutation cannot drift. Apply is a no-op when the probe
  reports nothing.
- `probeStartup` **fails on a partial readout** (answered != requested), the
  same guard `checkRegistryBatch` has: a truncated probe must not read as
  "absent" and certify the category as converged.
- `serviceExpectedState(tweak)` — extracted so check, batch and apply resolve
  the target `StartType` the same way (the ternary was duplicated in all three).

### Tests

- `tweaks_manager_test.go`: `startupTokenIsClosedSet`,
  `startupTargetForTokenRejectsUnknown`, `startupScriptsEmbedLiteralRegistryPaths`,
  `parseStartupProbe`, `startupUnknownTokens`, `startupConforms`,
  `startupRemovalScriptAvoidsMissingWmiMethod` (asserts it never emits
  `.Delete()`/`Invoke-CimMethod`, routes each kind to the right cmdlet, skips
  containers and uses `-LiteralPath`),
  `startupScriptsNeutralizeWildcardNames` (asserts
  `[WildcardPattern]::Escape` on the removal, `-contains` on the probe, no
  `Get-ItemProperty -Name` and no `-Filter`), `startupScriptsQuoteAdversarialNames`,
  a `StartupItem` entry in the existing batch-vs-single parity test, and
  `TestWindowsTweaksManager_StartupItemRoundTrip` — the only coverage of the
  destructive path: it creates a disposable HKCU Run value, proves the check
  reports drift, applies, proves convergence and re-applies for idempotency.
  The pure tables and script renderers run on Linux; the round-trip and the
  parity test skip off Windows and are exercised by the `windows-latest` CI job.
- `manifest_repo_test.go`: `expectedDebloat` 76 → 94, the `StartupItem` shape,
  unique lowercased `StartupItem` names (a duplicate would collapse in the
  probe readout and surface as a bogus "answered N of M" error), and a guard that
  no `-manual` id regressed to `Disabled`.

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
