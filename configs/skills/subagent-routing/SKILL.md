---
name: subagent-routing
description: >-
  Roteamento e delegação de subagentes: quando delegar, qual tipo usar (explore para varredura de código, general para pesquisa/multi-passo), paralelo vs sequencial e quando NÃO delegar. Use ao encarar exploração de codebase sem alvo definido, pesquisa na internet/docs, debug sem causa conhecida, ou múltiplos domínios independentes. Triggers: subagente, delegar, dispatch, explorar codebase, pesquisar na internet, debug, paralelizar, preservar contexto.
license: MIT
---

# Roteamento de Subagentes (Delegação Proativa)

## Princípio

Suba o trabalho para **preservar o contexto do coordenador** e **paralelizar domínios independentes**. Cada subagente recebe contexto isolado e autocontido (nunca herda a sessão). O coordenador gasta o seu contexto integrando e verificando, não varrendo tudo.

## Quando delegar (e para quem)

Tipos comuns: `explore` (read-only, varredura de código) e `general` (execução/pesquisa multi-passo). O CommandCode tem ainda o `plan` (planejamento) — e `explore`/`plan`/`review`/`general` são nomes reservados lá.

| Situação | Subagente | Paralelizar? |
|---|---|---|
| Exploração de codebase sem arquivo-alvo ("onde está X", "como funciona Y") | `explore` | Sim, se 2+ áreas independentes — vários na MESMA resposta |
| Pesquisa na internet / docs de lib / versões / fatos que mudam | `general` (+ `context7-auto`/`WebSearch`/`WebFetch`) | Sim, se fontes independentes |
| Debug sem causa conhecida | `explore`/`general` por domínio | **Não primeiro** — investigue a causa raiz; paralelo só com falhas independentes |
| Tarefa pesada multi-passo (build, suíte de testes, crawler) | `general` ou skill `vps-agent-dispatch` (remoto) | Conforme independência |
| Planejamento de implementação | `plan` (built-in dispatchável no CommandCode; no OpenCode é primary, não dispatchável) | — |

## Regras

- **Paralelo = vários dispatches NA MESMA resposta** (rodam concorrentes). Um por resposta = sequencial.
- **Prompt de cada subagente**: escopo único, autocontido, output esperado explícito e constraints ("não tocar X — outro agente é dono").
- **NÃO delegar/paralelizar quando**: falhas relacionadas (consertar uma pode consertar outra), estado compartilhado (mesmos arquivos/recursos), debug exploratório sem domínios definidos, ou quando entender o sistema exige o contexto inteiro.
- Delegue o "barulhento" (varredura ampla, output volumoso) para não poluir o contexto do coordenador.
- Mecânica de execução paralela: skill `dispatching-parallel-agents`. Múltiplos agentes no MESMO repo git: skill `parallel-agent-orchestration`.

## Verificação

- Contexto do coordenador preservado (não leu o dump inteiro do subagente — só o resumo).
- Subagentes em domínios disjuntos, sem edições conflitantes.
- Build + suíte completa rodados **após** a integração (nunca confiar só no retorno dos subagentes).
