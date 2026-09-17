---
name: variant-analysis
description: >-
  Caçar as outras instâncias de um bug já encontrado — as variantes de uma mesma causa raiz no resto do código. Use logo após achar vulnerabilidade, bug de lógica ou padrão ruim num arquivo específico, quando a pergunta vira "onde mais isso ocorre". Triggers: tem outros iguais, mesmo bug em outro lugar, variantes, padrão se repete, sibling bug, mesma causa raiz, auditar ocorrências.
license: MIT
---

# Variant Analysis

Ache as outras instâncias do bug que você já achou. Uma causa raiz costuma ter várias manifestações — raramente todas no módulo onde a primeira apareceu.

## Quando usar

- Vulnerabilidade encontrada e precisa procurar instâncias similares
- Bug de lógica ou padrão ruim achado num arquivo; a pergunta vira "onde mais?"
- Generalizar uma instância conhecida numa busca pelo padrão inteiro
- Triar candidatos parecidos contra uma causa raiz conhecida

## Quando NÃO usar

- Descoberta inicial sem bug em mãos (use `systematic-debugging` primeiro)
- Code review geral sem padrão conhecido para procurar
- Entender código desconhecido (explore antes)

## Os 5 passos

### 1. Entenda a causa raiz

Extraia POR QUE o código está errado, não o que ele faz. Liste as direções onde uma variante pode se esconder: identificadores relacionados, outras manifestações do mesmo erro, edge cases de tipo (null, vazio, limite).

### 2. Crie o match exato

Escreva um padrão (`rg`/`fd`) que casa SOMENTE a instância conhecida e confirme que acerta. Padrão que não casa nada significa que você entendeu o bug errado — toda busca construída sobre ele calibra contra o código errado.

```bash
rg -n "ExpandUserPath\(.*\)" internal/ --type go
```

### 3–4. Generalize um elemento por vez

Suba do match exato em direção à família do padrão, rodando e lendo todos os matches após cada mudança única. Pare quando mais da metade dos matches virar ruído.

- Nunca generalize vários elementos de uma vez (o ruído fica impossível de atribuir).
- Não restrinja ao módulo do bug original — varie o escopo primeiro para o repo inteiro.

### 5. Triage com severidade

Decida quais candidatos são reais, com severidade anexada (bloqueador / deve corrigir / ruído). Para cada real: arquivo:linha + por que é a mesma causa + fix sugerido.

### Escreva o relatório

Inclua os padrões que falharam e, quando fizer sentido, uma regra de prevenção (teste, lint, check no `doctor`).

## Por que as caçadas falham

1. **Escopo estreito** — procurar só no módulo do bug original
2. **Padrão específico demais** — procurar um atributo e perder a família
3. **Uma classe só** — perseguir uma manifestação da causa raiz
4. **Só happy path** — nunca tentar null, vazio, boundary
5. **Generalizar rápido demais** — abstrair vários elementos de uma vez

## Integração

- Entrada típica: Fase 4 do `systematic-debugging` achou a causa raiz → rode esta skill antes de declarar corrigido.
- Saída típica: correções via ciclo `test-driven-development` (RED por variante real) + gate `verification-before-completion`.
