# Global Agent Skills Reference

## Environment

- **Platform:** Windows 11 (win32)
- **Default shell:** MSYS2 bash (`C:\msys64\usr\bin\bash.exe`) — configured via `shell` in `~\.config\opencode\opencode.jsonc`
- **Fallback shell:** PowerShell 5.1 (use ONLY when truly necessary)
- **User home:** `C:\Users\eajdias-note`

### Toolchain (verified versions)

| Tool | Version | Location |
|------|---------|----------|
| opencode | 1.18.18 | global config `~\.config\opencode\opencode.jsonc` |
| node | 24.19.0 | Volta (`C:\Program Files\Volta\node.exe`, shims in `%LOCALAPPDATA%\Volta\bin`) |
| npm / pnpm | 11.17.0 / 11.22.0 | Volta |
| python + uv + ruff | 3.14.7 / 0.12.5 / 0.16.1 | `C:\Program Files\Python314`, uv/ruff via Winget in PATH |
| go + gopls | 1.26.6 | `C:\Program Files\Go`, gopls `~\go\bin\gopls.exe` |
| dotnet SDK + csharp-ls | 10.0.400 / 0.26.0 | `C:\Program Files\dotnet`, csharp-ls `~/.dotnet/tools/csharp-ls.exe` |
| rust + rust-analyzer | stable | `~/.cargo/bin/rust-analyzer.exe` |
| git + delta | 2.55.0.windows.4 / 0.19.2 | Git for Windows (fscache/preloadindex active), delta via Winget |
| gh (GitHub CLI) | 2.97.0 | `C:\Program Files\GitHub CLI\gh.exe` (authenticated) |
| pwsh | 7.6.5 | WindowsApps |
| winget | 1.29.290 | native (works from MSYS2 bash, verified) |
| rg / fd / fzf / bat / yq | 15.2 / 10.4 / 0.74 / 0.26 / 4.53 | Winget / MSYS2 inherit PATH |
| tree / zip / unzip | 2.3.2 / 3.0 / 6.0 | MSYS2 (`C:\msys64\usr\bin`) |
| choco | 2.7.3 | only chocolatey itself installed |
| docker / compose | 29.7.2 / v5.4.0 | Docker Desktop 4.87.0 (`C:\Program Files\Docker\Docker\resources\bin`) |
| rsync / jq / sshpass | 3.4.4 / 1.8.1 / 1.10 | MSYS2 (`C:\msys64\usr\bin`) |
| ssh | Git's ssh + `C:\Windows\System32\OpenSSH\ssh.exe` | client only |

## Shell Policy — Prefer MSYS2 Bash

Prefer MSYS2 bash for ALL commands. Use PowerShell ONLY when truly necessary (Windows-only operations: services, registry, scheduled tasks, Windows features, PowerShell modules/objects).

- MSYS2 bash (`C:\msys64\usr\bin\bash.exe`) is the configured default shell and provides `rsync`, `jq`, `sshpass`, `ssh`, `ssh-manager` (CLI). Windows tools like `winget` also work from MSYS2 bash (verified by user).
- **Workaround (Windows bug, opencode issue #41426):** the agent bash tool may ignore the `shell` config and run PowerShell 5.1 instead. If the bash tool is running PowerShell, force MSYS2 bash for any command with: `bash -lc "<command>"`.
- PowerShell-only scenarios: services (`Get-Service`, `sc.exe`, `Restart-Service`), registry, `schtasks`, Windows Features, anything requiring PowerShell objects/modules.
- Windows admin operations follow the `windows-admin` skill; Docker follows the `docker` skill; remote servers follow the `ssh-vps` skill.

## Platform Inventory

### Windows 11 (native)

- OpenSSH **client** installed (`C:\Windows\System32\OpenSSH\ssh.exe`); OpenSSH **server** (sshd) installed via `Add-WindowsCapability` (config: `Set-Service sshd -StartupType Automatic; Start-Service sshd`; firewall rule `OpenSSH-Server-In-TCP` usually auto-created).
- **ssh-agent service is DISABLED deliberately** (user choice: avoid unnecessary context cost) — do not re-enable without asking.
- NO `~/.ssh/config` file exists.
- PowerShellEditorServices v4.7.0 (LSP powershell) at `C:\Users\eajdias-note\Documents\PowerShell\Modules\PowerShellEditorServices`.

### WSL2

- Ubuntu (default distro, WSL2) — verify with `wsl -l -v`, run commands with `wsl -d Ubuntu -- <cmd>`.
- `docker-desktop` distro runs the Docker daemon backend.
- Files accessible via `\\wsl$\Ubuntu\...`; Linux binaries from Windows via `wsl.exe`.

### Docker

- Docker Desktop 4.87.0, daemon on WSL2 backend (server 29.7.2, os=linux). Start with `Start-Service com.docker.service` or Docker Desktop app; check `docker version`.
- Compose v5.4.0. See `docker` skill for workflow.
- Docker Hub MCP (`docker-hub`): local clone at `C:\Users\eajdias-note\Documents\docker-hub-mcp-server` (no npm package; run `node dist/index.js --transport=stdio`). **Disabled by default** — user enables in config when needed; optional auth env `HUB_USERNAME` + `HUB_PAT_TOKEN`.

### SSH / VPS (ssh-manager)

- CLI: `ssh-manager` + MCP `mcp-ssh-manager` v3.8.0 (Volta global). Config: `C:\Users\eajdias-note\.ssh-manager\.env` (`SSH_SERVER_<NAME>_HOST/_USER/_KEYPATH/...`).
- SSH keys: `C:\Users\eajdias-note\Documents\SSH-keys\<VPS>\<VPS>.pem`.
- MCP `ssh-manager` is **disabled by default** in opencode.jsonc — agent must ASK user to enable; CLI fallback: `ssh-manager server list|test|exec <server> "<cmd>"`.
- Full workflow in `ssh-vps` skill (registered servers, recovery checklist, best practices).

### MSYS2

- Full MSYS2 at `C:\msys64` (pacman available); `MSYS2_PATH_TYPE=inherit` set (User env) so it inherits the full Windows PATH (git/node/ssh-manager/rg/fd visible).
- `db_home: /%H` in `/etc/nsswitch.conf` unifies MSYS2 `$HOME` directly with `C:\Users\eajdias-note`.
- Provides rsync, jq, sshpass, tree, zip, unzip, openssh; default Windows Terminal profile.

## opencode Configuration (`~\.config\opencode\opencode.jsonc`)

- **shell:** `C:\msys64\usr\bin\bash.exe`
- **LSP servers (16):** typescript, pyright, pylsp, gopls, bash, sql, html, json, yaml, dockerfile, css (vscode-css-language-server), markdown (vscode-markdown-language-server), powershell (pwsh + PowerShellEditorServices), rust (rust-analyzer), csharp (csharp-ls), eslint (vscode-eslint-language-server)
- **MCP servers:** context7 (remote, enabled, CONTEXT7_API_KEY in header); ssh-manager (local, disabled by default); docker-hub (local, disabled by default)
- **Plugin:** `@tarquinen/opencode-dcp@latest` (DCP context compression; config `~\.config\opencode\dcp.jsonc`; `compress` tool in experimental.primary_tools)
- **Skills paths:** `~\.config\opencode\skills`, `~\.agents\skills`
- Config is NOT hot-reloaded — restart opencode after changes. Validate with `opencode debug config` (note: PowerShell `ConvertFrom-Json` fails on jsonc comments — expected).

## Skill Locations

- **opencode skills:** `~\.config\opencode\skills\` (**22 skills**)
- **agent skills:** `~\.agents\skills\` (34 skills)

### opencode skills (22)

| Skill | Purpose |
|-------|---------|
| `git-workflow` | Git & GitHub CLI workflow (semantic branches, conventional commits, PR lifecycle, conflict resolution) |
| `database-ops` | Dynamic multi-database management (PostgreSQL, MySQL, Firebird, MongoDB, SQLite, migrations, Docker DBs) |
| `universal-test-runner` | Multi-stack test runner & coverage (Node/TS, Python, Go, .NET, Rust) with TDD loop |
| `api-contract-design` | API design & validation (OpenAPI/Swagger, GraphQL SDL, gRPC/Protobuf, breaking change checks) |
| `ssh-vps` | SSH/VPS management via ssh-manager (monitoring, recovery) |
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

### Docker quick check

```bash
docker ps -a --format "table {{.Names}}\t{{.Status}}\t{{.Image}}"
```

### Parallel Agent Execution

Use `dispatching-parallel-agents` skill for independent tasks; each agent runs in its own context.

## Notes

- Scripts should be created in `~\` or temp directory, not in system folders; clean up test files after use.
- Playwright uses headless Chromium by default.
- When the agent's bash tool shows 'Windows PowerShell (5.1)', the `shell` config was ignored (opencode issue #41426) — wrap commands with `bash -lc "..."`.