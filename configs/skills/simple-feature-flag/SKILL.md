---
name: simple-feature-flag
description: >-
  Implementar feature flags simples e auditáveis em aplicações com banco de dados. Use quando precisar ligar/desligar funcionalidades por usuário operacional via painel, sem re-deploy nem variáveis de ambiente. Triggers: feature flag, flag, toggle, ligar função, desligar função, app_config, painel de config.
license: MIT
compatibility: opencode
---

# Feature Flag Simples e Auditável

## Quando usar

Flags liga/desliga que precisam ser controladas por usuário operacional (via painel/API), não por deploy ou env var.

## Passos

1. Tabela chave-valor: `app_config` (`key`, `value`, `atualizado_em`).
2. API: `GET/PUT /config/:key` protegida por JWT (painel edita a flag).
3. Serviços leem o toggle **no início** de cada ciclo (ex.: cron loga e pula quando off).
4. Sempre registrar `atualizado_em` para auditoria de quem/quando ligou.

## Verificação

- Flag `off` → feature não executa (job loga "skipped").
- Flag `on` via API → próximo ciclo executa sem restart.
- `atualizado_em` preenchido em toda mudança.