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
| 7 | Código morto (150 linhas) | `deadcode ./...` → 11 funções inalcançáveis; 4 delas só existem para o próprio teste (`AssessRemovalState`, `AssessJournaldPolicy`, `AdoptedDiskSwap`, `skillDescription`). | Manutenção paga por código que o produto não usa. |
| 8 | Cache do gate sem gatilho | Nasceu em `afa5dd1` ("chain every repository hook"); o v2 não instala hook nenhum (`manifests/shell.yaml`). | ~70 linhas de bash + ~98 de teste para economizar ~2s numa execução explícita. |

Itens 1–6 são gaps de produto; 7–8 são dívida de complexidade encontrada na auditoria de 2026-11-01.

## 1. Objetivo

Fechar os 6 gaps e as 2 dívidas acima com mudanças pequenas, testáveis e reversíveis, mantendo:

- `envctl doctor` com **0 WARN/0 ERROR** por padrão na máquina gerenciada;
- rede **opt-in** (doctor offline por default);
- nenhuma remoção automática de pacote;
- releases continuam manuais (pipeline), sem tag criada à mão.

## 2. Decisões de desenho

| Ponto | Decisão | Motivo |
|---|---|---|
| Comando de atualização | Novo `envctl self-update` com uma única flag read-only `--check`; sem `update --self` | `update` já tem contrato de toolchain; misturar binário confunde o `--only` e o relatório. |
| Fonte da versão | GitHub API `releases/latest` (`tag_name`), **sem cache** | Toda chamada de rede já está atrás de uma invocação explícita (`self-update`, `self-update --check`, `doctor --check-updates`); cache não protege caminho nenhum. |
| Instalação | Asset loose (`envctl-<os>-<arch>[.exe]`), checksum contra `SHA256SUMS.txt`, backup `.bak.YYYYMMDD-HHMMSS`, rename atômico | Mesmo contrato de nomes do `.goreleaser.yml`/bootstrap; backup segue a convenção do repo. |
| Doctor e rede | `doctor --check-updates` é opt-in; sem a flag o doctor continua offline. `dev` build fora de checkout vira INFO offline | Preserva o doctor utilizável em VPS/CI sem rede; o WARN de dev-em-checkout existe hoje e fica. |
| Órfãos | `doctor --unmanaged` opt-in, Arch-only, sempre INFO, uma linha agregada | Sem a flag o output atual não muda; `INFO` nunca vira WARN/`--fix`. |
| CHANGELOG | Consolidar em uma `[Unreleased]`; bloco da linha 359 → `1.11.0`; bloco da linha 522 → `1.10.0` | Evidência: `git tag --contains` dos commits que introduziram cada bloco. |
| Docs | Remover contagens de pacote da prosa; o manifesto é a fonte | Regra já declarada no `AGENTS.md` ("manifest wins"). |
| PATH | Um parser só em `environment`; `rcFiles`/`withinHome` viram **funções puras** que recebem `home`/`fish` | `WindowsEnvManager` é `struct{}`: injetar estado nele (options + tri-state `fishSet`) seria complexidade sem ganho. |

### 2.1 Orçamento de complexidade (o que ficou de fora)

Cortado da primeira versão deste plano, por não fechar gap nenhum:

| Cortado | Onde estava | Por que não entra |
|---|---|---|
| Cache 24h + `--refresh` | T1/T2 | Nenhuma chamada de rede é automática; o cache só serviria para quem roda `doctor --check-updates` 60×/h. Custo: arquivo, TTL, invalidação, testes. |
| `--version vX.Y.Z` | T1 | `bootstrap.sh` já fixa versão (`VERSION=`); rollback imediato é o `.bak`. Um caminho de download a menos. |
| `--dry-run` separado de `--check` | T1 | Os dois são read-only. `--check` passa a imprimir current → latest, asset e destino do backup. Duas flags viram uma. |
| `scripts/cut-release.sh` + target no Makefile | T3 | ~7 passos de coreografia `gh` para uma ação mensal; o gap real (tag sem seção) fecha com o guard de 6 linhas no workflow. |
| `.github/workflows/docs.yml` novo | T4 | Workflow novo = superfície nova. O `paths-ignore` do `ci.yml` é justamente o que torna os testes de doc inúteis em PR docs-only: apagar o filtro remove um caso especial em vez de criar outro. |
| Functional options no `WindowsEnvManager` | T5 | O manager é `struct{}`. `WithHome`/`WithFishInstalled` + tri-state `fishSet` só existiam para testabilidade; funções puras resolvem com zero estado. |
| 2 das 4 asserções do teste de CHANGELOG | T4 | "Unreleased seguida de heading" e "formato do link compare" não são a regressão que aconteceu; a regressão real é a seção duplicada. |

Mantidos porque o custo é o mesmo da alternativa ou menor: checksum + backup +
rename (evita brickar o binário), interface opcional `ExplicitPackageLister`
(idiomática em Go, não toca os 5 managers), teste de contagem nos docs (~15 linhas).

## 3. Impacto e contratos

**Superfície alterada:**

- `internal/ui/cli/` — novo comando `self-update` (`--check`); flags `--check-updates` e `--unmanaged` no doctor.
- `internal/usecase/` — `self_update.go` novo; `doctor_audit.go`/`doctor_system.go`/`doctor_packages.go`; `version.go` (`versionLess`).
- `internal/infra/environment/` — `PathListsDir` exportado; `rcFiles`/`withinHome` puros.
- `internal/infra/arch/` — `ListExplicit` (`pacman -Qeq`).
- `internal/domain/repository/interfaces.go` — interface opcional `ExplicitPackageLister`.
- `internal/domain/entity/`, `internal/infra/windows/`, `internal/infra/performance/`, `internal/infra/embedded/` — remoção de 8 símbolos sem chamador (T8).
- `internal/usecase/provision_performance.go` — `PerformanceOptions` adiante em vez de 3 primitivos (T8).
- `configs/bin/envctl-verify` — remoção do cache de estado (T9); `docs/verification.md` acompanha.
- `.github/workflows/release.yml` (guard) e `ci.yml` (remove `paths-ignore`).
- `CHANGELOG.md`, `docs/os-and-agent-matrix.md`, `docs/guides/provisioning.md`.

**Contratos que NÃO mudam:**

- `envctl run`, `doctor` sem flags, `update`, `commandcode`, `opencode`: mesmos flags e exit codes.
- Formato do relatório do doctor (contagens podem mudar quando flags novas são usadas; a única exceção sem flag é o INFO de dev-fora-de-checkout).
- `PackageManager` existente: a nova interface é opcional, não altera os 5 managers.
- `NewWindowsEnvManager()` continua sem parâmetros; `WindowsEnvManager` continua `struct{}`.
- Assets de release: nomes e `SHA256SUMS.txt` continuam os mesmos.

**Não-regressão:** `go build ./...`, `go vet ./...`, `go test -count=1 ./...`,
`golangci-lint run --new-from-rev=origin/main` verdes em cada PR; `doctor` ao vivo
na máquina gerenciada continua `0 warnings` sem flags.

## 4. Riscos, unknowns e breaking changes

**Riscos**

| Risco | Mitigação |
|---|---|
| Self-update trocar o binário errado / corromper | Checksum obrigatório; download em temp no mesmo dir; rename com rollback; backup preservado. |
| Windows não permitir sobrescrever o exe em execução | Renomear o exe atual para `.bak` antes de mover o novo; se o move falhar, restaurar o backup. Validar manualmente no Windows (CI não troca o binário em execução). |
| Rate limit do GitHub | Uma requisição por invocação explícita; erro de rede vira mensagem clara no comando e silêncio no doctor, nunca WARN. |
| `--unmanaged` poluir a saída | Opt-in, uma linha agregada, INFO; sem remoção automática. |
| Mover bloco errado no CHANGELOG | Evidência por `git tag --contains`; revisão do diff antes do merge. |
| `pacman -Qeq` indisponível/versão diferente | Feature só em Arch com manager pacman disponível; erro vira skip silencioso. |
| CI passar a rodar em PR docs-only | A matriz completa de testes 2-OS roda no PR de doc (`ci.yml:73-101`) — minutos, não segundos; aceito porque é o que faz `docs_claims_test.go` rodar onde o drift nasce. |

**Unknowns**

- Substituição atômica no Windows: confirmar em máquina real que renomear o exe em execução funciona (documentado pela Microsoft; validar no smoke test).
- Formato exato do JSON de `releases/latest`: fixar com `httptest` a partir de uma resposta real (`tag_name`, `assets[].name`).
- Se o repo um dia virar privado, a API anônima para de funcionar; o erro deve ser claro e o doctor silencioso.

**Breaking changes:** nenhuma. Comando e flags são aditivos; `WindowsEnvManager`
mantém a assinatura do construtor; `pathListsDir` é interno ao pacote `usecase` e
será substituído por equivalente exportado em `environment`.

## 5. Tarefas

### T1 — `envctl self-update`

**Escopo:** comando que baixa, verifica e instala a última release, com backup e
rollback. **Não** instala dependências, não usa sudo, não altera manifests, não
faz downgrade (para isso existe o `.bak` e o `bootstrap.sh`).

**Arquivos**

- criar: `internal/usecase/self_update.go`, `internal/usecase/self_update_test.go`
- criar: `internal/ui/cli/self_update.go`
- modificar: `internal/ui/cli/root.go` (registrar o comando)
- modificar: `internal/usecase/version.go` + teste (`versionLess`)
- modificar: `docs/guides/provisioning.md` (substituir o curl manual da linha 54)

**Interfaces**

```go
type SelfUpdateUseCase struct {
    client *http.Client
    now    func() time.Time
}

func NewSelfUpdateUseCase() *SelfUpdateUseCase
func (uc *SelfUpdateUseCase) Latest(ctx context.Context) (string, error)
func (uc *SelfUpdateUseCase) Apply(ctx context.Context, dest string) (Result, error)

type Result struct {
    To     string
    Backup string
    Asset  string
}

func assetName(goos, goarch string) string
func versionLess(a, b string) bool // "dev" é sempre mais antigo
```

- `Latest` faz um GET em `https://api.github.com/repos/eajdias/envctl/releases/latest`,
  valida `tag_name` (`vX.Y.Z`) e devolve a tag.
- `Apply` resolve a release, baixa
  `https://github.com/eajdias/envctl/releases/download/<tag>/<asset>` +
  `SHA256SUMS.txt`, confere o hash, grava temp no **mesmo diretório** de `dest`,
  cria `<dest>.bak.YYYYMMDD-HHMMSS`, renomeia o novo por cima e devolve `Result`.
  Em qualquer falha depois do backup, restaura o original.
- CLI: `self-update` chama `Apply(ctx, os.Executable())`; `--check` imprime
  `current → latest`, o asset e o caminho do backup, sem escrever. Se
  `!versionLess(appVersion, latest)`, imprime "already at latest" e sai 0.
- `versionLess` se compõe de `parseSemver` (`version.go:150` — numérico, descarta
  prerelease/metadados) + o caso especial `dev`. Não reutilizar `versionsDiffer`
  (`version.go:171`) no check da CLI: é igualdade, não ordem — um binário local
  mais novo apareceria como "diferente" e forçaria um downgrade.

**Passos**

- [ ] `TestSelfUpdateRejectsChecksumMismatch` (servidor `httptest` serve binário e sums adulterado) — RED.
- [ ] Implementar download + verificação até GREEN.
- [ ] `TestSelfUpdateInstallsVerifiedBinaryAndKeepsBackup` (temp dir como `dest`, sums correto) — RED.
- [ ] Implementar backup + rename + restauração em erro até GREEN.
- [ ] `TestSelfUpdateCheckWritesNothing` e `TestAssetName` (linux/windows × amd64/arm64); GREEN.
- [ ] Ligar o comando no `root.go`, atualizar `docs/guides/provisioning.md`.
- [ ] Rodar `go test ./internal/usecase/ -run TestSelfUpdate -v` e o gate.

**Verificação**

```bash
go test ./internal/usecase/ -run 'TestSelfUpdate|TestAssetName|TestVersionLess' -v
go build ./... && go vet ./...
golangci-lint run --new-from-rev=origin/main
/tmp/opencode/envctl-selfupdate self-update --check   # current/latest/asset, sem escrever
```

**Rollback:** reverter o PR; em runtime, restaurar `envctl.bak.<timestamp>`.
**Resultado esperado:** `--check` reporta a diferença sem tocar no binário; checksum
adulterado falha antes de qualquer escrita; instalação válida deixa o backup.

**Commit sugerido:** `feat(self-update): install the latest verified release with backup`

---

### T2 — Doctor: frescor contra a última release

**Escopo:** detectar `dev` build offline e, com `--check-updates`, comparar com a
última release. Reutiliza `Latest` de T1.

**Arquivos**

- modificar: `internal/usecase/doctor_audit.go` (campo `latestRelease func(ctx) (string, error)` + setter)
- modificar: `internal/usecase/doctor_system.go` (`auditEnvctlFreshness`)
- modificar: `internal/ui/cli/doctor.go` (flag `--check-updates`)
- modificar: `internal/ui/cli/root.go` (wire do setter)
- testar: `internal/usecase/doctor_audit_test.go`

**Comportamento**

- Sem flag: o check atual fica intacto — binário `dev` num checkout com tag
  continua WARN ("embedded templates are stale"; `envctlBinaryFresh` classifica
  `dev` como not-fresh de propósito, `doctor_system.go:668-682`). O INFO novo só
  preenche o silêncio de hoje: `envctlVersion == "dev"` **e** `repoRootFinder()`
  falhar (binário dev sem checkout acima para comparar).
- Com `--check-updates`: se `latest > current`, **WARN** com
  `run 'envctl self-update'`; se igual, **OK**; erro de rede → nenhum diagnóstico
  (apenas log debug). `dev` é sempre "behind", então fora de checkout vira WARN.
- `versionLess` cobre `dev`, `v2.1.5`, `2.1.5`, `2.2.0`, `10.0.0` (numérico, não
  lexicográfico).

**Passos**

- [ ] `TestAuditEnvctlFreshnessInfoForDevOutsideRepo` — dois casos num teste só
  (RED): sem repo acima → **INFO**; repo com tag → **WARN** como hoje
  (`TestAuditEnvctlFreshnessWarnsWhenStale`, `doctor_audit_test.go:284`,
  não pode regredir).
- [ ] Implementar o INFO até GREEN.
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
diferença; sem a flag, hosts com checkout não mudam e um `dev` fora de checkout
ganha o INFO que hoje é silêncio.

**Commit sugerido:** `feat(doctor): flag dev builds and stale releases`

---

### T3 — Guard de CHANGELOG no Release Pipeline

**Escopo:** impedir tag sem a seção correspondente no CHANGELOG. Só isso — o
ritual de release continua manual.

**Arquivos**

- modificar: `.github/workflows/release.yml`
- modificar: `docs/guides/provisioning.md` (documentar a ordem: PR da seção → merge → dispatch)

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

**Passos**

- [ ] Adicionar o step de validação.
- [ ] Simular a falha do guard: `grep -qE '^## \[9\.9\.9\]' CHANGELOG.md; echo $?` → 1.
- [ ] `actionlint`.

**Verificação**

```bash
actionlint
grep -qE '^## \[9\.9\.9\]' CHANGELOG.md; echo "exit=$?"   # exit=1 → guard rejeitaria
```

**Rollback:** reverter o PR; o workflow volta ao comportamento anterior.
**Resultado esperado:** dispatch com versão sem seção falha antes de criar a tag.

**Commit sugerido:** `ci(release): require the CHANGELOG section before tagging`

---

### T4 — CHANGELOG: uma `[Unreleased]` + teste rodando em PR de docs

**Escopo:** consolidar as 3 seções `[Unreleased]` e fazer o teste de estrutura
rodar onde o drift nasce.

**Arquivos**

- modificar: `CHANGELOG.md`
- criar: `internal/infra/embedded/changelog_test.go`
- modificar: `.github/workflows/ci.yml` (remover o bloco `paths-ignore`)

**Movimentos (com evidência já coletada)**

- Bloco da linha 359 (CachyOS gaming full restore) → `## [1.11.0]`
  (`git tag --contains 801aeda` → `v1.11.0`). Deduplicar contra os bullets já
  existentes em 1.11.0.
- Bloco da linha 522 (ubuntu-server performance baseline) → `## [1.10.0]`
  (`git tag --contains 753ae4f` → `v1.10.0`). Deduplicar.
- `## [Unreleased]` fica só na linha 14, vazia até o próximo release.

**Teste (`changelog_test.go`, usa `repoRootForDocs`):**

- exatamente uma `## [Unreleased]`;
- headings `## [x.y.z]` em ordem estritamente decrescente (numérica).

**Por que remover `paths-ignore` em vez de criar `docs.yml`:** o filtro atual pula
`docs/**`, `**.md`, `CHANGELOG.md` e `spec-agent/**`, o que significa que
`docs_claims_test.go` (já existente) nunca roda no PR que o quebraria. Apagar o
filtro é uma remoção de caso especial; um workflow novo é mais uma peça para
manter. Custo aceito: CI completo em PR docs-only.

**Passos**

- [ ] Mover/deduplicar os dois blocos.
- [ ] Escrever o teste e observar RED com o estado atual (3 Unreleased).
- [ ] Consolidar até GREEN.
- [ ] Remover o `paths-ignore` e validar com `actionlint`.

**Verificação**

```bash
grep -c "^## \[Unreleased\]" CHANGELOG.md          # 1
go test ./internal/infra/embedded/ -run 'TestChangelog|TestDocumentation' -v
actionlint
```

**Rollback:** reverter o PR; o CHANGELOG é histórico, sem impacto em runtime.
**Resultado esperado:** uma única `[Unreleased]`; PR docs-only passa a rodar os
testes de doc.

**Commit sugerido:** `docs(changelog): consolidate Unreleased and guard its shape`

---

### T5 — PATH: um parser, funções puras

**Escopo:** eliminar a duplicação `pathListsDir`/`path_entry.go` e tirar o estado
do host de dentro das funções que decidem os rc files.

**Arquivos**

- modificar: `internal/infra/environment/path_entry.go` (exportar `PathListsDir`)
- modificar: `internal/infra/environment/env_manager.go` (puros + call sites)
- modificar: `internal/infra/environment/env_manager_test.go`
- modificar: `internal/usecase/doctor_system.go` (usar `environment.PathListsDir`)
- modificar: `internal/usecase/doctor_system_test.go` (mover o teste de tabela)
- **não mexer:** `internal/usecase/provision_bootstrap_test.go` — o
  `t.Setenv("PATH", ...)` fica: os métodos do manager continuam resolvendo
  `fishAvailable()` no call time, e com fish instalado o composite path depende
  dele (o comentário em linha no `:22-28` já explica).

**Interfaces**

```go
// environment
func PathListsDir(pathVal string, sep byte, dir string, foldCase bool) bool
func rcFiles(home string, fish bool) []shellRC
func withinHome(home, path string) bool
```

- Sem options, sem campos novos: `WindowsEnvManager` continua `struct{}`. Os
  métodos resolvem `home`/`fish` uma vez e chamam as funções puras.
- `doctor_system.go` troca as 3 chamadas por `environment.PathListsDir`.
- Os testes do pacote `environment` chamam as funções puras com argumentos
  literais (sem `t.Setenv` de host). O teste de bootstrap do `usecase`
  (`TestPathStepPersistsEachDirOnce`) mantém o seu `t.Setenv("PATH", ...)`: a
  composição `pathStep → EnsurePathEntry → rcFiles(resolveHome(),
  fishAvailable())` continua dependendo do fish do host.

**Passos**

- [ ] Mover `pathListsDir` + teste de tabela para `environment`; observar RED
  (símbolo ainda não existe) e GREEN.
- [ ] Tornar `rcFiles`/`withinHome` puros e ajustar os call sites.
- [ ] Ajustar os testes do `environment`; remover os `t.Setenv` que a pureza
  tornou desnecessários lá (o de `provision_bootstrap_test.go` fica).
- [ ] Rodar o gate completo.

**Verificação**

```bash
go test ./internal/infra/environment/ -run 'Path|RCFiles|WithinHome' -v
go test ./internal/usecase/ -run 'Path' -v
go build ./... && go vet ./... && golangci-lint run --new-from-rev=origin/main
```

**Rollback:** reverter o PR; não há migração de dados.
**Resultado esperado:** um único parser; os testes de PATH do pacote
`environment` passam sem depender de fish/HOME do host; o teste de bootstrap
continua POSIX-only via `Setenv`, como hoje.

**Commit sugerido:** `refactor(path): share one PATH parser and make the rc helpers pure`

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

- modificar: `docs/os-and-agent-matrix.md` (linhas 37 e 189)
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

### T8 — Apagar código morto e parâmetros que fingem ser opcionais

**Escopo:** remover 150 linhas sem chamador e desfazer duas abstrações que só
existem para o teste passar. Zero mudança de comportamento.

**Evidência:** `go run golang.org/x/tools/cmd/deadcode@latest ./...` → 11 funções
inalcançáveis; cada uma conferida com `rg` para separar "sem chamador" de "só teste".

| Símbolo | Local | Linhas | Situação |
|---|---|---|---|
| `TweaksManager.EnsureTweaks` | `internal/infra/windows/tweaks_manager.go:888` | 50 | nenhum chamador |
| `ProvisionPerformanceUseCase.ExecutePerformance` | `internal/usecase/provision_performance.go:68` | 10 | nenhum chamador (o real é `...WithOptions`) |
| `saveManifestFile` + `SaveSkills` + `SaveGitConfigs` | `internal/infra/embedded/manifest_repo.go:552-575` | 24 | só entre si |
| `entity.DetectedDistroID` / `DetectedDistroVersion` | `internal/domain/entity/platform.go:47-55` | 9 | nenhum chamador |
| `performance.AssessJournaldPolicy` | `internal/infra/performance/journald_manager.go:149-174` | 26 | só testes |
| `entity.AssessRemovalState` | `internal/domain/entity/debloat.go:128-140` | 13 | só teste |
| `entity.HardwareState.AdoptedDiskSwap` | `internal/domain/entity/swap.go:176-182` | 7 | só teste |
| `skillDescription` | `internal/usecase/skill_contract.go:103-113` | 11 | só teste |

**Abstrações a desfazer**

1. `NewDoctorAuditUseCase(..., performanceInspectors ...*performance.PerformanceInspector)`
   (`doctor_audit.go:47`) — variádico para simular parâmetro opcional, seguido de
   `if len(...) > 0`. Vira parâmetro normal; os 3 call sites de teste passam `nil`.
2. `executePerformance(..., dryRun bool, allowDebloat bool, optionsNoReexec bool,
   optionsTimezone string)` (`provision_performance.go:92`) — desmonta
   `PerformanceOptions` em 3 primitivos: 7 parâmetros, ciclomática 48. Passa a
   receber `PerformanceOptions`, que já existe em `performance_options.go:50`.

**Também:** `version.go` (318 linhas, 20 funções) mistura semver, consulta npm/uv,
capacidade do fzf e arquivamento de binários shadowed. Mover
`fzfSupportsWalker`/`fzfHasWalker` para `provision_bootstrap.go` (são métodos de
`ProvisionBootstrapUseCase`) e `archiveShadowedUserCopies`, `isProviderRuntime` e
`defaultLatestProviderVersion` para `provision_providers.go`. `version.go` fica só
com versão.

**Não entra:** `ProvisionShellUseCase.Execute` (ciclomática 85,
`provision_shell.go:77`). É refactor de risco médio sem ganho de comportamento —
YAGNI vale para limpeza também. Fica registrado para quando a função for tocada.

**Passos**

- [ ] `rg` de confirmação (0 chamadores fora de teste) para cada símbolo antes de apagar.
- [ ] Apagar os 8 símbolos e os testes que só existiam para eles; a asserção de
  `skillDescription` passa a exercitar `parseSkillFrontmatter`, o caminho real.
- [ ] Trocar o variádico por parâmetro normal; ajustar os 3 call sites.
- [ ] `executePerformance` recebe `PerformanceOptions`; apagar o wrapper morto.
- [ ] Mover os helpers de `version.go`.
- [ ] Rodar o gate.

**Verificação**

```bash
go run golang.org/x/tools/cmd/deadcode@latest ./...   # sem saída para internal/
go build ./... && go vet ./... && go test -count=1 ./...
golangci-lint run --new-from-rev=origin/main
envctl doctor   # 0 warnings na máquina gerenciada
```

**Rollback:** reverter o PR.
**Resultado esperado:** ~150 linhas de produção removidas, 1 variádico a menos,
`executePerformance` com 4 parâmetros em vez de 7, comportamento idêntico.

**Commit sugerido:** `refactor: delete unreachable helpers and unwrap optional parameters`

---

### T9 — Enxugar o `envctl-verify`: remover o cache de estado

**Escopo:** remover o cache de estado do gate, cuja motivação original foi apagada
do projeto. **Não** remove o log de trilha — ver "não entra".

**Evidência**

- O cache nasceu em `afa5dd1` ("chain every repository hook") para rodar **a cada commit**.
- O v2 não instala hook nenhum: `manifests/shell.yaml` declara "invoked explicitly
  per project; installs no hooks and gates nothing automatically".
- O cache só liga quando `.commandcode/verify.sh` **não** existe
  (`configs/bin/envctl-verify:377`) — ou seja, nas execuções explícitas, onde ~2s
  não mudam nada.
- Custo atual medido: 65 linhas (320-384) + 5 (711-715) = **70 de bash**; 3 testes
  de cache = **98 linhas de Go** (37+35+26); parágrafo de doc ≈ 9. Símbolos
  envolvidos: `MAX_UNTRACKED_HASH_BYTES`, `stamp_file`, `has_oversized_untracked`,
  `local_tool_state`, `state_hash`, `current_state`.

**Contra-argumento registrado:** o subagente `verifier` roda o gate várias vezes
por sessão e o cache economiza ~2s por chamada. Se isso virar dor real, o cache
volta — com o gatilho correto (hook), não ligado justamente quando não há hook.

**Não entra:** `trail()` / `~/.envctl/verify.log` (15 linhas). Nada no código lê o
arquivo, mas é um log para o operador e responde "esse gate já pegou algo?".
Cortar seria over-correction: 15 linhas com consumidor humano declarado.

**Arquivos**

- modificar: `configs/bin/envctl-verify` (linhas 320-384 e 711-715)
- modificar: `internal/usecase/verify_script_test.go` (remover os 3 testes de cache)
- modificar: `docs/verification.md` (remover "Cache por estado da árvore" e a menção a "cache quente")

**Passos**

- [ ] Remover o bloco e o stamp; `current_state` deixa de existir.
- [ ] Remover `TestVerifyScriptDoesNotCacheUnavailablePackageLint`,
  `TestVerifyScriptCacheIncludesBaseRef` e `TestVerifyScriptDoesNotCacheRemovedLocalTool`.
- [ ] Ajustar a doc.
- [ ] Rodar o script neste repo nos 3 modos.

**Verificação**

```bash
bash -n configs/bin/envctl-verify
shellcheck configs/bin/envctl-verify
go test ./internal/usecase/ -run TestVerifyScript -v
envctl-verify --dry-run
envctl-verify --static
```

**Rollback:** reverter o PR; nenhum estado persistido além do stamp em `.git/`,
que pode ser apagado.
**Resultado esperado:** ~70 linhas a menos no gate (~177 no PR inteiro, contando
os 98 de teste e ~9 de doc); `--static`, `--git-push` e `--dry-run` com o mesmo
veredito de antes.

**Commit sugerido:** `refactor(verify): drop the state cache the hooks used to need`

---

## 6. Ordem e dependências

| Ordem | Task | Depende de | PR sugerido |
|---|---|---|---|
| 1 | T1 + T2 self-update e frescor | — | `feat/self-update` |
| 2 | T3 guard de release | — | `ci/release-guard` |
| 3 | T4 CHANGELOG + CI de docs | — | `docs/changelog-hygiene` |
| 4 | T5 PATH unificado | — | `refactor/path-parser` |
| 5 | T6 órfãos | — | `feat/doctor-unmanaged` |
| 6 | T7 docs counts | — | `docs/no-hardcoded-counts` |
| 7 | T8 código morto + params | — | `refactor/dead-code` |
| 8 | T9 cache do gate | — | `refactor/verify-cache` |

T1 e T2 vão no mesmo PR: T2 só consome `Latest` e `versionLess` de T1, e separar
duplicaria o wire no `root.go`. Cada PR é mergeável e revertível sozinho.

T8 é o melhor custo/benefício e não depende de nada: ~150 linhas removidas com
zero mudança de comportamento, provado por `go build`/`go test`. Pode ser o
primeiro PR se a prioridade for começar por risco zero.

## 7. Definition of Done

- [ ] As 8 tasks têm PR mergeado com evidência de gate no corpo.
- [ ] `go build ./...`, `go vet ./...`, `go test -count=1 ./...`,
  `golangci-lint run --new-from-rev=origin/main` verdes.
- [ ] `envctl doctor` sem flags na máquina gerenciada continua `0 warnings`.
- [ ] `envctl self-update --check` reporta current/latest e não escreve; checksum
  adulterado é rejeitado antes de escrever; instalação válida deixa `.bak`.
- [ ] `doctor --check-updates` avisa release antiga e fica silencioso em erro de rede.
- [ ] `doctor --unmanaged` reporta o quarteto restante em INFO; sem a flag nada muda.
- [ ] Release Pipeline falha com versão sem seção no CHANGELOG.
- [ ] `grep -c "^## \[Unreleased\]" CHANGELOG.md` → 1; PR docs-only roda os testes de doc.
- [ ] `rg "\b[0-9]+ pkgs\b" docs/` → vazio.
- [ ] Um parser de PATH só; os testes de PATH do `environment` não dependem de fish/HOME do host.
- [ ] `deadcode ./...` sem saída para `internal/`.
- [ ] `executePerformance` recebe `PerformanceOptions`; `NewDoctorAuditUseCase` sem variádico.
- [ ] `envctl-verify` sem o cache de estado; `--static`, `--git-push` e `--dry-run`
  com o mesmo veredito de antes.
- [ ] Nenhuma flag ou arquivo previsto em §2.1 aparece no diff.
- [ ] Nenhum segredo no diff; nenhum `git add -A`; worktrees removidos sem `--force`.

## 8. Rollback

- Cada task é um PR independente: `git revert <merge>` restaura o comportamento.
- Self-update: o binário anterior fica em `envctl.bak.YYYYMMDD-HHMMSS`; restaurar
  é copiar de volta. Nenhuma migração de dados, nenhum cache a limpar.
- Flags novas são default off; remover o PR não deixa estado persistido.
- CHANGELOG/docs: reversão textual, sem impacto em runtime.
- T8/T9: reversão de código; o único estado persistido é o stamp em `.git/envctl-verify.stamp`,
  que pode ser apagado.
- Nenhuma task desinstala pacote ou altera manifesto existente.
