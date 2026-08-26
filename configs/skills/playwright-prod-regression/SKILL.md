---
name: playwright-prod-regression
description: Rodar regressão Playwright contra ambiente de produção com segurança. Use quando validar que a aplicação em produção continua funcionando após deploy, sem mutar dados reais. Triggers: regressão, produção, playwright prod, smoke prod, validar produção, não mutar dados, console errors.
license: MIT
compatibility: opencode
---

# Regressão Playwright Contra Produção

## Quando usar

Validar produção após deploy/migração sem risco de alterar dados reais.

## Regras

1. **NUNCA mutar dados reais** — ações de escrita (aprovar, criar, deletar) só nos e2e; em produção validar presença/empty state.
2. **Empty state é correto** — validar condicionalmente: "sem tickets fechados hoje" pode ser o estado legítimo.
3. **Timing**: após navegação/ações, voltar à página certa e aguardar o refetch antes de assertar (o link pode não ter re-renderizado).
4. **Filtrar 401 esperados** (login errado em teste) dos `console errors` antes de reportar falha.

## Passos

1. Abrir fluxos principais (login, listagem, detalhe) com o perfil de teste.
2. Capturar console errors/network failures separando esperados de reais.
3. Assertar presença de dados e empty states conforme o contexto.
4. Reportar apenas falhas REAIS (erro não-filtrado, ausência inesperada).

## Verificação

- Nenhum registro criado/alterado em produção (diff antes/depois).
- Relatório com falhas reais apenas (401 esperados filtrados).