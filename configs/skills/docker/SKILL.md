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

## Shell Windows (PowerShell 7 + WSL2)

O ambiente usa **PowerShell 7** como shell padrão — nenhuma conversão de caminhos POSIX acontece. Docker roda nativamente do PowerShell.
- `docker exec -it <container> /bin/sh` funciona direto, sem conversão de caminhos.
- Em volumes no Windows, use caminhos absolutos no formato misto (ex: `docker run -v "C:/meu/projeto:/app"`).
- Para comandos Linux dentro de containers exigirem shell bash: `docker exec -it <container> bash`.

## Docker Hub (MCP docker-hub)

- MCP server local do repositório `docker/hub-mcp` (clonado em `~/Documents/docker-hub-mcp-server`; roda via `node dist/index.js --transport=stdio`; NÃO existe pacote npm).
- **Desabilitado por padrão** no opencode.json (`enabled: false`). Se o usuário quiser usar as ferramentas de busca/pesquisa de imagens e repositórios do Hub, o AGENTE NÃO edita o config — parar e pedir ao usuário para habilitar (`"docker-hub": { "enabled": true }` no bloco mcp do `~\.config\opencode\opencode.json`).
- Auth opcional via env `HUB_USERNAME` + `HUB_PAT_TOKEN` (só para operações autenticadas; leitura pública funciona sem).

## Regras

- Preferir `docker compose` quando o projeto tiver compose file; docker run avulso só para testes.
- Nunca apagar volumes/containers com dados sem confirmar com o usuário (`docker rm -f`, `docker volume rm`).
- Diagnóstico de "serviço caiu": `docker ps -a` → `docker logs` → `docker restart` → re-verificar com `docker ps` e healthcheck.
- Para VPS remotos com Docker, usar as ferramentas ssh_* / ssh-manager (ver skill `ssh-vps`).
