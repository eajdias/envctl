---
name: lsp-smoke-test
description: >-
  Smoke test de Language Server Protocol (LSP) servers. Use quando precisar validar que um servidor de linguagem instalado funciona (--version falha silenciosamente em vários) ou antes de registrá-lo no opencode/vscode. Triggers: lsp, language server, smoke test, intelephense, tailwindcss, taplo, --stdio, servidor de linguagem.
license: MIT
---

# Smoke Test de LSP Servers

## Quando usar

Validar um LSP recém-instalado antes de registrá-lo na config do agente/editor ou diagnosticar um que não responde. No OpenCode o registro é o bloco `lsp` do `opencode.json`; no CommandCode **não há registro** — ele usa o LSP do IDE conectado (`/ide` + tool `get_diagnostics`), então aqui o smoke test serve só para validar o binário.

## Passos

1. **NÃO usar `--version`** — vários LSPs (intelephense, tailwindcss-language-server) ignoram a flag e falham com "Connection input stream is not set".
2. Rodar com stdin fechado: `<comando> --stdio < /dev/null` (no PowerShell: `cmd /c "<comando> --stdio < NUL"`).
3. Verificar AUSÊNCIA do erro de conexão — sem o erro, o servidor está OK.
4. Casos especiais: `taplo` usa subcomando `taplo lsp stdio` (o `taplo lsp` sozinho foi removido no taplo 0.10 e sai com erro); `csharp-ls`/`gopls` respondem a `--version` normalmente.

## Verificação

- Exit/saída sem "input stream is not set".
- Comando reconhecido (help/version válidos onde aplicável).