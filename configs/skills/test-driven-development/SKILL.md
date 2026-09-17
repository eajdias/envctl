---
name: test-driven-development
description: >-
  Desenvolver em ciclo TDD red-green-refactor: teste falhando primeiro, código mínimo depois. Use em feature nova, bugfix, refatoração ou mudança de comportamento em TS/Node, Python ou Go. Triggers: tdd, red green refactor, teste primeiro, failing test, ciclo tdd, disciplina tdd, teste antes do código.
license: MIT
metadata:
  author: obra (superpowers)
  source: https://github.com/obra/superpowers
  adapted: envctl — ideia (lei de ferro + red-green-refactor) adaptada; ciclo e exemplos escritos do zero para TS/PY/GO com vitest/pytest/go test e wiring com universal-test-runner + verification-before-completion
---

# Test-Driven Development

**Lei de ferro:** `SEM CÓDIGO DE PRODUÇÃO SEM TESTE FALHANDO PRIMEIRO.` Se não viu o teste falhar pelo motivo certo, não sabe se ele testa a coisa certa.

## Quando usar

**Sempre:** feature nova, bugfix, refatoração, mudança de comportamento.

**Exceções (pergunte):** protótipo descartável, código gerado, arquivo de config.

"Vou pular o TDD só dessa vez"? Pare. Isso é racionalização.

## Ciclo Red-Green-Refactor

### RED — escreva o teste falhando

Um teste mínimo mostrando o comportamento esperado. Um comportamento por teste; nome claro; código real (sem mock, salvo inevitável).

### Verifique o RED — veja falhar (obrigatório, nunca pule)

Rode o teste e confirme: falha (não erro), mensagem esperada, falha porque o recurso não existe (não typo). Teste passou de primeira? Está testando comportamento existente — corrija o teste. Teste com erro? Corrija até falhar do jeito certo.

### GREEN — código mínimo

O mais simples que faz o teste passar. Sem features extras, sem refatorar outro código, sem "melhorar" além do teste.

### Verifique o GREEN — veja passar (obrigatório)

Teste passa + demais testes continuam verdes + saída limpa (sem erros/warnings).

### REFACTOR — limpe (só depois do verde)

Remova duplicação, melhore nomes, extraia helpers. Mantenha tudo verde. Sem comportamento novo.

### Repita

Próximo teste falhando para a próxima fatia.

## Ciclos por linguagem

### TS/Node (vitest)

```bash
npx vitest run path/to/file.test.ts        # RED: confirma FAIL esperado
# ... código mínimo ...
npx vitest run path/to/file.test.ts        # GREEN: confirma PASS
npx tsc --noEmit                           # type-check antes de declarar verde
```

### Python (pytest)

```bash
pytest tests/test_x.py -k test_nome -v     # RED: confirma FAIL esperado
# ... código mínimo ...
pytest tests/test_x.py -k test_nome -v     # GREEN: confirma PASS
ruff check . && ruff format --check .      # lint antes de declarar verde
```

### Go (go test)

```bash
go test ./pkg/x -run TestNome -v          # RED: confirma FAIL esperado
# ... código mínimo ...
go test ./pkg/x -run TestNome -v          # GREEN: confirma PASS
go vet ./pkg/x                             # vet antes de declarar verde
```

Cobertura e suíte completa via `universal-test-runner`; gate final de conclusão via `verification-before-completion`. Bug encontrado? Escreva o teste que o reproduz e entre no ciclo (integra com `systematic-debugging`: a Fase 4 dela já pede RED primeiro).

## Racionalizações comuns

| Desculpa | Realidade |
|---|---|
| "Simples demais para testar" | Código simples quebra. O teste leva 30 segundos. |
| "Testo depois" | Teste escrito depois passa de imediato — o que prova nada. Pode testar a coisa errada, a implementação em vez do comportamento. Sem ver falhar, nunca provou que ele pega o bug. |
| "Já testei manual" | Manual é ad-hoc: sem registro do que cobriu, sem re-run quando o código muda. "Funcionou quando tentei" ≠ cobertura. |
| "Apagar X horas é desperdício" | Sunk cost — o tempo já foi gasto. A escolha real: reescrever com TDD (alta confiança) vs. manter código não-confiável (bugs prováveis). |
| "Preciso explorar primeiro" | Ok. Jogue a exploração fora e comece com TDD. |
| "TDD me atrasa" | TDD pega bugs antes do commit, previne regressão, permite refatorar sem medo. Debugar em produção é mais lento. |

## Red flags — PARE e recomece

Código antes do teste · teste depois da implementação · teste passou de imediato · não sabe explicar por que o teste falhou · testes "depois" · "só dessa vez" · "já testei manual" · "manter de referência e adaptar" · "TDD é dogma, sou pragmático".

**Todos significam: apague o código. Recomece com TDD.**

## Checklist antes de declarar pronto

- [ ] Toda função/método novo tem teste
- [ ] Viu cada teste falhar antes de implementar
- [ ] Cada um falhou pelo motivo esperado (recurso ausente, não typo)
- [ ] Escreveu o código mínimo para passar
- [ ] Todos os testes passam; saída limpa
- [ ] Edge cases e erros cobertos
