# Princípios e Diretrizes Arquiteturais (envctl)

Este documento estabelece as diretrizes fundamentais que guiam o desenvolvimento do `envctl`.

---

## 1. Princípios de Engenharia

### A. Idempotência e Detecção de Drift
- Cada passo de provisionamento deve implementar os contratos do repositório:
  - `CheckPackage(ctx, target) (bool, string, error)`: Detecta se o componente já está no estado desejado.
  - `InstallPackage(ctx, target) error`: Executa a instalação ou configuração apenas se necessário.
  - `Verify(ctx, target) error`: Valida a integridade pós-instalação (versão, PATH, permissão).

### B. Isolamento de Responsabilidades (Clean Architecture)
- A camada de **Domínio** não possui dependências de pacotes externos, chamadas de sistema operacional diretas ou APIs específicas de plataforma.
- A camada de **Casos de Uso** orquestra o fluxo de negócio (ex: "Instalar Gerenciadores de Pacotes antes de Toolchains", "Configurar Shell antes de LSPs").
- A camada de **Infraestrutura** lida com a realidade suja do sistema operacional (subshells, pipes, registry, códigos de saída de processos, gerenciamento de pacotes por OS).

### C. Self-Contained Binary (`//go:embed`)
- O binário compilado (`envctl` / `envctl.exe`) carrega todos os manifestos declarativos, templates de configuração e as 59 Skills em tempo de compilação.
- Permite execução offline ou via pendrive sem requisições HTTP adicionais para baixar assets de configuração.

---

## 2. Ordem de Precedência do Provisionamento

1. **Bootstrap & Checagem de Ambiente:**
   - Detecção de OS (`windows`, `linux`, `darwin`) e arquitetura (`amd64`, `arm64`).
   - Validação de privilégios e permissões.
2. **Infraestrutura Base de Pacotes:**
   - Windows: Instalação dos pacotes essenciais via `Winget` e `Pacman`.
   - Linux: Instalação dos pacotes essenciais via `APT` / gerenciador de pacotes nativo.
3. **Configuração de Shell e Ambiente:**
   - Variáveis de ambiente (`NODE_PATH`, `MSYS2_ENV_CONV_EXCL`, `MSYS2_ARG_CONV_EXCL`).
   - Git: Otimizações globais (`core.fscache`, `core.preloadindex`, `core.longpaths`, `core.autocrlf input`, `delta`).
   - Terminal: Implantação de `.bashrc`, `.bash_profile`, Oh-My-Posh e `settings.json` do terminal com backup atômico.
4. **Toolchains & Language Servers (LSPs):**
   - Volta: Node.js e pacotes LSP globais (`typescript`, `pyright`, `bash-ls`, `dockerfile-ls`, `yaml-ls`, `sqllens`, etc.).
   - Python: `uv`, `ruff`.
   - Go: `gopls`.
   - .NET: `csharp-ls`.
   - Rust: `rustup component add rust-analyzer`.
5. **Ecossistema OpenCode & 59 Skills:**
   - Implantação de `opencode.jsonc`, `dcp.jsonc`, `tui.json`, `package.json` e `AGENTS.md`.
   - Extração das 59 Skills do OpenCode para `~/.config/opencode/skills` e `~/.agents/skills`.
6. **Verificação, Diagnóstico e Relatório:**
   - Auditoria completa via `envctl doctor` com 160+ checagens diagnósticas e opção de auto-remediação (`--fix`).
