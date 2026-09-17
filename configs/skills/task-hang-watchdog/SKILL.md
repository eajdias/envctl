---
name: task-hang-watchdog
description: >-
  Prevenir e recuperar terminais travados e tarefas autônomas presas. Use ao rodar comandos de shell longos ou não-interativos (build, teste, migração, servidor, watch), despachar agentes em background, ou quando uma tarefa simplesmente não prossegue. Estratégias: rodar em background, classificar o que é read-only (não espera input), tratar prompts interativos, detectar e matar processos travados, observar saídas longas via monitor. Triggers: terminal travou, comando não responde, tarefa presa, processo hung, demorou demais, background, travamento, não interativo, input prompt, kill processo.
license: MIT
---

# Task & Terminal Hang Watchdog

Evite que um comando ou subagente trave a sessão — e, se travar, recupere o controle.

## Prevenção (sempre, antes de rodar)

1. **Classifique o comando.** O shell do CommandCode/ReadCode é read-only-safe: `git status`, `ls`, `rg`, `cat` rodam sem prompt. Já `rm`, `apt-get install`, `npm init` pedem confirmação e **travam** o agente esperando input.
2. **Use `run_in_background: true`** em `shell_command` para tarefas longas (build, `npm run dev`, migração, servidor). Assim o agente não fica bloqueado e você lê a saída depois com `shell_output` ou `monitor_command`.
3. **Prompts interativos** nunca são respondidos pelo agente. Substitua por flags não-interativas:
   - `apt-get install -y ...`, `npm init -y`, `rm -f`, `--force`, `-y`/`--yes` sempre que existir.
   - Em scripts que pedem TTY (ssh, sudo com senha), use o `ssh-manager`/`ssh-vps` em vez de shell direto.
4. **Defina `timeout`** (ms) em `shell_command` — padrão 30s; suba para o que a tarefa exige. Se estourar, o shell é killado e você recebe o código de saída.
5. **Para saídas longas/continuas** (servidor, watch, log `-f`), use `monitor_command` — ele grava em log e te acorda quando algo relevante acontece, sem travar a sessão.

## Quando algo trava

Sinais: sem saída por muito tempo, prompt esperando `[y/N]`, `Enter`, senha; a tarefa não avança; CPU/disco parados.

**Diagnóstico rápido:**
- `shell_tasks` — lista tarefas em background e seus PIDs/status.
- `shell_output(id)` — vê o que o processo travado já emitiu.
- `exec.LookPath` de um processo "zombie" ou checagem de porta/port com `kill_shell(port:)`.

**Ação:**
- `kill_shell(taskId=...)` — mata a tarefa em background pelo id.
- `shell_output(id, wait: "exit")` — espera o processo atual terminar (último recurso).
- Se o **próprio shell do agente** travar (comando bloqueante em foreground), peça ao usuário para matar o processo pelo terminal/gerenciador de tarefas ou reiniciar a sessão (`/reload`).

## Tarefas autônomas (subagentes) que travam

Um subagente pode alucinar, entrar em loop de tool calls ou nunca retornar. Ver a skill `subagent-supervision` para monitorar e matar subagentes presos.

## Não fazer

- Nunca deixar `run_in_background: false` em algo que pode demorar mais que o timeout sem motivo.
- Nunca confiar que "vai pedir input e eu respondo" — o agente não responde prompts interativos.
- Nunca rodar servidores/watch em foreground na sessão principal.

## Verificação

- `shell_tasks` mostra tudo rodando.
- Saída do `monitor_command` no log indica se o processo terminou ou está ativo.
- Tarefa completa? Saída confirma (exit 0 / resultado esperado).
