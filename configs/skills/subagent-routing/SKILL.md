---
name: subagent-routing
description: >-
  Roteamento e despacho de subagentes: QUANDO delegar (explore/general/planner), COMO despachar em paralelo na mesma resposta (dispatch múltiplo), roteiro por situação, isolamento de contexto, integração final e quando NÃO delegar. A execução inline é o padrão; use subagente apenas quando o contexto bruto da pesquisa/planejamento for volumoso e o resultado puder ser compactado. Inclui mecânica de despacho paralelo e orquestração no mesmo repositório (fronteiras disjuntas, base comum). Use automaticamente para decidir se o isolamento de contexto compensa. Triggers: subagente, delegar, despachar, dispatch, paralelo, mesmo repositório, orquestrador, fronteira, contexto isolado, integrar, inline, preservar contexto.
license: MIT
---

# Subagent Routing (Roteamento & Despacho de Subagentes)

Suba trabalho para **preservar o contexto do coordenador** e **paralelizar domínios
independentes**. Cada subagente recebe contexto isolado e autocontido — nunca herda a
sessão. O coordenador gasta o seu contexto integrando e verificando, não varrendo tudo.

## Default: inline

Implementação, correção, investigação com arquivos conhecidos e alterações pequenas
continuam inline. O cache-hit e a continuidade da sessão são mais baratos que uma
sessão nova. Só despache quando a pesquisa/planejamento produzir um transcript
volumoso que o coordenador não precisa carregar adiante, quando houver paralelismo
real ou quando uma avaliação independente for explicitamente útil.

Todo subagente deve devolver um artefato compacto: decisão, evidência `file:line`,
arquivos afetados, riscos/unknowns e próximos passos. Não retransmita explorations
brutas ao coordenador.

## Quando delegar (e para quem)

| Situação | Subagente | Paralelizar? |
|---|---|---|
| Exploração de codebase sem arquivo-alvo ("onde está X", "como funciona Y") | `explore` | Sim — vários na mesma resposta, se 2+ áreas independentes |
| Pesquisa na internet / docs de lib / versões / fatos que mudam | `general` (+ `context7-auto`/`WebSearch`/`WebFetch`) | Sim, se o resultado puder ser compactado |
| Debug sem causa conhecida | `explore`/`general` por domínio | Não primeiro — investigue a causa raiz; paralelo só com falhas independentes |
| Tarefa pesada multi-passo (build, suíte, crawler) | `general` ou `vps-agent-dispatch` (remoto) | Conforme independência |
| Planejamento de implementação | `planner` (OpenCode) ou `plan` (CommandCode) | Não por padrão; apenas para plano extenso |

Tipos comuns: `explore` (read-only, varredura), `general` (execução/pesquisa),
`planner` (OpenCode, planejamento dispatchável). No CommandCode `plan` é built-in
dispatchável; no OpenCode o `planner` é a variante dispatchável e o `plan` nativo
continua sendo o agente primary.

## Mecânica do despacho paralelo

**Paralelo = vários `agent`/dispatch na MESMA resposta** (roda concorrente). Um por
resposta = sequencial.

Cada dispatch recebe:
- **Escopo único** — um domínio/problema
- **Contexto autocontido** — tudo que o agente precisa, sem depender da sessão
- **Output esperado explícito** — o que ele deve devolver (resumo, evidência `file:line`, decisão)
- **Constraints** — "não tocar X — outro agente é dono", não tocar fora da fronteira

Exemplo (3 dispatches simultâneos):
```
agent(explore) → "Onde está a lógica de cobrança? Retorne os paths."
agent(general) → "Qual a versão estável da lib X hoje? Use Context7."
agent(explore) → "Liste os endpoints de pagamento com method:line."
```

## Orquestração no mesmo repositório

Quando subagentes paralelos tocam o **mesmo repo git**, siga estes 5 passos:

1. **Base comum primeiro**: orquestrador aplica schema/shared/bootstrap num commit
   próprio, antes de despachar.
2. **Fronteiras disjuntas explícitas**: cada subagente recebe a lista exata de arquivos que
   pode tocar + proibições ("não tocar X — outro agente é dono").
3. **Despacho paralelo** em contexto isolado (sem estado compartilhado).
4. **Integração pelo orquestrador no final**: registros, exports, composição.
5. **Suíte completa após integração** (build + testes) — nunca confiar só nos testes
   individuais.

## Quando NÃO delegar

- Falhas relacionadas (consertar uma pode consentar outra) — investigue junto primeiro.
- Precisa entender o estado inteiro do sistema.
- Debug exploratório sem domínios definidos.
- Estado compartilhado (mesmos arquivos/recursos) entre agentes.

## Após o retorno

- Leia o resumo de cada subagente; verifique se não há conflitos.
- Rode a suíte completa (build + testes) após integrar.
- Confirme que nenhum arquivo fora da fronteira foi modificado (diff por agente).
