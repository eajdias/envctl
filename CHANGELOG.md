# Changelog

Todas as alterações notáveis no projeto **`envctl`** serão documentadas neste arquivo.

O formato é baseado no [Keep a Changelog](https://keepachangelog.com/pt-BR/1.0.0/),
e este projeto adere ao [Versionamento Semântico](https://semver.org/lang/pt-BR/).

---

## [1.8.0](https://github.com/eajdias/envctl/compare/v1.7.0...v1.8.0) (2026-09-26)


### Features

* **cli:** add windows, vps and cachyos full machine profiles ([6953f1e](https://github.com/eajdias/envctl/commit/6953f1e580e2714a6d7c45b65bfda7884a0dda12))


### Bug Fixes

* **release:** stop building and shipping darwin binaries ([#33](https://github.com/eajdias/envctl/issues/33)) ([47c691a](https://github.com/eajdias/envctl/commit/47c691a0f869dfd17684c064a61540242982c631))
* **skills:** stamp the quarantine mtime so the recovery window is real ([#36](https://github.com/eajdias/envctl/issues/36)) ([d5143fb](https://github.com/eajdias/envctl/commit/d5143fb128a21f592efee29893f8599c20935555))

## [1.7.0](https://github.com/eajdias/envctl/compare/v1.6.0...v1.7.0) (2026-09-26)


### Features

* **linux:** add OS-specific performance baseline ([#27](https://github.com/eajdias/envctl/issues/27)) ([786296e](https://github.com/eajdias/envctl/commit/786296ec22842d80267ca56e58ec8030d820c945))
* **opencode:** add dispatchable planner subagent and worktree safety checks ([8a80924](https://github.com/eajdias/envctl/commit/8a80924304186eb40f494d5f3b5f0739aace8246))
* **opencode:** dispatchable planner subagent, runtime-aware supervision, and a 12-skill catalog ([01b0ed0](https://github.com/eajdias/envctl/commit/01b0ed07998aa76d16ce631cbfb236c9e0cafca6))
* **skills:** expand engineering playbooks ([#30](https://github.com/eajdias/envctl/issues/30)) ([3f68d21](https://github.com/eajdias/envctl/commit/3f68d21f6cba8b746ca68823082fd3b9c2d0e976))


### Bug Fixes

* **backup:** prune nested backups recursively and keep them out of the snapshot ([f4f0f0f](https://github.com/eajdias/envctl/commit/f4f0f0ff553d6888072fe900a29ba91fcc388a30))
* **docs:** recover the gaming and debloat knowledge the catalog cut dropped ([2ec12d9](https://github.com/eajdias/envctl/commit/2ec12d9bdc473d38a2b685169cbf0581c9f26b2c))
* **skills:** bound the stale-skill quarantine with a recovery window ([e5f37be](https://github.com/eajdias/envctl/commit/e5f37be5ad46761d8f9d3cdf1190e3e311db88a4))
* **skills:** make supervision guidance runtime-aware per agent ([30b6edb](https://github.com/eajdias/envctl/commit/30b6edbe49e55ef784b56b552d91687156ceeee1))
* **skills:** restore the variant-analysis phase and re-point removed skills ([8e06582](https://github.com/eajdias/envctl/commit/8e0658282682addbd476d707e4d54da08cc5a709))
* **verify:** make inferred checks advisory ([#29](https://github.com/eajdias/envctl/issues/29)) ([c34613a](https://github.com/eajdias/envctl/commit/c34613a486ccd50ae0a24bd14c11762e6335af94))

## [1.6.0](https://github.com/eajdias/envctl/compare/v1.5.0...v1.6.0) (2026-09-24)


### Features

* **debloat:** absorb windows11-clean as opt-in run debloat ([#24](https://github.com/eajdias/envctl/issues/24)) ([9f8c80f](https://github.com/eajdias/envctl/commit/9f8c80f6631bafc3676fb07fabe283cb8c623cc3))
* **platform:** absorb CachyOS toolbox and default OpenCode v2 ([b7d12bc](https://github.com/eajdias/envctl/commit/b7d12bc3c6ec7619a37f763a52ee7f1a5beda23c))

## [1.5.0](https://github.com/eajdias/envctl/compare/v1.4.0...v1.5.0) (2026-09-24)


### Features

* **opencode:** install v2 on Windows via official installer ([1b628e1](https://github.com/eajdias/envctl/commit/1b628e1a4856642c248eb735caa5350aa26ccfe6))

## [1.4.0](https://github.com/eajdias/envctl/compare/v1.3.0...v1.4.0) (2026-09-24)


### Features

* **gaming:** absorb cachyos-init tuning into manifests, doctor and skill ([2663fc9](https://github.com/eajdias/envctl/commit/2663fc9aae59908fe35d6b96f4dea0aae55615af))


### Bug Fixes

* lsp-smoke-test wording, lsp-return monitor, windows CI skips ([#21](https://github.com/eajdias/envctl/issues/21)) ([d8592ea](https://github.com/eajdias/envctl/commit/d8592eabdcac95e5bbc92182543bd5f584e37183))

## [Unreleased]

- **Changed**: the release PR no longer waits for a maintainer to click "approve workflow". GitHub treats the release-please bot as an outside collaborator, so its `pull_request` run sat at `action_required` and every release needed a human. The CI pipeline now also runs on `push` to `release-please--**` — a push run needs no approval, and branch protection only requires that the checks reported on the head SHA — while the bot's `pull_request` run is not created at all, via `paths-ignore` on the two files it generates. Verified empirically on a throwaway `release-please--*` branch: Lint + Test on ubuntu and windows all green, no approval step.
- **Removed**: the release pipeline no longer builds or ships darwin binaries. `release.yml` had a
  `# 3. Darwin amd64 & arm64` section feeding `envctl-darwin-amd64`, `envctl-darwin-arm64` and their tarballs into
  `SHA256SUMS.txt` and the release upload, so every release advertised macOS artifacts for a platform whose manifest,
  bootstrap and lint all reject it. Build, checksum and upload entries are gone; the workflow builds Windows and Linux
  only, and its header now states the scope so the next reader does not re-derive it.

- **Changed**: the agent skill catalog is now **12 entries instead of 50**, chosen by a measured rule: a skill earns a catalog slot only when it holds something the model would not do by itself (user preference, non-obvious procedure, or a trap already paid for). Language, framework, tool and domain knowledge moved to `code-playbooks/references/*.md` (loaded on demand behind one index entry); mandatory behavior moved to the `AGENTS.md` manifests; envctl product knowledge left the global tier and stays in the repo. Measured with `cmdc -p`: 50 skills cost ~6.4k tokens per turn and, at the CommandCode default catalog budget, the runtime degrades to names-only and **auto-activation stops happening**; 12 entries fit the default, so descriptions reach the model with no environment variable. `clarify-before-acting` merges `grill-me` + `ask-questions-if-underspecified` + `grilling`, and `systematic-debugging` absorbs `variant-analysis` as its last phase.
- **Added**: `envctl` skill (operate/audit/provision the machine, product internals stay in the repo) and a doctor check that projects the catalog size against the CommandCode budget, so a silent fall back to names-only is reported instead of discovered later.
- **Fixed**: the skill catalog restructure had silently dropped the `variant-analysis` content — `docs/` and the spec claimed it was absorbed as the last phase of `systematic-debugging` while the skill still had only four phases. The phase is now really there (root-cause family search: exact match first, one generalization step at a time, edge cases, severity triage), and the nine references to removed skills that were left inside the surviving skills (`context7-auto`, `vps-agent-dispatch`, `aur-headless-install`, `api-contract-design`, `docs-sync`, `universal-test-runner`, `verification-before-completion`) now point at the MCP server, the SSH path, `code-playbooks/references/*.md` or the `AGENTS.md` rule.
- **Fixed**: the stale-skill quarantine had no retention, so it grew by one directory per removed skill forever (39 per runtime on the machine where the catalog went from 50 to 12). Quarantined entries now expire after a 30-day recovery window on the next deploy, the count is reported per runtime, and non-directory entries in the trash tree are left alone since that tree is not exclusively ours.
- **Fixed**: CI only ran on `push` to `main` and pull requests targeting `main`, so a feature-branch PR — where review actually happens — got no check at all. That gap is how a docs/impl drift reached `main` unverified in the first place. The `pull_request` trigger now covers every branch.
- **Fixed**: two product-knowledge sets were lost when their skills left the global catalog. `cachyos-gaming-setup` (Limine kernel cmdline, scx_bpfland scheduler, AMD/RADV tuning, MangoHud preset, Steam/Proton, emulator renderer table) and `windows-debloat` (the eight destructive Tier 3 PowerShell blocks) were deleted with no destination, while `docs/os-and-agent-matrix.md` still claimed the gaming tuning was "documented in the repo". Both are now repo guides — `docs/guides/cachyos-gaming.md` and `docs/guides/windows-debloat-tier3.md` — indexed from the README.
- **Fixed**: `envctl doctor` pointed users at two skills that no longer exist. A CachyOS box missing kernel parameters or the RADV driver was told to "see skill cachyos-gaming-setup", and no test asserted the string, so nothing caught it. The hints now reference the repo guides, and the other dangling references in code, manifests and docs (`lsp-smoke-test`, `ssh-vps`, `windows-debloat`) were re-pointed or removed.
- **Fixed**: six documents still claimed "50 skills" after the catalog cut (principles, architecture, manifests, both OS guides, ADR 0001) — the counting mistake `docs-sync.md` warns about, committed by the same change that introduced it.
- **Fixed**: provisioning backups accumulated one file per redeploy forever, because `pruneTimestampedBackups` skipped directories and therefore never reached the nested `skills/<name>/SKILL.md` tree. The prune is now recursive and groups by the original file's path, so two `SKILL.md` files in different directories do not compete for the same `keep_newest` slot. Measured on the machine: 41 backups pruned per runtime.
- **Fixed**: the reverse `snapshot` copied accumulated `.bak` files from a deployed skill tree into `configs/skills/`, which would have committed stale skill text (the old 50-skill `agent-memory` description) and shipped it to every machine on the next deploy. `copyDir` now skips provisioning backups in both directions.
- **Removed**: any macOS/darwin support or mention. `bootstrap.sh` now accepts Linux only, the `os:` lint rejects
  the darwin token, `DistroDarwin` and the doctor branches that only served it are gone, and the README section plus
  `docs/guides/macos.md` were removed.
  - **Correction (found while publishing this release):** the claim originally recorded here — that "the release never
    built darwin binaries" — was **false**. `release.yml` had a `# 3. Darwin amd64 & arm64` build section feeding
    `envctl-darwin-amd64`, `envctl-darwin-arm64` and their tarballs into `SHA256SUMS.txt` and the release upload, so this
    very release shipped darwin assets. The claim came from grepping the workflow with `head -12`, which truncated the
    evidence before that section. The build, checksum and upload entries are removed in the next patch; v1.7.0 itself
    still carries the darwin assets, because a published release's assets are immutable.

- **Added**: `envctl doctor` now validates the documented CommandCode agent frontmatter schema (`tools`/`disallowedTools`, `permissionMode`, `maxTurns`, `background`, `showOutput`, `model`, `reasoningEffort`) on top of the existing `name == filename` check, and names the offending field: a wrong value is ignored by the runtime in silence, so the agent used to load with fewer capabilities than its frontmatter asked for while the doctor stayed green. An unknown tool id is informational so a CommandCode upgrade cannot keep the doctor red; `agent`/`agent_output` in `tools` is a warning, since delegation is one level deep.
- **Added**: dispatchable `planner` subagent (`mode: subagent`, read-only, `spec-agent/**` only) in both OpenCode config templates, keeping the built-in `plan` as the primary Tab agent; `.opencode/opencode.json` standardizes project worktrees in `.worktrees/<type>-<slug>`.
- **Added**: `envctl doctor` audits the native OpenCode V2 config shape (`Config shape`) and reports worktree hygiene from `git worktree list --porcelain` (`prunable` → warning, `locked` → info, never auto-pruned).
- **Changed**: agent instructions now require proactive skill loading when a description matches the task, `subagent-routing` documents inline-first execution, and `subagent-supervision`/`task-hang-watchdog` use the real OpenCode V2 lifecycle (`sessionID`, `opencode api post /api/session/<id>/interrupt`) instead of nonexistent tools.

- **Fixed**: `envctl-verify` now treats inferred lint and formatting findings as non-blocking advisories, excludes all flat ESLint config variants from its fallback file selection, honors an explicit `package.json` `lint` script as the project's blocking authority, and invalidates hook caching when tools, comparison refs, overrides, or oversized untracked content change; the global pre-push also fails closed when its verifier is missing.

- **Added**: baseline de performance Linux separado por SO — `run performance` para Ubuntu Server 24.04+ (toolbox headless, `systemd-zram-generator` e drop-in sysctl com backup) e CachyOS (garantia de `zram-generator` sem sobrescrever o tuning existente); `doctor` agora reporta swap/zram/governor/scheduler/journald/fstrim/serviços como auditoria read-only. Swapfile, journald, governors, schedulers, serviços e kernel cmdline permanecem manuais/benchmark-gated.

- **Added**: cinco skills portáteis de engenharia — `go-development`, `python-development`, `mcp-tool-design`, `technical-research` e `frontend-markup` — catálogo 45→50 (46 Win / 46 Ubuntu / 48 CachyOS / 44 macOS).

- **Added**: 21 pacotes pacman (`os: arch,cachyos`) absorvidos do inventário CachyOS —
  `eza`, `zoxide`, `direnv`, `lazygit`, `lazydocker`, `tmux`, `sqlite`, `restic`, `rclone`,
  `rsync`, `btop`, `duf`, `glances`, `fastfetch`, `micro`, `meld`, `cmake`, `ninja`, `mosh`,
  `android-tools`, `unzip` (nomes/`check_command` verificados com `pacman -Si` no CachyOS) +
  `yt-dlp` via `uv tool` (portátil) — matriz §1: Arch 48→70 aplicáveis.
- **Added**: `gaming.yaml` 33→38 pkgs — `heroic-games-launcher`, `lutris`, `sunshine`
  (pacman) + `skyscraper-git`, `hactool` (paru/AUR).
- **Added**: skills portáteis `tailscale` (roadmap item 3, parcial) e `syncthing-ops`
  (via REST API, nunca `config.xml` na mão) — com `windows-debloat` (main), catálogo 41→45
  (Win 41, Ubuntu 41, CachyOS 43).
- **Added**: debloat opt-in do Windows 11 absorvido do `windows11-clean` — `envctl run debloat`
  (76 tweaks em `manifests/debloat.yaml`: 12 telemetria + 12 privacidade + 12 gaming-win +
  31 Appx + 9 serviços safe-only; tipos novos `Appx`/`Service` no TweaksManager com check
  idempotente; Xbox/Teams/Outlook, serviços de máquina e `Spooler`/`WSearch`/`NgcSvc`
  excluídos) + `doctor` com 5 linhas agregadas por categoria (`INFO` em drift, nunca
  `--fix`) + skill `windows-debloat` (`os: windows`; Tier 3 manual: OneDrive, hibernação,
  power plan, Teredo, `.wslconfig`, Copilot/Recall). Checks em batch (`CheckBatch`:
  1 spawn PowerShell por família em vez de 1 por tweak). `run all`/`run windows` intocados.
- **Fixed**: `run providers` e `run bootstrap` no Ubuntu/Debian passam a instalar e convergir
  OpenCode pelo canal oficial V2 (`https://opencode.ai/v2/install`, `~/.opencode/bin`), validar
  o major instalado e persistir o PATH em shells POSIX/fish. Instalações v1 user-local são
  atualizadas; no Arch, a propriedade é consultada no banco do pacman, cópias envctl locais
  são arquivadas com backup e o pacote do sistema continua sendo a autoridade.

## [v1.3.0] - 2026-09-22

### 🧹 OpenCode configs em formato nativo V2 + plan built-in (sem `lsp`, sem `dcp.jsonc`)

- **Removed**: bloco `lsp` de `configs/opencode.json` + `configs/opencode.linux.json` — o runtime do opencode v2 aceita e ignora (nenhum servidor inicia; sidebar nunca lista). Binários seguem provisionados (`manifests/lsp.yaml`, `run lsp`, bootstrap) para shell/IDE (`/ide` + `get_diagnostics`, modelo do CommandCode) e o `doctor` continua auditando presença + handshake stdio como toolchain.
- **Added**: `experimental.subagent_depth: 2` nos dois configs — o subagent tool lê `experimental.subagent_depth` (`??1`, erro orienta aumentar; verificado por strings no binário v2.0.8). Profundidade 2 cobre a topologia real (`primary → review/plan → explore/general`, folhas terminais); default 1 desabilitaria o dispatch aninhado.
- **Added**: `instructions` (URL `shell_strategy.md`) de volta nos dois configs — `Config.Instructions` ("Additional instruction files or patterns to include") é campo suportado no binário v2.0.8 (o guia oficial lista `instructions` como "require no migration"); o "nunca carregado" anterior era leitura de log de terceiro, nunca confirmada no `debug config` desta máquina.
- **Removed**: `configs/dcp.jsonc` + entrada `opencode_dcp` do `manifests/shell.yaml` (YAGNI: plugin V1 quebra o boot do v2 desde 2026-09-19, config sem consumidor) + cleanup `stale_opencode_dcp` que remove o órfão `~/.config/opencode/dcp.jsonc` nas máquinas.
- **Changed**: os dois configs reescritos no formato nativo V2 (`agents` com `system`, `permissions[]` com `shell`/`subagent`, `request.body.temperature`; `skills[]`; `mcp.servers` com `disabled` + `timeout:{catalog,execution}`; `plugins`) — `review` (custom novo) com `mode: primary` explícito; `plan` SEM `mode` (preserva o built-in — customs nunca sobrescrevem IDs pre-existentes; efetivo `primary` por merge). `opencode debug config` carrega com zero diagnostics.
- **Docs**: matriz §1–§2 (configs 27 Win / 25 Ubuntu / 25 Arch; LSP/binários + pruning nativo; assimetria #16), checklist LSP (só `lsp.yaml`, sem espelho no JSON), `skills.md`, `principles.md`, `manifests.md`, `README.md`, `SKILL-INDEX.md`, `REFERENCE.md` (seção `cli.json`/sidebar fora do envctl), seeds `configs/memory/` e linha `agent/LSP` do `envctl opencode --help` sincronizados.
- **Changed**: `agents.plan` reduzido à exceção mínima (`permissions: [{edit spec-agent/** allow}]`) nos dois configs — `description`/`request`/`steps`/`system` (~170 linhas) deletados, built-in `opencode.plan` cobre mode/questions/read-only por merge; `debug agents` prova: mode `primary`, 45 perms, `* deny` + `~/.opencode/plan/* allow` + `spec-agent/** allow`. Shell allowlist mantida no nível do agente (top-level `permissions` com `shell * ask` represaria o `build` — verify-then-global decidiu NÃO mover). Ensino de convenções movido p/ seção `## Planejamento` nos 3 `AGENTS.md`.

### 🛠️ OpenCode usability no CachyOS (ssh MCP, sem zscan, LSP com handshake, memória obrigatória, identidade arch)

- **Added**: MCP `ssh-manager` no `configs/opencode.linux.json` (espelho do Windows + `timeout: 30000`, `enabled: false` — liga por sessão via `/mcp`).
- **Removed**: blocos `zscan` (`@eajdias/zscan-run`) de `configs/opencode.json` + `configs/commandcode/mcp.json`; descrições sincronizadas (`shell.yaml`, matriz §2).
- **Added**: `doctor` acusa `Removed MCP entries` (warning nomeando o servidor) em `opencode.json`/`mcp.json` deployados e `AGENTS.md (identity coverage)` quando nenhuma variante de AGENTS casa com o host — 7 testes novos em `internal/usecase/doctor_audit_test.go`.
- **Added**: `doctor` valida handshake stdio de cada LSP provisionado (stdin fechado + ausência de erro de conexão, padrão da skill `lsp-smoke-test`; exit code mente — node servers saem 1 com EOF saudável, calibrado nos 14 ao vivo).
- **Fixed**: chave `yaml` → `yaml-ls` nos dois configs do opencode (+ `id: yaml-ls` no `lsp.yaml`) — a chave custom disparava um SEGUNDO servidor junto do builtin em `.yaml/.yml` (merge upstream + ids provados no binário `/usr/bin/opencode`).
- **Added**: `configs/AGENTS.arch.md` + `configs/commandcode/AGENTS.arch.md` (`os: arch,cachyos`: fish, paru, gaming, Cursor, sem usuário hardcoded); `AGENTS.linux.md` volta a ser só `debian,ubuntu`; contagem LSP corrigida (14, não 17).
- **Changed**: bullet de memória obrigatório nos 6 AGENTS (LOAD 1x projeto→global, SAVE ao errar/aprender via `memory-promotion`, REFLECT ao fechar).
- **Removed**: `pylsp` do `lsp` dos dois configs do opencode (13 entradas no Linux / 14 no Windows) — consenso 2026: servidor de tipos (`pyright`, Pylance backend, rápido e mantido) + `ruff` p/ lint/format; `pylsp` (comunitário, lento, era plugin) duplicava diagnósticos em `.py`. Binário segue provisionado (`lsp.yaml`, doctor, bootstrap) p/ IDE/shell.
- **Added**: regra `Nunca deduza, nunca insista` nos 6 AGENTS (todos os OS, dois agentes) — relato do usuário sobre estado local observável é evidência de primeira classe: re-testar na hora, hipótese como hipótese, sem repetir prescrição sem evidência nova.

### 🔒 Distro-strict OS scope + phase-0 and safety fixes (review 2026-09-22)

- **Changed**: bare `os: linux` banned from manifests — explicit `arch,cachyos` (CachyOS desktop) · `debian,ubuntu` (Ubuntu Server VPS) · shared-POSIX 4-list `arch,cachyos,debian,ubuntu` · portable omits `os:`; lint test `TestManifestOSLint` fails on bare-`linux`/quoted-`check_command`/unknown tokens (Policy 0).
- **Fixed**: phase 0 probes resolve on the toolchain PATH (`resolveOnToolchainPath`; `installedVersion` executes the resolved absolute path since `cmd.Env` never affects `LookPath` — non-login shells no longer reinstall Volta/Node/CLIs every run); `installStandaloneProvider` prefers pacman `extra` on Arch (no more curl shadow); `vps-agent-dispatch` drops the `npm install -g opencode-ai` fallback (per-distro channels).
- **Fixed**: `gaming.yaml` is Arch-only (19 entries `arch,cachyos`) + runtime refusal off-Arch; apt↔pacman cross-distro warning spam gone; AUR/gaming skills no longer deploy to Ubuntu (38 Win / 38 Ubuntu / 40 CachyOS); 11 quoted `check_command`s removed (ID fallback owns those checks).
- **Fixed**: Windows tweaks PS-quote `Name/Path` + typed `Value` (`psQuote`/`psValue`); `WriteWithBackup` tmp+rename with unique same-second suffixes; `CopyEmbeddedTree` diff-gated with `.bak` + exec-bit preservation.
- **Docs**: matrix/README/doctor-doc numbers recounted from manifests (configs 28 Win / 26 Ubuntu / 26 Arch via `MatchesOS`; skills 38/38/40; git 6/4/4; env 2/2/2; LSP 15 manifesto); Ubuntu row clarified (no Cursor/IDE, agent LSPs apply headless); `taplo` dual-channel documented as intentional exception (#15).

### 🐛 Plugins quebrados no opencode v2 removidos do config

- **Removed**: `@tarquinen/opencode-dcp@latest` e `@dietrichgebert/ponytail` de `configs/opencode.json` + `configs/opencode.linux.json` (resta só `@prevalentware/opencode-goal-plugin`) — ambos falham em todo boot no opencode v2.0.8 com `PluginModule.LoadError: Plugin must export a default definition with an id and an effect or setup function (cause: SchemaError(Expected object at ["default"]))` (export V1 `async (ctx) => {...}` em vez de `Plugin.define({id, setup})`; latest já é o quebrado: dcp 3.1.15, ponytail 4.10.0). `dcp.jsonc` segue provisionado para o retorno; re-adicionar após migração upstream (`https://opencode.ai/v2/docs/build/plugins/migrate-v1`).
- **Docs**: `configs/REFERENCE.md`, `configs/AGENTS.md`, `configs/AGENTS.linux.md` e seed `configs/memory/patterns.md` sincronizados (manifest vence doc).
- **Fixed**: `VoltaManager.IsInstalled` false-positive para pacotes scoped (`@playwright/cli`): `strings.Split(id, "@")[0]` é `""` para scoped, e qualquer linha do `volta list` com token isolado `"@"` (aparece quando há `~/package.json` com pin — `(current @ /path/package.json)`) casava via `HasPrefix(t, "@")`, então o `run volta` pulava a instalação enquanto o doctor acusava ausente. Matcher extraído para `voltaListContains` (escopo preservado, versão ignorada, guarda contra vazio) + `TestVoltaListContains` (7 casos).
- **Note**: pós-`v1.2.144`, este release `v1.3.0` consolida 4 commits da branch (`5f17887` usability CachyOS, `9a10a1a` spec, `8f332ef` distro-strict, `da5ab9d` docs sync) + o trabalho não-commitado de migração V2 acima — o workflow gera a tag automaticamente no merge (`v1.2.<commit-count>`, publicado como `v1.2.148`); `v1.3.0` é o número semântico pretendido (minor: remoção de comportamento `lsp`/`dcp` + plan built-in) — criar a tag manual no commit do merge se desejado.

## [v1.2.144] - 2026-09-18

### 🚀 Fase 0: os provedores prontos antes de tudo

- **Added**: `envctl run providers` — preflight que roda como **passo 0 do `run all`** (ou sozinho, quando for só isso): garante Volta (instala se faltar — instalador oficial no Linux, `Volta.Volta` via winget no Windows), um runtime Node default (usando o **mesmo spec do manifesto**, para os dois não divergirem) e os dois CLIs de agente. O `command-code` é atualizado via Volta quando o npm tem versão nova — Volta resolve o `latest`, então "faltando" e "desatualizado" são o mesmo comando. Validado rebaixando o pacote de propósito: a fase 0 reportou `1.55.1 -> 1.56.0` e, na execução seguinte, `1.56.0 (Volta, current)`.
- **Fixed**: o bootstrap instalava o opencode **por npm** e só usava o instalador oficial como fallback. O pacote npm `opencode-ai` está na linha **1.18.x** (14/09) enquanto as tags do upstream — `anomalyco/opencode`, ex-`sst/opencode` — já estão em **v2.0.7** (17/09), e é essa linha que o pacote do Arch (`extra`, 2.0.5) empacota: o caminho npm **rebaixaria** a máquina, e a cópia em `~/.local/bin` ainda venceria no PATH o binário mais novo. O bootstrap passou a usar o instalador oficial.
- **Added**: `opencode` declarado no manifesto nas plataformas em que existe como pacote (`pacman` no Arch/CachyOS, `SST.opencode` no winget). No Debian/Ubuntu segue o instalador oficial. Antes, o Windows não tinha **nenhum** caminho provisionado para o opencode.
- **Changed**: a fase 0 nunca sombreia binário que não é dela. Se o `opencode` vem de pacote do SO, ela **reporta** versão e origem em vez de instalar uma segunda cópia — mesma lição do `fzf`.
- **Added**: `paru` declarado no bloco pacman (`[cachyos]`), junto com `base-devel` e `git`. O `run paru` e os pacotes `type: paru` (ex.: `cursor-bin`) dependiam de um AUR helper que o CachyOS **não** traz por padrão; no Arch puro, onde paru não está em repo nenhum, o bootstrap constrói `paru-bin` do AUR.
- **Tests**: cinco testes em `internal/usecase/provision_providers_test.go` — token de versão, comparação de versões, classificação da origem do binário e um que trava a regressão que deixou o `cmdc` sem atualizar (a origem era `"Volta"` para exibição enquanto o `switch` comparava `"volta"`).
- **Changed**: `errcheck` não acusa mais `pterm.DefaultSpinner.Start` (`.golangci.yml`, `exclude-functions`): a função só inicia uma goroutine e sempre retorna erro nil, e spinner é cosmético — nenhum subsistema deve falhar por causa dele.
- **Docs**: `docs/os-and-agent-matrix.md` ganhou a seção da fase 0, as assimetrias **#9** (canais de versão do `opencode`) e **#10** (paru no Arch) e o checklist de "CLI de agente"; manifesto Arch foi de 25 para 29 pacotes e o `doctor` de 124 para 128 checks.
- **Docs**: `docs/roadmap.md` — os objetivos acordados para depois (Termux/Android como OS, validação de compatibilidade e o que migrar para lá, skills de Tailscale/Cloudflared, verificação profunda de SSH entre os OS, provedor local controlando provedor remoto por SSH, rename do projeto e instalação como serviço de background), cada um com o contexto já levantado para a próxima sessão não recomeçar. `AGENTS.md` e `README.md` apontam para ele.

## [v1.2.142] - 2026-09-18

### 🐛 Cache do verificador não mascarava mais edição em arquivo untracked

- **Fixed**: o cache por estado da árvore (introduzido em v1.2.140) cobria `HEAD`, `git status` e `git diff HEAD` — mas o `status` lista arquivo untracked **só pelo caminho**, então editar um arquivo que ainda não foi adicionado (o estado normal enquanto se escreve algo novo) parecia "nada mudou", e o hook repetia o veredito verde anterior sem rodar nada. O hash agora inclui o **conteúdo** dos untracked, com teto de 1 MiB por arquivo para não pesar.
- **Added**: `TestVerifyScriptHookModeReRunsAfterUntrackedEdit` — quebra o build num arquivo untracked depois de um run verde e exige que o gate volte a falar (é o guarda contra essa regressão).

## [v1.2.140] - 2026-09-18

### 🪝 Todos os hooks encadeados, e o gate endurecido

- **Fixed**: `core.hooksPath` faz o git ignorar o `.git/hooks` de **todo** repositório, e até aqui só o `pre-push` era encadeado — ou seja, repositórios com `pre-commit`/`commit-msg` próprios (hook local, framework pre-commit, lefthook) tinham esses hooks **silenciosamente desativados**. Um delegator compartilhado + um shim por hook (`pre-commit`, `prepare-commit-msg`, `commit-msg`, `post-commit`, `post-checkout`, `pre-rebase`) devolvem o controle ao hook do projeto, resolvido ao lado do próprio shim para não apontar para caminho inexistente. Repositórios husky nunca foram afetados: definem `core.hooksPath` **local**, que vence o global.
- **Added**: o verificador tem testes próprios (`internal/usecase/verify_script_test.go`) que constroem repositórios descartáveis e rodam o script real — sem stack, fora de repositório, com override do projeto, escape hatch, retry do hook, dry-run, teste Go quebrado no push e o mesmo fixture no modo hook. `go test ./...` (logo, o CI) passa a cobrir o gate que bloqueia pushes.
- **Changed**: escopo por modo — `--hook` roda **só os checks estáticos** (o passe de editor) e a suíte de testes fica no gate de push, então um turno nunca espera por ela.
- **Added**: **cache por estado da árvore** no modo hook: um turno que não mudou nada sai em ~20ms em vez de repetir checks cujo veredito não mudaria (carimbo em `.git/envctl-verify.stamp`).
- **Added**: `--dry-run` imprime as stacks detectadas e os checks que seriam executados; e falhas/skips são registrados em `~/.envctl/verify.log`, o que responde "esse gate já pegou algo?" e denuncia check que vive sendo pulado.
- **Docs**: `AGENTS.md` e `README.md` passam a apontar para a matriz OS × agente e para a doc da verificação, para uma sessão nova começar pelo estado do projeto em vez de redescobri-lo.

## [v1.2.138] - 2026-09-18

### 🧪 Verify stack-aware e o tooling das stacks

- **Added**: o `envctl-verify` detecta a stack pelo repositório e roda **só as presentes** — Node/TS (`tsc --noEmit`, `eslint`, `prettier --check`), Python (`ruff check`, `mypy .`, `pytest -q`), SQL (`sqlfluff lint`), shell (`shellcheck`, `shfmt -d`), Docker (`hadolint`) e PowerShell (`Invoke-ScriptAnalyzer`, `Invoke-Pester -CI`) — além da suíte Go que já existia.
- **Changed**: duas regras de escopo para não virar ruído: **linters só nos arquivos alterados** (mesmo princípio do `--new-from-rev` do CI) e **type checks/testes no repo inteiro**. A ferramenta resolve projeto-primeiro (`node_modules/.bin`, `.venv/bin`, `uv run --no-sync`), e o que não consegue rodar naquele projeto (shim do Volta sem dep local, `uv run` sem virtualenv) vira **skip nomeado**, nunca falha.
- **Added**: tooling provisionado — `typescript` e `prettier` (volta); `pytest`, `mypy`, `sqlfluff` (uv tool); `shfmt` (apt/pacman/winget); `hadolint` (binário de release no Linux + winget); `PSScriptAnalyzer` e `Pester` como novo tipo de tweak `PSModule` no `windows.yaml` (instalação e verificação idempotentes, escopo CurrentUser).
- **Fixed**: `PipManager.IsAvailable` também considera o `uv` — o Arch não embarca `python3 -m pip`, então todo o tooling Python era silenciosamente pulado ali.
- **Fixed**: o check de shell casa por **shebang** além de extensão, porque helpers em `bin/`/`hooks/` não têm extensão — foi assim que o próprio `envctl-verify` passou a ser lintado pelo gate.
- **Docs**: `docs/verification.md` com a tabela por stack e as regras de escopo; `docs/os-and-agent-matrix.md` (matriz OS × agente, assimetrias resolvidas e checklist de adições).
- **Nota**: clients de banco (`sqlite3`, `psql`, `mysql`, `redis-cli`) foram **cancelados** por decisão — virão por skills específicas.

## [v1.2.135] - 2026-09-18

### 🔎 OpenCode ganha a mesma validação de skills do CommandCode

- **Fixed**: a auditoria do OpenCode parava em `Exists(skillsDir)` e reportava "Active and deployed" — mas uma skill com frontmatter inválido é **ignorada pelo loader em runtime**, então o doctor dava OK num estado quebrado (falso OK em 40 skills). Os dois agentes agora compartilham `auditSkillTree`: presença, parse e validação de cada `SKILL.md` (o `name` precisa bater com o diretório, `description` não pode ser vazia) e comparação da contagem com o manifesto filtrado por OS.
- **Changed**: os achados agregam em **uma linha por agente** em vez de uma por skill — a contagem de checks cai de 149 para 110 sem perder sinal (o warning nomeia as skills afetadas e o motivo).
- **Evidência**: quebrando o `name` de uma skill real → `1 of 40 deployed skills will not load: docker (frontmatter 'name' … does not match directory 'docker')`; restaurando → EXCELLENT novamente.

## [v1.2.133] - 2026-09-18

### 🔍 Config escrito pelo agente deixa de ser reportado como drift

- **Fixed**: `~/.commandcode/settings.json` divergia da fonte em **toda** execução, porque o runtime do CommandCode anexa uma entrada de permissão a cada comando aprovado (gravando linhas de comando inteiras). O warning permanente escondia achados reais e sugeria defeito onde não havia. Agora o arquivo declara `runtime_managed: true`: o provisioning continua realinhando ao template (esse é o cleanup, com backup timestamped) e a auditoria reporta OK **explicando** que o agente escreve nele em runtime. A comparação byte-a-byte permanece para todo arquivo de que o envctl é autor único — unir as entradas ao template foi descartado justamente porque o runtime grava comandos inteiros, o que só faria o arquivo crescer com lixo.
- **Nota**: `doctor` volta a **149/149 (EXCELLENT)** com zero warnings.

## [v1.2.131] - 2026-09-18

### 🧭 fzf com walker nativo e remoção de shims mortos

- **Changed**: `FZF_DEFAULT_COMMAND` saiu do manifesto. O fzf 0.47 substituiu o fallback `find` por um walker embutido (`file,follow,hidden`, pulando `.git,node_modules`) — o mesmo motor que o comando `fd` usava —, então a variável só criava diferença de comportamento entre as máquinas.
- **Added**: o bootstrap instala a release atual do fzf **somente quando** a versão instalada é anterior ao walker (o Ubuntu 24.04 traz 0.44.1; Arch e Windows já vêm com versões novas, e o pacote da distro permanece, pois é ele que fornece os bindings de shell). O `doctor` passa a auditar a **versão**, não só a presença.
- **Fixed**: os probes de versão leem apenas stdout — o `runShell` mescla stderr, e o ruído de um profile quebrado entrava no valor parseado, o que fazia o gate de versão falhar e instalar um fzf sobre outro já adequado.
- **Fixed**: linhas de rc que carregavam o shim `~/.local/bin/env` do uv — que o uv só escreve quando `~/.local/bin` **não** está no PATH, e o envctl garante esse PATH — deixavam **todo login shell** imprimindo erro no stderr. O `run shell` remove a referência morta, inclusive em `~/.bash_profile`, mas apenas quando ele já existe (criá-lo faria o bash parar de ler `~/.profile`).

## [v1.2.129] - 2026-09-18

### 🧭 Stack enxuta: Cursor padronizado, .NET/VS Code/clientes GUI fora

- **Added**: **Cursor** como editor padronizado — `Anysphere.Cursor` (winget, Windows) e `cursor-bin` (paru, `os: arch`): primeira entrada AUR fora do `gaming` e primeiro uso real do filtro por família de distro. É o que habilita `/ide` + `get_diagnostics` nas máquinas com GUI; Ubuntu Server segue sem editor.
- **Removed**: `.NET SDK 8`, `csharp-ls` (+ LSP `csharp`, entradas nos configs do opencode e o `DotnetToolManager`), Visual Studio Code (+ o template `vscode_settings`), Termius, WinSCP e GitHub Desktop — nada disso está nas stacks usadas. Junto: `"dotnet*"`/`"cargo test*"` saíram das permissões dos agentes do opencode e `Shell(cargo test:*)`/`Shell(dotnet test:*)` do `settings.json` do CommandCode.
- **Fixed**: `docker` não existia no bloco pacman (o Ubuntu já tinha `docker.io`) — agora entram `docker`, `docker-compose` e `docker-buildx`.
- **Fixed**: `golangci-lint` não era provisionado, mas é exigido pelo `envctl-verify` e pelo CI — sem ele o check de lint era pulado em silêncio numa máquina nova. Entra no bootstrap Linux (install.sh) e no winget (`GolangCI.golangci-lint`), com auditoria no `doctor`.
- **Fixed**: `csharp` era declarado no `configs/opencode.linux.json` para um LSP que só existe no Windows; `stale_pw_ps1` estava duplicado no `cleanup`.
- **Changed**: `python-pipx` sai do bloco pacman — CLIs isoladas têm um único dono (`uv tool install`).
- Contagem de LSPs: 16 → 15.

## [v1.2.127] - 2026-09-18

### ⏱️ Verificação sem timeout (fail-fast)

- **Changed**: o verificador não impõe mais timeout por check. O conjunto completo roda em ~2,3s com cache quente (gofmt 21ms, build 478ms, vet 133ms, testes 299ms, cross-compile Windows 638ms, lint 0,7s quente / 3,2s frio); esperar minutos por um check travado é desperdício em automação — ou funciona, ou não funciona e reporta. O hook de turno fica apenas com o teto do próprio engine e o pre-push roda sem teto.
- **Docs**: `docs/verification.md` ganha a seção "Sem Timeout — Fail-Fast" com a medição por check.
- **Changed**: notas já lançadas saem de `[Unreleased]` — esta leva em `v1.2.126`, a anterior em `v1.2.115`.

## [v1.2.126] - 2026-09-18

### ✅ Quality Gates locais (hook Stop do CommandCode + pre-push do git)

- **Added**: `~/.local/bin/envctl-verify` — verificador único com detecção de stack (Go, ou `.commandcode/verify.sh` do projeto como override): `gofmt -l`, `go build`, `go vet`, `go test`, `GOOS=windows go build/vet` (quebra de plataforma aparece localmente, não num runner) e `golangci-lint --new-from-rev` com o mesmo gate de "somente findings novos" do CI — dívida legada nunca bloqueia, finding novo sempre bloqueia.
- **Added**: hook `Stop` do CommandCode (`~/.commandcode/settings.json`) rodando o verificador no fim de cada turno: em falha, `exit 2` devolve o stderr com o diagnóstico ao modelo, que corrige no mesmo turno. Anti-loop via `stop_hook_active`; escotilha `ENVCTL_SKIP_VERIFY=1`.
- **Added**: pre-push global do git (`core.hooksPath` → `~/.config/git/hooks/pre-push`): nenhum push sai sem verificação, de qualquer agente ou do terminal. Como `core.hooksPath` sobrepõe `.git/hooks` de todos os repositórios, o hook deployado **encadeia primeiro o pre-push local do repositório** (husky e afins seguem funcionando).
- **Added**: auditoria `Verify` no `doctor` (script e hook presentes e executáveis) — drift do wiring vira `WARN`.
- **Docs**: `docs/verification.md` (camadas, checks, encadeamento de hooks e variáveis de controle), link no README e no `doctor-and-idempotency.md`.

### 🧹 Higiene do store do OpenCode (`run cleanup`)

- **Added**: leitura do header SQLite (`page size`/`page count`/`freelist`) em Go puro — sem dependência nova — com `VACUUM` **somente quando há páginas livres**: dado vivo nunca é reescrito por trás do usuário. Store travado por um opencode em execução é reportado, não repetido. O `doctor` reusa o mesmo helper, então threshold, caminho (respeitando `XDG_DATA_HOME`) e medição não divergem.
- **Fixed**: `opencode.db` de 946 MB → 134 MB no host de referência (811 MB eram páginas livres), com `integrity_check: ok` e as 19 sessões preservadas.

### 🛡️ Config não-destrutivo e poda com quarentena

- **Added**: `merge:` em `ConfigFile` com dois modos — `ssh_hosts` preserva as stanzas `Host` do usuário que o template não define (inseridas **antes** do `Host *`, para manter a precedência do OpenSSH) e `json_deps` faz união de `dependencies`/`devDependencies` mantendo o template autoritativo. Ambos idempotentes; arquivo ilegível é preservado com `WARN` em vez de sobrescrito.
- **Fixed**: `~/.ssh/config` e `~/package.json` deixam de ser sobrescritos pelo template — o primeiro apagava hosts do usuário silenciosamente.
- **Changed**: skills obsoletas vão para `<dir do agente>/.envctl-trash/skills/<nome>-<timestamp>` em vez de `os.RemoveAll`, permanecendo recuperáveis.
- **Fixed**: arquivo recém-criado era rotulado "Already up to date"; agora `Created` / `Updated` / `Already up to date`.

### 🐧 Arch/CachyOS (distro-aware) e fish

- **Added**: detecção de distribuição via `/etc/os-release` (`entity.DetectedDistro`, cacheada com `sync.Once`) e `entity.MatchOS`, que resolve tanto os nomes de plataforma (`windows`, `linux`, `darwin`) quanto as famílias de distro (`arch`, `cachyos`, `debian`, `ubuntu`, incluindo listas `arch,cachyos`). Todos os filtros `os:` passam a usá-lo (`provision_packages`, `provision_lsp`, `provision_shell`, `provision_skills`, `doctor_audit`, `snapshot_sync`), então `os: arch` deixa de ser ignorado em silêncio.
- **Added**: fish como alvo de primeira classe na persistência de ambiente (`set -gx` em `~/.config/fish/config.fish`, além de `~/.profile` e `~/.bashrc`) — em hosts cujo shell de login é fish (CachyOS, Arch) as variáveis (`ENVCTL_TEMP`, PATH) não chegavam ao shell interativo. `EnsureEnvVars` agora exige a declaração em **todos** os arquivos de shell, não em um só, e `withinHome` impede escrita fora do HOME.
- **Fixed**: 8 entradas do bloco `pacman` apontavam para pacotes inexistentes (`fd-pacman`, `fzf-pacman`, `bat-pacman`, `ripgrep-pacman`, `tree-pacman`, `shellcheck-pacman`, `python-requests-pacman`, `python-openpyxl-pacman`), fazendo `pacman -S` falhar em qualquer host sem a ferramenta; nomes reais confirmados com `pacman -Si`.
- **Fixed**: `npm install -g` fixa o prefixo em `~/.local` (o npm da distro escreveria em `/usr/lib/node_modules`, exigindo root), o pip prefere `uv tool install` e só usa `--break-system-packages` quando o interpretador se declara PEP 668 "externally managed", e o fallback do `fd` usa pacman ou apt conforme a distro.

### 🗑️ Ferramentas fora do stack

- **Removed**: rustup/cargo/rust-analyzer, intelephense (PHP LSP) e Oh-My-Posh saíram de ponta a ponta (bootstrap, `RustupManager`, PATH do cargo, `lsp.yaml`, winget, tema e `~/.poshthemes`, init do perfil PowerShell e o tweak de Nerd Font que passava por `oh-my-posh font install`). Máquinas que já rodaram o bootstrap antigo convergem via novas entradas de `cleanup`. LSPs: 18 → 16.
- **Changed**: Windows Terminal aponta para `Cascadia Mono`, fonte que acompanha o Windows Terminal.

### 🔧 Lint do gate de CI

- **Fixed**: 7 findings introduzidos pelas mudanças acima (errcheck, G301, G703, G204) — erro de `ExpandUserPath` tratado, diretórios criados com 0750 e supressões inline justificadas nos dois `exec` cujo argv nunca passa por shell.

## [v1.2.115] - 2026-09-17

### 🧪 3 skills novas: TDD, docs-sync, variant-analysis (38 → 41)

- **Added**: **`test-driven-development`** — ciclo red-green-refactor em TS/Node (vitest), Python (pytest) e Go (`go test`): teste falhando primeiro, código mínimo depois. Ideia de obra/superpowers (MIT), ciclo e exemplos escritos do zero para as 3 linguagens, com wiring para `universal-test-runner` (execução/cobertura) + `verification-before-completion` (gate final). Uma skill única, não três especializadas: o núcleo (lei de ferro, racionalizações, red flags, checklist) é idêntico e só os comandos de ciclo mudam.
- **Added**: **`docs-sync`** — auditoria/atualização da documentação contra a implementação real (doc-first/code-first, missing/incorrect/structural), com Apêndice A de docstrings TS (TSDoc) / PY (Google style) / GO (godoc). Ideia de openai/openai-agents-python (MIT), workflow escrito do zero para a estrutura real (README/CHANGELOG/docs/manifests/configs), audit-only por default (reporta, não edita sem autorização).
- **Added**: **`variant-analysis`** — original, sem upstream (licença do trailofbits é CC-BY-SA, incompatível): caça às outras instâncias de um bug já encontrado em 5 passos (causa raiz → match exato com `rg`/`fd` → generalizar um elemento por vez → triage com severidade → relatório). Entrada típica: Fase 4 do `systematic-debugging`; saída: correções via TDD + gate de verificação.
- **Changed**: contagens normalizadas (38→41) em `README.md`, `AGENTS.md`, índices, `docs/skills.md`, attribution e documentos afetados; teste de manifesto (`expectedSkills`) acompanha.

### 🌐 Browser em dois trilhos (chrome-devtools MCP + playwright-cli)

- **Removed**: MCP `@playwright/mcp` dos 3 configs (opencode.json, opencode.linux.json, commandcode mcp.json) — duplicava o chrome-devtools no interativo e é token-heavy frente ao CLI nos fluxos repetíveis.
- **Added**: `pw` — wrapper node versionado (`configs/bin/pw.cjs` + shims `pw.cmd`/`pw`, provisionado em `~/.local/bin` via `shell.yaml`) que elimina o hang do Windows: spawn DETACHED + unref ele mesmo e espera pela sessão via `list` (matriz empírica: direto/execFileSync/attached travam, detached+unref retorna em ~2s, inclusive dentro de `opencode run`); `doctor` audita o wrapper (`Browser/pw wrapper`) e o bundle Chromium em ambos OS (Windows: `%LOCALAPPDATA%\ms-playwright`).
- **Changed**: skills `web-dashboard-automation` e `playwright-prod-regression` orientam a escolha (MCP interativo vs `pw` determinístico); AGENTS.md, SKILL-INDEX.md, README, guides, `docs/skills.md` e memória refletem os dois trilhos.

### 🩺 Doctor por-OS + browser headless no Linux

- **Fixed**: `doctor` não avisa mais de pacotes cujo gerenciador não existe na máquina (ex.: 22 warnings `pacman` em Ubuntu/Debian) — entradas de gerenciador ausente são puladas silenciosamente; volta/npm/apt/winget universais seguem auditados normalmente.
- **Added**: `doctor` valida referências `{file:...}` do `opencode.json` (ERROR se ausente — antes um `context7.key` faltante quebrava todo `opencode` com doctor verde).
- **Changed**: context7 remoto sem header de key (free quota funciona — verificado live: initialize + tools/list + resolve-library-id sem auth).

### 🧹 Consolidação do catálogo (41 → 38 skills) + 2 skills novas

- **Removed / merged** (redundâncias eliminadas): `dispatching-parallel-agents` + `parallel-agent-orchestration` foram absorvidas por `subagent-routing` (agora cobre roteamento, dispatch paralelo e orquestração no mesmo repo numa skill única de ~60 ln); `docker-build-local-vps-deploy` + `docker-desktop-wsl-restart` foram absorvidas por `docker` (agora cobre local/VPS/build-transport/restart WSL2 em ~50 ln); `using-git-worktrees` foi dobrado dentro de `git-workflow` (worktree é workflow git).
- **Added**: **`task-hang-watchdog`** — previne e recupera terminais travados e tarefas autônomas presas (background, timeout, classificar read-only vs interativos, matar processos hungidos via `kill_shell`, `monitor_command` para saídas longas); **`subagent-supervision`** — o coordenador vigia subagentes paralelos e, se um alucina/loopa/trava, mata via `agent_output(action: "kill")` e decide retry refinado ou escala ao usuário (sem retry infinito).
- **Trim**: `systematic-debugging` (278→~80 ln) e `verification-before-completion` (115→~50 ln) — mantidas apenas as mecânicas essenciais.
- **Changed**: contagens normalizadas (41→38) em `README.md`, `docs/skills.md` e documentos afetados; catálogo reescrito sem redundâncias.

### © Créditos de terceiros, caminhos próprios e `grill-me`

- **Added**: **atribuição de autoria** nas skills adotadas de terceiros. Cada uma agora traz `metadata.author`, `metadata.source` e `metadata.adapted`, com o repositório de origem: [obra/superpowers](https://github.com/obra/superpowers) (MIT) em `dispatching-parallel-agents`, `receiving-code-review`, `systematic-debugging`, `using-git-worktrees`, `verification-before-completion`, `writing-plans`; [mattpocock/skills](https://github.com/mattpocock/skills) (MIT) em `grill-me`, `grilling`, `handoff`; [hqhq1025/skill-optimizer](https://github.com/hqhq1025/skill-optimizer) (MIT) em `skill-miner`, `skill-personalizer`, `skill-generalizer`. `stop-slop` já creditava Hardik Pandya (hvpandya.com).
- **Added**: `LICENSE` (MIT) na raiz e `license: MIT` nas 41 skills — antes só 13 declaravam licença e o repo não tinha arquivo de licença, o que deixava as outras 28 implicitamente "todos os direitos reservados".
- **Added**: `docs/skills-attribution.md` — tabela de atribuição, o que foi adaptado em cada cópia, a pendência declarada (`ask-questions-if-underspecified` e `context7-auto` têm origem não confirmada) e o checklist para adotar skill de terceiro (verificar licença via `gh api`, preencher metadata, registrar na tabela).
- **Changed**: `memory-promotion` ganhou a regra de atribuição obrigatória ao criar/adotar skill; `docs/skills.md` e `README.md` linkam a tabela.
- **Fixed**: caminhos da skill upstream dentro do nosso projeto — o agente `plan` e `writing-plans` salvavam em `docs/superpowers/plans/` (pasta do projeto de origem); agora os planos vão para **`spec-agent/`** na raiz, e a permissão de escrita do agente `plan` acompanha (`opencode.json`, `opencode.linux.json`, `REFERENCE.md`, `writing-plans`).
- **Added**: **`grill-me`** reescrita como **porteiro de ambiguidade**: rubrica de 0 a 100 (alvo, ação, critério de pronto, escopo, restrições, referentes soltos), limiares (≤20 executa, 21–50 pergunta só os bloqueadores, ≥51 não executa) e limiar de risco — operação irreversível com ambiguidade > 0 bloqueia. As quatro variantes de AGENTS.md ganharam uma regra explícita: ambiguidade alta → perguntar antes de agir. Fronteira documentada com `ask-questions-if-underspecified` (mecânica de perguntar) e `grilling` (arguição de plano existente).
- **Changed**: `docs/skills-attribution.md` reenquadrado — a mensagem central agora é que **não há obrigação de fidelidade 1:1 com nenhum upstream**; toda skill do repositório é sua para adaptar ao seu estilo. O crédito ao upstream (no `metadata`) já cumpre o papel de reconhecer a ideia original. A tabela foi reescrita com "Como foi adaptado (medido)" em vez de categorizar como cópia.
- **Changed**: catálogo 40 → **41 skills** (Windows deploya 38, Linux 39).
- **Fixed**: o `metadata.adapted` de cada skill derivada foi corrigido para descrever **o que de fato foi feito**, medido contra o upstream em vez de estimado — 8 têm corpo **idêntico** ao upstream (só a descrição/triggers em PT-BR mudou), 2 têm uma edição pontual (`writing-plans` e `handoff`), `grilling` foi **ampliado** (+3 parágrafos) e `grill-me` foi **reescrito**. Importa para o crédito ser honesto: a rubrica de ambiguidade 0–100, os limiares e a cláusula de risco da `grill-me` são **originais daqui** — o upstream são 2 linhas que só mandam invocar `/grilling`.

### 🧩 OpenCode: AGENTS.md compacto + REFERENCE.md sob demanda

- **Changed**: `configs/AGENTS.md` (OpenCode Windows) reescrito compacto — **20,1 KB → 4,4 KB** (variante Linux: 13,4 KB → 4,6 KB), na mesma estrutura do CommandCode: **Ambiente · Regras · OpenCode · Referências (leia só se precisar)**.
- **Added**: `configs/REFERENCE.md` → `~/.config/opencode/REFERENCE.md` (categoria `opencode`): plugins, DCP (bandas, `compress`, protegidos), agentes (`review`/`plan`), memória (paths + promoção a skill), infra VPS, snippets (serviços, SSH, Docker, privilegiado) e higiene de scratch/encoding. **Consultado sob demanda — não é auto-carregado.**
- **Changed**: a diretiva "Agent Memory (OBRIGATÓRIO — ativo em TODA tarefa)" virou uma regra enxuta por demanda ("consulte `agent-memory` quando a tarefa parecer repetir algo já resolvido; registre quando aprender"). O mandato de carregar antecipadamente contrariava a economia de contexto; a mecânica completa (paths, classificação, promoção a skill) foi para o `REFERENCE.md`.

### ⚡ AGENTS.md compacto, skills sob demanda e provisionamento por agente

- **Changed (preferência corrigida)**: as skills voltam a ser **carregadas sob demanda**. Removido o mandato "carregue a skill ANTES de agir" dos 4 manifestos: a economia de contexto é justamente o motivo de usar skill em vez de MCP — o que entra no prompt a cada turno é apenas *nome + descrição*, e o corpo do `SKILL.md` só é lido quando a tarefa casa.
- **Added**: **`envctl commandcode`** e **`envctl opencode`** — cada agente tem seu próprio comando e provisiona **só o que é dele**. `ProvisionShellUseCase.Execute(ctx, categories...)` filtra diretórios, arquivos de config e itens de cleanup por `category` (campo novo também em `RestrictedDir`), e as etapas machine-level (variáveis de ambiente, git config, npm do `~/package.json`) são puladas quando há filtro. Verificado nos dois sentidos: `envctl commandcode` não altera nenhum arquivo do opencode, e vice-versa.
- **Added**: **`SKILL-INDEX.md` por agente** (`configs/commandcode/SKILL-INDEX.md` → `~/.commandcode/SKILL-INDEX.md`; `configs/SKILL-INDEX.md` → `~/.config/opencode/SKILL-INDEX.md`). A tabela *situação → skill* saiu do AGENTS.md e virou um arquivo **consultado sob demanda** — o agente só o abre quando precisa escolher entre skills.
- **Changed**: `configs/commandcode/AGENTS.md` reescrito compacto — **12,3 KB → 3,7 KB** (a variante Linux: 11,3 KB → 4,2 KB), agora com: Ambiente, Regras, uma seção curta do CommandCode (config, agentes, MCP, skills on-demand, taste, hot reload) e **"Referências (leia só se precisar)"** apontando os arquivos externos. Os manifestos do opencode perderam o bloco/tabela de skills (−3,7 KB cada).
- **Changed**: `docs/skills.md` documenta a carga sob demanda, o índice externo e o deploy por agente; `README.md` ganhou os dois comandos novos; `docs/manifests.md` idem.

### 🎯 Skills auto-invocáveis, escopo por ambiente e AGENTS.md como roteador

- **Fixed**: `handoff` carregava `disable-model-invocation: true` — era a **única skill impossível de ser auto-invocada** (nunca entra no catálogo do modelo). Flag removida e a descrição agora diz quando usar.
- **Changed**: 12 descrições reescritas no padrão "Use quando… Triggers: …" com gatilhos em PT-BR (o usuário escreve em português): `ask-questions-if-underspecified`, `writing-plans`, `systematic-debugging`, `dispatching-parallel-agents`, `grilling`, `stop-slop`, `using-git-worktrees`, `verification-before-completion`, `receiving-code-review`, `skill-miner`, `skill-personalizer`, `skill-generalizer`. Antes eram frases curtas em inglês (72–252 chars) sem vocabulário de gatilho, justamente as que dirigem o matching automático. Hoje **40/40** skills têm gatilho explícito; o catálogo sobe de ~3.0k para ~3.9k tokens.
- **Added**: **escopo por ambiente** nas skills — campo `os` no `Skill` (manifesto), `Skill.AppliesToOS()` e filtro no `ProvisionSkillsUseCase`. `windows-admin` e `docker-desktop-wsl-restart` só vão para Windows; `cachyos-gaming-setup`, `aur-headless-install` e `headless-gui-probe` só para Linux. Uma instalação Windows deploya **37** skills (antes 40) e as 3 linux-only são **podadas** (deixam de existir no catálogo) — sem skill irrelevante consumindo contexto nem disparando na máquina errada.
- **Fixed**: o check por skill do `doctor` exigia o manifesto inteiro mesmo no OS errado (e incluía skills desabilitadas) — agora aplica o mesmo filtro de ambiente/habilitação. `doctor` volta a **152/152, 0 warnings** após a mudança.
- **Changed**: `configs/skills/docker` deixou de ser Windows-cêntrico no corpo (a skill é deployada também em VPS Linux): agora detecta o ambiente — Docker Desktop/WSL2 no Windows, daemon systemd no Linux — antes de agir.
- **Changed**: os 4 manifestos (`configs/AGENTS.md`, `configs/AGENTS.linux.md`, `configs/commandcode/AGENTS.md`, `configs/commandcode/AGENTS.linux.md`) trocaram o **catálogo por nome** por uma **tabela situação → skill** com mandato explícito ("as skills são o método padrão; carregue ANTES de agir; não pergunte 'quer que eu use a skill X?'"). Uma lista de nomes não dá ao modelo nada para casar com a tarefa; a tabela roteia por intenção, que é o que efetivamente dispara a invocação automática. Bloco idêntico nos 4 arquivos (mesma padronização cross-agente/ambiente), com marcadores `[win]`/`[linux]` para as skills escopadas. (**Revertido logo em seguida**: o mandato proativo contraria a economia de contexto — a tabela foi movida para `SKILL-INDEX.md`, consultado sob demanda; ver a entrada acima.)

### 🧹 Poda de skills estrangeiras/obsoletas + paridade de plataforma no CommandCode

- **Removed**: 3 skills — `implementation-strategy` e `docs-sync` (conteúdo do projeto **openai-agents-python**: `mkdocs.yml`, `docs/ja|ko|zh`, `src/agents/`, `$openai-knowledge`/OpenAI Docs MCP, e um script `find_latest_release_tag.sh` que não existe neste repo) e `grill-me` (corpo de 2 linhas que só re-invocava `grilling`, via nome de tool do opencode). Catálogo: 43 → **40 skills provisionadas** (+1 built-in do opencode).
- **Fixed**: skills com referências mortas — `docker` (seção do MCP `docker-hub`, removido do repo há versões, virou uma nota de que o MCP não existe), `vps-agent-dispatch` (`npm install -g @opencode-ai/cli` → `opencode-ai`, pacote correto; scratch `/tmp` → `/temp`), `memory-promotion` (banda DCP obsoleta 85/75 e menção a DCP), `handoff` e `lsp-smoke-test` (nome de tool/config parametrizados por agente).
- **Fixed**: seeds de memória — removida a entrada DCP 85/75 (superada pela 90/80), corrigido o valor no registro de plugins, removido o módulo Node `playwright` da lista de libs globais (browser automation é MCP-only), numeração do `doctor` corrigida (12.6/12.6.1 → 11.6/11.6.1) e movidas para fora do seed global **11 entradas de DOMÍNIO** (projeto/fornecedor), que por regra pertencem à memória do projeto. As cópias por máquina não são afetadas (`seed_if_missing`).
- **Added**: `configs/commandcode/AGENTS.linux.md` + filtro `os:` em `manifests/shell.yaml` — a VPS Linux recebia o manifesto **Windows** do CommandCode (PowerShell 7.6.5, `C:\temp`), enquanto o opencode já tinha variante Linux.
- **Changed**: `docs/skills.md` reescrito como catálogo completo das 40 skills (antes listava ~20) + tabela de paridade OpenCode ↔ CommandCode (skills, memória, LSP, context pruning, regras globais).
- **Changed**: `manifests/packages.yaml` — adicionadas `pypdf`, `python-docx` e `lxml` ao provisionamento pip do Windows; os docs (`AGENTS.md` e memória) já as listavam como disponíveis.
- **Changed**: `configs/AGENTS.md` — removidas as linhas das 3 skills e contagem ajustada (41 ativas = 40 provisionadas + 1 built-in).
- **Changed**: removido o número hardcoded de verificações do `doctor` dos docs ("160+") — a contagem é dinâmica (155 nesta máquina Windows, menor no Linux), então os docs passam a descrever o escopo em vez de um número que não se sustenta.
- **Changed**: o teste do manifesto agora exige que `manifests/skills.yaml` e os diretórios embutidos de `configs/skills/` batam exatamente (skill embutida e não declarada nunca é deployada; declarada sem diretório quebra o provisionamento).

### 🧩 Paridade CommandCode: config, metadata e validação no doctor

- **Fixed**: `configs/commandcode/settings.json` — removido o `$schema` (`https://commandcode.ai/schema/settings.json` responde **404** e não é chave do registry de settings); removidas as entradas de permissão mortas herdadas do opencode (`todowrite`, `todoread`, `task`, `skill` — no CommandCode as tools são `todo_write`, `task_create/update/list/get/stop` e `activate_skill`) e as no-op (`Read`, `WebFetch`, `WebSearch`, `Shell(go test:*)`, `Shell(go vet:*)`); adicionadas regras `deny` (`Shell(git push --force*)`, `Shell(git reset --hard*)`, `Shell(git clean -*)`, `Edit(.git/**)`) e `ask` para segredos (`Read(.env*)`, `Read(~/.commandcode/auth.json)`).
- **Fixed**: `configs/commandcode/AGENTS.md` — a afirmação "Config is NOT hot-reloaded: restart CommandCode after changes" era falsa (agents, skills e memória são re-lidos a cada turno; `settings.json` vale no próximo round; só um update staged exige `/reload`); catálogo de skills corrigido — faltavam `aur-headless-install`, `cachyos-gaming-setup` e `headless-gui-probe` (a contagem final do ciclo está na entrada de poda acima); adicionados `.agents/skills/` (compat), escopo local de MCP, precedência de MCP e o conjunto completo de campos do frontmatter de agente.
- **Fixed**: agente `code-reviewer` — removido o `maxTurns: 40` (o default documentado do CommandCode é 100; o cap interrompia revisões longas).
- **Fixed**: `manifests/shell.yaml` — novo cleanup `stale_commandcode_memory_dir` remove `~/.commandcode/memory` (sobra do provisionamento antigo; o CommandCode não tem memory-dir); descrição do `commandcode_mcp` atualizada para os 5 servidores reais.
- **Fixed**: skills que referenciam caminhos exclusivos do opencode (`~/.config/opencode/memory/`, `.opencode/memory/`, `opencode.json`) foram neutralizadas/parametrizadas por agente — `agent-memory`, `memory-promotion`, `ssh-vps`, `docker`, `lsp-smoke-test`, `windows-admin`, `subagent-routing` (corrigido também `plan`/`goal` como agentes do CommandCode — `goal` não existe) e `skill-generalizer/references/platform-compatibility.md` (nova linha do Command Code).
- **Changed**: removido `compatibility: opencode` de 13 `SKILL.md` — no CommandCode o campo significa **requisitos de ambiente**, então o valor era metadado enganoso (e a rubrica de portabilidade do projeto pede frontmatter conservador).
- **Changed**: `manifests/skills.yaml` — as 25 descrições placeholder `"OpenCode agent skill <name>"` viraram descrições reais; `snapshot_sync.go` passa a derivar a descrição do frontmatter do `SKILL.md` em vez de gerar o placeholder.
- **Added**: `internal/usecase/skill_frontmatter.go` — parser/validador de frontmatter (Agent Skills) compartilhado; o `doctor` agora valida o JSON de `settings.json`/`mcp.json`, o frontmatter de cada agente custom (nome == arquivo, nomes reservados ignorados) e o das skills implantadas (nome == diretório, descrição não vazia), além de comparar a contagem implantada com o manifesto.
- **Changed**: contagens normalizadas em `README.md` e `docs/principles.md`.
- **Changed**: `.gitignore` — o escopo de projeto do CommandCode passa a versionar os artefatos autorais (`skills/`, `commands/`, `agents/`); `settings.json`, `settings.local.json` e `taste/` seguem locais por máquina.

### 🔥 Trim do firecrawl + browser automation MCP-only (43 skills)

- **Removed**: suite firecrawl completa — 5 skills (`firecrawl`, `firecrawl-crawl`, `firecrawl-map`, `firecrawl-scrape`, `firecrawl-search`), pacote volta `firecrawl-cli`, step de bootstrap Linux e check do `doctor`.
- **Removed**: via CLI de browser — skill `playwright-cli`, pacote volta `@playwright/cli`, scripts `pw-screenshot`/`pw-eval` (+ wrappers, entradas no `shell.yaml`, auditorias `CLI-Scripts`/`Chromium`/`Playwright` no `doctor`, bloco de instalação do Chromium no `run shell`, bloco `references` no `opencode.json`/`opencode.linux.json`, dep `playwright` no `user-package.json`). Browser automation é exclusivamente via MCPs `playwright` + `chrome-devtools` (com o bundle do Chrome deles).
- **Changed**: contagens sincronizadas (43 provisionadas + 1 built-in) em `AGENTS.md`, `README.md`, `docs/skills.md`, `docs/principles.md`, `docs/doctor-and-idempotency.md` e teste de manifesto.
- **Changed**: skill `git-workflow` — em divergência com `origin`, o upstream prevalece (rebase + resolver a favor do remoto; nunca force-push de intent local).

### 🔗 Referências do repo apontam para o remoto público

- **Fixed**: `configs/AGENTS.md`, `configs/commandcode/AGENTS.md` e `configs/skills/vps-provisioning/SKILL.md` referenciavam o checkout local (`C:\projetos\git-privado\envctl`); a fonte da verdade agora é o remoto https://github.com/eajdias/envctl — as cópias locais são gerenciadas e edição direta nelas é sobrescrita pelo `envctl run shell`.
- **Changed**: `.gitignore` passa a ignorar `.commandcode/` (scratch local do agente CommandCode — permissões de sessão e paths absolutos da máquina, não versionável).

### 🧭 Agentes opencode: `plan` volta ao default (built-in) e agente `goal` removido

- **Changed**: agente `plan` — removido o `"mode": "all"` de `opencode.json`/`opencode.linux.json`; volta ao default do opencode (**primary built-in**, não dispatchável via task tool). Prompt ajustado.
- **Removed**: agente `goal` (autônomo YOLO) de `opencode.json`/`opencode.linux.json` e `configs/commandcode/agents/goal.md` (+ entrada `commandcode_agent_goal` em `manifests/shell.yaml`) — o agente `build` cobre o fluxo. O plugin `@prevalentware/opencode-goal-plugin` **permanece** (o `/goal` funciona a partir de qualquer agente, inclusive `build`).

### 🧭 Dispatch de subagentes: diretiva proativa + skill `subagent-routing`

- **Added**: skill `subagent-routing` — roteamento de subagentes (quando delegar, `explore` vs `general`, paralelo vs sequencial, quando NÃO delegar).
- **Changed**: `configs/AGENTS.md`, `configs/AGENTS.linux.md` e `configs/commandcode/AGENTS.md` ganharam as seções **"Uso Proativo de SSH, Context7 e Busca Web (OBRIGATÓRIO)"** e **"Delegação a Subagentes (uso proativo)"**, e o `Tone`/`Zero Tolerância` passam a exigir sinalizar defaults subótimos com trade-offs e fechar tarefas com evidência fresca.
- **Changed**: prompts dos agentes `review`/`plan` (`opencode.json`/`opencode.linux.json`) reforçados para dispatch proativo de `explore`/`general`, paralelizando domínios independentes na mesma resposta.

### 🧠 DCP (context pruning) priorizando cache-hit + compressão manual

- **Changed**: `configs/dcp.jsonc` — `experimental.allowSubAgents: true`; banda de compressão automática **90%/80%** e `nudgeFrequency: 10` / `iterationNudgeThreshold: 30` (menos compressões e menos injeções de nudge = **cache-hit melhor**); `manualMode` segue **desligado** (ligá-lo desativa a compressão autônoma).
- **Changed**: `configs/AGENTS.md` e `configs/AGENTS.linux.md` — diretiva para chamar a tool `compress` **proativamente** (troca brusca de assunto / fim de sub-tarefa).

### 🔌 Paridade opencode ↔ CommandCode (agentes + MCP + memory)

- **Fixed**: agentes custom do CommandCode — `review.md`/`plan.md` usavam **nomes reservados** (`explore`/`plan`/`review`/`general`) e eram **ignorados** pelo CommandCode; o frontmatter ainda usava tool ids inválidos (`read`/`webfetch`/`websearch`). Substituídos por **`code-reviewer.md`** (nome válido; tools `read_file`, `read_directory`, `grep`, `glob`, `web_search`, `web_fetch`, `shell_command`) — read-only, evidência `file:line`, severidades + veredito. `plan.md` removido (o built-in **Plan** do CommandCode cobre). O cleanup passa a remover os `.md` stale (`review`/`plan`/`goal`).
- **Removed**: diretório vestigial `~/.commandcode/memory` do `shell.yaml` (a memória do CommandCode é o `AGENTS.md`).
- **Changed**: `configs/commandcode/mcp.json` — adicionados `ssh-manager` e `zscan` (`enabled: false`) para paridade com o opencode; `configs/commandcode/AGENTS.md` documenta built-ins/nomes reservados, memory = `AGENTS.md` e ganha o catálogo de skills.

### 🧹 Poda de skills fora do manifesto + bun no bootstrap Linux

- **Added**: `ProvisionSkillsUseCase.Execute` agora **poda** diretórios de skill que saíram do manifesto (guarda contra manifesto vazio; ignora dot-dirs; retorna os nomes removidos, exibidos pelo CLI). Coberto por `provision_skills_test.go`.
- **Added**: `bun`/`bunx` no bootstrap Linux (`provision_bootstrap.go`) + auditoria no `doctor` — necessário para os MCPs `bunx` nas VPSs.
- **Changed**: MCPs de browser (`playwright`, `chrome-devtools`) padronizados em `bunx` com versão pinada e `enabled: false` nos 3 configs; `docs/`, `README.md` e a memória global (`configs/memory/*.md`) sincronizados (7 lições novas de ambiente).

---

## [v1.2.0] - 2026-09-06

### 🆕 CommandCode: provisionamento equivalente ao OpenCode

- **Added**: Suporte completo a CommandCode — o `envctl` agora provisiona **OpenCode E CommandCode** simultaneamente, com mesmas skills, MCPs equivalentes e configuração de agentes traduzida.
- **Added**: Config templates em `configs/commandcode/` — `settings.json`, `AGENTS.md`, `mcp.json`, e agentes (`review.md`, `plan.md`, `goal.md`).
- **Added**: Skills são deployadas para **ambos** `~/.config/opencode/skills/` e `~/.commandcode/skills/` (mesmo formato SKILL.md).
- **Added**: Entradas em `manifests/shell.yaml` — config files, diretórios e cleanup entries para CommandCode.
- **Added**: Instalação do CommandCode CLI no bootstrap Linux (`provision_bootstrap.go`).
- **Added**: Health checks do `doctor` para CommandCode.
- **Changed**: `configs/commandcode/AGENTS.md` — shell atualizado para PowerShell 7.6.5 (`pwsh.exe`) com regras nativas PowerShell.
- **Changed**: `configs/skills/firecrawl-monitor/SKILL.md` — descrição simplificada (remoção de formatação e termos redundantes).
- **Motivo**: CommandCode é um agente LLM alternativo ao OpenCode; o envctl agora provisiona ambos, permitindo ao usuário escolher qual usar.

---

## [v1.1.49] - 2026-09-06

### ⚡ Provisionamento CommandCode + correções menores

- **Added**: Provisionamento inicial do CommandCode (configs, agents, MCP) — precede a release v1.2.0 que consolida o suporte.
- **Fixed**: `configs/commandcode/AGENTS.md` — shell corrigido de `cmd.exe` para PowerShell 7.6.5.
- **Changed**: `configs/skills/firecrawl-monitor/SKILL.md` — descrição simplificada (remoção de formatação e termos redundantes).
- **Motivo**: relato de caracteres `�` no terminal/console do opencode e ambiente Windows; fix anterior (profile + chcp) não alcança processos spawnados sem profile.

---

## [v1.1.47] - 2026-09-03

### ⚡ Agentes sem "tool negada" + correções da auditoria (ARM64, snapshot, doctor)

- **Changed**: `configs/opencode.json`/`opencode.linux.json` — agentes `plan`/`review`:
  - `bash`: `"*": "deny"` → `"*": "ask"` (read-only auto; demais comandos — firecrawl CLI, node, python, docker — pedem aprovação em vez de erro duro "tool negada").
  - `task: "allow"` (dispatch de explore/general) + `"subagent_depth": 2` global (subagent pode despachar sub-subagentes).
  - Removido `permission.skill.firecrawl-*: deny` global e `references.envctl` (path local versionado).
- **Fixed** `bootstrap.sh:83`: `${AUTH_HEADER[@]}` vazio + `set -u` quebrava no bash 3.2 (macOS) → `${AUTH_HEADER[@]+"${AUTH_HEADER[@]}"}`.
- **Fixed** `provision_bootstrap.go`: installers de gh/delta/yq/Go agora detectam `uname -m` (amd64/arm64) — VPS ARM64 suportada; versão do Node derivada do `packages.yaml` (sem drift).
- **Fixed** `snapshot_sync.go`: git.yaml preserva keys curadas ausentes na máquina; configs só reescritos quando o conteúdo difere; symlinks de skills seguem o alvo; erros de escrita logados.
- **Fixed** `doctor_audit.go`: check de git worktree detecta git ausente (sem falso DiagOK); config files auditam conteúdo (drift vs fonte provisionada), exceto seeds locais.
- **Fixed** `run cleanup` agora executa o `TempHygieneUseCase` (dead code eliminado; FixHint corrigido); `envctl run <desconhecido>` sai com exit 1.
- **Fixed** `tweaks_manager.go`: comparação numérica robusta de DWORD (YAML int/float vs registry); `packages.yaml`: check_command pip usa `py -m pip show` (evita o stub do Microsoft Store).
- **Changed** `manifests/shell.yaml`: dir `~/.config/opencode/secrets` (strict_acl, context7.key por máquina — nunca versionado). `configs/AGENTS.md`: catálogo de skills corrigido (74).
- **Motivo**: falha reportada no modo Review ("task general está negado... não tenho bash para o firecrawl CLI") + auditoria do projeto contra docs oficiais do opencode.

---

## [v1.1.46] - 2026-09-03

### 🐛 Fix: pipes bloqueados no bash read-only dos agentes `plan`/`review`

- **Fixed**: `configs/opencode.json` e `configs/opencode.linux.json` — allow list bash dos agentes `plan`/`review` ampliada. O opencode valida **cada segmento** de pipe/`&&` separadamente e o pattern casa com o resource = prefixo de arity do comando (`permission/arity.ts`): `rg x | head -5` exigia `head*` na lista; `dotnet test*` nunca casava porque `dotnet` não está no dict de arity (resource = `dotnet`). Adicionados: `git grep*`, `grep*`, `head*`, `tail*`, `wc*`, `sort*`, `uniq*`, `awk*`, `sed*`, `dotnet*` (substitui `dotnet test*`) e filtros PowerShell `Select-Object*`, `Where-Object*`, `Select-String*`, `Measure-Object*`, `Sort-Object*`, `Group-Object*`, `Out-String*`, `Format-Table*`. Prompts atualizados (pipes/`&&` permitidos desde que cada segmento seja read-only).
- **Motivo**: falha reportada no modo Review — "O pipe não é permitido. Vou usar rg puro."

---

## [v1.1.45] - 2026-09-03

### 🐛 Fix: agente `review` anulado por `review.md` stale + seeds de memória sincronizados

- **Fixed**: `~/.config/opencode/agents/review.md` (criado em 2026-08-24, pré-v1.1.39) **sobrescrevia** a definição JSON do agente `review` — `bash: deny` total anulava o bash granular do v1.1.44 (validado via `opencode debug config`). Correções:
  - Prompt do `review` no `opencode.json`/`opencode.linux.json` enriquecido com a metodologia que vivia no `.md` (severidades BLOCKER/MAJOR/MINOR/NIT, formato de achado, VERDICT, lente de segurança obrigatória, YAGNI check, sem linguagem performativa) — nada se perde com a remoção do arquivo.
  - `CleanupItem` ganhou `recursive` (`models.go` + `provision_shell.go` com `os.RemoveAll`) e `manifests/shell.yaml` passou a remover `~/.config/opencode/agents` inteiro no provisioning (fonte única de agentes = `opencode.json`).
- **Changed**: seeds de memória (`configs/memory/lessons.md`, `patterns.md`) sincronizados com a memória global — 7 lições novas (09-02: sudo no MCP ssh-manager, heredoc PHP, timeout não mata script; 09-03: mode built-in primary, defaults permissivos, refs superpowers, `.md` sobrescreve JSON) + pattern de agente read-only; entrada do `review` atualizada (agora em `opencode.json`, sem `agents/*.md`).
- **Motivo**: questionamento do usuário sobre utilidade das memórias globais para o projeto revelou o seed defasado e o `review.md` stale conflitante.

---

## [v1.1.44] - 2026-09-03

### ⚡ Agentes `plan` e `review`: bash read-only granular + escrita de planos

- **Changed**: `configs/opencode.json` e `configs/opencode.linux.json` — agentes `plan` e `review`:
  - `plan` agora `mode: all` (primary via Tab E subagent dispatchável via task tool — antes era primary-only por herdar o built-in).
  - `bash` trocado de `deny` total para granular read-only: `git log/status/diff/show`, `rg`, `go test/vet`, `pytest`, `npm test`, `dotnet test`, `cargo test` liberados; resto negado (`"*": "deny"`).
  - `plan`: `edit` liberado apenas em `docs/superpowers/plans/**` (salvar o plano); `review`: `edit` segue negado.
  - `task` restrito a `explore`; `todowrite` e `skill` liberados; `temperature: 0.1`; `steps: 25`.
  - Prompts: carregam `agent-memory` e (plan) `writing-plans`; instruem uso de context7/webfetch; wording corrigido (permission error, não "tool inválida").
- **Changed**: `configs/skills/writing-plans/SKILL.md` — removidas referências a skills inexistentes do pacote superpowers (`subagent-driven-development`, `executing-plans`, `superpowers:using-git-worktrees`); handoff adaptado ao fluxo local (task tool com subagent `general` por tarefa ou execução inline com checkpoints).
- **Changed**: `configs/AGENTS.md` — documentados os 3 agentes customizados (review/plan/goal).
- **Motivo**: falhas no agente `plan` — bash negado impedia coleta de evidências (estado do repo, baseline de testes), edit negado impedia salvar o plano, e o handoff da skill apontava para skills não instaladas. Fontes: docs oficiais opencode (agents/permissions/tools/mcp-servers) + obra/superpowers.

---

## [v1.1.43] - 2026-08-29

### 🔧 MCP (playwright, chrome-devtools): bunx + versão pinada

- **Changed**: comandos dos MCPs locais trocados de `npx -y @latest` para `bunx <pkg>@<versão>` (`@playwright/mcp@0.0.79`, `chrome-devtools-mcp@1.8.0`).
- **Motivo**: (1) `@latest` re-resolve o pacote a cada spawn (start lento/instável); (2) npx roda em processo `node` — se o agente der `Stop-Process node` (ex.: liberar porta), os MCPs caem no meio da sessão e o opencode **não reconecta** (status `failed`, tools somem do toolset); (3) bunx roda em `bun.exe` — sobrevive a kill de node e sobe mais rápido. Validado com handshake MCP (initialize + tools/list) sob bun: 24 tools (playwright), 29 tools (chrome-devtools).

---

## [v1.1.42] - 2026-08-29

### 🐛 Fix: 20 skills invisíveis para o opencode (frontmatter YAML inválido)

- **Fixed**: 20 skills (`agent-memory`, `git-workflow`, `database-ops`, `docker`, `windows-admin`, `vps-provisioning`, etc.) tinham `description` como scalar pleno com `: ` (ex.: `Triggers: git, ...`) — YAML interpreta `: ` como separador de mapping → erro de parse → o opencode **descarta a skill silenciosamente**. Todas convertidas para block scalar (`description: >-`), texto preservado.
- **Impacto**: `agent-memory` (obrigatória pelo AGENTS.md) e outras nunca carregavam; agora as 73 skills são descobertas. Requer reiniciar o opencode.

---

## [v1.1.41] - 2026-08-29

### 🐛 Fix: Windows Terminal fechando instantaneamente (commandline UTF-8)

- **Fixed**: o `commandline` do perfil PowerShell no Windows Terminal (`pwsh -Command "chcp 65001 > $null; $env:PATH = $env:PATH"`) executava o comando e saía sem `-NoExit`, fechando a aba/janela na hora. Removido de `configs/terminal-settings.json` — o encoding UTF-8 já é garantido pelo profile PowerShell (`[Console]::OutputEncoding/InputEncoding = UTF8` + `chcp 65001`), única camada necessária no launch.
- **Nota**: a validação do v1.1.40 (doctor 181/181) não cobriu a abertura real de um novo terminal via Windows Terminal — o profile é carregado só após o launch, e o `pwsh -Command` sem `-NoExit` encerra o processo antes disso.

---

## [v1.1.40] - 2026-08-29

### 🌐 Console UTF-8: correção do encoding Unicode (U+FFFD) no Windows

- **Fixed**: Console Windows com code page OEM 850 renderizava caracteres Unicode (🚀 ✔ ✘ ⚠ 🎉 e glyphs Nerd Font) como `�` (U+FFFD) — corrigido em 3 camadas:
  - Profile PowerShell (`configs/Microsoft.PowerShell_profile.ps1`) agora define `[Console]::OutputEncoding/InputEncoding = UTF8` + `chcp 65001` no início.
  - Windows Terminal (`configs/terminal-settings.json`): perfil PowerShell com `commandline: pwsh -Command "chcp 65001 > $null; $env:PATH = $env:PATH"`.
  - Bootstrap Windows (`bootstrap.ps1`) força UTF-8 antes do banner.
- **Added**: `envctl doctor` ganhou o check 12.6 *Console Code Page* — alerta WARN se `chcp != 65001` (com fix hint: reiniciar o Windows Terminal / `envctl run shell`).
- **Validado**: `doctor` 181/181, 0 WARN, 0 ERRO com o profile ativo (antes: 1 WARN Code Page 850).

---

## [v1.1.39] - 2026-08-28

### 🤖 Agentes OpenCode: review read-only sem erros de tool + modo goal (YOLO)

- **Fixed**: Agente `review` e `plan` agora bloqueiam `bash`/`edit` via `permission` (deny) em vez de remoção do toolset — modelos menores paravam de gerar erros "unavailable tool" e recebem negação limpa que sabem interpretar.
- **Added**: Agente primário `goal` (modo autônomo YOLO) — executa spec/objetivo até a conclusão sem pedir permissão, rastreia via goal tools (set_goal/create_goal/update_goal/get_goal), verifica com evidências antes de concluir (ponytail + verification-before-completion + agent-memory).
- **Changed**: `configs/opencode.json` e `configs/opencode.linux.json` ganharam seção `agent` (review, plan, goal) — distribuída a todas as VPSs via provisioning.

---

## [v1.1.38] - 2026-08-28

### 🆕 Day-0: provisionamento de VPS fresca (validado em zscanchatbot)

- **Fixed**: `apt-get update` roda antes do primeiro `apt-get install` (uma vez por processo) — VPS novas têm listas apt vazias e todo install falhava com `Unable to locate package` (`internal/infra/apt/apt_manager.go`).
- **Fixed**: No Linux, o passo *bootstrap* (Volta/Node/OpenCode/Go/Rust) roda ANTES de *packages* — pacotes gerenciados via Volta eram pulados como "manager not available" em VPS fresca (`internal/ui/cli/run.go`).
- **Fixed**: Bootstrap instala `fd-find` via apt (fallback) quando ausente, para o symlink `fd` convergir mesmo com o bootstrap antes do passo apt (`internal/usecase/provision_bootstrap.go`).
- **Fixed**: Managers de toolchain (volta, npm, dotnet, go, rustup) resolvem binários no PATH de toolchain (`~/.volta/bin`, `~/.local/bin`, `~/.cargo/bin`, `/usr/local/go/bin`, `~/go/bin`) — elimina falsos positivos do `doctor` em shells não-login (ssh/systemd) (`internal/infra/toolchain/*`).
- **Validado**: VPS nova `zscanchatbot` (AWS Ubuntu) — bootstrap → `run all` → `doctor` = **167/167, 0 WARN, 0 ERRO** (antes: 26 WARN).

---

## [v1.1.0] - 2026-08-28

### 🤖 Integração envctl ↔ OpenCode (controle autônomo de VPS/VM)

- **Added**: Seção `VPS Infrastructure (envctl)` no AGENTS.md global — provisioner, bootstrap de 1 linha, comandos, inventário via `ssh-manager server list`, regra "NOVAS VM/VPS — SEMPRE usar envctl" (nunca configurar VPS manualmente).
- **Added**: Skill `vps-provisioning` (workflow Day-0/Day-2 de provisionamento remoto idempotente) — catálogo de skills 72→73.
- **Added**: Protocolo "Limites Free & Fallback de Token" na skill `vps-agent-dispatch` — limite Free estourado na VPS → perguntar TOKEN ao usuário; recusou → agente executa os comandos via SSH.
- **Added**: Passo 4 no cadastro de VPS da skill `ssh-vps` — provisionar a VPS com envctl logo após o registro (VPS vira agente OpenCode orquestrável).
- **Added**: Seção `Provisioning (envctl)` no `AGENTS.linux.md` (auto-manutenção nas VPSs provisionadas) e referência `envctl` no `opencode.json`.
- **Added**: Seção "Workflow Inteligente Multi-VPS (OpenCode)" no README.
- **Fixed**: Contador de progresso do `envctl run all` no Windows mostrava "6/5" — total agora é incondicional (6 seções).
- **Changed**: Documentação alinhada ao estado real — contagem de skills 59→73, LSPs 16→18, remoção de referências a `tui.json` (arquivo removido do provisioning).
- **Changed**: Versionamento de release promovido de `v1.0.<n>` para `v1.1.<n>`.

---

## [v1.0.33] - 2026-08-27

### 🔐 Segredos e PII fora do versionamento

- **Fixed**: `CONTEXT7_API_KEY` removida dos configs versionados — agora referenciada via `{file:~/.config/opencode/secrets/context7.key}` (variável de arquivo do opencode); chave antiga deve ser rotacionada no dashboard do Context7 (esteve no histórico do repo).
- **Fixed**: MCP `docker-hub` com caminho absoluto do usuário removido do config versionado.
- **Fixed**: Perfil PowerShell aponta para o tema Oh-My-Posh provisionado (`~/.poshthemes/`) via `$HOME` em vez de caminho absoluto com username.
- **Fixed**: Fixture de teste de higiene temp sem username real.

### 🛡️ Revisão multiplataforma: doctor 100% + hardening de provisionamento Linux

- **Fixed**: `$HOME/go/bin` incluído no PATH do toolchain Linux (`linuxToolchainEnv`/`ensureProcessToolchainPath`) — `go install` (ex.: `gopls`) agora resolve em processo, sem depender de novo shell.
- **Fixed**: Instalação do Go SDK remove `/usr/local/go` antes de extrair (guia oficial — evita stdlib órfã corrompendo builds).
- **Fixed**: Diretórios na raiz do Linux (ex.: `/temp`) criados via `sudo` com sticky `1777` em vez de falhar com `permission denied`.
- **Fixed**: Leitura de env vars POSIX prioriza rc files (fonte de verdade literal `$HOME`) sobre o processo (valor expandido pelo shell) — elimina WARN falso de `NODE_PATH` e torna `run shell` verdadeiramente idempotente.
- **Fixed**: `csharp`/`csharp-ls` restritos a Windows (dotnet-tool não provisionado no Linux).
- **Added**: LSP `toml` (Taplo via `@taplo/cli`) no `manifests/lsp.yaml` — era configurado no opencode.json mas nunca provisionado/auditado.
- **Added**: Auditoria do `doctor` para `go`, `rustup`, `cargo` e `rust-analyzer` (seção Linux Bootstrap).
- **Changed**: Hint de sudo NOPASSWD usa o usuário real (`$USER`) em vez de hardcoded `ubuntu`.
- **Resultado**: `envctl doctor` = **166/166 (Linux VM)** e **179/179 (Windows)** — 0 WARN, 0 ERRO; idempotência validada (2ª execução sem reinstalações).

---

## [v1.0.23] - 2026-08-26

### 🛡️ Otimização de Contexto e Conformidade de Skills (Context7 Analysis)

- **Changed**: Otimização de contexto via `permission.skill` com `firecrawl-*: deny` no `opencode.json` e `opencode.linux.json` — reduz ~2.5k tokens de listagem de descrições por prompt mantendo ferramentas disponíveis via CLI e AGENTS.md.
- **Added**: Metadados de portabilidade `license: MIT` e `compatibility: opencode` no frontmatter de todas as 12 skills nativas do `envctl`.
- **Fixed**: `go.mod` sanitizado via `go mod tidy` — `pterm`, `cobra` e `yaml.v3` devidamente promovidos a dependências diretas.

---

## [v1.0.22] - 2026-08-26

### 🧠 Pipeline Memória → Skill (autônomo) + 11 Skills Promovidas + Arsenal de Agentes

- **Added**: Skill `memory-promotion` — classificação obrigatória de memórias (processo reutilizável → skill; lição/preferência → memória), criação de `SKILL.md` no local certo, **remoção da entrada da memória após promoção** (sem redundância), registro via `envctl snapshot`.
- **Added**: Parâmetro OBRIGATÓRIO `Skill Promotion` no `AGENTS.md` global — classificar a cada gravação de memória; skills globais novas sem dados pessoais são implementadas no envctl.
- **Added**: 11 skills promovidas da memória global: `phone-e164-normalization`, `bulk-postgres-import`, `parallel-agent-orchestration`, `web-dashboard-automation`, `docker-build-local-vps-deploy`, `nextjs-standalone-deploy`, `docker-desktop-wsl-restart`, `jwt-hs256-node`, `simple-feature-flag`, `playwright-prod-regression`, `lsp-smoke-test` (72 skills totais).
- **Added**: Memória global (sem PII — auditada) versionada como templates seed `configs/memory/*.md` com novo campo `ConfigFile.SeedIfMissing` — máquinas novas nascem com a baseline; adições locais nunca sobrescritas nem reverse-sync.
- **Added**: Arsenal global de agentes (complemento do v1.0.21): `jq`, `dust`, `hyperfine`, `shellcheck` (winget), `pyyaml`, `requests`, `openpyxl`, `beautifulsoup4` (pip), `axios`, `cheerio`, `papaparse` (node via NODE_PATH); comando `envctl run pip`; reinstalação automática de deps do `~/package.json` por mtime.

---

## [v1.0.21] - 2026-08-26

### ⚡ Overhaul de Performance do OpenCode + Stack PWSH/WSL + Arsenal Global de Agentes

- **Removed**: MSYS2 completamente do ambiente Windows (manifestos, configs, `PacmanManager`, comando `run pacman`, terminal-settings, docs, skills, CHANGELOG) — stack padronizada: **PowerShell 7 (primário) + WSL Ubuntu (secundário)**.
- **Changed**: Config do OpenCode unificada em `opencode.json` (fonte única JSON) — merge de plugins/LSPs/MCPs/instructions; `shell: "pwsh"` (resolução via PATH, suporta MSI e Store); `opencode.jsonc` e `tui.json` (plugin TUI não-funcional) removidos com auto-cleanup no provisioning.
- **Changed**: DCP configurado para compressão auto **85% max / 75% min**.
- **Removed**: Plugins sem função comprovada — `opencode-visual-cache` (TUI cosmético), `@vymalo/opencode-models-info` (inerte: nenhum provider com `modelsInfoUrl`), `opencode-thinking-fix` (no-op: 300+ inspects com `isReasoningModel:false`). Restam 3: dcp, ponytail, goal-plugin (verificados funcionais).
- **Fixed**: Duplicidade de skills — `~/.agents/skills` removido (eram symlinks pendentes após deleção), fonte única `~/.config/opencode/skills`.
- **Added**: `envctl run cleanup` (configs legados, tool-output >10MB, scratch `C:\temp`/`/temp` >24h) + auditoria no `doctor` (DB, tool-output, legacy config, TempFolder `ENVCTL_TEMP`, WSL Ubuntu).
- **Added**: Pasta de scratch padronizada para agentes LLM — `C:\temp` (Windows) / `/temp` (Linux), env var `ENVCTL_TEMP`; `pw-screenshot` salva lá por padrão.
- **Added**: Arsenal global de agentes — `bun` (winget), `jq`, `dust`, `hyperfine`, `shellcheck`; pip global `pyyaml`, `requests`, `openpyxl`, `beautifulsoup4`; node `axios`, `cheerio`, `papaparse`; comando `envctl run pip`; reinstalação automática de deps do `~/package.json` por mtime.

---

## [v1.0.19] - 2026-08-21

### 🔒 Segurança: PII removida de skills + memórias individuais por máquina

- **Fixed**: Removida tabela de servidores (IPs, usuários, caminhos de chaves) e exemplos com IP real da skill `ssh-vps` — dados reais agora ficam APENAS em `~/.config/opencode/extras/ssh_servers.md` (inventário local por máquina, nunca versionado). A skill consulta dinamicamente (`ssh-manager server list` / MCP `ssh_list_servers`).
- **Changed**: Memórias globais do agente (`~/.config/opencode/memory/`) deixam de ser provisionadas via `manifests/shell.yaml` e não são mais reverse-syncadas no `envctl snapshot` — são individuais por PC/VPS (exclusão defensiva para `memory/` e `extras/` no `snapshot_sync.go`).
- **Added**: Diretório `~/.config/opencode/extras` no manifest (extras locais por máquina, ex: `ssh_servers.md`).
- **Changed**: Reforço do uso ativo da skill `agent-memory` no `AGENTS.md` (Windows/Linux) — LOAD obrigatório no início de toda tarefa para todos os agentes/subagentes; SAVE imediato ao aprender.
- **Removed**: `configs/memory/*.md` (templates de memória global) do repositório.

---

## [v1.0.18] - 2026-08-20

### 🧹 Higiene de Temp & Scratch (Windows/Linux)

- **Added**: Regra obrigatória de higiene de temp no `AGENTS.md` (Windows e Linux) — todo scratch/download/build/cópia de banco criado em `/tmp` deve ser removido antes do fim da sessão, com comando de limpeza documentado e recomendação de subdir dedicado por sessão.
- **Added**: Novo `TempHygieneUseCase` (`internal/usecase/temp_hygiene.go`) — auditoria do diretório temp (`C:\temp` no Windows, `/tmp` no Linux) e poda de artefatos obsoletos: extrações de módulos nativos do runtime Bun (`.bdef*.dll`/`.feef*.node`), `node-compile-cache`, `tsx-*`, scratch de sessões de agentes (`opencode/` com idade > 6h), downloads de ferramentas (`zscan-*`, `Meslo.zip`), caches regeneráveis (WinGet/NuGet/MSBuild/VS Code), logs de instaladores e arquivos soltos de sessões. Arquivos travados por processos em execução são pulados com aviso.
- **Added**: `envctl doctor` ganhou a checagem `TempHygiene` no relatório e `envctl doctor --fix` uma 6ª etapa de limpeza automática de temp (com resumo de artefatos removidos/liberados/pulados).
- **Fixed**: Alinhamento `gofmt` em `models.go`, `manifest_repo.go`, `env_manager.go` e `root.go` (pré-existente).

---

## [v1.0.17] - 2026-08-20

### 🐛 Correção de Bug
- **Fixed**: Comando `version` exibia versão hardcoded `v1.0.0` (em `internal/ui/cli/root.go`). O `-ldflags "-X main.Version=$VERSION"` apontava para uma variável que não existia, fazendo todos os releases reportarem `v1.0.0`.
- **Changed**: `main.Version` agora é uma variável injetável em `cmd/envctl/main.go` (default `dev`) e propagada ao CLI via `InitApp(embeddedFS, version)`; o comando `version` imprime a versão real do build. Validado: `-X main.Version=v1.0.16` → `envctl v1.0.16`; build sem ldflags → `envctl dev`.

---

## [v1.0.16] - 2026-08-20

### 🐛 Correções de Bugs (Auditoria Profunda do Codebase)
- **Fixed**: `AGENTS.md` global passou a ser implantado em `~/.config/opencode/AGENTS.md` (antes `~/AGENTS.md`, caminho que o opencode nunca lê); regra **Zero Tolerância** promovida para o global + checagem explícita no `doctor`.
- **Fixed**: Seção `directories:` do `shell.yaml` agora é funcional (`LoadDirectories`) — pastas `~/.ssh/sockets`, `~/.local/bin`, `~/.poshthemes` passaram a ser provisionadas; paths `C:/projetos/*` corrigidos.
- **Fixed**: Filtros `os:` em `shell.yaml` (env vars/config files/dirs), `git.yaml` (`core.fscache`/`core.longpaths` → windows), `lsp.yaml` (powershell/gopls/rust/csharp → windows) e `packages.yaml` (~35 pacotes winget/volta/dotnet-tool → windows) — elimina falsos warnings no Linux.
- **Fixed**: `doctor` git worktree não gera mais falso warning fora de repositório (`rev-parse --is-inside-work-tree`).
- **Fixed**: Variáveis de ambiente agora persistem no Linux (`~/.profile`/`~/.bashrc`) + escape de aspas em comandos PowerShell.
- **Fixed**: `GoManager.IsInstalled` usa `exec.LookPath` no POSIX (não `where.exe`); `NpmManager.ListInstalled` lista pacotes reais; `VoltaManager.IsInstalled` match exato por token.
- **Fixed**: Snapshot preserva metadados — `os:` do `git.yaml`, e merge das skills (target_dir/os/enabled/files/description); não exporta `~/.ssh/config`; guarda "nothing to commit".
- **Fixed**: Panic por `nil` em `tweaks_manager` (`cmd.ProcessState.ExitCode()`); backup/perms honram `StrictACL` (0600 + ACL); `%VAR%` indefinida mantém literal; `file_logger` reporta `GOOS/GOARCH` reais.

### 🧱 Replicação Cross-Platform nas VPS Ubuntu (validação em produção)
- **Added**: Novo `ProvisionBootstrapUseCase` (`envctl run bootstrap`) — instala Volta + Node 24 LTS + pnpm, OpenCode CLI (npm com fallback curl), `gh`, `delta`, `yq`, `uv`, `ruff`, `oh-my-posh`, `fd` (symlink `fdfind`), `pylsp` (via uv, contornando PEP 668), `firecrawl-cli` e `stylelint`; integração do Volta no PATH de shells de login.
- **Added**: Configs Linux dedicados — `configs/opencode.linux.jsonc` (sem shell/MCPs Windows), `configs/AGENTS.linux.md`, `configs/ssh-config.linux`; entradas condicionais por OS no `shell.yaml`.
- **Added**: `apt_manager` com `sudo -n` quando não-root; `playwright install-deps chromium` no Linux; seção 12 do `doctor` (toolchain Linux) + resolução de PATH de toolchain nos LSPs.
- **Changed**: `packages.yaml` ganhou `sshpass` (apt) e `user-package.json`/`.skill-lock.json` normalizados.
- **Validado**: ambiente opencode global replicado e verificado na VPS `<SERVER>` (doctor 124/124 no Linux).

### ⚙️ DCP — Limites Adaptativos
- **Changed**: `dcp.jsonc` usa limites percentuais `"90%"/"80%"` da janela do modelo (adaptativo a modelos de contexto grande, ex. 1M) em vez de valores fixos.

### 🧠 Nova Skill `agent-memory` (Memória de Lições & Padrões)
- **Added**: Skill `agent-memory` com fluxo LOAD → ACT → SAVE → REFLECT — o agente lê a memória antes de cada tarefa e grava lições/patterns ao aprender ou ser corrigido (estilo "Taste" do Command Code).
- **Added**: Arquivos de memória globais (`~/.config/opencode/memory/{lessons,patterns}.md`) e por projeto (`.opencode/memory/*.md`, versionáveis em PR) + regra obrigatória de leitura no `AGENTS.md` (Windows/Linux).

### 🔌 Skills & Integrações
- **Added**: Regra de ativação do MCP `ssh-manager` na skill `ssh-vps` — o agente pede ao usuário ativar via `/mcp` ou `Ctrl+P` (hot-reload) quando perceber que é ideal.
- **Fixed**: Skill `vps-agent-dispatch` usa o pacote npm correto `opencode-ai` (era `@opencode-ai/cli`).

---

## [v1.0.13] - 2026-08-18

### 📚 Documentação & Guias Multi-OS
- **Added**: Suíte completa de documentação modular sob `docs/`:
  - `docs/architecture.md`: Clean Architecture, camadas internas, abstração de I/O e binário standalone (`//go:embed`).
  - `docs/manifests.md`: Especificação declarativa dos arquivos YAML (`packages.yaml`, `shell.yaml`, `git.yaml`, `lsp.yaml`, `windows.yaml`).
  - `docs/skills.md`: Catálogo das 59 skills de IA, orquestração remota (`vps-agent-dispatch`) e automação Playwright.
  - `docs/doctor-and-idempotency.md`: 160+ checagens diagnósticas, flag `--fix`, backups atômicos (`.bak.timestamp`) e trilha de auditoria.
  - `docs/guides/windows.md`: Guia de provisionamento para Windows 11 PRO / Server.
  - `docs/guides/linux.md`: Guia de provisionamento para Ubuntu / Debian / VPS (AWS & Oracle).
  - `docs/guides/macos.md`: Guia de provisionamento para macOS Apple Silicon e Intel.
- **Changed**: `README.md` reescrito para ser enxuto, moderno, direto ao ponto, com quickstart de 1 linha e links diretos para a documentação técnica.

---

## [v1.0.12] - 2026-08-18

### 🌍 Expansão Cross-Platform & Orquestração de Subagentes
- **Added**: Renomeação do projeto de `win11-new` para `envctl` (`github.com/eajdias/envctl`).
- **Added**: `AptPackageManager` (`internal/infra/apt/`) com suporte a instalações não-interativas no Debian/Ubuntu Linux.
- **Added**: `bootstrap.sh` para instalação de 1 linha em sistemas Unix/Linux/macOS.
- **Added**: `vps-agent-dispatch` Agent Skill para orquestração de subagentes remotos via SSH, executando tarefas pesadas em servidores VPS (AWS/Oracle) e retornando sumários cristalizados.
- **Added**: Matrix multiplataforma no GitHub Actions (`.github/workflows/ci.yml` e `.github/workflows/release.yml`) compilando binários para Windows (amd64/arm64), Linux (amd64/arm64) e macOS (amd64/arm64).
- **Added**: ADR `0001-cross-platform-architecture.md` documentando a arquitetura multiplataforma.
- **Fixed**: Guards condicionais em `tweaks_manager.go` e `tweaks_manager_test.go` para evitar chamadas ao PowerShell no Linux durante testes de CI.
- **Fixed**: Padrões ancorados no `.gitignore` para não colidir com `cmd/envctl/main.go` ou `configs/bin/`.

---

## [v1.0.11] - 2026-08-18

### 🛠️ Auto-Remediação do Doctor & Ferramentas de Build
- **Added**: Flag `--fix` no comando `envctl doctor` para auto-remediação automática de qualquer inconsistência detectada nos 160+ pontos de checagem.
- **Added**: `Makefile` e `Taskfile.yml` com alvos padronizados (`build`, `test`, `doctor`, `doctor-fix`, `snapshot`, `install`).
- **Added**: Suporte a autocompletar shell gerado pelo Cobra (`envctl completion [bash|zsh|fish|powershell]`).

---

## [v1.0.10] - 2026-08-18

### 🎭 Utilitários Playwright & Resolução de Módulos
- **Added**: Variável `NODE_PATH` apontando para `%USERPROFILE%\node_modules` para garantir resolução global de módulos Node no Windows.
- **Added**: `configs/bin/pw-screenshot` e `configs/bin/pw-screenshot.cmd` para captura instantânea de telas headless em alta resolução.
- **Added**: `configs/bin/pw-eval` e `configs/bin/pw-eval.cmd` para avaliação rápida de DOM e scripts via Playwright Node.js API em < 1s.
- **Added**: Aliases de `git worktree` (`gwc`, `gwl`, `gwr`) e auditoria de integridade de worktrees no `doctor`.
- **Added**: Template robusto de `~/.ssh/config` com multiplexação de sockets (`ControlMaster auto`, `ControlPath ~/.ssh/sockets/%r@%h:%p`) e keepalive.

---

## [v1.0.7] - [v1.0.9] - 2026-08-18

### 🌐 Playwright Node API Migration
- **Changed**: Substituição da CLI instável `@playwright/cli` pela arquitetura estável Node.js API (`const { chromium } = require('playwright')`).
- **Added**: Resolução dinâmica de `NODE_PATH` no perfil do PowerShell e no shell WSL Ubuntu (`cygpath -m`).
- **Added**: `configs/user-package.json` gerenciando a dependência do `playwright` na raiz do usuário (`~`).
- **Added**: Aliases Docker preservando volumes e caminhos de containers sem conversão de caminhos POSIX.

---

## [v1.0.4] - [v1.0.6] - 2026-08-18

### 🪟 Windows 11 PRO Registry Tweaks & LSPs Nativos
- **Added**: `WindowsTweaksManager` (`internal/infra/windows/`) gerenciando Win32 Long Paths (`LongPathsEnabled = 1`), Developer Mode (`AllowDevelopmentWithoutDevLicense = 1`), Explorer show extensions (`HideFileExt = 0`), Explorer show hidden files (`Hidden = 1`) e Dark Theme.
- **Added**: Instalação e verificação automática da fonte `MesloLGM Nerd Font`.
- **Added**: `GoManager` (`go install ...@latest`) e `RustupManager` (`rustup component add ...`) gerenciando `gopls` e `rust-analyzer`.
- **Added**: `csharp-ls` via `.NET global tools` e `marksman` via Winget.
- **Added**: Templates de configuração de Terminal (`terminal-settings.json`), perfil do PowerShell (`Microsoft.PowerShell_profile.ps1`) com Oh-My-Posh e VSCode User Settings.

---

## [v1.0.1] - [v1.0.3] - 2026-08-18

### 📜 Logging Persistente & Gerenciador Volta
- **Added**: `FileLogger` (`internal/infra/logger/`) gravando sessões estruturadas em `~/.envctl/logs/envctl-YYYYMMDD-HHMMSS.log` com sanitização de null bytes de consoles UTF-16.
- **Added**: `VoltaPackageManager` (`internal/infra/toolchain/`) para gerenciamento declarativo de Node.js LTS e CLIs globais (`firecrawl-cli`, `pnpm`, `stylelint`, `sqllens-language-server`).
- **Added**: Expansão de `manifests/packages.yaml` incluindo ferramentas de produtividade (VSCode, Windows Terminal, Oh-My-Posh, Termius, WinSCP, 7-Zip, Everything, Brave Nightly, GitHub Desktop, WSL).

---

## [v1.0.0] - 2026-08-18

### 🚀 Lançamento Inicial (Protótipo win11-new)
- **Added**: Arquitetura base em Go (Clean Architecture) com camadas `domain`, `usecase`, `infra` e `ui`.
- **Added**: Binário 100% standalone via `//go:embed` embutindo manifestos declarativos YAML e templates de configuração.
- **Added**: Gerenciadores de infraestrutura para `Winget`, `Dotnet Tool`, `Git` e `FileSystem`.
- **Added**: Backup atômico com timestamp (`.bak.YYYYMMDD-HHMMSS`) para alterações em arquivos de configuração existentes.
- **Added**: Catálogo inicial de 57 skills de agentes de IA para o OpenCode.
- **Added**: Comandos CLI Cobra com interface ANSI via PTerm (`run`, `doctor`, `snapshot`, `version`).
