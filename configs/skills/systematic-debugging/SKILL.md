---
name: systematic-debugging
description: >-
  Depuração sistemática em 4 fases: achar a causa raiz antes de propor correção (investigação →
  padrão → hipótese → teste).
when_to_use: >-
  Bug, erro, teste vermelho, exceção, intermitência ou comportamento inesperado — antes de tentar
  corrigir qualquer coisa.
license: MIT
metadata:
  author: obra (superpowers)
  source: https://github.com/obra/superpowers
  adapted: envctl — reescrito e enxuto (de ~289 para ~80 ln); metodologia preservada
---


# Systematic Debugging


## Triggers (lista estendida)

Viva na lista de catálogo do OpenCode, truncada em 249 chars pelo CommandCode — por isso o resumo
acima é curto. Quando a skill carregar, use esta lista para casar o pedido:

Triggers: bug, erro, falha, não funciona, quebrou, teste falhando, stack trace, exceção, comportamento inesperado, causa raiz, reproduzir, regressão, investigar.
**Lei de ferro:** `SEM CORREÇÃO SEM INVESTIGAÇÃO DE CAUSA RAIZ PRIMEIRO.` Se não completou a Fase 1, não proponha correção.

## Fase 1 — Causa raiz (antes de qualquer correção)

1. **Leia o erro por completo** — stack trace, caminhos, códigos, linhas.
2. **Reproduza de forma consistente** — passos exatos; se não reproduzir, colete mais dados, não chute.
3. **Cheque mudanças recentes** — `git diff`, commits, deps, ambiente.
4. **Em sistemas multi-componente**, instrumente cada fronteira (loga entrada/saída, estado) para achar ONDE quebra. Ver `root-cause-tracing.md` no diretório da skill.
5. **Trace o fluxo de dados** — suba até a origem do valor ruim; corrija na fonte, não no sintoma.

## Fase 2 — Padrão

- Ache exemplos funcionando no mesmo código; compare com o que quebra.
- Liste toda diferença (por menor que pareça) e entenda as dependências.

## Fase 3 — Hipótese e teste

- Formule UMA hipótese clara ("acho que X é a causa porque Y").
- Teste com a MENOR mudança possível, uma variável por vez.
- Não sabe? Diga "não entendo X" e peça ajuda — não finja.

## Fase 4 — Correção

1. Crie um teste que reproduza o falha (TDD: RED primeiro).
2. Corrija a causa raiz — UMA mudança, sem "já que estou aqui".
3. Verifique: teste passa, quebrou outro? Use `verification-before-completion` antes de declarar resolvido.
4. **Se a correção falhar:** < 3 tentativas → volte à Fase 1. **≥ 3 falhas → questione a arquitetura** (não tente a #4 sem discutir).

## Red flags — PARE e volte à Fase 1

"correção rápida depois investigo", "vamos tentar mudar X", "sem teste, verifico manual", "deve ser X", "mais uma tentativa (já tentei 2+)", listar correções sem ter rastreado o fluxo.

## Ferramentas de apoio (no diretório)

`root-cause-tracing.md` · `defense-in-depth.md` · `condition-based-waiting.md`
