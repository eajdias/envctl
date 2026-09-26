# Reestruturação do catálogo de skills — 50 → 12

> Data: 2026-09-26. Escopo: tier global (`~/.config/opencode`, `~/.commandcode`).
> Separado de `spec-agent/2026-09-25-commandcode-parity.md` (que fechou o lado CommandCode).

## Problema

Três problemas medidos, não opinativos:

1. **O catálogo é caro e quase inerte.** Medido com `cmdc -p`: com o default
   `COMMANDCODE_SKILL_CATALOG_CHAR_BUDGET=8000` o runtime cai para **nome + location**,
   sem nenhuma description — e nesse modo a **auto-ativação não acontece** (0 chamadas de
   `activate_skill`). Com budget 30.000 o modelo chamou `activate_skill` 2×. Ou seja: 48
   skills custam ~6,4k tokens/turn e, no default, não compram auto-ativação.
2. **O tier global carrega conhecimento do produto envctl.** 9 linhas em
   `configs/AGENTS.arch.md`, ~17 em `configs/REFERENCE.md` e 3 skills que são documentação
   dos stacks do próprio produto (`linux-performance-tuning` com 8 menções a `envctl run`,
   `vps-provisioning` com 3 em 36 linhas, `cachyos-gaming-setup`). Tudo isso já existe no
   tier certo: `AGENTS.md` e `docs/` do repo.
3. **A aritmética resolve o problema.** Catálogo = `25 + N×391` chars (overhead 142 +
   teto 249, ambos medidos). O default de 8.000 comporta **N ≤ 20**. Com N = 12 as
   descriptions aparecem **sem env var**.

## Regra de decisión (o critério que substitui gosto)

> Uma entrada só merece slot de catálogo se contiver **algo que o modelo não faria
> sozinho**: preferência do usuário, procedimento não-óbvio, ou armadilha já paga.
> "Como funciona Docker/Next/JWT/E.164" o modelo já sabe — isso é referência, ou não é nada.

Escopo, em 4 camadas:

| Camada | Entra | Nunca entra |
|---|---|---|
| AGENTS global (3+3) | Preferências do usuário + fatos da máquina | Como o envctl funciona por dentro |
| SKILL global (catálogo) | Processo generalizado que precisa disparar sozinho | Conhecimento de ferramenta que o modelo já sabe; receita de projeto |
| `references/*.md` | Conhecimento específico sob demanda (stack/ferramenta/domínio) | Regra obrigatória (é AGENTS) |
| Repo do envctl | Manifests, doctor, matriz, stacks, provisioning | — |

## Resultado: 12 entradas

| # | Entrada | Papel |
|---|---|---|
| 1 | `agent-memory` | mecanismo de memória do usuário |
| 2 | `clarify-before-acting` | fusão de `grill-me` + `ask-questions-if-underspecified` + `grilling` |
| 3 | `writing-plans` | convenção `spec-agent/` + método de plano |
| 4 | `systematic-debugging` | lei de 4 fases, absorvendo `variant-analysis` como fase 5 |
| 5 | `test-driven-development` | ciclo + comandos reais |
| 6 | `git-workflow` | branches/commits/PR + convenção de worktree |
| 7 | `subagent-routing` | inline-first + critério de despacho |
| 8 | `subagent-supervision` | vocabulary real por runtime (pago caro) |
| 9 | `task-hang-watchdog` | shell em background por runtime |
| 10 | `technical-research` | método: fonte/data/fato-inferência-desconhecido |
| 11 | `code-playbooks` | **gate**: índice → `references/*.md` |
| 12 | `envctl` | **operator**: bootstrap, `doctor`, `run *`; zero interno de produto |

### `code-playbooks` (gate) — referências

`go` · `python` · `node` · `nextjs` · `sql` · `testing` · `frontend` · `web-automation` ·
`mcp-api` · `backend-patterns` · `infra` · `docs-sync` · `skill-authoring`

Regra dura no corpo: **não responder sobre um tema sem ler o arquivo correspondente**.

### `envctl` (operator) — escopo

O que fica: o que é, bootstrap por OS, `doctor`/`doctor --fix`, `run shell|skills|lsp|all|cleanup`,
`opencode`/`commandcode`, segurança do `snapshot` (sync reverso, nunca em VPS), idempotência.
O que sai: manifests, doctor, matriz, stacks — pointer para o repo.

Frase de fronteira com `references/infra.md`: *infra = operar o que já existe · envctl = criar e auditar a máquina.*

## Decisões tomadas

- `docs-sync` → **regra do AGENTS** + `references/docs-sync.md` (sincronizar doc é obrigação
  constante, não descoberta; a norma já existe no AGENTS do repo, falta a rotina de auditoria).
- Infra → **entra** no gate como `references/infra.md` (técnica stack-independente), separada do operator.
- `context7-auto`, `stop-slop`, `handoff`, `receiving-code-review` → **regras do AGENTS** (comportamento obrigatório não gasta slot).
- `vps-provisioning`, `cachyos-gaming-setup`, `linux-performance-tuning` → conhecimento migra para `docs/` do repo; o global vira 1 linha por assunto.
- Slash command não é requisito do usuário → sem arquivo de arquivo morto por UX.
- **macOS: qualquer menção e uso removidos** (README, `bootstrap.sh`, `DistroDarwin`, conjunto fechado do lint). O lint passa a rejeitar `os: darwin`.

## Blast radius (tudo ou o doctor quebra)

- `internal/infra/embedded/manifest_repo_test.go`: `expectedSkills = 50` → 12
- `internal/usecase/skill_content_contract_test.go`: allowlist de `when_to_use` → 5 skills novas
- `manifests/skills.yaml`: 50 → 12 entradas; diretórios em `configs/skills/` Reduced para 12
- `configs/AGENTS*.md` (3) e `configs/commandcode/AGENTS*.md` (3): remove envctl, adds regras
- `configs/REFERENCE.md`: fica "como o agente funciona"; sai "como o envctl provisiona"
- `configs/SKILL-INDEX.md` + `configs/commandcode/SKILL-INDEX.md`: reescritos
- `README.md`: contagem por OS + remoção da seção macOS
- `docs/skills.md`, `docs/os-and-agent-matrix.md` (§1/§2/§3), `docs/doctor-and-idempotency.md`, `docs/skills-attribution.md`
- `CHANGELOG.md`, `AGENTS.md` (repo), memória (projeto)
- Novo check de doctor: projeção do catálogo de skills (`25 + N×391`) vs. budget, para nunca mais
  cair em names-only em silêncio.

## Fases

1. **A** — remover macOS (README, `bootstrap.sh`, `DistroDarwin`, lint) + teste
2. **B** — construir a árvore nova: 10 de processo (com as fusões), `code-playbooks` + 13 referências, `envctl`
3. **C** — manifest + constantes de teste + teste de contrato → `go test` verde
4. **D** — AGENTS globais (6) + split do REFERENCE
5. **E** — docs: README, skills.md, matriz, doctor doc, attribution, 2 SKILL-INDEX, CHANGELOG
6. **F** — check de doctor do orçamento de catálogo
7. **G** — verificação completa, reprovisionamento, `doctor` 0 WARN, commit único

## Definition of Done

- [x] Zero suporte ou menção **operacional** a plataforma não suportada (o registro da decisão fica nesta spec); `bootstrap.sh` só Linux, `DistroDarwin` removido, lint rejeita o token, branches darwin do doctor removidos, `docs/guides/macos.md` apagado, README/ADR/roadmap/architecture/principles/CONTRIBUTING limpos
- [x] `configs/skills/` com exatamente 12 diretórios, todos com frontmatter válido e `name` == diretório
- [x] `go test ./...` verde com `expectedSkills = 12`
- [x] AGENTS global sem assunto envctl exceto a linha de operação, com as regras novas (docs em dia, code-playbooks, ambiguity em `clarify-before-acting`, e as 4 regras de conduta nos manifests do CommandCode)
- [x] `code-playbooks` com índice + 14 referências, cada uma com o que **não** fazer além do como
- [x] `envctl` sem interno de produto: bootstrap, `doctor`, `run *`, segurança do `snapshot`, pointer pro repo
- [x] Docs sincronizados: README (12 skills, sem seções mortas), `docs/skills.md` reescrito, matriz §1/§2, `doctor-and-idempotency.md`, `skills-attribution.md`, ADR 0001, guides, REFERENCE (split) e os dois `SKILL-INDEX.md`
- [x] Check de doctor do orçamento de catálogo verde e testado (`TestSkillCatalogBudgetFindings`)
- [x] `envctl doctor` 208 checks, 0 WARN, 0 ERROR; 12 skills deployadas nos dois agentes
- [x] Catálogo projetado em ~4.717 chars — abaixo do default de 8.000, ou seja, as descriptions chegam ao modelo sem env var
- [x] Commit único, sem push

## Estado final (2026-09-26)

- Branch `feat/commandcode-parity`, 1 commit pendente com toda a reestruturação.
- Catálogo do CommandCode projetado: `25 + 12×391 = 4.717` chars. O default (8.000) passa a bastar, então a **auto-ativação volta a funcionar sem variável de ambiente** — era o objetivo mensurável da mudança.
- Referências criadas: `go`, `python`, `node`, `nextjs`, `sql`, `testing`, `docker`, `infra`, `frontend`, `web-automation`, `mcp-api`, `backend-patterns`, `skill-authoring`, `docs-sync`.
