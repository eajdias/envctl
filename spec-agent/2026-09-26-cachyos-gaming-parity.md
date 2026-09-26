# CachyOS gaming parity — fechar o delta entre o ambiente real e o provisionado

> Data: 2026-09-26 · Escopo: **só gaming/performance** (decisão do dono) ·
> Origem: comparação com `/home/eadiasold/Projetos/publico/cachyos-init`

## Objetivo

`envctl` visa replicar o ambiente CachyOS atual do dono. O `envctl doctor` está
verde (208/208, 0 WARN, 0 ERROR) e **nenhum** pacote de `gaming.yaml` falta na
máquina — mas o inverso não é verdade: **6 pacotes instalados não estão
declarados**, e a camada de tuning que o dono aplicou à mão é **auditada em 3 de
10 pontos**. Numa máquina nova, `run gaming` entrega ananicy **sem regras** e
nenhum dos 6 params de GPU que a máquina usa hoje.

## Premissa descartada (evidência)

Os itens originalmente apontados como faltantes **já existem** em
`manifests/gaming.yaml` e já estão instalados — não há trabalho a fazer neles:

| Item | `gaming.yaml` | Máquina |
|---|---|---|
| `heroic-games-launcher` | :43 | 2.22.3-2 |
| `lutris` | :49 | 0.5.22-2 |
| `sunshine` | :55 | 2026.922.203725-1 |
| `umu-launcher` | :155 | 1.4.3-1 |
| `wine-cachyos-opt` | :161 | 2:10.0.20260425-1 |
| `goverlay` | :193 | 1.9.2-1 |
| `xorg-server` | :225 | 21.1.24-1 |
| `plasma-x11-session` | :219 | 6.7.5-1 |
| `xf86-input-libinput` | :231 | — |
| `skyscraper-git` | :238 | instalado |
| `hactool` | :244 | instalado |

O bug do `xorg-server` já está fechado: `gaming.yaml:218` documenta
*"plasma-x11-session does NOT pull xorg-server — install together"* e o trio
X11 está declarado.

**Falso positivo descartado:** `dolphin` (CachyOS 26.08.1-1) **não** é o emulador
— é o gerenciador de arquivos do KDE (623 arquivos, `libdolphinprivate.so.26.08.1`,
`dolphin_update_splitviewsettings`). `dolphin-emu` (`gaming.yaml:62`) é o
emulador e é o dono de `/usr/bin/dolphin-emu`. Não há conflito a resolver.

## Delta real

### A. Pacotes instalados e não declarados

> **Correção 2026-09-26:** a primeira-passagem deste spec afirmou que os 8
> candidatos eram "leaf, `Required By:` vazio". **Isso estava errado** — o
> `pacman -Qi` deste sistema imprime labels em **português**
> (`Necessário para`, `Motivo da instalação`), e o parsing procurou
> `Required By` / `Install Reason`, casando vazio em todos. A tabela abaixo é a
> de `Required By` **real**, e ela muda a conclusão: 1 dos 8 não é gap nenhum e
> 4 precisam de pais declarados ou filhos declarados — decisão registrada em T1.

| Pacote | `Required By` real | Gap? |
|---|---|---|
| `cachyos-settings` | `None` (explícito) | **sim** — e é o pai de `cachyos-ananicy-rules` |
| `cachyos-ananicy-rules` | `cachyos-settings` | **sim**, mas o pai não é declarado |
| `protontricks` | `cachyos-gaming-meta` | **sim**, mas o pai não é declarado |
| `retroarch-assets-ozone` | `None` (explícito) | **sim** |
| `retroarch-assets-xmb` | `retroarch-assets-ozone` | **sim**, e o pai também não é declarado |
| `xf86-video-amdgpu` | `None` (explícito) | **sim** |
| `steam-devices` | `steam` | **não** — `steam` já é declarado em `gaming.yaml:11` |
| `scx-manager` | `cachyos-kernel-manager` | não — ver T1, rejeitado |
| `linux-firmware-amdgpu` | `linux-firmware` | não neste escopo — ver T1 e follow-up |

`cachyos-settings` é a peça central: possui `/usr/bin/game-performance` (o
wrapper que o **próprio guia** manda usar nos launch options do Steam), os
defaults em `/usr/lib/modprobe.d/`, `/usr/lib/sysctl.d/`,
`/usr/lib/modules-load.d/ntsync.conf`, e é o pai do ruleset do ananicy. O
envctl não declara nenhum arquivo dele.

### B. Camada de tuning: 10 params no cmdline, 3 auditados

`/proc/cmdline` real:
`mitigations=off preempt=full split_lock_detect=off amdgpu.runpm=0 amdgpu.aspm=0 pcie_aspm=off amdgpu.gpu_recovery=0 oops=panic panic=10 zswap.enabled=0`

| Situação | Params |
|---|---|
| já auditados | `preempt=full`, `split_lock_detect=off`, `zswap.enabled=0` |
| na máquina, sem guia e sem auditoria | `amdgpu.runpm=0`, `amdgpu.aspm=0`, `pcie_aspm=off`, `amdgpu.gpu_recovery=0` |
| choice deliberada, não é perf | `oops=panic`, `panic=10` |
| `mitigations=off` | fora de propósito por decisão (`doctor_audit.go:1057`) — **não muda** |

### C. Auditoria incompleta do preset

`auditGamingTuning` testa só `MESA_SHADER_CACHE_MAX_SIZE=` em
`~/.config/environment.d/gaming.conf` (`doctor_audit.go:1236`). Não testa
`RADV_PERFTEST=gpl` nem a existência do preset `MangoHud.conf`.

## Impacto e contratos

- **Superfície alterada:** `manifests/gaming.yaml` (38 → 44 entradas);
  `internal/usecase/doctor_audit.go` (`auditGamingTuning` + 2 helpers puros);
  `internal/usecase/doctor_audit_test.go`; `internal/infra/embedded/manifest_repo_test.go`;
  `docs/guides/cachyos-gaming.md`; `docs/os-and-agent-matrix.md`;
  `CHANGELOG.md` (seção `[Unreleased]`, se houver).
- **Contrato preservado:** `doctor` permanece **read-only** para tuning
  privilegiado — nenhum check novo é corrigido por `--fix`. A lição de projeto
  `.opencode/memory/lessons.md:28` é a razão: root por senha via agente gera
  faillock, reboot invalida a verificação na mesma sessão, e
  `mitigations=off` é decisão de segurança caso a caso.
- **Contrato preservado:** `performance_cachyos.yaml` continua com
  `sysctls: []` (invariante dupla em `provision_performance.go:113` e
  `manifest_repo.go:234-235`). Nada nesta spec toca em sysctl.
- **Dependência:** nenhum pacote novo em Go. Os 6 pacotes são todos Arch/CachyOS
  e usam `type: pacman`.
- **Não-regressão:** `TestMissingCmdlineParams` (a asserção de que
  `mitigations=off` nunca é exigido) e
  `TestDoctorAudit_GamingStackSilentWhenSteamAbsent` (gate de opt-in) precisam
  continuar passando sem alteração de semântica.

## Riscos

| Risco | Prob. | Impacto | Mitigação |
|---|---|---|---|
| Tornar params de GPU obrigatórios numa máquina Intel/NVIDIA gera WARN eterno | alta | quebra o contrato 0 WARN | tier separado gated por presença de `/sys/module/amdgpu` |
| Exigir `oops=panic`/`panic=10` numa máquina sem eles vira WARN permanente | alta | idem | check **INFO**, nunca WARN (mesmo padrão de `doctor_linux_performance_test.go:19`) |
| Audit de `.pacnew` como WARN deixa o doctor não-verde (há 1 pendente: `/etc/limine-snapper-sync.conf.pacnew`) | média | contrato 0 WARN | check **INFO** com a lista de arquivos |
| `scx-manager` é GUI e o dono tem `scx_loader` por config | baixa | pacote inútil | marcado como opcional no `name`, igual ao tratamento de `goverlay` |
| Remover `ppfeaturemask` do guia perde uma Receive que talvez funcione em outra placa | média | doc | a remoção é do texto, não do `doctor`; a lição da placa fica no CHANGELOG |

## Unknowns

| # | Pergunta | Bloqueia? | Dono | Prazo |
|---|---|---|---|---|
| U1 | `oops=panic` + `panic=10` é escolha intencional de estabilidade ou resíduo de debug? | não — o check é INFO nos dois casos | dono | follow-up |
| U2 | ~~`libx86emu` e `libmgba` são órfãos removíveis?~~ **RESOLVIDO: não são.** `pacman -Rns` recusou — `Required By: hwinfo` e `Required By: mgba-qt`. T8 cancelada. | não | — | fechado |
| U3 | Os 2 perfis `X360-GameCube-P1/P2.ini` da máquina foram criados à mão e não versionados — devem virar artefato? | não — fora de escopo (decisão: só corrigir o texto) | dono | follow-up |

**Breaking changes: nenhum.** Nenhuma assinatura, flag, exit code, schema ou
nome de arquivo muda. `gaming.yaml` ganha entradas; o schema de `Package` já as
suporta. Compatibilidade verificada por `go test ./...` (a suíte do manifesto
embutido valida o parse de todos os YAML) e por `envctl doctor` continuar verde.

---

## Tarefa 1 — Declarar os 6 pacotes de runtime que faltam

**Arquivos:** `manifests/gaming.yaml` (modificar) ·
`internal/infra/embedded/manifest_repo_test.go` (modificar)

**Interfaces:** consome `entity.Package{ID,Name,Type,OS,Category}`; produz as
mesmas 44 entradas em `LoadGamingPackages()`. Nenhuma assinatura nova.

**Decisão pai-vs-filho.** Para 4 dos 6 o pacote "real" tem um pai no repo:

| Filho | Pai | Decisão |
|---|---|---|
| `cachyos-ananicy-rules` | `cachyos-settings` (leve: confs em `/usr/lib/{modprobe.d,sysctl.d,modules-load.d}` + `game-performance`) | declarar **o pai** — ele traz o ruleset, o `game-performance` que o guia usa e os defaults CachyOS |
| `retroarch-assets-xmb` | `retroarch-assets-ozone` (leve) | declarar **os dois** filhos, para o intent ficar auditável |
| `protontricks` | `cachyos-gaming-meta` (pesado: 23 deps, incl. `alsa-plugins`, `lib32-gtk3`, `openal`) | declarar **o filho** — o pai arrasta 23 pacotes por um binário acessório |
| `steam-devices` | `steam` (**já declarado** em `gaming.yaml:11`) | **nada a fazer** — já vem |

Rejeitados de propósito:
- `cachyos-kernel-manager` + `scx-manager` — stack de kernel com hooks; o
  `scx_loader` já é declarado e o `scx-manager` é só GUI. Não é o caminho.
- `linux-firmware-amdgpu` — `Required By: linux-firmware`, e `linux-firmware`
  **não está no grupo `base`** do Arch/CachyOS. É gap real, mas pertence a
  `packages.yaml` (categoria system), não a `gaming.yaml` → follow-up F1.
- `cachyos-gaming-meta` — 4 dos seus 23 deps já são declarados individualmente
  (`proton-cachyos-slr`, `umu-launcher`, `wine-cachyos-opt`, `vulkan-tools`).

**Por que o RED é um teste e não um `pacman -Q`:** o teste trava a paridade no
CI, onde não há máquina CachyOS. Sem ele, o delta volta em silêncio.

- [x] RED: em `manifest_repo_test.go`, adicionar
      `TestGamingManifestDeclaresRuntimeParity`. Carrega o `gaming.yaml`
      embutido real e falha se qualquer um dos 6 IDs estiver ausente, ou se
      houver ID duplicado, `name` vazio, ou `type` fora de `{pacman, paru}`.
- [x] `go test ./internal/infra/embedded/ -run TestGamingManifestDeclaresRuntimeParity`
      → **FAIL** com os 6 IDs ausentes.
- [x] GREEN: inserir as 6 entradas em `gaming.yaml`, no estilo existente
      (`# comment` + `id`/`name`/`type`/`os`/`category`):
      - `cachyos-settings` → seção sistema, com comentário explicando que traz
        `game-performance`, os defaults modprobe/sysctl/modules-load e o
        `cachyos-ananicy-rules`.
      - `cachyos-ananicy-rules` → seção CPU/scheduler, junto de `ananicy-cpp`
        (`:180`); o `name` do `ananicy-cpp` ganha a nota *"requires
        cachyos-ananicy-rules; the daemon alone ships zero rules"*.
      - `protontricks` → seção Proton/Wine.
      - `retroarch-assets-ozone` e `retroarch-assets-xmb` → seção emuladores,
        junto de `retroarch`.
      - `xf86-video-amdgpu` → seção sessão X11, junto do trio.
- [x] Mesmo comando → **PASS**; `grep -c '^  - id:' manifests/gaming.yaml` → `44`.
- [x] Verificação: `go test ./...`.
- [x] Rollback: `git checkout manifests/gaming.yaml` (YAML puro, sem efeito na máquina).
- [x] Commit: `feat(gaming): declare the 6 runtime packages the real stack needs`
      (conventional commit → minor no release-please).

**Resultado esperado:** `grep -c '^  - id:' manifests/gaming.yaml` → `44` e
`TestGamingManifestDeclaresRuntimeParity` verde.

## Follow-up F1 (fora deste escopo) — `linux-firmware` em `packages.yaml`

`linux-firmware` não está no grupo `base` do Arch/CachyOS, então uma máquina
genuinamente nova não o tem, e sem ele não há `linux-firmware-amdgpu`. O
cachyos-init trata firmware como "Base CachyOS" (Fase 1), não como gaming.
Candidato a `packages.yaml` com `category: system`, `os: arch,cachyos`.
**Não implementado nesta spec** — depende de decisão sobre incluir firmware
AMD/Intel genérico num tool de dev.

---

## Tarefa 2 — Auditar regras do ananicy (o gap mais caro)

**Arquivos:** `internal/usecase/doctor_audit.go` (modificar) ·
`internal/usecase/doctor_audit_test.go` (modificar)

**Interfaces:** novo helper puro
`ananicyRuleCount(rulesDir string) int`; consome o path de `/etc/ananicy.d`.
`auditGamingTuning` emite um check novo `System: "Gaming"`,
`Target: "ananicy-rules"`.

- [x] RED: `TestAnanicyRuleCount` — diretório com regras → contagem > 0;
      diretório vazio ou ausente → 0. E
      `TestDoctorAudit_GamingTuningWarnsWhenAnanicyHasNoRules` — mock com
      `ananicy-cpp` ativo e `/etc/ananicy.d` sem arquivos de regra →
      1 `DiagWarning` em `Target: "ananicy-rules"` **com `FixHint` não vazio**.
- [x] `go test ./internal/usecase/ -run 'TestAnanicyRuleCount|TestDoctorAudit_GamingTuningWarnsWhenAnanicyHasNoRules'`
      → **FAIL** (undefined: `ananicyRuleCount`).
- [x] GREEN: implementar `ananicyRuleCount` contando entradas com extensão
      `.rules` mais os arquivos `00-types.types` / `ananicy.conf`, e o check em
      `auditGamingTuning` logo depois do bloco de serviços:
      - 0 regras → `DiagWarning`, `FixHint: "install cachyos-ananicy-rules (the ananicy-cpp daemon ships without rules)"`.
      - > 0 → `DiagOK` com a contagem.
- [x] Mesmo comando → **PASS**.
- [x] Verificação: `go test ./internal/usecase/`.
- [x] Rollback: `git checkout internal/usecase/doctor_audit.go`.
- [x] Commit: `feat(doctor): warn when ananicy runs without a ruleset`

**Resultado esperado:** numa máquina com `ananicy-cpp` e sem
`cachyos-ananicy-rules`, o doctor acusa 1 WARN nomeando o pacote — o cenário
que hoje passa silencioso.

---

## Tarefa 3 — Auditar os params de GPU e de panic do cmdline

**Arquivos:** `internal/usecase/doctor_audit.go` (modificar) ·
`internal/usecase/doctor_audit_test.go` (modificar)

**Interfaces:** duas constantes novas ao lado de `gamingKernelParams`:
`gamingAMDKernelParams []string` e `gamingPanicParams []string`; novo helper
puro `amdgpuModulePresent(root string) bool` (mesmo padrão root-prefixed de
`internal/infra/performance/inspector.go`).

- [x] RED: três testes —
  1. `TestGamingAMDKernelParams_CoversRealCmdline`: o cmdline real da máquina
     (`amdgpu.runpm=0 amdgpu.aspm=0 pcie_aspm=off amdgpu.gpu_recovery=0`)
     não produz ausentes; um cmdline parcial produz exatamente os que faltam.
  2. `TestDoctorAudit_Gaming_TuningSkipsAMDKernelParamsWithoutAMDGpu`: sem
     `/sys/module/amdgpu`, nenhum check AMD é emitido.
  3. `TestDoctorAudit_Gaming_TuningPanicParamsAreInfoNotWarn`: com
     `panic=10` ausente, o check sai `DiagInfo` — **nunca** `DiagWarning`.
- [x] `go test ./internal/usecase/ -run 'TestGamingAMD|TestDoctorAudit_Gaming_Tuning(SkipsAMD|Panic)'`
      → **FAIL** (undefined).
- [x] GREEN:
  - `gamingAMDKernelParams = []string{"amdgpu.runpm=0", "amdgpu.aspm=0", "pcie_aspm=off", "amdgpu.gpu_recovery=0"}`
  - `gamingPanicParams = []string{"oops=panic", "panic=10"}`
  - check AMD só quando `amdgpuModulePresent` — ausente → **nenhum** diag
    (silêncio, não INFO: numa máquina NVIDIA é irrelevante).
  - check de panic sempre `DiagInfo`, porque é escolha de estabilidade
    deliberada, não pré-requisito de performance.
  - **Não** mexer em `gamingKernelParams` nem na exclusão de `mitigations=off`
    (preservar `TestMissingCmdlineParams`).
- [x] Mesmo comando → **PASS**; `go test ./internal/usecase/` sem regressão.
- [x] Verificação: `go test ./...`.
- [x] Rollback: `git checkout internal/usecase/doctor_audit.go`.
- [x] Commit: `feat(doctor): audit the AMD GPU and panic kernel params actually in use`

**Resultado esperado:** na máquina real os 4 params AMD e os 2 de panic passam
a ser reportados; numa máquina sem amdgpu nenhum diag novo aparece.

---

## Tarefa 4 — Fechar a auditoria dos presets de usuário

**Arquivos:** `internal/usecase/doctor_audit.go` (modificar) ·
`internal/usecase/doctor_audit_test.go` (modificar)

**Interfaces:** sem assinatura nova; reusa `uc.fsManager.Exists` /
`ReadFile` já usados pelo bloco do preset shader.

- [x] RED: `TestDoctorAudit_Gaming_TuningChecksBothShaderCacheVars` — arquivo
      com só `MESA_SHADER_CACHE_MAX_SIZE=12G` e sem `RADV_PERFTEST` gera
      `DiagWarning`; com os dois, `DiagOK`. E
      `TestDoctorAudit_Gaming_TuningChecksMangoHudPreset` —
      `~/.config/MangoHud/MangoHud.conf` ausente gera `DiagWarning` com
      `FixHint` apontando `run shell`.
- [x] `go test ./internal/usecase/ -run 'TestDoctorAudit_Gaming_TuningChecks(Both|MangoHud)'`
      → **FAIL**.
- [x] GREEN: trocar a checagem de "contém `MESA_SHADER_CACHE_MAX_SIZE=`" por
      duas chaves obrigatórias, e adicionar o check do preset MangoHud logo em
  seguida. Ambos com `FixHint: "run 'envctl run shell'"` (é o que instala os
  dois, via `seed_if_missing` em `shell.yaml:288-303`).
- [x] Mesmo comando → **PASS**.
- [x] Rollback: `git checkout internal/usecase/doctor_audit.go`.
- [x] Commit: `feat(doctor): audit RADV_PERFTEST and the MangoHud preset`

**Resultado esperado:** os dois presets de `run shell` passam a ser verificados;
remover o `RADV_PERFTEST` da máquina passa a acusar WARN.

---

## Tarefa 5 — Reportar `.pacnew` pendentes (INFO)

**Arquivos:** `internal/usecase/doctor_audit.go` (modificar) ·
`internal/usecase/doctor_audit_test.go` (modificar)

**Interfaces:** helper puro `pendingPacnewFiles(etcDir string) []string`.

- [x] RED: `TestPendingPacnewFiles` — diretório com `a.conf.pacnew` e
      `sub/b.conf.pacnew` retorna os dois; sem nenhum, retorno vazio.
- [x] `go test ./internal/usecase/ -run TestPendingPacnewFiles` → **FAIL**.
- [x] GREEN: varrer `/etc` (1 nível + `pacman.d`) por `*.pacnew` e emitir
      **um único** `DiagInfo` `System: "Performance"`,
      `Target: "pacnew"`, com a lista. `DiagInfo` e não `DiagWarning` porque
      há 1 pendente nesta máquina (`/etc/limine-snapper-sync.conf.pacnew`) e um
      WARN permanente quebraria o contrato 0 WARN/0 ERROR.
- [x] Mesmo comando → **PASS**; `go test ./...` verde.
- [x] Rollback: `git checkout internal/usecase/doctor_audit.go`.
- [x] Commit: `feat(doctor): report pending .pacnew files as info`

**Resultado esperado:** o doctor passa a listar o `.pacnew` pendente sem
introduzir WARN.

---

## Tarefa 6 — Corrigir os 2 drifts de `docs/guides/cachyos-gaming.md`

**Arquivos:** `docs/guides/cachyos-gaming.md` (modificar)

**Sem código, sem teste** — é doc. Verificação é `rg` (abaixo) mais a
revisão de que nenhum check do `doctor` passou a documentar algo que o guia
não menciona.

- [x] `amdgpu.ppfeaturemask=0xffffffff` (hoje em :18): **remover**. Decisão do
      dono — a máquina divergiu e o guia não pode mandar o que não está lá.
- [x] :33 e :46: remover *"10 perfis X360 embutidos (`profiles/`)"* e
      *"nesta skill"* — o repo não tem `profiles/` (a skill
      `cachyos-gaming-setup` foi removida em 2026-09-26) e a máquina tem 2
      perfis criados à mão, não 10. Escrever o que é verdade: os perfis não são
      versionados pelo envctl e são criados/carregados na GUI do Dolphin.
- [x] Adicionar os 6 params que a máquina usa e o guia não documenta
      (`amdgpu.runpm=0`, `amdgpu.aspm=0`, `pcie_aspm=off`,
      `amdgpu.gpu_recovery=0`, e `oops=panic panic=10` marcados como
      *escolha de estabilidade, opcional*). Sem isso o doctor passa a auditar
      algo que a doc não explica.
- [x] Documentar `cachyos-ananicy-rules` como parte do stack (o guia hoje
      menciona `ananicy-cpp` sem o ruleset) e `steam-devices` na parte de
      controles.
- [x] §Verificação (:54-55): atualizar a lista de checks do `doctor` para
      refletir T2-T5.
- [x] Verificação:
      `rg -n 'ppfeaturemask|profiles/|embutidos' docs/guides/cachyos-gaming.md`
      → nenhum resultado de `ppfeaturemask` nem de promessa de perfil embutido.
- [x] Rollback: `git checkout docs/guides/cachyos-gaming.md`.
- [x] Commit: `docs(gaming): drop ppfeaturemask, fix the controller-profile claim, document the real cmdline`

**Resultado esperado:** o guia descreve exatamente o que a máquina faz e o que
o doctor audita — nenhum dos dois afirma algo que o outro contradiz.

---

## Tarefa 7 — Sincronizar a documentação de contagem

**Arquivos:** `docs/os-and-agent-matrix.md` (modificar), `CHANGELOG.md` (modificar)

- [x] `docs/os-and-agent-matrix.md:162` diz *"gaming (**38 pkgs**)"* → 44, e a
      lista de checks do `doctor` seção Gaming recebe os novos.
- [x] Conferir `README.md` e `docs/manifests.md` por outras menções a "38".
- [x] `CHANGELOG.md`: nota breve sob `[Unreleased]` se a seção existir; caso
      contrário, **não** criar seção de versão (a seção de versão é do
      release-please, via conventional commits).
- [x] Verificação: `rg -n '\b38\b' README.md docs/` → nenhum contexto de
      gaming.
- [x] Commit: `docs: sync the gaming package count and the doctor check list`

**Resultado esperado:** nenhuma doc afirma 38 pacotes quando são 44.

---

## Tarefa 8 — ~~Remover os 2 órfãos~~ CANCELADA

O dono executou e o **pacman recusou**:

```
erro: a remoção de libx86emu quebra a dependência "libx86emu" necessária por hwinfo
erro: a remoção de libmgba  quebra a dependência "libmgba"  necessária por mgba-qt
```

Confirmado com `pacman -Qi`: `Required By: hwinfo` e `Required By: mgba-qt`,
ambos `Install Reason: dependency`. **Não são órfãos** — a leitura original
deste spec (baseada no parsing quebrado) estava errada e a tarefa morre aqui.
Nada a fazer; `mgba-qt` e `hwinfo` seguem íntegros.

Registrado como lição: a tentativa foi inofensiva (o pacman valida a
dependência antes de escrever), mas a origem do erro foi minha — parsear
`pacman -Qi` sem fixar `LC_ALL` num sistema com locale pt-BR.

---

## Tarefa 9 — Gate final (bloqueante)

- [x] `go build ./...`
- [x] `go vet ./...`
- [x] `go test ./...` — todas as suítes, incluindo
      `TestMissingCmdlineParams`, `TestProvisionPerformanceCachyOSDoesNotApplySysctl`
      e as novas de T1-T5.
- [x] `golangci-lint run --new-from-rev=origin/main` (ou `envctl-verify --git-push`,
      que é o gate local do projeto).
- [x] `go run ./cmd/envctl doctor` → **0 WARN, 0 ERROR** e um total de checks
      maior que 208 (os novos entram como `DiagInfo`/`DiagOK`).
- [x] `grep -c '^  - id:' manifests/gaming.yaml` → `44`.

---

## Definition of Done

- [x] `gaming.yaml` tem 44 entradas e `TestGamingManifestDeclaresRuntimeParity` trava a paridade.
- [x] Os 6 pacotes estão declarados com `type` correto e `os: arch,cachyos`.
- [x] `ananicy-cpp` sem ruleset gera WARN; com ruleset gera OK.
- [x] Params de GPU só são exigidos quando `amdgpu` está presente; params de panic nunca dão WARN.
- [x] `RADV_PERFTEST` e o preset MangoHud são auditados.
- [x] `.pacnew` pendente sai como INFO, sem introduzir WARN.
- [x] `mitigations=off` continua fora de qualquer check (asserção de `TestMissingCmdlineParams` intacta).
- [x] `performance_cachyos.yaml` continua com `sysctls: []`.
- [x] Nenhum check novo é corrigido por `doctor --fix`.
- [x] Guia sem `ppfeaturemask`, sem promessa de perfis embutidos, e documentando os params reais.
- [x] `docs/os-and-agent-matrix.md` sem "38 pkgs".
- [x] `go build`, `go vet`, `go test ./...`, `golangci-lint` e `envctl doctor` com evidência fresca.
- [x] Diff sem segredos e sem arquivo fora de escopo (`errors.log` untracked fica).
- [x] Outra pessoa reproduz o resultado com os comandos acima.

## Execuação

Worktree isolado (o branch atual `feat/debloat-tier2-expansion` é de outra
sessão — lição de memória 2026-09-22):

```
git worktree add .worktrees/feat-cachyos-gaming-parity -b feat/cachyos-gaming-parity
```

Ordem executada: T1 → T2 → T3 → T4 → T5 → T6 → T7 → T9. T8 cancelada.

---

## Log de execução (2026-09-26)

Branch `feat/cachyos-gaming-parity`, worktree `.worktrees/feat-cachyos-gaming-parity`,
base `39039b5`. Todas as tarefas acima foram executadas; os desvios do plano
estão aqui porque a spec original os descrevia errado.

| Tarefa | Commit | Resultado |
|---|---|---|
| T1 | `04685b3` | `gaming.yaml` 38 → 44, `TestGamingManifestDeclaresRuntimeParity` |
| T2 | `0958e37` | check `ananicy-rules` + 3 testes |
| T3 | `a922d5f` | cmdline em 3 tiers + 4 testes |
| T4 | `02a1970` | `RADV_PERFTEST` + preset MangoHud + 3 testes |
| T5 | `522ff13` | `.pacnew` em INFO + 2 testes |
| T6 | `3ad8c57` | 3 contradições do guia corrigidas |
| T7 | `73001d4` | matriz 38 → 44, CHANGELOG sob `[Unreleased]` |
| — | `00856b8` | lições em `.opencode/memory/lessons.md` |
| T8 | cancelada | `pacman -Rns` recusou: `hwinfo` e `mgba-qt` precisam dos dois |

### Desvios do plano original

1. **A T1 virou 6 pacotes, não 8, e um deles mudou de forma.** A spec
   classificava os 8 candidatos como "leaf, `Required By:` vazio". O
   `pacman -Qi` deste sistema imprime os rótulos em **português**
   (`Necessário para`); o parsing procurou `Required By` e casou vazio em 8/8.
   Com o dado real: `steam-devices` é dependência de `steam` (já declarado) e
   saiu da lista; `scx-manager` e `linux-firmware-amdgpu` foram rejeitados com
   motivo (pai pesado / escopo `packages.yaml`); e `cachyos-ananicy-rules` foi
   declarado junto com o **pai** `cachyos-settings`, que é o que traz o
   `game-performance` que o guia manda usar.

2. **A T2 não conta regras; ela checa um marcador.** A spec pedia
   `ananicyRuleCount` lendo o diretório. `FileSystemManager` não expõe
   `ReadDir`, e adicionar o método à interface tocaria a implementação real e
   todos os mocks — desproporcional para o ganho. O check usa `Exists` num
   arquivo que só o ruleset possui (`00-types.types`, confirmado com
   `pacman -Ql`), o que cobre o caso que importa (pacote instalado com as
   regras removidas) e reaproveita o padrão testável que o check de
   `/usr/bin/X` já usava.

3. **A T5 é INFO, como a spec previa, e por isso não quebra o contrato.**
   Existe 1 `.pacnew` nesta máquina; um WARN aqui violaria 0 WARN/0 ERROR.

4. **Um WARNING alheio apareceu no meio e foi resolvido na direção certa.**
   `doctor` acusou `~/.config/opencode/AGENTS.md` divergente. A prova de que não
   era desta sessão: mtime `10:06:19`, depois do último commit (`10:05:25`),
   ausente do `git status` desta worktree, e `configs/AGENTS.arch.md` limpo nas
   duas worktrees. O diff eram 2 regras novas (worktree = sessão movida, sessão
   paralela) que a fonte não tinha. Rodar `run shell` teria apagado as regras do
   dono; as regras foram copiadas para a **fonte** e o warning sumiu.

5. **`DiagInfo` renderiza como `✔ OK`.** O renderizador em
   `internal/ui/cli/doctor.go:20` só conhece WARN e FAIL, então qualquer `INFO`
   cai no verde. Convenção pré-existente do projeto (vale para `swap`, `zram`,
   `worktree`, `debloat`), não introduzida aqui, e mexer exigiria tocar
   `internal/ui/cli/run.go`, que a outra sessão está editando.

### Evidência do gate (T9)

```
go build ./...                    OK
go vet ./...                      OK
go test -count=1 ./...            13 pacotes, 0 FAIL
golangci-lint --new-from-rev      0 issues
envctl-verify --git-push          7 checks passed
envctl doctor                     219 checks, 219 passed, 0 WARN, 0 ERROR
grep -c '^  - id:' gaming.yaml    44
```

Os 6 checks novos, no doctor real: `ananicy-rules` (ruleset present),
`kernel cmdline (amdgpu)` (4 checked), `kernel cmdline (panic)` (present),
`shader cache preset` (as 2 chaves), `MangoHud preset` (present), `pacnew`
(1 pendente: `limine-snapper-sync.conf.pacnew`, em INFO). Os 6 pacotes novos,
todos `Installed`.

### Pendente fora do escopo

**F1** — `linux-firmware` não está no grupo `base` do Arch/CachyOS, então uma
máquina nova não o tem e sem ele não há `linux-firmware-amdgpu`. Candidato a
`packages.yaml`, categoria `system`. Não implementado: depende de decisão sobre
incluir firmware AMD/Intel genérico num tool de dev.
