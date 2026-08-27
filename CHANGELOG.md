# Changelog

Todas as alterações notáveis no projeto **`envctl`** serão documentadas neste arquivo.

O formato é baseado no [Keep a Changelog](https://keepachangelog.com/pt-BR/1.0.0/),
e este projeto adere ao [Versionamento Semântico](https://semver.org/lang/pt-BR/).

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
