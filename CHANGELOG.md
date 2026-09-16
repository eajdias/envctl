# Changelog

Todas as alterações notáveis no projeto **`envctl`** serão documentadas neste arquivo.

O formato é baseado no [Keep a Changelog](https://keepachangelog.com/pt-BR/1.0.0/),
e este projeto adere ao [Versionamento Semântico](https://semver.org/lang/pt-BR/).

---

## [Unreleased]

### ⚡ AGENTS.md compacto, skills sob demanda e provisionamento por agente

- **Changed (preferência corrigida)**: as skills voltam a ser **carregadas sob demanda**. Removido o mandato "carregue a skill ANTES de agir" dos 4 manifestos: a economia de contexto é justamente o motivo de usar skill em vez de MCP — o que entra no prompt a cada turno é apenas *nome + descrição*, e o corpo do `SKILL.md` só é lido quando a tarefa casa.
- **Added**: **`envctl commandcode`** e **`envctl opencode`** — cada agente tem seu próprio comando e provisiona **só o que é dele**. `ProvisionShellUseCase.Execute(ctx, categories...)` filtra diretórios, arquivos de config e itens de cleanup por `category` (campo novo também em `RestrictedDir`), e as etapas machine-level (variáveis de ambiente, git config, npm do `~/package.json`) são puladas quando há filtro. Verificado nos dois sentidos: `envctl commandcode` não altera nenhum arquivo do opencode, e vice-versa.
- **Added**: **`SKILL-INDEX.md` por agente** (`configs/commandcode/SKILL-INDEX.md` → `~/.commandcode/SKILL-INDEX.md`; `configs/SKILL-INDEX.md` → `~/.config/opencode/SKILL-INDEX.md`). A tabela *situação → skill* saiu do AGENTS.md e virou um arquivo **consultado sob demanda** — o agente só o abre quando precisa escolher entre skills.
- **Changed**: `configs/commandcode/AGENTS.md` reescrito compacto — **12,3 KB → 3,7 KB** (a variante Linux: 11,3 KB → 4,2 KB), agora com: Ambiente, Regras, uma seção curta do CommandCode (config, agentes, MCP, skills on-demand, taste, hot reload) e **"Referências (leia só se precisar)"** apontando os arquivos externos. Os manifestos do opencode perderam o bloco/tabela de skills (−3,7 KB cada).
- **Changed**: `docs/skills.md` documenta a carga sob demanda, o índice externo e o deploy por agente; `README.md` ganhou os dois comandos novos; `docs/manifests.md` idem.

### 🎯 Skills auto-invocáveis, escopo por ambiente e AGENTS.md como roteador

- **Fixed**: `handoff` carregava `disable-model-invocation: true` — era a **única skill impossível de ser auto-invocada** (nunca entra no catálogo do modelo). Flag removida e a descrição agora diz quando usar.
- **Changed**: 12 descrições reescritas no padrão "Use quando… Triggers: …" com gatilhos em PT-BR (o usuário escreve em português): `ask-questions-if-underspecified`, `writing-plans`, `systematic-debugging`, `dispatching-parallel-agents`, `grilling`, `stop-slop`, `using-git-worktrees`, `verification-before-completion`, `receiving-code-review`, `skill-miner`, `skill-personalizer`, `skill-generalizer`. Antes eram frases curtas em inglês (72–252 chars) sem vocabulário de gatilho, justamente as que dirigem o matching automático. Hoje **40/40** skills têm gatilho explícito; o catálogo sobe de ~3.0k para ~3.9k tokens.
- **Added**: **escopo por ambiente** nas skills — campo `os` no `Skill` (manifesto), `Skill.AppliesToOS()` e filtro no `ProvisionSkillsUseCase`. `windows-admin` e `docker-desktop-wsl-restart` só vão para Windows; `cachyos-gaming-setup`, `aur-headless-install` e `headless-gui-probe` só para Linux. Uma instalação Windows deploya **37** skills (antes 40) e as 3 linux-only são **podadas** (deixam de existir no catálogo) — sem skill irrelevante consumindo contexto nem disparando na máquina errada.
- **Fixed**: o check por skill do `doctor` exigia o manifesto inteiro mesmo no OS errado (e incluía skills desabilitadas) — agora aplica o mesmo filtro de ambiente/habilitação. `doctor` volta a **152/152, 0 warnings** após a mudança.
- **Changed**: `configs/skills/docker` deixou de ser Windows-cêntrico no corpo (a skill é deployada também em VPS Linux): agora detecta o ambiente — Docker Desktop/WSL2 no Windows, daemon systemd no Linux — antes de agir.
- **Changed**: os 4 manifestos (`configs/AGENTS.md`, `configs/AGENTS.linux.md`, `configs/commandcode/AGENTS.md`, `configs/commandcode/AGENTS.linux.md`) trocaram o **catálogo por nome** por uma **tabela situação → skill** com mandato explícito ("as skills são o método padrão; carregue ANTES de agir; não pergunte 'quer que eu use a skill X?'"). Uma lista de nomes não dá ao modelo nada para casar com a tarefa; a tabela roteia por intenção, que é o que efetivamente dispara a invocação automática. Bloco idêntico nos 4 arquivos (mesma padronização cross-agente/ambiente), com marcadores `[win]`/`[linux]` para as skills escopadas. (**Revertido logo em seguida**: o mandato proativo contraria a economia de contexto — a tabela foi movida para `SKILL-INDEX.md`, consultado sob demanda; ver a entrada acima.)

### 🧹 Poda de skills estrangeiras/obsoletas + paridade de plataforma no CommandCode

- **Removed**: 3 skills — `implementation-strategy` e `docs-sync` (conteúdo do projeto **openai-agents-python**: `mkdocs.yml`, `docs/ja|ko|zh`, `src/agents/`, `$openai-knowledge`/OpenAI Docs MCP, e um script `find_latest_release_tag.sh` que não existe neste repo) e `grill-me` (corpo de 2 linhas que só re-invocava `grilling`, via nome de tool do opencode). Catálogo: 43 → **40 skills provisionadas** (+1 built-in do opencode).
- **Fixed**: skills com referências mortas — `docker` (seção do MCP `docker-hub`, removido do repo há versões, virou uma nota de que o MCP não existe), `vps-agent-dispatch` (`npm install -g @opencode-ai/cli` → `opencode-ai`, pacote correto; scratch `/tmp` → `/temp`), `memory-promotion` (banda DCP obsoleta 85/75 e menção a DCP), `handoff` e `lsp-smoke-test` (nome de tool/config parametrizados por agente).
- **Fixed**: seeds de memória — removida a entrada DCP 85/75 (superada pela 90/80), corrigido o valor no registro de plugins, removido o módulo Node `playwright` da lista de libs globais (browser automation é MCP-only), numeração do `doctor` corrigida (12.6/12.6.1 → 11.6/11.6.1) e movidas para fora do seed global **11 entradas de DOMÍNIO** (projeto/fornecedor), que por regra pertencem à memória do projeto. As cópias por máquina não são afetadas (`seed_if_missing`).
- **Added**: `configs/commandcode/AGENTS.linux.md` + filtro `os:` em `manifests/shell.yaml` — a VPS Linux recebia o manifesto **Windows** do CommandCode (PowerShell 7.6.5, `C:\temp`), enquanto o opencode já tinha variante Linux.
- **Changed**: `docs/skills.md` reescrito como catálogo completo das 40 skills (antes listava ~20) + tabela de paridade OpenCode ↔ CommandCode (skills, memória, LSP, context pruning, regras globais).
- **Changed**: `manifests/packages.yaml` — adicionadas `pypdf`, `python-docx` e `lxml` ao provisionamento pip do Windows; os docs (`AGENTS.md` e memória) já as listavam como disponíveis.
- **Changed**: `configs/AGENTS.md` — removidas as linhas das 3 skills e contagem ajustada (41 ativas = 40 provisionadas + 1 built-in).
- **Changed**: removido o número hardcoded de verificações do `doctor` dos docs ("160+") — a contagem é dinâmica (155 nesta máquina Windows, menor no Linux), então os docs passam a descrever o escopo em vez de um número que não se sustenta.
- **Changed**: o teste do manifesto agora exige que `manifests/skills.yaml` e os diretórios embutidos de `configs/skills/` batam exatamente (skill embutida e não declarada nunca é deployada; declarada sem diretório quebra o provisionamento).

### 🧩 Paridade CommandCode: config, metadata e validação no doctor

- **Fixed**: `configs/commandcode/settings.json` — removido o `$schema` (`https://commandcode.ai/schema/settings.json` responde **404** e não é chave do registry de settings); removidas as entradas de permissão mortas herdadas do opencode (`todowrite`, `todoread`, `task`, `skill` — no CommandCode as tools são `todo_write`, `task_create/update/list/get/stop` e `activate_skill`) e as no-op (`Read`, `WebFetch`, `WebSearch`, `Shell(go test:*)`, `Shell(go vet:*)`); adicionadas regras `deny` (`Shell(git push --force*)`, `Shell(git reset --hard*)`, `Shell(git clean -*)`, `Edit(.git/**)`) e `ask` para segredos (`Read(.env*)`, `Read(~/.commandcode/auth.json)`).
- **Fixed**: `configs/commandcode/AGENTS.md` — a afirmação "Config is NOT hot-reloaded: restart CommandCode after changes" era falsa (agents, skills e memória são re-lidos a cada turno; `settings.json` vale no próximo round; só um update staged exige `/reload`); catálogo de skills corrigido — faltavam `aur-headless-install`, `cachyos-gaming-setup` e `headless-gui-probe` (a contagem final do ciclo está na entrada de poda acima); adicionados `.agents/skills/` (compat), escopo local de MCP, precedência de MCP e o conjunto completo de campos do frontmatter de agente.
- **Fixed**: agente `code-reviewer` — removido o `maxTurns: 40` (o default documentado do CommandCode é 100; o cap interrompia revisões longas).
- **Fixed**: `manifests/shell.yaml` — novo cleanup `stale_commandcode_memory_dir` remove `~/.commandcode/memory` (sobra do provisionamento antigo; o CommandCode não tem memory-dir); descrição do `commandcode_mcp` atualizada para os 5 servidores reais.
- **Fixed**: skills que referenciam caminhos exclusivos do opencode (`~/.config/opencode/memory/`, `.opencode/memory/`, `opencode.json`) foram neutralizadas/parametrizadas por agente — `agent-memory`, `memory-promotion`, `ssh-vps`, `docker`, `lsp-smoke-test`, `windows-admin`, `subagent-routing` (corrigido também `plan`/`goal` como agentes do CommandCode — `goal` não existe) e `skill-generalizer/references/platform-compatibility.md` (nova linha do Command Code).
- **Changed**: removido `compatibility: opencode` de 13 `SKILL.md` — no CommandCode o campo significa **requisitos de ambiente**, então o valor era metadado enganoso (e a rubrica de portabilidade do projeto pede frontmatter conservador).
- **Changed**: `manifests/skills.yaml` — as 25 descrições placeholder `"OpenCode agent skill <name>"` viraram descrições reais; `snapshot_sync.go` passa a derivar a descrição do frontmatter do `SKILL.md` em vez de gerar o placeholder.
- **Added**: `internal/usecase/skill_frontmatter.go` — parser/validador de frontmatter (Agent Skills) compartilhado; o `doctor` agora valida o JSON de `settings.json`/`mcp.json`, o frontmatter de cada agente custom (nome == arquivo, nomes reservados ignorados) e o das skills implantadas (nome == diretório, descrição não vazia), além de comparar a contagem implantada com o manifesto.
- **Changed**: contagens normalizadas em `README.md` e `docs/principles.md`.
- **Changed**: `.gitignore` — o escopo de projeto do CommandCode passa a versionar os artefatos autorais (`skills/`, `commands/`, `agents/`); `settings.json`, `settings.local.json` e `taste/` seguem locais por máquina.

### 🔥 Trim do firecrawl + browser automation MCP-only (43 skills)

- **Removed**: suite firecrawl completa — 5 skills (`firecrawl`, `firecrawl-crawl`, `firecrawl-map`, `firecrawl-scrape`, `firecrawl-search`), pacote volta `firecrawl-cli`, step de bootstrap Linux e check do `doctor`.
- **Removed**: via CLI de browser — skill `playwright-cli`, pacote volta `@playwright/cli`, scripts `pw-screenshot`/`pw-eval` (+ wrappers, entradas no `shell.yaml`, auditorias `CLI-Scripts`/`Chromium`/`Playwright` no `doctor`, bloco de instalação do Chromium no `run shell`, bloco `references` no `opencode.json`/`opencode.linux.json`, dep `playwright` no `user-package.json`). Browser automation é exclusivamente via MCPs `playwright` + `chrome-devtools` (com o bundle do Chrome deles).
- **Changed**: contagens sincronizadas (43 provisionadas + 1 built-in) em `AGENTS.md`, `README.md`, `docs/skills.md`, `docs/principles.md`, `docs/doctor-and-idempotency.md` e teste de manifesto.
- **Changed**: skill `git-workflow` — em divergência com `origin`, o upstream prevalece (rebase + resolver a favor do remoto; nunca force-push de intent local).

### 🔗 Referências do repo apontam para o remoto público

- **Fixed**: `configs/AGENTS.md`, `configs/commandcode/AGENTS.md` e `configs/skills/vps-provisioning/SKILL.md` referenciavam o checkout local (`C:\projetos\git-privado\envctl`); a fonte da verdade agora é o remoto https://github.com/eajdias/envctl — as cópias locais são gerenciadas e edição direta nelas é sobrescrita pelo `envctl run shell`.
- **Changed**: `.gitignore` passa a ignorar `.commandcode/` (scratch local do agente CommandCode — permissões de sessão e paths absolutos da máquina, não versionável).

### 🧭 Agentes opencode: `plan` volta ao default (built-in) e agente `goal` removido

- **Changed**: agente `plan` — removido o `"mode": "all"` de `opencode.json`/`opencode.linux.json`; volta ao default do opencode (**primary built-in**, não dispatchável via task tool). Prompt ajustado.
- **Removed**: agente `goal` (autônomo YOLO) de `opencode.json`/`opencode.linux.json` e `configs/commandcode/agents/goal.md` (+ entrada `commandcode_agent_goal` em `manifests/shell.yaml`) — o agente `build` cobre o fluxo. O plugin `@prevalentware/opencode-goal-plugin` **permanece** (o `/goal` funciona a partir de qualquer agente, inclusive `build`).

### 🧭 Dispatch de subagentes: diretiva proativa + skill `subagent-routing`

- **Added**: skill `subagent-routing` — roteamento de subagentes (quando delegar, `explore` vs `general`, paralelo vs sequencial, quando NÃO delegar).
- **Changed**: `configs/AGENTS.md`, `configs/AGENTS.linux.md` e `configs/commandcode/AGENTS.md` ganharam as seções **"Uso Proativo de SSH, Context7 e Busca Web (OBRIGATÓRIO)"** e **"Delegação a Subagentes (uso proativo)"**, e o `Tone`/`Zero Tolerância` passam a exigir sinalizar defaults subótimos com trade-offs e fechar tarefas com evidência fresca.
- **Changed**: prompts dos agentes `review`/`plan` (`opencode.json`/`opencode.linux.json`) reforçados para dispatch proativo de `explore`/`general`, paralelizando domínios independentes na mesma resposta.

### 🧠 DCP (context pruning) priorizando cache-hit + compressão manual

- **Changed**: `configs/dcp.jsonc` — `experimental.allowSubAgents: true`; banda de compressão automática **90%/80%** e `nudgeFrequency: 10` / `iterationNudgeThreshold: 30` (menos compressões e menos injeções de nudge = **cache-hit melhor**); `manualMode` segue **desligado** (ligá-lo desativa a compressão autônoma).
- **Changed**: `configs/AGENTS.md` e `configs/AGENTS.linux.md` — diretiva para chamar a tool `compress` **proativamente** (troca brusca de assunto / fim de sub-tarefa).

### 🔌 Paridade opencode ↔ CommandCode (agentes + MCP + memory)

- **Fixed**: agentes custom do CommandCode — `review.md`/`plan.md` usavam **nomes reservados** (`explore`/`plan`/`review`/`general`) e eram **ignorados** pelo CommandCode; o frontmatter ainda usava tool ids inválidos (`read`/`webfetch`/`websearch`). Substituídos por **`code-reviewer.md`** (nome válido; tools `read_file`, `read_directory`, `grep`, `glob`, `web_search`, `web_fetch`, `shell_command`) — read-only, evidência `file:line`, severidades + veredito. `plan.md` removido (o built-in **Plan** do CommandCode cobre). O cleanup passa a remover os `.md` stale (`review`/`plan`/`goal`).
- **Removed**: diretório vestigial `~/.commandcode/memory` do `shell.yaml` (a memória do CommandCode é o `AGENTS.md`).
- **Changed**: `configs/commandcode/mcp.json` — adicionados `ssh-manager` e `zscan` (`enabled: false`) para paridade com o opencode; `configs/commandcode/AGENTS.md` documenta built-ins/nomes reservados, memory = `AGENTS.md` e ganha o catálogo de skills.

### 🧹 Poda de skills fora do manifesto + bun no bootstrap Linux

- **Added**: `ProvisionSkillsUseCase.Execute` agora **poda** diretórios de skill que saíram do manifesto (guarda contra manifesto vazio; ignora dot-dirs; retorna os nomes removidos, exibidos pelo CLI). Coberto por `provision_skills_test.go`.
- **Added**: `bun`/`bunx` no bootstrap Linux (`provision_bootstrap.go`) + auditoria no `doctor` — necessário para os MCPs `bunx` nas VPSs.
- **Changed**: MCPs de browser (`playwright`, `chrome-devtools`) padronizados em `bunx` com versão pinada e `enabled: false` nos 3 configs; `docs/`, `README.md` e a memória global (`configs/memory/*.md`) sincronizados (7 lições novas de ambiente).

---

## [v1.2.0] - 2026-09-06

### 🆕 CommandCode: provisionamento equivalente ao OpenCode

- **Added**: Suporte completo a CommandCode — o `envctl` agora provisiona **OpenCode E CommandCode** simultaneamente, com mesmas skills, MCPs equivalentes e configuração de agentes traduzida.
- **Added**: Config templates em `configs/commandcode/` — `settings.json`, `AGENTS.md`, `mcp.json`, e agentes (`review.md`, `plan.md`, `goal.md`).
- **Added**: Skills são deployadas para **ambos** `~/.config/opencode/skills/` e `~/.commandcode/skills/` (mesmo formato SKILL.md).
- **Added**: Entradas em `manifests/shell.yaml` — config files, diretórios e cleanup entries para CommandCode.
- **Added**: Instalação do CommandCode CLI no bootstrap Linux (`provision_bootstrap.go`).
- **Added**: Health checks do `doctor` para CommandCode.
- **Changed**: `configs/commandcode/AGENTS.md` — shell atualizado para PowerShell 7.6.5 (`pwsh.exe`) com regras nativas PowerShell.
- **Changed**: `configs/skills/firecrawl-monitor/SKILL.md` — descrição simplificada (remoção de formatação e termos redundantes).
- **Motivo**: CommandCode é um agente LLM alternativo ao OpenCode; o envctl agora provisiona ambos, permitindo ao usuário escolher qual usar.

---

## [v1.1.49] - 2026-09-06

### ⚡ Provisionamento CommandCode + correções menores

- **Added**: Provisionamento inicial do CommandCode (configs, agents, MCP) — precede a release v1.2.0 que consolida o suporte.
- **Fixed**: `configs/commandcode/AGENTS.md` — shell corrigido de `cmd.exe` para PowerShell 7.6.5.
- **Changed**: `configs/skills/firecrawl-monitor/SKILL.md` — descrição simplificada (remoção de formatação e termos redundantes).
- **Motivo**: relato de caracteres `�` no terminal/console do opencode e ambiente Windows; fix anterior (profile + chcp) não alcança processos spawnados sem profile.

---

## [v1.1.47] - 2026-09-03

### ⚡ Agentes sem "tool negada" + correções da auditoria (ARM64, snapshot, doctor)

- **Changed**: `configs/opencode.json`/`opencode.linux.json` — agentes `plan`/`review`:
  - `bash`: `"*": "deny"` → `"*": "ask"` (read-only auto; demais comandos — firecrawl CLI, node, python, docker — pedem aprovação em vez de erro duro "tool negada").
  - `task: "allow"` (dispatch de explore/general) + `"subagent_depth": 2` global (subagent pode despachar sub-subagentes).
  - Removido `permission.skill.firecrawl-*: deny` global e `references.envctl` (path local versionado).
- **Fixed** `bootstrap.sh:83`: `${AUTH_HEADER[@]}` vazio + `set -u` quebrava no bash 3.2 (macOS) → `${AUTH_HEADER[@]+"${AUTH_HEADER[@]}"}`.
- **Fixed** `provision_bootstrap.go`: installers de gh/delta/yq/Go agora detectam `uname -m` (amd64/arm64) — VPS ARM64 suportada; versão do Node derivada do `packages.yaml` (sem drift).
- **Fixed** `snapshot_sync.go`: git.yaml preserva keys curadas ausentes na máquina; configs só reescritos quando o conteúdo difere; symlinks de skills seguem o alvo; erros de escrita logados.
- **Fixed** `doctor_audit.go`: check de git worktree detecta git ausente (sem falso DiagOK); config files auditam conteúdo (drift vs fonte provisionada), exceto seeds locais.
- **Fixed** `run cleanup` agora executa o `TempHygieneUseCase` (dead code eliminado; FixHint corrigido); `envctl run <desconhecido>` sai com exit 1.
- **Fixed** `tweaks_manager.go`: comparação numérica robusta de DWORD (YAML int/float vs registry); `packages.yaml`: check_command pip usa `py -m pip show` (evita o stub do Microsoft Store).
- **Changed** `manifests/shell.yaml`: dir `~/.config/opencode/secrets` (strict_acl, context7.key por máquina — nunca versionado). `configs/AGENTS.md`: catálogo de skills corrigido (74).
- **Motivo**: falha reportada no modo Review ("task general está negado... não tenho bash para o firecrawl CLI") + auditoria do projeto contra docs oficiais do opencode.

---

## [v1.1.46] - 2026-09-03

### 🐛 Fix: pipes bloqueados no bash read-only dos agentes `plan`/`review`

- **Fixed**: `configs/opencode.json` e `configs/opencode.linux.json` — allow list bash dos agentes `plan`/`review` ampliada. O opencode valida **cada segmento** de pipe/`&&` separadamente e o pattern casa com o resource = prefixo de arity do comando (`permission/arity.ts`): `rg x | head -5` exigia `head*` na lista; `dotnet test*` nunca casava porque `dotnet` não está no dict de arity (resource = `dotnet`). Adicionados: `git grep*`, `grep*`, `head*`, `tail*`, `wc*`, `sort*`, `uniq*`, `awk*`, `sed*`, `dotnet*` (substitui `dotnet test*`) e filtros PowerShell `Select-Object*`, `Where-Object*`, `Select-String*`, `Measure-Object*`, `Sort-Object*`, `Group-Object*`, `Out-String*`, `Format-Table*`. Prompts atualizados (pipes/`&&` permitidos desde que cada segmento seja read-only).
- **Motivo**: falha reportada no modo Review — "O pipe não é permitido. Vou usar rg puro."

---

## [v1.1.45] - 2026-09-03

### 🐛 Fix: agente `review` anulado por `review.md` stale + seeds de memória sincronizados

- **Fixed**: `~/.config/opencode/agents/review.md` (criado em 2026-08-24, pré-v1.1.39) **sobrescrevia** a definição JSON do agente `review` — `bash: deny` total anulava o bash granular do v1.1.44 (validado via `opencode debug config`). Correções:
  - Prompt do `review` no `opencode.json`/`opencode.linux.json` enriquecido com a metodologia que vivia no `.md` (severidades BLOCKER/MAJOR/MINOR/NIT, formato de achado, VERDICT, lente de segurança obrigatória, YAGNI check, sem linguagem performativa) — nada se perde com a remoção do arquivo.
  - `CleanupItem` ganhou `recursive` (`models.go` + `provision_shell.go` com `os.RemoveAll`) e `manifests/shell.yaml` passou a remover `~/.config/opencode/agents` inteiro no provisioning (fonte única de agentes = `opencode.json`).
- **Changed**: seeds de memória (`configs/memory/lessons.md`, `patterns.md`) sincronizados com a memória global — 7 lições novas (09-02: sudo no MCP ssh-manager, heredoc PHP, timeout não mata script; 09-03: mode built-in primary, defaults permissivos, refs superpowers, `.md` sobrescreve JSON) + pattern de agente read-only; entrada do `review` atualizada (agora em `opencode.json`, sem `agents/*.md`).
- **Motivo**: questionamento do usuário sobre utilidade das memórias globais para o projeto revelou o seed defasado e o `review.md` stale conflitante.

---

## [v1.1.44] - 2026-09-03

### ⚡ Agentes `plan` e `review`: bash read-only granular + escrita de planos

- **Changed**: `configs/opencode.json` e `configs/opencode.linux.json` — agentes `plan` e `review`:
  - `plan` agora `mode: all` (primary via Tab E subagent dispatchável via task tool — antes era primary-only por herdar o built-in).
  - `bash` trocado de `deny` total para granular read-only: `git log/status/diff/show`, `rg`, `go test/vet`, `pytest`, `npm test`, `dotnet test`, `cargo test` liberados; resto negado (`"*": "deny"`).
  - `plan`: `edit` liberado apenas em `docs/superpowers/plans/**` (salvar o plano); `review`: `edit` segue negado.
  - `task` restrito a `explore`; `todowrite` e `skill` liberados; `temperature: 0.1`; `steps: 25`.
  - Prompts: carregam `agent-memory` e (plan) `writing-plans`; instruem uso de context7/webfetch; wording corrigido (permission error, não "tool inválida").
- **Changed**: `configs/skills/writing-plans/SKILL.md` — removidas referências a skills inexistentes do pacote superpowers (`subagent-driven-development`, `executing-plans`, `superpowers:using-git-worktrees`); handoff adaptado ao fluxo local (task tool com subagent `general` por tarefa ou execução inline com checkpoints).
- **Changed**: `configs/AGENTS.md` — documentados os 3 agentes customizados (review/plan/goal).
- **Motivo**: falhas no agente `plan` — bash negado impedia coleta de evidências (estado do repo, baseline de testes), edit negado impedia salvar o plano, e o handoff da skill apontava para skills não instaladas. Fontes: docs oficiais opencode (agents/permissions/tools/mcp-servers) + obra/superpowers.

---

## [v1.1.43] - 2026-08-29

### 🔧 MCP (playwright, chrome-devtools): bunx + versão pinada

- **Changed**: comandos dos MCPs locais trocados de `npx -y @latest` para `bunx <pkg>@<versão>` (`@playwright/mcp@0.0.79`, `chrome-devtools-mcp@1.8.0`).
- **Motivo**: (1) `@latest` re-resolve o pacote a cada spawn (start lento/instável); (2) npx roda em processo `node` — se o agente der `Stop-Process node` (ex.: liberar porta), os MCPs caem no meio da sessão e o opencode **não reconecta** (status `failed`, tools somem do toolset); (3) bunx roda em `bun.exe` — sobrevive a kill de node e sobe mais rápido. Validado com handshake MCP (initialize + tools/list) sob bun: 24 tools (playwright), 29 tools (chrome-devtools).

---

## [v1.1.42] - 2026-08-29

### 🐛 Fix: 20 skills invisíveis para o opencode (frontmatter YAML inválido)

- **Fixed**: 20 skills (`agent-memory`, `git-workflow`, `database-ops`, `docker`, `windows-admin`, `vps-provisioning`, etc.) tinham `description` como scalar pleno com `: ` (ex.: `Triggers: git, ...`) — YAML interpreta `: ` como separador de mapping → erro de parse → o opencode **descarta a skill silenciosamente**. Todas convertidas para block scalar (`description: >-`), texto preservado.
- **Impacto**: `agent-memory` (obrigatória pelo AGENTS.md) e outras nunca carregavam; agora as 73 skills são descobertas. Requer reiniciar o opencode.

---

## [v1.1.41] - 2026-08-29

### 🐛 Fix: Windows Terminal fechando instantaneamente (commandline UTF-8)

- **Fixed**: o `commandline` do perfil PowerShell no Windows Terminal (`pwsh -Command "chcp 65001 > $null; $env:PATH = $env:PATH"`) executava o comando e saía sem `-NoExit`, fechando a aba/janela na hora. Removido de `configs/terminal-settings.json` — o encoding UTF-8 já é garantido pelo profile PowerShell (`[Console]::OutputEncoding/InputEncoding = UTF8` + `chcp 65001`), única camada necessária no launch.
- **Nota**: a validação do v1.1.40 (doctor 181/181) não cobriu a abertura real de um novo terminal via Windows Terminal — o profile é carregado só após o launch, e o `pwsh -Command` sem `-NoExit` encerra o processo antes disso.

---

## [v1.1.40] - 2026-08-29

### 🌐 Console UTF-8: correção do encoding Unicode (U+FFFD) no Windows

- **Fixed**: Console Windows com code page OEM 850 renderizava caracteres Unicode (🚀 ✔ ✘ ⚠ 🎉 e glyphs Nerd Font) como `�` (U+FFFD) — corrigido em 3 camadas:
  - Profile PowerShell (`configs/Microsoft.PowerShell_profile.ps1`) agora define `[Console]::OutputEncoding/InputEncoding = UTF8` + `chcp 65001` no início.
  - Windows Terminal (`configs/terminal-settings.json`): perfil PowerShell com `commandline: pwsh -Command "chcp 65001 > $null; $env:PATH = $env:PATH"`.
  - Bootstrap Windows (`bootstrap.ps1`) força UTF-8 antes do banner.
- **Added**: `envctl doctor` ganhou o check 12.6 *Console Code Page* — alerta WARN se `chcp != 65001` (com fix hint: reiniciar o Windows Terminal / `envctl run shell`).
- **Validado**: `doctor` 181/181, 0 WARN, 0 ERRO com o profile ativo (antes: 1 WARN Code Page 850).

---

## [v1.1.39] - 2026-08-28

### 🤖 Agentes OpenCode: review read-only sem erros de tool + modo goal (YOLO)

- **Fixed**: Agente `review` e `plan` agora bloqueiam `bash`/`edit` via `permission` (deny) em vez de remoção do toolset — modelos menores paravam de gerar erros "unavailable tool" e recebem negação limpa que sabem interpretar.
- **Added**: Agente primário `goal` (modo autônomo YOLO) — executa spec/objetivo até a conclusão sem pedir permissão, rastreia via goal tools (set_goal/create_goal/update_goal/get_goal), verifica com evidências antes de concluir (ponytail + verification-before-completion + agent-memory).
- **Changed**: `configs/opencode.json` e `configs/opencode.linux.json` ganharam seção `agent` (review, plan, goal) — distribuída a todas as VPSs via provisioning.

---

## [v1.1.38] - 2026-08-28

### 🆕 Day-0: provisionamento de VPS fresca (validado em zscanchatbot)

- **Fixed**: `apt-get update` roda antes do primeiro `apt-get install` (uma vez por processo) — VPS novas têm listas apt vazias e todo install falhava com `Unable to locate package` (`internal/infra/apt/apt_manager.go`).
- **Fixed**: No Linux, o passo *bootstrap* (Volta/Node/OpenCode/Go/Rust) roda ANTES de *packages* — pacotes gerenciados via Volta eram pulados como "manager not available" em VPS fresca (`internal/ui/cli/run.go`).
- **Fixed**: Bootstrap instala `fd-find` via apt (fallback) quando ausente, para o symlink `fd` convergir mesmo com o bootstrap antes do passo apt (`internal/usecase/provision_bootstrap.go`).
- **Fixed**: Managers de toolchain (volta, npm, dotnet, go, rustup) resolvem binários no PATH de toolchain (`~/.volta/bin`, `~/.local/bin`, `~/.cargo/bin`, `/usr/local/go/bin`, `~/go/bin`) — elimina falsos positivos do `doctor` em shells não-login (ssh/systemd) (`internal/infra/toolchain/*`).
- **Validado**: VPS nova `zscanchatbot` (AWS Ubuntu) — bootstrap → `run all` → `doctor` = **167/167, 0 WARN, 0 ERRO** (antes: 26 WARN).

---

## [v1.1.0] - 2026-08-28

### 🤖 Integração envctl ↔ OpenCode (controle autônomo de VPS/VM)

- **Added**: Seção `VPS Infrastructure (envctl)` no AGENTS.md global — provisioner, bootstrap de 1 linha, comandos, inventário via `ssh-manager server list`, regra "NOVAS VM/VPS — SEMPRE usar envctl" (nunca configurar VPS manualmente).
- **Added**: Skill `vps-provisioning` (workflow Day-0/Day-2 de provisionamento remoto idempotente) — catálogo de skills 72→73.
- **Added**: Protocolo "Limites Free & Fallback de Token" na skill `vps-agent-dispatch` — limite Free estourado na VPS → perguntar TOKEN ao usuário; recusou → agente executa os comandos via SSH.
- **Added**: Passo 4 no cadastro de VPS da skill `ssh-vps` — provisionar a VPS com envctl logo após o registro (VPS vira agente OpenCode orquestrável).
- **Added**: Seção `Provisioning (envctl)` no `AGENTS.linux.md` (auto-manutenção nas VPSs provisionadas) e referência `envctl` no `opencode.json`.
- **Added**: Seção "Workflow Inteligente Multi-VPS (OpenCode)" no README.
- **Fixed**: Contador de progresso do `envctl run all` no Windows mostrava "6/5" — total agora é incondicional (6 seções).
- **Changed**: Documentação alinhada ao estado real — contagem de skills 59→73, LSPs 16→18, remoção de referências a `tui.json` (arquivo removido do provisioning).
- **Changed**: Versionamento de release promovido de `v1.0.<n>` para `v1.1.<n>`.

---

## [v1.0.33] - 2026-08-27

### 🔐 Segredos e PII fora do versionamento

- **Fixed**: `CONTEXT7_API_KEY` removida dos configs versionados — agora referenciada via `{file:~/.config/opencode/secrets/context7.key}` (variável de arquivo do opencode); chave antiga deve ser rotacionada no dashboard do Context7 (esteve no histórico do repo).
- **Fixed**: MCP `docker-hub` com caminho absoluto do usuário removido do config versionado.
- **Fixed**: Perfil PowerShell aponta para o tema Oh-My-Posh provisionado (`~/.poshthemes/`) via `$HOME` em vez de caminho absoluto com username.
- **Fixed**: Fixture de teste de higiene temp sem username real.

### 🛡️ Revisão multiplataforma: doctor 100% + hardening de provisionamento Linux

- **Fixed**: `$HOME/go/bin` incluído no PATH do toolchain Linux (`linuxToolchainEnv`/`ensureProcessToolchainPath`) — `go install` (ex.: `gopls`) agora resolve em processo, sem depender de novo shell.
- **Fixed**: Instalação do Go SDK remove `/usr/local/go` antes de extrair (guia oficial — evita stdlib órfã corrompendo builds).
- **Fixed**: Diretórios na raiz do Linux (ex.: `/temp`) criados via `sudo` com sticky `1777` em vez de falhar com `permission denied`.
- **Fixed**: Leitura de env vars POSIX prioriza rc files (fonte de verdade literal `$HOME`) sobre o processo (valor expandido pelo shell) — elimina WARN falso de `NODE_PATH` e torna `run shell` verdadeiramente idempotente.
- **Fixed**: `csharp`/`csharp-ls` restritos a Windows (dotnet-tool não provisionado no Linux).
- **Added**: LSP `toml` (Taplo via `@taplo/cli`) no `manifests/lsp.yaml` — era configurado no opencode.json mas nunca provisionado/auditado.
- **Added**: Auditoria do `doctor` para `go`, `rustup`, `cargo` e `rust-analyzer` (seção Linux Bootstrap).
- **Changed**: Hint de sudo NOPASSWD usa o usuário real (`$USER`) em vez de hardcoded `ubuntu`.
- **Resultado**: `envctl doctor` = **166/166 (Linux VM)** e **179/179 (Windows)** — 0 WARN, 0 ERRO; idempotência validada (2ª execução sem reinstalações).

---

## [v1.0.23] - 2026-08-26

### 🛡️ Otimização de Contexto e Conformidade de Skills (Context7 Analysis)

- **Changed**: Otimização de contexto via `permission.skill` com `firecrawl-*: deny` no `opencode.json` e `opencode.linux.json` — reduz ~2.5k tokens de listagem de descrições por prompt mantendo ferramentas disponíveis via CLI e AGENTS.md.
- **Added**: Metadados de portabilidade `license: MIT` e `compatibility: opencode` no frontmatter de todas as 12 skills nativas do `envctl`.
- **Fixed**: `go.mod` sanitizado via `go mod tidy` — `pterm`, `cobra` e `yaml.v3` devidamente promovidos a dependências diretas.

---

## [v1.0.22] - 2026-08-26

### 🧠 Pipeline Memória → Skill (autônomo) + 11 Skills Promovidas + Arsenal de Agentes

- **Added**: Skill `memory-promotion` — classificação obrigatória de memórias (processo reutilizável → skill; lição/preferência → memória), criação de `SKILL.md` no local certo, **remoção da entrada da memória após promoção** (sem redundância), registro via `envctl snapshot`.
- **Added**: Parâmetro OBRIGATÓRIO `Skill Promotion` no `AGENTS.md` global — classificar a cada gravação de memória; skills globais novas sem dados pessoais são implementadas no envctl.
- **Added**: 11 skills promovidas da memória global: `phone-e164-normalization`, `bulk-postgres-import`, `parallel-agent-orchestration`, `web-dashboard-automation`, `docker-build-local-vps-deploy`, `nextjs-standalone-deploy`, `docker-desktop-wsl-restart`, `jwt-hs256-node`, `simple-feature-flag`, `playwright-prod-regression`, `lsp-smoke-test` (72 skills totais).
- **Added**: Memória global (sem PII — auditada) versionada como templates seed `configs/memory/*.md` com novo campo `ConfigFile.SeedIfMissing` — máquinas novas nascem com a baseline; adições locais nunca sobrescritas nem reverse-sync.
- **Added**: Arsenal global de agentes (complemento do v1.0.21): `jq`, `dust`, `hyperfine`, `shellcheck` (winget), `pyyaml`, `requests`, `openpyxl`, `beautifulsoup4` (pip), `axios`, `cheerio`, `papaparse` (node via NODE_PATH); comando `envctl run pip`; reinstalação automática de deps do `~/package.json` por mtime.

---

## [v1.0.21] - 2026-08-26

### ⚡ Overhaul de Performance do OpenCode + Stack PWSH/WSL + Arsenal Global de Agentes

- **Removed**: MSYS2 completamente do ambiente Windows (manifestos, configs, `PacmanManager`, comando `run pacman`, terminal-settings, docs, skills, CHANGELOG) — stack padronizada: **PowerShell 7 (primário) + WSL Ubuntu (secundário)**.
- **Changed**: Config do OpenCode unificada em `opencode.json` (fonte única JSON) — merge de plugins/LSPs/MCPs/instructions; `shell: "pwsh"` (resolução via PATH, suporta MSI e Store); `opencode.jsonc` e `tui.json` (plugin TUI não-funcional) removidos com auto-cleanup no provisioning.
- **Changed**: DCP configurado para compressão auto **85% max / 75% min**.
- **Removed**: Plugins sem função comprovada — `opencode-visual-cache` (TUI cosmético), `@vymalo/opencode-models-info` (inerte: nenhum provider com `modelsInfoUrl`), `opencode-thinking-fix` (no-op: 300+ inspects com `isReasoningModel:false`). Restam 3: dcp, ponytail, goal-plugin (verificados funcionais).
- **Fixed**: Duplicidade de skills — `~/.agents/skills` removido (eram symlinks pendentes após deleção), fonte única `~/.config/opencode/skills`.
- **Added**: `envctl run cleanup` (configs legados, tool-output >10MB, scratch `C:\temp`/`/temp` >24h) + auditoria no `doctor` (DB, tool-output, legacy config, TempFolder `ENVCTL_TEMP`, WSL Ubuntu).
- **Added**: Pasta de scratch padronizada para agentes LLM — `C:\temp` (Windows) / `/temp` (Linux), env var `ENVCTL_TEMP`; `pw-screenshot` salva lá por padrão.
- **Added**: Arsenal global de agentes — `bun` (winget), `jq`, `dust`, `hyperfine`, `shellcheck`; pip global `pyyaml`, `requests`, `openpyxl`, `beautifulsoup4`; node `axios`, `cheerio`, `papaparse`; comando `envctl run pip`; reinstalação automática de deps do `~/package.json` por mtime.

---

## [v1.0.19] - 2026-08-21

### 🔒 Segurança: PII removida de skills + memórias individuais por máquina

- **Fixed**: Removida tabela de servidores (IPs, usuários, caminhos de chaves) e exemplos com IP real da skill `ssh-vps` — dados reais agora ficam APENAS em `~/.config/opencode/extras/ssh_servers.md` (inventário local por máquina, nunca versionado). A skill consulta dinamicamente (`ssh-manager server list` / MCP `ssh_list_servers`).
- **Changed**: Memórias globais do agente (`~/.config/opencode/memory/`) deixam de ser provisionadas via `manifests/shell.yaml` e não são mais reverse-syncadas no `envctl snapshot` — são individuais por PC/VPS (exclusão defensiva para `memory/` e `extras/` no `snapshot_sync.go`).
- **Added**: Diretório `~/.config/opencode/extras` no manifest (extras locais por máquina, ex: `ssh_servers.md`).
- **Changed**: Reforço do uso ativo da skill `agent-memory` no `AGENTS.md` (Windows/Linux) — LOAD obrigatório no início de toda tarefa para todos os agentes/subagentes; SAVE imediato ao aprender.
- **Removed**: `configs/memory/*.md` (templates de memória global) do repositório.

---

## [v1.0.18] - 2026-08-20

### 🧹 Higiene de Temp & Scratch (Windows/Linux)

- **Added**: Regra obrigatória de higiene de temp no `AGENTS.md` (Windows e Linux) — todo scratch/download/build/cópia de banco criado em `/tmp` deve ser removido antes do fim da sessão, com comando de limpeza documentado e recomendação de subdir dedicado por sessão.
- **Added**: Novo `TempHygieneUseCase` (`internal/usecase/temp_hygiene.go`) — auditoria do diretório temp (`C:\temp` no Windows, `/tmp` no Linux) e poda de artefatos obsoletos: extrações de módulos nativos do runtime Bun (`.bdef*.dll`/`.feef*.node`), `node-compile-cache`, `tsx-*`, scratch de sessões de agentes (`opencode/` com idade > 6h), downloads de ferramentas (`zscan-*`, `Meslo.zip`), caches regeneráveis (WinGet/NuGet/MSBuild/VS Code), logs de instaladores e arquivos soltos de sessões. Arquivos travados por processos em execução são pulados com aviso.
- **Added**: `envctl doctor` ganhou a checagem `TempHygiene` no relatório e `envctl doctor --fix` uma 6ª etapa de limpeza automática de temp (com resumo de artefatos removidos/liberados/pulados).
- **Fixed**: Alinhamento `gofmt` em `models.go`, `manifest_repo.go`, `env_manager.go` e `root.go` (pré-existente).

---

## [v1.0.17] - 2026-08-20

### 🐛 Correção de Bug
- **Fixed**: Comando `version` exibia versão hardcoded `v1.0.0` (em `internal/ui/cli/root.go`). O `-ldflags "-X main.Version=$VERSION"` apontava para uma variável que não existia, fazendo todos os releases reportarem `v1.0.0`.
- **Changed**: `main.Version` agora é uma variável injetável em `cmd/envctl/main.go` (default `dev`) e propagada ao CLI via `InitApp(embeddedFS, version)`; o comando `version` imprime a versão real do build. Validado: `-X main.Version=v1.0.16` → `envctl v1.0.16`; build sem ldflags → `envctl dev`.

---

## [v1.0.16] - 2026-08-20

### 🐛 Correções de Bugs (Auditoria Profunda do Codebase)
- **Fixed**: `AGENTS.md` global passou a ser implantado em `~/.config/opencode/AGENTS.md` (antes `~/AGENTS.md`, caminho que o opencode nunca lê); regra **Zero Tolerância** promovida para o global + checagem explícita no `doctor`.
- **Fixed**: Seção `directories:` do `shell.yaml` agora é funcional (`LoadDirectories`) — pastas `~/.ssh/sockets`, `~/.local/bin`, `~/.poshthemes` passaram a ser provisionadas; paths `C:/projetos/*` corrigidos.
- **Fixed**: Filtros `os:` em `shell.yaml` (env vars/config files/dirs), `git.yaml` (`core.fscache`/`core.longpaths` → windows), `lsp.yaml` (powershell/gopls/rust/csharp → windows) e `packages.yaml` (~35 pacotes winget/volta/dotnet-tool → windows) — elimina falsos warnings no Linux.
- **Fixed**: `doctor` git worktree não gera mais falso warning fora de repositório (`rev-parse --is-inside-work-tree`).
- **Fixed**: Variáveis de ambiente agora persistem no Linux (`~/.profile`/`~/.bashrc`) + escape de aspas em comandos PowerShell.
- **Fixed**: `GoManager.IsInstalled` usa `exec.LookPath` no POSIX (não `where.exe`); `NpmManager.ListInstalled` lista pacotes reais; `VoltaManager.IsInstalled` match exato por token.
- **Fixed**: Snapshot preserva metadados — `os:` do `git.yaml`, e merge das skills (target_dir/os/enabled/files/description); não exporta `~/.ssh/config`; guarda "nothing to commit".
- **Fixed**: Panic por `nil` em `tweaks_manager` (`cmd.ProcessState.ExitCode()`); backup/perms honram `StrictACL` (0600 + ACL); `%VAR%` indefinida mantém literal; `file_logger` reporta `GOOS/GOARCH` reais.

### 🧱 Replicação Cross-Platform nas VPS Ubuntu (validação em produção)
- **Added**: Novo `ProvisionBootstrapUseCase` (`envctl run bootstrap`) — instala Volta + Node 24 LTS + pnpm, OpenCode CLI (npm com fallback curl), `gh`, `delta`, `yq`, `uv`, `ruff`, `oh-my-posh`, `fd` (symlink `fdfind`), `pylsp` (via uv, contornando PEP 668), `firecrawl-cli` e `stylelint`; integração do Volta no PATH de shells de login.
- **Added**: Configs Linux dedicados — `configs/opencode.linux.jsonc` (sem shell/MCPs Windows), `configs/AGENTS.linux.md`, `configs/ssh-config.linux`; entradas condicionais por OS no `shell.yaml`.
- **Added**: `apt_manager` com `sudo -n` quando não-root; `playwright install-deps chromium` no Linux; seção 12 do `doctor` (toolchain Linux) + resolução de PATH de toolchain nos LSPs.
- **Changed**: `packages.yaml` ganhou `sshpass` (apt) e `user-package.json`/`.skill-lock.json` normalizados.
- **Validado**: ambiente opencode global replicado e verificado na VPS `<SERVER>` (doctor 124/124 no Linux).

### ⚙️ DCP — Limites Adaptativos
- **Changed**: `dcp.jsonc` usa limites percentuais `"90%"/"80%"` da janela do modelo (adaptativo a modelos de contexto grande, ex. 1M) em vez de valores fixos.

### 🧠 Nova Skill `agent-memory` (Memória de Lições & Padrões)
- **Added**: Skill `agent-memory` com fluxo LOAD → ACT → SAVE → REFLECT — o agente lê a memória antes de cada tarefa e grava lições/patterns ao aprender ou ser corrigido (estilo "Taste" do Command Code).
- **Added**: Arquivos de memória globais (`~/.config/opencode/memory/{lessons,patterns}.md`) e por projeto (`.opencode/memory/*.md`, versionáveis em PR) + regra obrigatória de leitura no `AGENTS.md` (Windows/Linux).

### 🔌 Skills & Integrações
- **Added**: Regra de ativação do MCP `ssh-manager` na skill `ssh-vps` — o agente pede ao usuário ativar via `/mcp` ou `Ctrl+P` (hot-reload) quando perceber que é ideal.
- **Fixed**: Skill `vps-agent-dispatch` usa o pacote npm correto `opencode-ai` (era `@opencode-ai/cli`).

---

## [v1.0.13] - 2026-08-18

### 📚 Documentação & Guias Multi-OS
- **Added**: Suíte completa de documentação modular sob `docs/`:
  - `docs/architecture.md`: Clean Architecture, camadas internas, abstração de I/O e binário standalone (`//go:embed`).
  - `docs/manifests.md`: Especificação declarativa dos arquivos YAML (`packages.yaml`, `shell.yaml`, `git.yaml`, `lsp.yaml`, `windows.yaml`).
  - `docs/skills.md`: Catálogo das 59 skills de IA, orquestração remota (`vps-agent-dispatch`) e automação Playwright.
  - `docs/doctor-and-idempotency.md`: 160+ checagens diagnósticas, flag `--fix`, backups atômicos (`.bak.timestamp`) e trilha de auditoria.
  - `docs/guides/windows.md`: Guia de provisionamento para Windows 11 PRO / Server.
  - `docs/guides/linux.md`: Guia de provisionamento para Ubuntu / Debian / VPS (AWS & Oracle).
  - `docs/guides/macos.md`: Guia de provisionamento para macOS Apple Silicon e Intel.
- **Changed**: `README.md` reescrito para ser enxuto, moderno, direto ao ponto, com quickstart de 1 linha e links diretos para a documentação técnica.

---

## [v1.0.12] - 2026-08-18

### 🌍 Expansão Cross-Platform & Orquestração de Subagentes
- **Added**: Renomeação do projeto de `win11-new` para `envctl` (`github.com/eajdias/envctl`).
- **Added**: `AptPackageManager` (`internal/infra/apt/`) com suporte a instalações não-interativas no Debian/Ubuntu Linux.
- **Added**: `bootstrap.sh` para instalação de 1 linha em sistemas Unix/Linux/macOS.
- **Added**: `vps-agent-dispatch` Agent Skill para orquestração de subagentes remotos via SSH, executando tarefas pesadas em servidores VPS (AWS/Oracle) e retornando sumários cristalizados.
- **Added**: Matrix multiplataforma no GitHub Actions (`.github/workflows/ci.yml` e `.github/workflows/release.yml`) compilando binários para Windows (amd64/arm64), Linux (amd64/arm64) e macOS (amd64/arm64).
- **Added**: ADR `0001-cross-platform-architecture.md` documentando a arquitetura multiplataforma.
- **Fixed**: Guards condicionais em `tweaks_manager.go` e `tweaks_manager_test.go` para evitar chamadas ao PowerShell no Linux durante testes de CI.
- **Fixed**: Padrões ancorados no `.gitignore` para não colidir com `cmd/envctl/main.go` ou `configs/bin/`.

---

## [v1.0.11] - 2026-08-18

### 🛠️ Auto-Remediação do Doctor & Ferramentas de Build
- **Added**: Flag `--fix` no comando `envctl doctor` para auto-remediação automática de qualquer inconsistência detectada nos 160+ pontos de checagem.
- **Added**: `Makefile` e `Taskfile.yml` com alvos padronizados (`build`, `test`, `doctor`, `doctor-fix`, `snapshot`, `install`).
- **Added**: Suporte a autocompletar shell gerado pelo Cobra (`envctl completion [bash|zsh|fish|powershell]`).

---

## [v1.0.10] - 2026-08-18

### 🎭 Utilitários Playwright & Resolução de Módulos
- **Added**: Variável `NODE_PATH` apontando para `%USERPROFILE%\node_modules` para garantir resolução global de módulos Node no Windows.
- **Added**: `configs/bin/pw-screenshot` e `configs/bin/pw-screenshot.cmd` para captura instantânea de telas headless em alta resolução.
- **Added**: `configs/bin/pw-eval` e `configs/bin/pw-eval.cmd` para avaliação rápida de DOM e scripts via Playwright Node.js API em < 1s.
- **Added**: Aliases de `git worktree` (`gwc`, `gwl`, `gwr`) e auditoria de integridade de worktrees no `doctor`.
- **Added**: Template robusto de `~/.ssh/config` com multiplexação de sockets (`ControlMaster auto`, `ControlPath ~/.ssh/sockets/%r@%h:%p`) e keepalive.

---

## [v1.0.7] - [v1.0.9] - 2026-08-18

### 🌐 Playwright Node API Migration
- **Changed**: Substituição da CLI instável `@playwright/cli` pela arquitetura estável Node.js API (`const { chromium } = require('playwright')`).
- **Added**: Resolução dinâmica de `NODE_PATH` no perfil do PowerShell e no shell WSL Ubuntu (`cygpath -m`).
- **Added**: `configs/user-package.json` gerenciando a dependência do `playwright` na raiz do usuário (`~`).
- **Added**: Aliases Docker preservando volumes e caminhos de containers sem conversão de caminhos POSIX.

---

## [v1.0.4] - [v1.0.6] - 2026-08-18

### 🪟 Windows 11 PRO Registry Tweaks & LSPs Nativos
- **Added**: `WindowsTweaksManager` (`internal/infra/windows/`) gerenciando Win32 Long Paths (`LongPathsEnabled = 1`), Developer Mode (`AllowDevelopmentWithoutDevLicense = 1`), Explorer show extensions (`HideFileExt = 0`), Explorer show hidden files (`Hidden = 1`) e Dark Theme.
- **Added**: Instalação e verificação automática da fonte `MesloLGM Nerd Font`.
- **Added**: `GoManager` (`go install ...@latest`) e `RustupManager` (`rustup component add ...`) gerenciando `gopls` e `rust-analyzer`.
- **Added**: `csharp-ls` via `.NET global tools` e `marksman` via Winget.
- **Added**: Templates de configuração de Terminal (`terminal-settings.json`), perfil do PowerShell (`Microsoft.PowerShell_profile.ps1`) com Oh-My-Posh e VSCode User Settings.

---

## [v1.0.1] - [v1.0.3] - 2026-08-18

### 📜 Logging Persistente & Gerenciador Volta
- **Added**: `FileLogger` (`internal/infra/logger/`) gravando sessões estruturadas em `~/.envctl/logs/envctl-YYYYMMDD-HHMMSS.log` com sanitização de null bytes de consoles UTF-16.
- **Added**: `VoltaPackageManager` (`internal/infra/toolchain/`) para gerenciamento declarativo de Node.js LTS e CLIs globais (`firecrawl-cli`, `pnpm`, `stylelint`, `sqllens-language-server`).
- **Added**: Expansão de `manifests/packages.yaml` incluindo ferramentas de produtividade (VSCode, Windows Terminal, Oh-My-Posh, Termius, WinSCP, 7-Zip, Everything, Brave Nightly, GitHub Desktop, WSL).

---

## [v1.0.0] - 2026-08-18

### 🚀 Lançamento Inicial (Protótipo win11-new)
- **Added**: Arquitetura base em Go (Clean Architecture) com camadas `domain`, `usecase`, `infra` e `ui`.
- **Added**: Binário 100% standalone via `//go:embed` embutindo manifestos declarativos YAML e templates de configuração.
- **Added**: Gerenciadores de infraestrutura para `Winget`, `Dotnet Tool`, `Git` e `FileSystem`.
- **Added**: Backup atômico com timestamp (`.bak.YYYYMMDD-HHMMSS`) para alterações em arquivos de configuração existentes.
- **Added**: Catálogo inicial de 57 skills de agentes de IA para o OpenCode.
- **Added**: Comandos CLI Cobra com interface ANSI via PTerm (`run`, `doctor`, `snapshot`, `version`).
