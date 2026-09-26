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
convenção é o path absoluto em `-w`.

### Task 3 (bloqueada) — `when_to_use` nas skills de roteamento

Pré-requisito: teste empírico de 1 skill (adicionar `when_to_use` localmente,
boot do OpenCode, procurar warning de loader, rodar `envctl doctor`). Se o
OpenCode reclamar → abortar e registrar. Se tolerar → aplicar só nas skills de
roteamento e medir o custo de prompt.

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

- [ ] Frontmatter de agente do CommandCode validado no `doctor`, com o campo culpado nomeado.
- [ ] `plan` documentado como subagent de planejamento do CommandCode, sem `planner` custom e sem alargar permissão.
- [ ] Duas rotas de worktree documentadas, com a ponte `cmdc -w <abs>`.
- [ ] Tabela de controles de visibilidade de skill por plataforma no REFERENCE.
- [ ] Matriz/doctor docs sincronizados; suíte, lint, cross-build e `git diff --check` com evidência fresca.
- [ ] `when_to_use` decidido com teste empírico (aplicado ou abortado com registro).
- [ ] Nenhuma mudança no lado OpenCode exceto o mínimo compartilhado, por seção por runtime.
