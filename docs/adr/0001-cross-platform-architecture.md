# ADR 0001: Arquitetura Cross-Platform e Orquestração de Subagentes Distribuídos

## Status
Aceito (Accepted)

## Contexto
O ecossistema de desenvolvimento e operação de serviços opera em múltiplos sistemas operacionais:
1. Estação de trabalho local no Windows 11 PRO 25H2 com PowerShell 7 como shell primário e WSL Ubuntu como subshell POSIX secundário.
2. Servidores remotos de homologação e produção (VPS Ubuntu 24.04 LTS na AWS EC2 e Oracle Cloud) executando contêineres Docker, bancos de dados e serviços backend.

Anteriormente, a replicação desses ambientes dependia de scripts manuais esparsos ou ferramentas com alto consumo de contexto para agentes de IA.

## Decisões Arquiteturais

### 1. Clean Architecture em Go
- **Domínio Puro**: Entidades `Package`, `ConfigFile`, `Skill`, `LSP`, `Diagnostic` e interfaces de repositório isoladas de detalhes de sistema operacional.
- **Multi-Gerenciadores de Pacotes**: Adaptadores modulares para `Winget`, `APT`, `Volta`, `Go`, `Rustup`, `Dotnet Tool`, e `UV/Pip`.
- **Filtro Declarativo por SO**: Suporte a campo `os` nos manifestos YAML para provisionar apenas pacotes aplicáveis à plataforma de execução (`windows`, `linux`, `darwin`).

### 2. Standalone Self-Contained Binary (`//go:embed`)
- Todos os manifestos declarativos (`manifests/`), templates de configuração (`configs/`) e as **59 Skills** de agentes de IA são embutidos diretamente no binário compilado.
- Garante instalação determinística e offline com zero dependências externas ou requisições HTTP adicionais em tempo de execução.

### 3. Orquestração Distribuída via `vps-agent-dispatch`
- A máquina local (notebook) atua como coordenadora global.
- Tarefas pesadas (diagnóstico de contêineres, compilação de imagens, suítes de testes headless) são despachadas via SSH para instâncias remotas do OpenCode nas VPSs (`opencode run`).
- O resultado é cristalizado e retornado para a máquina local, mantendo a janela de contexto local enxuta e de alto sinal.

### 4. CI/CD e Releases Automatizados por Push na Main
- GitHub Actions valida compilação e suíte de testes em Linux e Windows a cada Pull Request.
- A cada merge/push na branch `main`, é gerada uma nova Release pública no GitHub com binários pré-compilados para Windows (amd64, arm64), Linux (amd64, arm64) e macOS (amd64, arm64).

## Consequências
- **Positivas**:
  - Replicação de ambiente 1:1 realizável em menos de 2 minutos em qualquer máquina nova via one-liner (`bootstrap.ps1` ou `bootstrap.sh`).
  - Idempotência absoluta e auditoria via `envctl doctor` com 160+ checagens e auto-remediação (`--fix`).
  - Trilha de auditoria persistente gerada em disco a cada execução (`~/.envctl/logs/`).
