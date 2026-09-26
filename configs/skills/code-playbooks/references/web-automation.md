# Automação web

## Duas ferramentas, dois usos

| | MCP de browser | CLI de browser (`pw`) |
|---|---|---|
| Uso | sessão **interativa** com login/2FA | automação **determinística** em shell |
| Estado | precisa de GUI e atenção humana | headless, repetível, barato em token |
| Custo | alto, trava a sessão | baixo, cabe em script |

Regra: se dá para fazer determinístico no shell, não gaste sessão interativa.

## Sessão autenticada (dashboard, SaaS)

1. Faça login **no browser** uma vez; não tente resolver CSRF por código às cegas.
2. Inspecione a request real que a SPA faz (method, URL, headers, cookies) e replique.
3. Token CSRF e cookie de sessão vivem no storage do browser — extraia dali, não invente.
4. Streaming de resposta: o `fetch` da SPA é quem resolve; copiar o formato do payload
   sem o envelope quebra silenciosamente.

## E2E em produção

- **Nunca** mutar dado real: use conta de teste, IDs de teste e `dry-run` quando existir.
- Valide o que o deploy poderia quebrar: rota crítica, login, fluxo de compra/envio.
- Console error e requisição 4xx/5xx na navegação são falha, mesmo com "visual ok".
- Remova dados de teste ao final e registre qual conta foi usada.
