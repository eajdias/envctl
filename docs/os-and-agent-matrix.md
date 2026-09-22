# Matriz OS × Provedor (o que é provisionado onde)

Documento de referência para **validação** (bater o que está em disco contra o manifesto),
**organização** (onde cada coisa mora) e **adições novas** (o que precisa ser tocado em cada
camada). Todos os números vêm dos manifestos e do código — se divergirem, um dos dois mudou.

---

## 0. Escopo dos ambientes

| Ambiente | Papel | Particularidades |
| :--- | :--- | :--- |
| **Windows 11** (workstations/notebooks) | Trabalho — apps de cliente exigem o OS | WSL é usado para containers Docker; PowerShell 7 como shell |
| **Arch / CachyOS** (desktops/notebooks) | Pessoal | fish como shell; jogos (stack `gaming`); Cursor como IDE |
| **Ubuntu Server** (VPS/VM) | Administração, sem GUI | Sem Cursor/IDE (servidor, sem GUI) — binários LSP seguem instalados (shell/IDE), mas o bloco `lsp` do `opencode.json` foi removido: o runtime do opencode v2 ignora LSP; alvo do `vps-agent-dispatch` |
| **Termux (Android)** | Celulares | Ainda **não padronizado** no repo; planejado controle via SSH |

---

## 1. Por Sistema Operacional

| Dimensão | Windows 11 | Ubuntu/Debian | Arch/CachyOS |
| :--- | :--- | :--- | :--- |
| Gerenciadores | winget · volta · pip | apt · volta | **pacman · paru (AUR)** · volta |
| Pacotes declarados (aplicáveis) | **55** (30 winget · 15 volta · 10 pip) | **45** (27 apt · 15 volta · 3 uv-pip) | **48** (29 pacman · 15 volta · 3 uv-pip · 1 paru) |
| **Fase 0: provedores (`run providers`)** | winget (`SST.opencode`) + volta (`command-code`), atualizados quando o canal permite | instalador oficial do opencode + volta | pacman (`opencode`, `paru`) + volta (`command-code`) |
| Bootstrap de toolchain (`run bootstrap`) | não usa (winget/volta cobrem) | 18 passos: Volta+Node+pnpm, bun, Playwright, opencode CLI, cmdc CLI, gh, delta, yq, uv, ruff, pylsp, stylelint, golangci-lint, fd, **paru**, Go, PATH | idem, com **fd via pacman** e **paru via repo do CachyOS** (Arch puro: AUR) |
| Shell alvo da persistência | PowerShell 7 (perfil) + WSL | `.profile` + `.bashrc` | `.profile` + `.bashrc` + **fish (`set -gx`)** |
| Variáveis de ambiente | 2 | 2 | 2 |
| Configs aplicáveis | **27** (contados em `shell.yaml` via `MatchesOS` por distro) | **25** | **25** |
| Diretórios | 15 (12 + 3 só-Windows) | 13 (12 + 1 só-Linux) | 13 |
| Git global | 6 (4 + 2 win-only) | 4 | 4 |
| LSPs instaláveis (binários p/ shell/IDE; bloco `lsp` removido do `opencode.json` — runtime v2 ignora LSP) | **15** (14 + `pwsh`) | **14** | **14** |
| Skills por agente | **38** (37 + 1 só-Windows) | **38** (37 + `headless-gui-probe`) | **40** (37 + `aur-headless-install` + `cachyos-gaming-setup` + `headless-gui-probe`) |
| Editor/IDE | **Cursor** (`Anysphere.Cursor` via winget) | — (servidor, sem GUI) | **Cursor** (`cursor-bin` via paru; CachyOS já traz o Chaotic-AUR) |
| Tweaks de registro / módulos | **8** (6 DWord: `long-paths`, `developer-mode`, `explorer-show-ext`, `explorer-show-hidden`, `dark-mode-apps`, `dark-mode-system`; 2 `PSModule`: `PSScriptAnalyzer`, `Pester`) | — | — |
| Gaming (`run gaming`) | — | — | pacman + paru (Steam, Proton CachyOS, gamescope, MangoHud, emuladores, lact, scx, ananicy, X11 trio) + presets seed + doctor Gaming |
| Temp padrão (ENVCTL_TEMP) | `C:\temp` | `/temp` | `/temp` |
| Quality gates (`envctl-verify` + pre-push) | ✓ | ✓ | ✓ |

**Escopo por subsistema:** `run winget`/`run windows` são Windows-only; `run apt` é
Debian/Ubuntu; `run pacman`/`run paru`/`run gaming` são Arch; `run bootstrap` é Linux
(Windows usa winget/volta). `run providers` é portável e roda **antes de tudo** dentro de
`run all` (fase 0). `run all` despacha por OS e pula o que não é da plataforma.

### Fase 0 — `run providers`

Garante que os CLIs dos agentes existam e estejam atualizados **antes** de qualquer
provisionamento, para uma máquina nova chegar aos agentes sem passo manual (e para o
`doctor` de uma máquina recém-instalada não começar com pendências).

| Etapa | O que faz | Detalhe |
| :--- | :--- | :--- |
| Volta | instala se faltar | Linux: instalador oficial · Windows: `winget` `Volta.Volta`. Sem `volta self-update`: atualizar = rodar o instalador |
| Runtime Node | garante um default | Usa o **mesmo spec do manifesto** (`volta install node@…`), para os dois não divergirem |
| `command-code` (`cmdc`) | instala/atualiza via Volta | Compara a versão instalada com o `latest` do npm; Volta resolve o pacote, então "faltando" e "desatualizado" são o mesmo comando |
| `opencode` | instala se faltar; **nunca** por npm | Arch: pacote `extra` · Windows: `SST.opencode` (winget) · demais: instalador oficial |

**Regra que a fase 0 respeita:** o envctl só substitui binário que é dele
(`~/.local/bin`) ou do Volta. Binário de pacote do SO é **reportado**, nunca
sombreado — uma cópia em `~/.local/bin` venceria no PATH e congelaria a versão ali
instalada (mesma lição do `fzf`).

**Famílias de distro:** o campo `os:` aceita, além de `windows`/`linux`/`darwin`, as famílias
`arch`/`cachyos` e `debian`/`ubuntu` (via `entity.MatchOS`). Exemplo real: `cursor-bin` é
`os: arch` com `type: paru`, então só é tocado em Arch — em Ubuntu o gerenciador paru nem é
consultado.

---

## 2. Por Provedor de Agente

| Dimensão | OpenCode | CommandCode |
| :--- | :--- | :--- |
| Diretório | `~/.config/opencode` | `~/.commandcode` |
| Arquivos declarados | **11** | **7** |
| Config principal | `opencode.json` (variante win/linux) | `settings.json` (permissões + hooks) |
| Regras globais | `AGENTS.md` (win/linux) | `AGENTS.md` (win/linux) |
| Índice de consulta | `SKILL-INDEX.md` + `REFERENCE.md` | `SKILL-INDEX.md` |
| MCP | seção `mcp` no `opencode.json` (context7; ssh-manager + chrome-devtools disabled) | `mcp.json` (context7; chrome-devtools + ssh-manager disabled) |
| LSP | sem bloco `lsp` (removido 2026-09-22 — inerte no runtime v2; binários seguem provisionados p/ shell/IDE e `doctor` checa presença+handshake como toolchain) | **nenhuma** — `get_diagnostics` é IDE-only |
| Plugins | 1 (goal-plugin only; `dcp.jsonc` removido do provisioning em 2026-09-22 — YAGNI) | — |
| Contexto / pruning | nativo (`compaction` do v2; DCP removido) | — |
| Memória | seeds `lessons.md` + `patterns.md`, dir `memory` | — (memória vive no `AGENTS.md`; dir `memory` é limpo) |
| Agentes custom | `review`, `plan` (no JSON) | `agents/code-reviewer.md` |
| Permissões | no `opencode.json` | `settings.json`: 12 allow · 6 ask · 4 deny |
| Hooks | — | `Stop` → `envctl-verify --hook` |
| Skills (destino) | `~/.config/opencode/skills` | `~/.commandcode/skills` |
| Validação de skill no doctor | **frontmatter + contagem vs manifesto** | **frontmatter + contagem vs manifesto** |
| Diretórios criados | 4 (skills, memory, secrets 0700, extras) | 2 (skills, agents) |
| Cleanup dedicado | 4 entradas | 6 entradas |
| IDE integration | — (diagnósticos via lint/typecheck no v2; sem runtime LSP) | VS Code / Cursor / Windsurf via `/ide` |

### Checks do `doctor` por provedor

| Provedor | Checks |
| :--- | :--- |
| OpenCode | `AGENTS.md (global rules)` · `Config file references` (refs `{file:...}` do `opencode.json`) · `Database` (tamanho + páginas livres do `opencode.db`) · `Tool Output` (diretório) · `Skills` (frontmatter + contagem) |
| CommandCode | `CommandCode CLI` · `~/.commandcode/` · `Settings` (JSON válido) · `MCP config` (JSON válido) · `Agents` (frontmatter) · `Skills` (frontmatter + contagem) |
| Ambos | `LSP` (binário no PATH) · `Verify` (verificador + pre-push) · `Git` · `TempFolder` |

---

## 3. Assimetrias Conhecidas

| # | Assimetria | Estado |
| :-- | :--- | :--- |
| 1 | `csharp` era declarado no `configs/opencode.linux.json` sem o `csharp-ls` existir no Linux | **Resolvido** — .NET/C# fora da stack: pacote, LSP e entradas nos configs removidos |
| 2 | `stale_pw_ps1` duplicado no `cleanup` | **Resolvido** — id único |
| 3 | `docker` não existia no bloco pacman (só apt `docker.io`) | **Resolvido** — `docker` + `docker-compose` + `docker-buildx` via pacman |
| 4 | `golangci-lint` não era provisionado, mas é exigido pelo `envctl-verify` e pelo CI | **Resolvido** — bootstrap Linux (install.sh) + winget `GolangCI.golangci-lint` |
| 5 | `python-pipx` existia no Arch e não no Ubuntu | **Resolvido** — removido (padrão único: `uv tool`) |
| 6 | `FZF_DEFAULT_COMMAND` só existia no Windows, embora `fzf` seja instalado nos três | **Resolvido** — a variável saiu do manifesto: fzf ≥ 0.47 traz walker nativo (`file,follow,hidden`, skip `.git,node_modules`), e o bootstrap instala a release atual quando a distro traz uma versão antiga (Ubuntu 24.04 traz 0.44.1, sem walker) |
| 7 | Rigor de auditoria de skills difere: CommandCode validava frontmatter e contagem, OpenCode só a existência do diretório | **Resolvido** — validador compartilhado (`auditSkillTree`) para os dois agentes: presença, frontmatter e contagem vs manifesto, agregados em uma linha por agente (110 checks após a agregação). Uma skill com frontmatter inválido era **ignorada em runtime** com o doctor verde |
| 8 | `~/.bash_profile`, `.bashrc` e `.profile` referenciavam o shim `~/.local/bin/env` do uv, que nada recriava (o uv só o escreve quando `~/.local/bin` não está no PATH) — todo login shell imprimia erro no stderr | **Resolvido** — `run shell` remove a referência morta; o PATH de `~/.local/bin` é garantido pelo próprio envctl |
| 9 | O `opencode` tem **dois canais de versão** e o npm não é o mais novo: o pacote npm `opencode-ai` (`latest`) ficou em **1.18.31** (14/09), enquanto as tags do upstream (`anomalyco/opencode`, ex-`sst/opencode`) já estão em **v2.0.7** (17/09) — e é essa linha 2.x que o pacote do Arch (`extra`, 2.0.5) empacota. Instalar por npm/Volta **rebaixaria** a máquina | **Resolvido** — `opencode` é provisionado por pacote do SO (`pacman`/`winget`) ou pelo instalador oficial, nunca por npm; o bootstrap deixou de tentar npm e a fase 0 só reporta binário que não é dela (ver §1) |
| 10 | `paru` não vinha por padrão no CachyOS e o manifesto Arch não o declarava, embora `run paru` e o `type: paru` (ex.: `cursor-bin`) dependam dele | **Resolvido** — `paru` declarado no bloco pacman (repo `[cachyos]`); no Arch puro, que não tem paru em repo nenhum, o bootstrap constrói do AUR (`base-devel` + `git` também declarados) |
| 11 | `ssh-manager` MCP existia no `configs/opencode.json` (Windows) mas nunca no `configs/opencode.linux.json` — no CachyOS o agente só alcançava VPS via CLI | **Resolvido** — entrada espelhada no linux (`command: [mcp-ssh-manager]`, `timeout: 30000`, `enabled: false`); timeout pina os 30s do `chrome-devtools` (default 5s estoura no handshake) |
| 12 | `zscan` (`@eajdias/zscan-run`) vinha desde o squash inicial em `configs/opencode.json` + `configs/commandcode/mcp.json`, sem dono nem uso | **Resolvido** — blocos deletados nos dois configs; `doctor` acusa `Removed MCP entries` (warning nomeando o servidor) quando o deployado ainda declara; CHANGELOG/lessons e o prefixo `zscan-` do `temp_hygiene` (limpeza, não instala nada) ficam |
| 13 | `configs/AGENTS.linux.md` dizia "Ubuntu Server, usuário `ubuntu`" até no desktop CachyOS (`os: linux` cobria tudo) | **Resolvido** — split `configs/AGENTS.arch.md` (`os: arch,cachyos`: fish, paru, gaming, Cursor) vs linux (`os: debian,ubuntu`); `doctor` avisa `AGENTS.md (identity coverage)` quando nenhuma variante casa com o host (ex.: distro desconhecida) |
| 14 | Chave `yaml` no `lsp` do opencode duplicava o builtin `yaml-ls` (ids provados no binário: merge `item.extensions ?? existing?.extensions` dispara os dois em `.yaml/.yml`); `pylsp` + `pyright` disparavam duplo em `.py` | **Resolvido** — `yaml` renomeado para `yaml-ls` nos dois configs (+ `id: yaml-ls` no `lsp.yaml`); `pylsp` removido dos dois configs (pesquisa 2026: consenso é servidor de tipos `pyright`/`ty` + `ruff` p/ lint — `pylsp` legado, mais lento; binário segue provisionado p/ IDE/shell). Requisitos de spawn (typescript/pyright/eslint exigem dep no projeto, gopls exige `go`) documentados como causa esperada de "não ativa" |
| 15 | `taplo` tem dois canais (`taplo-cli` via pacman + `@taplo/cli` via npm) e o `run lsp` prefere o npm mesmo no Arch | **Exceção intencional** — mesmo dono nos dois canais, sem sombreamento entre gerenciadores. Revisitar só com skew de versão observado |
| 16 | opencode v2 ignora runtime LSP (`lsp` aceito mas inerte), `subagent_depth` top-level (WARN `omitted unsupported legacy setting`) e `instructions` (aceito, não carregado) | **Resolvido 2026-09-22** — bloco `lsp`, `subagent_depth` e `instructions` removidos dos dois configs (formato nativo V2: `agents`/`permissions[]`/`plugins`/`skills[]`/`mcp.servers`); `review` com `mode: primary` explícito, `plan` sem `mode` (preserva o built-in — customs só com IDs novos); binários LSP seguem provisionados p/ shell/IDE e o `doctor` os audita como toolchain |
| 17 | Retorno do runtime LSP no opencode v2 (hoje: config validada, nenhum servidor inicia, zero diagnósticos — [docs](https://dev.opencode.ai/v2/docs/lsp/)) passa batido sem monitoramento | **Monitoramento mensal** — conferir `pacman -Si opencode` (versão no `extra`) + changelog upstream; quando o runtime voltar: re-testar bloco `lsp` per-project (template em `patterns.md`), re-adicionar via `lsp.yaml` → JSON, sem plugin V2 antes da API sair de beta |

---

## 4. Cobertura por Stack

Levantamento do que o `envctl` provisiona hoje contra as stacks de uso real.
`—` = não provisionado.

| Stack | Provisionado | Lacuna |
| :--- | :--- | :--- |
| PowerShell / CMD | PowerShell 7 (winget) · **PSScriptAnalyzer + Pester** (módulos, `type: PSModule` no `windows.yaml`) | — |
| Bash | bash, `shellcheck`, **`shfmt`** | — |
| Fish | `fish` (pacman) | — (`fish_indent` vem com o fish) |
| Python | `uv`, `ruff`, `pylsp`, **`pytest`**, **`mypy`**, pip (fallback) | `ty` (Astral) como alternativa ao mypy |
| Go | `go`, `gopls`, `gofmt`/`go vet`, `golangci-lint` | — |
| TS/JS (Nest/Next) | `node`, `pnpm`, `typescript` (**`tsc`**), `prettier`, `typescript-language-server`, `eslint-ls` | — (eslint global não é provisionado de propósito: plugins resolvem do `node_modules` do projeto) |
| CSS/HTML | `vscode-css/html-language-server`, `stylelint` | — |
| SQL | `sqllens-language-server` (LSP), **`sqlfluff`** | — |
| PostgreSQL (pgvector) | — | **cancelado** — virá por skill específica |
| MySQL | — | **cancelado** — virá por skill específica |
| Redis | — | **cancelado** — virá por skill específica |
| SQLite | `sqllens-language-server` (LSP) | **cancelado** — virá por skill específica |
| Docker | Windows: Docker Desktop · Ubuntu: `docker.io` · Arch: docker trio · **`hadolint`** (bootstrap/release + winget) | — |
| Cursor IDE | Windows (winget) · Arch (paru) | — (é o editor padronizado; habilita `/ide` + `get_diagnostics`) |
| RAG / automações | libs de agente (`requests`, `bs4`, `pypdf`, `openpyxl`, `lxml`, `docx`, `yaml`) | libs de RAG pertencem ao venv do projeto (`uv`) |
| N8N | — | npm-based: pertence ao projeto (`bunx`/`npx`) |
| Gaming / debloat | `run gaming` (33 pkgs: Steam, Proton CachyOS, gamescope, MangoHud + GOverlay, emuladores, `lact`, scx, ananicy, X11 trio) · presets `gaming.conf`/`MangoHud.conf` (seed) · `doctor` seção Gaming (opt-in via Steam: pacotes + 4 serviços + `sched_ext` + cmdline + RADV + multilib) · skill `cachyos-gaming-setup` (tuning root/reboot) · 8 tweaks Windows (6 DWord + 2 módulos PowerShell) | telemetria/Game Bar no Windows |

**Fora da stack (removidos):** `.NET SDK 8`, `csharp-ls` (+ LSP `csharp`), Visual Studio Code
(+ `vscode_settings`), Termius, WinSCP, GitHub Desktop, Rust/Oh-My-Posh (remoção anterior).

---

## 5. Checklist para Adições Novas

**Pacote de sistema** → `manifests/packages.yaml`
1. Um item por gerenciador (`winget`/`apt`/`pacman`/`paru`).
2. `os:` explícito por distro, NUNCA bare `linux` — `windows` · `arch,cachyos` (desktop CachyOS) · `debian,ubuntu` (VPS Ubuntu Server) · conteúdo POSIX idêntico nas duas distros usa a lista `arch,cachyos,debian,ubuntu` · portável de verdade omite `os:`. Conjunto fechado: nova distro só com pedido explícito do dono (o lint `manifest_os_lint_test.go` falha em `os: linux` e em tokens desconhecidos).
3. `check_command` **sem aspas** — o parse é por `strings.Fields` (aspas quebram o comando).
4. Id igual ao nome real do pacote em cada gerenciador (nada de sufixos como `-pacman`).

**Arquivo de config** → `manifests/shell.yaml`
1. Escolha a categoria (`opencode`, `commandcode`) para que `envctl opencode`/`commandcode`
   não toquem no outro agente.
2. Variante por OS quando o conteúdo difere (`.linux.` no nome do arquivo em `configs/`).
3. Se o usuário edita o arquivo, declare `merge:` (`ssh_hosts`, `json_deps`) — sobrescrever
   só é seguro para arquivos 100% gerenciados.
4. `seed_if_missing: true` para baselines que não devem ser sobrescritas.
5. `executable: true` para scripts (o provisioning aplica 0755).

**Skill** → `configs/skills/<nome>/` + `manifests/skills.yaml`
1. `enabled: true` e `os:` só quando for específica de plataforma.
2. O diretório embutido e o manifesto precisam bater (o teste
   `TestLoadManifestsFromDiskOrEmbed` falha se divergirem).

**Agente custom** → `agents` no `configs/opencode*.json`
1. SEMPRE ID novo — nunca sobrescrever built-ins (`build`/`plan`/`general`/`explore`): única exceção documentada é o `plan` do envctl, reduzido a 1 regra (`edit spec-agent/** allow`) que estende o built-in por merge (efetivo `primary`).
2. `mode: primary` explícito no custom novo, `system` (nunca `prompt`), `permissions[]` nativas (`shell`/`subagent`, nunca `bash`/`task`).

**LSP** → `manifests/lsp.yaml` (binários p/ shell/IDE — `run lsp` + `doctor`)
1. Informe `install_type` (`volta`, `npm`, `pip`, `go`), `install_target` e `check_binary`.
2. NÃO espelhe entrada no `opencode.json`: o runtime v2 ignora o bloco `lsp` (assimetria #16) — foi removido dos dois configs em 2026-09-22.

**CLI de agente (provedor)** → `manifests/packages.yaml` + `run providers` (fase 0)
1. Confirme o **canal de versão** antes de escolher o gerenciador: o mesmo produto costuma ter
   linhas diferentes por canal (pacote do SO ≠ npm ≠ instalador oficial). Foi assim que o
   `opencode` quase foi rebaixado (assimetria #9).
2. Se o CLI já existe na máquina mas não é do envctl nem do Volta, **não instale** uma segunda
   cópia — reporte. Cópia em `~/.local/bin` vence no PATH e congela a versão ali instalada.
3. Um CLI que o próprio envctl garante entra em `providerCLIs()`
   (`internal/usecase/provision_providers.go`): com `voltaPkg` ele é atualizável; com
   `wingetID`/`installer` só é instalado quando falta.
4. Declare o `check_command` no manifesto para o `doctor` auditar a presença naquela plataforma.

**Sempre**: `gofmt`/`go build`/`go vet`/`go test` + `golangci-lint run --new-from-rev=origin/main`
antes do push (o `envctl-verify` já cobre os sete automaticamente, no hook e no pre-push).
