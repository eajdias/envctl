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
- **Determinístico / repetível / regressão**: `pw` (wrapper versionado em `~/.local/bin`, provisionado pelo envctl) — mesma CLI por baixo (`open`, `snapshot`, `click e15`, `screenshot`), token-efficient, sem travar o shell do agente. `--headed` para acompanhar, headless em VPS/sem display. No Linux passe sempre `--browser=chromium` (o default `chrome` procura `/opt/google/chrome/chrome` e falha).

## Por que `pw` e não `playwright-cli` direto

`playwright-cli open`/`attach` executado direto no shell do agente trava no Windows: o CLI spawna um daemon com `detached:true` (`session.js` → `spawn(process.execPath, args, {detached:true, stdio:["ignore","pipe",err]})`), que herda o Job Object — o shell espera a árvore inteira e nunca retorna, mesmo com output correto (issues microsoft/playwright#41530, opencode#24731, ambas `closed as not planned`). O `pw` (`~/.local/bin/pw.cjs`, node puro, sem timeout) faz o spawn DETACHED + unref ele mesmo e espera pela sessão via `list` — verificado empiricamente: `open https://example.com --browser=chromium` via `pw` retornou em ~2s dentro do `opencode run`, enquanto o mesmo comando cru trava. Skill `/playwright-cli` usa esse caminho. Nunca invoque `playwright-cli open`/`attach` cru no shell do agente no Windows — use `pw`.

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