# Next.js

## Docker com output standalone

`output: "standalone"` gera `.next/standalone` com `node_modules` mínimo. Quatro
pegadinhas que quebram build ou runtime:

1. `public/` **não** é copiado pelo standalone — copie no `Dockerfile`.
2. Variáveis `NEXT_PUBLIC_*` são inlinadas no **build**, não em runtime: passar em build.
3. `outputFileTracingRoot` precisa apontar para a raiz do projeto em monorepo.
4. Imagem final precisa do user sem root e de `HOSTNAME=0.0.0.0` para o healthcheck.

## Migração de major

- Major nova quebra build: leia o changelog, rode o codemod oficial e **só então** atualize
  dependências. Não pule major pulando o build intermediário.
- Verifique: `next.config` (opções renomeadas/removidas), `app/` vs `pages/`, async
  `params`/`searchParams`, `metadata` por rota.

## Gate

```bash
next build && next lint        # ou o script `lint` do package.json
docker build -t app . && docker run --rm -p 3000:3000 app
```

Healthcheck dentro do container, não só no host.
