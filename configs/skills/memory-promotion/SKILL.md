---
name: memory-promotion
description: >-
  Promover aprendizados (memórias) a skills reutilizáveis. Use SEMPRE ao gravar uma entrada em lessons.md/patterns.md (memória de projeto ou global) para classificar se o conteúdo é um PROCESSO reutilizável (→ vira skill, entrada removida) ou uma LIÇÃO/anti-padrão pontual (→ permanece na memória). Triggers: memória, lesson, pattern, aprendizado, promover, skill nova, memory-promotion, lição virou skill, classificar aprendizado.
license: MIT
---

# Memory → Skill Promotion

> **Regra raiz**: memórias podem e DEVEM ser promovidas a skills quando possível. Quando um aprendizado vira skill, a entrada na memória é REMOVIDA — sem redundância entre memória e skill.

## 1. Classificação (obrigatória ao gravar memória)

| O conteúdo descreve... | Vai para | Ação |
|---|---|---|
| **PROCESSO reutilizável** multi-passos (workflow com passos, critérios de decisão, checagem que se repete, receita de deploy/auditoria) | **SKILL** | Criar `SKILL.md`, **remover a entrada da memória** |
| **LIÇÃO/anti-padrão** curto (❌ não faça X → ✅ faça Y, porque...) | **MEMÓRIA** | Manter em lessons.md |
| **Fato do ambiente / preferência** (caminhos, ferramentas instaladas, gosto do usuário, bandas de tuning) | **MEMÓRIA** | Manter em patterns.md |
| Dado de DOMÍNIO de um projeto (API do fornecedor, formato de dados do cliente) | **MEMÓRIA DO PROJETO** | Arquivo de memória do projeto (`.opencode/memory/` no OpenCode; `AGENTS.md` no CommandCode) — nunca global |

Teste rápido: se a entrada tem **mais de 2 passos** ou é **aplicável a situações futuras repetidas** → skill. Se responde "o que não fazer / por quê" → memória.

## 2. Criar a skill (se classificar como skill)

1. Local correto:
   - Reutilizável entre projetos → **global**: `~/.config/opencode/skills/<nome>/SKILL.md` (OpenCode) / `~/.commandcode/skills/<nome>/SKILL.md` (CommandCode)
   - Específica do projeto/repo → **projeto**: `.opencode/skills/<nome>/SKILL.md` (OpenCode) / `.commandcode/skills/<nome>/SKILL.md` (CommandCode) (e depois registrar no envctl do projeto, se existir)
2. Nome: `[a-z0-9]+(-[a-z0-9]+)*` (ex.: `deploy-standalone`, `db-migration-audit`).
3. Frontmatter: `name` (obrigatório), `description` com triggers (obrigatório) — escrever descrição acionável ("Use quando... Triggers: ...").
4. Corpo: seções `## Quando usar`, `## Passos` (numerados, com comandos reais), `## Verificação` (como provar que funcionou). Português, conciso, sem "AI speak".
5. **Se a memória for promovida, REMOVER a entrada** do lessons.md/patterns.md (substituir por nada — a skill agora é a fonte). Exceção: guardar NA SKILL uma linha "Contexto: substitui a lição de <data>" se o contexto histórico importar.
6. **Atribuição (obrigatória)**: se a skill foi adotada/derivada de terceiro, preencha `metadata.author`, `metadata.source` e `metadata.adapted`, e declare a licença do upstream no campo `license`. Skill autoral declara `license: MIT` e **não** ganha metadata de terceiro. Registre a linha na tabela de `docs/skills-attribution.md`. Nunca deixe crédito implícito.

## 3. Registrar para provisionamento

- **Global** (skill em `~/.config/opencode/skills/`): rodar `envctl snapshot` (descobre skills novas → copia para `configs/skills/` + `manifests/skills.yaml`). Depois, commit manual e `envctl run skills` provisão em todas as máquinas.
- **Projeto com envctl**: adicionar a skill em `configs/skills/<nome>/` + entrada em `manifests/skills.yaml` (ou `envctl snapshot` no diretório do projeto).

## 4. Validação

- `envctl doctor` → check Skills OK (skill reconhecida).
- Sessão nova do agente → skill aparece na lista (`available_skills`/tool `skill` no OpenCode; `/skills` e `cmdc skills list` no CommandCode — este último valida o frontmatter e acusa skill rejeitada).
- A memória de origem NÃO contém mais a entrada promovida (grep confirma).
- `git status` mostra `configs/skills/<nome>/SKILL.md` novo + entrada em `manifests/skills.yaml`.

## Exemplos de promoção

| Memória (antes) | Skill (depois) |
|---|---|
| "❌ VPS fraca → build Docker local..." | `skill build-docker-local` — workflow completo: build → save → load → compose |
| "❌ Rodar CLI pesada → chamar função direta..." | `skill run-domain-function-directly` — critérios + passos |
| "✅ `bunx` com versão pinada" | **fica memória** (preferência, não processo) |
| "❌ asyncpg str em date → DataError" | **fica memória** (anti-padrão pontual) |