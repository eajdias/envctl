# Atribuição de Skills do envctl

**Não existe obrigação de fidelidade 1:1 com nenhum upstream.** Toda skill deste repositório é sua para adaptar, reescrever e ajustar às suas preferências, ao seu estilo de programação e ao que fizer sentido para o seu ambiente — independentemente de onde a ideia original veio. O que este documento registra é apenas **de onde a ideia veio** (para dar o crédito devido) e **o que foi adaptado** em cada cópia. Nenhuma skill aqui está presa a "espelhar" o autor original: se amanhã você quiser reescrever uma derivada do zero, pode.

O crédito ao upstream (declaro no `metadata` de cada `SKILL.md`) já cumpre o papel de reconhecer quem trouxe a ideia; o resto é seu.

```yaml
# formato usado no frontmatter das que vieram de terceiro
license: MIT
metadata:
  author: <autor/handle do upstream>
  source: https://github.com/<owner>/<repo>
  adapted: <o que foi mudado — ver a tabela>
```

---

## 1. Com upstream conhecido (ideia veio daqui)

| Upstream | Licença | Skills | Como foi adaptado (medido) |
|---|---|---|---|
| [obra/superpowers](https://github.com/obra/superpowers) | MIT | `dispatching-parallel-agents`, `receiving-code-review`, `systematic-debugging`, `using-git-worktrees`, `verification-before-completion` | Corpo **idêntico** ao upstream — só a `description`/triggers foram reescritos em PT-BR (o texto que dirige o matching). Você pode reescrever o corpo quando quiser. |
| [obra/superpowers](https://github.com/obra/superpowers) | MIT | `writing-plans` | Corpo idêntico + 1 edição: os planos agora vão para `spec-agent/` na raiz do projeto (o upstream gravava em `docs/superpowers/plans/`, pasta dele). |
| [mattpocock/skills](https://github.com/mattpocock/skills) | MIT | `grill-me`, `grilling`, `handoff` | `handoff`: corpo idêntico + a linha de invocação da tool parametrizada por agente. `grilling`: núcleo do *design tree* é do upstream; acrescentamos 3 parágrafos (rodadas/fronteira, dispatch de subagente para fatos, critério de fim). `grill-me`: **reescrito** — o upstream são 2 linhas mandando invocar `/grilling`; a rubrica 0–100, os limiares e a cláusula de risco são nossos. |
| [hqhq1025/skill-optimizer](https://github.com/hqhq1025/skill-optimizer) | MIT | `skill-miner`, `skill-personalizer`, `skill-generalizer` | Corpo **idêntico** — só a `description`/triggers em PT-BR. Mantidos `agents/openai.yaml`, `scripts/scan_sessions.py` + `references/` (skill-miner) e `references/` (personalizer/generalizer). |
| Hardik Pandya — [hvpandya.com](https://hvpandya.com) | MIT (no corpo da skill) | `stop-slop` | Atribuída no frontmatter; licença declarada no corpo. |

Licenças verificadas via API do GitHub em 2026-09-16 (`license.spdx_id = MIT`).

## 2. Sem upstream confirmado (suas)

`agent-memory`, `api-contract-design`, `ask-questions-if-underspecified`, `aur-headless-install`, `bulk-postgres-import`, `cachyos-gaming-setup`, `context7-auto`, `database-ops`, `docker`, `docker-build-local-vps-deploy`, `docker-desktop-wsl-restart`, `git-workflow`, `headless-gui-probe`, `jwt-hs256-node`, `lsp-smoke-test`, `memory-promotion`, `nextjs-standalone-deploy`, `parallel-agent-orchestration`, `phone-e164-normalization`, `playwright-prod-regression`, `simple-feature-flag`, `ssh-vps`, `subagent-routing`, `universal-test-runner`, `vps-agent-dispatch`, `vps-provisioning`, `web-dashboard-automation`, `windows-admin`

A maioria foi promovida da sua própria memória pelo `memory-promotion` (commit `216ccd9`) ou é específica do seu ambiente. **Pendência:** `ask-questions-if-underspecified` e `context7-auto` têm estilo de skill de terceiro e origem não confirmada. Se reconhecer o upstream, adicione `metadata`; enquanto não, seguem como suas.

## 3. Licença

Repositório com `LICENSE` (MIT) na raiz. Todas as skills declaram `license: MIT` no frontmatter. Como todos os upstreams são MIT, o conjunto é compatível — você pode adaptar qualquer uma livremente.

## 4. Ao adotar skill de terceiro daqui pra frente

1. Verifique a licença do upstream (`gh api repos/<owner>/<repo> --jq .license.spdx_id`). Prefira MIT/Apache-2.0/BSD.
2. Copie para `configs/skills/<nome>/` e preencha `metadata.author`/`source` + `adapted`.
3. Adapte o corpo como quiser — fidelidade ao upstream não é necessária.
4. Registre na tabela da seção 1.
5. Rode `envctl run skills`; confirme no `cmdc skills list`.
