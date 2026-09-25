---
name: subagent-supervision
description: >-
  Supervisionar subagentes que podem demorar, repetir tool calls, ficar sem progresso ou produzir conclusões sem evidência. Use quando houver subagente em background, task longa, loop, timeout, resultado suspeito ou necessidade de interromper e retentar com escopo refinado. Triggers: subagente não retorna, loop de tool calls, alucinação, sem progresso, demorou demais, interromper subagente, cancelar sessão, matar processo, retry, escalar, supervisor, background agent.
license: MIT
---

# Subagent Supervision (o coordenador vigia os subagentes)

Supervisão é uma responsabilidade do coordenador, mas execução, depuração e
verificação continuam inline por padrão. Um subagente é justificado apenas quando o
contexto que ele produz é volumoso e o retorno é compactado.

## Regra de segurança

- Só supervisione sessões que o coordenador atual criou ou recebeu explicitamente.
- Uma `sessionID` não é um PID e não prova que o processo pode ser morto.
- `interrupt` encerra a sessão LLM; não apaga worktree, não faz `reset`, `clean`,
  commit ou remoção de worktree.
- Para encerrar um processo externo, é necessário ter o PID rastreado e propriedade
  do processo. Nunca inferir PID pelo nome do agente.

## Ciclo de vida no OpenCode V2

### Subagent em foreground

O coordenador mantém a chamada ativa. Uma interrupção da sessão pai cancela a
execução filha; aguarde a confirmação antes de iniciar um retry.

### Subagent em background

1. Guarde o `sessionID`/identificador devolvido no dispatch.
2. Observe a notificação de conclusão ou consulte as sessões ativas:

   ```bash
   opencode api get /api/session/active
   ```

3. Se precisar interromper a sessão filha, use somente a API documentada do V2:

   ```bash
   opencode api post /api/session/<sessionID>/interrupt
   ```

4. Se a interrupção não encerrar um processo que o subagente iniciou, trate isso como
   dependência externa: identifique o PID, confirme o command line e finalize o
   processo separado. `SIGTERM` primeiro; `SIGKILL` somente após verificar que a
   interrupção suave não funcionou.

O comando acima é uma interrupção de sessão, não uma operação destrutiva de
filesystem. Shell, Git e worktrees continuam sob as regras de `git-workflow`.

## Sinais e decisão

| Sinal | Ação |
|---|---|
| Mesma tool call repetida sem avanço | `interrupt`; retry uma vez com escopo menor |
| Sem tool call ou sem saída por prazo inesperado | `interrupt`; preserve o worktree e escale se persistir |
| Resultado contraditório ou sem evidência | pedir nova evidência ou revisão independente; não matar por “alucinação” presumida |
| Processo externo ainda vivo após `interrupt` | agir somente com PID rastreado e autorização apropriada |
| Duas tentativas refinadas falharam | escalar ao usuário; nunca entrar em loop de retry |

## CommandCode e outros agentes

A vocabulary de runtime é diferente. Use apenas o controle de sessão/background
exposto pelo agente ativo. Se ele não expõe status, interrupção ou PID, reporte
o bloqueio em vez de inventar uma tool ou chamar um comando de outro runtime.

## Verificação

- [ ] A sessão interrompida realmente parou ou foi escalada com evidência.
- [ ] O worktree e os arquivos permaneceram preservados.
- [ ] O retry, quando houve, refineu o escopo e teve no máximo uma repetição.
- [ ] O coordenador recebeu um resumo compacto, não o transcript bruto.
