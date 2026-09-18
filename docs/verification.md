# Verificação Local (Quality Gates)

O `envctl` provisiona um verificador único e o conecta aos dois pontos por onde uma
mudança pode escapar da máquina sem ser checada. A ideia: o erro aparecer em segundos
no seu terminal, não minutos depois no CI.

---

## 🧩 As Duas Camadas

| Camada | Onde | Quando dispara | Efeito |
| :--- | :--- | :--- | :--- |
| **Hook `Stop` do CommandCode** | `~/.commandcode/settings.json` | Ao fim de cada turno do agente | Falha → `exit 2` e o **stderr com o diagnóstico volta para o modelo**, que corrige no mesmo turno |
| **Pre-push do git** | `~/.config/git/hooks/pre-push` (`core.hooksPath`) | Antes de qualquer `git push` | Falha → **push abortado**, venha o push do CommandCode, do opencode ou do terminal |

O hook de turno gateia o repositório do **cwd da sessão**: abra o `cmd` dentro do
projeto para o feedback automático valer ali. O pre-push é global e independe do agente.

---

## ✅ O Que o Verificador Roda

`~/.local/bin/envctl-verify` detecta a stack pelo conteúdo do repositório e roda **só as
stacks presentes**. Duas regras de escopo valem para todas elas:

- **Linters** rodam apenas nos **arquivos que a mudança toca** (comitted‑não‑pushed + working
  tree) — o mesmo princípio do `--new-from-rev` do golangci‑lint. Dívida legada não bloqueia
  um push; o que você acabou de escrever, sim.
- **Type checks e testes** rodam no repositório inteiro, porque é ali que está o sinal.

| Stack | Detectada por | Checks |
| :--- | :--- | :--- |
| **Go** | `go.mod` | `gofmt -l .` · `go build` · `go vet` · `go test` · `GOOS=windows go build/vet` · `golangci-lint --new-from-rev` |
| **Node/TS** | `package.json` | `tsc --noEmit` (com `tsconfig.json`) · `eslint` nos arquivos alterados (**só o binário local**, ele resolve os plugins do projeto) · `prettier --check` nos alterados (quando há config) |
| **Python** | `pyproject.toml`/`setup.py`/`requirements.txt` | `ruff check` nos alterados · `mypy .` (só se o projeto configurou) · `pytest -q` (exit 5 = "nenhum teste" não é falha) |
| **SQL** | `*.sql` alterados | `sqlfluff lint` (só com `.sqlfluff`/`[tool.sqlfluff]` definindo dialeto) |
| **Shell** | `*.sh`/`*.bash` **ou shebang** | `shellcheck` · `shfmt -d` |
| **Docker** | `Dockerfile*` alterado | `hadolint` |
| **PowerShell** | `*.ps1`/`*.psm1` alterados | `Invoke-ScriptAnalyzer` (Error/Warning) · `Invoke-Pester -CI` quando há `*.Tests.ps1` |

**Ferramenta resolve em cascata:** binário do projeto primeiro (`node_modules/.bin`,
`.venv/bin`), depois `uv run --no-sync` para Python (nunca instala nada), depois o PATH — a
versão que o projeto fixou ganha da global.

**Ferramenta ausente não é falha.** Se o binário não existe, o check é pulado em silêncio; se
ele existe mas não consegue rodar naquele projeto (shim do Volta sem dependência local,
`uv run` sem virtualenv), o check é **skip** e aparece nomeado no resumo do push.

Para uma stack não coberta — ou para uma definição de "pronto" própria —, um
`.commandcode/verify.sh` executável no projeto substitui a detecção acima.

---

## ⚙️ Variáveis de Controle

| Variável | Padrão | Efeito |
| :--- | :--- | :--- |
| `ENVCTL_SKIP_VERIFY` | `0` | `1` desliga a verificação naquela execução (push pontual) |
| `ENVCTL_VERIFY_MAX_LINES` | `25` | Linhas de saída por check no relatório (mantém o contexto enxuto) |

Saída enxuta por design: verde é silencioso, vermelho traz só o começo de cada falha.

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
funcionando) e só então roda o verificador.

---

## 🩺 Auditoria

O `doctor` verifica o wiring (script e hook presentes e executáveis) e reporta como
`Verify`. Um script ausente ou sem bit de execução aparece como `WARN` com o fix
`envctl run shell`.
