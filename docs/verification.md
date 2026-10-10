# Verificação Local (Quality Gates)

O `envctl` provisiona um verificador único — o `envctl-verify` — para ser rodado
**explicitamente** num projeto. A ideia continua a mesma: o erro aparece em segundos
no seu terminal, não minutos depois no CI.

---

## 🧩 O Modelo (v2): nenhuma instalação automática

**Desde a v2 o envctl não instala hooks nem gates globais.** Um `core.hooksPath`
global sobrepõe o `.git/hooks` de *todo* repositório — o `pre-commit`/`commit-msg`
dos outros projetos eram silenciados — e um hook de turno global gateava sessões
de qualquer repositório. Interferir em projetos que não pediram é inaceitável:
nada disso existe mais.

| Camada | Onde | Quando dispara | Efeito |
| :--- | :--- | :--- | :--- |
| **Invocação explícita** | `envctl-verify --static` / `--git-push` / `--dry-run` | Quando **você** (ou um agente) decide rodar | `--static` = checks estáticos · `--git-push` = gate completo com testes · `--dry-run` = mostra o que seria rodado |

Quem quiser automatizar roda o `envctl-verify` explicitamente a partir da automação
que o **projeto** já tiver (script, CI, task runner) — o envctl não instala nem
recomenda hooks: automação é assunto de cada projeto.

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
projeto e pode derrubar o gate. Se
não houver script, o binário local do ESLint é apenas um fallback advisory; arquivos de
configuração do próprio ESLint são excluídos dessa dedução.

**Ferramenta resolve em cascata:** binário do projeto primeiro (`node_modules/.bin`,
`.venv/bin` ou `.venv/Scripts/*.exe` no Windows), depois `uv run --no-sync` para Python
(nunca instala nada), depois o PATH — a versão que o projeto fixou ganha da global.

**Ferramenta ausente não é falha.** Se o binário não existe, o check opcional é omitido; se
ele existe mas não consegue rodar naquele projeto (shim do mise sem dependência local,
`uv run` sem virtualenv), o check é **skip** e aparece nomeado no resumo da execução. No
`--dry-run`, o script `package.json lint` com package manager indisponível aparece como
`[skip]`.

Para uma stack não coberta — ou para uma definição de "pronto" própria —, um
`.commandcode/verify.sh` executável no projeto substitui a detecção acima e continua
bloqueante.

---

## ⚙️ Variáveis de Controle

| Variável | Padrão | Efeito |
| :--- | :--- | :--- |
| `ENVCTL_SKIP_VERIFY` | `0` | `1` desliga a verificação naquela execução (execução pontual) |
| `ENVCTL_VERIFY_TIMEOUT` | `120` no `--static` / `600` no `--git-push` | Teto (segundos) por check; um check travado falha como `TIMED OUT` |
| `ENVCTL_VERIFY_MAX_LINES` | `25` | Linhas de saída por check no relatório (mantém o contexto enxuto) |

Saída enxuta por design: uma execução sem findings fica silenciosa; uma execução advisory-only
imprime o diagnóstico, registra `ADVISORY` e retorna 0; uma falha bloqueante traz só o começo
de cada falha.

---

## 🔀 Modos, Cache e Trilha

| Modo | O que roda |
| :--- | :--- |
| `--static` | **Só os checks estáticos** — build, type check, lint, formatação. A suíte de testes fica para o `--git-push`, então uma chamada rápida nunca espera por ela |
| `--git-push` | **O gate completo**, testes incluídos; somente findings bloqueantes retornam exit ≠ 0 |
| `--dry-run` | Não roda nada: imprime as stacks detectadas e cada check detectado como `[blocking]`, `[advisory]` ou `[skip]`, incluindo a política que seria aplicada |

**Cache por estado da árvore (só no modo `--static`).** Se nada mudou desde a última execução
verde, o verificador sai em ~20ms em vez de rodar os checks de novo — uma chamada que apenas
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

## ⏱️ Timeout por Check — Fail-Fast

Cada check roda sob um timeout individual (`ENVCTL_VERIFY_TIMEOUT`; **120s** no
`--static`, **600s** no `--git-push`). Um check que trava é um check quebrado e **falha
rápido no teto** com o marcador `TIMED OUT`, em vez de ser aguardado para sempre. O
conjunto saudável continua rápido com cache quente (gofmt 21ms, build 478ms, vet 133ms,
testes 299ms, cross-compile Windows 638ms, lint 0,7s quente).

Quando o coreutils `timeout` não está disponível, o check roda sem teto (comportamento
anterior) em vez de falhar.

---

## 🩺 Auditoria

O `doctor` verifica que o verificador está deployado e que **nenhum resquício do wiring
global legado sobrou** (os antigos hooks de `~/.config/git/hooks`, que sobrepunham os
hooks de todos os repositórios). O verificador ausente ou sem bit de execução aparece
como `WARN` com o fix `envctl run shell`; qualquer resquício de hook global também é
`WARN`, com a instrução de remoção (`remove ~/.config/git/hooks` +
`git config --global --unset core.hooksPath`).
