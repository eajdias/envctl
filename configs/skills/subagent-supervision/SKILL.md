---
name: subagent-supervision
description: >-
  Supervisionar subagentes que podem demorar, repetir tool calls, ficar sem progresso ou produzir conclusões sem evidência. Use quando houver subagente em background, task longa, loop, timeout, resultado suspeito ou necessidade de interromper e retentar com escopo refinado. A forma de interromper muda por runtime: CommandCode usa agent_output/agent_id e kill_shell; OpenCode V2 usa sessionID + opencode api. Triggers: subagente não retorna, loop de tool calls, alucinação, sem progresso, demorou demais, interromper subagente, cancelar sessão, matar processo, retry, escalar, supervisor, background agent.
license: MIT
---

# Subagent Supervision (o coordenador vigia os subagentes)

Supervisão é responsabilidade do coordenador; execução, depuração e verificação
continuam inline. Um subagente só se justifica quando o contexto que ele produz é
volumoso e o retorno é compactado.

## Antes de agir: identifique o runtime

A capacidade de interromper **muda por agente**. Nunca copie comando de um runtime
para o outro — a mesma skill é servida para os dois, e um tool inexistente no runtime
ativo é erro, não fallback.

| | OpenCode V2 | CommandCode |
|---|---|---|
| id do filho | `sessionID` | `agent_id` (`bg-1-…`) |
| ver status | `opencode api get /api/session/active` | `agent_output({})` lista todos |
| coletar resultado | notificação de conclusão | `agent_output({agent_id})` (default `wait`) |
| **interromper** | `opencode api post /api/session/<sessionID>/interrupt` | `agent_output({agent_id, action: "kill"})` |
| estados possíveis | sessão encerrada | `running`/`completed`/`failed`/`killed` |
| processo externo | `sessionID` **não** é PID: só com PID rastreado | `kill_shell({pid})` faz graceful→force |

Cada runtime tem o seu próprio toolset de background: o do CommandCode (`agent`
delegando com `run_in_background`, coleta e `kill` por id, `shell_command` em
background com leitura e parada por task) não existe no OpenCode V2, e
`sessionID`/`opencode api` não existem no CommandCode. Use a coluna do runtime ativo.

## CommandCode

- Dispatch com `run_in_background: true` (ou `background: true` no frontmatter do
  agente) devolve `agent_id` na hora, e o filho **sobrevive** ao cancelamento do
  turno pai — cada um tem abort controller próprio. Só um `kill` explícito encerra.
- `agent_output({agent_id})` → `wait` (bloqueia; abortar a *espera* **não** mata o
  agente e reporta "still running in the background"), `action: "status"` (peek sem
  block, com tempo decorrido), `action: "kill"` (interrompe).
- Shell segue o mesmo modelo: `shell_command` + `run_in_background` → task id, PID,
  cwd e path de log; `shell_output` (com `wait: "output"|"exit"`, `from_offset`, teto
  inline de 30k e resto no log), `shell_tasks`, `task_output` (bloqueante até 600s) e
  `task_stop` (por id) / `kill_shell` (por `taskId`, `pid` ou `port`).
- Long-lived observe com `monitor_command` + `monitor_events` (delta read): o runtime
  acorda o agente sozinho. **Não** faça poll.
- O painel Background (`Ctrl+B` no TUI) mostra e para o mesmo registry.
- Esperar sem segurar shell: tool `sleep` (acorda com input do usuário em ~1s);
  cadência com `/loop` e `cron_*`.

## OpenCode V2

- Guarde o `sessionID` devolvido no dispatch. Em foreground, interromper a sessão pai
  cancela a filha — aguarde a confirmação antes de qualquer retry.
- Status: `opencode api get /api/session/active`.
- Interrupção: `opencode api post /api/session/<sessionID>/interrupt`.
- Probe seguro de rota: um id inexistente devolve `SessionNotFoundError` (404) —
  prova o endpoint sem tocar em sessão viva.
- `sessionID` **não** é PID: processo externo só com PID rastreado e command line
  conferida (SIGTERM, espera, SIGKILL). Nunca inferir PID pelo nome do agente.

## Regras (valem para os dois)

- Só supervisione sessões/agentes que o coordenador atual criou ou recebeu
  explicitamente.
- `interrupt`/`kill` encerra a execução — não apaga worktree, não faz `reset`, `clean`,
  commit nem `worktree remove --force`.
- Resultado suspeito não é motivo de kill: peça evidência nova (`file:line`, saída de
  comando) ou revisão independente.
- Retry: **uma** tentativa, com escopo menor e prompt refinado. Duas falhas: escalar
  para o usuário. Nunca entrar em loop de retry.
- Devolva ao coordenador um resumo compacto: decisão, evidência, arquivos afetados,
  risco e próximo passo — nunca o transcript bruto.

## Verificação

- [ ] Runtime identificado e o comando veio da coluna certa.
- [ ] O filho realmente parou (ou escalado com evidência).
- [ ] Worktree, arquivos e logs preservados.
- [ ] Retry no máximo 1×, com escopo refinado.
- [ ] Coordenador recebeu resumo compacto, não o log bruto.
