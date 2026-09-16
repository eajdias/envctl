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

Write a handoff document summarising the current conversation so a fresh agent can continue the work. Save to the temporary directory of the user's OS - not the current workspace.

Include a "suggested skills" section in the document, naming which skills the next agent should invoke (`/<skill>` ou a tool `activate_skill` no CommandCode; tool `skill` no OpenCode).

Do not duplicate content already captured in other artifacts (specs, plans, ADRs, issues, commits, diffs). Reference them by path or URL instead.

Redact any sensitive information, such as API keys, passwords, or personally identifiable information.

If the user passed arguments, treat them as a description of what the next session will focus on and tailor the doc accordingly.
