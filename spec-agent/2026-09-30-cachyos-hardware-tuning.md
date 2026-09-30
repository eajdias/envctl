# Spec — CachyOS hardware knowledge + tuning audit + emulator config

> Data: 2026-09-30 · Escopo: absorver o conhecimento de hardware, tuning privilegiado e config de emuladores do `cachyos-init` · Origem: `cachyos-init/fase-01-base/` até `fase-09-validacao/`

## Objetivo

Absorver o conhecimento de hardware específico (AVX2, ReBAR, PCIe 2.0), tuning privilegiado (LACT fan curve, scx_loader, kwinrc) e config de emuladores do `cachyos-init` no `envctl`, permitindo a remoção segura do repositório legado.

## Decisões do dono (a confirmar)

1. **Hardware knowledge → documentação**: o conhecimento de AVX2, ReBAR, PCIe 2.0 e limites de emulação vai para `docs/guides/cachyos-gaming.md` como seção "Hardware". Não é check de doctor (é contexto, não drift).
2. **Tuning privilegiado → checks INFO**: LACT fan curve, scx_loader config, kwinrc compositing são auditados como INFO (nunca WARN). O dono decide se aplica ou não.
3. **Config de emuladores → checks INFO**: as configs dos emuladores são auditadas como INFO. O dono decide se aplica ou não.
4. **Nenhum auto-fix**: nenhum check novo é corrigido por `doctor --fix`. O tuning privilegiado continua manual.

## O que absorver

### A. Conhecimento de hardware (documentação)

| Conhecimento | Origem | Destino |
|---|---|---|
| Limitações AVX2 (Sandy Bridge x86-64-v2) | `cachyos-init/README.md` + `fase-07-emu/` | `docs/guides/cachyos-gaming.md` (seção Hardware) |
| ReBAR/PCIe 2.0 (~5% na RX 580) | `cachyos-init/fase-08-sistema/README.md` | `docs/guides/cachyos-gaming.md` (seção Hardware) |
| Limites de emulação (RPCS3/PS3 inviável, Switch AAA slideshow) | `cachyos-init/fase-07-emu/eden-retroarch.md` | `docs/guides/cachyos-gaming.md` (seção Emuladores) |

### B. Tuning privilegiado (audit no doctor, não auto-fix)

| Tuning | Origem | Check no doctor |
|---|---|---|
| LACT fan curve | `fase-04-gpu/README.md` | Novo check INFO: `/etc/lact/config.yaml` existe |
| `scx_loader` config | `fase-03-cpu/README.md` | Novo check INFO: `/etc/scx_loader/config.toml` existe |
| `kwinrc` compositing | `fase-06-kde/README.md` | Novo check INFO: `[Compositing]` no kwinrc |

### C. Config de emuladores (documentação + checks de doctor)

| Config | Origem | Check no doctor |
|---|---|---|
| Dolphin `GFXBackend=Vulkan`, `ShaderCompilationMode=2` | `fase-07-emu/controles.md` | Novo check INFO: `~/.config/dolphin-emu/Dolphin.ini` |
| RetroArch `video_driver="vulkan"` | `fase-07-emu/controles.md` | Novo check INFO: `~/.config/retroarch/retroarch.cfg` |
| PPSSPP `GraphicsBackend=3` | `fase-07-emu/controles.md` | Novo check INFO: `~/.config/ppsspp/PSP/SYSTEM/ppsspp.ini` |
| PCSX2 `Renderer=14`, `mtvu=true` | `fase-07-emu/controles.md` | Novo check INFO: `~/.config/PCSX2/inis/PCSX2.ini` |
| DuckStation `Renderer=Vulkan`, `ResolutionScale=5` | `fase-07-emu/controles.md` | Novo check INFO: `~/.config/duckstation/settings.ini` |
| Azahar `graphics_api=2`, async | `fase-07-emu/controles.md` | Novo check INFO: `~/.config/azahar-emu/qt-config.ini` |
| Eden `backend=1`, `resolution_setup=3` | `fase-07-emu/eden-retroarch.md` | Novo check INFO: `~/.config/eden/qt-config.ini` |
| Vita3K `backend-renderer: Vulkan` | `fase-07-emu/controles.md` | Novo check INFO: `~/.config/Vita3K/config.yml` |
| Cemu `api=1`, `AsyncCompile=true` | `fase-07-emu/bios-emuladores.md` | Novo check INFO: `~/.config/Cemu/settings.xml` |

## Impacto e contratos

- **Superfície alterada:** `docs/guides/cachyos-gaming.md` (seção Hardware + seção Emuladores); `internal/usecase/doctor_audit.go` (novos checks INFO); `internal/usecase/doctor_audit_test.go`; `docs/os-and-agent-matrix.md`; `CHANGELOG.md`.
- **Contrato preservado:** `doctor` permanece **read-only** para tuning privilegiado — nenhum check novo é corrigido por `--fix`.
- **Contrato preservado:** os novos checks são `DiagInfo` (verde no renderizador), nunca `DiagWarning`.
- **Dependência:** nenhuma biblioteca nova em Go.
- **Não-regressão:** `TestMissingCmdlineParams` e os testes de gaming precisam continuar passando sem alteração de semântica.

## Riscos

| Risco | Prob. | Impacto | Mitigação |
|---|---|---|---|
| Checks INFO de config de emuladores gerar ruído | baixa | doctor mais verbose | Checks são INFO (verde no renderizador), não WARN. |
| Config de emuladores divergir entre versões | média | check falso positivo | Checks são INFO, não WARN. O dono decide se atualiza. |
| LACT fan curve check falso positivo se LACT não estiver instalado | baixa | check falso positivo | Check só emite INFO se `/etc/lact/config.yaml` existir; caso contrário, silêncio. |

## Unknowns

| # | Pergunta | Bloqueia? | Dono | Prazo |
|---|---|---|---|---|
| U1 | O dono quer checks INFO para config de emuladores ou prefere manter só documentação? | sim | dono | antes de implementar |
| U2 | O dono quer que o envctl gerencie o LACT fan curve ou prefere manter manual? | sim | dono | antes de implementar |

**Breaking changes: nenhum.** Os novos checks são aditivos e INFO; nenhum check existente muda. Compatibilidade verificada por `go test ./...` e `envctl doctor` continuar verde.

## Tarefa 1 — Adicionar seção Hardware ao guia de gaming

**Arquivos:** `docs/guides/cachyos-gaming.md` (modificar)

**Sem código, sem teste** — é doc. Verificação é `rg` (abaixo).

- [ ] Adicionar seção "Hardware" ao `docs/guides/cachyos-gaming.md`:
  - Limitações AVX2 (Sandy Bridge x86-64-v2): só repos genéricos, AppImages legacy, testar SIGILL.
  - ReBAR/PCIe 2.0: ~5% de perda na RX 580, normal para a plataforma.
  - Limites de emulação: RPCS3/PS3 inviável, Switch AAA slideshow, xemu/simple64 SIGILL.
- [ ] Verificação: `rg -n 'AVX2|ReBAR|PCIe 2.0' docs/guides/cachyos-gaming.md` → presente.
- [ ] Rollback: `git checkout docs/guides/cachyos-gaming.md`.
- [ ] Commit: `docs(gaming): add hardware limitations section`

**Resultado esperado:** o guia documenta as limitações de hardware do i7-2600 + RX 580.

## Tarefa 2 — Adicionar checks INFO de tuning privilegiado

**Arquivos:** `internal/usecase/doctor_audit.go` (modificar) · `internal/usecase/doctor_audit_test.go` (modificar)

**Interfaces:** novo helper puro `lactConfigPresent(root string) bool`; novo helper puro `scxLoaderConfigPresent(root string) bool`; novo helper puro `kwinrcCompositingPresent(root string) bool`.

- [ ] RED: três testes —
  1. `TestDoctorAudit_Gaming_TuningChecksLactConfig` — `/etc/lact/config.yaml` existe → `DiagOK`; não existe → `DiagInfo`.
  2. `TestDoctorAudit_Gaming_TuningChecksScxLoaderConfig` — `/etc/scx_loader/config.toml` existe → `DiagOK`; não existe → `DiagInfo`.
  3. `TestDoctorAudit_Gaming_TuningChecksKwinrcCompositing` — `[Compositing]` no kwinrc → `DiagOK`; não existe → `DiagInfo`.
- [ ] `go test ./internal/usecase/ -run 'TestDoctorAudit_Gaming_TuningChecks(Lact|ScxLoader|Kwinrc)'` → **FAIL** (undefined).
- [ ] GREEN: implementar os 3 helpers e os 3 checks em `auditGamingTuning`:
  - `lactConfigPresent` → check `System: "Gaming"`, `Target: "lact-config"`
  - `scxLoaderConfigPresent` → check `System: "Gaming"`, `Target: "scx-loader-config"`
  - `kwinrcCompositingPresent` → check `System: "Gaming"`, `Target: "kwinrc-compositing"`
- [ ] Mesmo comando → **PASS**.
- [ ] Verificação: `go test ./internal/usecase/`.
- [ ] Rollback: `git checkout internal/usecase/doctor_audit.go`.
- [ ] Commit: `feat(doctor): audit LACT, scx_loader and kwinrc as info`

**Resultado esperado:** o doctor audita LACT, scx_loader e kwinrc como INFO.

## Tarefa 3 — Adicionar checks INFO de config de emuladores

**Arquivos:** `internal/usecase/doctor_audit.go` (modificar) · `internal/usecase/doctor_audit_test.go` (modificar)

**Interfaces:** novo helper puro `emulatorConfigPresent(root string, path string, key string) bool`.

- [ ] RED: `TestDoctorAudit_Gaming_TuningChecksEmulatorConfigs` — config de emulador existe → `DiagOK`; não existe → `DiagInfo`. → **FAIL** (undefined).
- [ ] `go test ./internal/usecase/ -run TestDoctorAudit_Gaming_TuningChecksEmulatorConfigs` → **FAIL**.
- [ ] GREEN: implementar o helper e os checks em `auditGamingTuning`:
  - Dolphin: `~/.config/dolphin-emu/Dolphin.ini` com `GFXBackend=Vulkan`
  - RetroArch: `~/.config/retroarch/retroarch.cfg` com `video_driver="vulkan"`
  - PPSSPP: `~/.config/ppsspp/PSP/SYSTEM/ppsspp.ini` com `GraphicsBackend=3`
  - PCSX2: `~/.config/PCSX2/inis/PCSX2.ini` com `Renderer=14`
  - DuckStation: `~/.config/duckstation/settings.ini` com `Renderer=Vulkan`
  - Azahar: `~/.config/azahar-emu/qt-config.ini` with `graphics_api=2`
  - Eden: `~/.config/eden/qt-config.ini` com `backend=1`
  - Vita3K: `~/.config/Vita3K/config.yml` com `backend-renderer: Vulkan`
  - Cemu: `~/.config/Cemu/settings.xml` com `api=1`
- [ ] Mesmo comando → **PASS**.
- [ ] Verificação: `go test ./internal/usecase/`.
- [ ] Rollback: `git checkout internal/usecase/doctor_audit.go`.
- [ ] Commit: `feat(doctor): audit emulator configs as info`

**Resultado esperado:** o doctor audita as configs dos emuladores como INFO.

## Tarefa 4 — Atualizar documentação

**Arquivos:** `docs/os-and-agent-matrix.md` (modificar) · `CHANGELOG.md` (modificar)

- [ ] `docs/os-and-agent-matrix.md`: lista de checks do doctor seção Gaming recebe os novos.
- [ ] `CHANGELOG.md`: nota sob `[Unreleased]`.
- [ ] Verificação: `rg -n 'lact-config|scx-loader-config|kwinrc-compositing' docs/` → presente.
- [ ] Rollback: `git checkout docs/ CHANGELOG.md`.
- [ ] Commit: `docs(gaming): sync doctor check list`

**Resultado esperado:** docs refletem os novos checks INFO.

## Tarefa 5 — Gate final (bloqueante)

- [ ] `go build ./...`
- [ ] `go vet ./...`
- [ ] `go test ./...` — todas as suítes, incluindo `TestDoctorAudit_Gaming_TuningChecksLactConfig`, `TestDoctorAudit_Gaming_TuningChecksScxLoaderConfig`, `TestDoctorAudit_Gaming_TuningChecksKwinrcCompositing`, `TestDoctorAudit_Gaming_TuningChecksEmulatorConfigs`.
- [ ] `golangci-lint run --new-from-rev=origin/main` (ou `envctl-verify --git-push`).
- [ ] `envctl doctor` → **0 WARN, 0 ERROR** e um total de checks maior que 219.

## Definition of Done

- [ ] `docs/guides/cachyos-gaming.md` tem seção Hardware + seção Emuladores atualizada.
- [ ] `doctor_audit.go` tem os novos checks INFO (LACT, scx_loader, kwinrc, configs de emuladores).
- [ ] `go build`, `go vet`, `go test ./...`, `golangci-lint` e `envctl doctor` com evidência fresca.
- [ ] Docs atualizadas: `os-and-agent-matrix.md`.
- [ ] `CHANGELOG.md` com notas sob `[Unreleased]`.
- [ ] Diff sem segredos e sem arquivo fora de escopo.
- [ ] Outra pessoa reproduz o resultado com os comandos acima.

## Execução

Worktree isolado:

```
git worktree add .worktrees/feat-cachyos-hardware -b feat/cachyos-hardware-tuning
```

Ordem executada: T1 → T2 → T3 → T4 → T5.
