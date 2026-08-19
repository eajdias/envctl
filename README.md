# envctl

> **Cross-Platform Environment Provisioner & Autonomous State Replicator**  
> Provisionador determinístico, idempotente e auditável em **Go** para replicação 1:1 de ambientes de desenvolvimento e servidores.

O projeto evoluiu de um script inicial de prototipagem focado exclusivamente em Windows 11 (`win11-new`) para um **ecossistema de provisionamento declarativo e orquestrador de subagentes autônomos multiplataforma em Go (`envctl`)**, cobrindo estações de trabalho e servidores remotos (Windows 11 PRO, Ubuntu / Debian Linux na AWS e Oracle Cloud, e macOS).  Ele transforma qualquer VPS remota em um trabalhador autônomo de IA via a skill vps-agent-dispatch e o OpenCode.

---

## ⚡ Início Rápido (Comando de 1 Linha)

### 🪟 Windows 11 (PowerShell)
```powershell
irm https://raw.githubusercontent.com/eajdias/envctl/main/bootstrap.ps1 | iex
```

### 🐧 Linux (Ubuntu / Debian / Servidores VPS)
```bash
curl -fsSL https://raw.githubusercontent.com/eajdias/envctl/main/bootstrap.sh | bash
```

### 🍎 macOS (Terminal)
```bash
curl -fsSL https://raw.githubusercontent.com/eajdias/envctl/main/bootstrap.sh | bash
```

---

## 🎯 O que o `envctl` Configura Automaticamente?

- **Shell & Utilitários de Alta Performance**: MSYS2 Bash / Zsh com `ripgrep`, `fd`, `fzf`, `bat`, `delta`, `tree`, `yq`, `jq`, `rsync`.
- **Toolchains Completas**: Node.js LTS (via Volta), Python 3.14 (`uv` + `ruff`), Go, .NET SDK, Rust (`rustup`), Docker CLI.
- **Language Server Protocol (16 LSPs)**: TypeScript, Pyright, Gopls, Bash-LS, Sqllens, Marksman, CSharp-LS, Rust-Analyzer, etc.
- **Ecossistema OpenCode & 59 Skills**: `opencode.jsonc`, `dcp.jsonc`, `tui.json`, plugins e **59 Skills de Agentes de IA** embutidas.
- **Orquestração de Subagentes Remotos**: Skill `vps-agent-dispatch` para delegar tarefas autônomas para servidores VPS via SSH.
- **Navegador Headless Playwright**: Scripts utilitários `pw-eval` e `pw-screenshot` prontos para automação web instantânea.

---

## 💻 Comandos Principais

```bash
# Provisionamento completo do ecossistema (Day-0)
envctl run all

# Auditoria e diagnóstico de saúde (160+ verificações)
envctl doctor

# Auto-remediação automática de qualquer divergência
envctl doctor --fix

# Provisionamento por subsistema modular
envctl run winget       # Pacotes Winget (Windows)
envctl run apt          # Pacotes APT (Debian/Ubuntu)
envctl run pacman       # Pacotes MSYS2
envctl run volta        # Node.js e ferramentas globais
envctl run shell        # Variáveis de ambiente, perfis e configs
envctl run skills       # Extração e sincronização das 59 Skills
envctl run lsp          # 16 Servidores de Linguagem (LSP)
envctl run windows      # Tweaks de registro, Developer Mode e fontes

# Snapshot reverso e sincronização de estado (Day-2)
envctl snapshot
```

---

## 📚 Documentação Técnica Completa

Para guias passo a passo detalhados, arquitetura e especificações:

### 📖 Guias de Execução por Sistema Operacional:
- 🪟 [**Guia Windows 11 PRO**](docs/guides/windows.md) — Instalação via PowerShell, binários `.exe`, ajustes de registro, MSYS2 e caminhos Docker.
- 🐧 [**Guia Linux (Ubuntu/Debian/VPS)**](docs/guides/linux.md) — Execução em servidores remotos, instâncias AWS/Oracle, orquestração de subagentes e WSL2.
- 🍎 [**Guia macOS (Darwin)**](docs/guides/macos.md) — Execução em Apple Silicon (M-series) e Intel.

### 🏛️ Engenharia & Especificações:
- 🏗️ [**Arquitetura de Software**](docs/architecture.md) — Clean Architecture, camadas internas, abstração de I/O e binário standalone (`//go:embed`).
- 📋 [**Manifestos Declarativos**](docs/manifests.md) — Estrutura e customização dos schemas YAML (`packages.yaml`, `shell.yaml`, `git.yaml`, `lsp.yaml`, `windows.yaml`).
- 🤖 [**Catálogo de Skills & Subagentes**](docs/skills.md) — As 59 skills catalogadas, orquestração remota (`vps-agent-dispatch`) e automação Playwright.
- 🩺 [**Doctor, Idempotência & Logs**](docs/doctor-and-idempotency.md) — 160+ pontos de diagnóstico, flag `--fix`, backups atômicos (`.bak.timestamp`) e trilha de auditoria em `~/.envctl/logs/`.
- 📐 [**Princípios & Decisões Arquiteturais (ADRs)**](docs/principles.md) — Diretrizes de idempotência, isolamento e contratos de repositório.

---

## 💎 Princípios Fundamentais
1. **100% Standalone via `//go:embed`**: Todas as 59 Skills e templates residem dentro do próprio binário executável compilado.
2. **Idempotência Estrita**: Executar 1 ou 100 vezes produz o mesmo estado final estável sem reinstalações redundantes.
3. **Backup Atômico com Timestamp**: Arquivos modificados sofrem backup automático (`.bak.YYYYMMDD-HHMMSS`) caso haja divergência de hash.
4. **Logging Persistente Estruturado**: Trilha de auditoria completa gerada em `~/.envctl/logs/envctl-YYYYMMDD-HHMMSS.log`.
5. **Zero Segredos**: Chaves e credenciais nunca residem no repositório; permissões seguras são aplicadas via ACLs e POSIX permissions.
