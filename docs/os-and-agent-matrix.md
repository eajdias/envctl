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
| **Ubuntu Server** (VPS/VM) | Administração, sem GUI | Sem Cursor/IDE (servidor, sem GUI) — binários LSP seguem instalados (shell/IDE), mas o bloco `lsp` do `opencode.json` foi removido: o runtime do opencode v2 ignora LSP; alvo de trabalho remoto por SSH |
| **Termux (Android)** | Celulares | Ainda **não padronizado** no repo; planejado controle via SSH |

---

## 1. Por Sistema Operacional

| Dimensão | Windows 11 | Ubuntu/Debian | Arch/CachyOS |
| :--- | :--- | :--- | :--- |
| Gerenciadores | winget · volta · pip | apt · volta | **pacman · paru (AUR)** · volta |
| Pacotes declarados (aplicáveis) | **55** (29 winget · 15 volta · 11 pip) | **63** em Ubuntu 24.04+ (44 apt · 15 volta · 4 uv-pip; **46** em Ubuntu/Debian antigos) | **70** (50 pacman · 15 volta · 4 uv-pip · 1 paru) |
| **Fase 0: provedores (`run providers`)** | instalador oficial V2 PowerShell (`~/.local/bin`) + volta (`command-code`), atualizados quando o canal permite | instalador oficial V2 (`~/.opencode/bin`) + volta | pacman (`opencode`, `paru`) + volta (`command-code`) |
| Bootstrap de toolchain (`run bootstrap`) | não usa (winget/volta cobrem) | 18 passos: Volta+Node+pnpm, bun, Playwright, opencode CLI, cmdc CLI, gh, delta, yq, uv, ruff, pylsp, stylelint, golangci-lint, fd, **paru**, Go, PATH | idem, mas OpenCode usa o mesmo `pacman` injetado; **fd via pacman** e **paru via repo do CachyOS** (Arch puro: AUR) |
| Shell alvo da persistência | PowerShell 7 (perfil) + WSL | `.profile` + `.bashrc` | `.profile` + `.bashrc` + **fish (`set -gx`)** |
| Variáveis de ambiente | 2 | 2 | 2 |
| Configs aplicáveis | **27** (contados em `shell.yaml` via `MatchesOS` por distro) | **25** | **25** |
| Diretórios | 15 (12 + 3 só-Windows) | 13 (12 + 1 só-Linux) | 13 |
| Git global | 6 (4 + 2 win-only) | 4 | 4 |
| LSPs instaláveis (binários p/ shell/IDE; bloco `lsp` removido do `opencode.json` — runtime v2 ignora LSP) | **15** (14 + `pwsh`) | **14** | **14** |
| Skills por agente | **12** (portáteis) | **12** (portáteis) | **12** (portáteis) |
| Editor/IDE | **Cursor** (`Anysphere.Cursor` via winget) | — (servidor, sem GUI) | **Cursor** (`cursor-bin` via paru; CachyOS já traz o Chaotic-AUR) |
| Tweaks de registro / módulos | **8** (6 DWord: `long-paths`, `developer-mode`, `explorer-show-ext`, `explorer-show-hidden`, `dark-mode-apps`, `dark-mode-system`; 2 `PSModule`: `PSScriptAnalyzer`, `Pester`) + debloat opt-in **`run debloat`** (76 em `debloat.yaml`: 12 telemetria + 12 privacidade + 12 gaming-win + 31 Appx + 9 serviços; Tier 3 manual na skill `windows-debloat`) | — | — |
| Gaming (`run gaming`) | — | — | pacman + paru (Steam, Proton CachyOS, gamescope, MangoHud, emuladores, lact, scx, ananicy, X11 trio) + presets seed + doctor Gaming |
| Temp padrão (ENVCTL_TEMP) | `C:\temp` | `/temp` | `/temp` |
| Quality gates (`envctl-verify` + pre-push) | ✓ (advisory lint + blocking tests) | ✓ (advisory lint + blocking tests) | ✓ (advisory lint + blocking tests) |

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
| `opencode` | instala/atualiza para **v2** se faltar ou se encontrar um v1 user-local; **nunca** por npm | Arch: pacote `extra` (binário do sistema não é sombreado) · Windows: instalador oficial V2 PowerShell (zip → `~/.local/bin`) · demais: instalador oficial V2 (`~/.opencode/bin`) |

**Regra que a fase 0 respeita:** o envctl substitui binários que são dele
(`~/.local/bin`, `~/.opencode/bin`) ou do Volta. Binários de pacote do SO são
consultados pelo banco do gerenciador e permanecem autoritativos: no Arch, uma
cópia envctl user-local é arquivada antes de usar `pacman`; no Ubuntu/Debian, um
v1 legado em `/usr/bin` é substituído pelo v2 user-local, porque a configuração
V2 não funciona com v1. Uma cópia local nunca deve vencer o pacote do Arch.

**Famílias de distro:** o campo `os:` aceita, além de `windows`/`linux`, as famílias
`arch`/`cachyos` e `debian`/`ubuntu` (via `entity.MatchOS`). Perfis novos usam
`target_distro` + `min_distro_version` para separar Ubuntu 24.04+ de Debian/Ubuntu
antigos e CachyOS de Arch genérico. Exemplo real: `cursor-bin` é `os: arch` com
`type: paru`, então só é tocado em Arch — em Ubuntu o gerenciador paru nem é consultado.
O comando `run performance` é opt-in: Ubuntu 24.04+ pode instalar zram e aplicar o
sysctl drop-in; CachyOS apenas garante `zram-generator` sem sobrescrever o tuning existente.

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
| Agentes custom | `review` (primary), `planner` (subagent dispatchable), `plan` (regra de `spec-agent/**` no built-in) | `agents/code-reviewer.md` |
| Permissões | no `opencode.json` | `settings.json`: 12 allow · 6 ask · 4 deny |
| Hooks | — | `Stop` → `envctl-verify --hook` |
| Skills (destino) | `~/.config/opencode/skills` | `~/.commandcode/skills` |
| Validação de skill no doctor | **frontmatter + contagem vs manifesto + orçamento do catálogo** | **frontmatter + contagem vs manifesto + orçamento do catálogo** |
| Diretórios criados | 4 (skills, memory, secrets 0700, extras) | 2 (skills, agents) |
| Cleanup dedicado | 4 entradas | 6 entradas |
| IDE integration | — (diagnósticos via lint/typecheck no v2; sem runtime LSP) | VS Code / Cursor / Windsurf via `/ide` |

### Checks do `doctor` por provedor

| Provedor | Checks |
| :--- | :--- |
| OpenCode | `AGENTS.md (global rules)` · `Config file references` (refs `{file:...}` do `opencode.json`) · `Config shape` (V2: sem `agent`/`permission` legacy, sem ações `bash`/`task`, subagent com `description`) · `Database` (tamanho + páginas livres do `opencode.db`) · `Tool Output` (diretório) · `Skills` (frontmatter + contagem) |
| CommandCode | `CommandCode CLI` · `~/.commandcode/` · `Settings` (JSON válido) · `MCP config` (JSON válido) · `Agents` (frontmatter `name` == arquivo **e** schema documentado: `tools`/`disallowedTools`, `permissionMode`, `maxTurns`, `background`, `showOutput`, `model`; `agent`/`agent_output` em `tools` = WARN) · `Skills` (frontmatter + contagem) |
| Ambos | `LSP` (binário no PATH) · `Verify` (verificador + pre-push) · `Git` · `git worktree` (parse de `--porcelain`: `prunable` = WARN, `locked` = INFO, nunca auto-poda) · `TempFolder` |

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
| 9 | O `opencode` tem **dois canais de versão** e o npm não é o mais novo: o pacote npm `opencode-ai` (`latest`) ficou em **1.18.31** (14/09), enquanto as tags do upstream (`anomalyco/opencode`, ex-`sst/opencode`) já estão em **v2.0.7** (17/09) — e é essa linha 2.x que o pacote do Arch (`extra`, 2.0.5) empacota. Instalar por npm/Volta **rebaixaria** a máquina | **Resolvido, estendido 2026-09-24** — `opencode` é provisionado pelo instalador oficial V2 (`https://opencode.ai/v2/install`; PowerShell no Windows, `~/.opencode/bin` no Linux) ou por pacote do SO (`pacman` no Arch), nunca por npm/Volta nem pelo winget (congelado na linha 1.x — manifests bot-authored com delay); `run providers` e `run bootstrap` convergem instalações v1 user-local, validam o major instalado, consultam `pacman -Q` antes de decidir ownership e arquivam cópias locais quando o pacote Arch é autoritativo (ver §1) |
| 10 | `paru` não vinha por padrão no CachyOS e o manifesto Arch não o declarava, embora `run paru` e o `type: paru` (ex.: `cursor-bin`) dependam dele | **Resolvido** — `paru` declarado no bloco pacman (repo `[cachyos]`); no Arch puro, que não tem paru em repo nenhum, o bootstrap constrói do AUR (`base-devel` + `git` também declarados) |
| 11 | `ssh-manager` MCP existia no `configs/opencode.json` (Windows) mas nunca no `configs/opencode.linux.json` — no CachyOS o agente só alcançava VPS via CLI | **Resolvido** — entrada espelhada no linux (`command: [mcp-ssh-manager]`, `timeout: 30000`, `enabled: false`); timeout pina os 30s do `chrome-devtools` (default 5s estoura no handshake) |
| 12 | `zscan` (`@eajdias/zscan-run`) vinha desde o squash inicial em `configs/opencode.json` + `configs/commandcode/mcp.json`, sem dono nem uso | **Resolvido** — blocos deletados nos dois configs; `doctor` acusa `Removed MCP entries` (warning nomeando o servidor) quando o deployado ainda declara; CHANGELOG/lessons e o prefixo `zscan-` do `temp_hygiene` (limpeza, não instala nada) ficam |
| 13 | `configs/AGENTS.linux.md` dizia "Ubuntu Server, usuário `ubuntu`" até no desktop CachyOS (`os: linux` cobria tudo) | **Resolvido** — split `configs/AGENTS.arch.md` (`os: arch,cachyos`: fish, paru, gaming, Cursor) vs linux (`os: debian,ubuntu`); `doctor` avisa `AGENTS.md (identity coverage)` quando nenhuma variante casa com o host (ex.: distro desconhecida) |
| 14 | Chave `yaml` no `lsp` do opencode duplicava o builtin `yaml-ls` (ids provados no binário: merge `item.extensions ?? existing?.extensions` dispara os dois em `.yaml/.yml`); `pylsp` + `pyright` disparavam duplo em `.py` | **Resolvido** — `yaml` renomeado para `yaml-ls` nos dois configs (+ `id: yaml-ls` no `lsp.yaml`); `pylsp` removido dos dois configs (pesquisa 2026: consenso é servidor de tipos `pyright`/`ty` + `ruff` p/ lint — `pylsp` legado, mais lento; binário segue provisionado p/ IDE/shell). Requisitos de spawn (typescript/pyright/eslint exigem dep no projeto, gopls exige `go`) documentados como causa esperada de "não ativa" |
| 15 | `taplo` tem dois canais (`taplo-cli` via pacman + `@taplo/cli` via npm) e o `run lsp` prefere o npm mesmo no Arch | **Exceção intencional** — mesmo dono nos dois canais, sem sombreamento entre gerenciadores. Revisitar só com skew de versão observado |
| 16 | opencode v2 ignora runtime LSP (`lsp` aceito mas inerte), `subagent_depth` top-level (WARN `omitted unsupported legacy setting`) e `instructions` (aceito, não carregado) | **Resolvido 2026-09-22** — bloco `lsp`, `subagent_depth` e `instructions` removidos dos dois configs (formato nativo V2: `agents`/`permissions[]`/`plugins`/`skills[]`/`mcp.servers`); `review` com `mode: primary` explícito, `plan` sem `mode` (preserva o built-in — customs só com IDs novos); binários LSP seguem provisionados p/ shell/IDE e o `doctor` os audita como toolchain |
| 17 | Retorno do runtime LSP no opencode v2 (hoje: config validada, nenhum servidor inicia, zero diagnósticos — [docs](https://dev.opencode.ai/v2/docs/lsp/)) passa batido sem monitoramento | **Monitoramento mensal** — conferir `pacman -Si opencode` (versão no `extra`) + changelog upstream; quando o runtime voltar: re-testar bloco `lsp` per-project (template em `patterns.md`), re-adicionar via `lsp.yaml` → JSON, sem plugin V2 antes da API sair de beta |
| 18 | O `plan` do envctl é `primary`, portanto **não** aparece no catálogo de subagentes do OpenCode: a skill `subagent-routing` mandava despachar `plan` via task tool e o `review` só podia cair um nível (`review` → `explore`/`general`) | **Resolvido 2026-09-25** — novo custom `planner` com `mode: subagent` (ID novo, `description` obrigatória, boundary read-only igual ao `plan`, sem subagentes aninhados) nos dois configs; `plan` continua `primary` para o Tab. Contrato travado por `TestOpenCodeConfigTemplates` + `doctor` (`Config shape`) |
| 19 | O `doctor` rodava `git worktree list` e **descartava a saída**: worktree `prunable` (gitdir apagado) ou `locked` ficava invisível, e o `CHANGELOG` antigo prometia um "worktree integrity audit" que não existia no código | **Resolvido 2026-09-25** — parser puro de `git worktree list --porcelain` + findings (`prunable` = WARNING com hint de `git worktree prune`; `locked` = INFO preservado); nenhuma remoção/auto-fix. Convenção de path fixada em `.worktrees/<type>-<slug>` (`.opencode/opencode.json` + `.gitignore`) |
| 20 | A skill `subagent-supervision` orquestrava tools que não existem nos **dois** runtimes da forma que o texto afirmava, e a correção seguinte (2026-09-25) overcorrigiu: removeu `agent_output`/`agent_id`/`kill_shell` do arquivo inteiro, quebrando o CommandCode, que **tem** essas tools nativamente | **Resolvido 2026-09-25 (2 tempos)** — (1) o texto virou o lifecycle do OpenCode V2 (`sessionID` + `opencode api`), o que é correto só lá; (2) pesquisa na doc oficial do CommandCode (docs/agents, docs/background-tasks, docs/worktrees) + grep no bundle instalado 1.65.0 provou que o CommandCode tem `agent`+`run_in_background` → `agent_id`, `agent_output({action:"wait"|"status"|"kill"})`, `shell_output`/`task_output`/`kill_shell`/`monitor_command`, e **não** tem `mode` nem `sessionID`. Skill compartilhada passou a ter **uma seção por runtime** (`## OpenCode V2` e `## CommandCode`) e o teste de conteúdo valida cada coluna isoladamente (term exigida numa, termo proibido na outra) — `TestSubagentSupervisionIsRuntimeAware` / `TestTaskHangWatchdogIsRuntimeAware` |
| 21 | O `doctor` só checava `name == filename` nos agentes do CommandCode: um valor de schema errado (tool inexistente, `permissionMode` inválido, `maxTurns` não-numérico, `agent` em `tools`) é **ignorado em silêncio** pelo runtime, e o agent continuava "válido" no doctor com capacidades diferentes das pretendidas — o mesmo modo de falha que o check `Config shape` eliminou no lado OpenCode | **Resolvido 2026-09-26** — `validateCommandCodeAgentFrontmatter` valida o schema documentado (docs/agents, conferido no bundle 1.65.0) e reporta **o campo** no check `Agents`: `WARN` para valor que muda comportamento ou impede carga, `INFO` para id de tool fora do catálogo (para um upgrade do CommandCode não deixar o doctor vermelho para sempre), e nada para chave desconhecida (o runtime também ignora). `tools`/`disallowedTools` aceitam `"a, b"`, lista YAML e `"*"`; `agent`/`agent_output` são sempre `WARN` (delegação tem um nível) |
| 22 | Convenção de worktree do repo (`.worktrees/`) não é alcançável no CommandCode: lá não existe chave de diretório em `settings.json` e o default é `~/.commandcode/worktrees/<repo>-<hash>/<name>` (fora do repo) | **Documentado 2026-09-26, sem config nova** — as duas rotas são `git worktree` de verdade, então o check `git worktree` do `doctor` já enxerga ambas; a ponte para a convenção do projeto é `cmdc -w "$PWD/.worktrees/<slug>"` (path absoluto é usado verbatim), registrada na skill `git-workflow` e nos `AGENTS` do CommandCode. `/worktree` e `enter_worktree` continuam no dir gerenciado — não há como redirecioná-los, e fingir convergência seria errado |
| 23 | A doc do CommandCode anuncia `when_to_use` como gatilho extra p/ auto-invocação, e o bundle confirma que ele existe — mas o teste no runtime mostrou que ele **nunca chegava ao modelo**: o catálogo é `description + "\n\n" + when_to_use` cortado em **249 caracteres**, e sem `COMMANDCODE_SKILL_CATALOG_CHAR_BUDGET` o modelo recebe **só nome + location**, sem description nenhuma | **Resolvido 2026-09-26 por medição** — 5 probes `cmdc -p` (budget unset → nomes; 20000 → nomes; 60000 → description truncada, `when_to_use` fora do corte; redesenho → ambos completos). As 7 skills de roteamento foram **redesenhadas**: description curta (o quê + quando, 102–123 chars), `when_to_use` com o gatilho em linguagem natural (98–122 chars) e a lista extendida movida para o corpo (`## Triggers`, lida só depois do load). Contrato travado em `TestWhenToUseFitsCommandCodeCatalog`: `description + 2 + when_to_use ≤ 247` **caracteres** (runes, não bytes — `String.slice` do JS conta caracteres). Ganho de contexto no catálogo do OpenCode: −1.9k chars/turn |

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
| Gaming / debloat | `run gaming` (38 pkgs: Steam, Proton CachyOS, gamescope, MangoHud + GOverlay, emuladores, Heroic/Lutris, Sunshine, scraper/tools, `lact`, scx, ananicy, X11 trio) · presets `gaming.conf`/`MangoHud.conf` (seed) · `doctor` seção Gaming (opt-in via Steam: pacotes + 4 serviços + `sched_ext` + cmdline + RADV + multilib) · tuning root/reboot documentado no repo · 8 tweaks Windows (6 DWord + 2 módulos PowerShell) + `run debloat` opt-in (76 tweaks Windows: telemetria/privacidade/gaming/Appx/serviços; `doctor` agrega por categoria em `INFO`, nunca `--fix`) · Tier 3 manual (OneDrive, hibernação, power plan, Teredo, `.wslconfig`, Copilot/Recall) documentado no repo | — |

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
3. Para ser **dispatchable**, `mode: subagent` + `description` não vazia (o pai escolhe pela description) + boundary read-only explícito; `planner` é o exemplo de referência e `TestOpenCodeConfigTemplates` + o check `Config shape` do `doctor` cobrem o contrato.

**LSP** → `manifests/lsp.yaml` (binários p/ shell/IDE — `run lsp` + `doctor`)
1. Informe `install_type` (`volta`, `npm`, `pip`, `go`), `install_target` e `check_binary`.
2. NÃO espelhe entrada no `opencode.json`: o runtime v2 ignora o bloco `lsp` (assimetria #16) — foi removido dos dois configs em 2026-09-22.

**Debloat (opt-in, Windows)** → `manifests/debloat.yaml` (`run debloat` + `doctor` seção Debloat)
1. Reusa o schema de `windows.yaml` (`id`, `description`, `path`/`name`/`value`/`type`, `category`); tipos novos: `Appx` (conforme = ausente, `path` vazio) e `Service` (`value` = `Disabled`/`Manual`, conforme = `StartType`).
2. Categorias fechadas: `telemetry`, `privacy`, `gaming`, `apps`, `services` (o `doctor` agrega 1 linha por categoria, `INFO` em drift — nunca `WARN`, nunca `--fix`).
3. Listas curadas e conservadoras: Xbox/Teams/Outlook, serviços de máquina (`Dell*`, `AnyDesk`, `Firebird*`, `Tailscale`, `sshd`), `Spooler`/`WSearch`/`SysMain`/`NgcSvc` e tudo do Tier 3 (OneDrive, hibernação, power plan, Teredo, `Binary`, `.wslconfig`) ficam FORA — Tier 3 vive na skill `windows-debloat` (`os: windows`).
4. `KeyboardDelay` e cia: conferir o tipo REG_* real no registro — o upstream declarava `DWord` para valor `REG_SZ` (o teste de manifesto trava `String`).

**CLI de agente (provedor)** → `manifests/packages.yaml` + `run providers` (fase 0)
1. Confirme o **canal de versão** antes de escolher o gerenciador: o mesmo produto costuma ter
   linhas diferentes por canal (pacote do SO ≠ npm ≠ instalador oficial). Foi assim que o
   `opencode` quase foi rebaixado (assimetria #9).
2. Se o CLI já existe na máquina mas não é do envctl nem do Volta, **não instale** uma segunda
   cópia — reporte. Cópia em `~/.local/bin`/`~/.opencode/bin` vence no PATH e congela a versão
   ali instalada. A única exceção é o v1 legado no Ubuntu/Debian, que precisa convergir para
   v2; no Arch, um binário do pacman nunca é sombreado.
3. Um CLI que o próprio envctl garante entra em `providerCLIs()`
   (`internal/usecase/provision_providers.go`): com `voltaPkg` ele é atualizável; com
   `windowsInstaller`/`installer` e `requiredMajor` ele instala, atualiza e valida a versão.
4. Declare o `check_command` no manifesto para o `doctor` auditar a presença naquela plataforma.

**Sempre**: `gofmt`/`go build`/`go vet`/`go test` + `golangci-lint run --new-from-rev=origin/main`
antes do push. O `envctl-verify` cobre os sete automaticamente: findings de lint/formatação são advisories, enquanto builds, vets, testes e comandos explícitos são bloqueantes.
