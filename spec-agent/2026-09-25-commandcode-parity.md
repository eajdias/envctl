# CommandCode Parity — as mesmas melhorias aplicadas ao OpenCode

> Passada separada e exclusiva do **CommandCode**. O lado OpenCode da branch
> `feat/subagent-orchestration-worktrees` está fechado; esta spec só toca a
> plataforma CommandCode (mais o mínimo de conteúdo compartilhado, sempre por
> seção por runtime).

## Goal

Levar ao CommandCode o mesmo conjunto de ganhos do lado OpenCode — skill
proativa, supervisão com toolset real, worktree auditável e validação de
config pelo `doctor` — **sem** perder o que o CommandCode já faz melhor (toolset
de background nativo, worktree gerenciado, `plan` já dispatchável) e sem
enfraquecer nenhum boundary de permissão.

## Evidence base (fonte primária, 2026-09-25)

- Docs oficiais: `https://commandcode.ai/docs/agents`, `/docs/background-tasks`,
  `/docs/worktrees`, `/docs/skills` (release atual = npm `command-code@1.65.4`).
- Binário instalado: `command-code@1.65.0` em
  `~/.volta/tools/image/packages/command-code/lib/node_modules/command-code/dist/cli.mjs`
  (grep confirma `RESERVED_AGENT_NAMES`, `agent_output`, `baseRef`, `enter_worktree`,
  frontmatter completo). **Docs são do release novo; a máquina tem 1.65.0 — toda
  afirmação de runtime foi conferida no bundle local antes de virar plano.**
- Lado OpenCode: `https://opencode.ai/v2/docs/skills`.

## Já está em paridade (nada a fazer)

| Ganho (OpenCode) | Situação no CommandCode | Prova |
|---|---|---|
| Skill carregada proativamente quando a description casa | Feito no mesmo commit das regras proativas | `configs/commandcode/AGENTS*.md` |
| Supervisão com o vocabulary real do runtime | Feito: skill com seções `## OpenCode V2` / `## CommandCode` | `configs/skills/subagent-supervision/SKILL.md` + teste por seção |
| Detecção de worktree `prunable`/`locked` no `doctor` | **Cobre os dois**: worktree gerenciado do CommandCode é um `git worktree add` real, então aparece em `git worktree list --porcelain` | grep `args:["worktree","add"]` no bundle + `internal/usecase/worktree_report.go` |
| Frontmatter de skill válido (name == diretório, description ≤1024) | Feito: o validador do envctl já exige `name == dirName`, e o CommandCode **pula** a skill silenciosamente quando isso falha | `internal/usecase/skill_frontmatter.go:76` |
| Delegação de planejamento | `plan` já é subagent registrado (`source:"bundled"`, `tools:["read_file"]`) e o `agent` tool cai nele sem nome explícito | bundle: `{name:"plan",…source:"bundled"}` |

## Gaps reais (o trabalho)

### Gap 1 — `doctor` não valida o schema de frontmatter de agente (paridade com `Config shape`)

`auditCommandCodeAgents` (`internal/usecase/doctor_audit.go:1603`) só checa
`name == filename` e nomes reservados. A doc define 10 campos com tipos e valores
 permitidos, e **todo erro aqui degrada a delegação em silêncio**:

| Campo | Erro silencioso documentado |
|---|---|
| `tools` ausente | agente fica **sem nenhuma tool** (default é nenhuma) |
| `tools`/`disallowedTools` com id inválido | id é simply ignorado; `agent`/`agent_output` nunca podem ser concedidos |
| `permissionMode` inválido | sessão vence; o valor é ignorado |
| `reasoningEffort` desconhecido | "dropped with a warning" e cai no default do model |
| `maxTurns` não-inteiro | ignorado (default 100) |
| `model` inexistente | cai no model da sessão |

Ou seja: um typo em `tools: read_file, read-file` deixa o agente quase cego e o
doctor fica verde. É exatamente o modo de falha que o check `Config shape`
eliminou do lado OpenCode.

**Tarefa 1** (código, testável): `validateCommandCodeAgentFrontmatter(dir, fm) []string`
em arquivo novo, regras derivadas da doc, integrada em `auditCommandCodeAgents`
(WARN com o campo culpado nomeado; chaves desconhecidas **não** avisam — a doc
diz que são ignoradas).

### Gap 2 — `planner` no CommandCode: recommendation is **not** to add it

| | OpenCode | CommandCode |
|---|---|---|
| quem planeja | `planner` (custom, dispatchável) | `plan` (built-in, já dispatchável) |
| pode escrever a spec? | sim, `edit spec-agent/**` | `plan` é read-only (`tools:["read_file"]`) |
| como escreveria a spec? | — | exigiria `write_file`/`edit_file` **sem escopo de path** (não existe `Edit(spec-agent/**)`), com `permissionMode: plan` as write tools somem |

Criar um `planner` custom no CommandCode daria **boundary mais fraco** (write no
repo inteiro) para ganhar a escrita do spec. Decisão recomendada: manter `plan`
read-only e deixar o **coordenador** materializar o arquivo em `spec-agent/` — a
spec sempre foi um artefato do coordenador. Custo de adicionar: regressão de
permissão. **Aguardando confirmação do usuário.**

### Gap 3 — Convenção de worktree: são doispathy que não converge

| | OpenCode | CommandCode |
|---|---|---|
| config | `worktree.directory: .worktrees` (projeto) | **não existe** setting de diretório |
| local | `.worktrees/<slug>` (dentro do repo, ignorado) | `~/.commandcode/worktrees/<repo>-<hash>/<name>` (fora do repo) |
| rota alternativa | — | `cmdc -w <caminho-absoluto>` usa o path **verbatim** |
| automação do modelo | — | `enter_worktree`/`/worktree` sempre no dir gerenciado |

Recomendação: documentar as duas rotas com honestidade em `git-workflow` +
`AGENTS` (`.worktrees/` para git/manual/OpenCode; `cmdc -w "$PWD/.worktrees/<slug>"`
para cair na mesma convenção; `/worktree` e `enter_worktree` no dir gerenciado, que
o `doctor` já enxerga). Sem config novo, porque não existe chave para isso.

### Gap 4 (opcional, otimização) — `when_to_use` e os controles de visibilidade

| Conceito | OpenCode | CommandCode |
|---|---|---|
| esconder do modelo | `metadata.opencode/autoinvoke: false` | `disable-model-invocation: true` |
| esconder do menu `/` | `slash: false` | `user-invocable: false` |
| gatilho extra p/ auto-invoke | (não documentado; vai na `description`) | `when_to_use` (anexado à description no catálogo) |

`when_to_use` só é útil no CommandCode e a árvore é compartilhada, então depende
de o loader do OpenCode tolerar chave desconhecida: a doc do OpenCode diz que
"aceita portability fields como `license`/`compatibility` mas não os interpreta",
o que **não** afirma tolerância genérica. Bloqueado até um teste empírico de 1
skill. Custo: `when_to_use` **aumenta o prompt** (é anexado à description de cada
skill), então só entra nas skills de roteamento se entrar. Prioridade baixa: é
otimização, não gap.

## Risks and Unknowns

- **Version skew docs↔binário**: doc 1.65.4 vs instalado 1.65.0. Mitigação: toda
  regra do Gap 1 foi conferida no bundle 1.65.0; a validação de `tools` usa a lista
  de ids observada no bundle, não só a da doc.
- **Lista de tool ids pode crescer**: id novo não pode virar WARN eterno (quebraria
  o contrato 0 WARN). Mitigação: lista desconhecida vira **INFO**, e só erro de
  *forma* (tipo/valor obviously inválido) é WARN.
- **`cmdc skills list --debug` no doctor**: o próprio runtime reporta skill pulada
  com motivo, mas o comando pode exigir auth e custar tempo. Fora do escopo; o
  validador do envctl já cobre o caso.
- **Mudança de conteúdo compartilhado** (frontmatter novo nas skills) tem efeito no
  OpenCode: por isso o Gap 4 nasce bloqueado, e qualquer coisa compartilhada passa
  por seção por runtime + teste por seção.

## Tasks

### Task 1 — Validador de frontmatter de agente do CommandCode

**Files**
- Create: `internal/usecase/commandcode_agent_schema.go`
- Create: `internal/usecase/commandcode_agent_schema_test.go`
- Modify: `internal/usecase/doctor_audit.go` (`auditCommandCodeAgents`)
- Modify: `docs/doctor-and-idempotency.md`

**Regras (da doc `docs/agents`, confirmadas no bundle 1.65.0)**
1. `permissionMode` ∈ {`default`, `accept-edits`, `yolo`, `plan`, `dont-ask`,
   `auto-accept`, `bypass`}; string vazia = herda.
2. `maxTurns` inteiro > 0 (aceita YAML int e string numérica).
3. `background` / `showOutput` booleanos.
4. `tools` / `disallowedTools`: string separada por vírgula/espaço ou lista YAML;
   `"*"` válido; cada id precisa estar no catálogo conhecido; `agent` e
   `agent_output` são **sempre inválidos** (o runtime remove).
5. `model` / `reasoningEffort` / `description`: string não vazia quando presente
   (o id do model não é validável offline → sem checagem de existência).

**TDD**
1. Testes de tabela: agente válido minimalista; cada campo inválido (um caso por
   regra); `tools` com id desconhecido → INFO, não WARN; `agent` em `tools` → WARN;
   `name` != filename continua coberto pelo teste existente.
2. `go test ./internal/usecase -run TestValidateCommandCodeAgentFrontmatter -v`
   → RED (função não existe).
3. Implementar o validador puro; integrar em `auditCommandCodeAgents` agregando os
   problemas no mesmo check `Agents` (1 linha agregada, como o padrão do projeto).
4. Rodar de novo → GREEN; `go test ./internal/usecase -v` para não regredir o
   check existente.

### Task 2 — Documentar as duas rotas de worktree e a decisão do planner

**Files**
- Modify: `configs/skills/git-workflow/SKILL.md` (tabela OpenCode × CommandCode)
- Modify: `configs/skills/subagent-routing/SKILL.md` (celula `plan` do CommandCode
  já está correta; explicitar por que **não** há `planner` custom lá)
- Modify: `configs/commandcode/AGENTS*.md` (3 variantes): rota de worktree +
  `plan` como subagent de planejamento
- Modify: `configs/REFERENCE.md` (tabela de visibilidade de skills por plataforma)
- Modify: `docs/os-and-agent-matrix.md` (assimetria nova #21)

Sem código. Sem config nova. Doc precisa dizer explicitamente que
`~/.commandcode/worktrees/` é o default do runtime e que a ponte para a nossa
convenção é o path absoluto em `-w`.### Task 3 (executada 2026-09-26) — `when_to_use`: medida, revertida, REAPLICADA com redesenho

**Encadeamento real:** (1) apliquei nas 7 skills → (2) medi no runtime → (3) descobri que
era no-op → (4) revertido → (5) o dono pediu resumo dentro de 247 → (6) redesenho
aplicado e **provado funcionando**.

**Medição (5 probes `cmdc -p`):**

| Probe | `COMMANDCODE_SKILL_CATALOG_CHAR_BUDGET` | O que o modelo recebeu |
|---|---|---|
| A | (unset) | só `<name>` + `<location>` — nenhuma description |
| B | 20000 | idem (orçamento insuficiente → só nomes) |
| C | 60000 | description truncada em ~249, `when_to_use` **fora** do corte |
| D | 60000 (outra skill) | idem — corte confirmado em outra skill |
| **E** | 60000, **após redesenho** | **description + `when_to_use` completos, sem corte** |
| **F** | 30000, **após redesenho** | **description + `when_to_use` completos** (valor recomendado) |
| **G** | 15000, após redesenho | description truncada ~140 chars, **sem** `when_to_use` |

**Default e limiar:** `Gs=8e3` = **8.000** chars (não 8 — scientific notation; uma regex frouxa leu só o `8`), piso `zs=20`, teto `Vs=250`. Catálogo "names only" das 48 skills = 6.910 chars (medido com o template real e os paths reais), logo o default já cai para nomes. Limiar p/ descrição completa nas 48 = `6.910 + 48×(142+249)` ≈ **25.700**; probes F/G confirmam 15.000 (truncado) e 30.000 (completo). Acima de ~26k o budget não muda nada porque o teto é 249/skill. Custo do valor recomendado: ~25.7k chars ≈ **6.4k tokens/turn** no CommandCode.

**Causa do no-op:** `truncateCatalogText(e.whenToUse ? desc + "\n\n" + whenToUse : desc, 250)`
com `(s, n) => s.length > n ? s.slice(0, n-1) + "…" : s`. O `when_to_use` entra depois
da description e o corte fica no início → inalcançável com description ≥ ~247 chars
(as 7 tinham 317–738).

**Redesenho (o que foi feito):**
- `description` = o quê + quando, 102–123 chars (era 317–738);
- `when_to_use` = gatilho em linguagem natural, 98–122 chars (campo exclusivo do
  CommandCode, ignorado pelo OpenCode sem custo);
- lista extendida de gatilhos → corpo, em `## Triggers` (lida só **depois** do load,
  portanto grátis no catálogo);
- contrato travado: `TestWhenToUseFitsCommandCodeCatalog` exige
  `description + 2 + when_to_use ≤ 247` **caracteres** (runes — `String.slice` do JS
  conta caracteres, e `len()` em Go contaria bytes: o teste daria 263 em vez de 243
  em `systematic-debugging`, o que foi exatamente o bug encontrado).

**Ganho de contexto (OpenCode, por turno):** catálogo das 7 cai de 3.648 para 1.607
chars (−479 tokens/turn). No CommandCode, as 7 passam a carregar description **e**
gatilho — o que antes era impossível.

## Verificação (ao final de cada task)

```bash
gofmt -l .
go build ./... && go vet ./... && go test ./...
golangci-lint run --new-from-rev=origin/main
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build ./cmd/envctl
git diff --check
# e, só depois de build novo:
./envctl commandcode && ./envctl doctor     # esperado: 0 WARN / 0 ERROR
```

## Definition of Done

- [x] Frontmatter de agente do CommandCode validado no `doctor`, com o campo culpado nomeado. — `internal/usecase/commandcode_agent_schema.go` + integração em `auditCommandCodeAgents`; prova end-to-end quebrando o `~/.commandcode/agents/code-reviewer.md` deployado (WARN nomeando `tools`) e restaurando (OK).
- [x] `plan` documentado como subagent de planejamento do CommandCode, sem `planner` custom e sem alargar permissão. — justificativa em `subagent-routing` ("por que não existe `planner` no CommandCode") e nos 3 `AGENTS` do CommandCode.
- [x] Duas rotas de worktree documentadas, com a ponte `cmdc -w <abs>`. — tabela em `git-workflow`, linha de worktree nos 3 `AGENTS` do CommandCode, seção do REFERENCE.
- [x] Tabela de controles de visibilidade de skill por plataforma no REFERENCE. — `autoinvoke` ↔ `disable-model-invocation`, `slash` ↔ `user-invocable`, `when_to_use` com o custo de prompt.
- [x] Matriz/doctor docs sincronizados; suíte, lint, cross-build e `git diff --check` com evidência fresca. — matriz #21/#22, `docs/doctor-and-idempotency.md`, CHANGELOG; `gofmt` limpo · build/vet ok · `go test ./...` sem FAIL · `golangci-lint` 0 issues · cross-build ok · `diff --check` limpo · `envctl doctor` 207/207 0 WARN 0 ERROR.
- [x] `when_to_use` decidido com teste empírico (aplicado ou abortado com registro). — **2026-09-26: APLICADO, MEDIDO, REVERTIDO e REAPLICADO COM REDESENHO** (Task 3). As 7 skills de roteamento têm `description + 2 + when_to_use ≤ 247` caracteres, com a lista extendida no corpo; probe `cmdc -p` confirma os dois campos chegando ao modelo sem corte.
- [x] Nenhuma mudança no lado OpenCode exceto o mínimo compartilhado, por seção por runtime. — **exceção registrada**: `configs/REFERENCE.md` (arquivo do OpenCode) recebeu as tabelas comparativas de worktree e visibilidade, porque é a referência operacional do OpenCode e ficaria incompleta sem elas. Nenhuma mudança de config, agente, permissão ou runtime do OpenCode. O `doctor` detectou o drift sozinho e `envctl opencode` sincronizou. **Ponto aberto para o dono**: manter assim ou mover a comparação para outro arquivo.

## Estado de execução (2026-09-26)

- Tasks 1 e 2: implementadas e verificadas em `feat/commandcode-parity` (filha de `feat/subagent-orchestration-worktrees`).
- Task 3: executada — aplicada, medida no runtime e revertida por ser no-op (ver acima).
- **Nada commitado nesta branch ainda** — 12 arquivos modificados + 2 novos, aguardando confirmação do dono.
- Divergência deliberada de escopo registrada: o `doctor` foi reprovisionado nas duas camadas (`envctl commandcode` e depois `envctl opencode`) porque o REFERENCE é arquivo do OpenCode.
