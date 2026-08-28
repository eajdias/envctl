---
name: vps-provisioning
description: Provisionar e manter VPS/VM remotas com envctl (bootstrap de 1 linha + provisionamento idempotente: OpenCode, skills, LSPs, toolchains). Use quando o usuário pedir para provisionar, configurar, atualizar, revisar ou diagnosticar uma VPS/VM nova ou existente ("provisionar vps", "provisionar vm", "setup da vps", "instalar opencode na vps", "nova vps", "nova vm", "bootstrap na vps", "atualizar skills na vps", "envctl", "replicar ambiente no servidor").
---

# VPS Provisioning with envctl

## Quando usar
- Provisionar VPS/VM nova (Day-0)
- Atualizar/sincronizar configs, skills ou toolchains em VPS existente (Day-2)
- Diagnosticar/reparar VPS sem envctl/OpenCode ou fora do estado esperado

## Regra de ouro
NUNCA configurar VPS manualmente (apt, node, opencode à mão). O `envctl` é o provisionador único: idempotente (rodar 1 ou N vezes produz o mesmo estado final), auditável (`envctl doctor`) e determinístico.

## Pré-requisitos
- SSH key configurada (`~/.ssh-manager/.env` ou `~/.ssh`)
- Inventário: `ssh-manager server list` / `ssh-manager exec <server> "uptime && df -h /"`

## Passos
1. Identificar a VPS: `ssh-manager server list`
2. Estado atual: `ssh <vps> "which envctl && envctl doctor"`
3. Sem envctl → bootstrap (1 linha):
   `ssh <vps> "curl -fsSL https://raw.githubusercontent.com/eajdias/envctl/main/bootstrap.sh | bash"`
4. Provisionar tudo: `ssh <vps> "envctl run all"`
5. Verificar: `ssh <vps> "envctl doctor"` (divergências: `envctl doctor --fix`)
6. Re-sync em VPS existente: `ssh <vps> "envctl run shell && envctl run skills"` — NUNCA `envctl snapshot` (é sync REVERSO máquina→repo, para uso onde o repo existe, ex.: dev local; em VPS remota não sincroniza nada)

## Depois do provisionamento
- Despachar tarefas remotas: skill `vps-agent-dispatch` (subagentes OpenCode via SSH)
- Fonte da verdade: repo `C:\projetos\git-privado\envctl` (`configs/` + `manifests/`); mudanças de config vão lá e são propagadas com `envctl run shell`

## Notas
- Não versionar IPs/usuários/chaves em skills ou configs provisionadas; inventário local por máquina em `~/.config/opencode/extras/ssh_servers.md`.
- Sessão SSH não-login não sourceia `.bashrc` — usar `ssh <vps> "bash -lc 'envctl doctor'"` se o PATH vier vazio.