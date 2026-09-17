---
name: docker
description: >-
  Docker na máquina local e em VPS. Use para listar/inspecionar/reiniciar containers, ver logs, subir/derrubar stacks, imagens, volumes, redes, compose, build local + transporte de imagem (VPS fraca), restart limpo do Docker Desktop (WSL2) e pull/push no Hub. Triggers: docker, container, containers, imagem, imagens, compose, docker compose, docker desktop, subir container, derrubar container, logs do container, restart container, docker hub, pull, push, docker ps, stack, vps fraca, build local, wsl exec error, backend.sock.
license: MIT
---

# Docker (máquina local + VPS)

## Contexto (detecte o ambiente)

Os comandos são os mesmos; o daemon não:

- **Windows (Docker Desktop + WSL2):** daemon roda na distro `docker-desktop`. Precisa estar no ar: serviço `com.docker.service` (`Start-Service` se parado) ou abrir o Docker Desktop. Paths Windows — `docker run -v "C:/projeto:/app"`. `docker exec -it <c> /bin/sh` funciona direto.
- **Linux (VPS/servidor):** daemon nativo via systemd — `systemctl status docker`, `sudo systemctl start docker`. Paths POSIX.

Valide sempre com `docker version` (client + server).

## Comandos essenciais

- `docker ps -a` (todos, com status); formato: `docker ps -a --format "table {{.Names}}\t{{.Status}}\t{{.Image}}"`
- `docker logs <c> --tail 100` (ou `-f`) — diagnóstico
- `docker restart <c>` / `stop` / `start`
- `docker images`, `docker image prune -f`, `docker system df`
- `docker exec -it <c> <cmd>` — só quando necessário; prefira logs
- `docker inspect <c>` — network, mounts, env
- `docker compose up -d` / `down` / `logs -f` / `ps`
- `docker pull <img>` / `docker push`

Diagnóstico de "serviço caiu": `docker ps -a` → `docker logs` → `docker restart` → re-verificar.

## Build local + transporte de imagem (VPS fraca)

Quando a VPS não aguenta build pesado (npm/tsc, multistage) ou o compose usa `image:` + `build:` juntos:

1. Build local: `docker build -t <app>:<tag> .`
2. Transportar: `docker save <app>:<tag> | gzip > dist/<app>.tar.gz`
3. Na VPS: `docker load -i <app>.tar.gz` (rápido, sem build)
4. `docker compose up -d` — com `image:` + `build:`, o `docker load` prevalece
5. Runtime SEM devDependencies: `npm ci --omit=dev` direto no stage runtime (Docker não apaga camadas)

## Restart limpo do Docker Desktop (WSL2)

`docker version` falha, `Wsl/ExecError`, `backend.sock: no such file or directory`:

1. `taskkill //F //IM "Docker Desktop.exe" //T`
2. `wsl --shutdown`
3. Reabrir Docker Desktop; aguardar ~45s
4. Validar: `docker version` (client + server)

**NUNCA** `wsl --unregister docker-desktop`. Se persistir, checar `com.docker.service`.

## Regras

- Preferir `docker compose` quando houver compose file
- Nunca apagar volumes/containers com dados sem confirmar
- Docker Hub = CLI `docker` (`search`/`pull`/`push`) ou site; não há MCP do Hub
- VPS remotas com Docker: ver skills `ssh-vps` / `vps-provisioning`
