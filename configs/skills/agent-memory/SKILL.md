---
name: agent-memory
description: Memória persistente de lições e padrões para o agente. Use para evitar repetir erros passados e aplicar preferências que já funcionaram (estilo "Taste" do Command Code). Aciona ao: iniciar tarefa (ler memória antes), cometer/descobrir um erro (gravar lição), ser corrigido pelo usuário (gravar), descobrir um padrão que funciona (gravar), ou quando algo já falhou antes ("lição", "memória", "aprendizado", "não repita", "isso já deu errado", "já tentamos isso").
---

# Agent Memory — Lessons & Working Patterns

Memória persistente do agente: arquivos markdown que devem ser **consultados no
início de cada tarefa** e **atualizados sempre que o agente aprende algo** — para
nunca repetir erros e sempre aplicar preferências que já funcionaram.

## Localização

- **Global:** `~/.config/opencode/memory/lessons.md` e `~/.config/opencode/memory/patterns.md`
- **Projeto:** `.opencode/memory/lessons.md` e `.opencode/memory/patterns.md` (versionado, revisável em PR)
- Se um arquivo de projeto não existir, crie-o com o template da seção "Formato".

## Fluxo obrigatório (LOAD → ACT → SAVE → REFLECT)

### 1. LOAD — antes de iniciar QUALQUER tarefa
Leia, nesta ordem: projeto `.opencode/memory/lessons.md`, projeto
`.opencode/memory/patterns.md`, global `~/.config/opencode/memory/lessons.md`,
global `~/.config/opencode/memory/patterns.md`.

Trate cada lição como **restrição ativa** da tarefa. Se um erro registrado estiver
prestes a se repetir, PARE e refaça conforme a lição.

### 2. ACT — trabalhe
Aplique os patterns que funcionam e evite os erros registrados.

### 3. SAVE — grave sempre que aprender
Gatilhos:
- Você cometeu um erro (mesmo que o usuário corrija ou você perceba sozinho)
- O usuário rejeitou/repetiu uma instrução → vire lição
- Um comando/abordagem funcionou e é reutilizável → vire pattern
- Uma preferência explícita do usuário (ex.: "sempre uso pnpm") → vire pattern

Formato acionável:
- **Lições:** `❌ Não faça X → ✅ Faça Y (porque Z)` + data + contexto
- **Patterns:** `✅ Quando <situação>, faça <o que funciona>` + fonte/data

Adicione ao final da seção correta. Se já existir lição idêntica, **atualize em
vez de duplicar** (renove a data).

### 4. REFLECT — ao finalizar a sessão/tarefa
- Revise os arquivos e pode duplicados/obsoletos (mantenha enxuto).
- Garanta que as lições estão no formato acionável ❌→✅.
- Projeto: deixe `.opencode/memory/*.md` commitável (o usuário revisa em PR).

## Formato (template de arquivo)

```markdown
# Agent Memory

> Consulte antes de cada tarefa; atualize a cada correção/aprendizado.
> Formato: ❌ <não faça> → ✅ <faça> (porque). Entradas curtas e acionáveis.

## Erros / Lições (não repetir)
<!-- - 2026-08-20 ❌ ... → ✅ ... (porque ...) -->

## Padrões / Preferências (o que funciona)
<!-- - 2026-08-20 ✅ Quando <situação>, faça <o que funciona> -->
```

## Regras

- NUNCA apague lições sem motivo claro (poda apenas no REFLECT).
- Não grave segredos/credenciais na memória — só referências genéricas.
- Memória é contexto, não documentação — mantenha curta e acionável.
- Prefira PT-BR (idioma do usuário) nas entradas.
