# Atribuição de Skills (autores e upstreams)

Metade das skills deste repositório **não foi escrita do zero aqui**. Este documento registra de onde cada uma veio, o que foi adaptado e qual é a regra para incluir skill de terceiro daqui pra frente.

> **Regra do projeto:** toda skill adotada de terceiro **declara o autor e o repositório de origem** no próprio `SKILL.md`, no bloco `metadata`, e mantém a licença do upstream. Skill sem upstream identificado declara-se autoral. Nada de crédito implícito.

```yaml
# formato usado no frontmatter
license: MIT
metadata:
  author: <autor/handle do upstream>
  source: https://github.com/<owner>/<repo>
  adapted: envctl (descricao e triggers em PT-BR, caminhos e comandos ajustados)
```

---

## 1. Derivadas de terceiros (crédito obrigatório)

| Upstream | Licença | Skills derivadas |
|---|---|---|
| [obra/superpowers](https://github.com/obra/superpowers) — *An agentic skills framework & software development methodology that works* | MIT | `dispatching-parallel-agents`, `receiving-code-review`, `systematic-debugging`, `using-git-worktrees`, `verification-before-completion`, `writing-plans` |
| [mattpocock/skills](https://github.com/mattpocock/skills) — *Skills for Real Engineers* | MIT | `grill-me`, `grilling`, `handoff` |
| [hqhq1025/skill-optimizer](https://github.com/hqhq1025/skill-optimizer) — *Agent Skills lifecycle toolkit* | MIT | `skill-miner`, `skill-personalizer`, `skill-generalizer` |
| Hardik Pandya — [hvpandya.com](https://hvpandya.com) | MIT (declarada no corpo da skill) | `stop-slop` |

Licenças verificadas via API do GitHub em 2026-09-16 (`license.spdx_id = MIT` em todos os três repositórios).

### O que foi adaptado nas cópias (medido, não estimado)

Corpo de cada `SKILL.md` comparado com o do upstream em 2026-09-16:

| Skill | Corpo | Observação |
|---|---|---|
| `dispatching-parallel-agents`, `receiving-code-review`, `systematic-debugging`, `using-git-worktrees`, `verification-before-completion`, `skill-miner`, `skill-personalizer`, `skill-generalizer` | **idêntico** | só a `description` (e os triggers em PT-BR) foi reescrita — é o texto contra o qual o agente faz o matching automático |
| `writing-plans` | idêntico + 1 edição | os planos vão para `spec-agent/` na raiz; o upstream gravava em `docs/superpowers/plans/` (pasta do projeto dele) |
| `handoff` | idêntico + 1 edição | a linha que citava a "Skill tool" foi parametrizada por agente (`activate_skill` / `/<skill>`) |
| `grilling` | **ampliado** (+3 parágrafos) | o núcleo do *design tree* é do upstream; acrescentamos as rodadas/fronteira, o dispatch de subagente para buscar fatos e o critério de fim |
| `grill-me` | **reescrito** | o upstream tem 2 linhas (`Call the Skill tool with "grilling"` + `disable-model-invocation: true`). A nossa versão é um porteiro de ambiguidade — rubrica 0–100, limiares e cláusula de risco são **originais daqui** |

Arquivos auxiliares mantidos: `agents/openai.yaml` em `grilling`, `handoff` e nos três `skill-*`; `scripts/scan_sessions.py` + `references/` em `skill-miner`; `references/` em `skill-personalizer` e `skill-generalizer`. **Não copiamos** o `plan-document-reviewer-prompt.md` do upstream `writing-plans` — ele não é referenciado em nenhum ponto da nossa cópia.

Nada aqui está sendo redistribuído como trabalho original: o crédito ao autor está no `metadata` de cada arquivo e a licença declarada é a do upstream.

## 2. Autorais do envctl (sem upstream)

As skills abaixo foram escritas aqui — a maioria **promovida da memória do próprio usuário** pelo pipeline `memory-promotion` (commit `216ccd9`, "11 promoted skills"), ou específicas deste ambiente (VPS, Windows, Docker, banco):

`agent-memory`, `api-contract-design`, `ask-questions-if-underspecified`, `aur-headless-install`, `bulk-postgres-import`, `cachyos-gaming-setup`, `context7-auto`, `database-ops`, `docker`, `docker-build-local-vps-deploy`, `docker-desktop-wsl-restart`, `git-workflow`, `headless-gui-probe`, `jwt-hs256-node`, `lsp-smoke-test`, `memory-promotion`, `nextjs-standalone-deploy`, `parallel-agent-orchestration`, `phone-e164-normalization`, `playwright-prod-regression`, `simple-feature-flag`, `ssh-vps`, `subagent-routing`, `universal-test-runner`, `vps-agent-dispatch`, `vps-provisioning`, `web-dashboard-automation`, `windows-admin`

**Pendência declarada:** duas destas têm estilo de skill de terceiro e **origem não confirmada** — `ask-questions-if-underspecified` e `context7-auto`. Se você reconhecer o upstream, adicione `metadata.author`/`source` nelas. Enquanto isso, ficam como autorais.

## 3. Licença do projeto

O repositório tem um `LICENSE` (MIT) na raiz, que cobre as skills autorais e o código do `envctl`. As derivadas mantêm a licença MIT dos seus upstreams — como todas são MIT, o conjunto é compatível.

## 4. Checklist para adotar skill de terceiro

1. Verifique a licença do upstream (`gh api repos/<owner>/<repo> --jq .license.spdx_id`). Só adote licença compatível (MIT/Apache-2.0/BSD); **não** adote sem licença ou copyleft forte sem decisão explícita.
2. Copie para `configs/skills/<nome>/`, mantenha o nome da skill igual ao do upstream quando fizer sentido.
3. Preencha o `metadata` com `author`, `source` e `adapted`.
4. Registre a linha na tabela da seção 1 deste arquivo.
5. Rode `envctl run skills` e confirme no `cmdc skills list` que a skill carrega.
