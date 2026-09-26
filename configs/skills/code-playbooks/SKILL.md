---
name: code-playbooks
description: >-
  Conhecimento por stack e ferramenta (Go, Python, Node/Next, SQL, testes, Docker, infra,
  frontend, MCP/API, docs): índice e leia o arquivo do tema.
when_to_use: >-
  Tarefa que depende de convenção específica de alguma dessas stacks, ferramenta ou domínio.
license: MIT
---


# Code Playbooks

Catálogo-ponte: **esta skill é o índice**. O conhecimento está nos arquivos de
`references/`, um por tema. Carregue o índice, escolha o tema, leia o arquivo.

> **Regra dura:** NÃO responda sobre um tema sem ler o arquivo correspondente.
> Responder pela tabela é responder errado — a tabela não tem o conteúdo.

## Índice

| Tema | Arquivo | Cobre |
|---|---|---|
| Go | `references/go.md` | idiomatismo, concorrência, portabilidade, test/vet |
| Python | `references/python.md` | versão, ambiente, packaging, async, tipos |
| Node / TypeScript | `references/node.md` | módulos, tsc/eslint, ESM, dependências |
| Next.js | `references/nextjs.md` | output standalone, Docker, migração de versão |
| SQL / Bancos | `references/sql.md` | dialeto, EXPLAIN, índice, migração, import em massa |
| Testes | `references/testing.md` | comandos por ecossistema, cobertura, o que prova o quê |
| Docker | `references/docker.md` | build, camadas, volumes, healthcheck, recovery |
| SSH / Infra | `references/infra.md` | chaves, multiplexer, serviços, Tailscale, Syncthing |
| Frontend | `references/frontend.md` | HTML semântico, a11y/WCAG, CSS, responsivo |
| Automação web | `references/web-automation.md` | sessão autenticada, CSRF, browser MCP vs CLI, E2E em produção |
| MCP / API | `references/mcp-api.md` | contrato de tool, inputSchema, OpenAPI/GraphQL/gRPC |
| Padrões de backend | `references/backend-patterns.md` | feature flag, auth, idempotência, paginação |
| Autoria de skill | `references/skill-authoring.md` | escrever, adaptar, generalizar, minerar skills |
| Auditoria de docs | `references/docs-sync.md` | achar drift entre código, manifestos e docs |

## fronteiras

- **`code-playbooks`**: *como se faz* neste domínio.
- **`git-workflow`**: versão, branches, PR e worktree (é processo, não domínio).
- **`envctl`**: *criar e auditar a máquina* (é ferramenta, não domínio).
- **`infra.md`**: operar o que já existe. Provisionar máquina é do `envctl`.

## Verificação

- [ ] Li o arquivo do tema antes de responder.
- [ ] Não afirmei convenção que não está no arquivo.
