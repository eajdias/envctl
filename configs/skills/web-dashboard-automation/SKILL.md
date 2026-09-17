---
name: web-dashboard-automation
description: >-
  Automatizar dashboards/SPAs web com autenticação e proteção CSRF. Use quando precisar extrair dados ou executar ações em sistemas SaaS/dashboards (fazer login, clicar, interceptar o request real, replicar chamadas). Triggers: dashboard, automação web, csrf, X-CSRF-Token, interceptar request, página autenticada, proxy de dashboard, saas.
license: MIT
---

# Automação de Dashboards Web Autenticados (CSRF-safe)

## Quando usar

Qualquer automação contra SPA/dashboard com sessão autenticada — extração de dados, ações (aprovar, exportar), integração sem API pública.

## Escolha da ferramenta

- **Exploratório / interativo / 2FA manual**: MCP `chrome-devtools` (habilite via `/mcp`) — `navigate_page`, `take_snapshot`, `click`, `fill`, `type_text`, `list_network_requests`. O browser abre visível: acompanhe e digite códigos manualmente quando preciso.
- **Determinístico / repetível / regressão**: `playwright-cli` via shell (`open`, `snapshot`, `click e15`, `screenshot`) — token-efficient, sem carregar schemas de MCP no contexto. `--headed` para acompanhar, headless em VPS/sem display. No Linux passe sempre `--browser=chromium` (o default `chrome` procura `/opt/google/chrome/chrome` e falha).

## Hang no Windows

`playwright-cli open`/`attach` trava o shell do agente no Windows (daemon `detached:true` herda o Job Object — issues microsoft/playwright#41530, opencode#24731, ambas `closed as not planned`). Regras:
- Prefira o MCP `chrome-devtools` no Windows sempre que possível.
- Se usar o CLI: rode com timeout explícito (`timeout 60 playwright-cli ...` no bash) e feche a sessão ao fim (`close`/`close-all`); nunca deixe sessão headed aberta sem `close`.
- Headed parado há >1h sem comando não consome nada além do processo parado — `close-all`/`kill-all` limpa.

## Passos

1. **Descobrir o endpoint real**: navegar até a ação e interceptar o request real — no MCP via `list_network_requests` + `get_network_request`; no CLI via `requests` + `request <n>` — captura URL exata, headers e payload (mais confiável que adivinhar rotas).
2. **Extrair o CSRF token**: ler `<meta name="csrf-token">` da página autenticada (padrão Yii2/SPA).
3. **Replicar a chamada** com os mesmos headers (incluindo `X-CSRF-Token` em POST/PUT/PATCH) e payload capturado.
4. Se a sessão expirar, re-autenticar e re-extrair o token antes de repetir.

## Regras

- Requisições MUTÁVEIS SEM o header CSRF → servidor responde 400 com HTML de erro CSRF (não JSON) — sempre enviar o token.
- Seletor de DOM pode mudar; o request capturado é determinístico — preferir replicar a chamada.

## Verificação

- Resposta da chamada replicada = mesma do clique real (status + corpo).
- Erros 400 CSRF ausentes no console; sessão válida durante toda a extração.