# OpenCode Environment Manifest

## System Architecture

- **OS:** Windows 11 Pro 25H2 (amd64)
- **Workstation:** `eajdias-note` | User: `eajdias-note`
- **Shell Primary:** MSYS2 Bash (`C:\msys64\usr\bin\bash.exe`) — fast POSIX CLI subshell (~42ms startup)
- **Shell Secondary:** PowerShell 7.6.5 (`pwsh.exe`)
- **Package Managers:** Winget (native Windows), MSYS2 Pacman (POSIX utilities), Volta (Node ecosystem), Pip/Uv (Python), Dotnet Tool (.NET)
- **Node Runtime:** Node v24.19.0 managed via Volta (`NODE_PATH="%USERPROFILE%\node_modules"`)
- **Global Tools:** `rg` (ripgrep), `fd`, `fzf`, `bat`, `delta`, `yq`, `ruff`, `gh`, `tree`, `zip/unzip`, `csharp-ls`, `pw-screenshot`, `pw-eval`
- **LSPs Registered:** 16 language servers in `opencode.jsonc` (TypeScript, Pyright, PyLSP, Gopls, Bash, SQL, HTML, JSON, YAML, Dockerfile, CSS, Markdown, PowerShell, Rust Analyzer, CSharp-LS, ESLint)
- **Git Optimizations:** `fscache=true`, `preloadindex=true`, `longpaths=true`, `autocrlf=input`, `delta` pager

## Conventions & Rules

- **Language:** Code/comments/commits in English. User communication in Portuguese (BR).
- **Style:** Clean Architecture, SOLID, idiomatic code per language, strict typing.
- **Git:** Semantic branches (`feat/...`, `fix/...`), conventional commits, PRs via `gh pr create`.
- **Testing:** Evidence before claims — test before declaring complete.
- **Security:** Strict ACLs on `~/Documents/SSH-keys`, `~/.ssh-manager`, `~/.ssh`. Never hardcode secrets.

## OpenCode Configuration

- **Global config:** `~\.config\opencode\opencode.jsonc`
- **Plugin:** `@tarquinen/opencode-dcp@latest` (DCP context compression; config `~\.config\opencode\dcp.jsonc`; `compress` tool in experimental.primary_tools)
- **Skills paths:** `~\.config\opencode\skills`, `~\.agents\skills`
- **Config is NOT hot-reloaded:** restart opencode after changes. Validate with `opencode debug config` (note: PowerShell `ConvertFrom-Json` fails on jsonc comments — expected).

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

### Docker quick check & Container Exec

```bash
docker ps -a --format "table {{.Names}}\t{{.Status}}\t{{.Image}}"

# MSYS2 Path Conversion Note:
# When executing commands inside containers (e.g. /bin/sh) or mounting volumes (-v /host:/container),
# MSYS2 may convert POSIX paths to Windows paths.
# The environment defines MSYS2_ARG_CONV_EXCL and aliases docker='MSYS_NO_PATHCONV=1 docker'.
# In raw subshells or scripts, use:
MSYS_NO_PATHCONV=1 docker exec -it <container> /bin/sh
```

### Parallel Agent Execution

Use `dispatching-parallel-agents` skill for independent tasks; each agent runs in its own context.

## Notes

- Scripts should be created in `~\` or temp directory, not in system folders; clean up test files after use.
- Playwright uses headless Chromium by default.
- When the agent's bash tool shows 'Windows PowerShell (5.1)', the `shell` config was ignored (opencode issue #41426) — wrap commands with `bash -lc "..."`.
