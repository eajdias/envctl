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

`~/.local/bin/envctl-verify` detecta a stack do repositório e executa:

| Check | Por quê |
| :--- | :--- |
| `gofmt -l .` | Formatação — barato e sempre cobrado no CI |
| `go build ./...` | Compilação |
| `go vet ./...` | Análise estática da stdlib |
| `go test ./...` | Testes |
| `GOOS=windows go build/vet ./...` | Windows é alvo suportado: quebra de plataforma aparece aqui, não num runner do CI |
| `golangci-lint run --new-from-rev=origin/main` | **Mesmo gate do CI**: dívida legada nunca bloqueia um push, finding novo sempre bloqueia |

Repositórios de outra stack (Python, Node, ...) não são adivinhados: defina um
`.commandcode/verify.sh` executável no projeto e ele passa a ser o verificador.

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
