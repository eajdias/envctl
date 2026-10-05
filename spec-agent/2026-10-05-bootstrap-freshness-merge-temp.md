# Spec — Bootstrap confiável em máquina nova + memória CommandCode não-destrutiva

> Data: 2026-10-05 · Escopo: eliminar as 7 fricções observadas ao provisionar um notebook Windows 11 a partir do repo (`C:\projetos\publico\envctl`, HEAD `1f6992d`) + fazer o `~/.commandcode/AGENTS.md` agregar conteúdo local em vez de sobrescrever · Origem: sessão de atualização 2026-10-05 (logs `~/.envctl/logs/envctl-20261005-*.log`)

## 0. O que aconteceu (evidência)

Sessão de `git pull` + rebuild + `run all` + `opencode` v2.0.15→v2.0.23 + `doctor` num notebook já provisionado. Nada quebrou, mas 7 fricções custaram ~40 min de investigação manual — cada uma delas seria um bloqueador silencioso numa máquina **nova**:

| # | Fricção | Evidência | Numa máquina nova seria |
|---|---|---|---|
| F1 | Binário instalado **v1.6.0** vs repo **v1.12.0**; templates são `//go:embed` (`assets.go`), então provisionar com o binário velho aplica configs de 6 versões atrás | `envctl version` → `v1.6.0`; CHANGELOG HEAD → `v1.12.0` | provisionamento inteiro com manifests/configs/skills obsoletos, sem nenhum aviso |
| F2 | `envctl` **não está no PATH do usuário**: `Get-Command envctl` vazio; binário só em `%LOCALAPPDATA%\envctl\` (fora do PATH) e `~/.local/bin` vazio | `where envctl` → não reconhecido; `bootstrap.ps1` extrai para `$LOCALAPPDATA\envctl` sem tocar no PATH | todo comando da doc/skill (`envctl doctor`, `envctl run all`) falha com "não reconhecido" logo após o bootstrap |
| F3 | Fase 0 (`run providers`) **não atualiza o OpenCode dentro do major**: instalado v2.0.15, latest v2.0.23, e `ensureProviderCLI` cai no `default` ("update it with its own manager") sem checar minor/patch | `provision_providers.go:368-396` — só `requiredMajor` é verificado; `opencode -v` → `2.0.15` após `run all` verde | OpenCode defasado (8 minors) + `configs/package.json` pinando `@opencode/plugin 2.0.15` enquanto o binário poderia estar em outra minor — skew silencioso plugin↔CLI |
| F4 | `opencode upgrade --method curl` **quebra no Windows** (bash path mangling: `/bin/bash: C:Userseajdias-note.cacheopencodeupdate-...: No such file`) | saída real do comando nesta sessão | usuário que seguir o hint do doctor ("update it with its own manager") cai num erro opaco; o caminho que funciona (instalador PS oficial) não está documentado em lugar nenhum |
| F5 | `configs/package.json` (declaração de plugins do OpenCode) **pina a versão do plugin na release do CLI** (`@opencode/plugin 2.0.15`) sem vínculo mecânico com o binário instalado | `configs/package.json` + descrição no `shell.yaml:85` ("matching the opencode release line, validated via npm view" — validação manual, não teste) | plugin desatualizado vs CLI novo (ou vice-versa) → `LoadError` de plugin V2 que o doctor não detecta (só checa `node_modules` existente, `provision_shell.go` ~`OpenCodePlugins`) |
| F6 | `C:\temp` (ENVCTL_TEMP) acumula ~1 GB de lixo de terceiros (DockerDesktop, WinGet, Brave updaters, `.tmp`) que o `run cleanup` **não classifica** → WARN `TempFolder` permanente, `doctor` nunca chega a 0 WARN | `temp_hygiene.go:classifyTempEntry` — allowlist fechada, sem esses prefixos; doctor final: 142/143, WARN em `C:\temp` 1058 MB | "estado saudável = 0 WARN" inalcançável na prática; o operador aprende a ignorar WARN — e passa a ignorar os que importam |
| F7 | `~/.commandcode/AGENTS.md` é **overwrite puro**: `shell.yaml:116-138` sem `merge:`, sem `seed_if_missing`. Lições gravadas pela skill `agent-memory` (seções `## Erros / Lições`) são **apagadas no próximo run** | tentativa de gravar lições nesta sessão abortada após confirmar o overwrite; skill `agent-memory` manda gravar no AGENTS.md dos tiers | memória do CommandCode é write-only: grava-se e perde-se; comportamento diverge do OpenCode (que tem `memory/lessons.md` + `patterns.md` com `seed_if_missing`) |

Fricção operacional adicional (sem mudança de código, mas documentada aqui): o wrapper de background da sessão ficou preso 13+ min num pipe herdado **depois** do `run all` ter concluído (log com sessão "Closed" às 12:01:43). Diagnóstico: conferir `~/.envctl/logs/envctl-<ts>.log` antes de assumir falha. Vira lição de memória do repo (Tarefa 6), não código.

## Decisões do dono (a confirmar)

1. **F1:** `run all` deve **recusar ou avisar** quando o binário em execução for mais velho que X commits/tags atrás do repo clonado — ou o bootstrap deve sempre rebuildar? (Proposta: WARN no doctor + `run providers` sugerindo rebuild; nunca auto-rebuild silencioso.)
2. **F2:** Qual o install dir canônico no Windows — `%LOCALAPPDATA%\envctl` (atual, fora do PATH) ou `~/.local/bin` (no PATH, convenção do repo)? Proposta: migrar para `~/.local/bin` + shim de compatibilidade, ou persistir `%LOCALAPPDATA%\envctl` no PATH do usuário via `run shell`.
3. **F7:** Formato da agregação no `AGENTS.md` do CommandCode — **seções delimitadas** (`<!-- envctl:managed:start/end -->` + `<!-- envctl:user:start/end -->`, proposta deste SPEC) ou **arquivo sidecar** (`AGENTS.local.md` + referência)? Proposta: seções delimitadas (um arquivo só, o runtime lê tudo; sidecar exige convenção de include que o CommandCode não documenta).
4. **F6:** Ampliar a allowlist do `classifyTempEntry` para caches de terceiros (DockerDesktop, WinGet, Brave updaters) ou **rebaixar o check `TempFolder` para INFO** quando o resíduo for só de terceiros? Proposta: as duas — classificar o seguro + INFO com breakdown por dono em vez de WARN agregado.

## Objetivo

Qualquer máquina nova sai de `bootstrap` → `run all` → `doctor` **verde (0 WARN/0 ERROR)** sem intervenção manual, e conteúdo local gravado no `AGENTS.md` do CommandCode **sobrevive** a runs subsequentes.

## Impacto e contratos

- **Superfície alterada:** `internal/usecase/provision_providers.go` (+ teste), `internal/usecase/doctor_audit.go` (+ teste), `internal/usecase/config_merge.go` (+ teste), `internal/usecase/temp_hygiene.go` (+ teste), `internal/domain/entity/models.go`, `manifests/shell.yaml`, `configs/commandcode/AGENTS*.md` (3 arquivos, marcadores), `configs/package.json` (ver Tarefa 3), `configs/skills/envctl/SKILL.md`, `configs/skills/agent-memory/SKILL.md`, `configs/memory/lessons.md`, `docs/os-and-agent-matrix.md`, `docs/doctor-and-idempotency.md`, `bootstrap.ps1`, `README.md`, `CHANGELOG.md` (`[Unreleased]`).
- **Contrato preservado:** idempotência com backup atômico (`.bak.YYYYMMDD-HHMMSS`, `keep_newest`) em toda escrita; `TestLoadManifestsFromDiskOrEmbed` (embed = disco); `TestShippedOpenCodeTemplatesHaveNativeShape`; `TestShippedCommandCodeAgentTemplatesMatchSchema`; ` doctor` continua sem mutar sem `--fix`; `update` não toca gerenciadores de SO (spec 2026-09-26); debloat continua `INFO` nunca `--fix`; `merge:` continua opt-in por arquivo (default overwrite).
- **Contrato preservado:** `mergeSSHHosts`/`mergeJSONDeps` intocados; o novo modo é aditivo (`MergeMarkdownSections`), com o mesmo comportamento em erro: destino imparseável → WARN + preserva o arquivo do usuário (padrão de `mergeJSONDeps`, `provision_shell.go:247-261`).
- **Dependência:** nenhuma biblioteca nova em Go.
- **Não-regressão:** `go test ./...` + `golangci-lint run --new-from-rev=origin/main` + `envctl-verify --git-push` verdes; `envctl doctor` 0 WARN/0 ERROR na máquina de referência após as mudanças.

## Riscos

| Risco | Prob. | Impacto | Mitigação |
|---|---|---|---|
| Auto-update do OpenCode dentro do major quebrar sessão ativa (binário em uso) | média | CLI trava no meio do run | Fase 0 só atualiza o binário quando **nenhum** `opencode serve`/`service` está ativo; senão WARN com o comando manual. Serviço é parado antes só com flag explícita (`--restart-agents`), nunca por default |
| Marcadores `envctl:managed` colidirem com conteúdo do usuário | baixa | merge corrompe o arquivo | Marcadores em comentário HTML numa linha própria; merge é por split exato — sem marcador, cai no fallback seguro (Tarefa 4); teste com AGENTS.md sem marcadores |
| Mover o install dir do Windows quebrar atalhos/docs existentes | média | bootstrap antigo aponta para lugar vazio | Shim: o instalador novo deixa um `envctl.exe` stub no local antigo que delega (ou avisa); `bootstrap.ps1 -Force` reinstala no local novo |
| Versão latest do OpenCode indisponível (rede) no meio da fase 0 | baixa | run falha sem motivo | Mesma regra do `update`: latest `""` → linha "desconhecido", mantém instalado, nunca falha o run (spec 2026-09-26 §6) |
| Rebaixar TempFolder para INFO esconder vazamento real de scratch | baixa | disco enche sem aviso | INFO carrega breakdown por dono (top-5 entradas); WARN mantido quando o dono majoritário é `opencode`/agente (scratch próprio), só terceiros viram INFO |

## Unknowns

| # | Pergunta | Bloqueia? | Dono | Prazo |
|---|---|---|---|---|
| U1 | Install dir canônico no Windows: `~/.local/bin` ou PATH persistido para `%LOCALAPPDATA%\envctl`? (Decisão 2) | sim (Tarefa 1) | dono | antes de implementar |
| U2 | Agregação via seções delimitadas ou `AGENTS.local.md` sidecar? (Decisão 3) | sim (Tarefa 4) | dono | antes de implementar |
| U3 | WARN→INFO no TempFolder de terceiros, ampliar allowlist, ou os dois? (Decisão 4) | não (default = os dois, reversível) | dono | na revisão do SPEC |
| U4 | `run all` deve **falhar** ou só **avisar** com binário stale? (Decisão 1) | sim (Tarefa 2) | dono | antes de implementar |

**Breaking changes:** nenhum por default. Tarefa 4 muda a semântica de escrita do `AGENTS.md` do CommandCode (overwrite → merge) — é o ponto do pedido; rollback = remover o `merge:` do manifest. Tarefa 1 (install dir) só quebra quem chama o caminho absoluto antigo — coberto pelo shim.

---

## Tarefa 1 — envctl no PATH após o bootstrap (F2)

**Arquivos:** `bootstrap.ps1` (modificar) · `internal/usecase/doctor_audit.go` (modificar, novo check) · `internal/usecase/doctor_audit_test.go` (modificar) · `configs/skills/envctl/SKILL.md` (modificar, § "Quando algo falha")

**Interfaces:** consome o install dir efetivo; produz `envctl` resolvível no PATH do usuário + check `doctor` que acusa quando não está.

- [ ] RED: `TestDoctorReportsEnvctlNotOnPath` — simula PATH sem nenhum dir que contenha `envctl.exe` → espera `DiagWarning` (`System: "Envctl"`, `Target: "PATH"`, hint `run bootstrap.ps1 -Force` ou `run shell`). → **FAIL** (check não existe).
- [ ] `go test ./internal/usecase/ -run TestDoctorReportsEnvctlNotOnPath` → **FAIL**.
- [ ] GREEN: `bootstrap.ps1` persiste o install dir no PATH do usuário (`[Environment]::SetEnvironmentVariable('Path', "$p;$InstallDir", 'User')`, idempotente — só se `notlike "*$InstallDir*"`; mesmo padrão do instalador do OpenCode em `provision_providers.go:200-201`); novo check no doctor (`envctl` resolvível via `exec.LookPath`, respeitando o toolchain PATH do repo).
- [ ] Mesmo comando → **PASS**.
- [ ] Verificação: `go test ./internal/usecase/` + `bootstrap.ps1 -DryRun`-equivalente revisto à mão (não há harness PS; revisão + teste manual numa VM/sandbox).
- [ ] Rollback: `git checkout bootstrap.ps1 internal/usecase/doctor_audit.go`.
- [ ] Commit: `feat(bootstrap): persist install dir on user PATH and audit it in doctor`

**Resultado esperado:** máquina nova pós-bootstrap resolve `envctl` em qualquer shell; doctor acusa WARN se algo remover o dir do PATH.

## Tarefa 2 — Doctor acusa binário stale vs repo (F1)

**Arquivos:** `internal/usecase/doctor_audit.go` (modificar) · `internal/usecase/doctor_audit_test.go` (modificar) · `configs/skills/envctl/SKILL.md` (modificar, § "Provisionar uma máquina nova": rebuild antes do run)

**Interfaces:** consome `git -C <repo> describe --tags` (quando o checkout existir) vs versão embutida (`main.Version`, `cmd/envctl/main.go:10`); produz WARN com o comando de rebuild.

- [ ] RED: `TestDoctorWarnsOnStaleBinary` — version embutida `v1.6.0`, repo em `v1.12.0-9-g1f6992d` (fixture) → espera `DiagWarning` (`System: "Envctl"`, `Target: "Binary freshness"`, hint com `go build -ldflags "-X main.Version=$(git describe --tags --always)"`). Sem checkout do repo → sem check (não é erro, é ausência de fonte). → **FAIL**.
- [ ] `go test ./internal/usecase/ -run TestDoctorWarnsOnStaleBinary` → **FAIL**.
- [ ] GREEN: implementa a comparação (parse de `git describe --tags --always`; `dev` = sempre WARN quando houver checkout — build sem ldflags nunca é confiável para provisionar).
- [ ] Mesmo comando → **PASS**.
- [ ] Verificação: `go test ./internal/usecase/`.
- [ ] Rollback: `git checkout internal/usecase/doctor_audit.go`.
- [ ] Commit: `feat(doctor): warn when running binary predates the repo checkout`

**Resultado esperado:** `doctor` nunca fica verde com binário de 6 versões atrás; a skill manda rebuildar antes de `run all` (fecha F1 sem auto-rebuild silencioso — decisão U4 define se vira erro no futuro).

## Tarefa 3 — Fase 0 atualiza OpenCode dentro do major + pin plugin↔CLI (F3, F4, F5)

**Arquivos:** `internal/usecase/provision_providers.go` (modificar) · `internal/usecase/provision_providers_test.go` (modificar) · `manifests/shell.yaml` (modificar, descrição de `opencode_package_json`) · `configs/skills/envctl/SKILL.md` (modificar: caminho oficial de upgrade no Windows) · `docs/os-and-agent-matrix.md` (modificar, § Fase 0)

**Interfaces:** consome a latest oficial (`https://opencode.ai/update/api/latest/cli/npm`, já usada no `windowsInstaller`, `provision_providers.go:200`); produz binário atualizado intra-major + plugin alinhado.

- [ ] RED: `TestProvidersUpdatesOpenCodeWithinMajor` — instalado `2.0.15`, latest `2.0.23` (fixture HTTP) → espera upgrade via instalador oficial (não `default`/"update it with its own manager"). Serviço `opencode` ativo (fixture) → espera WARN + skip, sem matar processo. Latest indisponível (`""`) → OK "desconhecido", sem falha. → **FAIL** (hoje cai no `default`, `provision_providers.go:389-396`).
- [ ] `go test ./internal/usecase/ -run TestProvidersUpdatesOpenCodeWithinMajor` → **FAIL**.
- [ ] GREEN: novo case em `ensureProviderCLI` — `tool.binary == "opencode"`, major igual, minor/patch menor → roda o instalador oficial da plataforma (Windows: `windowsInstaller` já existente; Linux: `installer`/`pacman` conforme ownership — sem reinventar: reaproveita `installStandaloneProvider`); guarda: serviço ativo → WARN + hint (`opencode service stop` + `run providers`), nunca mata sozinho. No Windows o instalador baixa `opencode-windows-<arch>.zip` de `https://opencode.ai/files/bin/$ver/` para `~/.local/bin` com backup `.bak.<ts>` (comportamento validado nesta sessão: 2.0.15→2.0.23 OK).
- [ ] Mesmo comando → **PASS**.
- [ ] Documentar em `SKILL.md` + matriz: **`opencode upgrade --method curl` NÃO funciona no Windows** (bash path mangling — erro real desta sessão); caminho oficial = instalador PS (`~/.local/bin`, mesma URL da fase 0). No Linux, `opencode upgrade` funciona.
- [ ] Pin plugin↔CLI: `configs/package.json` passa a ser **gerado/validado** contra a minor instalada — no mínimo, teste `TestOpenCodePluginMatchesCLILine` que falha se `@opencode/plugin` divergir do major.minor do binário de referência documentado (ou: fase 0 reinstala `node_modules` de `~/.config/opencode` quando o CLI muda de minor — escolher na implementação; o SPEC trava só o invariante: **plugin major.minor == CLI major.minor**).
- [ ] Verificação: `go test ./internal/usecase/` + `opencode debug config` zero warnings após upgrade real.
- [ ] Rollback: `git checkout internal/usecase/provision_providers.go manifests/shell.yaml configs/package.json`.
- [ ] Commit: `feat(providers): update opencode within major and pin plugin line to CLI`

**Resultado esperado:** `run all` entrega OpenCode na latest intra-major (nunca rebaixa major sem decisão — `requiredMajor` continua valendo); plugin e CLI andam juntos; o operador Windows tem um caminho documentado que funciona.

## Tarefa 4 — `AGENTS.md` do CommandCode agrega em vez de sobrescrever (F7)

**Arquivos:** `internal/domain/entity/models.go` (modificar) · `internal/usecase/config_merge.go` (modificar) · `internal/usecase/config_merge_test.go` (modificar) · `internal/usecase/provision_shell.go` (modificar, novo case no switch) · `internal/usecase/doctor_audit.go` (modificar, auditoria do merge) · `manifests/shell.yaml` (modificar, `merge: markdown_sections` nos 3 `commandcode_agents_manifest*`) · `configs/commandcode/AGENTS.md` + `AGENTS.linux.md` + `AGENTS.arch.md` (modificar, marcadores) · `configs/skills/agent-memory/SKILL.md` (modificar, documentar as seções) · `docs/os-and-agent-matrix.md` (modificar, checklist "Arquivo de config" + §2)

**Interfaces:** consome `MergeMarkdownSections MergeMode = "markdown_sections"`; produz destino = template (seção gerenciada) + conteúdo do usuário (seções próprias) preservado byte a byte.

Design (espelha `mergeSSHHosts`/`mergeJSONDeps`, mesma filosofia — template vence no que é dele, usuário nunca perde o que é seu):

```
<!-- envctl:managed:start --> ...template... <!-- envctl:managed:end -->
<!-- envctl:user:start --> ...conteúdo local (lições, prefs)... <!-- envctl:user:end -->
```

- Template-fonte ganha os marcadores (as 3 variantes `AGENTS*.md`); o bloco gerenciado é tudo entre `managed:start/end`.
- Destino existente: extrai o que está **fora** do bloco gerenciado (ou tudo, se não houver marcadores — primeira migração) → preserva como bloco `user` após o gerenciado. Idempotente: re-run com destino já marcado = byte-identical (teste de round-trip duplo, padrão de `TestMergeSSHHostsIsIdempotent`).
- Destino inexistente: escreve template + bloco `user` vazio (com o header `## Erros / Lições` + `## Padrões / Preferências` da skill `agent-memory`, para o agente saber onde gravar).
- Destino imparseável/sem marcadores + conteúdo que não casa com o template antigo: **WARN + preserva** (fallback seguro, igual `mergeJSONDeps` em `provision_shell.go:247-261`) — nunca overwrite cego.
- Doctor: `ConfigFile` com `merge: markdown_sections` reporta `Present on disk (merged with user content)` em vez de drift (mesmo ramo de `Merge != MergeOverwrite`, `doctor_audit.go:213-217`).

- [ ] RED: `TestMergeMarkdownSectionsPreservesUserBlock` (template novo + destino com lições locais → gerenciado atualizado, lições intactas); `TestMergeMarkdownSectionsIsIdempotent` (merge(merge(x)) == merge(x)); `TestMergeMarkdownSectionsWithoutMarkersKeepsEverything` (destino legado sem marcadores → nada perdido, tudo vira bloco user). → **FAIL** (modo não existe).
- [ ] `go test ./internal/usecase/ -run TestMergeMarkdownSections` → **FAIL**.
- [ ] GREEN: `MergeMarkdownSections` em `models.go` + `mergeMarkdownSections()` em `config_merge.go` + case no switch de `provision_shell.go:243-246` + `merge: markdown_sections` nos 3 ids do `shell.yaml`.
- [ ] Mesmos comandos → **PASS**.
- [ ] `agent-memory/SKILL.md`: documenta que no CommandCode as seções vivem no bloco `user` do `AGENTS.md` (nunca editar o bloco `managed` — próximo run sobrescreve).
- [ ] Verificação: `go test ./internal/usecase/` + round-trip real: gravar lição fixture → `run shell` → lição presente + gerenciado atualizado; `doctor` sem WARN de drift no AGENTS.md.
- [ ] Rollback: `git checkout manifests/shell.yaml` (volta a overwrite) — conteúdo user já escrito permanece no arquivo (o rollback nunca apaga disco).
- [ ] Commit: `feat(commandcode): merge local sections into AGENTS.md instead of overwriting`

**Resultado esperado:** lições de `agent-memory` sobrevivem a `run all`; o template continua evoluindo no bloco gerenciado; o invariante "manifesto e código ganham da doc" continua valendo **dentro** do bloco gerenciado.

## Tarefa 5 — TempFolder sem WARN crônico (F6)

**Arquivos:** `internal/usecase/temp_hygiene.go` (modificar) · `internal/usecase/temp_hygiene_test.go` (modificar) · `internal/usecase/doctor_audit.go` (modificar, breakdown + INFO/WARN por dono) · `docs/doctor-and-idempotency.md` (modificar)

**Interfaces:** consome as entradas de `C:\temp` (`/temp`); produz classificação estendida + diagnóstico com top-5 por tamanho.

- [ ] RED: `TestClassifyTempEntryThirdPartyCaches` — `DockerDesktop/`, `DockerDesktopUpdates/`, `WinGet/`, `BraveComponentUpdater_*`, `*.tmp` órfãos antigos, `scoped_dir*` → `remove` ou `dono: terceiro` conforme decisão U3; `opencode/` ativo (< 6h) e `commandcode/` → nunca remove. → **FAIL** (hoje: `never`).
- [ ] `go test ./internal/usecase/ -run TestClassifyTempEntry` → **FAIL**.
- [ ] GREEN: estende `classifyTempEntry` (caches regeneráveis de terceiros: winget/docker/brave/vscode-updater = remove seguro; `.tmp`/`.tmp.lua` órfãos com idade > 24h = remove); doctor passa a reportar breakdown (top-5 entradas por MB + dono) e só WARN quando o majoritário é scratch próprio (`opencode/`, `commandcode/`, `node-compile-cache/`, `tsx-*`); resíduo só-de-terceiros = INFO.
- [ ] Mesmo comando → **PASS**.
- [ ] Verificação: `go test ./internal/usecase/` + `run cleanup` + `doctor` na máquina de referência.
- [ ] Rollback: `git checkout internal/usecase/temp_hygiene.go internal/usecase/doctor_audit.go`.
- [ ] Commit: `feat(temp): classify third-party caches and attribute TempFolder warn by owner`

**Resultado esperado:** `doctor` verde numa máquina real; WARN de temp volta a significar "seu scratch vazou", não "o Docker existe".

## Tarefa 6 — Memória do repo: lições desta sessão

**Arquivos:** `configs/memory/lessons.md` (modificar) · `configs/skills/envctl/SKILL.md` (modificar)

- [ ] Adicionar a `configs/memory/lessons.md` (formato ❌→✅, curto e acionável):
  - `opencode upgrade --method curl` no Windows → instalador PS oficial (`https://opencode.ai/files/bin/$ver/opencode-windows-x64.zip` → `~/.local/bin`, backup `.bak.<ts>`).
  - Task background "travada" após `run all` → conferir `~/.envctl/logs/envctl-<ts>.log` (sessão "Closed" = concluído; pipe herdado prende o wrapper — `kill_shell`).
  - `envctl` fora do PATH pós-bootstrap → caminho absoluto `%LOCALAPPDATA%\envctl\envctl.exe` até a Tarefa 1.
  - Build sem `-ldflags "-X main.Version=$(git describe --tags --always)"` = versão `dev` → templates embutidos sem rastreabilidade (relacionado à Tarefa 2).
- [ ] `SKILL.md` do envctl: § "Provisionar uma máquina nova" ganha a ordem **rebuild → run all → doctor** (com o comando de build) até a Tarefa 2 existir; § "Quando algo falha" ganha o passo "confira o log de execução antes de assumir falha".
- [ ] Verificação: `rg -n 'opencode-win|kill_shell|ldflags' configs/memory/lessons.md configs/skills/envctl/SKILL.md` → hits.
- [ ] Rollback: `git checkout configs/memory/lessons.md configs/skills/envctl/SKILL.md`.
- [ ] Commit: `docs(memory): session lessons — opencode upgrade on Windows, stuck wrapper, stale binary`

**Resultado esperado:** a próxima sessão não reinveste os 40 min desta.

## Tarefa 7 — Docs, matriz e gate final (bloqueante)

**Arquivos:** `docs/os-and-agent-matrix.md` · `docs/doctor-and-idempotency.md` · `README.md` · `CHANGELOG.md` (`[Unreleased]`)

- [ ] Matriz §1: linha `envctl` no PATH (Tarefa 1) + nota de freshness do binário (Tarefa 2); § Fase 0: update intra-major do OpenCode + caminho Windows (Tarefa 3); §2: `markdown_sections` no CommandCode (Tarefa 4); § TempFolder por dono (Tarefa 5).
- [ ] `doctor-and-idempotency.md`: novos checks (PATH, freshness), `markdown_sections` ≠ drift, TempFolder INFO/WARN por dono.
- [ ] Checklist "Arquivo de config" (matriz §5): regra nova — arquivo que o usuário edita **exige** `merge:` ou `seed_if_missing`; overwrite puro só para 100% gerenciado (fecha a classe do bug F7, não só a instância).
- [ ] `README.md`: quickstart de máquina nova = bootstrap → rebuild → `run all` → `doctor` (até Tarefas 1–2; depois simplifica).
- [ ] `CHANGELOG.md`: notas sob `[Unreleased]` (release-please gera a versão — nunca escrever seção de versão à mão).
- [ ] Gate: `go build ./...` · `go vet ./...` · `go test ./...` · `golangci-lint run --new-from-rev=origin/main` (ou `envctl-verify --git-push`) · `envctl.exe doctor` → **0 WARN, 0 ERROR** na máquina de referência · `opencode debug config` zero warnings.
- [ ] Rollback: `git checkout docs/ README.md CHANGELOG.md`.
- [ ] Commit: `docs(envctl): sync bootstrap-freshness-merge-temp contracts`

## Definition of Done

- [ ] Máquina Windows nova (ou VM/sandbox limpa): `bootstrap.ps1` → `envctl` resolvível no shell seguinte → `run all` → OpenCode latest intra-major com plugin alinhado → `doctor` **0 WARN / 0 ERROR**.
- [ ] `AGENTS.md` do CommandCode com lições locais → `run shell` → lições intactas + bloco gerenciado atualizado; `doctor` sem drift.
- [ ] `go build`, `go vet`, `go test ./...`, `golangci-lint` 0 issues — evidência fresca no PR.
- [ ] Docs e matriz refletem os 4 contratos novos (PATH, freshness, markdown_sections, temp por dono); checklist §5 proíbe overwrite em arquivo editável.
- [ ] `CHANGELOG.md` com notas sob `[Unreleased]`; sem seção de versão escrita à mão.
- [ ] Diff sem segredos/PII e sem arquivo fora de escopo; `TestLoadManifestsFromDiskOrEmbed` verde (embed = disco).
- [ ] Outra pessoa reproduz o resultado com os comandos do SPEC.

## Execução sugerida

Worktree isolado: `.worktrees/feat-bootstrap-freshness`, branch `feat/bootstrap-freshness-merge-temp`, base `origin/main`. Ordem: Tarefa 6 (memória, sem código, desbloqueia as outras) → Tarefa 4 (merge — muda semântica de escrita, quanto antes em teste melhor) → Tarefas 1–2 (bootstrap/PATH/freshness) → Tarefa 3 (providers) → Tarefa 5 (temp) → Tarefa 7 (docs + gate). Cada tarefa um commit convencional (`feat`/`fix`); merge da `chore(main): release` do release-please continua sendo o que shipa tag + binários.
