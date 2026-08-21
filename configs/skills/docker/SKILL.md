---
name: docker
description: Gerenciamento de Docker na máquina local. Use quando o usuário pedir para listar/inspecionar/reiniciar containers, ver logs, subir/derrubar stacks, imagens, volumes, redes, docker compose, ou operar o Docker Hub. Triggers: docker, container, containers, imagem, imagens, compose, docker compose, docker desktop, subir container, derrubar container, logs do container, restart container, docker hub, pull, push, docker ps, stack.
---

# Docker na máquina local (Windows + Docker Desktop)

## Contexto

- Docker Desktop 4.87.0 instalado via winget. Binário: `C:\Program Files\Docker\Docker\resources\bin\docker.exe` (no PATH após restart do terminal).
- Backend: WSL2 — daemon roda na distro `docker-desktop` (kernel Linux 29.7.2). Compose v5.4.0.
- Daemon precisa estar rodando: serviço `com.docker.service` (Start-Service se parado) ou iniciar Docker Desktop. Verificar com `docker version` / `docker info`.

## Comandos essenciais

- `docker ps -a` — todos os containers (com status); `docker ps -a --format "table {{.Names}}\t{{.Status}}\t{{.Image}}"`
- `docker logs <container> --tail 100` (ou `-f` para seguir) — diagnóstico
- `docker restart <container>` / `docker stop` / `docker start`
- `docker images`, `docker image prune -f` (limpeza), `docker system df` (uso de disco)
- `docker exec -it <container> <cmd>` — só quando necessário; preferir logs/inspeção sem entrar no container
- `docker inspect <container>` — detalhes (network, mounts, env)
- `docker compose up -d` / `down` / `logs -f` / `ps` — na pasta do projeto (arquivos compose são lidos do diretório local; atenção à montagem de volumes entre WSL2 e C:\)
- `docker pull <imagem>` / `docker push`

## MSYS2 / Windows Path Conversion (Docker Exec & Volumes)

No MSYS2 Bash no Windows, argumentos iniciados com `/` (ex: `/bin/sh`, `/var/run/docker.sock`) podem sofrer conversão automática para caminhos Windows locais (ex: `C:\msys64\usr\bin\sh`), quebrando comandos dentro do container.
- O ambiente possui `MSYS2_ARG_CONV_EXCL` configurado e aliases `docker='MSYS_NO_PATHCONV=1 docker'`.
- Em comandos manuais ou scripts avulsos, garanta o uso de `MSYS_NO_PATHCONV=1 docker exec <container> /bin/sh` ou barra dupla (`//bin/sh`) para prevenir erros como `OCI runtime exec failed: exec: "C:/msys64/...": no such file or directory`.
- Em volumes no Windows, prefira caminhos absolutos no formato Windows misto (ex: `docker run -v "C:/meu/projeto:/app"`) ou com `MSYS_NO_PATHCONV=1`.

## Docker Hub (MCP docker-hub)

- MCP server local do repositório `docker/hub-mcp` (clonado em `~/Documents/docker-hub-mcp-server`; roda via `node dist/index.js --transport=stdio`; NÃO existe pacote npm).
- **Desabilitado por padrão** no opencode.jsonc (`enabled: false`). Se o usuário quiser usar as ferramentas de busca/pesquisa de imagens e repositórios do Hub, o AGENTE NÃO edita o config — parar e pedir ao usuário para habilitar (`"docker-hub": { "enabled": true }` no bloco mcp do `~\.config\opencode\opencode.jsonc`).
- Auth opcional via env `HUB_USERNAME` + `HUB_PAT_TOKEN` (só para operações autenticadas; leitura pública funciona sem).

## Regras

- Preferir `docker compose` quando o projeto tiver compose file; docker run avulso só para testes.
- Nunca apagar volumes/containers com dados sem confirmar com o usuário (`docker rm -f`, `docker volume rm`).
- Diagnóstico de "serviço caiu": `docker ps -a` → `docker logs` → `docker restart` → re-verificar com `docker ps` e healthcheck.
- Para VPS remotos com Docker, usar as ferramentas ssh_* / ssh-manager (ver skill `ssh-vps`).
