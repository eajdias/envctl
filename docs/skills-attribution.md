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
| [obra/superpowers](https://github.com/obra/superpowers) | MIT | `receiving-code-review`, `systematic-debugging`, `verification-before-completion` | Metodologia e parte do corpo preservadas; descrições, exemplos e integração local foram adaptados ao envctl. |
| [obra/superpowers](https://github.com/obra/superpowers) | MIT | `test-driven-development` | **Ideia** (lei de ferro + red-green-refactor) adaptada — ciclo e exemplos **escritos do zero** para TS/Node (vitest), Python (pytest) e Go (go test), com wiring para `universal-test-runner` + `verification-before-completion`. |
| [openai/openai-agents-python](https://github.com/openai/openai-agents-python) | MIT | `docs-sync` | **Ideia** (doc-first/code-first, missing/incorrect/structural) adaptada — workflow **escrito do zero** para a estrutura real (README/CHANGELOG/docs/manifests/configs), audit-only por default + Apêndice A (docstrings TSDoc/Google-style/godoc). |
| [obra/superpowers](https://github.com/obra/superpowers) | MIT | `writing-plans` | Ideia e estrutura originalmente preservadas; o corpo foi adaptado ao fluxo local, com riscos, unknowns, rollback, Definition of Done e `spec-agent/`. |
| [mattpocock/skills](https://github.com/mattpocock/skills) | MIT | `grill-me`, `grilling`, `handoff` | `handoff`: adaptado com schema explícito de estado, redaction e referências a artefatos. `grilling`: o núcleo do *design tree* foi preservado e recebeu rodadas, fronteira de evidência e critério de fim. `grill-me`: reescrito — o upstream são 2 linhas mandando invocar `/grilling`; a rubrica 0–100, os limiares e a cláusula de risco são nossos. |
| [hqhq1025/skill-optimizer](https://github.com/hqhq1025/skill-optimizer) | MIT | `skill-miner`, `skill-personalizer`, `skill-generalizer` | Corpo **idêntico** — só a `description`/triggers em PT-BR. Mantidos `agents/openai.yaml`, `scripts/scan_sessions.py` + `references/` (skill-miner) e `references/` (personalizer/generalizer). |
| Hardik Pandya — [hvpandya.com](https://hvpandya.com) | MIT (no corpo da skill) | `stop-slop` | Atribuída no frontmatter; licença declarada no corpo. |

Licenças verificadas via API do GitHub em 2026-09-16 (`license.spdx_id = MIT`).

## 2. Sem upstream confirmado (suas)

`agent-memory`, `api-contract-design`, `ask-questions-if-underspecified`, `aur-headless-install`, `bulk-postgres-import`, `cachyos-gaming-setup`, `context7-auto`, `database-ops`, `docker`, `frontend-markup`, `git-workflow`, `go-development`, `headless-gui-probe`, `jwt-hs256-node`, `linux-performance-tuning`, `lsp-smoke-test`, `mcp-tool-design`, `memory-promotion`, `nextjs-standalone-deploy`, `phone-e164-normalization`, `playwright-prod-regression`, `python-development`, `simple-feature-flag`, `ssh-vps`, `subagent-routing`, `subagent-supervision`, `syncthing-ops`, `tailscale`, `task-hang-watchdog`, `technical-research`, `universal-test-runner`, `variant-analysis`, `vps-agent-dispatch`, `vps-provisioning`, `web-dashboard-automation`, `windows-admin`, `windows-debloat`

A maioria foi promovida da sua própria memória pelo `memory-promotion` (commit `216ccd9`) ou é específica do seu ambiente. **Pendência:** `ask-questions-if-underspecified` e `context7-auto` têm estilo de skill de terceiro e origem não confirmada. Se reconhecer o upstream, adicione `metadata`; enquanto não, seguem como suas.

## 3. Licença

Repositório com `LICENSE` (MIT) na raiz. Todas as skills declaram `license: MIT` no frontmatter. Como todos os upstreams são MIT, o conjunto é compatível — você pode adaptar qualquer uma livremente.

## 4. Ao adotar skill de terceiro daqui pra frente

1. Verifique a licença do upstream (`gh api repos/<owner>/<repo> --jq .license.spdx_id`). Prefira MIT/Apache-2.0/BSD.
2. Copie para `configs/skills/<nome>/` e preencha `metadata.author`/`source` + `adapted`.
3. Adapte o corpo como quiser — fidelidade ao upstream não é necessária.
4. Registre na tabela da seção 1.
5. Rode `envctl run skills`; confirme no `cmdc skills list`.
