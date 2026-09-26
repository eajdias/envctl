# Docker

## Dockerfile

- Build **multistage**: stage de build com toolchain, imagem final só com runtime.
- Ordene camadas do que muda menos para o que muda mais: cache invalidado é tempo.
- `.dockerignore` com `.git`, `node_modules`, `.worktrees`, artefatos de build.
- `USER` sem root na imagem final; bind mount sobrescreve o usuário, então alinhe os dois.
- `HEALTHCHECK` no container, não só no host: é o compose/orchestrator que decide.

## Compose

- `depends_on` só ordena início; readiness é `healthcheck` + `condition: service_healthy`.
- Named volume para dado, bind mount para código em dev.
- `profiles` para serviço opcional; `restart: unless-stopped` em serviço que deve voltar.

## Operação

- `docker ps -a --format "table {{.Names}}\t{{.Status}}\t{{.Image}}"` antes de agir.
- `docker logs --tail 50 <ctr>` e `docker exec -it <ctr> sh` para diagnóstico.
-.Container parado com saída ≠ 0 quase sempre é a causa; leia o log, não o status.
- Nome de rede/volume explícito quando dois projetos coexistem na mesma máquina.
- Limpe imagem com `docker image prune`; **nunca** `prune -a` sem confirmar que não
  quebra ambiente que você não provisionou.
