# OpenCode Environment Manifest

## System Architecture

- **OS:** Windows 11 Pro 25H2 (amd64)
- **Workstation:** `<hostname>` | User: `<user>`
- **Shell Primary (OpenCode):** PowerShell 7.6.5 (`pwsh.exe`) — **shell default do OpenCode no Windows**. Todos os comandos dos agentes LLM são executados via PowerShell.
- **Shell Secondary:** WSL Ubuntu 26.04 (`wsl.exe -d Ubuntu`) — subshell POSIX para scripts legados e ferramentas Linux, usar apenas quando necessário (`wsl -e bash -lc "..."`).
- **Package Managers:** Winget (native Windows), Volta (Node ecosystem), Pip/Uv (Python), Dotnet Tool (.NET), APT (via WSL Ubuntu)
- **Node Runtime:** Node v24.19.0 managed via Volta (`NODE_PATH="%USERPROFILE%\node_modules"`)
- **Global Tools:** `rg` (ripgrep), `fd`, `fzf`, `bat`, `delta`, `yq`, `jq`, `ruff`, `gh`, `tree`, `zip/unzip`, `bun` (runtime JS/TS rápido — `bunx` substitui `npx`), `dust` (disk usage rápido — `du` trava no NTFS), `hyperfine` (benchmark de comandos), `shellcheck` (lint de bash p/ WSL/Linux), `csharp-ls`, `pw-screenshot`, `pw-eval`
- **Agent libs globais (Windows — uso direto em scripts, sem venv/node_modules por projeto):** Node via `NODE_PATH=%USERPROFILE%\node_modules` — `playwright`, `axios`, `cheerio`, `papaparse` (CSV); Python global — `pyyaml`, `requests`, `openpyxl` (xlsx), `beautifulsoup4` (HTML), `pypdf`, `python-docx`, `lxml`. SQLite via `python -c "import sqlite3"` (stdlib).
- **LSPs Registered:** 18 language servers in `opencode.json` (TypeScript, Pyright, PyLSP, Gopls, Bash, SQL, HTML, JSON, YAML, Dockerfile, CSS, Markdown, PowerShell, Rust Analyzer, CSharp-LS, ESLint, TOML, PHP)
- **Git Optimizations:** `fscache=true`, `preloadindex=true`, `longpaths=true`, `autocrlf=input`, `delta` pager

## Conventions & Rules

- **Language:** Code/comments/commits in English. User communication in Portuguese (BR).
- **Style:** Clean Architecture, SOLID, idiomatic code per language, strict typing.
- **Git:** Semantic branches (`feat/...`, `fix/...`), conventional commits, PRs via `gh pr create`.
- **Testing:** Evidence before claims — test before declaring complete.
- **Security:** Strict ACLs on `~/Documents/SSH-keys`, `~/.ssh-manager`, `~/.ssh`. Never hardcode secrets.
- **Zero Tolerância:** qualquer WARNING (lint, compilador, ts(6xxx), etc.) ou ERROR encontrado deve ser corrigido imediatamente, seja pré-existente ou novo. Nunca ignorar ou deixar para depois — dívida técnica não é acumulada. Três reforços obrigatórios:
  1. **Falhas pré-existentes NÃO são desculpa**: qualquer warning/erro/falha encontrada — nova ou pré-existente, no código ou em testes — deve ser corrigida **no mesmo turno**, conforme a regra de Zero Tolerância acima. Proibido "reportar e seguir" ou "documentar para depois".
  2. **Evidência antes de afirmação**: exibir as saídas reais de lint/type-check/testes na resposta final; se o comando não foi rodado, a verificação não conta.
  3. **Se algo não pôde ser corrigido**: a tarefa permanece **não concluída** — reportar explicitamente o bloqueio e o motivo, sem declarar sucesso parcial.

## OpenCode Configuration

- **Global config:** `~\.config\opencode\opencode.json` (padrão único — JSON, não JSONC; configs antigas `opencode.jsonc` são removidas automaticamente)
- **Global rules:** `~\.config\opencode\AGENTS.md` — auto-carregado em todas as sessões opencode (este arquivo)
- **Shell (Windows):** PowerShell 7 (`C:\Program Files\PowerShell\7\pwsh.exe`) — **os agentes LLM DEVEM executar comandos via PowerShell nativo**. Regras:
  1. Comandos do shell do OpenCode são PowerShell: `Get-ChildItem`, `Test-Path`, etc. — não usar sintaxe bash por padrão.
  2. Ferramentas CLI (rg, fd, gh, git, docker, node, npm, npx) funcionam normalmente no PowerShell — sem wrapper.
  3. Scripts POSIX legados que exigem Linux/bash: usar WSL — `wsl -e bash -lc "..."` (nunca ao contrário).
  4. Variáveis de ambiente usam `$env:NOME` no PowerShell (ex.: `$env:ENVCTL_TEMP`).
- **Plugins (3):** `@tarquinen/opencode-dcp@latest`, `@dietrichgebert/ponytail`, `@prevalentware/opencode-goal-plugin`
- **Agentes customizados (3):** `review` (primary — revisão read-only, bash read-only para evidências), `plan` (mode `all` — primary via Tab E subagent via task tool; planejamento read-only: bash read-only, escrita apenas em `docs/superpowers/plans/`, carrega `writing-plans`/`agent-memory`/context7), `goal` (primary — modo autônomo com goal tools). Usar `plan` antes de implementações multi-passos e `review` antes de concluir/commitar.
- **Skills paths:** `~\.config\opencode\skills` (fonte única — sem duplicatas)
- **Agent Memory (OBRIGATÓRIO — ativo em TODA tarefa):** a skill `agent-memory` deve ser CARREGADA (tool `skill` com name `agent-memory`) e seus arquivos LIDOS no INÍCIO de qualquer tarefa — antes de qualquer exploração/código: `.opencode\memory\lessons.md` e `.opencode\memory\patterns.md` do projeto (se existirem) + `~\.config\opencode\memory\lessons.md` e `~\.config\opencode\memory\patterns.md` globais. Isso vale para TODOS os agentes/subagentes (task, explore, general, etc.). Ao final da tarefa (ou ao cometer erro / ser corrigido / descobrir padrão), GRAVE a lição/pattern no arquivo correspondente — não deixe para depois. NUNCA repita lições registradas. Memórias globais e `.opencode/memory` de projeto são individuais por máquina: **nunca versionar memórias globais**; `.opencode/memory/*.md` de projeto é versionável apenas SEM dados privados (skill `agent-memory`).
- **Skill Promotion (OBRIGATÓRIO — parâmetro fixo em TODA gravação de memória):** ao gravar QUALQUER entrada em lessons.md/patterns.md (projeto ou global), CLASSIFIQUE antes de salvar: a entrada descreve um **PROCESSO reutilizável multi-passos** (workflow, critérios de decisão, checagem que se repete)? → **PROMOVA a skill**: carregue a skill `memory-promotion` e siga o procedimento (criar `SKILL.md` no local correto — global `~\.config\opencode\skills\<nome>\` ou projeto `.opencode\skills\<nome>\` —, registrar para provisionamento via `envctl snapshot`, e **REMOVER a entrada da memória** — sem redundância entre memória e skill). A entrada é LIÇÃO/anti-padrão, fato do ambiente ou preferência? → permanece na memória. Nunca manter o mesmo conteúdo em memória E skill. O ideal: skills novas por projeto e globais crescem conforme o uso; as globais (sem dados pessoais) são implementadas no envctl (`configs/skills/` + `manifests/skills.yaml`).
- **Config is NOT hot-reloaded:** restart opencode after changes. Validate with `opencode debug config` (note: PowerShell `ConvertFrom-Json` fails on jsonc comments — expected).

## VPS Infrastructure (envctl)

- **Provisioner:** `envctl` (CLI Go standalone; repo: `C:\projetos\git-privado\envctl` — fonte única de configs/skills/agentes; binário embutido: `~/.local/bin/envctl` nas VPSs provisionadas)
- **Bootstrap (1 linha):** `curl -fsSL https://raw.githubusercontent.com/eajdias/envctl/main/bootstrap.sh | bash`
- **Comandos:** `envctl run all` (Day-0 completo), `envctl run shell` / `envctl run skills` (re-sync de configs/skills), `envctl doctor` / `envctl doctor --fix` (auditoria e auto-remediação), `envctl snapshot` (sync REVERSO máquina→repo — NÃO usar em VPS remota)
- **NOVAS VM/VPS — SEMPRE usar envctl:** nunca configurar servidor manualmente; fluxo padrão: SSH → bootstrap one-liner → `envctl run all` → `envctl doctor`. Provisionamento é idempotente (rodar N vezes = mesmo estado final). Workflow completo na skill `vps-provisioning`.
- **Nova conexão SSH:** ao cadastrar VPS/VM, registrar seguindo os padrões (ssh-manager + inventário local — skill `ssh-vps`, workflow de 3 passos) e SEMPRE provisionar a VPS com envctl (bootstrap + `envctl run all`) — a VPS ganha OpenCode próprio (plano Free) e vira orquestrável.
- **Limite Free na VPS:** se o OpenCode remoto estourar o limite free, PERGUNTAR ao usuário se quer registrar TOKEN (`opencode auth login` na VPS; o agente nunca manuseia o token). Se o usuário recusar, executar os comandos diretamente via SSH (skill `vps-agent-dispatch`, seção Limites Free & Fallback).
- **Inventário:** dinâmico via `ssh-manager server list` + arquivo local `~/.config/opencode/extras/ssh_servers.md` (nunca versionar IPs/usuários/chaves em arquivos provisionados). VPSs conhecidas: `homologacaochatbot`, `zscanchatbot`, `zscanchatcomercial`, `zscanproxyprod`, `zscanintranet`, `zscansaclocal`.
- **Orquestração remota:** skill `vps-agent-dispatch` (subagentes OpenCode remotos via SSH).

## Skill Locations

- **Skills (fonte única):** `~\.config\opencode\skills\` (**74 skills** — opencode 40 + firecrawl 33 + playwright 1)

### opencode skills (40)

| Skill | Purpose |
|-------|---------|
| `git-workflow` | Git & GitHub CLI workflow (semantic branches, conventional commits, PR lifecycle, conflict resolution) |
| `database-ops` | Dynamic multi-database management (PostgreSQL, MySQL, Firebird, MongoDB, SQLite, migrations, Docker DBs) |
| `universal-test-runner` | Multi-stack test runner & coverage (Node/TS, Python, Go, .NET, Rust) with TDD loop |
| `api-contract-design` | API design & validation (OpenAPI/Swagger, GraphQL SDL, gRPC/Protobuf, breaking change checks) |
| `ssh-vps` | SSH/VPS management via ssh-manager (monitoring, recovery) |
| `vps-provisioning` | Provision/manage VPSs via envctl bootstrap (Day-0/Day-2, idempotent) |
| `windows-admin` | Windows 11 administration (services, registry, winget, firewall) |
| `docker` | Docker Desktop/containers/compose + Docker Hub MCP |
| `context7-auto` | Fetch up-to-date library docs before code |
| `writing-plans` | Create implementation plans |
| `systematic-debugging` | Debug bugs methodically |
| `verification-before-completion` | Verify before claiming done |
| `receiving-code-review` | Handle code review feedback |
| `stop-slop` | Remove AI writing patterns |
| `grill-me` / `grilling` | Stress-test plans/thinking |
| `skill-miner` | Discover skills from session history |
| `skill-generalizer` | Make private skills publishable |
| `skill-personalizer` | Adapt skills to user preferences |
| `customize-opencode` | Edit opencode configuration |
| `docs-sync` | Audit doc coverage vs code |
| `ask-questions-if-underspecified` | Clarify requirements |
| `dispatching-parallel-agents` | Run independent tasks in parallel |
| `handoff` | Compact conversation for handoff |
| `using-git-worktrees` | Isolated feature work |
| `implementation-strategy` | Choose compatibility-aware scope |
| `bulk-postgres-import` | Bulk upsert in PostgreSQL over high-latency SSH |
| `docker-build-local-vps-deploy` | Build Docker locally, transport image to weak VPS |
| `docker-desktop-wsl-restart` | Restart Docker Desktop when WSL2 backend fails |
| `jwt-hs256-node` | JWT HS256 without external deps in Node.js |
| `lsp-smoke-test` | Smoke test LSP servers before registering |
| `memory-promotion` | Promote lessons to reusable skills |
| `nextjs-standalone-deploy` | Next.js standalone output deploy & v16 migration |
| `parallel-agent-orchestration` | Parallel subagents on the same git repo |
| `phone-e164-normalization` | Normalize BR phone numbers to E.164 |
| `playwright-prod-regression` | Safe Playwright regression against production |
| `simple-feature-flag` | Simple auditable feature flags in DB-backed apps |
| `web-dashboard-automation` | Automate authenticated dashboards/SPAs |

### agent skills (34) — Firecrawl + Playwright

Firecrawl (33): `firecrawl` (CLI base), `firecrawl-scrape`, `firecrawl-search`, `firecrawl-crawl`, `firecrawl-map`, `firecrawl-download`, `firecrawl-interact`, `firecrawl-parse`, `firecrawl-monitor`, `firecrawl-deep-research`, `firecrawl-research-papers`, `firecrawl-research-index`, `firecrawl-market-research`, `firecrawl-competitive-intel`, `firecrawl-developer-index`, `firecrawl-lead-gen`, `firecrawl-lead-research`, `firecrawl-company-directories`, `firecrawl-shop`, `firecrawl-build`, `firecrawl-build-scrape`, `firecrawl-build-search`, `firecrawl-build-interact`, `firecrawl-build-onboarding`, `firecrawl-knowledge-base`, `firecrawl-knowledge-ingest`, `firecrawl-website-design-clone`, `firecrawl-demo-walkthrough`, `firecrawl-seo-audit`, `firecrawl-qa`, `firecrawl-dashboard-reporting`, `firecrawl-workflows`, `firecrawl-agent`

Playwright (1): `playwright-cli` — use Node.js API, NOT `playwright-cli` command (blocks terminal):

```javascript
const { chromium } = require('playwright');
(async () => {
  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage();
  await page.goto('https://example.com');
  await page.screenshot({ path: 'screenshot.png' });
  await browser.close();
})();
```

## Tool Access

### Allowed Tools by Skill

- **playwright-cli:** `Bash(node:*)`, `Bash(npm:*)`, `Bash(npx:*)`
- **firecrawl skills:** `Bash(firecrawl:*)`, `Bash(npx:*)`, `Bash(node:*)`
- **general skills:** `Bash(*)`, `Read`, `Write`, `Edit`, `Glob`, `Grep`

## Common Patterns

### Running Firecrawl Commands

```bash
# Scrape a page
firecrawl scrape "https://example.com"
# Search
firecrawl search "query"
# Crawl site
firecrawl crawl "https://example.com" --limit 100
```

Firecrawl auth: CLI reads key from `%APPDATA%\firecrawl-cli\credentials.json` (NOT an env var). `firecrawl --status` shows auth/concurrency/credits.

### SSH exec on a VPS (CLI fallback)

```bash
ssh-manager server list
ssh-manager exec <server> "uptime && df -h /"
```

### Docker quick check & Container Exec

```bash
docker ps -a --format "table {{.Names}}\t{{.Status}}\t{{.Image}}"
docker exec -it <container> /bin/sh
```

Docker commands run natively from PowerShell without path-conversion workarounds.

### Parallel Agent Execution

Use `dispatching-parallel-agents` skill for independent tasks; each agent runs in its own context.

## Temp & Scratch Hygiene (Mandatory)

- **Pasta de scratch padrão dos agentes LLM: `C:\temp`** (variável `ENVCTL_TEMP` — criada pelo envctl na raiz do disco, SEM relação com o OpenCode). Todo arquivo temporário criado por agentes — scripts do Playwright CLI (`pw-screenshot`, `pw-eval`), downloads, builds, extrações, screenshots — **DEVE** ir para `C:\temp`, nunca para pastas do opencode (`~/.local/share/opencode`, `~/.cache/opencode`), do projeto ou do sistema.
- **Nunca deixar scratch para trás**: todo arquivo criado em `C:\temp` durante uma sessão **DEVE ser removido antes do fim da sessão**. `C:\temp` é de identificação e exclusão fáceis por estar na raiz do disco.
- **Big downloads/extracts**: se um tarball/zip ou output de build for necessário apenas para produzir um resultado, baixar/extrair em `C:\temp\<tarefa>\`, usar e deletar na mesma sessão.
- **After finishing a task**: rodar o cleanup pass sobre o scratch criado:
  ```powershell
  Get-ChildItem C:\temp | Select-Object Name, LastWriteTime
  Remove-Item -Recurse -Force C:\temp\<seu-scratch>   # substitua pelo caminho exato
  ```
  Preferir um subdiretório dedicado por sessão (ex.: `C:\temp\opencode-<tarefa>`) para que o cleanup seja um único `Remove-Item`.
- **envctl hygiene**: `envctl doctor` reporta acúmulo em cache/DB/tool-output/temp; `envctl run cleanup` remove duplicatas de plugins, tool-output >10 MB e scratch em `C:\temp` com mais de 24h.
- Playwright usa Chromium headless por padrão.
- Quando o shell do agente mostrar 'Windows PowerShell (5.1)', o config de shell foi ignorado (opencode issue #41426) — reiniciar o opencode para aplicar o pwsh 7.
