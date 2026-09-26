# Índice de Skills (CommandCode)

> Catálogo de **12 skills**. Regra de entrada: só entra o que o modelo
> não faria sozinho (preferência sua, procedimento não-óbvio, armadilha paga).
> Conhecimento de ferramenta e receita de projeto ficam em `code-playbooks/references/`.
> Índice é para desempate — o corpo só é lido quando a skill carrega.

## Quando usar o quê

| Situação | Skill |
|---|---|
| Precisar que a tarefa está ambígua (perguntar antes de agir) | `clarify-before-acting` |
| Planejar antes de tocar produção (spec em `spec-agent/`) | `writing-plans` |
| Depurar achando a causa raiz e depois as variantes | `systematic-debugging` |
| Mudar comportamento com teste primeiro | `test-driven-development` |
| Git, branches, commits, PR, worktree | `git-workflow` |
| Decidir se delega a um subagente (inline é o padrão) | `subagent-routing` |
| Vigiar/interromper subagente (vocabulary por runtime) | `subagent-supervision` |
| Comando travado, tarefa presa, shell em background | `task-hang-watchdog` |
| Pesquisar com fonte atual, separando fato de inferência | `technical-research` |
| Lembrar lições/padrões entre sessões | `agent-memory` |
| Convenção de stack, ferramenta ou domínio (Go, Python, SQL, infra, frontend, MCP…) | `code-playbooks` |
| Auditar ou provisionar o ambiente da máquina | `envctl` |

## Regras que valem sempre (vêm do `AGENTS.md`, não de skill)

- Evidência antes de afirmação: rode o comando e mostre a saída.
- Zero tolerância a WARNING/ERROR.
- Sem nunca deduzir: re-teste antes de afirmar.
- Sem clichê; handoff quando o contexto encher; review com rigor.
- **Docs em dia**: manifesto e código ganham da doc (roteiro em `code-playbooks/references/docs-sync.md`).
