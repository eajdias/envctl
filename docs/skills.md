# Catálogo de Skills

Doze skills. O critério de entrada é deliberadamente estreito:

> Uma skill só merece slot de catálogo se contiver **algo que o modelo não faria
> sozinho** — preferência do usuário, procedimento não-óbvio, ou armadilha já paga.
> Conhecimento de ferramenta, receita de projeto e comportamento obrigatório não entram
> como skill: viram `references/` (carregado sob demanda) ou linha do `AGENTS.md`.

Isso é medido, não estético: o catálogo é injetado em **todo turno**. Medido com
`cmdc -p`, 48 skills custavam ~6,4k tokens/turn e, com o budget default do CommandCode,
o runtime degrada o catálogo para **nome + location** — nesse modo a auto-ativação
**não acontece** (0 chamadas de `activate_skill` observadas). Com 12 skills, o catálogo
cabe no default e a description volta a chegar ao modelo.

## As 12

| # | Skill | Papel | Por que merece slot |
|---|---|---|---|
| 1 | `agent-memory` | lições e padrões por tier, LOAD→ACT→SAVE→REFLECT | sobrevive a `/clear` e à troca de sessão |
| 2 | `clarify-before-acting` | mede ambiguidade 0–100 e pergunta o mínimo | preferência tua: perguntar antes de executar errado |
| 3 | `writing-plans` | spec/plano com riscos, rollback e verificação | convenção do repo + método |
| 4 | `systematic-debugging` | 4 fases até a causa raiz, depois as variantes | lei de depuração do projeto |
| 5 | `test-driven-development` | red-green-refactor com comandos reais | disciplina que o modelo não aplica sozinho |
| 6 | `git-workflow` | branches, commits, PR e worktree | convenções tuas + a lição do trabalho sequestrado |
| 7 | `subagent-routing` | quando delegar (inline é o padrão) | evita gastar token delegando por hábito |
| 8 | `subagent-supervision` | status/interrupção/retry por runtime | vocabulary real, comprado caro |
| 9 | `task-hang-watchdog` | timeout, background, log, interrupção | comando sem timeout trava a sessão |
| 10 | `technical-research` | fonte, data, fato vs inferência | método que evita opinião com ar de citação |
| 11 | `code-playbooks` | **catálogo-ponte** → `references/*.md` | 14 temas, 1 entrada |
| 12 | `envctl` | auditar/provisionar a máquina | o agente pode precisar em qualquer repo |

## O gate `code-playbooks`

Uma entrada para todo o conhecimento específico, carregado em dois hops:

```
code-playbooks/
  SKILL.md
  references/{go,python,node,nextjs,sql,testing,docker,infra,
              frontend,web-automation,mcp-api,backend-patterns,
              skill-authoring,docs-sync}.md
```

O `SKILL.md` é só a tabela. A regra dura do corpo: **não responda sobre um tema sem ler
o arquivo correspondente**.

## Regras que saíram do catálogo

| Era skill | Onde está agora |
|---|---|
| `verification-before-completion`, `stop-slop`, `handoff`, `receiving-code-review`, `context7-auto` | `AGENTS.md` — comportamento obrigatório, não descoberta |
| `variant-analysis` | fase 5 do `systematic-debugging` |
| `grilling` + `ask-questions-if-underspecified` | `clarify-before-acting` |
| `universal-test-runner` | `references/testing.md` |
| `docs-sync` | regra no `AGENTS.md` + `references/docs-sync.md` |
| `go-development`, `python-development`, `nextjs-standalone-deploy`, `database-ops`, `docker`, `frontend-markup`, `ssh-vps`, `tailscale`, `syncthing-ops`, `mcp-tool-design`, `api-contract-design`, `jwt-hs256-node`, `simple-feature-flag`, `lsp-smoke-test`, `headless-gui-probe`, `aur-headless-install`, `phone-e164-normalization`, `bulk-postgres-import`, `skill-miner`, `skill-generalizer`, `skill-personalizer` | `code-playbooks/references/` ou AGENTS do projeto que usa |
| `cachyos-gaming-setup` | `docs/guides/cachyos-gaming.md` (repositório) |
| `windows-debloat` | `docs/guides/windows-debloat-tier3.md` (repositório) |
| `vps-provisioning`, `linux-performance-tuning` | docs do repo do envctl (o global vira uma linha) |
| `windows-admin` | só Windows; conhecimento do produto |

## Contrato do catálogo

- `description` é o **único** sinal de ativação que o modelo tem antes do load: escreva
  "use quando" na primeira frase.
- No CommandCode, `description + when_to_use` são cortados em **249 caracteres**
  (semântica de `String.slice`: conta caracteres, não bytes). O repo garante
  `description + 2 + when_to_use ≤ 247` nas skills que declaram o campo, e
  `TestWhenToUseFitsCommandCodeCatalog` trava isso.
- Fechado em 12 pela aritmética: `25 + N×391 ≤ 8000` (default) ⇒ `N ≤ 20`.
