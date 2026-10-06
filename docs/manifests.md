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
├── skills.yaml      # Catálogo das 12 skills de agentes de IA (com escopo por ambiente)
└── windows.yaml     # Tweaks de registro, Developer Mode e fontes do Windows 11
```

## 🗺️ Seção → arquivo → comando

| Seção | Arquivo (fonte) | Comando |
| :--- | :--- | :--- |
| pacotes | `manifests/packages.yaml` | `envctl run packages` (`run apt`/`run winget`/`run pacman` filtram por gerenciador) |
| performance | `manifests/performance_ubuntu.yaml`, `manifests/performance_cachyos.yaml` | `envctl run performance` (perfis `run vps` / `run cachyos`) |
| shell/env/configs | `manifests/shell.yaml` | `envctl run shell` |
| git | `manifests/git.yaml` | `envctl run shell` (configs globais) |
| LSP | `manifests/lsp.yaml` | `envctl run lsp` |
| skills | `manifests/skills.yaml` | `envctl run skills` |
| windows tweaks | `manifests/windows.yaml` | `envctl run tweaks` (perfil `run windows`) |
| debloat | `manifests/debloat.yaml`, `manifests/debloat_linux.yaml` | `envctl run debloat` |
| extras | `manifests/extras.yaml` | `envctl run extras` |

---

## 📄 1. `manifests/packages.yaml`

Define todos os pacotes gerenciados. Fonte: `manifests/packages.yaml`
(campos reais: `id`, `name`, `type`, `os`, `category`, `check_command` —
ver o arquivo; nenhum exemplo é recopilado aqui).

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

Perfis de performance são aplicados pelo perfil `envctl run vps`
(Ubuntu Server 24+) e `envctl run cachyos` (CachyOS), ou pelo comando
standalone `envctl run performance`; eles não fazem parte de `doctor --fix`.

A identidade do perfil **não carrega versão**: é `ubuntu-server`, e o piso de
release vive no manifesto como `min_distro_version`. A frota já roda Ubuntu
26.04, então um nome que codificasse "24.04" seria mentira. Seletor:

- Ubuntu Server com `VERSION_ID >= min_distro_version` usa `performance_ubuntu.yaml`;
- CachyOS usa `performance_cachyos.yaml`;
- Debian, Ubuntu antigo e Arch genérico são rejeitados — `run vps` **falha**
  com o motivo, em vez de seguir e pular o tuning em silêncio.

Entradas Ubuntu 24+ no `packages.yaml` podem usar `target_distro: ubuntu` e
`min_distro_version: "24.04"` para impedir bleed entre distribuições.

### Seções do perfil `ubuntu-server`

| Seção | O que faz | Onde escreve |
| --- | --- | --- |
| `packages` | `systemd-zram-generator`, `tzdata` | gerenciador de pacotes |
| `sysctls` | rede, com `policy: min` onde o valor é um piso | `/etc/sysctl.d/90-envctl-performance.conf` |
| `tiers` | bandas de RAM detectadas; `vfs_cache_pressure` por banda | resolvido em runtime |
| `swap` | adota um swapfile existente ou cria um clampado | `/swapfile.envctl` + `/etc/fstab` |
| `zram` | liga ou desliga o zram conforme a banda | `dev-zram0.swap` |
| `journald` | teto com piso de espaço livre | `/etc/systemd/journald.conf.d/90-envctl-journald.conf` |
| `limits` | soft de descritores, preservando o hard do host | `system.conf.d/` e `security/limits.d/` |
| `timezone` | verifica por padrão; aplicar é opt-in | `timedatectl` |

`vm.swappiness` **não** é declarado: é derivado da topologia de swap medida
(150 com zram ativo, 10 só em disco). `fs.file-max` usa `policy: min`, então o
nunca rebaixa um teto que o host já tem melhor.

O perfil **não** toca governor de CPU, scheduler de I/O, mitigations, kernel
cmdline, `crashkernel` nem serviços sem relação — todos continuam atrás de
benchmark e aprovação explícita.

A remoção de pacotes é a única etapa destrutiva e é **opt-in**
(`--allow-debloat`), com a lista em `manifests/debloat_linux.yaml` e a guarda
`NEEDRESTART_MODE=l` escrita antes do primeiro purge. Detalhes e rollback em
[`docs/guides/ubuntu-server-baseline.md`](guides/ubuntu-server-baseline.md).

```bash
envctl run performance --dry-run
envctl run performance
envctl run performance --allow-debloat   # inclui a remoção de pacotes
```

---

## 📄 3. `manifests/shell.yaml`

Define variáveis de ambiente, arquivos de configuração e diretórios. Fonte:
`manifests/shell.yaml` (seções `environment_variables`, `config_files`,
`directories`, `cleanup` — ver o arquivo; nenhum exemplo é recopilado aqui).

---

## 📄 4. `manifests/git.yaml`

Define configurações globais do Git. Fonte: `manifests/git.yaml` (lista
`configs` com `key`, `value`, `os` — ver o arquivo; nenhum exemplo é
recopilado aqui).

---

## 📄 5. `manifests/lsp.yaml`

Registra os servidores de linguagem utilizados por agentes de IA e IDEs.
Fonte: `manifests/lsp.yaml` (campos reais: `id`, `language`, `server_name`,
`command`, `args`, `install_type`, `install_target`, `check_binary` — ver o
arquivo; nenhum exemplo é recopilado aqui).

---

## 📄 6. `manifests/windows.yaml`

Define ajustes de registro do Windows 11 para desenvolvedores. Fonte:
`manifests/windows.yaml` (campos reais: `id`, `description`, `path`,
`name`, `value`, `type`, `category` — ver o arquivo; nenhum exemplo é
recopilado aqui).

---

## 📄 7. `manifests/debloat.yaml`

Debloat do Windows 11 absorvido do `windows11-clean` — aplicado pelo perfil
`envctl run windows` ou standalone via `envctl run debloat`. São **99 tweaks**
(38 registro + 34 Appx + 9 serviços `Disabled` + 11 serviços `Manual` + 4
startup entries + 3 `Command` + 1 `Onedrive` + suporte a `Binary`) em 8
categorias (`telemetry`, `privacy`, `gaming`, `apps`, `services`, `startup`,
`power`, `onedrive`). Reusa o schema de `windows.yaml` com tipos extras:

```yaml
tweaks:
  - id: "telemetry-allow-telemetry"   # 12 registry (telemetry) + 12 (privacy) + 12 (gaming)
    path: "HKLM\\SOFTWARE\\Microsoft\\Windows\\CurrentVersion\\Policies\\DataCollection"
    name: "AllowTelemetry"
    type: "DWord"
    value: 0
    category: "telemetry"

  - id: "appx-microsoft-copilot"      # 34 remoções (só o Xbox suite fica de fora)
    name: "Microsoft.Copilot"
    type: "Appx"                      # conforme = ausente; aplica Remove-AppxPackage -AllUsers
    category: "apps"

  - id: "svc-diagtrack"               # 9 serviços -> Disabled
    name: "DiagTrack"
    type: "Service"                   # conforme = StartType; value Disabled/Manual
    value: "Disabled"
    category: "services"

  - id: "svc-wsearch-manual"          # 11 serviços -> Manual (WSearch, SysMain, NgcSvc,
    name: "WSearch"                   #   wbengine, OneSyncSvc, Dell*, fb*)
    type: "Service"
    value: "Manual"
    category: "services"

  - id: "startup-microsoftedge"       # 4 startup entries
    name: "MicrosoftEdge"             # conforme = ausente; sem path e sem value
    type: "StartupItem"               # remove o valor/atalho do Run key / pasta Startup
    category: "startup"

  - id: "tier3-teredo"                # 3 Command tweaks (Teredo, PowerPlan, Hibernation)
    name: "Teredo"                    # pares check/apply idempotentes no conjunto fechado
    type: "Command"                   # de tier3_scripts.go (sem path/value no manifest)
    category: "gaming"

  - id: "tier3-user-preferences-mask" # Binary registry: value = lista de bytes
    path: "HKCU:\\Control Panel\\Desktop"
    name: "UserPreferencesMask"
    value: [144, 18, 3, 128, 16, 0, 0, 0]
    type: "Binary"
    category: "gaming"

  - id: "tier3-onedrive"              # scripts fixos; a pasta do usuário nunca é tocada
    name: "OneDrive"
    type: "Onedrive"
    category: "onedrive"
```

`StartupItem` sonda e remove contra um conjunto **fechado**: as duas Run keys
(`HKCU`/`HKLM ...\CurrentVersion\Run`) e as duas pastas `Startup`
(`ApplicationData` e `CommonApplicationData`). Não passa por
`Win32_StartupCommand` — essa classe é uma `CIM_Setting` cujo MOF publicado lista
só properties (não existe `Delete`) e o `Location` dela é inconsistente entre
formatos. Ler os locais diretamente torna a garantia estrutural: um serviço não
é um valor em Run key nem um arquivo em pasta `Startup`, então não há o que
classificar errado.

A sonda reporta um **token** por alvo (`RUN_HKCU`, `DIR_PROGRAMDATA`, ...), nunca
um path: nenhum caminho atravessa a fronteira do processo (um `%APPDATA%` não
ASCII é corrompido pelo code page do console) e nenhum separador consegue
truncar. `startupTargetForToken()` recusa token desconhecido, então um alvo fora
do conjunto não chega a uma remoção. As pastas resolvem por
`[Environment]::GetFolderPath()`, não por `%APPDATA%`/`%ProgramData%`, que
sumem em contexto não interativo (serviço, tarefa agendada). O nome da Run key
é escapado com `[WildcardPattern]::Escape()` porque `-Name` do provider de
registro **sempre** é wildcard — sem isso, um nome com `*` apagaria vários
valores. Check, batch e apply passam todos por `probeStartup`, e uma sonda
parcial vira erro em vez de "ausente" (senão o doctor certificaria convergência
inexistente).

O `doctor` audita uma linha agregada por categoria (`Debloat / category <nome>`):
`OK` quando aplicada, `INFO` com `run 'envctl run debloat'` quando há drift — nunca
`WARN`/`ERROR` e nunca no `--fix`. Checks usam `CheckBatch` (1 spawn PowerShell por
família: registro, Appx, serviços, startup) em vez de 1 por tweak. O Tier 3 destrutivo
(OneDrive, hibernação, power plan, Teredo, `.wslconfig`, Copilot/Recall) é manual e
vive em `docs/guides/windows-debloat-tier3.md`.

---

## 📄 8. `manifests/extras.yaml`

Aplicativos **opcionais** preferidos do dono, fora do provisionamento padrão:
o repo é público e neutro — nada de extras instala sem opt-in explícito.
São **15 pacotes winget** (Windows) e **12 pacotes pacman** (Arch/CachyOS) de
apps comuns (Brave, Obsidian, Steam, Tailscale, VLC, ONLYOFFICE, Syncthing,
Moonlight, WinSCP, Wireshark, Nmap, Termius, TreeSize, BCUninstaller,
LockHunter; brave-origin-bin, obsidian, onlyoffice-bin, vlc, transmission-qt,
picard, rustdesk-bin, anydesk-bin, tailscale, boosteroid, alacritty, mpv).

**Zero PII**: só IDs de pacote — apps de negócio/contas ficam fora. `RustDesk` e
`Npcap` (Windows) não têm pacote winget (installer próprio) — instalação manual,
documentada no manifesto.

```bash
envctl run extras                 # só os opcionais
envctl run all --with-extras      # perfil completo + opcionais
```

O `doctor` **não** audita extras: são preferência, não stack — a presença é
verificada pelo próprio `run extras` (idempotente).
