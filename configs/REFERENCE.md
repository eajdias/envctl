# OpenCode — referência operacional

**Consulte sob demanda.** Aqui fica o detalhe que **não** precisa estar no prompt a cada turno. Abra este arquivo quando for mexer em plugin, DCP, agentes, memória, VPS, ou quando precisar dos snippets operacionais.

## Plugins (1)

- `@prevalentware/opencode-goal-plugin` — modo goal: tools `get_goal`/`set_goal`/`update_goal` + slash `/goal`. Não cria toggle no Tab.
- REMOVED (2026-09-19, opencode v2): `@tarquinen/opencode-dcp@latest` and `@dietrichgebert/ponytail` fail to load (`PluginModule.LoadError: Expected object at ["default"]` — both export a V1 async function instead of `Plugin.define({id, setup})`; only the goal-plugin exports the V2 object shape). Re-add after upstream migrates per https://opencode.ai/v2/docs/build/plugins/migrate-v1. `dcp.jsonc` REMOVED from provisioning on 2026-09-22 (YAGNI: no consumer left — re-add config + plugin together on return).
- O array `plugins` vive **somente** em `opencode.json` (arrays não mesclam entre arquivos de config).
- Ao sugerir plugin novo: valide com `npm view <pkg>` antes de gravar (a maioria não é oficial/maduro) e só mantenha com evidência de funcionamento.

## Context pruning (nativo)

DCP removido em 2026-09-22 (plugin V1 quebra o boot do v2; `dcp.jsonc` era config sem consumidor — YAGNI). Pruning agora é o compaction nativo (`compaction.keep.tokens` + checkpoints); CommandCode usa `/compact` nativo. Chame `compress` proativamente ao trocar de assunto bruscamente ou ao concluir uma sub-tarefa cujo contexto verbatim não será mais usado.

## Agentes customizados

Definidos no `opencode.json` — **não** existe mais `~/.config/opencode/agents/` (o provisioning remove o diretório; um `.md` lá sobrescreveria o JSON silenciosamente).

- **`review`** (primary): revisão read-only — bash granular read-only para evidências, `edit` negado, `subagent: allow`. Severidades BLOCKER/MAJOR/MINOR/NIT (nit só se pedido), evidência obrigatória `file:line`, YAGNI check contra callers reais, veredito APPROVE/REQUEST-CHANGES.
- **`plan`** (built-in + 1 regra: `edit spec-agent/** allow` por merge — efetivo primary; **não** dispatchável via tool de subagente): planejamento read-only interativo; escrita apenas em `spec-agent/` na raiz do projeto (convenção do envctl — nada de pastas da skill upstream); carrega `writing-plans`, `agent-memory` e context7.
- **`planner`** (`mode: subagent`): variante **dispatchável** do planejamento, para isolar pesquisa/plano volumoso sem gastar contexto do coordenador. Mesmo boundary read-only (`edit` negado fora de `spec-agent/**`, sem subagentes aninhados), retorna só caminho do plano + resumo/risks/rollback/verificação. Regra de ouro: execução é **inline** por padrão; `planner` só quando o contexto bruto é volumoso e o retorno é compactável.
- **`reviewer`** (`mode: subagent`): o `review` **dispatchável**. Como `review` é primary, ele não aparece no catálogo de subagentes — este é o par do `\"Use the reviewer subagent to review my current changes\"`. Mesmo contrato de severidades e veredito, `edit`/`subagent` negados, bash read-only. É o que fecha a paridade com o `code-reviewer` do CommandCode.
- **`verifier`** (`mode: subagent`): roda o gate (`envctl-verify --git-push`, `go build/vet/test`, `golangci-lint`) e devolve **saída real**. Proibido consertar código, afrouxar threshold, pular teste ou adicionar ignore para fazer o gate passar; check que não rodou é reportado NOT RUN, nunca como pass. Fecha com `GATE GREEN | GATE RED | GATE NOT RUN`.
- **`docs-writer`** (`mode: subagent`): confere doc contra manifesto/código e corrige o drift. `edit * deny` + `docs/**` e `README.md` em allow, com `CHANGELOG.md` em deny explícito (o release-please é dono do changelog). Manifesto vence a doc; afirmação não verificada não entra.
- **`memory-keeper`** (`mode: subagent`): fecha a tarefa gravando lições/patterns. `edit * deny` + `**/memory/**` em allow. Classifica antes de gravar (lição vs pattern), projeto antes de global, dedupe, e nunca grava segredo.
- No CommandCode os mesmos papéis são arquivos em `configs/commandcode/agents/` (`code-reviewer`, `verifier`, `docs-writer`, `memory-keeper`) + entradas no `manifests/shell.yaml`; lá não existe `mode` nem edit com escopo de path, então o limite de escrita vira regra do prompt. Nomes reservados do runtime: `explore`/`plan`/`review`/`general`.
- O agente `goal` foi removido (o `build` cobre o fluxo); `/goal` continua funcionando pelo plugin.
- `envctl doctor` audita o shape do `opencode.json` (V1 `agent`/`permission` na raiz, ações `bash`/`task`, subagent sem `description`, mode inválido, **campos legacy V1 por agente** — `prompt`/`permission`/`tools`/`temperature`/`top_p`/`disable`/`maxSteps`) — o problema aparece como `Config shape` em vez de config ignorada em silêncio. Os templates embarcados são travados por `TestShippedOpenCodeTemplatesHaveNativeShape` e `TestShippedCommandCodeAgentTemplatesMatchSchema`.

## Skills: ativação proativa

O catálogo (nome + `description`) já vem no prompt a cada turno; o corpo é lido só quando a description casa. Por isso as descriptions carregam gatilhos explícitos (`Triggers:`) e os `AGENTS.md` mandam carregar a skill **antes de agir** quando ela casa. `SKILL-INDEX.md` é só desempate — não cole o índice no prompt. Critério de edição de description: gatilho em PT-BR **e** EN quando o termo técnico é em inglês, sem inflar o texto.

**Controles de visibilidade são INVERTIDOS entre as plataformas** — a mesma skill é servida aos dois, então um nome copiado do agente errado liga/desliga a coisa errada:

| Conceito | OpenCode V2 | CommandCode |
|---|---|---|
| esconder do modelo (não autoinvoca) | `metadata.opencode/autoinvoke: false` | `disable-model-invocation: true` |
| esconder do menu de comandos | `slash: false` (ou `metadata.opencode/slash`) | `user-invocable: false` |
| o que o modelo vê | `id`, `name` e `description` **inteira** | `description` + `when_to_use`, **cortada em 249 caracteres** |
| gatilho extra p/ auto-invoke | não existe campo — o gatilho vai na `description` | `when_to_use` (7 skills de roteamento) |
| limites | `name` ≤64 kebab-case, `description` ≤1024, chaves estranhas ignoradas | `name` ≤64 e **precisa** ser igual ao diretório, `description` ≤1024 obrigatória, chaves estranhas ignoradas |

**Regra do catálogo do CommandCode (medida no runtime com `cmdc -p`):**
1. **Default = 8.000 chars** (`Gs=8e3` no bundle). Com esse valor o modelo recebe
   **só `<name>` + `<location>`** — sem description, então matching por description
   não acontece. Medido: o catálogo "names only" das 48 skills já tem 6.910 chars.
2. Com budget, cada entrada é cortada em 249 caracteres (`Vs=250` → `slice(0,248)+"…"`),
   contando **caracteres** (semântica de `String.slice` do JS, não bytes). O piso
   `zs=20` é o mínimo por skill que o runtime tenta antes de cair para nomes.
   **Limiar medido para as 48 skills: ~25.700** (`6.910 + 48 × (142 + 249)`) — probes
   confirmam: 15.000 → description truncada ~140 chars sem `when_to_use`; 20.000/25.000
   → ainda cai para nomes; **30.000 → description + `when_to_use` completos**.
   Acima de ~26k o budget não muda nada (teto é 249/skill), então use 30.000.
   Custo: o catálogo passa a ~25.7k chars ≈ **6.4k tokens por turno** no CommandCode.
3. Contrato do envctl: **`description + 2 + when_to_use ≤ 247` caracteres** nas 7
   skills de roteamento. O resumo curto fica na description, o gatilho em linguagem
   natural vai no `when_to_use`, e a lista extendida de gatilhos foi para o corpo
   (`## Triggers`), que só é lida **depois** que a skill carrega — grátis no catálogo.
4. `TestWhenToUseFitsCommandCodeCatalog` trava esse limite (contando runes, como o
   runtime) e a lista de skills; editar a description desses 7 exige recalcular.

Consequência prática: a **primeira frase** da description é o que o modelo lê nos dois
runtimes — escreva o "quando usar" mais forte ali. O texto rico em gatilho é material do
OpenCode e do corpo da skill, não do catálogo do CommandCode.

O envctl valida o denominador comum (`name` == diretório, description não vazia e ≤1024) no `auditSkillTree` dos dois agentes. `when_to_use` é lido **só** pelo CommandCode, mas a árvore é compartilhada — e isso é seguro: o loader do OpenCode lê apenas `id`/`name`/`description`/`autoinvoke` e não valida chave desconhecida (verificado no binário `opencode v2.0.16`), então o campo é ignorado lá sem custo de prompt.

## Worktrees (convenção do repositório)

- Path: `.worktrees/<type>-<slug>` (ex.: `.worktrees/feat-agent-skills`); config de projeto `.opencode/opencode.json` (`worktree.directory`) + `watcher.ignore` para `.worktrees/**`; `/.worktrees/` no `.gitignore`.
- **No CommandCode não existe chave de diretório**: o runtime gerencia em `~/.commandcode/worktrees/<repo>-<hash>/<name>` (fora do repo, branch `worktree-<slug>`, base `origin` por padrão) via `/worktree` e `enter_worktree`; a ponte para a nossa convenção é `cmdc -w "$PWD/.worktrees/<slug>"` (path absoluto é usado verbatim). Sem `worktree.directory` para inventar.
- Um branch por worktree; dois agentes nunca dividem a mesma árvore. Criar a partir de base explícita (`origin/main`), com consentimento, e passar o path no prompt.
- Detecção antes de criar: `git worktree list --porcelain` + `git rev-parse --git-dir` vs `--git-common-dir`.
- `envctl doctor` reporta entrada `prunable` como **WARNING** e `locked` como **INFO** (nunca auto-poda/auto-destrai) — vale para as duas rotas, porque ambas são `git worktree` de verdade. `git worktree prune` só depois de revisar cada entrada.
- Worktree isola arquivos, não containers/portas/volumes/serviços — recursos concorrentes precisam de nome próprio.

## Supervisão de subagentes (OpenCode V2)

- Terminologia real: dispatch devolve `sessionID`; **não existe** `agent_output`/`agent_id` no toolset V2.
- Foreground: interromper a sessão pai cancela a filha; aguardar confirmação antes de retry.
- Background: `opencode api get /api/session/active` para status e `opencode api post /api/session/<sessionID>/interrupt` para interromper a sessão filha. Probe seguro de rota: ID inexistente devolve `SessionNotFoundError` (404) — não mata sessão viva.
- `sessionID` **não** é PID. Processo externo só é encerrado com PID rastreado + command line conferida: `SIGTERM` primeiro, `SIGKILL` só depois. Preservar worktree/logs (nada de `reset --hard`, `clean`, `worktree remove --force`).
- Regra de retry: no máximo **uma** repetição com escopo refinado; depois escalar.
- **O CommandCode tem o vocabulary oposto** (`agent_id`, `agent_output({action:"kill"})`, `run_in_background`, `kill_shell`, `monitor_command`) — ver doc oficial dele. A skill compartilhada `subagent-supervision` mantém as duas colunas em seções separadas: ao "limpar" este arquivo, nunca apague o vocabulary do CommandCode achando que é relic; o teste de conteúdo valida cada coluna por seção. No CommandCode use só o controle exposto pelo runtime ativo.

## Memória do agente

Skill `agent-memory`:

- **Global:** `~/.config/opencode/memory/lessons.md` + `patterns.md` (seed do envctl; conteúdo por máquina é preservado — `seed_if_missing`).
- **Projeto:** `.opencode/memory/lessons.md` + `patterns.md` (versionável apenas SEM dados privados).
- Formato: lições `❌ não faça X → ✅ faça Y (porque Z)` + data; patterns `✅ quando <situação>, faça <o que funciona>`.
- Ao gravar, classifique com `code-playbooks/references/skill-authoring.md`: processo reutilizável multi-passos → **vira skill** (criar `SKILL.md`, registrar em `configs/skills/` + `manifests/skills.yaml` e **remover a entrada da memória**); lição/anti-padrão/preferência → permanece na memória. Nunca manter o mesmo conteúdo em memória E skill.
- Nunca versionar memória global; nunca gravar segredos nem dados de domínio/fornecedor na memória global.


## Ferramentas e padrões

- **Acesso a tools das skills:** `Bash(*)`, `Read`, `Write`, `Edit`, `Glob`, `Grep`.
- **Serviços:** systemd (`systemctl status <svc>`, `journalctl -u <svc> --no-pager -n 50`, `sudo systemctl restart <svc>`); Docker (`docker ps -a`, `docker logs --tail 50 <ctr>`, `docker restart <ctr>`); PM2 (`pm2 list`, `pm2 logs <id> --lines 50`, `pm2 restart <id>`).
- **Privilegiado (Linux, non-root com sudo sem senha):** `sudo -n apt-get update`, `sudo -n systemctl restart nginx`.
- **SSH em outra VPS:** `ssh <host> "uptime && df -h /"`; fallback por CLI: `ssh-manager exec <server> "..."`.
- **Docker:** `docker ps -a --format "table {{.Names}}\t{{.Status}}\t{{.Image}}"` e `docker exec -it <container> /bin/sh`. No Windows roda nativamente do PowerShell, sem workaround de conversão de caminho.

## TUI (cli.json — fora do envctl)

`~/.config/opencode/cli.json` é preferência de terminal do usuário (tema, tabs, sidebar) — o envctl **não** gerencia: `session.sidebar: "auto"` esconde a sidebar sozinha em terminal estreito (alargue a janela ou `Ctrl+P > Open settings`). Não versinar nem sobrescrever via provisioning.

## Higiene de scratch

- **Windows:** scratch em `C:\temp` — limpe ao fim da sessão com `Remove-Item -Recurse -Force C:\temp\<seu-scratch>` (prefira um subdiretório por sessão).
- **Linux:** scratch em `/temp` — limpe com `rm -rf /temp/<seu-scratch>`.
- `envctl doctor` reporta acúmulo em cache/DB/tool-output/temp; `envctl run cleanup` remove duplicatas de plugins, tool-output >10 MB e scratch com mais de 24h.
- **Encoding (Windows):** o opencode spawna `pwsh -NoLogo -NoProfile -NonInteractive -Command` e o profile **nunca** carrega no bash tool — com o beta UTF-8 do Windows desligado (ACP/OEMCP=850) acentos viram `�`. Se o `doctor` acusar *System Code Page* 850, ative "Beta: Use Unicode UTF-8 para todo o mundo" e reinicie (não é bug do projeto).
- Se o shell do agente mostrar "Windows PowerShell (5.1)", o config de shell foi ignorado (opencode #41426) — reinicie o opencode.
- Scripts: criar em `C:\temp`/`/temp` ou no home, nunca em pastas de sistema. Automação de browser determinística: `playwright-cli` no shell (browsers próprios em `~/.cache/ms-playwright`, via `install-browser chromium`).
