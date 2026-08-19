# Changelog

Todas as alterações notáveis no projeto **`envctl`** serão documentadas neste arquivo.

O formato é baseado no [Keep a Changelog](https://keepachangelog.com/pt-BR/1.0.0/),
e este projeto adere ao [Versionamento Semântico](https://semver.org/lang/pt-BR/).

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
- **Added**: Variável de ambiente `MSYS2_ENV_CONV_EXCL=NODE_PATH` para garantir integridade na resolução de módulos Node entre Windows e MSYS2.
- **Added**: `configs/bin/pw-screenshot` e `configs/bin/pw-screenshot.cmd` para captura instantânea de telas headless em alta resolução.
- **Added**: `configs/bin/pw-eval` e `configs/bin/pw-eval.cmd` para avaliação rápida de DOM e scripts via Playwright Node.js API em < 1s.
- **Added**: Aliases de `git worktree` (`gwc`, `gwl`, `gwr`) e auditoria de integridade de worktrees no `doctor`.
- **Added**: Template robusto de `~/.ssh/config` com multiplexação de sockets (`ControlMaster auto`, `ControlPath ~/.ssh/sockets/%r@%h:%p`) e keepalive.

---

## [v1.0.7] - [v1.0.9] - 2026-08-18

### 🌐 Playwright Node API Migration & MSYS2 Path Conversion
- **Changed**: Substituição da CLI instável `@playwright/cli` pela arquitetura estável Node.js API (`const { chromium } = require('playwright')`).
- **Added**: Resolução dinâmica de `NODE_PATH` em `configs/.bashrc` e `/etc/profile.d/node_path.sh` utilizando caminhos mistos (`cygpath -m`).
- **Added**: `configs/user-package.json` gerenciando a dependência do `playwright` na raiz do usuário (`~`).
- **Added**: Variável `MSYS2_ARG_CONV_EXCL` e wrappers `MSYS_NO_PATHCONV=1` nos aliases do Docker (`docker`, `docker-compose`, `kubectl`) para impedir que o MSYS2 corrompa caminhos e volumes de containers.

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
- **Added**: Gerenciadores de infraestrutura para `Winget`, `MSYS2 Pacman`, `Dotnet Tool`, `Git` e `FileSystem`.
- **Added**: Backup atômico com timestamp (`.bak.YYYYMMDD-HHMMSS`) para alterações em arquivos de configuração existentes.
- **Added**: Catálogo inicial de 57 skills de agentes de IA para o OpenCode.
- **Added**: Comandos CLI Cobra com interface ANSI via PTerm (`run`, `doctor`, `snapshot`, `version`).
