---
name: lsp-smoke-test
description: >-
  Smoke test de Language Server Protocol (LSP) servers. Use quando precisar validar que um servidor de linguagem instalado funciona (--version falha silenciosamente em vários) ou antes de registrá-lo no opencode/vscode. Triggers: lsp, language server, smoke test, intelephense, tailwindcss, taplo, --stdio, servidor de linguagem.
license: MIT
compatibility: opencode
---

# Smoke Test de LSP Servers

## Quando usar

Validar um LSP recém-instalado antes de registrar em config (opencode.json/VS Code) ou diagnosticar um que não responde.

## Passos

1. **NÃO usar `--version`** — vários LSPs (intelephense, tailwindcss-language-server) ignoram a flag e falham com "Connection input stream is not set".
2. Rodar com stdin fechado: `<comando> --stdio < /dev/null` (no PowerShell: `cmd /c "<comando> --stdio < NUL"`).
3. Verificar AUSÊNCIA do erro de conexão — sem o erro, o servidor está OK.
4. Casos especiais: `taplo` usa subcomando `taplo lsp`; `csharp-ls`/`gopls` respondem a `--version` normalmente.

## Verificação

- Exit/saída sem "input stream is not set".
- Comando reconhecido (help/version válidos onde aplicável).