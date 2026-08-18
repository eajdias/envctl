# Princípios e Diretrizes Arquiteturais (win11-new)

Este documento estabelece as diretrizes fundamentais que guiam o desenvolvimento do `win11-new`.

---

## 1. Princípios de Engenharia

### A. Idempotência e Detecção de Drift
- Cada passo de provisionamento deve implementar a interface:
  - `CheckInstalled(ctx, target) (bool, error)`: Detecta se o componente já está no estado desejado.
  - `Install(ctx, target) error`: Executa a instalação ou configuração apenas se necessário.
  - `Verify(ctx, target) error`: Valida a integridade pós-instalação (versão, PATH, permissão).

### B. Isolamento de Responsabilidades (Clean Architecture)
- A camada de **Domínio** não possui dependências de pacotes externos, chamadas de sistema operacional diretas ou APIs do Windows.
- A camada de **Casos de Uso** orquestra o fluxo de negócio (ex: "Instalar Winget antes de Toolchains", "Configurar MSYS2 antes de LSPs").
- A camada de **Infraestrutura** lida com a realidade suja do sistema operacional (subshell, pipes, registry, códigos de saída de processos).

### C. Self-Contained Binary (`//go:embed`)
- O binário `.exe` gerado deve carregar todos os templates de configuração e diretórios de Skills em tempo de compilação.
- Permite execução offline ou via pendrive sem requisições HTTP adicionais para baixar assets de configuração.

---

## 2. Ordem de Precedência do Provisionamento

1. **Bootstrap & Checagem de Ambiente:**
   - Validação de privilégios de Administrador.
   - Detecção de arquitetura (`AMD64`) e versão do Windows (`Windows 11 PRO`).
2. **Infraestrutura Base de Pacotes:**
   - Instalação dos pacotes essenciais via `Winget` (Git, MSYS2, Volta, Go, .NET, Python, Docker Desktop, VS Code).
3. **Configuração de Shell e Ambiente:**
   - MSYS2: Execução de pacman (`rsync`, `jq`, `sshpass`, `tree`, `zip`, `unzip`), ajuste de `/etc/nsswitch.conf` (`db_home: /%H`) e `MSYS2_PATH_TYPE=inherit`.
   - Git: Otimizações globais (`core.fscache`, `core.preloadindex`, `core.longpaths`, `core.autocrlf input`).
   - Terminal: Implantação de `.bashrc`, `.bash_profile`, Oh-My-Posh e `settings.json` do Windows Terminal.
4. **Toolchains & Language Servers (LSPs):**
   - Volta: Fixação de Node.js v24.19 e instalação dos 8 pacotes LSP globais.
   - Python: Instalação de `uv` e `ruff`.
   - Go: `gopls`.
   - .NET: `csharp-ls`.
   - Rust: `rustup component add rust-analyzer`.
5. **Ecossistema OpenCode & Skills:**
   - Implantação de `opencode.jsonc`, `dcp.jsonc` e `AGENTS.md` no `%USERPROFILE%\.config\opencode`.
   - Extração das 22 Skills do OpenCode para `~/.config/opencode/skills`.
   - Extração das 34 Agent Skills para `~/.agents/skills`.
6. **Verificação e Relatório de Integridade:**
   - Smoke test de todos os binários e Language Servers no PATH.
