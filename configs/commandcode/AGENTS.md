# CommandCode Environment Manifest

## System Architecture

- **OS:** Windows 11 Pro 25H2 (amd64)
- **Workstation:** `<hostname>` | User: `<user>`
- **Shell Primary (CommandCode):** PowerShell 7.6.5 (`pwsh.exe`) — **shell default do CommandCode no Windows**. Todos os comandos dos agentes LLM são executados via PowerShell.
- **Shell Secondary:** WSL Ubuntu 26.04 (`wsl.exe -d Ubuntu`) — subshell POSIX para scripts legados e ferramentas Linux, usar apenas quando necessário (`wsl -e bash -lc "..."`).
- **Package Managers:** Winget (native Windows), Volta (Node ecosystem), Pip/Uv (Python), Dotnet Tool (.NET), APT (via WSL Ubuntu)
- **Node Runtime:** Node v24.19.0 managed via Volta (`NODE_PATH="%USERPROFILE%\node_modules"`)
- **Global Tools:** `rg` (ripgrep), `fd`, `fzf`, `bat`, `delta`, `yq`, `jq`, `ruff`, `gh`, `tree`, `zip/unzip`, `bun` (runtime JS/TS rápido — `bunx` substitui `npx`), `dust` (disk usage rápido — `du` trava no NTFS), `hyperfine` (benchmark de comandos), `shellcheck` (lint de bash p/ WSL/Linux), `csharp-ls`
- **Agent libs globais (Windows — uso direto em scripts, sem venv/node_modules por projeto):** Node via `NODE_PATH=%USERPROFILE%\node_modules` — `playwright`, `axios`, `cheerio`, `papaparse` (CSV); Python global — `pyyaml`, `requests`, `openpyxl` (xlsx), `beautifulsoup4` (HTML), `pypdf`, `python-docx`, `lxml`. SQLite via `python -c "import sqlite3"` (stdlib).
- **LSPs:** CommandCode integrates with IDE language servers automatically (VS Code, Cursor, Windsurf). No explicit LSP configuration needed.
- **Git Optimizations:** `fscache=true`, `preloadindex=true`, `longpaths=true`, `autocrlf=input`, `delta` pager

## Conventions & Rules

- **Language:** Code/comments/commits in English. User communication in Portuguese (BR).
- **Style:** Clean Architecture, SOLID, idiomatic code per language, strict typing.
- **Git:** Semantic branches (`feat/...`, `fix/...`), conventional commits, PRs via `gh pr create`.
- **Testing:** Evidence before claims — test before declaring complete.
- **Security:** Strict ACLs on `~/Documents/SSH-keys`, `~/.ssh-manager`, `~/.ssh`. Never hardcode secrets.
- **Zero Tolerância:** qualquer WARNING (lint, compilador, ts(6xxx), etc.) ou ERROR encontrado deve ser corrigido imediatamente, seja pré-existente ou novo. Nunca ignorar ou deixar para depois — dívida técnica não é acumulada. **Fechamento:** ao concluir uma tarefa com TODOs, reconcilie a lista (nada pendente) e RE-EXECUTE a verificação (build/test/lint) para evidência fresca.

## CommandCode Configuration

- **Global config:** `~/.commandcode/settings.json` (padrão único)
- **Global rules:** `~/.commandcode/AGENTS.md` — auto-carregado em todas as sessões CommandCode (este arquivo)
- **Shell (Windows):** PowerShell 7 (`C:\Program Files\PowerShell\7\pwsh.exe`) — **os agentes LLM DEVEM executar comandos via PowerShell nativo**. Regras:
  1. Comandos do shell do CommandCode são PowerShell: `Get-ChildItem`, `Test-Path`, etc. — não usar sintaxe bash por padrão.
  2. Ferramentas CLI (rg, fd, gh, git, docker, node, npm, npx) funcionam normalmente no PowerShell — sem wrapper.
  3. Scripts POSIX legados que exigem Linux/bash: usar WSL — `wsl -e bash -lc "..."` (nunca ao contrário).
  4. Variáveis de ambiente usam `$env:NOME` no PowerShell (ex.: `$env:ENVCTL_TEMP`).
- **Custom Agents:** `~/.commandcode/agents/` — markdown (frontmatter: `name`, `description`, `tools`, `model`, `reasoningEffort`, `maxTurns`, `background`). Built-ins do CommandCode: **General** (default), **Explore** (read-only), **Plan** (read-only). **Nomes reservados** (`explore`/`plan`/`review`/`general`) são IGNORADOS — por isso o agente custom é `code-reviewer` (read-only, evidência file:line).
- **Skills paths:** `~/.commandcode/skills/` (global) + `.commandcode/skills/` (project)
- **MCP:** `~/.commandcode/mcp.json` (user-scope) + `.mcp.json` (project-scope). Browser automation (padrão único): `playwright` (`bunx @playwright/mcp@0.0.79 --browser chrome`) e `chrome-devtools` (`bunx chrome-devtools-mcp@1.8.0 --no-usage-statistics`) — ambos `enabled: false` (opt-in via `/mcp`), pinados em `bunx` (nunca `npx @latest`). O `playwright-cli` (Node API) segue disponível como alternativa.
- **Taste Learning:** CommandCode continuously learns from accept/reject/edit signals stored in `.commandcode/taste/taste.md`
- **Memory:** `~/.commandcode/AGENTS.md` (user tier) é a memória do CommandCode — NÃO existe memory-dir de lessons/patterns (isso é do opencode).
- **Config is NOT hot-reloaded:** restart CommandCode after changes.

## Skills (40)

Provisionadas em `~/.commandcode/skills` (mesmo conjunto do opencode; catálogo detalhado em `docs/skills.md`). Carregue sob demanda via a tool `skill`; delegue exploração/pesquisa conforme `subagent-routing`.

- **Engenharia:** `git-workflow`, `database-ops`, `universal-test-runner`, `api-contract-design`, `systematic-debugging`, `verification-before-completion`, `receiving-code-review`, `using-git-worktrees`, `writing-plans`
- **Infra remota:** `vps-agent-dispatch`, `ssh-vps`, `vps-provisioning`, `docker`, `windows-admin`
- **Contexto/meta:** `agent-memory`, `memory-promotion`, `context7-auto`, `docs-sync`, `ask-questions-if-underspecified`, `dispatching-parallel-agents`, `subagent-routing`, `parallel-agent-orchestration`, `handoff`, `grilling`, `grill-me`, `stop-slop`, `skill-miner`, `skill-generalizer`, `skill-personalizer`, `implementation-strategy`
- **Domínio:** `bulk-postgres-import`, `docker-build-local-vps-deploy`, `docker-desktop-wsl-restart`, `jwt-hs256-node`, `lsp-smoke-test`, `nextjs-standalone-deploy`, `phone-e164-normalization`, `playwright-prod-regression`, `simple-feature-flag`, `web-dashboard-automation`

## VPS Infrastructure (envctl)

- **Provisioner:** `envctl` (CLI Go standalone; repo: `C:\projetos\git-privado\envctl`)
- **Bootstrap (1 linha):** `curl -fsSL https://raw.githubusercontent.com/eajdias/envctl/main/bootstrap.sh | bash`
- **Comandos:** `envctl run all` (Day-0 completo), `envctl run shell` / `envctl run skills` (re-sync de configs/skills), `envctl doctor` / `envctl doctor --fix` (auditoria e auto-remediação), `envctl snapshot` (sync REVERSO máquina→repo — NÃO usar em VPS remota)

## Uso Proativo de SSH, Context7 e Busca Web (OBRIGATÓRIO)

- **SSH / VPS** — skills `ssh-vps`, `vps-agent-dispatch`, `vps-provisioning`: para operações em servidores remotos, prefira a skill em vez de SSH cru (`ssh-vps` para operar serviços, `vps-agent-dispatch` para delegar tarefas pesadas a um OpenCode remoto, `vps-provisioning` para provisionar com envctl).
- **Context7 (docs de bibliotecas)** — skill `context7-auto` + MCP `context7`: ANTES de escrever código com lib/framework/API, busque documentação atualizada; não confie na memória do modelo.
- **Busca na internet** — ferramentas `WebSearch`/`WebFetch`: use para fatos que mudam (versões, preços, docs públicas). Para páginas dinâmicas, use os MCPs de browser (`playwright` / `chrome-devtools`).

## Delegação a Subagentes (uso proativo)

Carregue a skill `subagent-routing` ao decidir delegar. Para preservar o contexto do coordenador: **exploração** de codebase sem alvo → `explore`; **pesquisa na internet/docs** → `general`; **debug** investiga a causa raiz primeiro. Paralelize domínios independentes na MESMA resposta; **não** paralelize falhas relacionadas / estado compartilhado / debug exploratório. Mecânica: `dispatching-parallel-agents`; mesmo repo git: `parallel-agent-orchestration`.

## Temp & Scratch Hygiene (Mandatory)

- **Pasta de scratch padrão dos agentes LLM: `C:\temp`** (variável `ENVCTL_TEMP` — criada pelo envctl na raiz do disco). Todo arquivo temporário criado por agentes **DEVE** ir para `C:\temp`, nunca para pastas do `.commandcode/` ou do projeto.
- **Nunca deixar scratch para trás**: todo arquivo criado em `C:\temp` durante uma sessão **DEVE ser removido antes do fim da sessão**.
- **After finishing a task**: rodar o cleanup pass sobre o scratch criado.
