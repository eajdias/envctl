# Arquitetura de Software do envctl

O `envctl` foi projetado utilizando os preceitos fundamentais da **Clean Architecture** (Arquitetura Limpa), visando desacoplamento total entre regras de negócio/domínio e as complexidades específicas de cada sistema operacional (Windows, Linux, macOS) e gerenciadores de pacotes.

---

## 🏛️ Visão Estrutural das Camadas

```
envctl/
├── cmd/
│   └── envctl/                  # Entrypoint da aplicação (main.go, injeção de dependências)
├── internal/
│   ├── domain/                  # Camada de Domínio (Entidades e Interfaces/Contratos)
│   │   ├── entity/              # Modelos puros: Package, ConfigFile, Skill, LSP, WindowsTweak, Diagnostic
│   │   └── repository/          # Interfaces: PackageManager, FileSystemManager, WindowsTweaksManager, Logger
│   ├── usecase/                 # Casos de Uso da Aplicação
│   │   ├── provision_packages.go# Instalador multi-gerenciador de pacotes
│   │   ├── provision_shell.go   # Provisionador de shell, variáveis e configs com backup atômico
│   │   ├── provision_skills.go  # Extração e atualização das 59 Skills
│   │   ├── provision_lsp.go     # Instalação e validação dos 16 LSPs
│   │   ├── provision_system.go  # Customizações de sistema e registro (Windows)
│   │   ├── doctor_audit.go      # Auditoria diagnóstica de conformidade (160+ checagens)
│   │   └── snapshot_sync.go     # Sincronizador reverso e criador de PR no GitHub
│   ├── infra/                   # Camada de Infraestrutura (Implementações concretas)
│   │   ├── winget/              # Adaptador para Windows Package Manager
│   │   ├── apt/                 # Adaptador para APT (Debian/Ubuntu)
│   │   ├── toolchain/           # Adaptadores para Volta, Go, Rustup, Dotnet, UV/Pip
│   │   ├── windows/             # Adaptador de Registro e Fontes Windows
│   │   ├── git/                 # Adaptador Git e GitHub CLI
│   │   ├── filesystem/          # Operações de I/O, backup atômico (.bak.timestamp) e ACLs
│   │   ├── logger/              # Logger persistente com dump em disco (~/.envctl/logs/)
│   │   └── embedded/            # Sistema de arquivos embutido no binário (//go:embed)
│   └── ui/                      # Interface com o Usuário
│       └── cli/                 # Comandos Cobra e Interface Rica em ANSI via PTerm
├── manifests/                   # Manifestos declarativos YAML
├── configs/                     # Templates de configuração embutidos
└── docs/                        # Documentação técnica e Guias por OS
```

---

## 🧩 Detalhamento das Camadas

### 1. Camada de Domínio (`internal/domain`)
- **Entidades (`entity/models.go`)**: Modelos puros sem dependências externas.
  - `Package`: Representa um pacote a ser instalado, seu tipo (`winget`, `apt`, `volta`, `dotnet-tool`, `go`, `rustup`, `pip`), binário esperado e filtro de OS.
  - `ConfigFile`: Arquivo de configuração gerenciado, permissões esperadas e caminho expandido.
  - `Skill`: Skill de agente de IA (OpenCode), metadados e arquivos de referência associados.
  - `LSP`: Servidor de linguagem (Language Server Protocol), gerenciador de pacote nativo e linguagens suportadas.
  - `WindowsTweak`: Chave de registro, recurso opcional ou fonte de sistema.
  - `Diagnostic`: Item de auditoria do subsistema `doctor` com severidade (`OK`, `WARN`, `ERROR`).
- **Repositórios e Contratos (`repository/interfaces.go`)**:
  - `PackageManager`: Interface universal com `CheckPackage`, `InstallPackage` e `VerifyPackage`.
  - `FileSystemManager`: Operações com caminho dinâmico (`~`, `%VAR%`, `$VAR`), backup atômico e restrição de permissões (Windows ACLs via `icacls` ou POSIX `chmod`).
  - `WindowsTweaksManager`: Gestão de tweaks de registro e instalação de fontes.
  - `ManifestRepository`: Carga e persistência dos manifestos declarativos.
  - `Logger`: Gravação assíncrona/síncrona de eventos com trilha de auditoria em arquivo.

### 2. Camada de Casos de Uso (`internal/usecase`)
Orquestra o fluxo de negócio do provisionador sem acoplamento a implementações concretas:
- **`ProvisionPackagesUseCase`**: Itera pelos manifestos, filtra pelo OS corrente (`runtime.GOOS`) e orquestra a instalação em lote chamando os adaptadores específicos.
- **`ProvisionShellUseCase`**: Configura variáveis de ambiente globais, copia arquivos com backup atômico, instala dependências e executa hooks pós-instalação (ex: download do Chromium para Playwright).
- **`ProvisionSkillsUseCase`**: Extrai as 59 Skills do sistema embutido para o diretório local do OpenCode (`~/.config/opencode/skills/`).
- **`ProvisionLSPsUseCase`**: Garante a presença dos 16 servidores de linguagem registrados.
- **`ProvisionSystemUseCase`**: Aplica ajustes de registro, Developer Mode e fontes no Windows (ignorado de forma segura em Linux/macOS).
- **`DoctorAuditUseCase`**: Executa uma bateria de 160+ checagens diagnósticas cobrindo todo o ecossistema.
- **`SnapshotSyncUseCase`**: Lê o estado vivo da máquina e sincroniza manifestos e configs localmente (sem automação de git/PR).

### 3. Camada de Infraestrutura (`internal/infra`)
Implementa os adaptadores para os sistemas operacionais e ferramentas CLI:
- **Execução Segura de Subprocessos**: Utiliza timeouts contextuais, sanitização de saídas (remoção de null bytes de consoles UTF-16) e logging de stdout/stderr.
- **Gerenciadores de Pacotes Concretos**:
  - `WingetManager`: `winget.exe install --exact --id ... --silent --accept-package-agreements`
  - `AptManager`: `apt-get install -y --no-install-recommends ...`
  - `VoltaManager`: `volta install ...`
  - `DotnetToolManager`: `dotnet tool install --global ...`
  - `GoManager`: `go install ...@latest`
  - `RustupManager`: `rustup component add ...`
  - `PipManager`: `pip install ...` / `uv pip install ...`
- **Filesystem Atômico**: Cria backups com formato `.bak.YYYYMMDD-HHMMSS` antes de modificar qualquer arquivo existente em disco caso o hash SHA-256 do conteúdo tenha divergido.

### 4. Camada de Apresentação & UI (`internal/ui/cli`)
- Desenvolvida com **Cobra CLI** e **PTerm**.
- Suporta ANSI styling, spinners de progresso, tabelas formatadas e árvore diagnóstica visual.
- Suporta autocompletar nativo para Bash, Zsh, Fish e PowerShell.

---

## 📦 Binário 100% Standalone (`//go:embed`)

Todo o ecossistema de manifestos (`manifests/*.yaml`) e templates de configuração (`configs/**/*`) é compilado diretamente dentro do binário Go através do pacote padrão `embed`:

```go
package envctl

import "embed"

//go:embed all:manifests
var EmbeddedManifests embed.FS

//go:embed all:configs
var EmbeddedConfigs embed.FS
```

Isso garante que o binário gerado (`envctl` ou `envctl.exe`) seja totalmente autônomo, não dependendo de conexão de rede ou arquivos externos no momento do provisionamento inicial.
