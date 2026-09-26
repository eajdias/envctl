---
name: systematic-debugging
description: >-
  Depuração sistemática em 5 fases: causa raiz antes de correção, e depois as variantes do mesmo
  bug.
when_to_use: >-
  Bug, erro, teste vermelho, exceção, intermitência ou comportamento inesperado — e, já corrigido,
  procurar as outras instâncias.
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

Triggers: bug, erro, falha, não funciona, quebrou, teste falhando, stack trace, exceção, comportamento inesperado, causa raiz, reproduzir, regressão, investigar, procurar outras instâncias, onde mais isso acontece, variante do mesmo bug.
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
3. Verifique: teste passa, quebrou outro? A regra de evidência do `AGENTS.md` (verificação bloqueante, o que não pôde ser verificado mantém a tarefa **não concluída**) vale aqui: sem comando rodado, não declare resolvido.
4. **Se a correção falhar:** < 3 tentativas → volte à Fase 1. **≥ 3 falhas → questione a arquitetura** (não tente a #4 sem discutir).

## Fase 5 — Variantes ("onde mais isso acontece?")

Corra **antes** de declarar corrigido. Uma causa raiz quase sempre tem várias manifestações, e raramente todas estão no módulo onde a primeira apareceu.

1. **Extraia o por, não o quê.** Liste as direções onde uma variante pode se esconder: identificadores relacionados, outras manifestações do mesmo erro, edge cases de tipo (nil, vazio, limite).
2. **Crie o match exato** e confirme que acerta **só** a instância conhecida. Padrão que não casa nada significa que você entendeu o bug errado — toda busca construída sobre ele calibra contra o código errado.
   ```bash
   rg -n "ExpandUserPath\(.*\)" internal/ --type go
   ```
3. **Generalize um elemento por vez**, rodando e lendo todos os matches a cada mudança. Pare quando mais da metade virar ruído. Nunca generalize vários elementos de uma vez (o ruído fica impossível de atribuir) nem restrinja ao módulo original: varie o escopo primeiro para o repo inteiro.
4. **Teste o infeliz**: nil, vazio, boundary, happy path invertido. Variante que só aparece em edge case é a que a suíte não pega.
5. **Triage com severidade** (bloqueador / deve corrigir / ruído). Para cada real: `arquivo:linha` + por que é a mesma causa + fix sugerido.

Relatório: inclua os padrões que **falharam** (a lista do que não era o bug vale tanto quanto a do que era) e, quando couber, uma regra de prevenção — teste, lint ou check no `doctor`.

**Por que as caçadas falham:** escopo estreito · padrão específico demais (perde a família) · perseguir só uma classe · só happy path · generalizar rápido demais.

## Red flags — PARE e volte à Fase 1

"correção rápida depois investigo", "vamos tentar mudar X", "sem teste, verifico manual", "deve ser X", "mais uma tentativa (já tentei 2+)", listar correções sem ter rastreado o fluxo.

## Ferramentas de apoio (no diretório)

`root-cause-tracing.md` · `defense-in-depth.md` · `condition-based-waiting.md`
