# CommandCode Environment Manifest

## System Architecture

- **OS:** Windows 11 Pro 25H2 (amd64)
- **Workstation:** `<hostname>` | User: `<user>`
- **Shell Primary (CommandCode):** PowerShell 7.6.5 (`pwsh.exe`) — **shell default do CommandCode no Windows**. Todos os comandos dos agentes LLM são executados via PowerShell.
- **Shell Secondary:** WSL Ubuntu 26.04 (`wsl.exe -d Ubuntu`) — subshell POSIX para scripts legados e ferramentas Linux, usar apenas quando necessário (`wsl -e bash -lc "..."`).
- **Package Managers:** Winget (native Windows), Volta (Node ecosystem), Pip/Uv (Python), Dotnet Tool (.NET), APT (via WSL Ubuntu)
- **Node Runtime:** Node v24.19.0 managed via Volta (`NODE_PATH="%USERPROFILE%\node_modules"`)
- **Global Tools:** `rg` (ripgrep), `fd`, `fzf`, `bat`, `delta`, `yq`, `jq`, `ruff`, `gh`, `tree`, `zip/unzip`, `bun` (runtime JS/TS rápido — `bunx` substitui `npx`), `dust` (disk usage rápido — `du` trava no NTFS), `hyperfine` (benchmark de comandos), `shellcheck` (lint de bash p/ WSL/Linux), `csharp-ls`, `pw-screenshot`, `pw-eval`
- **Agent libs globais (Windows — uso direto em scripts, sem venv/node_modules por projeto):** Node via `NODE_PATH=%USERPROFILE%\node_modules` — `playwright`, `axios`, `cheerio`, `papaparse` (CSV); Python global — `pyyaml`, `requests`, `openpyxl` (xlsx), `beautifulsoup4` (HTML), `pypdf`, `python-docx`, `lxml`. SQLite via `python -c "import sqlite3"` (stdlib).
- **LSPs:** CommandCode integrates with IDE language servers automatically (VS Code, Cursor, Windsurf). No explicit LSP configuration needed.
- **Git Optimizations:** `fscache=true`, `preloadindex=true`, `longpaths=true`, `autocrlf=input`, `delta` pager

## Conventions & Rules

- **Language:** Code/comments/commits in English. User communication in Portuguese (BR).
- **Style:** Clean Architecture, SOLID, idiomatic code per language, strict typing.
- **Git:** Semantic branches (`feat/...`, `fix/...`), conventional commits, PRs via `gh pr create`.
- **Testing:** Evidence before claims — test before declaring complete.
- **Security:** Strict ACLs on `~/Documents/SSH-keys`, `~/.ssh-manager`, `~/.ssh`. Never hardcode secrets.
- **Zero Tolerância:** qualquer WARNING (lint, compilador, ts(6xxx), etc.) ou ERROR encontrado deve ser corrigido imediatamente, seja pré-existente ou novo. Nunca ignorar ou deixar para depois — dívida técnica não é acumulada.

## CommandCode Configuration

- **Global config:** `~/.commandcode/settings.json` (padrão único)
- **Global rules:** `~/.commandcode/AGENTS.md` — auto-carregado em todas as sessões CommandCode (este arquivo)
- **Shell (Windows):** PowerShell 7 (`C:\Program Files\PowerShell\7\pwsh.exe`) — **os agentes LLM DEVEM executar comandos via PowerShell nativo**. Regras:
  1. Comandos do shell do CommandCode são PowerShell: `Get-ChildItem`, `Test-Path`, etc. — não usar sintaxe bash por padrão.
  2. Ferramentas CLI (rg, fd, gh, git, docker, node, npm, npx) funcionam normalmente no PowerShell — sem wrapper.
  3. Scripts POSIX legados que exigem Linux/bash: usar WSL — `wsl -e bash -lc "..."` (nunca ao contrário).
  4. Variáveis de ambiente usam `$env:NOME` no PowerShell (ex.: `$env:ENVCTL_TEMP`).
- **Custom Agents:** `~/.commandcode/agents/` — markdown files with frontmatter (review, plan, goal)
- **Skills paths:** `~/.commandcode/skills/` (global) + `.commandcode/skills/` (project)
- **MCP:** `~/.commandcode/mcp.json` (user-scope) + `.mcp.json` (project-scope)
- **Taste Learning:** CommandCode continuously learns from accept/reject/edit signals stored in `.commandcode/taste/taste.md`
- **Config is NOT hot-reloaded:** restart CommandCode after changes.

## VPS Infrastructure (envctl)

- **Provisioner:** `envctl` (CLI Go standalone; repo: `C:\projetos\git-privado\envctl`)
- **Bootstrap (1 linha):** `curl -fsSL https://raw.githubusercontent.com/eajdias/envctl/main/bootstrap.sh | bash`
- **Comandos:** `envctl run all` (Day-0 completo), `envctl run shell` / `envctl run skills` (re-sync de configs/skills), `envctl doctor` / `envctl doctor --fix` (auditoria e auto-remediação), `envctl snapshot` (sync REVERSO máquina→repo — NÃO usar em VPS remota)

## Temp & Scratch Hygiene (Mandatory)

- **Pasta de scratch padrão dos agentes LLM: `C:\temp`** (variável `ENVCTL_TEMP` — criada pelo envctl na raiz do disco). Todo arquivo temporário criado por agentes **DEVE** ir para `C:\temp`, nunca para pastas do `.commandcode/` ou do projeto.
- **Nunca deixar scratch para trás**: todo arquivo criado em `C:\temp` durante uma sessão **DEVE ser removido antes do fim da sessão**.
- **After finishing a task**: rodar o cleanup pass sobre o scratch criado.
