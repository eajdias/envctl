# Spec — Migração Volta → mise (+ npm existente)

**Data:** 2026-10-06 · Skill `writing-plans` · Análise que motivou: volta EOL oficial
**Status do volta:** unmaintained desde 2025-11-14 (`volta-cli/volta` README + issue #2080),
maintainers recomendam `mise`. Sem urgência (o que funciona continua), mas no roadmap.
**Escopo:** trocar o volta por **mise (runtimes) + npm existente (globals)**. `packages.yaml`
continua fonte única de verdade — nenhum `mise.toml` versionado.

## 0. Decisão de desenho (medida, não assumida)

| Ponto | Decisão | Fonte |
|---|---|---|
| Quem instala o quê | mise: `node`, `go`. npm (`NpmManager`, `--prefix ~/.local`, já existe): todos os globals JS (packages + LSPs) | `toolchain_managers.go:21-80`; `lsp.yaml` 10× `install_type: volta` |
| Por que não `npm:` backend do mise | `NpmManager` já testado; menos um backend p/ auditar | idem |
| Por que não `mise.toml` no repo | manifesto continua canônico; `MiseManager` traduz `type: mise` → CLI | filosofia "manifestos são fonte" (Fase 6-T4) |
| Trust | não-problema: sem `mise.toml` de projeto não há o que confiar; global é operator-owned | `mise.jdx.dev` docs `trust`/`faq` (2026-09) |
| Go hoje | bootstrap baixa `go.dev/VERSION?m=text` (**latest**) via tarball | `provision_bootstrap.go:587-592` → `mise install go@latest` preserva o comportamento exato |
| Modelo de PATH | shims do mise (`~/.local/share/mise/shims`, confirmar U-M3) — 1 dir, funciona em shell não-login (lição #21); `fnm env` recriaria o problema do volta | — |
| `update.go` hoje | `GroupVolta` cobre `PackageTypeVolta` **e** `PackageTypeNpm` (`:29-30`), ambos via `volta install` | `update.go:19-110` — a migração **separa**: `GroupMise` (runtimes) + `GroupNpm` novo (globals via `npm i -g`) |
| LSP | `provision_lsp.go:65` já é genérico (`managers[InstallType]`) — trocar o campo no yaml basta, sem código | `:65-95` |
| Fora de escopo | uv (python), pacman/apt/winget (sistema), tasks do mise, `mise.toml` versionado | — |

## 1. Impacto e contratos

**Superfície:** `internal/infra/toolchain/mise_manager.go` novo + `volta_manager.go` deletado;
`provision_bootstrap.go` (`ensureVolta`, go tarball, `pathStep` dirs, VOLTA_HOME);
`provision_providers.go` (fase 0); `update.go` (grupos); `executil.ToolchainDirs`
(`~/.volta/bin` → shims); `env_manager` PATH dirs; `doctor_packages.go:238-248`
(tabela toolchain); `doctor_agents.go` (refs volta); `manifests/packages.yaml`
(9 entradas) + `lsp.yaml` (10); `root.go` wire; `entity.PackageTypeVolta` →
`PackageTypeMise`; testes (`volta_manager_test`, `provision_providers_test`,
`update_test`, `toolchain_path_test`, `verify_script_test`); matriz + docs §1/§5 +
`Volta.Volta` winget + `code-playbooks/references/docker.md`.

**Contratos que NÃO mudam:** CLI (`run`, `doctor`, `update`, flags, exit codes),
`doctor` 0 WARN/0 ERROR como meta, formato do gate, `NpmManager` (`--prefix ~/.local`).

**Não-regressão:** `go build ./... && go vet ./... && go test ./... &&
golangci-lint run --new-from-rev=origin/main` verdes em cada PR; `doctor`
ao vivo numa máquina com volta instalado deve convergir sem WARN novo.

## 2. Tarefas (uma entrega verificável por item, PRs pequenos na ordem)

- [x] **M1 — `MiseManager` novo (sem ligar em nada).** (feito 2026-10-06: TDD RED→GREEN; `mise ls --json` shape confirmado na doc oficial + e2e do mise; `PackageTypeMise` adicionado, volta intocado)
      Arquivos: criar `internal/infra/toolchain/mise_manager.go` (+ teste).
      Interfaces: implementar `repository.PackageManager` (medir em
      `volta_manager.go:16-97` quais métodos o wire/testes exigem:
      `Type/IsAvailable/IsInstalled/Install/ListInstalled`); `IsAvailable` =
      `mise --version` resolvível; `Install` = `mise install <tool>@<versão>`;
      `ListInstalled` = mínimo (`mise ls` parse — ver U-M2).
      TDD: RED = teste novo `TestMiseManagerReportsMissingTool` falha (sem
      binário); GREEN = passa com fake `PATH` (mesmo padrão dos testes de
      toolchain existentes).
      Verificação: `go test ./internal/infra/toolchain/ -v` + gate.
      Rollback: deletar o arquivo (nada o referencia).
      Resultado: manager compila e testa isolado; volta intocado.
- [x] **M2 — bootstrap: `ensureVolta` → `ensureMise` + Go via mise.** (feito 2026-10-06: `mise use -g node@<spec> pnpm`, tarball Go → `mise use -g go@latest` + `go version`, stylelint/cmdc via npm prefix, pathStep shims; providers em paralelo via subagent)
      Arquivos: `internal/usecase/provision_bootstrap.go` (+ teste).
      `volta install node@24 pnpm` (`:365`) → `mise install node@24 pnpm`
      (canal de instalação do mise por OS — ver U-M1); bloco tarball Go
      (`:587-592`) → `mise install go@latest`; `pathStep` Volta/Go dirs →
      shims dir; VOLTA_HOME sai do script (era `shellEnv`, não PATH).
      TDD: RED = `TestBootstrapInstallsNodeViaMise` (adaptado do teste de
      bootstrap existente) falha chamando volta; GREEN = chama mise.
      Verificação: `go test ./internal/usecase/ -run TestBootstrap -v` + gate.
      Rollback: revert do PR. Resultado: máquina nova sem volta instala node+go.
- [x] **M3 — manifests: 9 packages + 11 LSPs.** (feito 2026-10-06: `node`→mise, 7 globals + `command-code`→npm, 11 `install_type`→npm no lsp.yaml — todos alvos npm genuínos, gopls/powershell/taplo já eram go/winget/npm; `Volta.Volta`→`jdx.mise` no winget; `TestLSPSingleSource` verde)
      Arquivos: `manifests/packages.yaml:237-307` (`node@24.19.0` → `type: mise`;
      8 globals → `type: npm`), `manifests/lsp.yaml` (10× `install_type: volta`
      → `npm`). Nenhum código (LSP é genérico; packages passam por managers).
      Trava: `TestLSPSingleSource` + testes de manifest existentes.
      Verificação: `go test ./internal/infra/embedded/ -v` + gate.
      Rollback: revert. Resultado: zero `volta` em `manifests/` (`rg` vazio).
- [x] **M4 — `update.go`: `GroupVolta` → `GroupMise` + `GroupNpm`.** (feito 2026-10-06: grupos separados; `applyUpdate` npm usa `--prefix` via `toolchain.UserLocalPrefix` exportado (display mostra forma portátil); `run volta`→`run mise`; `--only` aceita mise/npm/uv/go; testes migrados)
      Arquivos: `internal/usecase/update.go` (+ `update_test.go`).
      `automatableGroup`: `PackageTypeMise` → `GroupMise` (`mise install
      <target>@latest`), `PackageTypeNpm` → `GroupNpm` novo (`npm install -g
      <target>@latest` via toolchain); `GroupVolta` deletado com seus 3
      `case`s (`:49-50,85-86,100-101`).
      TDD: RED = tabela de `update_test.go` esperando `GroupVolta` falha;
      GREEN = novos grupos resolvem.
      Verificação: `go test ./internal/usecase/ -run TestUpdate -v` + gate.
      Rollback: revert. Resultado: `rg GroupVolta` vazio.
- [x] **M5 — PATH toolchain: `~/.volta/bin` → shims.** (feito junto no M2: `ToolchainDirs` = opencode/local/shims/`~/go/bin`, sem `/usr/local/go/bin`; `VOLTA_HOME` fora do `ToolchainEnv`; U-M3 resolvido: `~/.local/share/mise/shims` na doc oficial)
      Arquivos: `internal/infra/executil/toolchain.go` (`ToolchainDirs`),
      `provision_bootstrap.go`/`provision_providers.go` (dirs do `pathStep`),
      `internal/infra/environment/env_manager.go` (nada — é genérico).
      Verificação: `go test ./internal/infra/executil/ ./internal/usecase/`
      + `bash -lc 'command -v node'` pós-`run bootstrap` numa máquina real.
      Rollback: revert. Resultado: `rg volta/bin` vazio fora de testes legados.
- [x] **M6 — doctor: tabela toolchain + refs.** (feito 2026-10-06: linhas volta/node/stylelint + hint command-code no Windows)
      Arquivos: `doctor_packages.go:238-248` (`volta`→`mise`, `node (via Volta)`→`(via mise)`,
      `stylelint (via Volta)`→`(via npm)`), `doctor_agents.go` (refs volta).
      Verificação: `go test ./internal/usecase/ -run TestDoctor -v` + doctor
      ao vivo 0 WARN/0 ERROR + gate.
      Rollback: revert. Resultado: `rg -i volta internal/usecase/doctor_*` vazio.
- [x] **M7 — testes e entidade.** (feito 2026-10-06: `volta_manager*.go` deletados, `PackageTypeVolta` removida, wire limpo; `verify_script_test` mantém fixtures "Volta error" como texto; `rg volta internal/` zerado fora disso)
      Arquivos: deletar `volta_manager_test.go` (M1 traz o de mise);
      `provision_providers_test.go`, `toolchain_path_test.go`,
      `verify_script_test.go` (refs volta); `entity/models.go`
      (`PackageTypeVolta` → `PackageTypeMise`).
      Verificação: `go test ./...` + `rg -iw volta internal/ manifests/` vazio
      (docs/guide à parte, M8).
      Rollback: revert. Resultado: a palavra `volta` só resta em docs/histórico.
- [x] **M8 — docs/matriz.** (feito 2026-10-06, roteiro docs-sync: `pw`→shims mise, `pw.cjs`/verify mantêm fallback legado documentado; README/AGENTS/matriz/principles/verification/manifests/roadmap/AGENTS.arch×2/memory-seed atualizados; `architecture.md` deixado p/ o dono; histórico #9 intacto)
      Arquivos: matriz §1/§5, guias (canal de instalação por OS), `Volta.Volta`
      winget fora, `code-playbooks/references/docker.md` (menção volta).
      Verificação: `rg -iw volta docs/ configs/` só com contexto histórico +
      testes de embedded verdes.
      Rollback: revert. Resultado: doc conta a história nova.
- [x] **M9 — validação ao vivo (Fase 5).** (feito 2026-10-06, homologacaochatbot Ubuntu 24.04: `run vps` convergiu via mise — node v24.19.0 pinado + pnpm, shims com mtime do run; `doctor` 152/148/4/0, warns só host-owned; 2ª run idempotente, mesmo veredito)
      `vps_oracle_2`: `run bootstrap` instala node+go via mise → `run vps` →
      `doctor` 0/0; segunda run idempotente. Windows: na estação do dono
      (sem dockur — sem KVM em Windows).
      Rollback máquina: `~/.volta` permanece no disco (não deletado pela
      migração); volta reinstalável pelo canal antigo.

## 3. Riscos

- **mise quebra no Windows/ARM sem HW p/ testar** (prob. média/impacto alto) —
  mitigação: canal oficial por OS verificado no U-M1; ciclo dockur (M9) antes de mergear na `main`.
- **shims dir muda entre versões do mise** (baixa/médio) — U-M3 fixa o path com teste (`mise --version` + `mise where`); se mudar, 1 const muda.
- **`mise install` interativo pedindo trust em config de projeto** (baixa/médio) —
  não há `mise.toml` de projeto no desenho; se um dia houver, só com `[tools]` plain (safe, sem trust).
- **Velocidade do projeto mise** (baixa/alto) — pinar o canal de instalação; só usar
  `install/use/exec/ls` (superfície estável há anos).

## 4. Unknowns

| id | pergunta | bloqueia | dono | próximo passo |
|---|---|---|---|---|
| U-M1 | canal oficial de instalação por OS | M2 | **resolvido 2026-10-06:** Linux `curl https://mise.run \| sh` (→ `~/.local/bin/mise`); Windows `winget install jdx.mise` (doc oficial `installing-mise`, 2026-10-04). arm64 coberto pelo install.sh (checar no M9) |
| U-M2 | `ListInstalled` mínimo do `MiseManager` (`mise ls --json` estável?) | M1 | quem implementar | `mise ls --help` na versão pinada antes de codar |
| U-M3 | path exato dos shims (`~/.local/share/mise/shims`?) | M5 | quem implementar | `mise where --help` / `mise doctor` na máquina de teste |
| U-M4 | `~/.volta` antigo: deletar na migração ou deixar? | M9 | dono | decisão: **deixar** (rollback de máquina); cleanup posterior se quiser |

## 5. Breaking changes

Nenhuma em CLI/flags/exit codes/manifests externos (manifests são embedados,
consumidor único é o próprio binário). Mudança de estado de máquina: `~/.volta`
deixa de entrar no PATH (permanece no disco, U-M4). `envctl update` passa a
atualizar node via mise e globals via npm — mesma versão final, outro instalador.

## 6. Definition of DoD

- [x] `rg -iw volta internal/ manifests/ cmd/` vazio (só fixtures "Volta error" em `verify_script_test`, congelado por T9)
- [x] homolog instalou node+go+globals sem volta; `doctor` 152/148/4/0 (warns só host-owned: 3 sysctl + reboot); dockur descartado (sem KVM)
- [x] segunda `run` idempotente; `update --only npm` moveu 5 globals ao vivo (playwright, bash-ls, command-code, stylelint, typescript), 0 falhas; node fica pinado (providers own it — `update` nunca move runtime, por desenho)
- [x] gate cheio verde em cada PR (CI 4/4 no #74); `golangci-lint run ./...` 0 findings
- [x] matriz/docs contam a história nova; spec §6 da simplificação atualizada (fnm→mise)
- [x] rollback de máquina documentado (volta reinstalável, `~/.volta` intacto)
