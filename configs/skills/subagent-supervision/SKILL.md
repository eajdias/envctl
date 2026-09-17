---
name: subagent-supervision
description: >-
  O coordenador monitora o progresso de subagentes paralelos e, se perceber que um está alucinando, em loop de tool calls, travado ou sem progresso, mata o subagente com `agent_output(action: "kill")` e decide se tenta de novo (com prompt refinado) ou escala ao usuário. Use logo após despachar 2+ subagentes em paralelo ou quando um demorar além do esperado. Triggers: subagente não retorna, loop de tool calls, alucinação, sem progresso, demorou demais, matar subagente, retry, escalar, coordenador, background agent, agent_output kill.
license: MIT
---

# Subagent Supervision (o coordenador vigia os subagentes)

Ao despachar subagentes em paralelo (via `agent(run_in_background: true)`), o coordenador **não espera passivamente**. Ele monitora progresso e age quando algo sai do esperado.

## Quando aplicar

- Você despachou 2+ subagentes em background e precisa que **todos** terminem bem.
- Um subagente está demorando mais que o razoável.
- Você suspeita que um está alucinando, em loop ou travado.

## Como monitorar

Subagentes em background devolvem um `agent_id`. Com ele:

- `agent_output(agent_id, action: "status")` — está rodando, concluído ou morto?
- `agent_output(agent_id, action: "wait")` — bloqueia até terminar (use com parcimônia).
- `agent_output(agent_id, action: "kill")` — **mata o subagente** e libera o coordenador.

## Detectar problemas

Sinais de que um subagente precisa ser morto:

- **Loop de tool calls:** mesma ferramenta/chamada repetida dezenas de vezes sem avançar.
- **Alucinação:** outputs que contradizem fatos conhecidos, inventam arquivos/APIs, "alucinam" conclusões.
- **Sem progresso:** está rodando há muito mais que o esperado sem produzir resultado parcial.
- **Travado:** não emite tool calls nem outputs por um longo período.

## Ação ao detectar

1. **Mate:** `agent_output(agent_id, action: "kill")`.
2. **Decida:**
   - **Retry** — se faz sentido: despache um novo subagente com o **mesmo escopo mas prompt refinado** (explique o que estava errado, dê fronteiras mais restritas, peça outputs intermediários).
   - **Escale ao usuário** — se 2+ tentativas no mesmo escopo falharam, pare e explique o bloqueio. Não entre em loop de retry infinito.
3. **Integre** — quando os subagentes saudáveis retornarem, combine os resultados e valide (build + suíte completa).

## Regras

- Nunca deixe um subagente problemático rodando "só mais um pouco" indefinidamente — mate cedo.
- Nunca faça retry infinito no mesmo escopo: 2 tentativas, depois escale ao usuário.
- Sempre valide a integração final (build/testes) — subagentes individuais podem passar e o conjunto quebrar.
- Preserve o contexto do coordenador: monitore via `agent_output(status)` em vez de esperar cada resultado inteiro.

## Verificação

- Todos os subagentes despachados terminaram (status concluído ou kill justificado)?
- Resultados integrados e suíte passando?
- Nenhum loop/alcoolização sobreviveu?
