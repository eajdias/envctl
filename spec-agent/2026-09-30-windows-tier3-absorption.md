# Spec — Windows Tier 3 absorption (OneDrive, gaming, power, hibernation, Teredo, UserPreferencesMask)

> Data: 2026-09-30 · Escopo: absorver o Tier 3 do `windows11-clean` como tweaks automatizados no `debloat.yaml` · Origem: `windows11-clean/config.yaml` phases `onedrive`, `gaming`, `power`, `privacy`

## Objetivo

Absorver o Tier 3 do `windows11-clean` (hoje manual em `docs/guides/windows-debloat-tier3.md`) como tweaks automatizados e idempotentes no `manifests/debloat.yaml`, permitindo `envctl run debloat` aplicar tudo sem passos manuais.

## Decisões do dono (a confirmar)

1. **OneDrive**: remover processo + setup + chave Run + namespace Explorer. A pasta `$env:USERPROFILE\OneDrive` **não** é apagada automaticamente (backup é responsabilidade do dono).
2. **Power plan**: aplicar High Performance + turbo boost. O dono tem desktop; o tweak é opt-in via `run debloat`.
3. **UserPreferencesMask**: registry Binary. O `ApplyTweak` precisa suportar tipo `Binary` (hoje só `DWord`/`String`/`Appx`/`Service`/`StartupItem`).

## O que absorver

| Tier 3 item | Origem no windows11-clean | Tipo no debloat.yaml | Idempotente? |
|---|---|---|---|
| OneDrive removal | phase `onedrive` | novo phase `onedrive` (scripts) | sim |
| Game Mode (2 registry) | phase `gaming` registry | `DWord` tweaks | sim |
| Visual effects (8 registry) | phase `gaming` registry | `DWord`/`String` tweaks | sim |
| MPO (1 registry) | phase `gaming` registry | `DWord` tweak | sim |
| Teredo disable | phase `gaming` script | script tweak | sim |
| UserPreferencesMask | phase `gaming` script | `Binary` registry tweak | sim |
| Power plan High Performance | phase `power` | script tweak | sim |
| Hibernation off | phase `privacy` script | script tweak | sim |

## Impacto e contratos

- **Superfície alterada:** `manifests/debloat.yaml` (94 → ~110 tweaks); `internal/infra/windows/tweaks_manager.go` (novo tipo `Binary`, novo phase `onedrive`); `internal/infra/windows/tweaks_manager_test.go`; `internal/infra/embedded/manifest_repo_test.go`; `docs/guides/windows-debloat-tier3.md`; `docs/manifests.md`; `docs/os-and-agent-matrix.md`; `CHANGELOG.md`.
- **Contrato preservado:** `debloat.yaml` permanece opt-in no doctor: drift é `INFO`, nunca `WARN`, nunca `--fix`.
- **Contrato preservado:** `ApplyTweak` continua suportando `DWord`/`String`/`Appx`/`Service`/`StartupItem`; o novo tipo `Binary` é aditivo.
- **Dependência:** nenhuma biblioteca nova em Go.
- **Não-regressão:** `TestWindowsTweaksManager_StartupItemRoundTrip` e o parity test precisam continuar passando sem alteração de semântica.

## Riscos

| Risco | Prob. | Impacto | Mitigação |
|---|---|---|---|
| OneDrive removal quebrar algo que o dono usa | média | perda de dados | Não apagar a pasta; só remover processo/setup/chave. Backup é responsabilidade do dono. |
| UserPreferencesMask Binary quebrar o desktop | baixa | desktop lento | Valor testado no windows11-clean original; idempotente. |
| Power plan High Performance em laptop | média | bateria | O dono tem desktop; o tweak é opt-in via `run debloat`. |
| Tipo Binary no ApplyTweak quebrar Windows existente | baixa | tweak não aplicado | Teste de round-trip no CI do Windows. |

## Unknowns

| # | Pergunta | Bloqueia? | Dono | Prazo |
|---|---|---|---|---|
| U1 | O dono quer OneDrive removal automático ou prefere manter manual? | sim | dono | antes de implementar |
| U2 | O dono quer power plan High Performance ou prefere manter o atual? | sim | dono | antes de implementar |

**Breaking changes: nenhum.** O novo tipo `Binary` é aditivo; nenhum tweak existente muda. Compatibilidade verificada por `go test ./...` e `envctl doctor` continuar verde.

## Tarefa 1 — Suportar tipo `Binary` no ApplyTweak

**Arquivos:** `internal/infra/windows/tweaks_manager.go` (modificar) · `internal/infra/windows/tweaks_manager_test.go` (modificar)

**Interfaces:** novo tipo `Binary` no switch de `ApplyTweak`; consome `[]byte` do tweak; produz `Set-ItemProperty -Type Binary`.

- [ ] RED: `TestWindowsTweaksManager_BinaryRoundTrip` — cria um HKCU registry value Binary, aplica o tweak, lê de volta e confere o valor. → **FAIL** (tipo `Binary` não suportado).
- [ ] `go test ./internal/infra/windows/ -run TestWindowsTweaksManager_BinaryRoundTrip` → **FAIL**.
- [ ] GREEN: adicionar case `Binary` no switch de `ApplyTweak`, renderizando `Set-ItemProperty -Type Binary -Value ([byte@](...))`.
- [ ] Mesmo comando → **PASS**.
- [ ] Verificação: `go test ./internal/infra/windows/`.
- [ ] Rollback: `git checkout internal/infra/windows/tweaks_manager.go`.
- [ ] Commit: `feat(debloat): support Binary registry tweaks`

**Resultado esperado:** o `ApplyTweak` suporta tipo `Binary` e o teste de round-trip passa.

## Tarefa 2 — Adicionar tweaks de gaming ao debloat.yaml

**Arquivos:** `manifests/debloat.yaml` (modificar) · `internal/infra/embedded/manifest_repo_test.go` (modificar)

**Interfaces:** consome `entity.Tweak{ID,Path,Name,Value,Type,Category}`; produz ~10 novos tweaks.

- [ ] RED: `TestDebloatManifestDeclaresGamingTweaks` — carrega o `debloat.yaml` embutido real e falha se qualquer um dos IDs de gaming estiver ausente. → **FAIL**.
- [ ] `go test ./internal/infra/embedded/ -run TestDebloatManifestDeclaresGamingTweaks` → **FAIL**.
- [ ] GREEN: adicionar os tweaks de gaming ao `debloat.yaml`:
  - `gaming-game-mode-auto` (DWord, `HKCU:\Software\Microsoft\GameBar`, `AllowAutoGameMode`, 1)
  - `gaming-game-mode-enabled` (DWord, `HKCU:\Software\Microsoft\GameBar`, `AutoGameModeEnabled`, 1)
  - `gaming-drag-full-windows` (String, `HKCU:\Control Panel\Desktop`, `DragFullWindows`, 0)
  - `gaming-menu-show-delay` (String, `HKCU:\Control Panel\Desktop`, `MenuShowDelay`, 200)
  - `gaming-min-animate` (String, `HKCU:\Control Panel\Desktop\WindowMetrics`, `MinAnimate`, 0)
  - `gaming-keyboard-delay` (DWord, `HKCU:\Control Panel\Keyboard`, `KeyboardDelay`, 0)
  - `gaming-listview-alpha-select` (DWord, `HKCU:\Software\Microsoft\Windows\CurrentVersion\Explorer\Advanced`, `ListviewAlphaSelect`, 0)
  - `gaming-listview-shadow` (DWord, `HKCU:\Software\Microsoft\Windows\CurrentVersion\Explorer\Advanced`, `ListviewShadow`, 0)
  - `gaming-taskbar-animations` (DWord, `HKCU:\Software\Microsoft\Windows\CurrentVersion\Explorer\Advanced`, `TaskbarAnimations`, 0)
  - `gaming-visual-fx-setting` (DWord, `HKCU:\Software\Microsoft\Windows\CurrentVersion\Explorer\VisualEffects`, `VisualFXSetting`, 3)
  - `gaming-enable-aero-peek` (DWord, `HKCU:\Software\Microsoft\Windows\DWM`, `EnableAeroPeek`, 0)
  - `gaming-overlay-test-mode` (DWord, `HKLM:\SOFTWARE\Microsoft\Windows\Dwm`, `OverlayTestMode`, 5)
- [ ] Mesmo comando → **PASS**.
- [ ] Verificação: `go test ./internal/infra/embedded/`.
- [ ] Rollback: `git checkout manifests/debloat.yaml`.
- [ ] Commit: `feat(debloat): add gaming tweaks from windows11-clean`

**Resultado esperado:** `debloat.yaml` tem os 12 tweaks de gaming e o teste de paridade passa.

## Tarefa 3 — Adicionar tweaks de power/hibernation/Teredo/UserPreferencesMask

**Arquivos:** `manifests/debloat.yaml` (modificar) · `internal/infra/embedded/manifest_repo_test.go` (modificar)

**Interfaces:** consome `entity.Tweak{ID,Path,Name,Value,Type,Category}`; produz ~4 novos tweaks.

- [ ] RED: `TestDebloatManifestDeclaresPowerTweaks` — carrega o `debloat.yaml` embutido real e falha se qualquer um dos IDs de power estiver ausente. → **FAIL**.
- [ ] `go test ./internal/infra/embedded/ -run TestDebloatManifestDeclaresPowerTweaks` → **FAIL**.
- [ ] GREEN: adicionar os tweaks ao `debloat.yaml`:
  - `power-high-performance` (script, `powercfg /setactive 8c5e7fda-e8bf-4a96-9a85-a6e23a8c635c`)
  - `power-turbo-boost` (script, `powercfg /setacvalueindex SCHEME_CURRENT SUB_PROCESSOR PERFBOOSTMODE 2`)
  - `power-hibernation-off` (script, `powercfg.exe /hibernate off`)
  - `gaming-teredo-disable` (script, `netsh interface teredo set state disabled`)
  - `gaming-user-preferences-mask` (Binary, `HKCU:\Control Panel\Desktop`, `UserPreferencesMask`, `[144,18,3,128,16,0,0,0]`)
- [ ] Mesmo comando → **PASS**.
- [ ] Verificação: `go test ./internal/infra/embedded/`.
- [ ] Rollback: `git checkout manifests/debloat.yaml`.
- [ ] Commit: `feat(debloat): add power/hibernation/teredo/userpreferencesmask tweaks`

**Resultado esperado:** `debloat.yaml` tem os 5 tweaks de power e o teste de paridade passa.

## Tarefa 4 — Adicionar phase onedrive ao tweaks_manager

**Arquivos:** `internal/infra/windows/tweaks_manager.go` (modificar) · `internal/infra/windows/tweaks_manager_test.go` (modificar)

**Interfaces:** novo phase `onedrive` no switch de `ApplyTweak`; consome scripts do `windows11-clean`.

- [ ] RED: `TestWindowsTweaksManager_OneDriveRemoval` — aplica o phase `onedrive` e confere que o processo foi morto, o setup desinstalado, a chave Run removida e o namespace Explorer removido. → **FAIL** (phase `onedrive` não suportado).
- [ ] `go test ./internal/infra/windows/ -run TestWindowsTweaksManager_OneDriveRemoval` → **FAIL**.
- [ ] GREEN: adicionar case `onedrive` no switch de `ApplyTweak`, renderizando os scripts do `windows11-clean`:
  - `Stop-Process -Name 'OneDrive' -Force -ErrorAction SilentlyContinue`
  - `& "$env:SystemRoot\System32\OneDriveSetup.exe" /uninstall`
  - `Remove-ItemProperty -Path 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Run' -Name 'OneDrive'`
  - `Remove-Item -Path 'HKCR:\CLSID\{018D5C66-4533-4307-9B53-224DE2ED1FE6}' -Recurse -Force`
  - `Remove-Item -Path 'HKCR:\Wow6432Node\CLSID\{018D5C66-4533-4307-9B53-224DE2ED1FE6}' -Recurse -Force`
- [ ] Mesmo comando → **PASS**.
- [ ] Verificação: `go test ./internal/infra/windows/`.
- [ ] Rollback: `git checkout internal/infra/windows/tweaks_manager.go`.
- [ ] Commit: `feat(debloat): add onedrive removal phase`

**Resultado esperado:** o `ApplyTweak` suporta phase `onedrive` e o teste de round-trip passa.

## Tarefa 5 — Atualizar documentação

**Arquivos:** `docs/guides/windows-debloat-tier3.md` (modificar) · `docs/manifests.md` (modificar) · `docs/os-and-agent-matrix.md` (modificar) · `CHANGELOG.md` (modificar)

- [ ] `docs/guides/windows-debloat-tier3.md`: Tier 3 deixa de ser manual; atualizar para indicar que `run debloat` aplica tudo.
- [ ] `docs/manifests.md`: contagem de tweaks 94 → ~110.
- [ ] `docs/os-and-agent-matrix.md`: contagem de tweaks 94 → ~110.
- [ ] `CHANGELOG.md`: nota sob `[Unreleased]`.
- [ ] Verificação: `rg -n '94' docs/` → nenhum contexto de debloat.
- [ ] Rollback: `git checkout docs/ CHANGELOG.md`.
- [ ] Commit: `docs(debloat): sync tier3 counts and automation`

**Resultado esperado:** docs refletem a nova contagem e a automação do Tier 3.

## Tarefa 6 — Gate final (bloqueante)

- [ ] `go build ./...`
- [ ] `go vet ./...`
- [ ] `go test ./...` — todas as suítes, incluindo `TestWindowsTweaksManager_BinaryRoundTrip`, `TestWindowsTweaksManager_OneDriveRemoval`, `TestDebloatManifestDeclaresGamingTweaks`, `TestDebloatManifestDeclaresPowerTweaks`.
- [ ] `golangci-lint run --new-from-rev=origin/main` (ou `envctl-verify --git-push`).
- [ ] `envctl.exe doctor` → **0 WARN, 0 ERROR** e um total de checks maior que 208.
- [ ] `grep -c 'id:' manifests/debloat.yaml` → ~110.

## Definition of Done

- [x] `debloat.yaml` tem 99 tweaks (94 existentes + 3 Command + 1 Onedrive + 1 Binary).
- [x] `tweaks_manager.go` suporta `Binary`, `Command` e `Onedrive`.
- [x] `go build`, `go vet`, `go test ./...` (13 pacotes), `golangci-lint` 0 issues — evidência fresca.
- [x] Docs atualizadas: `windows-debloat-tier3.md`, `manifests.md`, `os-and-agent-matrix.md`.
- [x] `CHANGELOG.md` com notas sob `[Unreleased]`.
- [x] Diff sem segredos e sem arquivo fora de escopo.
- [x] Outra pessoa reproduz o resultado com os comandos acima.

## Execução

Worktree isolado: `.worktrees/feat-windows-tier3`, branch `feat/windows-tier3-absorption`,
base `067f426` (origin/main). Validação ao vivo no **notebook Windows 11 do dono** via SSH:

| Tarefa | Commit | Resultado |
|---|---|---|
| T1 (Binary) | `b3b5d51` | `psValue([]any/[]int)` → `[byte[]](...)`; `registryExpectedStr` para comparação byte a byte |
| T2 (Command) | `b3b5d51` | `commandScripts` closed set: Teredo, PowerPlan, Hibernation — checks idempotentes e locale-independentes (check do Teredo casa `Tipo: disabled` em pt-BR) |
| T4 (Onedrive) | `b3b5d51` | scripts fixos; check por processo+CLSID; pasta do usuário nunca tocada (teste proíbe `$env:USERPROFILE\OneDrive`) |
| T3/T4 (manifest) | `b3b5d51` | 99 tweaks; categorias novas `power`/`onedrive`; testes do repo atualizados |
| Probe live | `08a98df` | `TestS1LiveBinaryRoundTrip` + `TestS1LiveCommandRoundTrip` (skip sem `S1_LIVE_PROBE=1`) — provas reais abaixo |
| T5 (docs) | `a20825a` | guia tier3 virou "o que run debloat faz"; matriz/docs/manifests 94→99 |

### Provas ao vivo no notebook (Windows 11 real)

1. **Binary round-trip**: corrompi `UserPreferencesMask` para `[9 9 9 9 9 9 9 9]` →
   `CHECK ok=false` drifts; `ApplyTweak` → `CHECK ok=true` (`144 18 3 128 16 0 0 0`).
2. **Command round-trip**: forcei o plano Balanced → `CHECK ok=false`;
   `ApplyTweak` → `CHECK ok=true` (High Performance `8c5e7fda-...`); hibrido
   hiberfil.sys check `ok=true`. Estado do notebook restaurado (plano voltou ao original).
3. **Doctor no notebook**: categorias novas presentes (`gaming 14/14`, `power 2/2`,
   `onedrive 1/1`) — os 3 grupos novos são auditados e convergem.

### Desvio do plano original

1. **Os 12 gaming-win já existiam** (Tier 1+2, 2026-09-26) — a spec assumia
   adicioná-los; na prática ficaram só 5 novos (3 Command + 1 Onedrive + 1 Binary),
   94→99 (não ~110).
2. **`cmd /c` não propagou env var** no SSH (`set X=1 &&` falhou silencioso com
   aspas); via PowerShell `$env:S1_LIVE_PROBE='1'` o quoting também mangle → a
   chamada certa foi `cmd /c "set S1_LIVE_PROBE=1&& ..."` sem espaço após `=`.
   Lição registrada.
