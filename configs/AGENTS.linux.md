# OpenCode Environment Manifest (Linux Server)

## System Architecture

- **OS:** Ubuntu Server (LTS) (amd64) — provisioned by envctl
- **User:** `ubuntu` (non-root, passwordless sudo)
- **Shell Primary:** Bash (`/bin/bash`)
- **Package Managers:** APT (system packages), Volta (Node ecosystem), npm, pip/uv (Python), cargo/rustup (Rust, optional)
- **Node Runtime:** Node v24 LTS managed via Volta (`~/.volta`)
- **Global Tools:** `rg` (ripgrep), `fd` (via `fdfind` symlink), `fzf`, `bat`, `delta`, `yq`, `gh`, `uv`, `ruff`, `bun` (`bunx` substitui `npx`), `tree`, `zip/unzip`, `oh-my-posh`
- **OpenCode:** CLI installed via npm global (`opencode-ai`) into `~/.local/bin` (fallback: official install script)
- **LSPs Registered:** `opencode.json` (TypeScript, Pyright, PyLSP, Gopls, Bash, SQL, HTML, JSON, YAML, Dockerfile, CSS, Markdown, Rust Analyzer, CSharp-LS, ESLint, TOML, PHP — no PowerShell on Linux)
- **Git:** `preloadindex=true`, `autocrlf=input`, `init.defaultBranch=main`, `delta` pager (no fscache/longpaths — Windows-only)

## Conventions & Rules

- **Language:** Code/comments/commits in English. User communication in Portuguese (BR).
- **Style:** Clean Architecture, SOLID, idiomatic code per language, strict typing.
- **Git:** Semantic branches (`feat/...`, `fix/...`), conventional commits, PRs via `gh pr create`.
- **Testing:** Evidence before claims — test before declaring complete.
- **Security:** Strict ACLs on `~/.ssh`. Never hardcode secrets.
- **Zero Tolerância:** qualquer WARNING (lint, compilador, ts(6xxx), etc.) ou ERROR encontrado deve ser corrigido imediatamente, seja pré-existente ou novo. Nunca ignorar ou deixar para depois — dívida técnica não é acumulada. Três reforços obrigatórios:
  1. **Falhas pré-existentes NÃO são desculpa**: qualquer warning/erro/falha encontrada — nova ou pré-existente, no código ou em testes — deve ser corrigida **no mesmo turno**, conforme a regra de Zero Tolerância acima. Proibido "reportar e seguir" ou "documentar para depois".
  2. **Evidência antes de afirmação**: exibir as saídas reais de lint/type-check/testes na resposta final; se o comando não foi rodado, a verificação não conta.
  3. **Se algo não pôde ser corrigido**: a tarefa permanece **não concluída** — reportar explicitamente o bloqueio e o motivo, sem declarar sucesso parcial.
  4. **Fechamento com evidência fresca**: ao concluir uma tarefa com TODOs, reconcile a lista (nada pendente) e RE-EXECUTE a verificação (build/test/lint) para exibir saída atual — não basta afirmar que passou.

## OpenCode Configuration

- **Global config:** `~/.config/opencode/opencode.json` (padrão único — JSON, não JSONC)
- **Global rules:** `~/.config/opencode/AGENTS.md` — auto-carregado em todas as sessões opencode (este arquivo)
- **Plugin:** `@tarquinen/opencode-dcp@latest` (DCP context compression; config `~/.config/opencode/dcp.jsonc`; `compress` tool in experimental.primary_tools)
- **Skills paths:** `~/.config/opencode/skills` (fonte única — sem duplicatas)
- **MCPs (browser automation — padrão único):** `playwright` (`bunx @playwright/mcp@0.0.79 --browser chrome`) e `chrome-devtools` (`bunx chrome-devtools-mcp@1.8.0 --no-usage-statistics`) — ambos `enabled: false` (opt-in por sessão via `/mcp`); `bun`/`bunx` são instalados pelo bootstrap Linux.
- **DCP (context pruning):** compressão automática na banda **90% / 80%** (`dcp.jsonc`; `allowSubAgents: true` → também em subagentes) — limiar alto = menos compressões, priorizando cache-hit. Chame a tool `compress` **proativamente** ao trocar de assunto bruscamente ou concluir uma sub-tarefa cujo contexto verbatim não será mais usado; `manualMode` fica **desligado** (ligá-lo desativa a compressão autônoma).
- **Agent Memory (OBRIGATÓRIO — ativo em TODA tarefa):** a skill `agent-memory` deve ser CARREGADA (tool `skill` com name `agent-memory`) e seus arquivos LIDOS no INÍCIO de qualquer tarefa — antes de qualquer exploração/código: `.opencode/memory/lessons.md` e `.opencode/memory/patterns.md` do projeto (se existirem) + `~/.config/opencode/memory/lessons.md` e `~/.config/opencode/memory/patterns.md` globais. Isso vale para TODOS os agentes/subagentes (task, explore, general, etc.). Ao final da tarefa (ou ao cometer erro / ser corrigido / descobrir padrão), GRAVE a lição/pattern no arquivo correspondente — não deixe para depois. NUNCA repita lições registradas. Memórias globais e `.opencode/memory` de projeto são individuais por máquina: **nunca versionar memórias globais**; `.opencode/memory/*.md` de projeto é versionável apenas SEM dados privados (skill `agent-memory`).
- **Config is NOT hot-reloaded:** restart opencode after changes. Validate with `opencode debug config`.

### Skills — uso AUTOMÁTICO (obrigatório)

As skills são o **método padrão** deste ambiente e devem ser usadas **sem o usuário pedir**. Se a situação casar com a tabela abaixo, **carregue a skill ANTES de agir** e siga o procedimento dela. Não pergunte "quer que eu use a skill X?" — use. Nunca improvise um procedimento que já existe como skill.

- **Invocação:** `/<skill>`; a tool do modelo é `skill` (com `name: <skill>`).
- **Built-in:** `customize-opencode` — use ao editar `opencode.json` / `dcp.jsonc` / agentes.
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

## Service Management

- **systemd:** `systemctl status <svc>`, `journalctl -u <svc> --no-pager -n 50`, `sudo systemctl restart <svc>`
- **Docker:** `docker ps -a`, `docker logs --tail 50 <ctr>`, `docker restart <ctr>`
- **PM2:** `pm2 list`, `pm2 logs <id> --lines 50`, `pm2 restart <id>`

## Provisioning (envctl)

- Esta máquina foi provisionada pelo `envctl` (`~/.local/bin/envctl`); nunca editar configs manualmente — usar:
  - `envctl doctor` / `envctl doctor --fix` (auditoria e auto-remediação)
  - `envctl run shell` (re-sync configs) / `envctl run skills` (re-sync skills)
- Bootstrap em máquina nova: `curl -fsSL https://raw.githubusercontent.com/eajdias/envctl/main/bootstrap.sh | bash`
- Atualizar o binário: baixar release `envctl-linux-amd64` para `~/.local/bin/envctl`
- NUNCA usar `envctl snapshot` aqui (é sync REVERSO máquina→repo, para uso onde o repo existe, ex.: dev local)

## Common Patterns

### Privileged operations (non-root user)

```bash
sudo -n apt-get update
sudo -n systemctl restart nginx
```

### SSH exec on another VPS

```bash
ssh <host> "uptime && df -h /"
```

### Docker quick check & Container Exec

```bash
docker ps -a --format "table {{.Names}}\t{{.Status}}\t{{.Image}}"
docker exec -it <container> /bin/sh
```

## Uso Proativo de SSH, Context7 e Busca Web (OBRIGATÓRIO)

- **SSH / VPS** — skills `ssh-vps`, `vps-agent-dispatch`, `vps-provisioning`: para operações em servidores remotos, prefira a skill em vez de SSH cru (`ssh-vps` para operar serviços, `vps-agent-dispatch` para delegar tarefas pesadas a um OpenCode remoto, `vps-provisioning` para provisionar com envctl).
- **Context7 (docs de bibliotecas)** — skill `context7-auto` + MCP `context7`: ANTES de escrever código com lib/framework/API, busque documentação atualizada; não confie na memória do modelo.
- **Busca na internet** — ferramentas built-in `WebSearch`/`WebFetch`: use para fatos que mudam (versões, preços, docs públicas). Para páginas dinâmicas, use os MCPs de browser (`playwright` / `chrome-devtools`).

## Delegação a Subagentes (uso proativo)

Carregue a skill `subagent-routing` ao decidir delegar. Para preservar o contexto do coordenador: **exploração** de codebase sem alvo → `explore`; **pesquisa na internet/docs** → `general`; **debug** investiga a causa raiz primeiro. Paralelize domínios independentes na MESMA resposta; **não** paralelize falhas relacionadas / estado compartilhado / debug exploratório. Mecânica: `dispatching-parallel-agents`; mesmo repo git: `parallel-agent-orchestration`.

## Temp & Scratch Hygiene (Mandatory)

- **Pasta de scratch padrão dos agentes LLM: `/temp`** (variável `ENVCTL_TEMP` — criada pelo envctl na raiz do disco, SEM relação com o OpenCode). Todo arquivo temporário criado por agentes — downloads, builds, extrações — **DEVE** ir para `/temp`, nunca para pastas do opencode, do projeto ou do sistema.
- **Never leave scratch behind**: todo arquivo criado em `/temp` durante uma sessão **DEVE ser removido antes do fim da sessão**.
- **Big downloads/extracts**: se um tarball/zip ou build output for necessário apenas para produzir um resultado (ex.: `*.tar.gz`, `*.zip`, `*.FDB` copies, `opencode/` scratch), baixar/extrair em `/temp/<tarefa>\`, usar e deletar na mesma sessão.
- **After finishing a task**: rodar o cleanup pass sobre o scratch criado:
  ```bash
  ls -lh /temp | head -40
  rm -rf /temp/<seu-scratch>   # substitua pelo caminho exato
  ```
  Preferir um subdiretório dedicado por sessão (ex.: `/temp/opencode-<tarefa>`) para que o cleanup seja um único `rm -rf`.
- **envctl hygiene**: `envctl doctor` reporta acúmulo em cache/DB/tool-output/temp; `envctl run cleanup` remove duplicatas de plugins, tool-output >10 MB e scratch em `/temp` com mais de 24h.

## Notes

- Scripts should be created in `/temp` or the home directory, not in system folders; clean up test files after use.
- Playwright uses headless Chromium by default (system deps installed via `sudo npx playwright install-deps chromium`).
- Paths are POSIX (`~/.local/bin`, `~/.config/opencode`, `~/.ssh`).
