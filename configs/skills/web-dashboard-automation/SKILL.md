---
name: web-dashboard-automation
description: >-
  Automatizar dashboards/SPAs web com autenticação e proteção CSRF. Use quando precisar extrair dados ou executar ações em sistemas SaaS/dashboards (fazer login, clicar, interceptar o request real, replicar chamadas). Triggers: dashboard, automação web, csrf, X-CSRF-Token, interceptar request, página autenticada, proxy de dashboard, saas.
license: MIT
compatibility: opencode
---

# Automação de Dashboards Web Autenticados (CSRF-safe)

## Quando usar

Qualquer automação contra SPA/dashboard com sessão autenticada — extração de dados, ações (aprovar, exportar), integração sem API pública.

## Passos

1. **Descobrir o endpoint real**: com Playwright, clicar na ação manualmente e interceptar via `page.on('request')` — captura URL exata, headers e payload (mais confiável que adivinhar rotas).
2. **Extrair o CSRF token**: ler `<meta name="csrf-token">` da página autenticada (padrão Yii2/SPA).
3. **Replicar a chamada** com os mesmos headers (incluindo `X-CSRF-Token` em POST/PUT/PATCH) e payload capturado.
4. Se a sessão expirar, re-autenticar e re-extrair o token antes de repetir.

## Regras

- Requisições MUTÁVEIS SEM o header CSRF → servidor responde 400 com HTML de erro CSRF (não JSON) — sempre enviar o token.
- Seletor de DOM pode mudar; o request capturado é determinístico — preferir replicar a chamada.

## Verificação

- Resposta da chamada replicada = mesma do clique real (status + corpo).
- Erros 400 CSRF ausentes no console; sessão válida durante toda a extração.