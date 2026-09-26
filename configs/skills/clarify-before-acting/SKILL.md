---
name: clarify-before-acting
description: >-
  Mede a ambiguidade do pedido (0-100) e pergunta o mínimo antes de executar; também quando o alvo
  é o plano, não o pedido.
when_to_use: >-
  Pedido vago, com pronome solto ou mais de uma leitura; ou risco de você presumar uma premissa.
license: MIT
---


# Clarify Before Acting

## Regra de ouro

Melhor 3 perguntas do que executar a coisa errada e depois desfazer. A pergunta é
barata; o retrabalho não é.

## 1. Meça a ambiguidade (0–100)

Some os pontos do que **falta**:

| Sinal | Pontos |
|---|---|
| Sem arquivo ou pasta citado | 30 |
| "arruma isso", "melhora", "faz aí" | 25 |
| Pronome sem referente ("isso", "aquilo") | 20 |
| Mais de uma interpretação plausível | 20 |
| Critério de pronto indefinido | 15 |
| Requisito contraditório | 10 |

- **0–30**: execute. Não pergunte por formalidade.
- **31–60**: assuma o padrão mais óbvio, **diga qual foi** e siga.
- **61–100**: **pergunte antes de agir**.

O nível de detalhe da pergunta é proporcional ao que falta, não ao que você já sabe.

## 2. As perguntas mínimas

1. **Escopo**: o que entra e o que não entra.
2. **Alvo**: qual arquivo, qual serviço, qual ambiente.
3. **Critério de pronto**: como sei que terminou.
4. **Premissa**: o que estou assumindo sem ter sido dito.

Uma pergunta por vez, com **default explícito** e a opção de pular ("ou sigo com X?").

## 3. Quando o alvo é o plano, não o pedido

Antes de implementar uma decisão (arquitetura, refatoração ampla, escolha de
ferramenta), derrube a premissa frágil primeiro:

- Qual premissa, se falsa, invalida o plano inteiro?
- O que já foi **provado** sobre ela? (evidência `file:line`, não opinião)
- Qual é o plano B se ela for falsa?
- O que acontece se **não** fizermos nada?

Itere até sobrar só o que sustenta a decisão. Se a resposta for "não sei ainda",
isso **é** o achado — não um bloqueio para fingir certeza.

## 4. O que NÃO é motivo para perguntar

- O que você descobre em segundos lendo o código (`git log`, `rg`, `docs/`).
- Preferência que já está na memória ou no `AGENTS.md`.
- Escolha técnica reversível e sem custo: decida, registre a decisão e siga.

## Verificação

- [ ] A ambiguidade foi pontuada, ou eu disse por que não precisei pontuar.
- [ ] Se perguntou: foram as perguntas mínimas, com default.
- [ ] Se assumiu: o padrão ficou explícito na resposta.
- [ ] Nenhuma pergunta redundante com algo que eu já tinha lido.
