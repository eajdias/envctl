---
name: parallel-agent-orchestration
description: Orquestrar subagentes paralelos trabalhando no mesmo repositório git sem conflitos. Use quando dividir tarefas grandes entre agentes simultâneos (refatoração, features independentes, migrações) no mesmo repo. Triggers: subagentes, paralelo, parallel agents, orquestrador, dividir tarefa, agentes simultâneos, conflito de edição.
license: MIT
compatibility: opencode
---

# Orquestração de Subagentes Paralelos no Mesmo Repo

## Quando usar

Tarefa grande divisível em unidades independentes executadas por agentes em paralelo no MESMO repositório.

## Passos

1. **BASE COMUM PRIMEIRO**: o orquestrador aplica schema/shared/bootstrap em um commit próprio, antes de despachar os subagentes.
2. **Fronteiras DISJUNTAS explícitas**: cada subagente recebe a lista exata de arquivos que pode tocar + proibições claras ("não tocar X — outro agente é dono").
3. **Despacho paralelo** com cada agente em seu contexto (sem estado compartilhado).
4. **Integração pelo orquestrador no final**: registros de módulos, `app.module`, exports, etc.
5. **Suíte completa** após integração (build + testes) — nunca confiar só nos testes individuais.

## Verificação

- `git status` limpo de conflitos de edição entre fronteiras.
- Build + teste da suíte completa passando após a integração.
- Nenhum arquivo fora da fronteira de cada agente modificado (diff por agente).