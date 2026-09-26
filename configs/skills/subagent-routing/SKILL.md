---
name: subagent-routing
description: >-
  Roteamento de subagentes: quando delegar (inline é o padrão), qual tipo, despacho paralelo no
  repo e integração.
when_to_use: >-
  Tarefa bulky (pesquisa, varredura, plano extenso) ou o usuário pede delegar, paralelizar,
  dividir, economizar contexto.
license: MIT
---


# Subagent Routing (Roteamento & Despacho de Subagentes)


## Triggers (lista estendida)

Viva na lista de catálogo do OpenCode, truncada em 249 caracteres pelo CommandCode — por isso o
resumo da description acima é curto. Quando a skill carregar, use esta lista para casar o pedido:

Triggers: subagente, delegar, despachar, dispatch, paralelo, mesmo repositório, orquestrador,
fronteira, contexto isolado, integrar, inline, preservar contexto.
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
| Pesquisa na internet / docs de lib / versões / fatos que mudam | `general` (+ MCP `context7`/`WebSearch`/`WebFetch`) | Sim, se o resultado puder ser compactado |
| Debug sem causa conhecida | `explore`/`general` por domínio | Não primeiro — investigue a causa raiz; paralelo só com falhas independentes |
| Tarefa pesada multi-passo (build, suíte, crawler) | `general`, ou máquina remota por SSH | Conforme independência |
| Planejamento de implementação | `planner` (OpenCode) ou `plan` (CommandCode) | Não por padrão; apenas para plano extenso |
| Revisão de código/diff com severidades e veredito (segundo parecer) | `reviewer` (OpenCode) ou `code-reviewer` (CommandCode) | Não — um por vez, no fim da tarefa |
| Prova de que a árvore está verde (gate: build, vet, test, lint) | `verifier` (OpenCode e CommandCode) | Não — é um gate sequencial |
| Sincronizar docs contra manifesto/código | `docs-writer` (OpenCode e CommandCode) | Não — um por vez, depois que o código parou |
| Fechar lições e patterns na memória | `memory-keeper` (OpenCode e CommandCode) | Não — fecho da tarefa |

Tipos comuns: `explore` (read-only, varredura), `general` (execução/pesquisa),
`planner` (OpenCode, planejamento dispatchável), `reviewer`/`code-reviewer`
(revisão read-only), `verifier` (gate), `docs-writer` (drift de doc) e
`memory-keeper` (memória). No CommandCode `plan` é built-in dispatchável; no
OpenCode o `planner` é a variante dispatchável e o `plan` nativo continua sendo
o agente primary.

**Os dois runtimes nomeiam o reviewer diferente, e isso é intencional:** no
CommandCode o agente se chama `code-reviewer`; no OpenCode, `review` é
`mode: primary` e por isso **não** aparece no catálogo de subagentes (mesma
semântica que excluiu o `plan`), então o dispatchável se chama `reviewer`.
Despeje o `reviewer` no Tab do OpenCode quando quiser revisar na mão — é o mesmo
corpo de prompt do `review`.

**Por que não existe `planner` no CommandCode:** o `plan` de lá já é dispatchável
(built-in, `tools: ["read_file"]`) e read-only. Um agente custom que gravasse a
spec precisaria de `write_file`/`edit_file` **sem escopo de path** — não existe
`Edit(spec-agent/**)` naquele runtime, e `permissionMode: plan` esconde as write
tools. Trocar boundary forte por convenience de escrita não compensa: o `plan`
planeja e o **coordenador** materializa `spec-agent/`.

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
