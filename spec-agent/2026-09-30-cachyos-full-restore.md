# Spec — CachyOS full restore: provisionar tudo que hoje é só auditoria

> Data: 2026-09-30 · Escopo: fechar o gap do S3 — o `envctl` deve reconstruir o
> stack de gaming completo num PC formatado, com detecção de capacidade (AVX2)
> e **zero PII**. Extende `2026-09-30-cachyos-hardware-tuning.md` (auditoria INFO já feita).

## Decisões do dono (confirmadas)

1. **Tudo automatizado**: `run gaming` provisiona também o tuning manual atual
   (cmdline, LACT, scx_loader, kwinrc, configs de emuladores, Eden) — a
   finalidade do projeto é voltar ao "default" após formatar.
2. **Detecção de AVX2**: o envctl reconhece se o desktop tem AVX2 e configura
   corretamente (build do Eden legacy vs standard; INFO no doctor).
3. **Zero PII**: nada de BIOS/keys/ROMs/conta no repo — só templates genéricos
   e paths `~/Games/...`; dump de keys continua manual (legal, do console).
4. **Sudo interativo**: `sudo -v` no início do run; passos privilegiados usam
   `sudo -n`; se falhar, instrui `sudo envctl run gaming`. Nunca senha no código.
5. **Eden automatizado**: download do AppImage legacy pinnado (v0.2.1, URL
   oficial fixa), smoke test SIGILL, `.desktop`, config seed.
6. **Curva LACT embutida**: `40:0.2, 55:0.35, 65:0.55, 75:0.8, 85:1.0` vira o
   default provisionado, com detecção do device AMD real via `/sys/class/drm`.

## Gap atual (o que `run gaming` NÃO faz hoje)

| Item | Hoje | Alvo |
|---|---|---|
| Comfigs dos 9 emuladores (Vulkan, resolução, async) | só audita (INFO) | **seed** `configs/emulators/*` via `run shell` |
| kwinrc `[Compositing]` bypass | só audita | **merge de seção** (preserva o resto) com backup |
| `/etc/scx_loader/config.toml` | só audita | **escreve** via sudo (bpfland/Auto) |
| `/etc/lact/config.yaml` | só audita | **gera** via sudo com GPU detectada + curva |
| `/etc/default/limine` cmdline | só audita (WARN) | **edita** via sudo: params universais + AMD (se amdgpu) + panic; roda `limine-update`; avisa reboot |
| Eden AppImage | fora (manual) | **download** pinned + smoke SIGILL + `.desktop` + config |
| Capability AVX2 | não existe | **detecta** (`/proc/cpuinfo`) e informa no doctor |

## O que NÃO entra (mantido manual, sem PII)

- BIOS/firmware/keys dos emuladores (dump do próprio console — legal).
- ROMs (`~/Games/*` — conteúdo do dono; só a estrutura de pasta é seedada).
- Login/lançamento do Steam (conta), perfis de controle Dolphin (GUI).
- `mitigations=off` continua **fora** (decidido: decisão de segurança não é
  automática em nenhum host — asserção `TestMissingCmdlineParams` preservada).

## Impacto e contratos

- **Superfície:** `internal/usecase/gaming_tuning.go` (novo);
  `internal/usecase/gaming_tuning_test.go` (novo); `internal/usecase/doctor_audit.go`;
  `internal/usecase/doctor_audit_test.go`; `manifests/shell.yaml` (configs de
  emuladores); `configs/emulators/*` (novos templates); `manifests/gaming.yaml`
  (Eden seção comentada -> documentada como runtime); `internal/ui/cli/run.go`;
  `docs/guides/cachyos-gaming.md`; `docs/os-and-agent-matrix.md`; `CHANGELOG.md`.
- **Contratos preservados:** `doctor` remain read-only para tuning? **Não mais**
  — o `run gaming` agora aplica. O `doctor` continua read-only (nunca `--fix`
  para root); a aplicação é exclusiva do `run`. Checks INFO viram verificação
  de conteúdo (não só existência) quando o run provisionou.
- **Não-regressão:** `TestMissingCmdlineParams` (mitigations fora),
  `TestProvisionPerformanceCachyOSDoesNotApplySysctl` (sysctl continua `[]`),
  `TestGamingEmulatorConfigs_CoverRealMachine` (tabela de chaves).

## Riscos

| Risco | Prob. | Impacto | Mitigação |
|---|---|---|---|
| Editar `/etc/default/limine` errado quebra o boot | baixa | host não boota | Backup atômico `.bak.` + diff do KERNEL_CMDLINE antes/depois + só adiciona params faltantes (nunca remove); `limine-update` só se o arquivo mudou; instrução de fallback no menu Limine |
| Sudo -n expirado no meio do run | média | run parcial | `sudo -v` no início refresca o timestamp; cada passo privilegiado falha limpo com instrução |
| LACT GPU id não encontrado em host não-AMD | baixa | passo pulado | Passo AMD gated por detecção `/sys/class/drm/*/device` com vendor amd; ausente = skip silencioso (doc) |
| Curva LACT agressiva queima GPU em host desconhecido | baixa | hardware | Curva conservadora (tetos 85°C); documentada; `fan_control_enabled` explícito; performance_level auto |
| Eden download mudou de URL/hash | baixa | download falha | URL pinnada v0.2.1 (commit fixo no spec); falha limpa com instrução manual; nenhuma magia de "latest" |
| Seed de config de emu sobrescreve ajuste manual do usuário | média | perda de ajuste | `seed_if_missing: true` (só escreve se o arquivo não existe) — num PC formatado é exatamente o caso |

## Unknowns

| # | Pergunta | Bloqueia? | Dono | Prazo |
|---|---|---|---|---|
| U1 | Versão exata do Eden legacy a pinnar (`Eden-Linux-v0.2.1-legacy-gcc-standard.AppImage`, 66MB) confirmada? | não — v0.2.1 é a do host validado | dono | confirmação no smoke test |
| U2 | SHA256 do AppImage pinned para validação? | não — smoke SIGILL + existência bastam | dono | follow-up |
| U3 | `KERNEL_CMDLINE` no `/etc/default/limine` do host tem formato `KEY=value` único (não múltiplo)? | sim (formato do parse) | código mede no host antes | antes de T6 |

**Breaking changes:** nenhum de API. `run gaming` ganha passos (aditivo).

## Tarefas (ordem de dependência)

### T0 — Detecção de capacidade (AVX2 + AMD GPU)

**Arquivos:** `internal/usecase/hardware_capability.go` (novo) ·
`internal/usecase/hardware_capability_test.go` (novo)

**Interfaces:** `cpuHasAVX2(cpuinfo string) bool` (pure); `amdgpuDevices(sysfs string) []string` (lista de devices com vendor amd).

- [ ] RED: `TestCpuHasAVX2` (flags com/sem `avx2`; linha não-flags não conta) e
      `TestAmdgpuDevices` (sysfs fake com vendor 0x1002 → listado; vendor intel → vazio).
- [ ] GREEN: ler `/proc/cpuinfo` (injectável), detectar `avx2` na linha `flags`.
- [ ] Verificação: `go test ./internal/usecase/ -run 'TestCpuHasAVX2|TestAmdgpuDevices'`.
- [ ] Commit: `feat(gaming): detect CPU AVX2 and AMD GPU capability`

### T1 — Sudo scaffolding

**Arquivos:** `internal/usecase/gaming_tuning.go` (novo) · `internal/usecase/gaming_tuning_test.go` (novo)

**Interfaces:** `sudoAvailable() bool` (`exec.LookPath("sudo")`); `runPrivileged(ctx, args...) error` (roda `sudo -n`; erro explícito se não).

- [ ] RED: `TestRunPrivileged` com PATH fake → erro claro sem sudo.
- [ ] GREEN: `sudo -v` no início do `run gaming` (via run.go); helpers de exec.
- [ ] Commit: `feat(gaming): add sudo scaffolding for privileged tuning steps`

### T2 — Seed das configs de emuladores

**Arquivos:** `configs/emulators/{dolphin,retroarch,ppsspp,pcsx2,duckstation,azahar,eden,vita3k,cemu}.*` (novos) · `manifests/shell.yaml` (novos `config_files` com `seed_if_missing: true` e `os: arch,cachyos`)

**Conteúdo:** templates com os valores do host validado (tabela `gamingEmulatorConfigs` da tarefa T3 anterior — as MESMAS chaves). Paths `~/Games/<sistema>` genéricos. Sem BIOS/keys.

- [ ] RED: `TestShellManifestDeclaresEmulatorConfigs` (paridade com `gamingEmulatorConfigs` do doctor).
- [ ] GREEN: templates + entradas no shell.yaml.
- [ ] Verificação: `go test ./internal/infra/embedded/ -run TestShellManifestDeclaresEmulatorConfigs`.
- [ ] Commit: `feat(gaming): seed emulator configs as user templates`

### T3 — kwinrc merge de seção

**Arquivos:** `internal/usecase/gaming_tuning.go` · `internal/usecase/gaming_tuning_test.go`

**Interfaces:** `mergeKwinrcCompositing(data []byte) []byte` — garante seção `[Compositing]` e as 2 chaves, preservando o resto; idempotente (se já tem as 2, no-op).

- [ ] RED: `TestMergeKwinrcCompositing` (adiciona seção ausente; preserva outras seções; no-op quando pronto).
- [ ] GREEN: implementação + escrita com backup atômico via `WriteWithBackup` do user home.
- [ ] Commit: `feat(gaming): provision the kwinrc compositing bypass with section merge`

### T4 — scx_loader config via sudo

**Arquivos:** `internal/usecase/gaming_tuning.go` · `internal/usecase/gaming_tuning_test.go`

**Interfaces:** `scxLoaderConfig(data []byte) []byte` — `default_sched="scx_bpfland"` + `default_mode="Auto"`; já igual → no-op.

- [ ] RED: `TestScxLoaderConfig` (gera/estável).
- [ ] GREEN: escreve `/etc/scx_loader/config.toml` via sudo com backup atômico; `systemctl enable --now scx_loader` se inativo.
- [ ] Commit: `feat(gaming): provision the scx_loader scheduler config`

### T5 — LACT config via sudo com GPU detectada

**Arquivos:** `internal/usecase/gaming_tuning.go` · `internal/usecase/gaming_tuning_test.go`

**Interfaces:** `lactConfigFor(device string) []byte` — YAML com `fan_control_enabled`, curve conservadora, `performance_level: auto`.

- [ ] RED: `TestLactConfigFor` (YAML parseável, contém o device id).
- [ ] GREEN: detecta device AMD (T0), gera `/etc/lact/config.yaml` via sudo com backup, habilita `lactd`.
- [ ] Commit: `feat(gaming): provision the LACT fan curve with GPU detection`

### T6 — Kernel cmdline via sudo

**Arquivos:** `internal/usecase/gaming_tuning.go` · `internal/usecase/gaming_tuning_test.go`

**Interfaces:** `applyCmdlineParams(current, wanted []string) string` — adiciona só os faltantes; `cmdlineParamLine(...)` renderiza.

- [ ] RED: `TestApplyCmdlineParams` (já tem → no-op; falta → adiciona; preserva outros).
- [ ] GREEN: U3 medido no host → edita `/etc/default/limine` via sudo com backup, `limine-update` se mudou, avisa reboot.
- [ ] Commit: `feat(gaming): provision the kernel cmdline tuning with backup`

### T7 — Eden AppImage download

**Arquivos:** `internal/usecase/gaming_tuning.go` · `internal/usecase/gaming_tuning_test.go` · `configs/gaming/eden.desktop` (novo)

**Interfaces:** `edenURL(avx2 bool, version string) string` (pure, sem PII); download para `~/Games/switch/`? Não — `~/.local/bin/eden`? AppImage vai para `~/Applications/eden/`.

- [ ] RED: `TestEdenURL` (legacy vs standard pela AVX2).
- [ ] GREEN: download (HTTP via `http.Get` com timeout), `chmod +x`, SIGILL smoke (`eden --version` com timeout curto; exit 0 ou sem `Illegal instruction` = ok), `.desktop` seed, config seed (vem do T2), aviso de firmware/keys manuais.
- [ ] Commit: `feat(gaming): download and install the pinned Eden AppImage`

### T8 — Doctor: capability INFO + verificação de conteúdo

**Arquivos:** `internal/usecase/doctor_audit.go` · `internal/usecase/doctor_audit_test.go`

**Interfaces:** novo check INFO `System: "Gaming", Target: "cpu-capability"` (AVX2 presente/ausente — contexto p/ build do Eden e emuladores viáveis); checks existentes passam a verificar **conteúdo** onde o run provisiona (lact curve, scx bpfland, kwinrc keys — já é o caso), com INFO quando ausente.

- [ ] RED: `TestDoctorAudit_Gaming_TuningCpuCapability` (fake cpuinfo).
- [ ] GREEN: emitir INFO AVX2.
- [ ] Commit: `feat(doctor): report CPU capability as gaming context`

### T9 — run.go wiring + docs + gate

**Arquivos:** `internal/ui/cli/run.go` · `docs/guides/cachyos-gaming.md` · `docs/os-and-agent-matrix.md` · `CHANGELOG.md`

- [ ] `run gaming`: `sudo -v` → pacotes → passos T2-T7 (privilegiados com `sudo -n`) → aviso final (reboot + firmware/keys manuais).
- [ ] Docs: guia passa a dizer "provisionado" e lista os passos manuais restantes (BIOS/keys/ROMs/login Steam).
- [ ] Gate: `go build ./... && go vet ./... && go test ./... && golangci-lint run --new-from-rev=origin/main`.
- [ ] `envctl doctor` na máquina real: checks de conteúdo OK, 0 WARN novos, capability INFO presente.
- [ ] Commit: `feat(gaming): wire the full restore profile into run gaming` + `docs(gaming): full restore scope`

## Definition of Done

- [ ] Num PC formatado (CachyOS), `envctl run gaming` → pacotes + presets + configs de emuladores + kwinrc + scx_loader + LACT + cmdline + Eden, com `sudo -v` e zero prompts no meio.
- [ ] Com AVX2 ausente: Eden legacy baixado; com AVX2: build standard (URL correta).
- [ ] Sem AVX2 detectado, o doctor emite capability INFO e o guia explica o impacto.
- [ ] Zero PII: nenhum template contém BIOS/keys/conta; `rg -i 'bios|key|pass|senha|serial|account' configs/emulators/` → só referências genéricas de doc.
- [ ] `mitigations=off` continua fora (testes intactos).
- [ ] All gates verdes com evidência fresca; doctor 0 WARN novos na máquina do dono.
- [ ] Diff sem segredos e sem arquivo fora de escopo; reproduzível por outra pessoa.

## Execução

No worktree `.worktrees/feat-cachyos-hardware` (branch `feat/cachyos-hardware-tuning`),
em cima dos commits de auditoria já feitos. Ordem: T0 → T1 → T2 → T3 → T4 → T5 → T6 → T7 → T8 → T9.

## Log de execução

| Tarefa | Status | Commit | Resultado |
|---|---|---|---|
| T0 | pendente | — | — |
| T1 | pendente | — | — |
| T2 | pendente | — | — |
| T3 | pendente | — | — |
| T4 | pendente | — | — |
| T5 | pendente | — | — |
| T6 | pendente | — | — |
| T7 | pendente | — | — |
| T8 | pendente | — | — |
| T9 | pendente | — | — |