---
name: grill-me
description: >-
  Porteiro de ambiguidade: mede de 0 a 100 o quão claro está o pedido e, quando a nota é alta, faz as perguntas de esclarecimento ANTES de executar qualquer coisa — melhor perguntar a mais do que executar errado e ter que desfazer depois. Use no início de qualquer tarefa em que o alvo, o escopo ou o critério de pronto não estejam evidentes: prompt curto ou vago ("arruma isso", "melhora", "deixa melhor", "faz aí"), pronomes soltos sem referente, múltiplas interpretações plausíveis, nenhum arquivo/pasta citado, ou quando você perceber que está prestes a adivinhar. Triggers: ambíguo, vago, confuso, não ficou claro, não sei o que você quer, faltou contexto, qual arquivo, em que pasta, esclarecer antes, grill, ambiguity, clarify first.
license: MIT
metadata:
  author: Matt Pocock
  source: https://github.com/mattpocock/skills
  adapted: envctl (descricao e triggers em PT-BR, caminhos e comandos ajustados)
---

# Grill Me — porteiro de ambiguidade

**Regra de ouro: é melhor fazer 3 perguntas do que executar a coisa errada e depois desfazer.**

## 1. Meça a ambiguidade (0–100)

Antes de agir, dê uma nota ao pedido somando os pontos do que **falta**:

| Sinal que falta | Pontos |
|---|---|
| Alvo físico: qual arquivo, pasta, repositório, serviço ou host | +25 |
| Ação: o que exatamente deve mudar (verbo vago: "melhora", "arruma", "otimiza") | +20 |
| Critério de pronto: como se sabe que terminou | +20 |
| Escopo: o que está dentro e o que está fora | +15 |
| Restrição: compatibilidade, versão, prazo, estilo, ambiente | +10 |
| Referente: pronomes soltos ("isso", "aquilo", "ele") sem antecedente | +10 |

Se o pedido tem **duas leituras plausíveis**, a nota é no mínimo 50 — mesmo que na hora pareça claro.

## 2. Compare com o limiar

- **0–20** — claro. Execute direto, sem perguntar nada.
- **21–50** — falta detalhe, não direção. Pergunte **só os bloqueadores** (1–3 perguntas), oferecendo o default que você adotaria.
- **51–100** — **não execute nada ainda**. Pergunte primeiro: nesse nível, qualquer trabalho tem boa chance de ser jogado fora.

Ajuste pelo risco: se a operação é **destrutiva ou difícil de reverter** (apagar, sobrescrever, force-push, migration, deploy, gasto de dinheiro, mensagem a terceiro), trate ambiguidade > 0 como bloqueio — pergunte mesmo com a nota baixa.

## 3. Como perguntar

Use a tool `ask_user_question` (até 4 perguntas, 2–4 opções cada) ou o padrão da skill `ask-questions-if-underspecified`:

- Pergunte o **mínimo que destrava** — não faça interrogatório.
- Ofereça **opções concretas**, com uma marcada como **(recomendado)**.
- Diga o que você faria se o usuário não respondesse (default declarado).
- Separe "preciso saber" de "seria bom saber".
- Aceite um atalho: "pode usar os defaults" libera a execução.

## 4. Depois de perguntar

Reafirme em 1–3 frases o que você entendeu (objetivo, escopo, pronto, restrições) e só então comece. Se o usuário mandar seguir sem responder, liste as premissas que está assumindo e execute.

## 5. Nunca

- Nunca invente o requisito que falta nem "escolha o mais provável" só para não incomodar.
- Nunca execute operação irreversível com ambiguidade acima do limiar de risco.
- Nunca pergunte o que uma leitura barata responde (config do repo, estrutura de pastas, `git status`).
- Nunca transforme isto em burocracia: com nota ≤ 20, execute e mostre o resultado.

## Fronteira com as skills vizinhas

- **`ask-questions-if-underspecified`** — a *mecânica* de perguntar bem (templates, opções, resposta compacta). É para onde você vai depois de decidir perguntar.
- **`grilling`** — *arguição* adversarial de um plano/ideia que já existe, em rodadas de design tree. Outro propósito: aqui a ambiguidade é do pedido; lá é das premissas.
