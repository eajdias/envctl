# Manifestos Declarativos do envctl

O `envctl` é orientado a **infraestrutura declarativa como código** (IaC). Todas as ferramentas, variáveis de ambiente, servidores de linguagem, skills de IA e ajustes de sistema são definidos em arquivos YAML na pasta `manifests/`.

---

## 📁 Estrutura dos Manifestos

```
manifests/
├── packages.yaml    # Pacotes de sistema, toolchains e aplicativos de produtividade
├── performance_ubuntu.yaml # Perfil opt-in Ubuntu Server 24.04+ (zram + sysctl)
├── performance_cachyos.yaml # Perfil opt-in CachyOS (zram; sem tuning genérico)
├── git.yaml         # Otimizações de performance e configurações globais do Git
├── shell.yaml       # Variáveis de ambiente, diretórios protegidos e templates de arquivo
├── lsp.yaml         # Servidores de linguagem (LSP) para IDEs e OpenCode
├── skills.yaml      # Catálogo das 45 Skills de Agentes de IA (com escopo por ambiente)
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
    os: debian,ubuntu
    description: "Ripgrep nativo para Ubuntu/Debian"

  # Toolchain Node.js via Volta
  - name: node@24.19.0
    type: volta
    test_binary: node
    description: "Node.js LTS runtime gerenciado pelo Volta"

  - name: pnpm
    type: volta
    test_binary: pnpm
    description: "Gerenciador de pacotes Node.js gerenciado pelo Volta"

  # Ferramentas Go
  - name: golang.org/x/tools/gopls@latest
    type: go
    test_binary: gopls
    description: "Language Server oficial para Go"
```

### Tipos de Gerenciadores Suportados (`type`):
| Tipo | Gerenciador | Comando de Instalação |
| :--- | :--- | :--- |
| `winget` | Windows Package Manager | `winget install --exact --id <name> --silent` |
| `apt` | Advanced Package Tool (Debian/Ubuntu) | `apt-get install -y --no-install-recommends <name>` |
| `volta` | Volta Toolchain Manager | `volta install <name>` |
| `go` | Go Toolchain | `go install <name>` |
| `pip` | Python PIP / UV | `pip install <name>` |

---

## 📄 2. `manifests/performance_ubuntu.yaml` e `performance_cachyos.yaml`

Perfis de performance são carregados somente pelo comando explícito
`envctl run performance`; eles não fazem parte de `run all` nem de
`doctor --fix`. O seletor exige o ID e a versão exatos do sistema:

- Ubuntu `>= 24.04` usa `performance_ubuntu.yaml`;
- CachyOS usa `performance_cachyos.yaml`;
- Debian, Ubuntu antigo e Arch genérico são rejeitados.

Entradas Ubuntu 24.04+ no `packages.yaml` podem usar
`target_distro: ubuntu` e `min_distro_version: "24.04"` para impedir bleed entre
distribuições. O perfil Ubuntu pode instalar `systemd-zram-generator` e gravar
`/etc/sysctl.d/90-envctl-performance.conf`; o perfil CachyOS apenas garante o
pacote `zram-generator` quando ausente. Nenhum dos dois cria swapfile ou altera
journald, scheduler, governor, serviços, kernel cmdline ou mitigations.

```bash
envctl run performance --dry-run
envctl run performance
```

---

## 📄 3. `manifests/shell.yaml`

Define variáveis de ambiente, diretórios restritos e o mapeamento de templates de configuração para o sistema de arquivos do usuário.

```yaml
env_vars:
  - name: NODE_PATH
    value: "%USERPROFILE%\\node_modules"
    target: User
    os: windows
    description: "Resolução global de módulos Node.js para scripts de automação"

  - name: ENVCTL_TEMP
    value: "C:\\temp"
    target: User
    os: windows
    description: "Pasta de scratch padrão dos agentes LLM na raiz do disco"

config_files:
  - source: configs/opencode.json
    destination: ~/.config/opencode/opencode.json
    description: "Configuração central do OpenCode com agentes, plugins e MCPs (padrão único JSON, formato nativo V2)"

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

## 📄 4. `manifests/git.yaml`

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

## 📄 5. `manifests/lsp.yaml`

Registra os 15 servidores de linguagem utilizados por agentes de IA e IDEs (14 aplicáveis no Linux — `pwsh` é windows-only), associando cada um ao seu gerenciador nativo:

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
  - name: marksman
    package_type: winget
    command: marksman server
    languages: [markdown]
```

---

## 📄 6. `manifests/windows.yaml`

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
```

---

## 📄 7. `manifests/debloat.yaml`

Debloat opt-in do Windows 11 absorvido do `windows11-clean` — **só** via `envctl run debloat`
(nunca no `run all`/`run windows`). Reusa o schema de `windows.yaml` com dois tipos extras:

```yaml
tweaks:
  - id: "telemetry-allow-telemetry"   # 12 registry (telemetry) + 12 (privacy) + 12 (gaming)
    path: "HKLM\\SOFTWARE\\Microsoft\\Windows\\CurrentVersion\\Policies\\DataCollection"
    name: "AllowTelemetry"
    type: "DWord"
    value: 0
    category: "telemetry"

  - id: "appx-microsoft-copilot"      # 31 remoções (Xbox/Teams/Outlook excluídos)
    name: "Microsoft.Copilot"
    type: "Appx"                      # conforme = ausente; aplica Remove-AppxPackage -AllUsers
    category: "apps"

  - id: "svc-diagtrack"               # 9 serviços safe-only (Spooler/WSearch/NgcSvc fora)
    name: "DiagTrack"
    type: "Service"                   # conforme = StartType; value Disabled/Manual
    value: "Disabled"
    category: "services"
```

O `doctor` audita uma linha agregada por categoria (`Debloat / category <nome>`):
`OK` quando aplicada, `INFO` com `run 'envctl run debloat'` quando há drift — nunca
`WARN`/`ERROR` e nunca no `--fix`. Checks usam `CheckBatch` (1 spawn PowerShell por
família: registro, Appx, serviços) em vez de 1 por tweak. O Tier 3 destrutivo (OneDrive, hibernação, power
plan, Teredo, `.wslconfig`, Copilot/Recall) vive na skill `windows-debloat`.
