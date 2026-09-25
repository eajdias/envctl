---
name: task-hang-watchdog
description: >-
  Prevenir e recuperar terminais travados e tarefas autônomas presas. Use ao rodar comandos de shell longos ou não interativos (build, teste, migração, servidor, watch), despachar agentes em background, ou quando uma tarefa simplesmente não prossegue. O mecanismo muda por runtime: CommandCode usa run_in_background/shell_output/kill_shell/monitor_command; OpenCode V2 usa timeout da tool e interrupção de sessão. Triggers: terminal travou, comando não responde, tarefa presa, processo hung, demorou demais, background, travamento, não interativo, input prompt, kill processo.
license: MIT
---

# Task & Terminal Hang Watchdog

Evite que um comando ou subagente trave a sessão — e, se travar, recupere o controle
sem perder trabalho.

## Runtime: o vocabulário muda

| | OpenCode V2 | CommandCode |
|---|---|---|
| comando longo | tool `shell` com `timeout`; output na própria chamada | `shell_command` + `run_in_background: true` → task id, PID, cwd e log em disco |
| ler saída | (mesma chamada) | `shell_output` (30k inline + log em disco) · `task_output` (bloqueante até 600s) |
| listar / parar | `ps` + `kill` no shell | `shell_tasks` · `task_stop` (taskId) · `kill_shell` (taskId/pid/port) |
| observar longo | log no scratch + `tail` | `monitor_command` + `monitor_events` (wake automático, sem poll) |
| esperar sem segurar shell | tool `sleep` | tool `sleep` (acorda com input em ~1s) |
| recuperar sessão travada | `subagent-supervision` (OpenCode) | `agent_output({agent_id, action: "kill"})` |

Cada runtime traz o seu próprio toolset de background; a coluna vale só para o agente
em uso.

## Prevenção (vale para os dois)

1. **Classifique o comando.** Read-only (`git status`, `ls`, `rg`, `cat`) não espera
   input. Comandos que removem arquivos, instalam pacotes ou abrem prompt exigem flags
   não interativas (`-y`, `--yes`, `--force`) e aprovação quando o runtime pedir.
2. **Defina timeout real.** No CommandCode o foreground tem 30s de default, clamp de
   10 min e `timeout: 0` como opt-out; o que passa disso é background. No OpenCode o
   timeout é parâmetro da tool shell.
3. **Saída longa vira log** no scratch padrão (`ENVCTL_TEMP`), em subdir por tarefa —
   nunca no repositório.
4. **Prompt interativo não se responde por adivinhação.** SSH, sudo com senha e TTY
   usam a skill do runtime local (`ssh-vps`, `aur-headless-install`).

## CommandCode

- `shell_command` + `run_in_background: true` → task id, PID, cwd e path de log
  (stdout/stderr espelhado em disco, então a saída completa sempre existe).
- `shell_output` → o acumulado até agora + status; aceita `wait: "output"|"exit"`,
  `timeout_ms`, `from_offset` e `max_chars` (teto inline de 30k, resto no log).
  `task_output` é a leitura mais rica: bloqueia até `timeoutMs` (default 30s, teto
  600s) e devolve a cauda + exit code + log path.
- `shell_tasks` lista tudo (id, status, PID, comando, cwd, log); `includeStopped:
  false` mostra só o que está rodando.
- Parar: `task_stop({taskId})` (SIGTERM no process group + estado `stopped`) ou
  `kill_shell({taskId|pid|port})`. PID não rastreado recebe graceful→force
  (SIGTERM → 5s → SIGKILL) e nunca reporta "kill bem-sucedido" falso.
- `monitor_command` + `monitor_events` para observar log contínuo: o runtime acorda o
  agente sozinho (`checkAfterMs`/exit) — não faça poll.
- Sinal de morte nunca é exit 0: o runtime reporta `128+N` (SIGKILL → 137, SIGTERM →
  143) com o nome do signal.
- Teto de foreground: 30s default, clamp de 10 min, `timeout: 0` como opt-out.

## OpenCode V2

- Comando longo: tool `shell` com `timeout` explícito; não existe task em background
  gerenciada por tool, então o que precisar sobreviver ao turno vai para log no
  scratch e é lido depois.
- Recuperação de sessão travada: skill `subagent-supervision` (interrupt de
  `sessionID`); processo externo só com PID rastreado, verificado por
  `ps`/command line, com SIGTERM antes de SIGKILL.

## Quando algo trava

1. **Confirme o estado real** antes de agir: `git status`, log, processo. Não presuma.
2. **Interrompa o que você possui**: sessão (skill `subagent-supervision`) ou task
   rastreada (`task_stop`/`kill_shell` no CommandCode; PID verificado no OpenCode).
3. **Suave antes de forçado.** No CommandCode, `kill_shell` com PID não rastreado já
   faz SIGTERM → 5s de espera → SIGKILL e reporta erro real se o processo sobreviver
   (nunca "kill bem-sucedido" falso). No OpenCode, faça você: PID rastreado,
   SIGTERM, espera, SIGKILL.
4. **Preserve log, worktree e arquivos.** `reset --hard`, `clean`,
   `worktree remove --force` e `kill -9` nunca são a primeira resposta.

## Não fazer

- Comando potencialmente longo sem timeout; servidor/watch em foreground na sessão
  principal sem necessidade.
- Confiar que o agente responderá um prompt interativo.
- Matar processo por padrão de nome em vez de task id ou PID verificado.
- Fazer poll de `monitor_events`/`shell_output` esperando o runtime acordar o agente.
- Deixar side effect (container, porta, serviço) sem nome próprio quando duas tasks
  rodam em paralelo — worktree isola arquivos, não recursos.

## Verificação

- [ ] Comando com timeout e propriedade do processo conhecida.
- [ ] Saída disponível no log ou no resultado da task.
- [ ] Interrupção suave tentada antes de encerramento forçado.
- [ ] Nenhum arquivo, worktree ou estado Git apagado como parte do recovery.
