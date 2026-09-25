---
name: task-hang-watchdog
description: >-
  Prevenir e recuperar terminais travados e tarefas autônomas presas. Use ao rodar comandos de shell longos ou não interativos (build, teste, migração, servidor, watch), despachar agentes em background, ou quando uma tarefa simplesmente não prossegue. Prefira comandos com timeout e saída persistente; nunca deixe um prompt interativo bloquear a sessão. Triggers: terminal travou, comando não responde, tarefa presa, processo hung, demorou demais, background, travamento, não interativo, input prompt, kill processo.
license: MIT
---

# Task & Terminal Hang Watchdog

Evite que um comando ou subagente trave a sessão — e, se travar, recupere o
controle sem perder trabalho.

## Prevenção

1. **Classifique o comando.** Comandos read-only como `git status`, `ls`, `rg` e
   `cat` não esperam input. Comandos que removem arquivos, instalam pacotes ou
   abrem prompts exigem flags não interativas (`-y`, `--yes`, `--force`) e
   aprovação quando a tool do runtime exigir.
2. **Defina um timeout real** para builds, testes, migrações, servidores e
   watchers. O timeout deve ser proporcional à tarefa; não use um valor que corta
   uma operação legítima.
3. **Persistir saída longa** em log no scratch padronizado quando o comando
   precisar continuar depois do turno. Use `ENVCTL_TEMP` e um subdir por tarefa.
4. **Não responda prompts interativos por adivinhação.** Para SSH, sudo com senha
   ou TTY, use a skill e o fluxo do runtime local.

## Runtime

As ferramentas de background e shell são específicas do agente:

- **OpenCode V2:** use o timeout da tool shell; para uma sessão filha, siga
  `subagent-supervision` e use a API de interrupção documentada.
- **CommandCode:** use apenas o controle de background/timeout exposto pela
  sessão atual; não copie nomes de tools de outro runtime.

## Quando algo trava

Sinais: sem saída por muito tempo, prompt esperando confirmação, senha ou
`Enter`; a tarefa não avança; CPU/disco parados.

1. Verifique o estado real com `git status`, logs e o processo do shell.
2. Encerre apenas o processo que você sabe que possui; comece por uma interrupção
   suave e só force o encerramento depois de confirmar que a interrupção não
   funcionou.
3. Se o processo é um subagente, prefira interromper a sessão pelo identificador
   retornado no dispatch; `sessionID` não é PID.
4. Preserve logs, worktree e arquivos para diagnóstico. Nunca use `reset --hard`,
   `clean`, remoção de worktree ou `kill -9` como primeira resposta.

## Não fazer

- Não deixar um comando potencialmente longo sem timeout.
- Não confiar que o agente responderá um prompt interativo.
- Não rodar servidor/watch em foreground na sessão principal sem necessidade.
- Não matar processos por padrão de nome; use PID e comando verificados.

## Verificação

- [ ] O comando tem timeout e propriedade de processo conhecida.
- [ ] A saída está disponível no log ou no resultado da task.
- [ ] Uma interrupção foi tentada antes de encerramento forçado.
- [ ] Nenhum arquivo, worktree ou estado Git foi apagado como parte do recovery.
