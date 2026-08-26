---
name: nextjs-standalone-deploy
description: Deploy de Next.js com output standalone e migração para v16. Use quando dockerizar Next.js (multi-stage, ARG NEXT_PUBLIC, public/), atualizar de v15 para v16, ou debugar quebras de build/standalone. Triggers: next.js, nextjs, standalone, next 16, next15, NEXT_PUBLIC, dockerfile next, migration next, output standalone.
---

# Deploy Next.js Standalone + Migração v16

## Quando usar

Dockerizar app Next.js com `output: 'standalone'` ou migrar de v15 para v16.

## Deploy standalone (Dockerfile multi-stage)

1. `output: 'standalone'` no next.config.
2. Dockerfile: stage deps → build com `ARG NEXT_PUBLIC_API_URL` + `ENV NEXT_PUBLIC_API_URL` ANTES do `npm run build` (é inlined no BUILD, não lido em runtime).
3. Stage runtime: copiar `.next/standalone` + `.next/static` + **`public/`** (`COPY --from=builder /app/public ./public` — standalone NÃO inclui `public/` por padrão → 404 de assets).
4. `CMD node apps/web/server.js`; expor `PORT`/`HOSTNAME` via env.
5. Validar o chunk embutido: `docker run --rm --entrypoint sh <img> -c 'grep -roh "<dominio-api>" /app/.next/'`.

## Quebras do Next.js 16 (vs 15)

1. `middleware.ts` → `proxy.ts` (export `proxy`, runtime nodejs).
2. Route group `(x)/page.tsx` NÃO cria segmento de URL — colide com a raiz; usar pasta real.
3. Turbopack NÃO resolve imports NodeNext `.js`→`.ts` de pacotes transpilados → `experimental.extensionAlias: { '.js': ['.ts','.tsx','.js','.jsx'] }` + `next dev/build --webpack`.
4. Campo `eslint` removido do NextConfig (build não roda lint).
5. `NEXT_PUBLIC_*` inlined no BUILD — mudar default exige rebuild.

## Verificação

- 404 de assets ausente (public/ no runtime).
- API apontada para o domínio correto (grep no chunk).
- Build e runtime sem erros após migração; páginas da raiz sem colisão de rota.