# OpenCode Environment Manifest (Linux Server)

## System Architecture

- **OS:** Ubuntu Server (LTS) (amd64) — provisioned by envctl
- **User:** `ubuntu` (non-root, passwordless sudo)
- **Shell Primary:** Bash (`/bin/bash`)
- **Package Managers:** APT (system packages), Volta (Node ecosystem), npm, pip/uv (Python), cargo/rustup (Rust, optional)
- **Node Runtime:** Node v24 LTS managed via Volta (`~/.volta`)
- **Global Tools:** `rg` (ripgrep), `fd` (via `fdfind` symlink), `fzf`, `bat`, `delta`, `yq`, `gh`, `uv`, `ruff`, `tree`, `zip/unzip`, `oh-my-posh`
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

## OpenCode Configuration

- **Global config:** `~/.config/opencode/opencode.json` (padrão único — JSON, não JSONC)
- **Global rules:** `~/.config/opencode/AGENTS.md` — auto-carregado em todas as sessões opencode (este arquivo)
- **Plugin:** `@tarquinen/opencode-dcp@latest` (DCP context compression; config `~/.config/opencode/dcp.jsonc`; `compress` tool in experimental.primary_tools)
- **Skills paths:** `~/.config/opencode/skills` (fonte única — sem duplicatas)
- **Agent Memory (OBRIGATÓRIO — ativo em TODA tarefa):** a skill `agent-memory` deve ser CARREGADA (tool `skill` com name `agent-memory`) e seus arquivos LIDOS no INÍCIO de qualquer tarefa — antes de qualquer exploração/código: `.opencode/memory/lessons.md` e `.opencode/memory/patterns.md` do projeto (se existirem) + `~/.config/opencode/memory/lessons.md` e `~/.config/opencode/memory/patterns.md` globais. Isso vale para TODOS os agentes/subagentes (task, explore, general, etc.). Ao final da tarefa (ou ao cometer erro / ser corrigido / descobrir padrão), GRAVE a lição/pattern no arquivo correspondente — não deixe para depois. NUNCA repita lições registradas. Memórias globais e `.opencode/memory` de projeto são individuais por máquina: **nunca versionar memórias globais**; `.opencode/memory/*.md` de projeto é versionável apenas SEM dados privados (skill `agent-memory`).
- **Config is NOT hot-reloaded:** restart opencode after changes. Validate with `opencode debug config`.

## Service Management

- **systemd:** `systemctl status <svc>`, `journalctl -u <svc> --no-pager -n 50`, `sudo systemctl restart <svc>`
- **Docker:** `docker ps -a`, `docker logs --tail 50 <ctr>`, `docker restart <ctr>`
- **PM2:** `pm2 list`, `pm2 logs <id> --lines 50`, `pm2 restart <id>`

## Common Patterns

### Privileged operations (non-root user)

```bash
sudo -n apt-get update
sudo -n systemctl restart nginx
```

### Running Firecrawl Commands

```bash
firecrawl scrape "https://example.com"
firecrawl search "query"
firecrawl crawl "https://example.com" --limit 100
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

## Temp & Scratch Hygiene (Mandatory)

- **Pasta de scratch padrão dos agentes LLM: `/temp`** (variável `ENVCTL_TEMP` — criada pelo envctl na raiz do disco, SEM relação com o OpenCode). Todo arquivo temporário criado por agentes — scripts do Playwright CLI, downloads, builds, extrações — **DEVE** ir para `/temp`, nunca para pastas do opencode, do projeto ou do sistema.
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
