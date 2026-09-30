# Plano — Absorver o restante dos três projetos legados

> Data: 2026-09-30 · Escopo: windows11-clean (Tier 3), cachyos-init (hardware + tuning + emuladores) · Origem: decisão do dono de absorver o que falta antes de apagar os repositórios legados

## Objetivo

Fechar o delta restante entre o `envctl` e os dois projetos legados, permitindo a remoção segura dos repositórios locais e do git. O `envctl` já absorveu:
- **windows11-clean**: 94 tweaks de debloat (Tier 1+2) — falta o Tier 3 (OneDrive, gaming, power, hibernation, Teredo, UserPreferencesMask)
- **cachyos-init**: 44 pacotes de gaming + presets de usuário — falta o conhecimento de hardware, tuning privilegiado e config de emuladores

## Decisões do dono (a confirmar)

1. **windows11-clean Tier 3**: absorver como **tweaks automatizados** no `debloat.yaml` (não como guia manual). O dono quer rodar `envctl run debloat` e ter tudo aplicado, sem passos manuais.
2. **cachyos-init**: absorver o conhecimento de hardware como **documentação + checks de doctor** (não como auto-fix). O tuning privilegiado (kernel cmdline, LACT fan curve) continua manual, mas o doctor deve auditá-lo.

## Estrutura do plano

Duas specs independentes, cada uma com escopo próprio:

| Spec | Projeto | Escopo | Arquivo |
|---|---|---|---|
| S1 | windows11-clean | Tier 3: OneDrive, gaming, power, hibernation, Teredo, UserPreferencesMask | `spec-agent/2026-09-30-windows-tier3-absorption.md` |
| S3 | cachyos-init | Hardware knowledge + tuning audit + emulator config | `spec-agent/2026-09-30-cachyos-hardware-tuning.md` |

**S2 cancelado** — o dono optou por não absorver a otimização dinâmica por tier de RAM do `ubuntu-VPS-init`. A intenção é sempre performance; os limites de containers continuam definidos manualmente pelo dono.

## S1 — windows11-clean Tier 3

### O que absorver

| Tier 3 item | Origem | Destino no envctl |
|---|---|---|
| OneDrive removal | `windows11-clean/config.yaml` phase `onedrive` | `debloat.yaml` + `tweaks_manager.go` (novo phase) |
| Gaming tweaks (Game Mode, visual effects, MPO, Teredo) | `config.yaml` phase `gaming` | `debloat.yaml` (registry tweaks) |
| Power plan (High Performance) | `config.yaml` phase `power` | `debloat.yaml` (script tweak) |
| Hibernation off | `config.yaml` phase `privacy` script | `debloat.yaml` (script tweak) |
| Teredo disable | `config.yaml` phase `gaming` script | `debloat.yaml` (script tweak) |
| UserPreferencesMask | `config.yaml` phase `gaming` script | `debloat.yaml` (binary registry tweak) |

### Decisões de design

1. **OneDrive**: remover processo + setup + chave Run + namespace Explorer. A pasta `$env:USERPROFILE\OneDrive` **não** é apagada automaticamente (backup é responsabilidade do dono). O tweak é idempotente: se OneDrive já foi removido, o check reporta OK.
2. **Gaming tweaks**: Game Mode (2 registry), visual effects (8 registry), MPO (1 registry), Teredo (1 script). Todos idempotentes.
3. **Power plan**: `powercfg /setactive` + turbo boost. Idempotente.
4. **Hibernation**: `powercfg /hibernate off`. Idempotente.
5. **UserPreferencesMask**: registry Binary. O `ApplyTweak` precisa suportar tipo `Binary` (hoje só `DWord`/`String`/`Appx`/`Service`/`StartupItem`).

### Arquivos afetados

- `manifests/debloat.yaml` (modificar — novos tweaks)
- `internal/infra/windows/tweaks_manager.go` (modificar — novo tipo `Binary`, novo phase `onedrive`)
- `internal/infra/windows/tweaks_manager_test.go` (modificar — testes dos novos tweaks)
- `internal/infra/embedded/manifest_repo_test.go` (modificar — contagem de tweaks)
- `docs/guides/windows-debloat-tier3.md` (modificar — Tier 3 deixa de ser manual)
- `docs/manifests.md` (modificar — contagem)
- `docs/os-and-agent-matrix.md` (modificar — contagem)
- `CHANGELOG.md` (modificar — nota sob `[Unreleased]`)

### Riscos

| Risco | Prob. | Impacto | Mitigação |
|---|---|---|---|
| OneDrive removal quebrar algo que o dono usa | média | perda de dados | Não apagar a pasta; só remover processo/setup/chave. Backup é responsabilidade do dono. |
| UserPreferencesMask Binary quebrar o desktop | baixa | desktop lento | Valor testado no windows11-clean original; idempotente. |
| Power plan High Performance em laptop | média | bateria | O dono tem desktop; o tweak é opt-in via `run debloat`. |

### Unknowns

| # | Pergunta | Bloqueia? | Dono | Prazo |
|---|---|---|---|---|
| U1 | O dono quer OneDrive removal automático ou prefere manter manual? | sim | dono | antes de implementar |
| U2 | O dono quer power plan High Performance ou prefere manter o atual? | sim | dono | antes de implementar |

### Rollback

- `git checkout manifests/debloat.yaml internal/infra/windows/tweaks_manager.go`
- Os tweaks são idempotentes e reversíveis (registry pode ser restaurado, serviços podem ser reabilitados)

---

## S2 — ubuntu-VPS-init: otimização dinâmica por tier de RAM

### O que absorver

O `ubuntu-VPS-init/scripts/99-optimize-performance.sh` ajusta limites de containers com base na RAM detectada:

| Tier | RAM | PG buffers | Redis maxmem | CW concurrency | CW threads | CW mem | EVO mem | N8N mem | N8N prune |
|---|---|---|---|---|---|---|---|---|---|
| ECO | ≤2GB | 128MB | 128mb | 0 | 5 | 1G | 512M | 700M | true |
| BALANCED | ≤4GB | 512MB | 256mb | 1 | 10 | 1.5G | 700M | 1G | true |
| PERFORMANCE | >4GB | 1GB | 512mb | 2 | 15 | 3G | 1G | 2G | false |

### Decisões de design

1. **Não absorver os Docker Compose files** — o escopo do envctl é provisionamento de sistema, não orquestração de containers. A stack de containers (n8n, Chatwoot, Evolution API) continua no `ubuntu-VPS-init` ou migra para outro repositório.
2. **Absorver a lógica de tier de RAM** — o `performance_ubuntu.yaml` já tem tiers (`tiny`, `small`, `medium`, `large`). A otimização dinâmica pode ser um novo campo no manifesto: `container_limits` por tier.
3. **Aplicar via `run performance`** — o comando já detecta o tier de RAM. Os limites de containers seriam aplicados automaticamente.

### Arquivos afetados

- `manifests/performance_ubuntu.yaml` (modificar — novo campo `container_limits` por tier)
- `internal/usecase/provision_performance.go` (modificar — aplicar limites de containers)
- `internal/usecase/provision_performance_test.go` (modificar — testes dos novos limites)
- `docs/guides/ubuntu-server-baseline.md` (modificar — documentar os limites)
- `CHANGELOG.md` (modificar — nota sob `[Unreleased]`)

### Riscos

| Risco | Prob. | Impacto | Mitigação |
|---|---|---|---|
| Limites de containers quebrar serviços que já rodam | média | serviço indisponível | Aplicar só em containers criados pelo envctl; containers existentes são adotados (não modificados). |
| Tier de RAM não corresponder ao uso real | baixa | subutilização | Os tiers já são validados na frota atual. |

### Unknowns

| # | Pergunta | Bloqueia? | Dono | Prazo |
|---|---|---|---|---|
| U1 | O dono quer que o envctl gerencie os limites dos containers ou prefere manter isso manual? | sim | dono | antes de implementar |
| U2 | Os containers (n8n, Chatwoot, Evolution API) continuam no ubuntu-VPS-init ou migram para outro repositório? | sim | dono | antes de implementar |

### Rollback

- `git checkout manifests/performance_ubuntu.yaml internal/usecase/provision_performance.go`
- Os limites de containers são reversíveis (docker compose up -d com valores antigos)

---

## S3 — cachyos-init: hardware knowledge + tuning audit + emulator config

### O que absorver

#### A. Conhecimento de hardware (documentação + checks de doctor)

| Conhecimento | Origem | Destino |
|---|---|---|
| Limitações AVX2 (Sandy Bridge x86-64-v2) | `cachyos-init/README.md` + `fase-07-emu/` | `docs/guides/cachyos-gaming.md` (seção Hardware) |
| ReBAR/PCIe 2.0 (~5% na RX 580) | `cachyos-init/fase-08-sistema/README.md` | `docs/guides/cachyos-gaming.md` (seção Hardware) |
| Limites de emulação (RPCS3/PS3 inviável, Switch AAA slideshow) | `cachyos-init/fase-07-emu/eden-retroarch.md` | `docs/guides/cachyos-gaming.md` (seção Emuladores) |

#### B. Tuning privilegiado (audit no doctor, não auto-fix)

| Tuning | Origem | Check no doctor |
|---|---|---|
| `mitigations=off` | `fase-02-boot/README.md` | Já é auditado (INFO, nunca WARN) — **não muda** |
| `amdgpu.ppfeaturemask=0xffffffff` | `fase-02-boot/README.md` | Removido do guia (decisão do dono) — **não volta** |
| LACT fan curve | `fase-04-gpu/README.md` | Novo check INFO: `/etc/lact/config.yaml` existe |
| `MESA_SHADER_CACHE_MAX_SIZE=12G` | `fase-04-gpu/README.md` | Já é auditado — **não muda** |
| `RADV_PERFTEST=gpl` | `fase-04-gpu/README.md` | Já é auditado — **não muda** |
| `scx_loader` config | `fase-03-cpu/README.md` | Novo check INFO: `/etc/scx_loader/config.toml` existe |
| `kwinrc` compositing | `fase-06-kde/README.md` | Novo check INFO: `[Compositing]` no kwinrc |

#### C. Config de emuladores (documentação + checks de doctor)

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

### Decisões de design

1. **Hardware knowledge → documentação**: o conhecimento de AVX2, ReBAR, PCIe 2.0 e limites de emulação vai para `docs/guides/cachyos-gaming.md` como seção "Hardware". Não é check de doctor (é contexto, não drift).
2. **Tuning privilegiado → checks INFO**: LACT fan curve, scx_loader config, kwinrc compositing são auditados como INFO (nunca WARN). O dono decide se aplica ou não.
3. **Config de emuladores → checks INFO**: as configs dos emuladores são auditadas como INFO. O dono decide se aplica ou não.
4. **Nenhum auto-fix**: nenhum check novo é corrigido por `doctor --fix`. O tuning privilegiado continua manual.

### Arquivos afetados

- `docs/guides/cachyos-gaming.md` (modificar — seção Hardware + seção Emuladores)
- `internal/usecase/doctor_audit.go` (modificar — novos checks INFO)
- `internal/usecase/doctor_audit_test.go` (modificar — testes dos novos checks)
- `docs/os-and-agent-matrix.md` (modificar — lista de checks)
- `CHANGELOG.md` (modificar — nota sob `[Unreleased]`)

### Riscos

| Risco | Prob. | Impacto | Mitigação |
|---|---|---|---|
| Checks INFO de config de emuladores gerar ruído | baixa | doctor mais verbose | Checks são INFO (verde no renderizador), não WARN. |
| Config de emuladores divergir entre versões | média | check falso positivo | Checks são INFO, não WARN. O dono decide se atualiza. |

### Unknowns

| # | Pergunta | Bloqueia? | Dono | Prazo |
|---|---|---|---|---|
| U1 | O dono quer checks INFO para config de emuladores ou prefere manter só documentação? | sim | dono | antes de implementar |
| U2 | O dono quer que o envctl gerencie o LACT fan curve ou prefere manter manual? | sim | dono | antes de implementar |

### Rollback

- `git checkout docs/guides/cachyos-gaming.md internal/usecase/doctor_audit.go`
- Os checks INFO não afetam o comportamento do sistema

---

## Definition of Done

- [ ] S1: `debloat.yaml` tem os novos tweaks de Tier 3 e `tweaks_manager.go` suporta tipo `Binary` e phase `onedrive`
- [ ] S1: `go build ./...`, `go vet ./...`, `test ./...`, `golangci-lint` passam
- [ ] S1: `envctl doctor` mostra os novos checks de debloat
- [ ] S2: `performance_ubuntu.yaml` tem `container_limits` por tier e `provision_performance.go` aplica os limites
- [ ] S2: `go build ./...`, `go vet ./...`, `go test ./...`, `golangci-lint` passam
- [ ] S3: `docs/guides/cachyos-gaming.md` tem seção Hardware + seção Emuladores atualizada
- [ ] S3: `doctor_audit.go` tem os novos checks INFO (LACT, scx_loader, kwinrc, configs de emuladores)
- [ ] S3: `go build ./...`, `go vet ./...`, `go test ./...`, `golangci-lint` passam
- [ ] S3: `envctl doctor` mostra os novos checks INFO
- [ ] Docs atualizadas: `manifests.md`, `os-and-agent-matrix.md`, `windows-debloat-tier3.md`, `ubuntu-server-baseline.md`
- [ ] `CHANGELOG.md` com notas sob `[Unreleased]`
- [ ] Diff sem segredos e sem arquivo fora de escopo
- [ ] Outra pessoa reproduz o resultado com os comandos acima

## Execução

Duas specs independentes, cada uma em seu own worktree:

```
git worktree add .worktrees/feat-windows-tier3 -b feat/windows-tier3-absorption
git worktree add .worktrees/feat-cachyos-hardware -b feat/cachyos-hardware-tuning
```

Ordem recomendada: S1 (mais simples) → S3 (mais complexo). Cada spec tem seu próprio gate de verificação.

## Log de execução

| Spec | Status | Commit | Resultado |
|---|---|---|---|
| S1 | **concluído** | `b3b5d51`, `08a98df`, `a20825a` (branch `feat/windows-tier3-absorption`) | Tier 3 absorvido: Binary + Command + Onedrive; 94→99 tweaks; validado ao vivo no notebook Windows 11 |
| S3 | **concluído** | `c2085bc`..`723153f` (branch `feat/cachyos-hardware-tuning`) | hardware doc + 3 checks INFO privilegiados + 9 checks INFO emuladores + full restore (AVX2, Eden, LACT, cmdline, scx, kwinrc); doctor 232 checks |

## Log de execução S3 (2026-09-30)

Branch `feat/cachyos-hardware-tuning`, worktree `.worktrees/feat-cachyos-hardware`.
Validação real no host descrito pelo cachyos-init (i7-2600 + RX 580): todos os
12 arquivos alvo existem e os 12 checks saem `✔ OK` no `envctl doctor`.

| Tarefa | Commit | Resultado |
|---|---|---|
| T1 | `c2085bc` | seção Hardware no guia (AVX2, ReBAR/PCIe 2.0, teto de emulação, BIOS) |
| T2 | `c2085bc` | 3 checks INFO: `lact-config`, `scx-loader-config`, `kwinrc-compositing` + 5 testes |
| T3 | `18a489f` | 9 checks INFO de emuladores + tabela + `configHasKey` + 3 testes (incl. paridade com a máquina real) |
| T4 | `04a0c2c` | guia Verificação, matriz, CHANGELOG `[Unreleased]` |
| T5 | — | `go build`/`vet`/`test` (13 pacotes) + `golangci-lint --new-from-rev` 0 issues; `envctl doctor` 231 checks, único WARN é `REFERENCE.md` pré-existente do main |

### Desvios do plano original (S3)

1. **O kwinrc é lido via `fsManager.ReadFile` com `~` literal, não `os.ReadFile`**
   — é arquivo do user profile; o path literal casa com o mock sem expansão
   (mesmo padrão do `gamingConfPath`).
2. **PPSSPP/azahar/eden usam fragmento de linha real** (`GraphicsBackend = 3`,
   `graphics_api=2`, `backend=1`) medido no arquivo do host, não o valor
   "ideal" da spec — o test de paridade `TestGamingEmulatorConfigs_CoverRealMachine`
   é o guarda contra drift.
3. **Cemu `<api>1</api>` casa em XML com várias `<api>` tags** — o fragmento
   exato desambigua do `<api>3</api>` (segundo GPU).
