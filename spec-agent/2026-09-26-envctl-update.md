# Spec — `envctl update`: rotina de atualização do toolchain global

**Data:** 2026-09-26
**Escopo aprovado:** atualizar o **toolchain global** que o envctl provisiona (não as dependências
do projeto, não pacotes do SO).
**Default aprovado:** **aplica direto, sem perguntar.** As três stacks são user-local, sem sudo e
reversíveis, então automatizar é seguro; `--dry-run` é o preview. Gerenciadores de SO nunca são
automatizados.

---

## 1. Objetivo

Hoje, atualizar um tool global exige saber o mecanismo de cada um: `volta install typescript`,
`uv tool upgrade pytest`, `go install gopls@latest` na mão. O `run providers` (fase 0) já faz
check-vs-latest **apenas dos CLIs de agente** (`opencode`, `command-code`). As outras 37 tools
do manifesto ficam sem caminho.

`envctl update` fecha isso: um comando que sabe, por tool, **qual é a versão instalada, qual é a
latest, e qual comando a atualiza** — e só executa se a pessoa mandar.

## 2. Fora de escopo (decidido)

| item | por quê |
|---|---|
| Dependências do projeto (`go get -u`, `pnpm update`, `pip -U`) | a skill `infra.md` define a fronteira: *"provisionar serviço de um repo é do repo"* |
| Atualização do SO (`paru -Syu`, `apt upgrade`) | risco de kernel/bootloader; e no Arch, **upgrade parcial de pacman é proibido** (quebra o sistema) |
| Qualquer coisa do winget/Windows | canal do SO, idem |

> **Guard de correção (Arch):** os 50 pacotes `type: pacman` do manifesto **não** entram no
> updater. Atualizar um subconjunto via `pacman -S <pkg>` é *partial upgrade*, que o Arch
> proíbe explicitamente. Pacotes do manifesto só sobem com a atualização do sistema, que aqui é
> manual por decisão de escopo. Isso precisa estar escrito no `--help`, não implícito.

## 3. Superfície atualizada (medida no manifesto)

| grupo | como está no manifesto | como checa latest | como atualiza |
|---|---|---|---|
| **Node/tooling npm** | `packages.yaml` `type: volta` (15) + `lsp.yaml` `install_type: volta` (11) = **26** | `npmLatest()` — `registry.npmjs.org/<pkg>/latest` (já existe, `provision_providers.go:579`) | `volta install <pkg>@latest` |
| **Python (uv tools)** | `packages.yaml` `type: pip` (10) | `uv tool list` (versão instalado) + `uv tool upgrade` (o próprio uv resolve a latest) | `uv tool upgrade <tool>` |
| **Go** | `lsp.yaml` `install_type: go` (1 — `gopls`) | `go list -m -f {{.Version}}` no módulo, ou o `--version` do binário | `go install <pkg>@latest` |
| **OS / Windows / pip-do-sistema** | `winget` 30, `apt` 44, `pacman` 50, `paru` 1 | — | fora de escopo |

**Fora do inventário do manifesto, mas global e do mesmo domínio:** `golangci-lint`, `go`,
`gh`, `yq`, `delta`, `fd` e `fzf` são instalados pelo **bootstrap** (`provision_bootstrap.go`),
não pelo manifesto. `golangci-lint` tem instalador oficial (`install.sh`), `go` tem tarball
oficial, e o resto é gerenciador de SO — **esses ficam de fora** e o comando deve dizer isso, em
vez de silenciar.

## 4. Comportamento

```
envctl update                 # aplica as atualizações de Node/Python/Go, sem perguntar
envctl update --dry-run       # imprime o comando exato de cada update, não executa
envctl update --only volta    # filtra por grupo (volta|uv|go)
envctl update --list          # só o inventário, sem consultar latest
```

**Sem fase de prompt.** As três stacks são user-local (sem `sudo`, sem tocar `/etc`, sem
bootloader) e cada update é unitário e reversível — a versão anterior fica registrada no report.
Por isso o default aplica direto. A linha que protege contra automação destrutiva continua valendo
para o **SO**, que está fora do inventário.

### Fase 1 — inventário
Para cada tool elegível: `installed` → `available`, agrupado, com o comando que faria a mudança.
Reaproveita `installedVersion()` e `npmLatest()` (já existem em
`provision_providers.go:548` e `:579`).

### Fase 2 — apply
Executa pelo mesmo `runWithToolchain()` que o resto do projeto usa (mesmo PATH resolvido, mesma
captura de output). Confirma o resultado: **re-checa a versão depois** e reporta `installed ->
agora` ou a falha com o comando para rodar à mão. Um update que falha em silêncio é pior que
não tentar. **Uma tool que falha não interrompe as outras** — cada update é unitário.

## 5. Arquivos afetados

| arquivo | mudança |
|---|---|
| `internal/usecase/update.go` | novo — `UpdateUseCase`, relatório por grupo, offer/apply |
| `internal/usecase/update_test.go` | novo — TDD por grupo, dry-run, latest indisponível, falha isolada |
| `internal/ui/cli/update.go` | novo — comando cobra + flags |
| `internal/usecase/app_context.go` | novo wiring no container |
| `internal/usecase/doctor_audit.go` | **só** se reutilizar helper; sem check novo (update não é estado de saúde) |
| `configs/bin/envctl-verify` | **não** toca (gate é de projeto, update é de ambiente) |
| `docs/os-and-agent-matrix.md` | linha nova em §2 (o que envctl faz) + menção no `Agente custom`-style checklist |
| `docs/doctor-and-idempotency.md` | `update` ≠ `doctor`: o primeiro muda, o segundo só relata |
| `AGENTS.md` / `configs/REFERENCE.md` | linha do comando novo |
| `CHANGELOG.md` | `[Unreleased]` (release-please gera a versão) |

**Fora:** `manifests/**` — o updater **lê** o manifesto, não o escreve. Nenhuma versão fica
pinada; "latest" é sempre consultado na hora.

## 6. Riscos e guards

| risco | guard |
|---|---|
| update de tool quebra o gate (ex.: `golangci-lint` novo mais estrito) | `golangci-lint` **fora** do inventário (seu instalador e `curl \| sh`); quem roda o gate e o `envctl-verify` |
| partial upgrade no Arch | ferramentas de `pacman` fora do escopo, escrito no `--help` e no report |
| aplicar sem perguntar em script/CI | e o default **e** e seguro: as tres stacks sao user-local, sem sudo, unitarias e reversiveis. O que e destrutivo (SO) e justamente o que nao entra |
| rede caiu ao consultar latest | `npmLatest()` ja devolve `""` => a linha vira "latest desconhecido" e **nada e atualizado** |
| `volta install pkg@latest` re-resolve e muda a toolchain inteira | usar o mesmo `runWithToolchain`/`ensureProviderCLI` que a fase 0 já usa, sem inventar caminho novo |
| tool some do manifesto entre check e apply | reler o manifesto no apply e casar por `install_target`, não por índice |

## 7. Rollback

- Cada update é **unitário por tool**: se `uv tool upgrade pytest` falhar, nada mais é tocado, e o
  comando anterior fica no lugar (uv guarda a versão anterior em `~/.local/share/uv/tools/`).
- `--dry-run` é o preview; sem ele o default aplica, e o report sempre mostra a versão anterior.
- Reverter de propósito: `volta install <pkg>@<versão-anterior>` (o report mostra a versão
  anterior, então ela fica registrada) ou `uv tool install <tool>==<versão-anterior>`.

## 8. Verificação

TDD, um comportamento por vez:

```sh
go test ./internal/usecase/ -run TestUpdate -v     # dry-run não executa; OS nunca entra no inventário
go build ./... && go vet ./... && go test ./...
golangci-lint run --new-from-rev=origin/main
bash configs/bin/envctl-verify --git-push
```

Testes que **precisam** existir, porque cobrem os guards:

1. `--dry-run` imprime o comando e não muda nada (comparar versão antes/depois)
2. o caminho de apply **não tem prompt** (o default aplica; nada no código pergunta)
3. latest indisponível ⇒ linha "desconhecido", nunca "atualizado"
4. falha no apply ⇒ as outras tools do grupo continuam como estavam
5. `pacman`/`apt`/`winget` **nunca** aparecem no inventário (o guard do Arch)
6. tool que sumiu do manifesto entre check e apply é ignorada, não aplicada por índice

## 9. Ordem sugerida

1. `--list` (inventário puro, sem rede) — mostra o terreno e já é útil
2. fase 1 report (rede, sem efeito)
3. apply por grupo, um grupo por vez
4. offer interativo por último (o que tem CI é o resto)
