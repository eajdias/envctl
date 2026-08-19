---
name: ssh-vps
description: Operar VPS/servidores remotos da empresa via SSH (Linux + Windows OpenSSH) — monitorar, diagnosticar e recuperar serviços (systemd, Docker, PM2) em ~10 servidores. Use quando o usuário pedir para verificar, monitorar, reiniciar, corrigir ou gerenciar um servidor/VPS ("ssh", "vps", "server", "servidor", "instância", "monitorar servidor", "health check", "reiniciar serviço", "conectar no", "ssh-manager", "mcp de ssh", "subir serviço", "serviço caiu").
---

# SSH VPS Operations

## Contexto
O usuário é o gerente de infraestrutura da empresa. Existem ~10 VPS (Linux e Windows), cada um rodando um serviço 'cloud' das operações (systemd, Docker, PM2). O uso é sob demanda: o usuário pede, você monitora/diagnostica/recupera.

## Infraestrutura local (Windows)
- `ssh` (OpenSSH client do Windows) — disponível direto no PowerShell/Git Bash.
- `ssh-manager` CLI + `mcp-ssh-manager` (npm global via Volta, v3.8.0).
- `rsync`, `jq`, `sshpass` — instalados via MSYS2 (`C:\msys64\usr\bin`, já no PATH).
- Chaves SSH: `C:\Users\eajdias-note\Documents\SSH-keys\<VPS>\<VPS>.pem` (+ .txt com dados de conexão).

## MCP `ssh-manager` — DESATIVADO por padrão
O MCP está registrado no opencode global (`~/.config/opencode/opencode.jsonc`, seção `mcp.ssh-manager`) mas com `"enabled": false`, para não carregar ~43k tokens de contexto desnecessariamente.

**Regra de ativação — o AGENT NUNCA edita o config por conta própria:**
- O MCP é ativado SOMENTE pelo usuário, sob demanda.
- Quando o agente precisar das ferramentas `ssh_*`, ele deve PARAR e PEDIR ao usuário para ativar o MCP, explicando o motivo.
- Enquanto o MCP estiver desativado, use a CLI (Opção A) — funciona sem o MCP.

## Servidores registrados
Config: `C:\Users\eajdias-note\.ssh-manager\.env` (formato dotenv; chaves `SSH_SERVER_<NOME>_HOST/_USER/_PORT/_KEYPATH/_PASSPHRASE/_PLATFORM/...`; nome do servidor = parte entre `SSH_SERVER_` e `_HOST`, minúsculo).

| Nome | Host | Usuário | OS | Chave | Observação |
|---|---|---|---|---|---|
| zscanintranet | 3.132.24.204 | ubuntu | Ubuntu 20.04 (AWS) | Documents\SSH-keys\ZscanIntranet\ZscanIntranet.pem | Intranet da Zscan (legado) |

Para registrar novo VPS: siga o padrão (pasta em `Documents\SSH-keys\<VPS>\` + entrada no `.env` com caminho em forward-slash `C:/Users/...` e `PLATFORM=windows` para alvos Windows) e atualize esta tabela.

## Como usar

### Opção A — CLI (padrão, sem MCP)
```
ssh-manager server list                 # lista servidores
ssh-manager server test <nome>          # testa conexão
ssh-manager exec <nome> "<comando>"     # executa comando no servidor
ssh-manager exec zscanintranet "systemctl status nginx"
```
Nota: sintaxe é `ssh-manager exec <servidor> <comando>` (não existe 'exec run').

### Opção B — MCP (após habilitar + restart)
Ferramentas principais: `ssh_list_servers`, `ssh_execute`, `ssh_upload`, `ssh_download`, `ssh_sync`, `ssh_health_check`, `ssh_service_status`, `ssh_process_manager`, `ssh_execute_sudo`, `ssh_group_execute`, `ssh_tunnel_create`, `ssh_db_dump/import/list/query`, backups e sessões.

### Opção C — ssh direto / rsync (fallback)
```
ssh -i "C:\Users\eajdias-note\Documents\SSH-keys\ZscanIntranet\ZscanIntranet.pem" ubuntu@3.132.24.204 "uptime"
rsync -avz -e "ssh -i <chave>" ./dir/ ubuntu@3.132.24.204:/home/ubuntu/dir/
```

## Checklist de diagnóstico/recuperação (comum: serviço caiu)
1. Visão geral: `uptime`, `df -h /`, `free -h`, `systemctl is-system-running`.
2. Identificar o serviço: `systemctl list-units --type=service --state=failed`; Docker: `docker ps -a`; PM2: `pm2 list`.
3. Logs: `journalctl -u <serviço> --no-pager -n 50`; Docker: `docker logs --tail 50 <ctr>`; PM2: `pm2 logs <id> --lines 50`.
4. Recuperar: `sudo systemctl restart <svc>` / `docker restart <ctr>` / `pm2 restart <id>`.
5. Verificar de novo (status + is-system-running) e reportar ao usuário.

## Boas práticas
- Comece SEMPRE com comandos read-only; só use sudo quando necessário (`ssh_execute_sudo`, ou `sudo -S` com SUDO_PASSWORD se definido no `.env`).
- NADA de comandos interativos no remoto (vim, top, htop, nano) — use `cat`, `ps aux`, `systemctl status`.
- Processos longos: `nohup <cmd> > /tmp/x.log 2>&1 &`.
- Arquivos pequenos (<1MB): base64 ou `ssh_upload`; grandes: rsync (`ssh_sync` usa rsync).
- Alvos Windows (`PLATFORM=windows`): shell é PowerShell — comandos Linux (systemctl etc.) NÃO funcionam; use `Get-Service`, `sc.exe`, `Restart-Service`.
- Nunca exponha chaves/senhas no output; não logue segredos.
- Alterações no `.env` são lidas na hora (hot reload), mas o CLI usa o processo atual — reinicie se precisar.