# envctl (Cross-Platform Environment Provisioner & State Replicator)

> Provisionador idempotente, determinístico e auditável em **Go** para replicação 1:1 de ambientes de desenvolvimento e servidores no **Windows 11 PRO** e **Ubuntu / Debian Linux**.

---

## 🎯 Objetivo
Transformar estações de trabalho recém-formatadas (Windows 11) ou servidores remotos (VPS Ubuntu / Debian na AWS / Oracle Cloud) em ambientes de desenvolvimento e produção completos, padronizados e idênticos:
- **Shell e Utilitários de Alta Performance**: MSYS2 Bash no Windows, Zsh/Bash no Linux, ripgrep, fd, fzf, bat, tree, delta, yq, rsync, jq.
- **Toolchains Completas**: Node.js (via Volta), Python (Python 3.14 + uv + ruff), Go, .NET SDK, Rust (rustup), Docker.
- **Language Server Protocol (16 LSPs)**: TypeScript, Pyright, Gopls, Bash-LS, Sqllens, Marksman, CSharp-LS, Rust-Analyzer, etc.
- **Ecossistema OpenCode**: Configurações globais (`opencode.jsonc`, `dcp.jsonc`, `tui.json`, plugins), e todas as **59 Skills** de agentes catalogadas e sincronizadas.
- **Orquestração Remota de Subagentes**: Skill `vps-agent-dispatch` para delegar tarefas autônomas do notebook para servidores remotos via SSH com economia máxima de contexto.

---

## 🚀 Como Instalar e Rodar (Zero Pré-requisitos)

### No Windows 11 (PowerShell)
```powershell
irm https://raw.githubusercontent.com/eajdias/envctl/main/bootstrap.ps1 | iex
```

### No Linux (Ubuntu / Debian / Oracle / AWS)
```bash
curl -fsSL https://raw.githubusercontent.com/eajdias/envctl/main/bootstrap.sh | bash
```

---

## 💻 Subcomandos Principais

```bash
# Provisionamento completo de todo o ecossistema (Day-0)
envctl run all

# Provisionamento por subsistema modular
envctl run winget       # Apenas pacotes winget (Windows)
envctl run apt          # Apenas pacotes APT (Ubuntu/Debian)
envctl run pacman       # Apenas pacotes MSYS2
envctl run volta        # Node.js e ferramentas globais via Volta
envctl run shell        # Variáveis de ambiente, perfis e configurações
envctl run skills       # Extração e validação das 59 Skills de IA
envctl run lsp          # Servidores de linguagem (LSP)
envctl run windows      # Tweaks de registro, Developer Mode e fontes

# Auditoria e Diagnóstico de Saúde (160+ pontos de checagem)
envctl doctor

# Auto-remediação automática de avisos/erros detectados
envctl doctor --fix

# Snapshot e Sincronização Reversa (Day-2)
envctl snapshot
```

---

## 🏛️ Arquitetura de Software (Clean Architecture)

```
envctl/
├── cmd/
│   └── envctl/                  # Entrypoint da aplicação (main.go, injeção de dependências)
├── internal/
│   ├── domain/                  # Camada de Domínio (Entidades e Interfaces/Contratos)
│   │   ├── entity/              # Entidades puras: Package, ConfigFile, Skill, LSP, WindowsTweak, Diagnostic
│   │   └── repository/          # Interfaces: PackageManager, FileSystemManager, WindowsTweaksManager, Logger
│   ├── usecase/                 # Casos de Uso da Aplicação
│   │   ├── provision_packages.go# Instalador multi-gerenciador de pacotes
│   │   ├── provision_shell.go   # Provisionador de shell, variáveis e configs com backup atômico
│   │   ├── provision_skills.go  # Extração e atualização das 59 Skills
│   │   ├── provision_lsp.go     # Instalação e validação dos 16 LSPs
│   │   ├── provision_system.go  # Customizações de sistema e registro (Windows)
│   │   ├── doctor_audit.go      # Auditoria diagnóstica de conformidade
│   │   └── snapshot_sync.go     # Sincronizador reverso e criador de PR no GitHub
│   ├── infra/                   # Camada de Infraestrutura (Implementações concretas)
│   │   ├── winget/              # Adaptador para Windows Package Manager
│   │   ├── apt/                 # Adaptador para APT (Debian/Ubuntu)
│   │   ├── msys2/               # Adaptador para MSYS2 Pacman
│   │   ├── toolchain/           # Adaptadores para Volta, Go, Rustup, Dotnet, UV/Pip
│   │   ├── windows/             # Adaptador de Registro e Fontes Windows
│   │   ├── git/                 # Adaptador Git e GitHub CLI
│   │   ├── filesystem/          # Operações de I/O, backup atômico (.bak.timestamp) e ACLs
│   │   ├── logger/              # Logger persistente com dump em disco (~/.envctl/logs/)
│   │   └── embedded/            # Sistema de arquivos embutido no binário (//go:embed)
│   └── ui/                      # Interface com o Usuário
│       └── cli/                 # Comandos Cobra e Interface Rica em ANSI via PTerm
├── manifests/                   # Manifestos declarativos YAML (packages, git, shell, skills, lsp, windows)
├── configs/                     # Templates de configuração embutidos
└── docs/                        # Documentação técnica e ADRs
```

---

## 💎 Princípios & Garantias
1. **100% Standalone via `//go:embed`**: Todas as 59 Skills e templates de configuração residem dentro do próprio binário executável compilado.
2. **Idempotência Estrita**: Executar o utilitário 1 ou 100 vezes produz o mesmo estado final estável sem reinstalações redundantes.
3. **Backup Atômico com Timestamp**: Qualquer arquivo de configuração existente sofre backup seguro (`.bak.YYYYMMDD-HHMMSS`) antes de modificações.
4. **Logging Persistente Completo**: Cada execução gera uma trilha de auditoria completa em `~/.envctl/logs/envctl-YYYYMMDD-HHMMSS.log` capturando comandos, saídas e decisões de idempotência.
5. **Zero Segredos**: Chaves e credenciais nunca residem no repositório; o ecossistema cria as pastas restritas com ACLs adequadas e instrui a injeção manual segura.
