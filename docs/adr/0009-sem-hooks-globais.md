# ADR 0009: Sem hooks globais — o gate é ferramenta, não interceptação

## Status
Aceito. **Alterna parcialmente o ADR 0006** (as âncoras globais de hook são
removidas; o verificador único permanece).

## Contexto
O ADR 0006 ancorava o gate em dois pontos automáticos: o hook `Stop` global do
CommandCode e um `pre-push` global do git via `core.hooksPath`. Na prática isso
interferia em projetos que não pediram nada:

- `core.hooksPath` global faz o git ignorar o `.git/hooks` de **todo**
  repositório da máquina — o `pre-commit`/`commit-msg` dos outros projetos eram
  **silenciados** (o delegator/shim do ADR 0006 era paliativo dessa causa-raiz).
- O `pre-push` global rodava lint, tsc e a suíte de testes — e podia **bloquear
  o push** — de qualquer repositório.
- O hook `Stop` global gateava o fim de turno do agente em qualquer projeto.

Diretriz do usuário (2026-10-10): "O envctl nunca deve interferir em pre-commit
de outros repos. Remova qualquer hook/pre-commit/gate/etc global que possa
interferir em outro repo e lance a v2."

## Decisão
- **O envctl não instala hooks nem gates globais — nunca.** Nada de
  `core.hooksPath`, hooks de git (pre-push, pre-commit, commit-msg, …) nem hook
  de turno do agente em configuração global.
- **`envctl-verify` permanece** como ferramenta de invocação explícita
  (`--hook`, `--git-push`, `--dry-run`), deployada em `~/.local/bin`.
- **Gate automático é opt-in do próprio projeto**: um hook **local** daquele
  repositório (`.git/hooks/pre-push`, husky, lefthook) pode chamar
  `envctl-verify --git-push`. O git executa os hooks do projeto sem nenhuma
  camada do envctl por cima.
- **O doctor audita a ausência** do wiring legado: resquícios de
  `~/.config/git/hooks` geram `WARN` com instrução de remoção (inclusive
  `git config --global --unset core.hooksPath`).
- **Upgrade de máquinas provisionadas**: entrada de `cleanup` no manifesto remove
  `~/.config/git/hooks`; o `core.hooksPath` global é desfeito manualmente (o
  doctor orienta e o CHANGELOG documenta).

## Consequências
- Nenhum repositório da máquina tem hooks ou turnos de agente interceptados pelo
  envctl — a garantia é estrutural, não de filtragem.
- Quem dependia do gate automático perde-o até optar por um hook local
  (breaking intencional da v2).
- O contrato do verificador com os agentes não muda: os agentes o invocam
  explicitamente (verifier subagent, `--git-push`).
