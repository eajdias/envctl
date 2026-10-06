# Spec — Simplificação do envctl: código, CI/CD e rotina de teste multi-OS

**Data:** 2026-10-05
**Roadmap:** próximo do item 6 (rename) — esta spec nasce da análise de sessão
(`análise por que commandcode/opencode não atualizam` → `volta x mise x fnm` →
`simplificação máxima do projeto e do CI/CD`), com pesquisa web/context7 e
medição no código.
**Escopo:** **spec, sem implementação.** O recorte é o que foi julgado
implementável e verificável; o resto fica adiado com o motivo.

---

## 0. Decisões já tomadas nesta sessão (executadas)

| # | Ação | Evidência |
|---|---|---|
| 1 | PR #70 merged (`branches-ignore` no branch do release-please) | checks 4/4, `c41eb29` |
| 2 | PR #71 merged: Taskfile.yml removido (duplicava o Makefile; Windows-oriented: buildava `envctl.exe` até no Linux) | `9a643f1` |
| 3 | PR #72 merged: gofmt dos 12 arquivos do advisory do gate | `999b73a` |
| 4 | A futura saída do **Volta** será **fnm + npm** — não mise (decisão registrada; ver §6) | análise da sessão |
| 5 | CI/CD atual avaliado: **já enxuto** (3 workflows, papéis limpos, least-privilege, paths filter). A simplificação restante é o goreleaser (§3) + higiene de CI sem risco de release (§3c) | `docs/os-and-agent-matrix.md` §1 |
| 6 | Verificação SPEC×código (2026-10-05): medições confirmadas com correções (§2) + fases novas 1c/1d/2g–2j/6 a partir dos gaps encontrados | análise + medição direta |

## 1. Objetivo

Simplificar ao máximo o projeto **mantendo todas as funcionalidades**, simplificar
o CI/CD/release ao mínimo de moving parts, e estabelecer a **rotina de teste
multi-OS** (Ubuntu Server real via `vps_oracle_2` + Windows local via
dockur/windows) como gate de release.

## 2. O que já existe (medido, não assumido)

| peça | estado | evidência |
|---|---|---|
| CI | 3 workflows: `ci.yml` (actionlint+lint+test ubuntu/windows+build), `release-please.yml` (bot + attach via `workflow_call`), `release.yml` (build manual 4 targets + sha256 + upload) | `.github/workflows/` |
| release | release-please dono da tag; v1.13.0 publicada com **11 assets confirmados**; poluição histórica de 74 tags já resolvida pelo desenho atual | `gh release view v1.13.0` |
| release manual | ~51 linhas imperativas (não ~80): `release.yml:63-87` Build (25 linhas, 4× `go build`), `:89-97` checksum (9), `:99-115` upload (17); pré-req ausente: `checkout@v7` sem `fetch-depth: 0` (shallow quebra o goreleaser — corrigir no T2) | `.github/workflows/release.yml` |
| bootstrap arm64 | **BUG**: `bootstrap.ps1:36-40` rejeita não-amd64, mas a release entrega `envctl-windows-arm64.{exe,zip}` (`release.yml:74-76,106,108`); `bootstrap.sh:24-31` já mapeia `aarch64→arm64` | Fase 1d (nova) |
| bootstrap baixa assets **por nome fixo** | `bootstrap.sh:74` → `envctl-${OS}-${ARCH}.tar.gz`; `bootstrap.ps1:77,93` → `envctl-windows-amd64.zip` | leitura direta |
| Makefile/Taskfile | Taskfile removido (PR #71); Makefile único, 44 linhas, cross-OS | `Makefile` |
| código | **31.9k LOC**: usecase 15.9k (29 código + 29 testes), infra 11.8k, domain 2.5k, ui 1.6k | `find \| xargs wc -l` |
| interfaces | **17 em `domain/repository/`**; só `PackageManager` tem polimorfismo real (**8 impls**: pacman/apt/winget/paru/**volta**/npm/pip/go — o SPEC dizia 7, faltava `toolchain/volta_manager.go`); `FileSystemManager`, `ManifestRepository`, `GitManager`, `WindowsEnvManager`/`WindowsTweaksManager` + ~9 de performance têm **1 impl** | grep das `type X interface` + `root.go:100-109` |
| testes injetam | por interfaces locais próprias (`latestVersionFn`, `UpdateEnv`), **não** pelas do repository | `provision_providers.go:37`, `update.go:69` |
| KVM local (dockur/windows) | **viável, verificado**: `/dev/kvm` presente, 16 flags VT-x, 22GB RAM (14GB livres), 588GB disco, Docker 29.8.2 | medição direta |
| alvo Ubuntu real | `vps_oracle_2` no inventário Tailscale (`100.92.37.112`, user `ubuntu`, key) | `ssh_list_servers` |
| gofmt baseline | **zero** (PR #72) | `gofmt -l .` |

## 2b. Impacto e contratos

**Superfície alterada:** `.github/workflows/release.yml` + `.goreleaser.yml`
novo (Fase 1); `ci.yml` + `Makefile` (Fase 1c, CI-only); `bootstrap.sh` +
`bootstrap.ps1` (Fase 1d); `internal/domain/repository/*.go` + construtores
dos usecases + `internal/ui/cli/root.go` (wire) + `internal/infra/*` (Fase 2);
`internal/usecase/version.go` novo + `internal/infra/executil/*` + `fs_manager.go`
(Fase 2g); `provision_bootstrap.go`, `provision_providers.go`,
`cleanup_opencode.go`, `run.go`, `update.go`+`update_env.go`,
`performance_options.go`, `doctor_linux_performance.go` (Fase 2h);
`configs/opencode.linux.json` deletado + `shell.yaml` overlays + `configs/*.md`
(Fases 2d/2i); `manifests/lsp.yaml`/`packages.yaml`, `shell.yaml` pares OS,
`debloat.yaml` rename, hooks git (Fase 2j); `docs/*.md` (Fase 6, só texto);
~14 arquivos de `internal/usecase/` fundidos (Fase 3); nada na Fase 5 toca o
repo (só docs de evidência).

**Contratos que NÃO mudam em nenhuma fase:** CLI (`run`, `doctor`, `update`,
`snapshot`, `opencode`, `commandcode`, flags `--fix/--dry-run/--only`),
exit codes, `manifests/*.yaml` (source of truth), `configs/` (templates),
nomes dos 11 assets de release (contrato com `bootstrap.sh:74` e
`bootstrap.ps1:77,93`), formato do `doctor` (219 checks) e do gate
(`envctl-verify --hook/--git-push`, findings advisory vs blocking).

**Dependências:** Fase 1 depende de `goreleaser-action@v7` + `~> v2`
(terceiro action pinado, como os demais); Fases 2–3 dependem só do toolchain
Go 1.26.6 + golangci-lint já usados; Fase 5 depende de Tailscale
(`vps_oracle_2`) + Docker/KVM local.

**Não-regressão:** `go build ./...`, `go vet ./...`, `go test ./...`,
`golangci-lint run --new-from-rev=origin/main` e `envctl-verify --git-push`
verdes em **cada PR**; `gh release view` com 11 assets após a Fase 1;
`doctor` 0 WARN/0 ERROR nas máquinas de teste após a Fase 5.

## 3. Fase 1 — CI/CD: `release.yml` → GoReleaser

**Objetivo:** trocar ~51 linhas de shell imperativo (4 `go build` + zip/tar +
sha256 + `gh release upload` — `release.yml:63-115`) por ~30 linhas
declarativas (`.goreleaser.yml`) + 1 step, testável local com
`goreleaser release --snapshot --clean`.

**Contrato inegociável — nomes dos assets** (quem baixa é o bootstrap, por URL
de nome fixo):

| asset atual | como produzir no goreleaser |
|---|---|
| `envctl-linux-amd64.tar.gz` / `-arm64` | `archives` com `name_template: '{{ .ProjectName }}-{{ .Os }}-{{ .Arch }}'`, formato `tar.gz` |
| `envctl-windows-amd64.zip` / `-arm64` | `format_overrides: [goos: windows → zip]`, mesmo `name_template` |
| `envctl-linux-amd64` / `.exe` soltos | `archives` adicional `format: binary` (mesmo `name_template`) |
| `bootstrap.sh` / `bootstrap.ps1` | `extra_files` nos archives (dentro do tar/zip) **e** `files:` globais p/ attach solto |
| `SHA256SUMS.txt` | `checksum.name_template` |

**Integração:** release-please continua dono da tag/release; o job
`attach-assets` chama o goreleaser via `workflow_call` com `mode: keep-existing`
(notes do bot preservados) e `make_latest: false` (bot já marcou). `goreleaser-action@v7`,
`fetch-depth: 0`, `version: '~> v2'`.

**Tarefas (uma entrega verificável por item):**

- [x] **T1 — `.goreleaser.yml` criado.** (feito: `36c355b`)
      Arquivos: criar `.goreleaser.yml`; ler `cmd/envctl/main.go:9`
      (ldflags `main.Version`), `.github/workflows/release.yml:66-87`
      (4 targets atuais).
      Interfaces: `builds[0] = {id: envctl, main: ./cmd/envctl, binary: envctl,
      goos: [linux, windows], goarch: [amd64, arm64], CGO_ENABLED=0,
      ldflags: [-s -w -X main.Version={{ .Version }}]}`;
      `archives[0] = {name_template: '{{ .ProjectName }}-{{ .Os }}-{{ .Arch }}',
      formats: [tar.gz], format_overrides: [windows→zip], files: [bootstrap.sh,
      bootstrap.ps1]}`; `checksum.name_template: SHA256SUMS.txt`;
      `release: {mode: keep-existing, make_latest: false}`.
      Passos: escrever o yaml; rodar `goreleaser check`.
      TDD: RED = `goreleaser check` falha ("no .goreleaser.yml") antes;
      GREEN = passa depois.
      Verificação: `goreleaser check`.
      Rollback: `git checkout -- .goreleaser.yml` (arquivo novo, sem efeito).
      Resultado: `goreleaser check` exit 0.
- [x] **T2 — `release.yml` reescrito (4 targets → 1 step).** (feito: `36c355b`)
      Arquivos: modificar `.github/workflows/release.yml` (manter `on:`,
      `inputs.version`, `get_version`; trocar os steps Build/Checksum/Upload
      por `goreleaser/goreleaser-action@v7` com `args: release --clean`;
      **adicionar `fetch-depth: 0` no checkout** — hoje é shallow (default 1) e
      o goreleaser precisa de tags+histórico).
      Colapsar o `Determine Version` 3-way (`release.yml:46-61`) para
      `${{ inputs.version || github.event.release.tag_name }}` + o guard
      `gh release view` (os dois callers reais sempre fornecem: bot passa
      `tag_name`, `release: published` sempre tem `tag_name`).
      Passos: editar; `actionlint` local (se instalado) ou `bash -n` nos runs.
      Verificação: `./actionlint -color` (mesmo gate do CI).
      Rollback: `git checkout -- .github/workflows/release.yml`.
      Resultado: diff mostra −~50/+~15 linhas, triggers intactos.
- [x] **T3 — snapshot local com paridade de nomes.** (feito: `36c355b`)
      Arquivos: nenhum (artefato em `dist/`, ignorado).
      Comando: `goreleaser release --snapshot --clean` +
      `ls dist/ | sort` comparado por script contra a lista da v1.13.0
      (`envctl-linux-amd64[.tar.gz]`, `envctl-linux-arm64[.tar.gz]`,
      `envctl-windows-amd64[.exe|.zip]`, `envctl-windows-arm64[.exe|.zip]`,
      `SHA256SUMS.txt`, `bootstrap.sh`, `bootstrap.ps1` = 11 assets).
      Verificação: diff vazio entre as duas listas.
      Rollback: `rm -rf dist/` (não versionado).
      Resultado: 11/11 nomes idênticos.
- [ ] **T4 — PR + merge na janela de release.**
      Arquivos: os dois acima. Comando: `gh pr create --base main` →
      checks verdes → merge commit → aguardar próxima release do bot →
      `gh release view vX.Y.Z --json assets --jq '.assets[].name'`.
      Verificação: 11 assets + `bootstrap.sh` instalando da release nova
      (`curl` do tarball + `./envctl --version`).
      Rollback: revert do merge; tag antiga intacta.
      Resultado: release publicada pelo goreleaser, bootstraps ok.

**Riscos:** (a) nomes divergentes → bootstrap quebra na instalação de máquina
nova — mitigação: T3 antes do PR;
(b) goreleaser cria release em vez de anexar → `mode: keep-existing` + rodar
encadeado pelo `workflow_call` (nunca no `release: published`).
**Rollback:** revert do commit; `release.yml` manual volta a valer; tag antiga
permanece intacta (goreleaser só anexa).
**Breaking changes:** nenhuma para o consumidor (nomes preservados por contrato).

## 3b. Fase 1b — CI: gate `golangci` total (fim do `only-new-issues`)

**Objetivo:** remover a exceção de dívida legada do `ci.yml:71`
(`only-new-issues: true`) e travar o gate cheio. Medido hoje:
**59 findings** totais (35 errcheck, 15 gosec, 4 staticcheck, 2 unparam,
2 nilerr, 1 ineffassign) — caiu de 68, cabe em 1–2 PRs.

**Tarefas:**

- [x] **T1 — zerar os 59.** (feito: `33d799b`) Arquivos: os apontados pelo
      `golangci-lint run ./...` completo (maioria errcheck em paths
      Windows/SSH + gosec G304/G104 já parcialmente excluídos no
      `.golangci.yml`).
      Passos: `golangci-lint run ./...` → corrigir por linter
      (errcheck: tratar retornos; gosec: `//nolint` com motivo onde o
      padrão do provisioner exige path variável, como já faz o G204 em
      `provision_providers.go:449`).
      TDD: RED = lista de 59 antes; GREEN = lista vazia depois
      (não há comportamento novo — o "teste" é o próprio linter).
      Verificação: `golangci-lint run ./...` exit 0.
      Rollback: revert por PR. Resultado: 0 findings.
- [x] **T2 — remover a exceção.** (feito: `33d799b`) Arquivo: `.github/workflows/ci.yml`
      (deletar `only-new-issues: true` + o comentário de dívida).
      Verificação: `./actionlint -color` + CI verde no PR.
      Rollback: revert. Resultado: gate cheio em todo PR futuro.

**Riscos:** falso-positivo de linter em path Windows-only não testável no
Linux — mitigação: `//nolint:<linter>` com motivo (precedente G204 já
aceito no repo), nunca `exclude-rules` genérica nova.
**Breaking changes:** nenhuma (só o gate fica mais estrito).

## 3c. Fase 1c — CI hygiene (sem risco de release, 1 PR)

**Objetivo:** travar reprodutibilidade e remover steps redundantes do CI sem
tocar release. Arquivos: `.github/workflows/ci.yml`, `Makefile`,
`configs/bin/envctl-verify` (só comentário de paridade).

**Tarefas:**

- [x] **T1 — pins reprodutíveis.** (feito: `60dd152`) `setup-go`: `go-version: 'stable'` →
      `go-version-file: go.mod` (3 spots: `ci.yml:63-64,96-97`,
      `release.yml:40-41`; `go.mod:3` manda `1.26.6`); dropar `cache: true`
      (é o default do setup-go p/ módulos). `golangci-lint-action`:
      `version: latest` → `v2.x` pinada (dependabot já vigia
      `github-actions`). Verificação: CI verde; `stable` nunca mais flutua o
      build. Rollback: revert. Resultado: toolchain e linter determinísticos.
- [x] **T2 — actionlint via action.** (feito: `60dd152`) Trocar `ci.yml:46`
      (`bash <(curl …download-actionlint.bash) 1.7.12` + binário untracked no
      workspace, versão que o dependabot nunca bumpa) por
      `rhysd/actionlint-github-action` pinada. Verificação: gate actionlint
      verde (inclui shellcheck dos `run:`, como comenta `:44-45`).
      Rollback: revert. Resultado: −2 shells manuais.
- [x] **T3 — dropar steps subsumidos.** (feito: `60dd152`) `go mod download`
      (`ci.yml:99-100`, `release.yml:43-44`) é warm-up no-op (vet/test/build
      usam o module cache do setup-go) — deletar, mantendo o gate real
      `tidy + git diff --exit-code` (`ci.yml:66-69`). `Build Executable`
      (`ci.yml:110-111`) só re-prova o link que vet+test já compilaram (a
      release prova o link multi-OS) — deletar (~30-60s por leg).
      Verificação: CI verde. Rollback: revert. Resultado: CI mais rápido,
      mesmo gate.
- [x] **T4 — `paths` → `paths-ignore`.** (feito: `60dd152`) O filter atual (`ci.yml:14-20`) não
      dispara p/ `.golangci.yml`, `Makefile`, `bootstrap.sh/ps1` — justo os
      arquivos que quebram o CI. Inverter p/
      `paths-ignore: ['docs/**','spec-agent/**','CHANGELOG.md','**.md']`:
      PR docs-only pula, edição em lint-config/Makefile/bootstrap gateia.
      Verificação: PR só-docs pula; PR Makefile roda. Rollback: revert.
- [x] **T5 — Makefile dev-only.** (feito: `60dd152`) Deletar `make build-windows`
      (`Makefile:11-12` gera ELF Linux com nome `.exe` sem `GOOS=windows` —
      o mesmo smell do Taskfile removido no PR #71; a release cross-compila);
      `make lint` → `golangci-lint run --new-from-rev=origin/main ./...`
      (hoje o bare `run ./...` mostra dívida que o CI nunca reporta —
      alinhar ao comando de não-regressão do §2b); `make
      doctor/doctor-fix/run-all/snapshot` → `go run ./cmd/envctl <args>`
      (hoje churnam o artefato `./envctl`, gitignored mas suja a árvore;
      `make clean` deixa de precisar do caso especial).
      Verificação: `make lint` == veredito do CI; árvore limpa após
      `make doctor`. Rollback: revert. Resultado: dev e CI falam igual.
- [x] **T6 — paridade CI ↔ `envctl-verify`.** (feito: `60dd152`) O script espelha o gate Go
      (`envctl-verify:382-395`: build/vet/test + `GOOS=windows` + `golangci
      --new-from-rev`) manualmente — pode driftar (vide T5). Adicionar step
      no CI rodando `configs/bin/envctl-verify --git-push` como sinal de
      paridade local, ou no mínimo comentário nos dois arquivos apontando o
      espelho. Verificação: comentário/step presente. Resultado: nunca mais
      "CI verde, hook vermelho".

**Nota (sem tarefa):** pins tag-only sem SHA (`checkout@v7` ×4, `setup-go@v7`
×3, `golangci@v9`, `release-please@v5`) + dependabot semanal = stay-put
decidido; SHA-pinning dobra o churn de bump p/ este tamanho de repo.
Registrado p/ ninguém "otimizar" sem contexto.

**Riscos:** `go-version-file` expor drift se o runner não tiver a toolchain —
mitigação: setup-go instala a toolchain do `go.mod` automaticamente.
**Breaking changes:** nenhuma (só CI/dev-local).

## 3d. Fase 1d — bootstraps: arm64 + contrato único

**Objetivo:** corrigir o BUG `bootstrap.ps1:36-40` (rejeita não-amd64 enquanto
a release entrega `envctl-windows-arm64.{exe,zip}`) e travar o contrato
sh↔ps1 p/ a classe de drift não voltar.

**Tarefas:**

- [x] **T1 — ps1 aprende arm64.** (feito: `80588e2`) Espelhar o mapa de arch do
      `bootstrap.sh:24-31` (`x86_64→amd64`, `aarch64|arm64→arm64`) +
      interpolação de URL/padrão em `bootstrap.ps1:77,93` (hoje hardcoded
      `envctl-windows-amd64.zip`).
      Verificação: regressão amd64 (`bootstrap.ps1` instala da release atual)
      + interpolação arm64 revisada; ciclo arm64 real quando houver HW.
      Rollback: revert. Resultado: Windows ARM instala; 11-asset contract
      vale nos dois scripts.
- [x] **T2 — contrato espelhado.** (feito: `80588e2`) Bloco de comentário idêntico nos dois
      scripts (`sh:49-106` vs `ps1:71-134` têm a mesma escada gh CLI →
      HTTPS direto (+`GITHUB_TOKEN`) → `go build` + atalho de binário local +
      PATH ensure): REPO, asset templates, `VERSION=latest` env/param, ordem
      dos tiers. Verificação: diff dos blocos vazio (a menos de sintaxe de
      comentário). Rollback: revert. Resultado: um script nunca mais aprende
      arch que o outro não tem.
- [ ] **T3 — validar contra a release do goreleaser.** Fase 1-T3/T4 com os
      dois bootstraps: `bootstrap.sh` (amd64+arm64) e `bootstrap.ps1`
      (amd64+arm64) instalando da release nova + `./envctl --version`.
      Rollback: n/a (validação). Resultado: Fase 1 fecha com os dois OSes.

**Riscos:** testar arm64 sem HW — mitigação: interpolação provada por leitura
+ regressão amd64; ciclo arm64 real fica como pendência registrada.
**Breaking changes:** nenhuma (amd64 byte-idêntico).

## 4. Fase 2 — Código: interfaces 1:1 → concretos (sem quebrar Clean Arch)

**Objetivo:** cortar a indireção que só custa navegação de agente LLM, mantendo
`PackageManager` (polimorfismo real, 8 impls) e `Logger`.

**Nota TDD:** esta fase é refatoração pura (zero mudança de comportamento), então
o ciclo é invertido e declarado: BASELINE = `go test ./...` verde **antes**;
cada PR termina com o **mesmo** `go test ./...` verde **depois**. Qualquer RED
no depois é regressão e bloqueia o merge — não existe "teste novo" porque nenhum
comportamento novo nasce aqui. O gate real do repo
(`go build ./... && go vet ./... && go test ./... &&
golangci-lint run --new-from-rev=origin/main`) é a verificação de cada PR.

**Tarefas (um PR por domínio, sequenciais — mesma área do código):**

- [x] **T1 — `fix/concrete-fs`: `FileSystemManager` → concreto.** (feito pelo outro agente em paralelo; gate verde)
      Arquivos: `internal/domain/repository/interfaces.go` (remover a
      interface), `internal/infra/filesystem/fs_manager.go` (o struct
      `FileSystemManager` vira o tipo canônico), todos os construtores de
      usecase que hoje recebem `repository.FileSystemManager`
      (grep: `provision_shell.go`, `provision_skills.go`, `snapshot_sync.go`
      + testes `*_test.go` correspondentes), `internal/ui/cli/root.go` (wire).
      Interfaces: o struct expõe os mesmos métodos; nenhuma assinatura muda,
      só o tipo nominal no parâmetro.
      Verificação: gate real do repo. Rollback: revert do PR.
      Resultado: zero referência a `repository.FileSystemManager` (`grep -rn`
      vazio fora da definição removida).
- [x] **T2 — `fix/concrete-manifest`: `ManifestRepository` → concreto.** (feito pelo outro agente em paralelo; gate verde)
      Arquivos: `interfaces.go` + `internal/infra/embedded/*` (impl única) +
      consumidores (`provision_providers.go:manifestNodeSpec`,
      `provision_packages.go`, `provision_lsp.go` + testes).
      Verificação/rollback/resultado: idem T1 para `ManifestRepository`.
- [x] **T3 — `fix/concrete-git-windows`: `GitManager`, `WindowsEnvManager`,
      `WindowsTweaksManager`, `PackageRemover` → concretos.** (feito pelo outro agente em paralelo; gate verde)
      Arquivos: `interfaces.go` + `internal/infra/git/*`,
      `internal/infra/windows/*` + consumidores + testes.
      Verificação/rollback/resultado: idem, por nome.
- [x] **T4 — `fix/concrete-perf`: ~9 interfaces de
      `internal/domain/repository/performance.go` → concretos.** (feito 2026-10-06: 9 structs exportados, `performance.go` deletado, `auditLinuxPerformance` recebe snapshot — sem seam de teste; gate cheio verde)
      Arquivos: `performance.go` + `internal/infra/performance/*` +
      `internal/usecase/performance_options.go`,
      `sysctl_intent_audit.go`, `doctor_linux_performance.go`,
      `provision_performance.go` + testes.
      Verificação/rollback/resultado: idem; `repository/performance.go`
      deletado ou reduzido a zero interfaces.

**Riscos:** quebrar wire do `root.go` (mitigação: um PR por domínio, compila é o
critério); perder seam de teste — **falso risco**: os testes hoje injetam por
interfaces locais (`UpdateEnv`, `latestVersionFn`), não pelas do repository.
**Rollback:** revert por PR.
**Breaking changes:** nenhuma (interno; CLI/manifests/configs intactos).

## 4b. Fase 2b — `run.go` table-driven (mata ~350 linhas de boilerplate)

**Objetivo:** `internal/ui/cli/run.go` tem 748 linhas para 19 subcomandos, dos
quais ~15 são o mesmo corpo de 6 linhas (`PrintBanner()` +
`runXProvisioning()`). Só `performance` tem flags e só `all`/`all-bare` têm
lógica de dispatch. A string de erro da linha 34 lista os subsistemas
**manualmente** — adicionar um subsistema exige editar 2 lugares (fonte clássica
de drift; hoje a lista já diverge do `Use:` real se alguém esquecer).

**Tarefas:**

- [x] **T1 — tabela + loop.** (feito 2026-10-06: 16 triviais em `runTargets`, 5 explícitos; help byte-idêntico provado por diff; gate verde)
      Interfaces: `type runTarget struct { name, short string; run func(cmd
      *cobra.Command, args []string) error }` + `var runTargets = []runTarget{
      {"winget", "Provision Winget…", func…{ PrintBanner();
      runPackagesProvisioning(entity.PackageTypeWinget); return nil }}, …}`.
      O loop `for _, t := range runTargets { cmd.AddCommand(…Use: t.name,
      Short: t.short, RunE: t.run) }` gera os 15 simples; `all`, `windows`,
      `vps`, `cachyos`, `performance` (com flags), `debloat`, `providers`,
      `bootstrap` continuam explícitos (têm lógica própria).
      A string de erro do `RunE` raiz deriva de `runTargets` (+ os explícitos)
      — nunca mais manual.
      TDD: RED = `go test ./internal/ui/cli/ -run TestRunHelp` (teste novo,
      ver abaixo) falha antes (help lista X nomes, tabela lista Y);
      GREEN = passa depois.
      Verificação: `go run ./cmd/envctl run --help` byte-idêntico ao antes
      (diff da saída) + `go run ./cmd/envctl run <subsistema-inexistente>`
      mostra a lista completa derivada + gate cheio.
      Rollback: revert do PR. Resultado: `run.go` ~748→~400 linhas, help
      idêntico.
- [x] **T2 — teste de contrato do help.** (feito 2026-10-06: `run_help_test.go` — erro via dispatch aninhado como produção, lista≡subcomandos; help coberto; gate verde)
      `internal/ui/cli/run_help_test.go`: asserts que todo nome na mensagem de
      erro existe como subcomando registrado e vice-versa (trava o drift para
      sempre).
      Verificação: `go test ./internal/ui/cli/ -v`. Rollback: revert junto.
      Resultado: adicionar subsistema sem registrar quebra o teste.

**Riscos:** mudar texto de help que script externo parseia — mitigação: T1
exige saída byte-idêntica.
**Breaking changes:** nenhuma (help idêntico por contrato testado).

## 4c. Fase 2c — `doctor_audit.go` split (o maior arquivo do repo)

**Objetivo:** `internal/usecase/doctor_audit.go` tem **2227 linhas**; o
`Execute` sozinho vai da linha 92 à ~1310 (~1218 linhas, até
`auditGamingStack:1311`) com ~50 sites `addDiag` inline. É o inverso da Fase 3
(aqui é split, não merge):
cada domínio vira `doctor_<domínio>.go` com os métodos já existentes
(`auditGamingStack`, `auditDebloat`, `auditLSPHandshake`,
`auditCommandCodeAgents`, `auditSkillTree`, `auditEnvctlFreshness`,
`auditOpenCodeFileRefs`, `auditRemovedMCPEntries`, …) movidos, e o `Execute`
vira ~40 linhas só de despacho.

**Tarefas (um PR por grupo, moves puros):**

- [x] **T1 — `doctor_packages.go`:** moves das seções de pacotes/toolchain. (feito 2026-10-06 com T2–T4 em passe único: `doctor_audit.go` 2135→168 linhas, Execute só despacha; inventário 43 decls 1:1 + 18 métodos novos; doctor ao vivo 237 checks; gate verde. Variância honesta: packages 408/agents 525/system 570/gaming 623 — seções medidas maiores que a estimativa; novo split só com motivo)
- [x] **T2 — `doctor_agents.go`:** OpenCode/CommandCode/skills/MCP/agents. (feito junto — ver T1)
- [x] **T3 — `doctor_system.go`:** shell/git/worktree/temp/envctl-freshness. (feito junto — ver T1)
- [x] **T4 — `doctor_gaming.go`:** `auditGamingStack` + `auditGamingTuning` + helpers + `auditDebloat`. (feito junto — ver T1; único ajuste semântico: `configFiles` do escopo do Execute virou load local em `auditOpenCodeMCPRefs`, mesmo conteúdo determinístico)
      helpers puros (`amdgpuModulePresent`, `pendingPacnewFiles`,
      `missingGamingConfKeys`, `kwinrcCompositingEnabled`, `configHasKey`,
      `missingCmdlineParams`, `multilibEnabled`) + `auditDebloat` (Windows;
      vai junto por ser "stack opcional auditada" como gaming).
      Cada PR: `git diff --stat` mostra só renames + o `Execute` encolhendo;
      `go test ./internal/usecase/ -run TestDoctor -v` + gate cheio.
      Rollback: revert por PR. Resultado: nenhum arquivo >400 linhas no
      usecase; `Execute` só despacha.

**Riscos:** mover função com dependência não vista (mitigação: compila é o
critério; moves puros não mudam assinatura).
**Breaking changes:** nenhuma.

## 4d. Fase 2d — `opencode.json` único (mata ~810 linhas duplicadas)

**Objetivo:** `configs/opencode.json` (814 linhas) e
`configs/opencode.linux.json` (813 linhas) diferem em **1 única linha**
(`"shell": "pwsh"`, só no Windows). Hoje quem edita um e esquece o outro cria
drift silencioso entre OSes — o risco #1 de config do repo. O deploy já é por
OS (`manifests/shell.yaml:30-44`: `opencode_config` → `os: windows`,
`opencode_config_linux` → `os: arch,cachyos,debian,ubuntu`).

**Tarefas:**

- [x] **T1 — base única + overlay.** (feito 2026-10-06, opção B: `configs/opencode.json` sem `shell` + `withWindowsShellOverlay` no deploy da entrada `opencode_config`; linux byte-idêntico ao antigo; teste de overlay verde; gate verde). Detalhe original:
      `configs/opencode.linux.json`; `configs/opencode.json` vira a base
      (sem `shell`); `shell.yaml`: entrada Windows ganha aplicação do overlay
      (opção A, preferida: reaproveitar o modo `merge:` — verificar se
      `json_deps`/`markdown_sections` cobre patch de chave única; se não,
      opção B: o provisioner injeta `"shell": "pwsh"` quando `os == windows`).
      Decidir A vs B lendo `config_merge.go:316` antes de codar (é o unknown
      técnico; dono: quem implementar).
      TDD: RED = `TestShippedOpenCodeTemplatesHaveNativeShape` +
      `TestOpenCodeConfigTemplates` falham com a base sem `shell` no Windows;
      GREEN = passam com o overlay aplicado (o teste valida o **deployado**,
      não o arquivo-fonte).
      Verificação: `envctl opencode` numa máquina Windows e numa Linux +
      `diff` do JSON deployado contra o atual (só a chave `shell` difere por
      OS) + gate cheio.
      Rollback: revert (os dois arquivos voltam). Resultado: −813 linhas,
      drift impossível por construção.
- [ ] **T2 — AGENTS: manter como está — COM RESSALVA.** `configs/AGENTS.arch.md`
      (58) vs `.linux.md` (54) vs `.md` (48): ~65% é boilerplate (`## Regras`
      ~30 linhas quase idênticas), diferenças reais só em `## Ambiente`
      (OS/shell/CLIs/scratch/git/gaming/editor), `## Serviços` (só linux/arch)
      e worktree (só arch). **Não unificar agora** (o template gerenciado muda
      por OS de verdade), mas registrado: quem unificar depois usa o mesmo
      padrão base+overlay da T1. Vale p/ `configs/commandcode/AGENTS*.md`
      (53/60/61, ~85% boilerplate — ver Fase 2i). (Registrado para
      ninguém "otimizar" depois sem ler o diff.)

**Riscos:** o modo `merge:` não cobrir o caso (mitigação: opção B, 10 linhas
no provisioner, com teste).
**Breaking changes:** nenhuma (JSON deployado idêntico por OS).

## 4e. Fase 2e — helper único de PATH em shell (3 call sites → 1)

**Objetivo:** "adicione X ao PATH no bash+fish" existe em **3 call sites de
shell-script** (`provision_providers.go:156-168` `openCodePathInstaller` —
usado em `:433` e `provision_bootstrap.go:330`,
`provision_bootstrap.go:407-422,709-732` Volta/Go/OpenCode) + o dono Go
`env_manager.go:354-408` (`EnsurePathEntry`, o bom — `provision_shell.go:89`
já o usa). Cada script reimplementa o branch bash (`export PATH=`) vs fish
(`set -gx PATH`) + o `grep -Fq` idempotente. (O mesmo padrão de "PATH de
toolchain montado à mão" existe em Go em 4 lugares — ver Fase 2g-A.)

**Tarefas:**

- [ ] **T1 — `ensureShellPathEntry(dir)`.** (desenho 2026-10-06, aguardando: função canônica com backup atômico + guard substring — compatível com linhas legadas se chamada com forma `$HOME/...`; go exige 2 chamadas; VOLTA_HOME fica no script; providers/bootstrap não têm envManager — exige wire no `root.go`; reestruturar cada call site preservando diags)
      `internal/infra/environment/env_manager.go` (já é o dono de
      persistência): 1 função Go que garante a linha em `.bashrc`,
      `.profile` e `config.fish` (idempotente, com backup atômico do repo).
      Migrar os 4 call sites; os `const` de shell-script viram chamadas.
      Verificação: `go test ./internal/infra/environment/ -v` (teste novo:
      aplica 2x → 1 linha por arquivo) + gate cheio.
      Rollback: revert do PR. Resultado: 1 implementação, 4 usos.

**Riscos:** diferença sutil de quoting entre os scripts atuais (mitigação:
teste com os conteúdos reais atuais como fixture).
**Breaking changes:** nenhuma (linhas resultantes idênticas).

## 4f. Fase 2f — pacman+paru unificados (~60 linhas)

**Objetivo:** `pacman_manager.go` (`-S --noconfirm --needed` + fallback
`sudo -n`) e `paru_manager.go` (`-S --noconfirm --needed --skipreview`, sem
sudo) são o mesmo manager com 2 diferenças. apt (`install -y
--no-install-recommends`) e winget têm semântica própria — **ficam**.

**Tarefas:**

- [x] **T1 — `archManager{bin string, skipReview, useSudo bool}`.** (feito 2026-10-06: `internal/infra/arch/` com struct parametrizada por dados — bin/queryBin/types/listArgs/skipReview/useSudo cobrem também `ListInstalled` (`-Q` vs `-Qm`) além dos 2 diffs da spec; `pacman/`+`paru/` deletados, construtores e wire preservados — só imports do `root.go` mudaram; testes de Type movidos; gate verde). Detalhe original:
      Arquivos: `internal/infra/pacman/` + `internal/infra/paru/` →
      1 struct parametrizada (2 construtores `NewPacmanManager()` /
      `NewParuManager()` preservados — o wire em `root.go:103-104` não muda).
      Verificação: `go test ./internal/infra/pacman/ ./internal/infra/paru/ -v`
      + gate cheio. Rollback: revert. Resultado: −~60 linhas.

**Riscos:** `--skipreview` no pacman por engano (mitigação: teste existente de
cada manager cobre as flags — rodar antes e depois).
**Breaking changes:** nenhuma.

## 4g. Fase 2g — helpers Go compartilhados (exec/version/backup/diag)

**Objetivo:** matar 7 grupos de duplicação de helper espalhados por
usecase+infra, sem mudar comportamento. Fazer **antes** da Fase 2h (os dedupes
usam esses helpers) e **primeiro o H** (encolhe todos os diffs seguintes).

**Tarefas (um PR por letra, ou agrupar A–C + D–F + H se o revisor preferir
PRs maiores):**

- [x] **T1-A — toolchain PATH ×4 → 1.** (feito 2026-10-06: `executil.{ToolchainDirs,ToolchainPath,ToolchainEnv,ExecTool}`; 5 call sites viram wrappers; `execTool`/`lookPathWithEnv` deletados; lista única inclui `~/.opencode/bin` — iguala probe≡execução do `runWithToolchain`; gate verde). A lista unificada era esta:
      (`~/.opencode/bin, ~/.local/bin, ~/.volta/bin, /usr/local/go/bin,
      ~/go/bin`) é montada em `provision_bootstrap.go:54-74`
      (`linuxToolchainEnv`), `:48-50` (`shellEnv`), `:94-114`
      (`ensureProcessToolchainPath`), `provision_providers.go:62-68`
      (`toolchainEnv`) e `toolchain_managers.go:22-50` (`execTool`, mesma
      lista menos `~/.opencode/bin`, rewrite manual `:38-43`) com subsets
      diferentes. Criar `executil.ToolchainEnv(home) []string` +
      `executil.ExecTool(ctx,name,args…)`; os 4 viram wrappers finos.
      Verificação: `go test ./internal/usecase/ ./internal/infra/toolchain/`
      + gate cheio. Resultado: ~−60 linhas + fim do subset-drift.
- [x] **T1-B — LookPath ×2 → 1.** (feito 2026-10-06: `executil.LookPathIn(path,name)` com semântica providers + `IsExecutableFile` windows-aware; `toolAvailable` sem spawn; `file_checks.go` deletado; gate verde). Fundiam-se:
      (`lookPathInEnv`) vs `toolchain_managers.go:54-70`
      (`lookPathWithEnv`) = mesmo loop SplitList+Stat+exec-bit. Fundir em
      `executil.LookPathIn(path,name)`; `toolAvailable`
      (`provision_bootstrap.go:79-89`, que spawna `bash -lc "command -v"`)
      passa a usar o helper (~−10 linhas + 1 spawn por probe).
      Verificação/rollback: idem.
- [x] **T1-C — `ProbeCheckCommand` bypasses.** (feito 2026-10-06: `ProbeCheckCommand` resolve via `ExecTool`; preâmbulos npm/go/volta colapsados; apt/pacman/paru/winget/pip ganham resolução toolchain — superset, veredito idêntico no PATH real; gate verde). Antes:
      `executil.go:10-22` existia e
      pacman/paru/pip usam, mas `NpmManager.IsInstalled`
      (`toolchain_managers.go:88-95`) e `GoManager.IsInstalled` (`:282-308`,
      com spawn `bash -lc command -v` no Linux `:302-304` e `where.exe` no
      Windows `:297-300`) re-splitam `CheckCommand` à mão. Unificar o
      preâmbulo de todos os `IsInstalled` (~−20 linhas).
- [x] **T1-D — `version.go` único (~150 linhas espalhadas).** (feito 2026-10-06: `internal/usecase/version.go` com 18 funcs + `latestFromJSON` novo; tabela de teste fundida; gate verde). Fontes unificadas:
      `provision_providers.go:170-178` (`versionMajorAtLeast`), `:598-603`
      (`printableVersion`, usado também em `doctor_audit.go:1850` e
      `provision_bootstrap.go:257,279,290`), `:607-633`+`:781-794`
      (`installedVersion`, `installSource`, `classifyInstallSource`),
      `:636-691` (`npmLatest`/`defaultLatestProviderVersion` — o mesmo parse
      de substring `"version":"` 2×), `:654-667` (`firstVersionToken`),
      `:698-767` (`openCodeWithinMajorUpdateNeeded`+`parseSemver`,
      `versionsDiffer`+`normalizeVersion`); `provision_bootstrap.go:342-348`
      (`toolVersion`, 3º wrapper de `firstVersionToken`), `:674-696`
      (`fzfSupportsWalker`+`fzfHasWalker`, 3º compare semver);
      `update.go:62-64` (`isProviderRuntime`), `update_env.go:40-75`
      (`uvToolVersionOf`+`goLatestOf`). Criar `internal/usecase/version.go`
      (`parseSemver/majorAtLeast/withinMajor/versionsDiffer/firstToken/latestFromJSON`);
      `fzfHasWalker` e `openCodeWithinMajorUpdateNeeded` viram callers de
      3 linhas. Dedupar a tabela de teste (`update_test.go:66-80` +
      `provision_providers_test.go:47-48` testam o mesmo `versionsDiffer`).
- [x] **T1-E — backup ×3 → 1.** (feito 2026-10-06: `fs.{BackupPathFor,ArchivePath,storeWithBackup}`; `WriteWithBackup` e `CopyEmbeddedTree` no mesmo núcleo — backup preserva modo live na árvore; `archiveUserOpenCode` deletado; gate verde). Núcleos unificados:
      `provision_providers.go:560-579` (`archiveUserOpenCode` — mesmo loop
      stamp+`-1,-2`, depois `Rename` em vez de retornar path) vs
      `fs_manager.go:165-199`+`:292-303` (`WriteWithBackup` vs
      `CopyEmbeddedTree`, mesmo read-compare-backup-write com fallback de
      mode diferente). Expor `fs.BackupPathFor` + `fs.ArchivePath(path)`,
      `CopyEmbeddedTree` chama o núcleo de `WriteWithBackup` (~−40 linhas).
- [x] **T1-F — `psQuote` admitido.** (feito 2026-10-06: `executil.PSQuote`, duplicata no `tweaks_manager.go` removida, teste usa o canon; gate verde). Antes:
      *"mirrors environment.psQuote … duplicated to avoid an infra→infra
      import for 3 lines"* vs `env_manager.go:23-26`. Mover p/
      `executil.PSQuote` (ou `infra/internal/psquote`); 0 comportamento,
      mata o vetor de drift que a Fase 2-T3 senão cimentaria no concreto.
- [x] **T1-H — ctors `entity.Diagnostic` (~60 sites, −4 linhas cada).** (feito 2026-10-06: real eram 275 sites; `entity.{OK,Info,Warn,Error}` em `diagnostic.go` novo, 269 migrados, 6 de categoria computada mantidos literais, testes-intencionalmente literais; gate cheio verde)
      `grep entity.Diagnostic{ internal/usecase` → 60+ sites (doctor ~50,
      bootstrap ~15, `doctor_linux_performance.go:23-184` 15× mesma forma,
      `provision_shell.go:152-445` 12×). Adicionar
      `diag.OK/Warn(+fixHint)/Info(…)` (ou métodos em
      `*AuditReport`/`*BootstrapResult`); texto idêntico, só boilerplate
      some (~−300 linhas latentes). **Fazer primeiro** — encolhe os diffs de
      todas as fases seguintes.

**Riscos:** helper compartilhado com semântica sutilmente diferente por caller
(mitigação: teste com fixtures dos conteúdos reais atuais antes de migrar
cada call site).
**Breaking changes:** nenhuma.

## 4h. Fase 2h — dedupes usecase/CLI/testes (usa a 2g)

**Objetivo:** fundir lógicas duplicadas de domínio e podar código morto/teste
inchado, preservando comportamento. Depende da 2g (helpers prontos) e da 2c
(splits do doctor antes desses moves).

**Tarefas:**

- [x] **T1 — `step` vs `configStep`: split falso.** (feito 2026-10-06: núcleo `ensureStep(target, installed(), script)` + wrappers finos `step`/`configStep` com textos honestos por tipo — diags byte-idênticos, log passa a espelhar o diag (2 expects ajustados); `nameAlreadyPresent` + comentário mortos deletados; gate verde). Detalhe original:
      `provision_bootstrap.go:117-215` (`configStep:148` + comentário de 60
      linhas explicando por que `step` "não modelava configs" +
      `nameAlreadyPresent:174` retornando a constante `"config"` p/ 1 caller).
      Fix: `step(ctx,result,target,installed func()bool,script string)` —
      `hasTool(name)` e `doneCheck` são ambos predicados `installed`; deleta
      `configStep`+`nameAlreadyPresent` (~−40 linhas) e o comentário mentiroso.
      (Forma-irmã já existe em `provision_packages.go:99-197`
      `provisionListMode` — a longo prazo, `step` do bootstrap é aquela
      função especializada p/ shell scripts.)
- [x] **T2 — OpenCode-V2 ensure ×2 → 1.** (feito 2026-10-06: loop de archive convergente extraído p/ `archiveShadowedUserCopies` em `version.go` — decisão "o que é V2" já era compartilhada via `version.go`; fusão total recusada com motivo: diags/logs têm sistema e hints distintos por comando e unificá-los mudaria saída visível; textos 100% preservados; gate verde). Detalhe original:
      `provision_bootstrap.go:221-292` (`ensureOpenCodeV2`, ~70 linhas)
      duplica a fase 0 em `provision_providers.go:300-530` +
      `requireStandaloneProviderVersion:584-596` (ambos chamam
      `installSource`, `archiveUserOpenCode`, `versionMajorAtLeast`,
      `openCodePathInstaller`, `toolVersion`/`installedVersion`). Extrair
      `ensureOpenCodeV2(ctx,result/add,pacmanMgr)` compartilhado; providers +
      bootstrap viram callers (~−50 linhas + os dois paths nunca mais
      discordam sobre o que é "V2 instalado").
- [x] **T3 — temp-prune com 2 donos → 1.** (feito 2026-10-06: step 4 do `CleanupOpenCodeUseCase` deletado — `TempHygieneUC` dono único (já rodava em sequência no `runCleanup`); `tempOwner`×`classifyTempEntry` já documentado no código; gate verde)
      (`TempHygieneUseCase.Cleanup`, owned, testado) vs
      `cleanup_opencode.go:111-140` (step 4, walk ad-hoc de `/temp`|`C:\temp`
      com idade 24h, chama `dirSize:temp_hygiene.go:180-194` compartilhado);
      `run.go:662-708` roda os dois em sequência. Deletar o step 4 do
      `CleanupOpenCodeUseCase`; `TempHygieneUC` vira dono único de `/temp`.
      Também: `tempOwner:233-252` espelha `classifyTempEntry:50-122` —
      derivar um do outro ou comentar o acoplamento.
- [ ] **T4 — wrappers perf que não agregam.** `performance_options.go:103-108`
      (`assessJournald`, delegate de 1 linha p/
      `performance.AssessJournaldPolicy`) → inline nos callers, deletar;
      `:123-227` (`assessSysctlIntent`, ~100 linhas puras) sobrepõe
      `infra/performance/sysctl_resolution.go` (146) + `sysctl_manager.go`
      (256) — audit e apply precisam compartilhar 1 função (o arquivo já
      tenta via `performanceSysctlIntent:274`; verificar que não há 2ª
      implementação da mesma regra); `doctor_linux_performance.go:12-186`
      (8× `if empty → Info else Info`) → table-driven
      `{target, emptyDetail, formatSnapshot}` (~−140 linhas).
- [ ] **T5 — profiles CLI 5/7 iguais → tabela.**
      `run.go:299-417` (`runWindowsProfile` 7 steps, `runVPSProfile` 7,
      `runCachyOSProfile` 8 — providers/packages/shell/skills/LSP
      idênticos). Extrair `profileSteps []step{name,fn}` + deltas por OS
      (windows: tweaks+debloat; vps: bootstrap+perf-required; cachyos:
      bootstrap+gaming+perf-optional). Idem
      `runPackagesProvisioning:470`/`runGamingProvisioning:490`/`runExtrasProvisioning:535`
      (spinner+callback+count idênticos exceto `Execute` vs `ExecuteGaming`
      vs `ExecuteExtras`) → `runPackageList(label,execFn)` (copiar o padrão
      de `runTweakStack:728-748`). ~−80 linhas, help byte-idêntico (contrato
      da 2b).
- [ ] **T6 — tiny files → fold (a Fase 3 esqueceu estes).**
      `file_checks.go:23` (`isExecutableFile`, 1 caller
      `provision_providers.go:124`) → `executil`; `sudo_exec.go:52`
      (`sudoAvailable`, `SudoPreflight`, `runPrivileged` — 1 caller
      `provision_gaming_tuning.go`) → `executil` (+ `run.go:276-288`
      `requireSudoNOPASSWD` duplica `sudoAvailable`+`sudo -n true` — reusar);
      `update_env.go:116` (wiring `realUpdateEnv` + 3 adapters de 1 linha)
      → fold em `update.go` (253) = 1 arquivo ~360 linhas;
      `doctor_linux_performance.go:207` (1 método + 2 helpers triviais) =
      primeiro move natural da 2c; `cleanup_commandcode.go:28-49`
      (`Execute` só deleta `settings.jsonc` se existir) → step 5 do
      `CleanupOpenCodeUseCase` (deleta 1 arquivo + 1 wire
      `root.go:55,141,691`).
- [ ] **T7 — over-abstraction leftovers (incluir na Fase 2).**
      `update.go:69-73` (`UpdateEnv` 3 métodos) + `update_env.go:16-25`
      (`realUpdateEnv` 6 func fields) + `:79-95` adapters — 116 linhas p/
      wrapar funcs (`installedVersion`, `npmLatest`, `runWithToolchain`);
      construir com 2 fields (`run`, `latestVersion`) ou chamar direto;
      `performance_options.go:18-25` (`rebootPendingProbe`,
      `func(path)bool` em volta de `os.Stat`) → inline;
      `root.go:154-162` (`packageInstalledProbe` closure) duplica o lookup
      `managers[pkg.Type].IsInstalled` de `provision_packages.go:147` —
      passar o map, não closure; `provision_tweaks.go` vs
      `provisionListMode` — mesmo pipeline load→filter→check→install→diag;
      a longo prazo 1 `provisionList` genérico parametrizado.
- [ ] **T8 — micro-dead code.** `sourceLabel`
      (`provision_providers.go:797-804`, verificar callers — se só diag,
      inline); `categoryAllowed` (`provision_shell.go:58-68`, 10 linhas de
      `slices.Contains`) → `slices.Contains` direto; `packageOwnershipProbe`
      (`provision_packages.go:82-87`, 1 special-case `opencode`
      `CheckCommand=""`) → empurrar o override p/ o manifest ou p/ o
      manager pacman, deletar o hook. (`statfs_other.go:12` vs
      `statfs_linux.go:36` **ficam** — split por build-tag é idiomático;
      listado só p/ ninguém "fundir".)
- [ ] **T9 — test/mock sprawl.**
      `doctor_audit_test.go:19-132` hand-rolla 4 mocks p/ a superfície de 17
      métodos (+ `mockGamingPackageManager:1115`,
      `extras_provision_test.go:47-58` 5º mock) — após a Fase 2 a maioria
      morre; antes disso, compartilhar 1 `usecase/fakes_test.go`.
      `verify_script_test.go:936` (maior teste do usecase, mas testa o bash
      `configs/bin/envctl-verify` de 709 linhas, não Go) → mover p/ BATS em
      `configs/bin/` ou encolher p/ smoke-table (hoje cada push paga ~900
      linhas de scaffolding de repo throwaway).
      `provision_bootstrap_test.go:308` re-executa `openCodePathInstaller`
      /`goPathInstaller` via `bash` real — colapsa com a Fase 2e.

**Riscos:** fundir paths que discordam em edge-case não coberto por teste
(mitigação: rodar os testes existentes antes e depois de cada T; são o
contrato).
**Breaking changes:** nenhuma.

## 5. Fase 3 — Código: consolidar os 29 arquivos de usecase (~15)

**Objetivo:** menos arquivos pra atravessar por sessão de agente (menos tokens,
menos lugar errado). Regra: só junta arquivos do **mesmo domínio** que se
referenciam entre si; nada de arquivo god.

**Tarefas:**

- [ ] **T1 — `gaming.go`:** fundir `gaming_tuning.go` +
      `provision_gaming_tuning.go` + `hardware_capability.go` (+ testes
      correspondentes → `gaming_test.go`); `sudo_exec.go` **não entra aqui**
      (vai p/ `executil` na Fase 2h-T6).
      Zero diff semântico: `git diff` pós-move mostra só renames/hunks
      package-internal. Verificação: `go test ./internal/usecase/ -run
      'TestGaming|TestHardware' -v` + `go test ./internal/infra/executil/ -v`
      (sudo já mora lá desde a 2h-T6) + gate cheio.
      Rollback: revert do PR.
      Resultado: 3 arquivos de código → 1.
- [ ] **T2 — `skill_contract.go`:** `skill_frontmatter.go` +
      `skill_catalog_budget.go` (+ testes). Verificação:
      `go test ./internal/usecase/ -run TestSkill -v` + gate cheio.
      Rollback/resultado: idem.
- [ ] **T3 — `cleanup_agents.go`:** `cleanup_opencode.go` +
      `cleanup_commandcode.go` (+ testes); o `Execute` do commandcode
      (`:28-49`, só deleta `settings.jsonc`) vira step 5 do opencode
      (detalhe na Fase 2h-T6). Verificação:
      `go test ./internal/usecase/ -run TestCleanup -v` + gate cheio.
      Rollback/resultado: idem.
- [ ] **T4 — varredura de citações:** `grep -rn` de cada nome de arquivo
      extinto em `docs/`, `spec-agent/`, `configs/` e comentários Go
      (a matriz cita `provision_providers.go` por nome — conferir cada
      ocorrência). Verificação: grep vazio. Rollback: commit de doc separado,
      revertível isolado.
      Resultado: nenhuma referência a arquivo extinto.

**Riscos:** citações de doc quebradas (T4 é a mitigação, antes do merge);
conflito com sessão paralela (regra do AGENTS global: `git status` + timestamps
antes de julgar `undefined` como bug próprio).
**Rollback:** revert por PR.

## 5b. Fase 2i — configs duplicados (mesmo molde da 2d)

**Objetivo:** eliminar 4 duplicações de config com o mesmo padrão base+overlay
ou fonte-única+2 deploys da Fase 2d. Ordem: LSP/mcp primeiro (drift funcional),
depois SKILL-INDEX/ssh (só linhas).

**Tarefas:**

- [x] **T1 — LSP fonte única (DRIFT FUNCIONAL, prioridade máxima).** (feito 2026-10-06: bloco `category: lsp` removido do `packages.yaml` + `TestLSPSingleSource`; U4 verificado: `run lsp` cobre os 6, profiles rodam lsp, `update.collect` já dedupava pelos dois; gate verde). Detalhe original:
      `manifests/packages.yaml:285-319` (6 LSPs `type:volta`) duplicam
      `manifests/lsp.yaml:8-24,35-42,72-87,53-60,80-87,89-105` (mesmo
      `install_target`/`check_binary`, schema diferente
      `install_type/install_target/check_binary` vs `type/check_command`).
      Hoje adicionar LSP num e esquecer o outro = doctor verde + runtime
      quebrado. Fix: `lsp.yaml` fonte única; `packages.yaml` deriva ou
      remove o bloco LSP (`run lsp` já instala via `lsp.yaml`).
      Trava: teste que falha se um `install_target` existir num e não no
      outro (ou pós-remoção: grep vazio do bloco em `packages.yaml`).
      Verificação: `go test ./internal/infra/embedded/ -run TestManifest`
      + gate cheio. Rollback: revert. Resultado: 1 fonte p/ LSP.
- [x] **T2 — mcp paridade.** (feito 2026-10-06: `TestMCPServerParityAcrossAgents` trava os 5 IDs nos dois schemas; controle negativo provado — remover `ssh-manager` quebra o teste; gate verde). Detalhe original:
      context7/brave/exa/ssh-manager/chrome-devtools, schema OpenCode) vs
      `configs/commandcode/mcp.json:1-32` (mesmos 5, schema CommandCode) —
      drift real já ocorrido 1× (matriz: `zscan` #12, `ssh-manager` #11
      faltante no linux). Fix: tabela de mapeamento ID↔ID + teste de
      paridade (lista de IDs) travando os 5 (ou gerar um do outro).
      Verificação: teste de paridade verde + gate cheio. Rollback: revert.
- [x] **T3 — SKILL-INDEX fonte única.** (feito: `configs/commandcode/SKILL-INDEX.md` deletado, ambos os deploys usam `configs/SKILL-INDEX.md` — diff dos deployados vazio)
      `configs/SKILL-INDEX.md:1` vs `configs/commandcode/SKILL-INDEX.md:1`
      (só o título difere; linhas 3-31 idênticas, 12 skills). Deploy duplo
      em `shell.yaml:70-76` + `:143-148`. Fix: fonte única
      `configs/SKILL-INDEX.md`, duas entradas de deploy (mesmo padrão da
      2d, sem overlay). −31 linhas. Verificação: diff dos deployados vazio
      (a menos do título) + gate cheio. Rollback: revert.
- [ ] **T4 — ssh-config base + overlay.** (investigação 2026-10-06: delta real = 3 linhas `Control*` (linux) + 2 `IdentityFile` (windows); mesmo veredito da 2d — `merge:ssh_hosts` não combina dois templates; implementação pós-Fase-2)
      `configs/ssh-config:1-30` vs
      `ssh-config.linux:1-33`: delta = linux adiciona `ControlMaster/
      ControlPath/ControlPersist` (`:11-13`), windows adiciona
      `IdentityFile ~/Documents/SSH-keys/*` (`:22-23`). Deploy em
      `shell.yaml:293-309` (ambos `merge:ssh_hosts`). Fix: base única +
      overlay de 5 linhas (mesma decisão A-vs-B da 2d-T1).
      Verificação: deployado por OS idêntico ao atual + gate cheio.
- [x] **T5 — agentes: prompt canônico por papel (só trava, não unifica).** (feito: pareamento documentado em `docs/os-and-agent-matrix.md` §2 + 3 testes-trava nomeados)
      `opencode.json:519-523,617-621,690-694`
      (verifier/docs-writer/memory-keeper) espelham
      `commandcode/agents/*.md` (+ `code-reviewer.md` ↔ `review/reviewer`
      `:177,435`; `REFERENCE.md:41` já admite). Formatos são incompatíveis
      (JSON `system/permissions[]` vs md frontmatter) — **não** gerar um do
      outro agora; só documentar o pareamento + os testes existentes
      (`TestShippedOpenCodeTemplatesHaveNativeShape`,
      `TestShippedCommandCodeAgentTemplatesMatchSchema`) como travas.
      Registrado p/ ninguém "unificar" sem adaptador de formato.
- [x] **T6 — CommandCode AGENTS: −1 linha/arquivo.** (feito: regra `Docs:` fundida numa linha nos 3 arquivos, mantida a variante com `(ler docs-sync.md)`)
      `configs/commandcode/AGENTS.md:19≈:25` (+ `.linux.md`, `.arch.md`):
      regra "Docs:" duplicada (a 2ª só adiciona `(ler docs-sync.md)`).
      Fundir numa linha. Verificação: teste de schema verde.
      **NÃO** unificar os 3 arquivos (ver 2d-T2 ressalva: ~85% boilerplate,
      mas o gerenciado muda por OS).
- [x] **T7 — `.commandcode/settings.json` reset.** (feito: `diff` vazio contra o canônico; nota: `.commandcode/*` já é gitignored — allows de sessão nunca foram versionados, só o arquivo local)
      O projetual (`:6-51`) tem ~20 `Shell(...)` one-off de sessão
      versionados por acidente vs canônico
      `configs/commandcode/settings.json:15-28` (12 allows genéricos).
      Resetar p/ o canônico; regra: nunca commitar allows de sessão.
      Verificação: diff vazio contra o canônico (+ allows locais ficam no
      settings do usuário, fora do repo).
- [x] **T8 — catálogo: `skills.yaml` fonte (trava, não gera ainda).** (feito 2026-10-06: `TestSkillCatalogParityAcrossSources` trava nomes `skills.yaml`≡`SKILL-INDEX.md`≡`docs/skills.md`; controle negativo provado; gate verde)
      `manifests/skills.yaml:9-67` vs `configs/SKILL-INDEX.md:12-23` vs
      `docs/skills.md:18-31` (mesmos 12 nomes, 3 redações). Direção:
      gerar índice+doc do `skills.yaml` + frontmatter (o teste já conta);
      nesta fase, só travar com teste de contagem/paridade de nomes.
      Geração total fica p/ depois da Fase 6.

**Riscos:** remover bloco LSP de `packages.yaml` quebrar `run packages` que
alguém usa p/ instalar LSP — mitigação: verificar que `run lsp` cobre os 6
antes de remover; manter shim de aviso por 1 release se preciso.
**Breaking changes:** nenhuma (deployados idênticos).

## 5c. Fase 2j — manifests: pares OS + nomes + hooks

**Objetivo:** generalizar a 2d p/ o `shell.yaml` inteiro (pares/trios que
diferem só em `os/source/destination/value`) + simetria de nomes + 1 shim de
hook.

**Tarefas:**

- [ ] **T1 — sintaxe `os_values` (ou overlay).**
      Alvos: `environment_variables:5-27` (NODE_PATH ×2, ENVCTL_TEMP ×2 —
      só `value/os`), `directories:465-472` (`C:/temp` vs `/temp`, mesma
      description), `config_files:46-68` (AGENTS ×3), `:116-141` (commandcode
      AGENTS ×3), `:199-212` (`pw.cmd` vs `pw`), `:235-275` (7 hooks git —
      só `id/source/destination`), `cleanup:518-550` (`stale_pylsp*`,
      `stale_pw_screenshot/eval*` parametrizáveis por `{name,os}`).
      Proposta: `os_values:{windows:…,linux:…}` ou template único + overlay
      por OS (mesma decisão A-vs-B da 2d-T1, reaproveitando `merge:` onde
      couber). ~−40 entradas quase-idênticas.
      **NÃO** unificar: `extras.yaml:18-106` vs `:109-179` e winget/apt/
      pacman em `packages.yaml` (matriz intencional, IDs diferentes por
      gerenciador); `performance_cachyos.yaml:1-18` (só `zram-generator`,
      `sysctls:[]`) → avaliar perfil `cachyos` dentro de 1
      `performance.yaml` único em vez de arquivo próprio.
      Verificação: `TestLoadManifestsFromDiskOrEmbed` + deployado por OS
      idêntico + gate cheio. Rollback: revert por PR.
- [ ] **T2 — rename `debloat.yaml` → `debloat_windows.yaml`.** (bloqueado 2026-10-06: `manifest_repo.go:206` é arquivo ativo da Fase 2-T2 do outro agente)
      740 linhas windows-implícitas vs `debloat_linux.yaml` (81) — simetria
      de nome. Verificação: grep de referências atualizado (inclui T4 da
      Fase 3). Rollback: revert.
- [ ] **T3 — hooks git ×6 → 1 shim.** (parcial 2026-10-06: pre-push reusa `_envctl-delegate`, fim do chain duplicado; falta consolidação das 8 entradas `shell.yaml` — exige campo lista no loader, Go pós-Fase-2)
      `configs/git/hooks/pre-commit:1-6` idênticos exceto nome +
      `_envctl-delegate:1-24` + `pre-push:1-36` que reinventa o delegate
      (`:11-16`). Fix: 1 shim parametrizado (ou symlinks gerados) + 1
      entrada `shell.yaml` com lista em vez de 7.
      Verificação: hooks instalados byte-idênticos + gate cheio.
- [ ] **T4 — embed: excluir o desnecessário + erro explícito.**
      `assets.go:9` (`//go:embed all:manifests all:configs`) inclui tudo
      (`bin/`, `emulators/`, `git/hooks/`, `.gitkeep`); modo de falha
      provado (binário velho/dir errado = "Already up to date" silencioso,
      lições `.opencode/memory/lessons.md:17-18`). Fix: (a) remover fallback
      disco-quando-CWD-errado (sempre embed, erro explícito); (b) `go:embed`
      só do necessário (excluir `.gitkeep`, documentar). Os testes
      `manifest_repo_test.go` + `emulator_configs_test.go` +
      `extras_manifest_test.go` pinam disco≡embed — mantê-los verdes.
      Verificação: `go test ./internal/infra/embedded/ -v` + build de dir
      errado falha explícito. Rollback: revert.

**Riscos:** sintaxe `os_values` exigir mudança no loader de manifest
(mitigação: se o loader ficar complexo demais, fallback p/ overlay por OS
igual à 2d — 2 arquivos em vez de sintaxe nova).
**Breaking changes:** nenhuma (deployados idênticos; rename interno com
grep de referências).

## 5d. Fase 6 — docs: matar schemas fantasmas + drift (só texto)

**Objetivo:** eliminar superfícies de drift de documentação. Só texto —
nenhum teste Go muda (mas os `*_test.go` de embedded precisam continuar
verdes: é por isso que cada T manda rodar).

**Tarefas:**

- [x] **T1 — `docs/manifests.md` sem schemas fantasmas.** (feito: 5 exemplos fantasmas removidos, tabela seção→arquivo→comando adicionada)
      `:27-59` (`test_binary,description`), `:128-155`
      (`env_vars,restricted_dirs`), `:163-180`
      (`git_configs[],description,pager`), `:188-206`
      (`name,package_type,languages[]`), `:214-236` (`name,key`) usam campos
      que o código nunca lê (reais: `packages.yaml:8-14`,
      `shell.yaml:4,414`, `git.yaml:1-18`, `lsp.yaml:8-16`,
      `windows.yaml:2-8`). Apagar exemplos, manter tabela
      seção→arquivo→comando + link. Elimina 6 superfícies de drift.
- [x] **T2 — priority drift (ATUAL, corrigir já).** (feito: guia referencia o manifesto, sem recopiar YAML)
      `ubuntu-server-baseline.md:52` cita `priority: -2`, manifesto
      `performance_ubuntu.yaml:62` declara `-1` (piso medido, lição
      2026-09-26). Guia passa a referenciar o manifesto, não recopiar YAML.
- [x] **T3 — guias linux×windows → 1 com tabs.** (feito: `docs/guides/provisioning.md` §§1–3 com blocos por OS; §§4+ ficam em `linux.md`/`windows.md`)
      `docs/guides/linux.md:19-79` vs `windows.md:9-76` (~60% template:
      1-liner/binário/fonte, só troca `bootstrap.sh+run vps` por
      `bootstrap.ps1+run windows`). Guia único com tabs OS; §§4+ específicos
      ficam.
- [x] **T4 — contagens: manifestos são fonte.** (feito: matrix/doctor/architecture/AGENTS/README apontam p/ manifestos; 12 skills mantido — estável e pinado pelos testes)
      `os-and-agent-matrix.md:25,30,33` ≈ `doctor-and-idempotency.md:25,39`
      ≈ `architecture.md:5,21,71` ≈ `AGENTS.md:5` ≈ `README.md:32-35`
      (55/63/70 pkgs, 27/25 configs, 14/13 LSPs). Remover números das docs
      ("ver manifesto") ou gerar. (`AGENTS.md:52-54` já diz que manifestos
      são fonte — fazer valer.)
- [x] **T5 — `architecture.md:94-107` → link.** (feito: aponta p/ `assets.go`)
      Recopia `assets.go:7-10` (`//go:embed`). Apontar p/ o arquivo.

**Prioridade interna:** T2 primeiro (drift ativo), depois T1 (6 schemas
fantasmas), resto em qualquer ordem. Verificação: `rg` dos campos fantasmas
vazio + `*_test.go` de embedded verdes.
**Breaking changes:** nenhuma (só texto).

## 6. Fase 4 — Volta → fnm + npm (quando sair)

**Decisão da sessão:** **fnm, não mise** — mise é framework poliglota (133MB,
`activate` ~4ms/prompt, backend `npm:` com `aube`/trust/lockfile) que
sobrepõe o que o envctl já cobre com pacman+uv+go. O volta tá EOL
(nov/2025, banner unmaintained, mantenedores recomendam mise) mas funciona —
sem urgência.

**Tarefas (spec dedicada quando disparar; esta só registra o rumo):**
- [ ] `manifests/packages.yaml`: `node@24.19.0` sai de `type: volta` →
      runtime via fnm (novo install_type ou bootstrap step); os 14 npm
      globals migram `type: volta` → `type: npm` (o `NpmManager` já existe,
      `toolchain_managers.go:104`, com `--prefix ~/.local`, sem sudo)
- [ ] fase 0 (`provision_providers.go`): `ensureVolta` morre;
      `ensureNodeRuntime` vira fnm; `command-code` vira npm global
      (`npm i -g command-code@latest` = canal oficial da doc do CommandCode)
- [ ] `update.go`: `GroupVolta` → grupo npm (`npm install -g pkg@latest`)
- [ ] `toolchain_managers.go:25`: PATH `~/.volta/bin` → fnm multishell
- [ ] matriz + docs §1/§5 + `Volta.Volta` winget + testes
      (`update_test.go`, `provision_providers_test.go`)

**Critério de disparo:** volta quebrar num update de OS/Node (o motivo EOL), ou
o `cmdc update` nativo voltar a ser exigido (hoje o conflito self-updater x
volta é contornado pelo `envctl update`).

## 7. Fase 5 — Rotina de teste multi-OS (gate de release)

**Ubuntu (real, `vps_oracle_2` — `100.92.37.112`, user `ubuntu`, key):**
por release (não a cada PR):

- [ ] **T1 — baseline read-only.** (bloqueado 2026-10-06: `go build` passa mas `go vet` ainda quebra em `*_test.go` da Fase 2 — sem binário atual p/ `scp`; binário da release testaria código velho, contra o propósito) Comando (da estação, via Tailscale):
      `scp ./envctl ubuntu@vps_oracle_2:/tmp/envctl && ssh ubuntu@vps_oracle_2
      /tmp/envctl doctor` — registra saída (contagem de checks, WARN/ERROR).
      Verificação: doctor completa sem erro de SSH/conexão.
      Rollback: n/a (read-only). Resultado: baseline arquivada.
- [ ] **T2 — convergência + idempotência.** `ssh ubuntu@vps_oracle_2
      '/tmp/envctl run vps && /tmp/envctl doctor'` → **0 WARN / 0 ERROR**;
      rodar `run vps` **segunda vez** → sem diff de estado (idempotente).
      Verificação: segunda run não muda nada; doctor 0/0.
      Rollback: `.bak.*` atômicos do provisioning (nunca `snapshot` — é sync
      reverso máquina→repo).
      Resultado: saídas registradas em `docs/verification.md` (ou seção que a
      matriz indicar).

**Windows (local, dockur/windows — KVM verificado nesta máquina):**

- [ ] **T3 — subir a VM.** `compose.yml` **fora do repo** (`/temp/dockur/`,
      nunca versionado — não é artefato do projeto):
      `VERSION: "11"`, `devices: [/dev/kvm, /dev/net/tun]`,
      `cap_add: NET_ADMIN`, ports `8006` (browser) + `3389` (RDP),
      volume `./windows:/storage`, `stop_grace_period: 2m`.
      Comando: `docker compose up -d` → browser em `:8006` até o desktop.
      Verificação: desktop visível; `Test-Connection` ok dentro da VM.
      Rollback: `docker compose down -v` (VM descartável entre ciclos).
      Resultado: VM pronta.
- [ ] **T4 — ciclo Windows.** Dentro da VM: `bootstrap.ps1` →
      `envctl run windows` → `envctl doctor` → 0 WARN/0 ERROR.
      Verificação: doctor 0/0. Rollback: `down -v` + recomeçar do zero.
      Resultado: saída registrada; fechar o ciclo apagando a VM.
- [ ] **Restrição de recursos:** 4GB+ RAM pro container — fechar sessões
      pesadas (emuladores/jogos) antes de subir.

**Riscos:** oracle-vps-2 é máquina real de homelab (Tailscale) — `run vps`
muta estado; mitigação: snapshot `envctl snapshot`? **Não** — snapshot é sync
REVERSO (máquina→repo); o rollback real é o `--allow-debloat`/debloat não
aplicar sem flag + os `.bak.*` atômicos do provisioning.
**Unknown U2:** dockur/windows provisiona Windows 11 Pro evaluation — o
`run windows` completa (Cursor via winget, tweaks registry) numa edição
evaluation? Provalmente sim (winget/tweaks não dependem de licença), mas é
unknown a validar no primeiro ciclo.

## 8. Ordem de execução e dependências

1. Fase 1c (CI hygiene) — CI-only, sem risco de release; vai primeiro e não
   move a contagem de findings da 1b no meio do caminho
2. Fase 1 (goreleaser) + 1d-T1/T2 no mesmo trem — independente; janela:
   próxima release minor/patch; a 1d-T3 valida contra a release nova
3. Fase 1b (golangci total) — por último no bloco CI, p/ o pinning da 1c não
   mover a contagem mid-count; pode ir junto ou antes da 1 se preferir
4. Fase 5 (rotina de teste) — independente; valida as Fases 1/1b/1d de graça
5. Fase 2 (concretos) — 4 PRs pequenos, sequenciais (mesma área do código)
6. Fase 2g-H primeiro (ctors diag encolhem todos os diffs seguintes), depois
   2g-A–F (helpers que a 2h usa)
7. Fases 2b–2f — independentes entre si após a Fase 2 (2d toca `shell.yaml` +
   templates, então vai **depois** da 2 para não rebasar wire; 2c antes da
   Fase 3 — splits antes de merges)
8. Fase 2i-T1/T2 (LSP fonte única + mcp paridade) — drift funcional, fazer
   cedo junto com a 2d (mesmo molde); 2i-T3–T8 em qualquer ordem após a 2d
9. Fase 2j-T2 (rename debloat) + T3 (hooks) + T4 (embed) — independentes, após
   a 2d; 2j-T1 (`os_values`) **depois** da 2d (reusa a decisão A-vs-B)
10. Fase 2h — após 2g + 2c estáveis (dedupes usam helpers e moves do doctor);
    2h-T5 (profiles CLI) por último, com a superfície do usecase estável
11. Fase 3 (consolidação) — **só depois da Fase 2 + 2c + 2h estáveis** (renames
    depois de concretos, splits e dedupes evitam rebase triplo)
12. Fase 6 (docs só-texto) — a qualquer momento após a 2i/2j (T2 priority
    drift pode ir hoje, é 1 linha); gate = `*_test.go` de embedded verdes
13. Fase 4 (fnm) — quando o critério §6 disparar; spec própria

**Restrições que valem para todas as fases de código:**
- `internal/infra/embedded/*_test.go` (`docs_claims_test`,
  `emulator_configs_test`, `extras_manifest_test`) pinam docs/manifests —
  qualquer consolidação de doc/config tem que mantê-los verdes (é por isso
  que a Fase 2d-T2 ressalva os AGENTS e que a Fase 6 é só-texto com gate de
  teste).
- `configs/bin/envctl-verify` (709 linhas bash) **não** é reescrito em Go:
  ele é intencionalmente bash (roda sem toolchain na máquina-alvo e como
  Stop hook lendo JSON do stdin). A Fase 1c-T6 só adiciona paridade CI↔script
  (step ou comentário), não reescreve. Fora de escopo por decisão, não por
  falta de análise.
- Action pins tag-only + dependabot = stay-put (Fase 1c, nota sem tarefa):
  SHA-pinning dobra o churn p/ este tamanho de repo.

## 9. Unknowns

| id | pergunta | bloqueia | dono | próximo passo |
|---|---|---|---|---|
| U1 | goreleaser re-build de tag existente (`workflow_dispatch` p/ re-attach) precisa `--skip=validate`? (tag já existe → o check de "não vou re-taggear" pode reclamar) | só o modo manual de re-build, não o fluxo principal | quem implementar a Fase 1 | testar com `goreleaser release --snapshot` + tag local |
| U2 | `envctl run windows` completa em Windows 11 Pro evaluation (dockur)? | nada — é validação do ciclo Windows | Fase 5 primeiro ciclo | rodar e registrar |
| U3 | `os_values` (2j-T1) vs overlay por OS igual à 2d? | só o formato da T1, não o objetivo (~−40 entradas saem de um jeito ou de outro) | quem implementar a 2j | prototipar no loader; se complexo, fallback p/ overlay |
| U4 | Bloco LSP sai do `packages.yaml` direto ou vira shim de aviso por 1 release? | só a remoção, não a fonte única (trava por teste já vale) | quem implementar a 2i-T1 | verificar que `run lsp` cobre os 6 antes de remover |

## 10. Definition of Done

- [ ] Fase 1: release publicada com goreleaser, **11 assets com nomes idênticos**
      aos atuais, bootstraps instalando da release nova (prova: baixar
      `envctl-linux-amd64.tar.gz` da release e rodar `--version`)
- [ ] Fase 1b: `golangci-lint run ./...` 0 findings, `only-new-issues`
      removido, CI verde
- [ ] Fase 1c: `go-version-file: go.mod` nos 3 spots, `cache: true` fora,
      actionlint via action, `go mod download` + `Build Executable` fora,
      `paths-ignore` ativo, `make lint` == CI, `go run` sem artefato `./envctl`
- [ ] Fase 1d: `bootstrap.ps1` instala amd64 (regressão) + interpola arm64;
      blocos de contrato espelhados nos dois scripts
- [x] Fase 2: 17→2 interfaces no repository; build/vet/test/lint verdes em
      cada PR; zero mudança de comportamento (testes intactos)
- [ ] Fases 2b–2f: `run.go` ~748→~400 linhas com help byte-idêntico (teste de
      contrato verde); `Execute` do doctor só despacha, nenhum arquivo >400
      linhas no usecase; −813 linhas de config opencode; 1 helper de PATH
      com teste de idempotência; pacman+paru unificados com flags provadas
      pelos testes existentes
- [ ] Fase 2g: `version.go` único, `ToolchainEnv`/`ExecTool`/`LookPathIn`/
      `ProbeCheckCommand`/`BackupPathFor`/`PSQuote` compartilhados, ctors
      `diag.*` nos ~60 sites (texto idêntico)
- [ ] Fase 2h: `configStep` fundido, `ensureOpenCodeV2` único, temp com dono
      único, wrappers perf inline/table-driven, profiles CLI em tabela
      (help byte-idêntico), tiny files foldados, `verify_script_test`
      movido/encolhido
- [ ] Fase 2i: LSP fonte única (teste de paridade verde), mcp 5/5 travados,
      SKILL-INDEX fonte única, ssh base+overlay, settings projetual ==
      canônico
- [ ] Fase 2j: `os_values` (ou overlay) nas ~40 entradas, rename
      `debloat_windows.yaml`, 1 shim de hook, embed sem fallback silencioso
- [ ] Fase 3: 29→~15 arquivos de código no usecase (pós-folds da 2h);
      grep na doc sem referência a arquivo extinto
- [ ] Fase 6: campos fantasmas do `manifests.md` zerados (`rg` vazio),
      priority citado == manifesto, guias unificados, contagens removidas,
      `*_test.go` de embedded verdes
- [ ] Fase 5: `doctor` 0/0 + idempotência comprovada na `vps_oracle_2` e na VM
      dockur, saídas registradas
- [ ] cada fase: PR convencional separado, merge commit, bot no comando das
      versões (nada de tag manual)
- [ ] rollback provado ou limite irreversível documentado (todas as fases de
      código/CI têm revert por PR como rollback)
