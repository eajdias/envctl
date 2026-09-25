---
name: verification-before-completion
description: >-
  Exigir evidência empírica ANTES de declarar algo pronto, corrigido ou passando: rodar build, testes e lint e mostrar a saída real antes de commitar ou abrir PR. Sem verificação fresca nesta mensagem, sem claim de sucesso. Use automaticamente ao concluir tarefa, fechar TODO ou delegar a agentes. Triggers: terminei, está pronto, funcionou, corrigido, passou nos testes, evidência, antes de commitar, validar, build/test/lint, não afirmar sem rodar, regressão.
license: MIT
metadata:
  author: obra (superpowers)
  source: https://github.com/obra/superpowers
  adapted: envctl — reescrito e enxuto (de ~126 para ~50 ln); mecânica preservada
---

# Verification Before Completion

**Lei de ferro:** `SEM CLAIM DE CONCLUSÃO SEM EVIDÊNCIA FRESCA DE VERIFICAÇÃO.` Se não rodou o comando de verificação nesta mensagem, não pode dizer que passou.

## O gate (antes de qualquer claim de sucesso)

1. **Identifique** qual comando prova o claim.
2. **Rode** o comando completo (saída fresca).
3. **Leia** a saída + exit code + contagem de falhas.
4. **Verifique** se a saída confirma o claim — se não, reporte o estado real com evidência.
5. **Só então** faça o claim. Pular qualquer passo = mentir, não verificar.

## Quando aplicar

Antes de: claim de sucesso/conclusão, satisfação ("pronto!", "done!"), commitar, abrir PR, fechar TODO, passar pra próxima tarefa, confiar em relatório de agente.

## Padrões

- Testes: `✅ [rode] 34/34 passam` · `❌ "deve passar"`
- Build: `✅ [rode] exit 0` · `❌ "lint passou" (lint ≠ compilador)`
- Agente: `✅ pediu sucesso → checo diff → verifico` · `❌ confio no relatório`

## Red flags — PARE

"deve", "provavelmente", "parece que", satisfação antes da verificação, commitar sem rodar, confiar em agente, "só dessa vez", "parcial já basta", qualquer wording de sucesso sem ter rodado.
