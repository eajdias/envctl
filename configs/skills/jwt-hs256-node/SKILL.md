---
name: jwt-hs256-node
description: Implementar JWT HS256 sem dependências externas em Node.js. Use quando precisar de autenticação interna leve (tokens de sessão/API) sem instalar jsonwebtoken ou similar. Triggers: jwt, token, hs256, auth, autenticação, createHmac, timingSafeEqual, assinatura.
license: MIT
compatibility: opencode
---

# JWT HS256 em Node.js sem Dependências

## Quando usar

Auth interna leve (tokens de curta duração, service-to-service) onde instalar `jsonwebtoken` é overkill.

## Passos

1. Header + payload em JSON, base64url (`Buffer.from(x).toString('base64url')`).
2. Assinatura: `createHmac('sha256', secret)` sobre `header.payload`, base64url.
3. Token = `header.payload.signature`.
4. **Verificação timing-safe**: `timingSafeEqual` exige buffers de TAMANHOS IGUAIS → comparar `createHash('sha256')` dos dois lados (hash fixa o tamanho) antes do `timingSafeEqual`.
5. Validação: expiração (`exp`) no payload, assinatura confere, algoritmo forçado HS256 (nunca `none`).

## Verificação

- Token gerado e verificado em ciclo (assinatura confere).
- Token adulterado (payload editado) → verificação falha.
- Comparação timing-safe sem exceção de tamanho.