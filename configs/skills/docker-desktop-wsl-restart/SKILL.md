---
name: docker-desktop-wsl-restart
description: Reiniciar Docker Desktop no Windows quando o WSL2 backend falha. Use quando docker reportar Wsl/ExecError, backend.sock não encontrado, daemon não responde ou montagem docker-desktop stale. Triggers: docker não sobe, docker desktop, wsl exec error, backend.sock, docker daemon, reiniciar docker, wsl shutdown, com.docker.service.
license: MIT
compatibility: opencode
---

# Restart Limpo do Docker Desktop (backend WSL2)

## Quando usar

`docker version` falha, erro `Wsl/ExecError`, `backend.sock: no such file or directory`, ou daemon não responde mesmo com o serviço iniciado.

## Passos

1. `taskkill //F //IM "Docker Desktop.exe" //T` — matar processos zumbis.
2. `wsl --shutdown` — derruba a distro `docker-desktop` e o mount `/mnt/wsl/docker-desktop` stale.
3. Reabrir Docker Desktop.
4. **Aguardar ~45s** antes de validar (o daemon sobe assíncrono).
5. Validar com `docker version` (client + server respondendo).

## Regras

- **NUNCA** desregistrar a distro `docker-desktop` (`wsl --unregister`) nessa fase — só restart limpo recria os sockets compartilhados.
- Se persistir após 2 tentativas, checar `com.docker.service` (serviço Windows) antes de mexer na distro.

## Verificação

- `docker version` → Server Engine com versão, sem erro de socket.
- `docker ps` lista containers existentes.