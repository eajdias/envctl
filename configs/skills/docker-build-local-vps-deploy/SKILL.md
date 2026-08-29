---
name: docker-build-local-vps-deploy
description: >-
  Deploy de aplicações Docker em VPS fraca — build local, transporte por imagem e load no destino. Use quando a VPS não aguenta build pesado (tsc/npm), deploy demora ou trava, ou precisa atualizar serviços sem build na máquina remota. Triggers: vps, deploy docker, docker load, build local, imagem tar.gz, vps fraca, transporte imagem.
license: MIT
compatibility: opencode
---

# Deploy Docker em VPS Fraca (build local + save/load)

## Quando usar

VPS com pouca CPU/RAM onde o build (npm/tsc, multistage) demora ou trava. Sempre que o compose usar `image:` + `build:` juntos.

## Passos

1. **Build local**: `docker build -t <app>:<tag> .` na máquina de desenvolvimento.
2. **Transportar a imagem**: `docker save <app>:<tag> | gzip > dist/<app>.tar.gz`.
3. **Na VPS**: `docker load -i <app>.tar.gz` (rápido, sem build).
4. `docker compose up -d` — com `image:` + `build:` no compose, o `docker load` prevalece (não builda).
5. **Runtime SEM devDependencies**: rodar `npm ci --omit=dev` DIRETO no stage runtime (nunca confiar em `npm prune --omit=dev` — Docker não apaga camadas, a imagem mantém o tamanho).

## Verificação

- `docker images` na VPS mostra a tag carregada; `docker compose up -d` não dispara build.
- Imagem final sem node_modules de dev: `docker run --rm --entrypoint sh <img> -c 'ls node_modules | head'` — sem pacotes de dev.
- Aplicação responde após `up -d` sem logs de erro de módulo ausente.