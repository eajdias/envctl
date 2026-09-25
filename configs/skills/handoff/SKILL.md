---
name: handoff
description: >-
  Compacta a conversa atual em um documento de handoff autocontido, para a próxima sessão ou outro agente continuar o trabalho sem reler o histórico. Use ao encerrar uma sessão longa, quando o contexto estiver cheio, ao transferir trabalho entre agentes/modelos, ou quando o usuário disser "vou continuar depois", "outra sessão", "resumir o contexto", "passar o bastão". Triggers: handoff, passar o bastão, resumir contexto, continuar depois, nova sessão, transferir trabalho, compactar conversa, contexto cheio.
argument-hint: "What will the next session be used for?"
license: MIT
metadata:
  author: Matt Pocock
  source: https://github.com/mattpocock/skills
  adapted: envctl — descricao/triggers em PT-BR; invocacao de skill parametrizada por agente
---

# Handoff

Crie um handoff curto e autocontido para a próxima sessão ou para outro agente continuar sem reler a conversa. Ele é um estado operacional, não uma transcrição: preserve o que impede erro, desperdício ou repetição e remova conversa fiada e tentativas fracassadas sem consequência.

Salve o documento no diretório temporário do sistema operacional, usando `ENVCTL_TEMP` quando existir. Nunca salve o handoff no workspace do projeto, em um diretório de artefatos do projeto ou em um caminho que possa ser confundido com código-fonte.

## Schema mínimo

Use estes títulos, mesmo quando o valor for `Nenhum` ou uma frase curta. São o mínimo para retomar com segurança; não transforme o handoff em relatório histórico.

```markdown
# Handoff: [nome curto]

## Objetivo
[O que a próxima sessão deve conseguir fazer e por quê.]

## Concluído
- [Resultado observável, não apenas a intenção de fazer.] Referência: [artefato existente, se houver].

## Decisões e constraints
- **Decisão:** [escolha e motivo].
- **Constraint:** [limite, preferência ou requisito que não pode ser ignorado].

## Estado verificável
- Comando: `[comando executado]` → [resultado observado].
- [Estado, caminho, versão ou configuração importante, com fonte/evidência.]

## Pendências
1. [Trabalho que ainda falta fazer, com bloqueio ou dependência se houver.]

## Próximo passo
[Uma ação executável para começar; não repetir o objetivo inteiro.]

## Skills sugeridas
- `skill-name` — [por que é útil]. Carregar via `/<skill>` ou `activate_skill` no CommandCode; via tool `skill` no OpenCode.
```

Se o usuário passou argumentos, trate-os como o foco da próxima sessão e deixe isso explícito no objetivo e no próximo passo. Se não passou, derive o foco do objetivo atual sem inventar requisitos.

## O que preservar

- **Objetivo e resultado:** formule o que deve acontecer, não a longa história da conversa.
- **Decisões e restrições:** registre a escolha e o motivo para o próximo agente não voltar atrás.
- **Estado verificável:** só afirme o que foi observado nesta sessão. Inclua comandos, resultados, caminhos e versões úteis; marque hipóteses como hipóteses.
- **Pendências:** ordene por dependência e destaque bloqueios, decisões que precisam do usuário e artefatos que ainda não existem.
- **Próximo passo:** escolha a menor ação que destrava o restante; não transforme a pendência inteira em uma instrução vaga.
- **Skills sugeridas:** indique apenas skills relacionadas ao próximo trabalho e por quê.

## Não duplicar artefatos

Se já existe uma spec, plano, ADR, issue, decisão, commit, diff, log ou relatório de testes, cite o caminho, URL ou identificador e explique em uma linha o que ele resolve. Não copie seu conteúdo para o handoff. Se o artefato não for suficiente, registre a lacuna ou a evidência que falta, em vez de reconstruir tudo.

## Redaction e segurança

Remova valores de chaves, senhas, tokens, cookies, cabeçalhos de autorização, dados pessoais, hosts privados, identificadores de usuário e outros segredos. Redija também comandos e saídas que os exponham; use apenas o nome da variável ou uma referência segura. Preserve regras, decisões e nomes de arquivos que não revelam dado sensível. Nunca invente uma versão redigida como se fosse o valor original.

Antes de finalizar, leia o handoff como se fosse a única entrada disponível: a próxima sessão deve entender o objetivo, o estado real, o que falta e qual ação fazer agora. Informe o caminho do arquivo criado.
