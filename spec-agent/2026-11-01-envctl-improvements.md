# Spec — Melhorias de operação do envctl

**Data:** 2026-11-01
**Status:** aguardando aprovação (nenhum código de produção foi alterado)
**Origem:** sessão que corrigiu a detecção de PATH (v2.1.4) e removeu o quarteto X11
do stack Wayland (v2.1.5).

## 0. Problema e evidência

| # | Gap | Evidência desta sessão | Impacto |
|---|---|---|---|
| 1 | Não existe self-update | A máquina rodava `envctl dev` de 15/out enquanto o v2.1.3 já existia; atualizar exigiu `curl` manual + conferência de SHA256 + backup. `envctl update` só cobre toolchain (`internal/ui/cli/update.go`). | O binário instalado envelhece sem aviso; templates embutidos ficam stale. |
| 2 | Release sem guard de CHANGELOG | `.github/workflows/release.yml` valida formato e tag inexistente, mas não confere `## [X.Y.Z]`; o próprio CHANGELOG avisa que "o tag precisa conter a seção". | Tag/release sem a seção correspondente; correção manual depois. |
| 3 | CHANGELOG com 3 `[Unreleased]` | `grep -n "^## \[Unreleased\]"` → linhas 14, 359 e 522. Os blocos antigos nunca foram movidos: `git tag --contains 801aeda` → `v1.11.0`; `git tag --contains 753ae4f` → `v1.10.0`. | Histórico ambíguo; o próximo release pode mover a seção errada. |
| 4 | Docs com contagem hardcoded | A matriz dizia "44 pkgs" em 2 lugares; o guia repetia o quarteto X11 em 2 pontos. Atualizamos à mão para 40. | Drift garantido a cada mudança de manifesto. |
| 5 | Dois parsers de PATH + ambiente global | `pathListsDir` (`internal/usecase/doctor_system.go:48`) e `path_entry.go`; `fishAvailable()`/`resolveHome()` leem o host dentro do manager, o que tornou `TestPathStepPersistsEachDirOnce` dependente de fish instalado. | Divergência de semântica e testes não determinísticos. |
| 6 | Pacotes órfãos invisíveis | Após remover o quarteto, `xorg-server`, `xf86-input-libinput` e `xf86-video-amdgpu` continuam instalados e nada reporta. `ListInstalled` existe, mas usa `pacman -Q` (inclui dependências). | Máquina acumula pacotes que o manifesto não gerencia mais. |

## 1. Objetivo

Fechar os 6 gaps acima com mudanças pequenas, testáveis e reversíveis, mantendo:

- `envctl doctor` com **0 WARN/0 ERROR** por padrão na máquina gerenciada;
- rede **opt-in** (doctor offline por default);
- nenhuma remoção automática de pacote;
- releases continuam manuais (pipeline), sem tag criada à mão.

## 2. Decisões de desenho

| Ponto | Decisão | Motivo |
|---|---|---|
| Comando de atualização | Novo `envctl self-update` (`--check`, `--version`, `--dry-run`), não `update --self` | `update` já tem contrato de toolchain; misturar binário confunde o `--only` e o relatório. |
| Fonte da versão | GitHub API `releases/latest` (`tag_name`) + cache 24h em `os.UserCacheDir()/envctl/latest-release.json`; `--refresh` ignora o cache | Rate limit anônimo é 60/h; cache evita castigar a API em cada doctor. |
| Instalação | Download do asset loose (`envctl-<os>-<arch>[.exe]`), verificação contra `SHA256SUMS.txt`, backup `.bak.YYYYMMDD-HHMMSS`, rename atômico | Mesmo contrato de nomes do `.goreleaser.yml`/bootstrap; backup segue a convenção do repo. |
| Doctor e rede | `doctor --check-updates` é opt-in; sem a flag o doctor continua offline. `dev` build vira INFO offline | Preserva o doctor utilizável em VPS/CI sem rede. |
| Órfãos | `doctor --unmanaged` opt-in, Arch-only, sempre INFO, uma linha agregada | Sem a flag o output atual não muda; `INFO` nunca vira WARN/`--fix`. |
| CHANGELOG | Consolidar em uma `[Unreleased]`; bloco da linha 359 → `1.11.0`; bloco da linha 522 → `1.10.0` | Evidência: `git tag --contains` dos commits que introduziram cada bloco. |
| Docs | Remover contagens de pacote da prosa; o manifesto é a fonte | Regra já declarada no `AGENTS.md` ("manifest wins"). |
| PATH | Mover `pathListsDir` para `environment` e injetar home/fish via functional options | Um parser só; testes determinísticos sem `t.Setenv("PATH", ...)`. |

## 3. Impacto e contratos

**Superfície alterada:**

- `internal/ui/cli/` — novo comando `self-update`; flags `--check-updates` (doctor) e `--unmanaged` (doctor).
- `internal/usecase/` — `self_update.go` novo; `doctor_audit.go`/`doctor_system.go`/`doctor_packages.go`; `version.go` (comparação semver).
- `internal/infra/environment/` — `PathListsDir` exportado; `WindowsEnvManager` com options.
- `internal/infra/arch/` — `ListExplicit` (`pacman -Qeq`).
- `internal/domain/repository/interfaces.go` — interface opcional `ExplicitPackageLister`.
- `.github/workflows/release.yml`, `.github/workflows/docs.yml` novo, `scripts/cut-release.sh` novo.
- `CHANGELOG.md`, `docs/os-and-agent-matrix.md`, `docs/guides/provisioning.md`.

**Contratos que NÃO mudam:**

- `envctl run`, `doctor` sem flags, `update`, `commandcode`, `opencode`: mesmos flags e exit codes.
- Formato do relatório do doctor (contagens podem mudar apenas quando flags novas são usadas).
- `PackageManager` existente: a nova interface é opcional, não altera os 5 managers.
- Assets de release: nomes e `SHA256SUMS.txt` continuam os mesmos.

**Não-regressão:** `go build ./...`, `go vet ./...`, `go test -count=1 ./...`,
`golangci-lint run --new-from-rev=origin/main` verdes em cada PR; `doctor` ao vivo
na máquina gerenciada continua `0 warnings` sem flags.

## 4. Riscos, unknowns e breaking changes

**Riscos**

| Risco | Mitigação |
|---|---|
| Self-update trocar o binário errado / corromper | Checksum obrigatório; download em temp no mesmo dir; rename com rollback; `--dry-run`; backup preservado. |
| Windows não permitir sobrescrever o exe em execução | Renomear o exe atual para `.bak` antes de mover o novo; se o move falhar, restaurar o backup. Validar manualmente no Windows (CI não troca o binário em execução). |
| Rate limit do GitHub | Cache 24h + `--refresh`; erro de rede no doctor com flag vira INFO/silêncio, nunca WARN. |
| `--unmanaged` poluir a saída | Opt-in, uma linha agregada, INFO; sem remoção automática. |
| Mover bloco errado no CHANGELOG | Evidência por `git tag --contains`; revisão do diff antes do merge. |
| `pacman -Qeq` indisponível/versão diferente | Feature só em Arch com manager pacman disponível; erro vira skip silencioso. |

**Unknowns**

- Substituição atômica no Windows: confirmar em máquina real que renomear o exe em execução funciona (documentado pela Microsoft; validar no smoke test).
- Formato exato do JSON de `releases/latest`: fixar com `httptest` a partir de uma resposta real (`tag_name`, `assets[].name`).
- Se o repo um dia virar privado, a API anônima para de funcionar; o erro deve ser claro e o doctor silencioso.

**Breaking changes:** nenhuma. Comando e flags são aditivos; `NewWindowsEnvManager`
passa a aceitar options variádicas (callers existentes compilam sem mudança);
`pathListsDir` é interno ao pacote `usecase` e será substituído por equivalente
exportado em `environment`.

## 5. Tarefas

### T1 — `envctl self-update`

**Escopo:** comando que baixa, verifica e instala a última release (ou uma versão
explícita), com backup e rollback. **Não** instala dependências, não usa sudo,
não altera manifests.

**Arquivos**

- criar: `internal/usecase/self_update.go`, `internal/usecase/self_update_test.go`
- criar: `internal/ui/cli/self_update.go`
- modificar: `internal/ui/cli/root.go` (registrar o comando)
- modificar: `internal/usecase/version.go` + teste (comparação semver)
- modificar: `docs/guides/provisioning.md` (substituir o curl manual por `envctl self-update`)

**Interfaces**

```go
type SelfUpdateUseCase struct {
    client     *http.Client
    goos       string
    goarch     string
    executable func() (string, error)
    now        func() time.Time
}

func NewSelfUpdateUseCase() *SelfUpdateUseCase
func (uc *SelfUpdateUseCase) Latest(ctx context.Context, refresh bool) (string, error)
func (uc *SelfUpdateUseCase) Apply(ctx context.Context, target string) (Result, error)

type Result struct {
    From, To string
    Backup   string
    Asset    string
}
```

- `assetName(goos, goarch)` → `envctl-linux-amd64`, `envctl-windows-amd64.exe`, etc.
- `Latest` consulta `https://api.github.com/repos/eajdias/envctl/releases/latest`,
  valida `tag_name` (`vX.Y.Z`) e grava o cache.
- `Apply` baixa `https://github.com/eajdias/envctl/releases/download/<tag>/<asset>`
  e `SHA256SUMS.txt`, confere o hash, grava temp no diretório do executável,
  cria `.bak.YYYYMMDD-HHMMSS`, renomeia e devolve `Result`.
- CLI: `self-update` (aplica), `--check` (só compara), `--version vX.Y.Z`
  (target explícito), `--dry-run` (mostra URL/asset/backup, não escreve).

**Passos**

- [ ] Escrever `TestSelfUpdateRejectsChecksumMismatch` (servidor `httptest` serve binário e sums adulterado) e observar RED.
- [ ] Implementar download + verificação até GREEN.
- [ ] Escrever `TestSelfUpdateInstallsVerifiedBinaryAndKeepsBackup` (temp dir como exe, sums correto) e observar RED.
- [ ] Implementar backup + rename + restauração em erro até GREEN.
- [ ] Escrever `TestSelfUpdateDryRunWritesNothing` e `TestLatestUsesCache`; GREEN.
- [ ] Ligar o comando no `root.go`, atualizar `docs/guides/provisioning.md`.
- [ ] Rodar `go test ./internal/usecase/ -run TestSelfUpdate -v` e o gate.

**Verificação**

```bash
go test ./internal/usecase/ -run 'TestSelfUpdate|TestVersionCompare' -v
go build ./... && go vet ./...
golangci-lint run --new-from-rev=origin/main
/tmp/opencode/envctl-selfupdate self-update --check   # mostra current/latest sem escrever
/tmp/opencode/envctl-selfupdate self-update --dry-run  # mostra asset/URL/backup
```

**Rollback:** reverter o PR; em runtime, restaurar `envctl.bak.<timestamp>`.
**Resultado esperado:** `self-update --check` reporta versão atual e última;
`--dry-run` não altera o binário; teste de checksum adulterado falha antes de escrever.

**Commit sugerido:** `feat(self-update): install the latest verified release with backup`

---

### T2 — Doctor: frescor contra a última release

**Escopo:** detectar `dev` build offline e, com `--check-updates`, comparar com a
última release. Reutiliza `Latest`/cache de T1.

**Arquivos**

- modificar: `internal/usecase/doctor_audit.go` (campo `latestRelease func(ctx, bool) (string, error)` + setter)
- modificar: `internal/usecase/doctor_system.go` (`auditEnvctlFreshness`)
- modificar: `internal/ui/cli/doctor.go` (flag `--check-updates`)
- modificar: `internal/ui/cli/root.go` (wire do setter)
- testar: `internal/usecase/doctor_audit_test.go`

**Comportamento**

- Sem flag: mantém o check atual (repo checkout) e adiciona **INFO** quando
  `envctlVersion == "dev"` ("unversioned build; release provenance unknown").
- Com `--check-updates`: se `latest > current`, **WARN** com
  `run 'envctl self-update'`; se igual, **OK**; erro de rede → nenhum diagnóstico
  (apenas log debug).
- Comparação semver em `version.go` (`versionLess(a, b)`), com teste de
  `dev`, `v2.1.5`, `2.1.5`, `2.2.0`, `10.0.0`.

**Passos**

- [ ] `TestAuditEnvctlFreshnessFlagsDevBuild` (sem rede) — RED.
- [ ] Implementar o INFO de `dev` — GREEN.
- [ ] `TestAuditEnvctlFreshnessWarnsBehindLatest` com `latestRelease` fake — RED/GREEN.
- [ ] `TestAuditEnvctlFreshnessSilentOnNetworkError` — GREEN.
- [ ] Adicionar a flag e o wire; rodar o gate.

**Verificação**

```bash
go test ./internal/usecase/ -run TestAuditEnvctlFreshness -v
envctl doctor --check-updates   # numa release antiga deve mostrar 1 WARN com fix hint
envctl doctor                   # sem flag: output atual, 0 warnings na máquina gerenciada
```

**Rollback:** flag default off; reverter o PR remove o check.
**Resultado esperado:** `doctor --check-updates` numa release antiga aponta a
diferença; sem a flag nada muda.

**Commit sugerido:** `feat(doctor): flag dev builds and stale releases`

---

### T3 — Guard no Release Pipeline + `cut-release.sh`

**Escopo:** impedir tag sem seção no CHANGELOG e automatizar o ritual de release
(branch docs → PR → merge → dispatch → verificação).

**Arquivos**

- modificar: `.github/workflows/release.yml`
- criar: `scripts/cut-release.sh`
- modificar: `Makefile` (`release:` chamando o script)
- modificar: `docs/guides/provisioning.md` (como cortar release)

**Workflow (antes do step que cria a tag):**

```yaml
- name: Validate CHANGELOG section
  env:
    VERSION: ${{ inputs.version }}
  run: |
    section="${VERSION#v}"
    grep -qE "^## \[${section}\]" CHANGELOG.md || {
      echo "::error::CHANGELOG.md is missing the ## [${section}] section"; exit 1;
    }
```

**Script:** `scripts/cut-release.sh <vX.Y.Z> [--dry-run] [--no-merge]`

1. valida árvore limpa, branch `main`, `git fetch`, `gh auth status`;
2. calcula a versão anterior com `git describe --tags --abbrev=0`;
3. cria `docs/release-<versão>` e insere a seção
   `## [<versão>](...compare/v<prev>...v<versão>) (<data>)` logo após
   `## [Unreleased]` (o conteúdo existente passa a pertencer à nova seção);
4. commit `docs: move Unreleased entries to <versão> for release`, push, PR;
5. `gh pr checks --watch` (se não houver checks, segue — PR docs-only é skip por design);
6. `gh pr merge --merge`; `gh workflow run release.yml -f version=<versão>`; `gh run watch`;
7. confere `gh release view <versão> --json tagName,assets` (11 assets).

**Passos**

- [ ] Adicionar o step de validação no workflow.
- [ ] Escrever o script com `--dry-run` (imprime cada comando, não altera nada).
- [ ] Testar localmente: `bash -n scripts/cut-release.sh`, `shellcheck scripts/cut-release.sh`, `scripts/cut-release.sh v9.9.9 --dry-run`.
- [ ] Simular a falha do guard: `grep -qE '^## \[9\.9\.9\]' CHANGELOG.md; echo $?` → 1.
- [ ] Rodar `actionlint`.

**Verificação**

```bash
bash -n scripts/cut-release.sh
shellcheck scripts/cut-release.sh
actionlint
scripts/cut-release.sh v9.9.9 --dry-run   # nenhum efeito colateral
```

**Rollback:** reverter o PR; o workflow volta ao comportamento anterior.
**Resultado esperado:** dispatch com versão sem seção falha antes de criar a tag;
`--dry-run` mostra o plano completo sem tocar no repo.

**Commit sugerido:** `ci(release): require the CHANGELOG section and script the cut`

---

### T4 — CHANGELOG: uma `[Unreleased]` + teste + CI de docs

**Escopo:** consolidar as 3 seções `[Unreleased]` e impedir regressão estrutural.

**Arquivos**

- modificar: `CHANGELOG.md`
- criar: `internal/infra/embedded/changelog_test.go`
- criar: `.github/workflows/docs.yml`

**Movimentos (com evidência já coletada)**

- Bloco da linha 359 (CachyOS gaming full restore) → `## [1.11.0]`
  (`git tag --contains 801aeda` → `v1.11.0`). Deduplicar contra os bullets já
  existentes em 1.11.0.
- Bloco da linha 522 (ubuntu-server performance baseline) → `## [1.10.0]`
  (`git tag --contains 753ae4f` → `v1.10.0`). Deduplicar.
- `## [Unreleased]` fica só na linha 14, vazia até o próximo release.

**Teste (`changelog_test.go`, usa `repoRootForDocs`):**

- exatamente uma `## [Unreleased]`;
- headings `## [x.y.z]` em ordem estritamente decrescente;
- `[Unreleased]` seguida imediatamente de um heading de versão (sem conteúdo);
- links de versão no formato `compare/v<prev>...v<x.y.z>`.

**Workflow `docs.yml`:** dispara em `pull_request` com
`paths: ['**.md', 'docs/**', 'spec-agent/**', '.github/workflows/docs.yml']` e roda
`go test ./internal/infra/embedded/ -run 'TestChangelog|TestDocumentation'`.

**Passos**

- [ ] Mover/deduplicar os dois blocos.
- [ ] Escrever o teste e observar RED com o estado atual (3 Unreleased).
- [ ] Consolidar até GREEN.
- [ ] Adicionar o workflow e validar com `actionlint`.

**Verificação**

```bash
grep -c "^## \[Unreleased\]" CHANGELOG.md          # 1
go test ./internal/infra/embedded/ -run TestChangelog -v
actionlint
```

**Rollback:** reverter o PR; o CHANGELOG é histórico, sem impacto em runtime.
**Resultado esperado:** uma única `[Unreleased]`; PR docs-only passa a ter check.

**Commit sugerido:** `docs(changelog): consolidate Unreleased and guard its shape`

---

### T5 — PATH: um parser e ambiente injetado

**Escopo:** eliminar a duplicação `pathListsDir`/`path_entry.go` e remover a
dependência de estado global nos testes do `WindowsEnvManager`.

**Arquivos**

- modificar: `internal/infra/environment/path_entry.go` (exportar `PathListsDir`)
- modificar: `internal/infra/environment/env_manager.go` (options + métodos)
- modificar: `internal/infra/environment/env_manager_test.go`
- modificar: `internal/usecase/doctor_system.go` (usar `environment.PathListsDir`)
- modificar: `internal/usecase/doctor_system_test.go` (mover o teste de tabela)
- modificar: `internal/usecase/provision_bootstrap_test.go` (remover o `t.Setenv("PATH", ...)`)

**Interfaces**

```go
type EnvManagerOption func(*WindowsEnvManager)
func WithHome(home string) EnvManagerOption
func WithFishInstalled(installed bool) EnvManagerOption
func NewWindowsEnvManager(opts ...EnvManagerOption) *WindowsEnvManager

func PathListsDir(pathVal string, sep byte, dir string, foldCase bool) bool
```

- `rcFiles`, `withinHome` e `fishAvailable` passam a usar os campos do manager
  (`home`, `fishInstalled`, `fishSet`); sem option, comportamento atual.
- `doctor_system.go` troca as 3 chamadas por `environment.PathListsDir`.
- Testes do environment constroem `NewWindowsEnvManager(WithHome(home), WithFishInstalled(false))`
  e não dependem mais de `t.Setenv("HOME"/"PATH")`.

**Passos**

- [ ] Mover `pathListsDir` + teste de tabela para `environment`; observar RED
  (símbolo ainda não existe) e GREEN.
- [ ] Adicionar as options e migrar `rcFiles`/`withinHome`/`fishAvailable`.
- [ ] Ajustar os testes para usar as options; remover os `t.Setenv` desnecessários.
- [ ] Rodar o gate completo.

**Verificação**

```bash
go test ./internal/infra/environment/ -run 'Path' -v
go test ./internal/usecase/ -run 'Path' -v
go build ./... && go vet ./... && golangci-lint run --new-from-rev=origin/main
```

**Rollback:** reverter o PR; não há migração de dados.
**Resultado esperado:** um único parser; testes de PATH passam sem depender de
fish/HOME do host.

**Commit sugerido:** `refactor(path): share one PATH parser and inject the shell environment`

---

### T6 — Pacotes não gerenciados (Arch, INFO, opt-in)

**Escopo:** relatar pacotes explícitos do pacman que não estão em nenhum manifesto.
Nunca remove, nunca vira WARN, nunca roda sem `--unmanaged`.

**Arquivos**

- modificar: `internal/domain/repository/interfaces.go` (interface opcional)
- modificar: `internal/infra/arch/arch_manager.go` + teste
- modificar: `internal/usecase/doctor_packages.go` (novo `auditUnmanagedPackages`)
- modificar: `internal/usecase/doctor_audit.go` (campo `unmanaged bool` + setter)
- modificar: `internal/ui/cli/doctor.go` (flag `--unmanaged`)
- criar: `internal/usecase/doctor_packages_test.go`

**Interfaces**

```go
type ExplicitPackageLister interface {
    ListExplicit(ctx context.Context) ([]entity.Package, error)
}
```

- `archManager.ListExplicit` usa `pacman -Qeq` (explícitos, repo + AUR).
- `auditUnmanagedPackages`: só Linux + manager pacman disponível + flag ligada;
  une `LoadPackages`, `LoadGamingPackages`, `LoadExtrasPackages` (filtrando OS);
  compara IDs; emite **um** `entity.Info` com contagem e os primeiros nomes
  (ex.: `12 explicit packages not declared in any manifest: xorg-server, ...`).
- Fix hint: "review manually; envctl never uninstalls packages".

**Passos**

- [ ] `TestListExplicitParsesPacmanQeq` com fake de `exec` (ou comando `printf` em script temp) — RED/GREEN.
- [ ] `TestAuditUnmanagedReportsOnlyUndeclared` com `fstest.MapFS` (packages/gaming/extras) e lister fake — RED/GREEN.
- [ ] `TestAuditUnmanagedSilentWithoutFlag` — GREEN.
- [ ] Ligar a flag e rodar o gate.

**Verificação**

```bash
go test ./internal/infra/arch/ -run TestListExplicit -v
go test ./internal/usecase/ -run TestAuditUnmanaged -v
envctl doctor --unmanaged | grep -i "not declared"
envctl doctor   # sem flag: contagem/avisos inalterados
```

**Rollback:** flag default off; reverter o PR.
**Resultado esperado:** `--unmanaged` lista o quarteto X11 restante e outros
explícitos fora do manifesto, em INFO; sem a flag, nada muda.

**Commit sugerido:** `feat(doctor): report explicit packages outside the manifests`

---

### T7 — Tirar contagens hardcoded dos docs

**Escopo:** remover "40 pkgs" da prosa e travar a regressão com teste.

**Arquivos**

- modificar: `docs/os-and-agent-matrix.md` (2 ocorrências)
- modificar: `internal/infra/embedded/docs_claims_test.go`

**Passos**

- [ ] Trocar `40 pkgs` por "pacotes declarados em `gaming.yaml`" nas duas linhas.
- [ ] Adicionar `TestDocsNoHardcodedPackageCounts` que falha se a matriz casar
  `\b\d+ pkgs\b`.
- [ ] Rodar o teste e o `rg` de conferência.

**Verificação**

```bash
rg -n "\b[0-9]+ pkgs\b" docs/          # vazio
go test ./internal/infra/embedded/ -run TestDocsNoHardcodedPackageCounts -v
```

**Rollback:** reverter o PR.
**Resultado esperado:** docs sem contagem de pacotes; teste impede reintrodução.

**Commit sugerido:** `docs(matrix): stop hardcoding package counts`

---

## 6. Ordem e dependências

| Ordem | Task | Depende de | PR sugerido |
|---|---|---|---|
| 1 | T1 self-update | — | `feat/self-update` |
| 2 | T2 doctor frescor | T1 (`Latest`/cache) | junto com T1 ou `feat/doctor-release-freshness` |
| 3 | T3 release guard + script | — | `ci/release-guard` |
| 4 | T4 CHANGELOG + docs CI | — | `docs/changelog-hygiene` |
| 5 | T5 PATH unificado | — | `refactor/path-parser` |
| 6 | T6 órfãos | — | `feat/doctor-unmanaged` |
| 7 | T7 docs counts | — | `docs/no-hardcoded-counts` |

Cada PR é independente e mergeável sozinho; T1+T2 podem ir no mesmo PR porque
compartilham o cliente/cache. Releases: cada merge entra em `[Unreleased]`; o
corte continua manual (T3 automatiza o ritual).

## 7. Definition of Done

- [ ] As 7 tasks têm PR mergeado com evidência de gate no corpo.
- [ ] `go build ./...`, `go vet ./...`, `go test -count=1 ./...`,
  `golangci-lint run --new-from-rev=origin/main` verdes.
- [ ] `envctl doctor` sem flags na máquina gerenciada continua `0 warnings`.
- [ ] `envctl self-update --check` e `--dry-run` funcionam; checksum adulterado
  é rejeitado antes de escrever; backup é criado.
- [ ] `doctor --check-updates` avisa release antiga e fica silencioso em erro de rede.
- [ ] `doctor --unmanaged` reporta o quarteto restante em INFO; sem a flag nada muda.
- [ ] Release Pipeline falha com versão sem seção no CHANGELOG; `cut-release.sh --dry-run`
  não altera nada.
- [ ] `grep -c "^## \[Unreleased\]" CHANGELOG.md` → 1; teste de estrutura verde;
  PR docs-only roda o check de docs.
- [ ] `rg "\b[0-9]+ pkgs\b" docs/` → vazio.
- [ ] Um parser de PATH só; testes de PATH não dependem de fish/HOME do host.
- [ ] Nenhum segredo no diff; nenhum `git add -A`; worktrees removidos sem `--force`.

## 8. Rollback

- Cada task é um PR independente: `git revert <merge>` restaura o comportamento.
- Self-update: o binário anterior fica em `envctl.bak.YYYYMMDD-HHMMSS`; restaurar
  é copiar de volta. Nenhuma migração de dados.
- Flags novas são default off; remover o PR não deixa estado persistido (exceto
  o cache `~/.cache/envctl/latest-release.json`, que pode ser apagado).
- CHANGELOG/docs: reversão textual, sem impacto em runtime.
- Nenhuma task desinstala pacote ou altera manifesto existente.
