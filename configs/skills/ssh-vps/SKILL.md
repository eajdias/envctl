---
name: ssh-vps
description: Operar VPS/servidores remotos da empresa via SSH (Linux + Windows OpenSSH) — monitorar, diagnosticar e recuperar serviços (systemd, Docker, PM2) em ~10 servidores. Use quando o usuário pedir para verificar, monitorar, reiniciar, corrigir ou gerenciar um servidor/VPS ("ssh", "vps", "server", "servidor", "instância", "monitorar servidor", "health check", "reiniciar serviço", "conectar no", "ssh-manager", "mcp de ssh", "subir serviço", "serviço caiu").
---

# SSH VPS Operations

## Contexto
O usuário é o gerente de infraestrutura da empresa. Existem ~10 VPS (Linux e Windows), cada um rodando um serviço 'cloud' das operações (systemd, Docker, PM2). O uso é sob demanda: o usuário pede, você monitora/diagnostica/recupera.

## Infraestrutura local
- `ssh` (OpenSSH client) — disponível direto no terminal.
- `ssh-manager` CLI + `mcp-ssh-manager` (npm global via Volta).
- `rsync`, `jq`, `sshpass` — ferramentas auxiliares.
- Chaves SSH: `~/Documents/SSH-keys/<VPS>/<VPS>.pem` (+ .txt com dados de conexão).

## MCP `ssh-manager` — DESATIVADO por padrão
O MCP está registrado no opencode global (`~/.config/opencode/opencode.json`, seção `mcp.ssh-manager`) com `"enabled": false`, para não carregar ~43k tokens de contexto desnecessariamente.

**Regra — o AGENT NUNCA edita o config por conta própria.**

**Quando PEDIR ativação:** se você PERCEBER que a tarefa ficaria melhor/mais segura com as ferramentas MCP, PARE e PEÇA ao usuário para ativar o MCP, explicando o motivo. Cenários em que é ideal:
- Operações em **múltiplos VPS ao mesmo tempo** (`ssh_group_execute`)
- **Health check / status consolidado** de vários servidores (`ssh_health_check`, `ssh_service_status`)
- **DB ops** (`ssh_db_dump/import/query`), **túnel SSH** (`ssh_tunnel_create`), **backups/sessões**
- Upload/download/rsync estruturado (`ssh_upload/download/sync`) e tarefas repetitivas longas

**Como pedir (mensagem ao usuário):** explique o motivo e peça para ativar o MCP:
1. No opencode, use o comando `/mcp` (ou `Ctrl+P` → busque "mcp")
2. Faça o toggle para **ativar** o `ssh-manager` — é hot-reload, não precisa reiniciar
3. Avise quando estiver ativo para eu prosseguir com as ferramentas `ssh_*`

**Enquanto o MCP estiver desativado:** use a CLI (Opção A) ou ssh/rsync (Opção C) — funcionam sem o MCP.

## Servidores registrados — inventário LOCAL (nunca versionar)

**Os dados reais de servidores (IPs, usuários, caminhos de chaves) ficam APENAS em arquivos locais por máquina — NUNCA neste SKILL.md nem em qualquer arquivo versionado.**

Fonte de consulta (em ordem):
1. `~/.config/opencode/extras/ssh_servers.md` — inventário local de servidores (individual por PC/VPS, nunca commitado)
2. `~/.ssh-manager/.env` — config do ssh-manager (formato dotenv; chaves `SSH_SERVER_<NOME>_HOST/_USER/_PORT/_KEYPATH/_PASSPHRASE/_PLATFORM/...`; nome do servidor = parte entre `SSH_SERVER_` e `_HOST`, minúsculo)
3. `ssh-manager server list` (CLI) ou `ssh_list_servers` (MCP) — lista dinâmica

**⚠️ REGRA DE SEGURANÇA:** nunca adicione IPs, usuários, hostnames ou caminhos de chaves reais em `SKILL.md`, README, docs ou qualquer arquivo versionado. Se o usuário fornecer, registre apenas no inventário local (`~/.config/opencode/extras/ssh_servers.md`).

## Cadastro de nova VPS (workflow rápido)

Quando o usuário pedir para cadastrar VPS, siga estes 3 passos:

**1. Ler o .txt** — caminho informado pelo usuário (geralmente `~/Documents/SSH-keys/<Nome>/<Nome>.txt`). Extrair: host, user, key path.

**2. Append no `.env`** — adicione bloco no final de `~/.ssh-manager/.env`:
```
# Server: <NomeDisplay>
SSH_SERVER_<NOMEUpper>_HOST=<ip>
SSH_SERVER_<NOMEUpper>_USER=<user>
SSH_SERVER_<NOMEUpper>_PORT=22
SSH_SERVER_<NOMEUpper>_KEYPATH=~/Documents/SSH-keys/<Pasta>/<Arquivo>.pem
SSH_SERVER_<NOMEUpper>_DESCRIPTION="<descrição>"
```
- Nome do servidor = `<NOMEUpper>` em minúsculo no MCP (ex: `<NOME>` → `<SERVER>`)
- Key path: use `~` para home do usuário, nunca caminho absoluto hardcoded — o ssh-manager expande `~` automaticamente (`os.homedir()`)

**Localização das chaves por plataforma** (a pasta exata vem do .txt do usuário):
- Windows: `~/Documents/SSH-keys/<VPS>/<VPS>.pem`
- Linux/macOS: `~/.ssh/<VPS>.pem` (padrão `ssh-config.linux`)

**3. Testar e atualizar inventário local** — `ssh_manager_ssh_execute` com `echo "OK - $(hostname)"`; depois ADICIONE/ATUALIZE o servidor em `~/.config/opencode/extras/ssh_servers.md` (nome, host, usuário, OS, chave, observação). NUNCA adicione a tabela neste SKILL.md.

**4. Provisionar com envctl (VPS vira agente OpenCode)** — após cadastro validado, NUNCA deixar a VPS crua: rode o bootstrap do envctl na VPS (1 linha) e provisione:
```
ssh <vps> "curl -fsSL https://raw.githubusercontent.com/eajdias/envctl/main/bootstrap.sh | bash"
ssh <vps> "envctl run all && envctl doctor"
```
Assim a VPS ganha OpenCode + skills próprios (plano Free) e passa a ser orquestrável via `vps-agent-dispatch`. Se o usuário só quiser monitorar a VPS, pule este passo. Detalhes do fluxo completo: skill `vps-provisioning`.

## Como usar

### Opção A — CLI (padrão, sem MCP)
```
ssh-manager server list                 # lista servidores
ssh-manager server test <nome>          # testa conexão
ssh-manager exec <nome> "<comando>"     # executa comando no servidor
ssh-manager exec <nome> "systemctl status nginx"
```
Nota: sintaxe é `ssh-manager exec <servidor> <comando>` (não existe 'exec run').

### Opção B — MCP (após habilitar via `/mcp` ou `Ctrl+P`, hot-reload)
Ferramentas principais: `ssh_list_servers`, `ssh_execute`, `ssh_upload`, `ssh_download`, `ssh_sync`, `ssh_health_check`, `ssh_service_status`, `ssh_process_manager`, `ssh_execute_sudo`, `ssh_group_execute`, `ssh_tunnel_create`, `ssh_db_dump/import/list/query`, backups e sessões.

### Opção C — ssh direto / rsync (fallback)
```
ssh -i "<caminho da chave do inventário local>" <user>@<host> "uptime"
rsync -avz -e "ssh -i <chave>" ./dir/ <user>@<host>:/home/<user>/dir/
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
- Nunca escreva IPs/usuários/caminhos de chaves reais em arquivos versionados — registre no inventário local `~/.config/opencode/extras/ssh_servers.md`.
- Alterações no `.env` são lidas na hora (hot reload), mas o CLI usa o processo atual — reinicie se precisar.