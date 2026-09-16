# CommandCode Environment Manifest (Linux Server)

## System Architecture

- **OS:** Ubuntu Server (LTS) (amd64) — provisioned by envctl
- **User:** `ubuntu` (non-root, passwordless sudo)
- **Shell Primary (CommandCode):** Bash (`/bin/bash`) — o shell do CommandCode nesta máquina é POSIX, não PowerShell
- **Package Managers:** APT (system packages), Volta (Node ecosystem), npm, uv/pip (Python), cargo/rustup (Rust, optional)
- **Node Runtime:** Node v24 LTS managed via Volta (`~/.volta`)
- **Global Tools:** `rg` (ripgrep), `fd` (via `fdfind` symlink), `fzf`, `bat`, `delta`, `yq`, `gh`, `uv`, `ruff`, `bun` (`bunx` substitui `npx`), `tree`, `zip/unzip`, `oh-my-posh`
- **CommandCode CLI:** `cmdc`, instalado por npm global (`command-code`) em `~/.local/bin`
- **LSPs:** CommandCode não tem configuração de LSP — ele expõe o diagnóstico do IDE conectado (`/ide` + tool `get_diagnostics`). Os binários de LSP instalados servem ao OpenCode e ao shell.
- **Git:** `preloadindex=true`, `autocrlf=input`, `init.defaultBranch=main`, `delta` pager (no fscache/longpaths — Windows-only)

## Conventions & Rules

- **Language:** Code/comments/commits in English. User communication in Portuguese (BR).
- **Style:** Clean Architecture, SOLID, idiomatic code per language, strict typing.
- **Git:** Semantic branches (`feat/...`, `fix/...`), conventional commits, PRs via `gh pr create`.
- **Testing:** Evidence before claims — test before declaring complete.
- **Security:** Strict ACLs on `~/.ssh`. Never hardcode secrets.
- **Zero Tolerância:** qualquer WARNING (lint, compilador, ts(6xxx), etc.) ou ERROR encontrado deve ser corrigido imediatamente, seja pré-existente ou novo. Nunca ignorar ou deixar para depois — dívida técnica não é acumulada. **Fechamento:** ao concluir uma tarefa com TODOs, reconcilie a lista (nada pendente) e RE-EXECUTE a verificação (build/test/lint) para evidência fresca.

## CommandCode Configuration

- **Global config:** `~/.commandcode/settings.json` (padrão único). Use `cmdc config list|get|set` em vez de editar à mão.
- **Global rules:** `~/.commandcode/AGENTS.md` — auto-carregado em todas as sessões CommandCode (este arquivo).
- **Custom Agents:** `~/.commandcode/agents/` — markdown (frontmatter: `name`, `description`, `tools` — `"*"` = todas, omitido = nenhuma, `disallowedTools`, `model` — omitido/`inherit` segue a sessão, `reasoningEffort`, `maxTurns` — default 100, `permissionMode`, `background`, `showOutput`). Built-ins: **General** (default), **Explore** (read-only), **Plan** (read-only). **Nomes reservados** (`explore`/`plan`/`review`/`general`) são IGNORADOS — por isso o agente custom é `code-reviewer` (read-only, evidência file:line).
- **Skills paths:** `~/.commandcode/skills/` (global) + `.commandcode/skills/` (project) + compat `.agents/skills/`. Invocação: `/<skill>` ou `/skill:<nome>` (a tool do modelo é `activate_skill`).
- **MCP:** user-scope `~/.commandcode/mcp.json` (o que o envctl gerencia) + project-scope `.mcp.json` + local-scope `~/.commandcode/projects/<slug>/mcp.json`. Precedência: local > project > user. Browser automation (padrão único): `playwright` (`bunx @playwright/mcp@0.0.79 --browser chrome`) e `chrome-devtools` (`bunx chrome-devtools-mcp@1.8.0 --no-usage-statistics`) — ambos `enabled: false` (opt-in via `/mcp`), pinados em `bunx` (nunca `npx @latest`).
- **Taste Learning:** aprende continuamente de sinais accept/reject/edit. Projeto: `.commandcode/taste/`; global: `~/.commandcode/taste/`. Nunca editar à mão — use a tool `taste` ou `/taste`.
- **Memory:** `~/.commandcode/AGENTS.md` (user tier) é a memória do CommandCode — NÃO existe memory-dir de lessons/patterns (isso é do opencode).
- **Hot reload:** agents, skills e memória são re-lidos a cada turno/request — **sem restart**. Valores de `settings.json` (`cmdc config`) passam a valer no início do próximo round. Só um update já baixado exige `/reload`.
- **Environment:** `/temp` é o scratch padrão dos agentes (`ENVCTL_TEMP`); remover o scratch ao fim da sessão.

## Skills — uso AUTOMÁTICO (obrigatório)

As skills são o **método padrão** deste ambiente e devem ser usadas **sem o usuário pedir**. Se a situação casar com a tabela abaixo, **carregue a skill ANTES de agir** e siga o procedimento dela. Não pergunte "quer que eu use a skill X?" — use. Nunca improvise um procedimento que já existe como skill.

- **Invocação:** `/<skill>` (ou `/skill:<nome>`); a tool do modelo é `activate_skill`.
- **Marcadores:** `[win]` = só existe em máquinas Windows; `[linux]` = só existe em Linux. Sem marcador = em qualquer ambiente.
- **Regra de ouro:** se a tarefa é "como fazer X neste ambiente", procure na tabela antes de improvisar.

| Quando a situação for… | Skill |
|---|---|
| Escrever, alterar ou configurar código com lib/framework/API | `context7-auto` |
| Bug, teste vermelho, erro intermitente, comportamento inesperado | `systematic-debugging` |
| Antes de declarar "pronto"/corrigido/passando (inclusive antes de commitar ou abrir PR) | `verification-before-completion` |
| Implementar algo multi-passo, feature nova ou refatoração ampla | `writing-plans` |
| Pedido ambíguo, incompleto ou com premissas não ditas | `ask-questions-if-underspecified` |
| Endurecer, questionar ou stress-testar um plano/decisão | `grilling` |
| Receber code review ou comentários de PR | `receiving-code-review` |
| Commit, branch, PR, rebase, conflito, tag/release | `git-workflow` |
| Isolar o trabalho num workspace próprio (feature paralela) | `using-git-worktrees` |
| Decidir delegar / explorar codebase / pesquisar na web | `subagent-routing` |
| Executar 2+ tarefas independentes em paralelo | `dispatching-parallel-agents` |
| Vários subagentes no mesmo repositório git | `parallel-agent-orchestration` |
| Schema, migration, query, backup/restore de banco | `database-ops` |
| Inserir/atualizar muitos registros no PostgreSQL | `bulk-postgres-import` |
| Rodar testes, medir cobertura, rodar benchmark | `universal-test-runner` |
| Desenhar/validar contrato de API (OpenAPI, GraphQL, gRPC) | `api-contract-design` |
| Servidor/VPS remoto: monitorar, diagnosticar, reiniciar serviço | `ssh-vps` |
| Provisionar, atualizar ou auditar VPS/VM com envctl | `vps-provisioning` |
| Rodar tarefa pesada (build, suíte, crawler) numa VPS | `vps-agent-dispatch` |
| Containers, compose, imagens, volumes, Docker Hub | `docker` |
| Windows: serviços, registro, tarefas agendadas, firewall, winget | `windows-admin` `[win]` |
| Docker Desktop não sobe / erro de backend WSL2 | `docker-desktop-wsl-restart` `[win]` |
| Deploy de imagem Docker em VPS fraca (build local) | `docker-build-local-vps-deploy` |
| Deploy de Next.js standalone / migração v15→v16 | `nextjs-standalone-deploy` |
| Ligar/desligar funcionalidade sem re-deploy | `simple-feature-flag` |
| JWT HS256 em Node.js sem dependências | `jwt-hs256-node` |
| Padronizar telefone BR (E.164) | `phone-e164-normalization` |
| Regressão/validação contra produção sem mutar dados | `playwright-prod-regression` |
| Dashboard/SPA autenticado (login, extrair dados, executar ação via HTTP) | `web-dashboard-automation` |
| Automação de browser (abrir, clicar, extrair, screenshot) | MCPs `playwright` / `chrome-devtools` — habilite via `/mcp` |
| Validar servidor de linguagem (LSP) que não responde | `lsp-smoke-test` |
| Texto, README, doc ou resposta com cara de IA | `stop-slop` |
| Encerrar sessão longa / transferir contexto para outra sessão ou agente | `handoff` |
| Início de qualquer tarefa (ler lições) e fim (gravar lições) | `agent-memory` |
| Vai gravar entrada em lessons/patterns | `memory-promotion` |
| Descobrir/criar skill a partir de uso repetido | `skill-miner` |
| Ajustar skill recém-baixada ao ambiente do usuário | `skill-personalizer` |
| Publicar/compartilhar skill (remover dados privados) | `skill-generalizer` |
| Jogos/emulação no CachyOS (kernel, GPU, Steam, emuladores) | `cachyos-gaming-setup` `[linux]` |
| Instalar pacote AUR sem TTY/senha | `aur-headless-install` `[linux]` |
| Validar app GUI (Qt/SDL) sem display | `headless-gui-probe` `[linux]` |

**Delegação (sempre proativa):** `subagent-routing` decide *quem* delegar; `dispatching-parallel-agents` e `parallel-agent-orchestration` dão a *mecânica*. Catálogo completo em `docs/skills.md`.

## Provisioning (envctl)

- Esta máquina foi provisionada pelo `envctl` (`~/.local/bin/envctl`); nunca editar configs manualmente — usar:
  - `envctl doctor` / `envctl doctor --fix` (auditoria e auto-remediação)
  - `envctl run shell` (re-sync configs) / `envctl run skills` (re-sync skills)
- Bootstrap em máquina nova: `curl -fsSL https://raw.githubusercontent.com/eajdias/envctl/main/bootstrap.sh | bash`
- Atualizar o binário: baixar release `envctl-linux-amd64` para `~/.local/bin/envctl`
- NUNCA usar `envctl snapshot` aqui (é sync REVERSO máquina→repo, para uso onde o repo existe, ex.: dev local)

## Service Management

- **systemd:** `systemctl status <svc>`, `journalctl -u <svc> --no-pager -n 50`, `sudo systemctl restart <svc>`
- **Docker:** `docker ps -a`, `docker logs --tail 50 <ctr>`, `docker restart <ctr>`
- **PM2:** `pm2 list`, `pm2 logs <id> --lines 50`, `pm2 restart <id>`

## Uso Proativo de SSH, Context7 e Busca Web (OBRIGATÓRIO)

- **SSH / VPS** — skills `ssh-vps`, `vps-agent-dispatch`, `vps-provisioning`: para operações em servidores remotos, prefira a skill em vez de SSH cru (`ssh-vps` para operar serviços, `vps-agent-dispatch` para delegar tarefas pesadas a um OpenCode remoto, `vps-provisioning` para provisionar com envctl).
- **Context7 (docs de bibliotecas)** — skill `context7-auto` + MCP `context7`: ANTES de escrever código com lib/framework/API, busque documentação atualizada; não confie na memória do modelo.
- **Busca na internet** — ferramentas `WebSearch`/`WebFetch`: use para fatos que mudam (versões, preços, docs públicas). Para páginas dinâmicas, use os MCPs de browser (`playwright` / `chrome-devtools`) — vêm `enabled: false`, então habilite via `/mcp` antes de usar.

## Delegação a Subagentes (uso proativo)

Carregue a skill `subagent-routing` ao decidir delegar. Para preservar o contexto do coordenador: **exploração** de codebase sem alvo → `explore`; **pesquisa na internet/docs** → `general`; **debug** investiga a causa raiz primeiro. Paralelize domínios independentes na MESMA resposta; **não** paralelize falhas relacionadas / estado compartilhado / debug exploratório. Mecânica: `dispatching-parallel-agents`; mesmo repo git: `parallel-agent-orchestration`.

## Temp & Scratch Hygiene (Mandatory)

- **Pasta de scratch padrão dos agentes LLM: `/temp`** (variável `ENVCTL_TEMP` — criada pelo envctl na raiz do disco). Todo arquivo temporário criado por agentes **DEVE** ir para `/temp`, nunca para pastas do `.commandcode/`, do projeto ou do sistema.
- **Nunca deixar scratch para trás**: todo arquivo criado em `/temp` durante uma sessão **DEVE ser removido antes do fim da sessão**.
- **envctl hygiene**: `envctl doctor` reporta acúmulo em cache/DB/tool-output/temp; `envctl run cleanup` remove scratch em `/temp` com mais de 24h.
