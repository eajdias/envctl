---
name: docker
description: >-
  Gerenciamento de Docker na máquina local. Use quando o usuário pedir para listar/inspecionar/reiniciar containers, ver logs, subir/derrubar stacks, imagens, volumes, redes, docker compose, ou operar o Docker Hub. Triggers: docker, container, containers, imagem, imagens, compose, docker compose, docker desktop, subir container, derrubar container, logs do container, restart container, docker hub, pull, push, docker ps, stack.
---

# Docker (máquina local ou VPS)

## Contexto

Detecte o ambiente antes de agir — os comandos são os mesmos, o daemon não:

- **Windows (Docker Desktop + WSL2):** Docker Desktop instalado via winget; binário em `C:\Program Files\Docker\Docker\resources\bin\docker.exe` (só entra no PATH após reiniciar o terminal). O daemon roda na distro `docker-desktop` (backend WSL2, kernel Linux). Precisa estar no ar: serviço `com.docker.service` (`Start-Service` se parado) ou abrir o Docker Desktop. Se o daemon não responder mesmo com o serviço iniciado, use a skill `docker-desktop-wsl-restart`.
- **Linux (VPS/servidor):** daemon nativo via systemd — `systemctl status docker`, `sudo systemctl start docker`. Não há Docker Desktop nem WSL2 aqui.

Valide sempre com `docker version` (client + server) e `docker info`.

## Comandos essenciais

- `docker ps -a` — todos os containers (com status); `docker ps -a --format "table {{.Names}}\t{{.Status}}\t{{.Image}}"`
- `docker logs <container> --tail 100` (ou `-f` para seguir) — diagnóstico
- `docker restart <container>` / `docker stop` / `docker start`
- `docker images`, `docker image prune -f` (limpeza), `docker system df` (uso de disco)
- `docker exec -it <container> <cmd>` — só quando necessário; preferir logs/inspeção sem entrar no container
- `docker inspect <container>` — detalhes (network, mounts, env)
- `docker compose up -d` / `down` / `logs -f` / `ps` — na pasta do projeto (arquivos compose são lidos do diretório local; atenção à montagem de volumes entre WSL2 e C:\)
- `docker pull <imagem>` / `docker push`

## Shell (Windows vs Linux)

No **Windows**, o ambiente usa **PowerShell 7** como shell padrão — nenhuma conversão de caminhos POSIX acontece. Docker roda nativamente do PowerShell.
- `docker exec -it <container> /bin/sh` funciona direto, sem conversão de caminhos.
- Em volumes no Windows, use caminhos absolutos no formato misto (ex: `docker run -v "C:/meu/projeto:/app"`).
- Para comandos Linux dentro de containers exigirem shell bash: `docker exec -it <container> bash`.

No **Linux**, caminhos são POSIX e não há camada de conversão: os mesmos comandos acima funcionam sem ajuste.

## Docker Hub

Não existe MCP do Docker Hub neste ambiente (foi removido) — use a CLI `docker` (`docker search`, `docker pull`, `docker push`) ou o site do Hub. Nunca editar config de MCP por conta própria.

## Regras

- Preferir `docker compose` quando o projeto tiver compose file; docker run avulso só para testes.
- Nunca apagar volumes/containers com dados sem confirmar com o usuário (`docker rm -f`, `docker volume rm`).
- Diagnóstico de "serviço caiu": `docker ps -a` → `docker logs` → `docker restart` → re-verificar com `docker ps` e healthcheck.
- Para VPS remotos com Docker, usar as ferramentas ssh_* / ssh-manager (ver skill `ssh-vps`).
