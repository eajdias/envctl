# Verificação Local (Quality Gates)

O `envctl` provisiona um verificador único e o conecta aos dois pontos por onde uma
mudança pode escapar da máquina sem ser checada. A ideia: o erro aparecer em segundos
no seu terminal, não minutos depois no CI.

---

## 🧩 As Duas Camadas

| Camada | Onde | Quando dispara | Efeito |
| :--- | :--- | :--- | :--- |
| **Hook `Stop` do CommandCode** | `~/.commandcode/settings.json` | Ao fim de cada turno do agente | Roda os **checks estáticos** (o que um editor diria ao salvar). Findings de lint/formatação são informativos; builds, vets, testes e comandos explícitos podem bloquear |
| **Pre-push do git** | `~/.config/git/hooks/pre-push` (`core.hooksPath`) | Antes de qualquer `git push` | Roda o **gate completo, com a suíte de testes**. Somente falhas bloqueantes abortam o push; advisories são mostradas e registradas |

O hook de turno gateia o repositório do **cwd da sessão**: abra o `cmd` dentro do
projeto para o feedback automático valer ali. O pre-push é global e independe do agente.

**Os outros hooks também são encadeados.** `core.hooksPath` faz o git ignorar o
`.git/hooks` de *todo* repositório, então o envctl instala um delegator e um shim para
`pre-commit`, `prepare-commit-msg`, `commit-msg`, `post-commit`, `post-checkout` e
`pre-rebase`, que devolvem o controle ao hook local do projeto. Repositórios husky não são
afetados de qualquer forma: eles definem `core.hooksPath` **local**, e a config do
repositório vence a global.

---

## ✅ O Que o Verificador Roda

`~/.local/bin/envctl-verify` detecta a stack pelo conteúdo do repositório e roda **só as
stacks presentes**. O escopo continua sendo:

- **Linters e formatadores** rodam apenas nos **arquivos que a mudança toca** (committed-não-pushed
  + working tree) — o mesmo princípio do `--new-from-rev` do golangci-lint. `gofmt -l .` é a
  exceção: por ser uma operação de módulo, varre o repositório inteiro.
- **Type checks e testes** rodam no repositório inteiro, porque é ali que está o sinal.

A política de severidade é explícita:

| Severidade | Checks | Efeito |
| :--- | :--- | :--- |
| **Bloqueante** | `.commandcode/verify.sh`, script `lint` declarado em `package.json`, `go build`, `go vet`, testes, `tsc`, `mypy` e Pester | Exit `2` quando falham |
| **Advisory** | Linters e formatadores inferidos: `gofmt`, `golangci-lint`, ESLint sem script explícito, Prettier, Ruff, SQLFluff, ShellCheck, shfmt, Hadolint e PSScriptAnalyzer | Diagnóstico e `ADVISORY` na trilha; não bloqueia |
| **Skip** | Ferramenta ausente ou inutilizável no ambiente | Resumo nomeia o skip; não é falha nem advisory |

O verifier não edita o projeto. Ele apenas informa; um agente LLM ou o próprio time decide
se uma diretriz advisory é relevante para aquele repositório.

| Stack | Detectada por | Checks |
| :--- | :--- | :--- |
| **Go** | `go.mod` | `gofmt -l .` (advisory) · `go build` · `go vet` · `go test` · `GOOS=windows go build/vet` · `golangci-lint --new-from-rev` (advisory) |
| **Node/TS** | `package.json` | `package.json` `lint` via pnpm/npm/yarn/bun (bloqueante, quando declarado) · `tsc --noEmit` (com `tsconfig.json`) · ESLint local nos arquivos alterados (advisory; **sem** `eslint.config.{js,mjs,cjs,ts,mts,cts}`) · `prettier --check` nos alterados (advisory, quando há config) |
| **Python** | `pyproject.toml`/`setup.py`/`requirements.txt` | `ruff check` nos alterados (advisory) · `mypy .` (só se o projeto configurou) · `pytest -q` (exit 5 = "nenhum teste" não é falha) |
| **SQL** | `*.sql` alterados | `sqlfluff lint` (advisory, só com `.sqlfluff`/`[tool.sqlfluff]` definindo dialeto) |
| **Shell** | `*.sh`/`*.bash` **ou shebang** | `shellcheck` · `shfmt -d` (advisory) |
| **Docker** | `Dockerfile*` alterado | `hadolint` (advisory) |
| **PowerShell** | `*.ps1`/`*.psm1` alterados | `Invoke-ScriptAnalyzer` (advisory) · `Invoke-Pester -CI` quando há `*.Tests.ps1` (bloqueante) |

**Script explícito do projeto.** Quando `package.json` declara `scripts.lint`, o verifier
executa `pnpm run lint`, `npm run lint`, `yarn lint` ou `bun run lint` conforme o
`packageManager`/lockfile; sem metadata, `npm` é o fallback. Esse comando é a autoridade do
projeto e pode abortar o push. Se
não houver script, o binário local do ESLint é apenas um fallback advisory; arquivos de
configuração do próprio ESLint são excluídos dessa dedução.

**Ferramenta resolve em cascata:** binário do projeto primeiro (`node_modules/.bin`,
`.venv/bin` ou `.venv/Scripts/*.exe` no Windows), depois `uv run --no-sync` para Python
(nunca instala nada), depois o PATH — a versão que o projeto fixou ganha da global.

**Ferramenta ausente não é falha.** Se o binário não existe, o check opcional é omitido; se
ele existe mas não consegue rodar naquele projeto (shim do Volta sem dependência local,
`uv run` sem virtualenv), o check é **skip** e aparece nomeado no resumo do push. No
`--dry-run`, o script `package.json lint` com package manager indisponível aparece como
`[skip]`.

Para uma stack não coberta — ou para uma definição de "pronto" própria —, um
`.commandcode/verify.sh` executável no projeto substitui a detecção acima e continua
bloqueante.

---

## ⚙️ Variáveis de Controle

| Variável | Padrão | Efeito |
| :--- | :--- | :--- |
| `ENVCTL_SKIP_VERIFY` | `0` | `1` desliga a verificação naquela execução (push pontual) |
| `ENVCTL_VERIFY_MAX_LINES` | `25` | Linhas de saída por check no relatório (mantém o contexto enxuto) |

Saída enxuta por design: uma execução sem findings fica silenciosa; uma execução advisory-only
imprime o diagnóstico, registra `ADVISORY` e retorna 0; uma falha bloqueante traz só o começo
de cada falha.

---

## 🔀 Modos, Cache e Trilha

| Modo | O que roda |
| :--- | :--- |
| `--hook` (fim de turno) | **Só os checks estáticos** — build, type check, lint, formatação. A suíte de testes fica para o push, então um turno nunca espera por ela |
| `--git-push` | **O gate completo**, testes incluídos; somente findings bloqueantes abortam |
| `--dry-run` | Não roda nada: imprime as stacks detectadas e cada check detectado como `[blocking]`, `[advisory]` ou `[skip]`, incluindo a política que seria aplicada |

**Cache por estado da árvore (só no modo hook).** Se nada mudou desde a última execução
verde, o verificador sai em ~20ms em vez de rodar os checks de novo — um turno que apenas
leu arquivos não paga nada. O carimbo fica em `.git/envctl-verify.stamp`, nunca na árvore de
trabalho. Execuções com skip, overrides executáveis e arquivos untracked maiores que 1 MiB
não são cacheadas, para que uma ferramenta ou um conteúdo não observado não possa produzir um
verde falso. O fingerprint inclui `HEAD`, a ref de comparação (`origin/main`/`origin/master`),
a disponibilidade dos executáveis globais e o estado dos candidatos locais (`node_modules`,
`.venv` e `venv`, incluindo `Scripts/*.exe`); instalar, remover ou trocar uma ferramenta
invalida o cache.

**Trilha.** Falhas, advisories e skips são registrados em `~/.envctl/verify.log` (uma linha
agregada por execução, com timestamp, modo, repositório e resultado). Execuções verdes não
escrevem nada. É o que responde "esse gate já pegou algo?" com evidência em vez de memória —
e o que denuncia um check que vive sendo pulado.

---

## ⏱️ Sem Timeout — Fail-Fast

O verificador **não** impõe timeout: o conjunto completo roda em ~2s com cache quente
(gofmt 21ms, build 478ms, vet 133ms, testes 299ms, cross-compile Windows 638ms, lint
0,7s quente / 3,2s frio). Esperar minutos por um check travado seria desperdício em
automação — ou funciona, ou não funciona e reporta.

O hook de turno roda sob o teto do próprio engine do CommandCode, que é a única
proteção contra um check patológico. O pre-push não tem teto: ali o tempo gasto é o
tempo necessário antes de liberar um push.

---

## 🪝 Encadeamento com Hooks Locais

`core.hooksPath` sobrepõe `.git/hooks` de **todos** os repositórios, então o hook
deployado invoca primeiro o pre-push local do repositório (husky e afins continuam
funcionando) e só então roda o verificador. Um hook local que falhar continua abortando o
push antes mesmo do verificador global; se o verifier não estiver disponível, o hook global
falha fechado em vez de permitir um push sem gate.

---

## 🩺 Auditoria

O `doctor` verifica o wiring (script e hook presentes e executáveis) e reporta como
`Verify`. Um script ausente ou sem bit de execução aparece como `WARN` com o fix
`envctl run shell`.
