---
name: docs-sync
description: >-
  Auditar ou atualizar a documentação contra a implementação real: achar docs faltando, incorretas ou desatualizadas e propor correções pontuais. Use automaticamente ao auditar cobertura de docs, sincronizar docs com código, checar README/CHANGELOG/manifests após mudança. Triggers: docs desatualizada, sincronizar docs, doc coverage, doc drift, doc faltando, changelog desatualizado, readme desatualizado, auditar docs.
license: MIT
metadata:
  author: openai/openai-agents-python (ideia)
  source: https://github.com/openai/openai-agents-python
  adapted: envctl — workflow escrito do zero para a estrutura real (README/CHANGELOG/docs/manifests/configs/AGENTS); sem mkdocs, sem traduções, sem MCP openai-knowledge; audit-only por default + Apêndice A (docstrings TS/PY/GO)
---

# Docs Sync

Achar gaps e imprecisões comparando o comportamento real (código, manifests, configs) com a documentação, e propor correções pontuais.

**Regra de autorização:** audit/proposta-only → só reporta, não edita. Edição só com pedido explícito do usuário ou plano já aprovado. Em dúvida, reporte e pare.

## Workflow

### 1. Confirme escopo e branch

- Branch atual vs. default (geralmente `main`). Prefira analisar a branch atual para acompanhar mudanças em voo.
- Limite o inventário ao tópico pedido ou ao diff da branch. Inventário completo só em auditoria explicitamente abrangente.
- Evite trocar de branch se houver mudanças locais — prefira inspeção read-only (`git show main:<path>`).

### 2. Monte o inventário de comportamento (limitado ao escopo)

- Foque no user-facing: exports públicos, comandos CLI, opções de config, variáveis de ambiente, valores default, comportamentos documentados.
- Evidência por item: caminho do arquivo + símbolo/setting.
- Buscas dirigidas: `rg "Settings"`, `rg "Config"`, `rg "os.Getenv"`, nomes de comandos/flags.

### 3. Passe doc-first: revise as páginas existentes

Caminhe por cada página relevante (`README.md`, `CHANGELOG.md`, `docs/*.md`, `CONTRIBUTING.md`) e identifique menções faltando: opções suportadas importantes (flags opt-in, env vars), pontos de customização, features novas. Proponha adições onde o usuário razoavelmente esperaria encontrá-las.

### 4. Passe code-first: mapeie comportamento → docs

- Revise a arquitetura de informação dos docs e determine a melhor página/seção para cada item, seguindo os padrões existentes.
- Identifique o que não tem página ou tem página sem conteúdo correspondente.
- Quando o gap está numa referência gerada a partir de comentários de código, prefira atualizar o comentário no código em vez de editar a página gerada.

### 5. Classifique gaps e imprecisões

- **Missing**: existe no código/manifests, ausente nos docs.
- **Incorrect/outdated**: nomes, defaults ou comportamentos divergem.
- **Structural** (opcional): páginas sobrecarregadas, overviews faltando, tópicos mal agrupados.

### 6. Reporte (e só edite se autorizado)

- Audit-only: evidência + local sugerido + edições propostas, e pare.
- Com autorização: faça as edições, mantendo estilo e navegação existentes.

## Formato de saída

```markdown
Docs Sync Report

- Doc-first findings
  - Página + conteúdo faltando -> evidência + ponto de inserção sugerido
- Code-first gaps
  - Comportamento + evidência -> página/seção sugerida (ou página faltando)
- Incorrect ou outdated
  - Arquivo + problema + info correta + evidência
- Structural (opcional)
  - Mudança proposta + motivo
- Proposed edits
  - Arquivo -> resumo da mudança
```

## Apêndice A — docstrings por linguagem (TS/PY/GO)

Quando o gap pede documentar API/função, siga o padrão da linguagem do arquivo:

**TS (TSDoc):**

```ts
/**
 * Normaliza telefone BR para E.164.
 * @param raw - Telefone em qualquer formato ("(11) 98765-4321", "5511987654321")
 * @returns Dígitos E.164 sem "+" ("5511987654321")
 * @throws {TypeError} Se `raw` não é string.
 */
```

**Python (docstring, Google style):**

```python
def normalize_br_phone(raw: str) -> str:
    """Normaliza telefone BR para E.164.

    Args:
        raw: Telefone em qualquer formato ("(11) 98765-4321").

    Returns:
        Dígitos E.164 sem "+" ("5511987654321").

    Raises:
        ValueError: Se `raw` não é string.
    """
```

**Go (godoc — começa com o nome, sem @param/@returns):**

```go
// NormalizeBRPhone normaliza telefone BR para E.164 (dígitos sem "+").
// Retorna erro se raw não contém DDD + número válidos; malformados são
// descartados com log, nunca com panic.
```

Regra: documente contrato (params, retorno, erro), não implementação. Sem docstring em código privado trivial de uma linha.
