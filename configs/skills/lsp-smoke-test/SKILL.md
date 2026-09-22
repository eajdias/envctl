---
name: lsp-smoke-test
description: >-
  Smoke test de Language Server Protocol (LSP) servers. Use quando precisar validar que um servidor de linguagem instalado funciona (--version falha silenciosamente em vários) ou antes de registrá-lo no opencode/vscode. Triggers: lsp, language server, smoke test, intelephense, tailwindcss, taplo, --stdio, servidor de linguagem.
license: MIT
---

# Smoke Test de LSP Servers

## Quando usar

Validar um LSP recém-instalado antes de registrá-lo na config do editor ou diagnosticar um que não responde. Registro por runtime: VS Code/Cursor via extensão; OpenCode v2 **sem registro** — o bloco `lsp` foi removido do `opencode.json` em 2026-09-22 (runtime aceita e ignora; diagnósticos do agente via lint/typecheck) — então aqui o smoke test serve só para validar o binário provisionado (`run lsp` + `doctor`). Template per-project guardado em `patterns.md` para quando o runtime voltar. No CommandCode **não há registro** — ele usa o LSP do IDE conectado (`/ide` + tool `get_diagnostics`), então aqui o smoke test serve só para validar o binário.

## Passos

1. **NÃO usar `--version`** — vários LSPs (intelephense, tailwindcss-language-server) ignoram a flag e falham com "Connection input stream is not set".
2. Rodar com stdin fechado: `<comando> --stdio < /dev/null` (no PowerShell: `cmd /c "<comando> --stdio < NUL"`).
3. Verificar AUSÊNCIA do erro de conexão — sem o erro, o servidor está OK.
4. Casos especiais: `taplo` usa subcomando `taplo lsp stdio` (o `taplo lsp` sozinho foi removido no taplo 0.10 e sai com erro); `csharp-ls`/`gopls` respondem a `--version` normalmente.

## Verificação

- Exit/saída sem "input stream is not set".
- Comando reconhecido (help/version válidos onde aplicável).