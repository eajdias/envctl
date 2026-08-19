# Manifestos Declarativos do envctl

O `envctl` é orientado a **infraestrutura declarativa como código** (IaC). Todas as ferramentas, variáveis de ambiente, servidores de linguagem, skills de IA e ajustes de sistema são definidos em arquivos YAML na pasta `manifests/`.

---

## 📁 Estrutura dos Manifestos

```
manifests/
├── packages.yaml    # Pacotes de sistema, toolchains e aplicativos de produtividade
├── git.yaml         # Otimizações de performance e configurações globais do Git
├── shell.yaml       # Variáveis de ambiente, diretórios protegidos e templates de arquivo
├── lsp.yaml         # Servidores de linguagem (LSP) para IDEs e OpenCode
├── skills.yaml      # Catálogo e mapeamento das 59 Skills de Agentes de IA
└── windows.yaml     # Tweaks de registro, Developer Mode e fontes do Windows 11
```

---

## 📄 1. `manifests/packages.yaml`

Define todos os pacotes gerenciados, seus tipos, binários de teste e filtros de sistema operacional.

```yaml
packages:
  # Pacotes de Sistema Windows via Winget
  - name: BurntSushi.ripgrep.MSVC
    type: winget
    test_binary: rg
    os: windows
    description: "Ripgrep - Busca de texto ultra-rápida"

  # Pacotes de Sistema Linux via APT
  - name: ripgrep
    type: apt
    test_binary: rg
    os: linux
    description: "Ripgrep nativo para Ubuntu/Debian"

  # Pacotes MSYS2 via Pacman
  - name: tree
    type: pacman
    test_binary: tree
    os: windows
    description: "Tree utilitário para visualização de diretórios"

  # Toolchain Node.js via Volta
  - name: node@24.19.0
    type: volta
    test_binary: node
    description: "Node.js LTS runtime gerenciado pelo Volta"

  - name: firecrawl-cli
    type: volta
    test_binary: firecrawl
    description: "CLI oficial do Firecrawl para raspagem e busca web"

  # Ferramentas .NET
  - name: csharp-ls
    type: dotnet-tool
    test_binary: csharp-ls
    description: "Language Server para C# / .NET"

  # Ferramentas Go
  - name: golang.org/x/tools/gopls@latest
    type: go
    test_binary: gopls
    description: "Language Server oficial para Go"

  # Componentes Rust via Rustup
  - name: rust-analyzer
    type: rustup
    test_binary: rust-analyzer
    description: "Language Server oficial para Rust"
```

### Tipos de Gerenciadores Suportados (`type`):
| Tipo | Gerenciador | Comando de Instalação |
| :--- | :--- | :--- |
| `winget` | Windows Package Manager | `winget install --exact --id <name> --silent` |
| `apt` | Advanced Package Tool (Debian/Ubuntu) | `apt-get install -y --no-install-recommends <name>` |
| `pacman` | MSYS2 Pacman | `pacman -S --needed --noconfirm <name>` |
| `volta` | Volta Toolchain Manager | `volta install <name>` |
| `dotnet-tool`| .NET CLI Global Tools | `dotnet tool install --global <name>` |
| `go` | Go Toolchain | `go install <name>` |
| `rustup` | Rustup Component Manager | `rustup component add <name>` |
| `pip` | Python PIP / UV | `pip install <name>` |

---

## 📄 2. `manifests/shell.yaml`

Define variáveis de ambiente, diretórios restritos e o mapeamento de templates de configuração para o sistema de arquivos do usuário.

```yaml
env_vars:
  - name: NODE_PATH
    value: "%USERPROFILE%\\node_modules"
    target: User
    os: windows
    description: "Resolução global de módulos Node.js para scripts de automação"

  - name: MSYS2_ENV_CONV_EXCL
    value: "NODE_PATH"
    target: User
    os: windows
    description: "Impede corrupção de caminhos Windows no MSYS2"

  - name: MSYS2_ARG_CONV_EXCL
    value: "/bin;/usr;/var;/etc;/app;/tmp;/opt;--entrypoint;-v;--volume;--mount;--workdir;-w"
    target: User
    os: windows
    description: "Preserva caminhos e argumentos em comandos Docker"

config_files:
  - source: configs/.bashrc
    destination: ~/.bashrc
    description: "Aliases unificados e configuração do shell Bash"

  - source: configs/opencode.jsonc
    destination: ~/.config/opencode/opencode.jsonc
    description: "Configuração central do OpenCode com LSPs e MCPs"

restricted_dirs:
  - path: ~/Documents/SSH-keys
    mode: "0700"
    os: windows
    description: "Chaves privadas SSH com permissões restritas (ACLs)"
  - path: ~/.ssh/sockets
    mode: "0700"
    description: "Sockets de multiplexação de conexões SSH"
```

---

## 📄 3. `manifests/git.yaml`

Define configurações globais do Git com foco em máxima performance em repositórios massivos e sistemas Windows/Linux:

```yaml
git_configs:
  - key: core.fscache
    value: "true"
    description: "Habilita cache do sistema de arquivos para operações Git ultra-rápidas"
  - key: core.preloadindex
    value: "true"
    description: "Pré-carrega o índice em paralelo durante operações de status/diff"
  - key: core.longpaths
    value: "true"
    description: "Permite caminhos longos (> 260 caracteres) no Windows"
  - key: core.autocrlf
    value: "input"
    description: "Converte CRLF para LF no commit, mantendo LF no checkout"
  - key: core.pager
    value: "delta"
    description: "Configura o Delta como pager padrão para diffs estruturados"
```

---

## 📄 4. `manifests/lsp.yaml`

Registra os 16 servidores de linguagem utilizados por agentes de IA e IDEs, associando cada um ao seu gerenciador nativo:

```yaml
lsps:
  - name: typescript-language-server
    package_type: volta
    command: typescript-language-server --stdio
    languages: [typescript, javascript, typescriptreact, javascriptreact]
  - name: pyright
    package_type: volta
    command: pyright-langserver --stdio
    languages: [python]
  - name: gopls
    package_type: go
    command: gopls
    languages: [go]
  - name: rust-analyzer
    package_type: rustup
    command: rust-analyzer
    languages: [rust]
  - name: csharp-ls
    package_type: dotnet-tool
    command: csharp-ls
    languages: [csharp]
  - name: marksman
    package_type: winget
    command: marksman server
    languages: [markdown]
```

---

## 📄 5. `manifests/windows.yaml`

Define ajustes de registro do Windows 11 para desenvolvedores, visualização do Windows Explorer, modo escuro e fontes tipográficas:

```yaml
tweaks:
  - name: "Win32 Long Paths"
    path: "HKLM\\SYSTEM\\CurrentControlSet\\Control\\FileSystem"
    key: "LongPathsEnabled"
    type: "DWord"
    value: 1
    description: "Remove o limite clássico de 260 caracteres no Windows"

  - name: "Developer Mode"
    path: "HKLM\\SOFTWARE\\Microsoft\\Windows\\CurrentVersion\\AppModelUnlock"
    key: "AllowDevelopmentWithoutDevLicense"
    type: "DWord"
    value: 1
    description: "Habilita criação de symlinks sem privilégios de Administrador"

  - name: "Show File Extensions"
    path: "HKCU\\Software\\Microsoft\\Windows\\CurrentVersion\\Explorer\\Advanced"
    key: "HideFileExt"
    type: "DWord"
    value: 0
    description: "Exibe sempre as extensões de arquivo no Explorer"

  - name: "MesloLGM Nerd Font"
    type: "Font"
    name: "Meslo"
    description: "Instala fonte com suporte completo a glifos e ícones no terminal"
```
