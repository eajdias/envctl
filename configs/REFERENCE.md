# OpenCode — referência operacional

**Consulte sob demanda.** Aqui fica o detalhe que **não** precisa estar no prompt a cada turno. Abra este arquivo quando for mexer em plugin, DCP, agentes, memória, VPS, ou quando precisar dos snippets operacionais.

## Plugins (1)

- `@prevalentware/opencode-goal-plugin` — modo goal: tools `get_goal`/`set_goal`/`update_goal` + slash `/goal`. Não cria toggle no Tab.
- REMOVED (2026-09-19, opencode v2): `@tarquinen/opencode-dcp@latest` and `@dietrichgebert/ponytail` fail to load (`PluginModule.LoadError: Expected object at ["default"]` — both export a V1 async function instead of `Plugin.define({id, setup})`; only the goal-plugin exports the V2 object shape). Re-add after upstream migrates per https://opencode.ai/v2/docs/build/plugins/migrate-v1. `dcp.jsonc` REMOVED from provisioning on 2026-09-22 (YAGNI: no consumer left — re-add config + plugin together on return).
- O array `plugins` vive **somente** em `opencode.json` (arrays não mesclam entre arquivos de config).
- Ao sugerir plugin novo: valide com `npm view <pkg>` antes de gravar (a maioria não é oficial/maduro) e só mantenha com evidência de funcionamento.

## Context pruning (nativo)

DCP removido em 2026-09-22 (plugin V1 quebra o boot do v2; `dcp.jsonc` era config sem consumidor — YAGNI). Pruning agora é o compaction nativo (`compaction.keep.tokens` + checkpoints); CommandCode usa `/compact` nativo. Chame `compress` proativamente ao trocar de assunto bruscamente ou ao concluir uma sub-tarefa cujo contexto verbatim não será mais usado.

## Agentes customizados

Definidos no `opencode.json` — **não** existe mais `~/.config/opencode/agents/` (o provisioning remove o diretório; um `.md` lá sobrescreveria o JSON silenciosamente).

- **`review`** (primary): revisão read-only — bash granular read-only para evidências, `edit` negado, `subagent: allow`. Severidades BLOCKER/MAJOR/MINOR/NIT (nit só se pedido), evidência obrigatória `file:line`, YAGNI check contra callers reais, veredito APPROVE/REQUEST-CHANGES.
- **`plan`** (built-in + 1 regra: `edit spec-agent/** allow` por merge — efetivo primary; **não** dispatchável via task tool): planejamento read-only; escrita apenas em `spec-agent/` na raiz do projeto (convenção do envctl — nada de pastas da skill upstream); carrega `writing-plans`, `agent-memory` e context7.
- O agente `goal` foi removido (o `build` cobre o fluxo); `/goal` continua funcionando pelo plugin.

## Memória do agente

Skill `agent-memory`:

- **Global:** `~/.config/opencode/memory/lessons.md` + `patterns.md` (seed do envctl; conteúdo por máquina é preservado — `seed_if_missing`).
- **Projeto:** `.opencode/memory/lessons.md` + `patterns.md` (versionável apenas SEM dados privados).
- Formato: lições `❌ não faça X → ✅ faça Y (porque Z)` + data; patterns `✅ quando <situação>, faça <o que funciona>`.
- Ao gravar, classifique com a skill `memory-promotion`: processo reutilizável multi-passos → **vira skill** (criar `SKILL.md`, registrar em `configs/skills/` + `manifests/skills.yaml` e **remover a entrada da memória**); lição/anti-padrão/preferência → permanece na memória. Nunca manter o mesmo conteúdo em memória E skill.
- Nunca versionar memória global; nunca gravar segredos nem dados de domínio/fornecedor na memória global.

## Infra VPS (envctl)

- Provisioner: `envctl` (repo `https://github.com/eajdias/envctl` — fonte única de configs/skills/agentes). Bootstrap de 1 linha: `curl -fsSL https://raw.githubusercontent.com/eajdias/envctl/main/bootstrap.sh | bash`.
- **VPS nova: sempre envctl** — SSH → bootstrap → `envctl run all` → `envctl doctor` (idempotente). Nunca configurar servidor à mão. Workflow completo: skill `vps-provisioning`.
- Ao cadastrar conexão SSH: ssh-manager + inventário local (skill `ssh-vps`, workflow de 3 passos) e **sempre** provisionar a VPS — ela ganha OpenCode próprio (plano Free) e vira orquestrável.
- Limite Free estourado na VPS: **perguntar** ao usuário se quer registrar token (`opencode auth login` na VPS — o agente nunca manuseia o token). Se recusar, executar os comandos direto via SSH (skill `vps-agent-dispatch`).
- Inventário: `ssh-manager server list` + `~/.config/opencode/extras/ssh_servers.md`. VPSs conhecidas: `homologacaochatbot`, `zscanchatbot`, `zscanchatcomercial`, `zscanproxyprod`, `zscanintranet`, `zscansaclocal`. **Nunca versionar IPs/usuários/chaves.**
- Atualizar o binário numa VPS: baixar o release `envctl-linux-amd64` para `~/.local/bin/envctl`.

## Ferramentas e padrões

- **Acesso a tools das skills:** `Bash(*)`, `Read`, `Write`, `Edit`, `Glob`, `Grep`.
- **Serviços:** systemd (`systemctl status <svc>`, `journalctl -u <svc> --no-pager -n 50`, `sudo systemctl restart <svc>`); Docker (`docker ps -a`, `docker logs --tail 50 <ctr>`, `docker restart <ctr>`); PM2 (`pm2 list`, `pm2 logs <id> --lines 50`, `pm2 restart <id>`).
- **Privilegiado (Linux, non-root com sudo sem senha):** `sudo -n apt-get update`, `sudo -n systemctl restart nginx`.
- **SSH em outra VPS:** `ssh <host> "uptime && df -h /"`; fallback por CLI: `ssh-manager exec <server> "..."`.
- **Docker:** `docker ps -a --format "table {{.Names}}\t{{.Status}}\t{{.Image}}"` e `docker exec -it <container> /bin/sh`. No Windows roda nativamente do PowerShell, sem workaround de conversão de caminho.

## TUI (cli.json — fora do envctl)

`~/.config/opencode/cli.json` é preferência de terminal do usuário (tema, tabs, sidebar) — o envctl **não** gerencia: `session.sidebar: "auto"` esconde a sidebar sozinha em terminal estreito (alargue a janela ou `Ctrl+P > Open settings`). Não versinar nem sobrescrever via provisioning.

## Higiene de scratch

- **Windows:** scratch em `C:\temp` — limpe ao fim da sessão com `Remove-Item -Recurse -Force C:\temp\<seu-scratch>` (prefira um subdiretório por sessão).
- **Linux:** scratch em `/temp` — limpe com `rm -rf /temp/<seu-scratch>`.
- `envctl doctor` reporta acúmulo em cache/DB/tool-output/temp; `envctl run cleanup` remove duplicatas de plugins, tool-output >10 MB e scratch com mais de 24h.
- **Encoding (Windows):** o opencode spawna `pwsh -NoLogo -NoProfile -NonInteractive -Command` e o profile **nunca** carrega no bash tool — com o beta UTF-8 do Windows desligado (ACP/OEMCP=850) acentos viram `�`. Se o `doctor` acusar *System Code Page* 850, ative "Beta: Use Unicode UTF-8 para todo o mundo" e reinicie (não é bug do projeto).
- Se o shell do agente mostrar "Windows PowerShell (5.1)", o config de shell foi ignorado (opencode #41426) — reinicie o opencode.
- Scripts: criar em `C:\temp`/`/temp` ou no home, nunca em pastas de sistema. Automação de browser determinística: `playwright-cli` no shell (browsers próprios em `~/.cache/ms-playwright`, via `install-browser chromium`).
