# OpenCode Environment Manifest (Linux Server)

## Ambiente

- **OS:** Ubuntu Server (LTS) (amd64) · usuário `ubuntu` (não-root, sudo sem senha)
- **Shell:** Bash (`/bin/bash`) é o shell do OpenCode — use sintaxe POSIX, não PowerShell.
- **CLIs no PATH:** `rg` (ripgrep), `fd` (via `fdfind`), `fzf`, `bat`, `delta`, `yq`, `gh`, `uv`, `ruff`, `bun`/`bunx` (substitui `npx`), `git`, `docker`, `systemctl`, `opencode`.
- **Scratch:** `/temp` (`ENVCTL_TEMP`). Todo arquivo temporário vai para lá e é removido ao fim da sessão — nunca em `.opencode/`, no projeto ou no sistema.
- **Git:** `preloadindex`, `autocrlf=input`, `init.defaultBranch=main`, pager `delta` (sem `fscache`/`longpaths` — são do Windows).

## Regras

- Código, comentários e commits em inglês; conversa com o usuário em PT-BR.
- **Tom:** direto, informal, sem rodeio; code first e explicação depois (≤3 linhas). Se um default proposto pelo usuário for subótimo, aponte e proponha o melhor com trade-offs.
- Clean Architecture, SOLID, tipagem estrita, padrões do repositório em questão.
- Branches semânticas (`feat/`, `fix/`), conventional commits, PRs via `gh pr create`.
- **Evidência antes de afirmação:** exiba a saída real de build/test/lint; sem comando rodado, a verificação não conta.
- **Nunca deduza:** não afirme estado, causa ou diagnóstico sem comprovação executada nesta sessão; diante de relato do usuário sobre estado local observável, re-teste na hora e trate a hipótese como hipótese — a contraprova do usuário é evidência de primeira classe, e repetir prescrição sem evidência nova é erro.
- **Docs:** após mexer em código/config, confira a doc que descreve isso (ler `code-playbooks/references/docs-sync.md`); manifesto e código ganham da doc.
- **Regra de ouro do catálogo:** carregue `code-playbooks` e leia `references/<tema>.md` antes de seguir convenção de stack.
- **Zero tolerância a WARNING/ERROR:** corrija no mesmo turno, inclusive pré-existente — falha pré-existente não é desculpa; o que não pôde ser corrigido mantém a tarefa **não concluída** (reporte o bloqueio). Ao fechar TODOs, reconcilie a lista e RE-EXECUTE a verificação.
- Nunca hardcode segredos. ACLs restritas em `~/.ssh`.
- Delegue o trabalho barulhento (varredura ampla, output volumoso) para preservar o contexto; o critério completo está na skill `subagent-routing`.
- **Ambiguidade alta** (o alvo, o escopo ou o critério de pronto não estão claros)? **Pergunte antes de agir** — a skill `clarify-before-acting` mede isso de 0 a 100. Melhor perguntar a mais do que executar errado e ter que desfazer.

## OpenCode

- **Config:** `~/.config/opencode/opencode.json` (padrão único, JSON — `opencode.jsonc`/`tui.json` são removidos pelo provisioning). **Regras:** `~/.config/opencode/AGENTS.md` (este arquivo), auto-carregado. **Config não é hot-reload:** reinicie o opencode e valide com `opencode debug config`.
- **Agentes:** `review` (primary explícito), `plan` (primary built-in) e `planner` (subagent read-only de planejamento) — use `planner` para pesquisa/plano volumoso, `plan` no Tab para planejamento interativo e `review` antes de concluir/commitar. Detalhe em REFERENCE.md.
- **Plugins:** `opencode-goal-plugin` (dcp + ponytail removidos em 2026-09-19: quebram no opencode v2, ver REFERENCE.md). Detalhe em REFERENCE.md.
- **MCP:** `context7` (docs); `ssh-manager` + `chrome-devtools` (`disabled: true` — habilite com `/mcp`); automação determinística via `pw` no shell (wrapper versionado de `playwright-cli`, `code-playbooks/references/web-automation.md`). Remoto via CLI `ssh-manager` + `code-playbooks/references/infra.md`.
- **LSP:** binários instalados para shell/IDE; o runtime v2 ignora LSP — diagnóstico do agente via lint/typecheck.
- **Skills:** carregadas **sob demanda** — o catálogo (nome + descrição) já vem no prompt e o corpo só é lido quando a tarefa casa ou você invoca `/<skill>`. Para escolher entre elas, veja `~/.config/opencode/SKILL-INDEX.md`; não leia por padrão. **Proactive:** Se a descrição de uma skill casar com a tarefa, carregue-a com a tool `skill` antes de agir; o índice é apenas para desempate.
- **Memória:** no início de toda tarefa carregue `agent-memory` e leia projeto → global (1x por invocação); ao errar, ser corrigido ou descobrir padrão reutilizável, grave lição/pattern na hora classificando antes: ver `code-playbooks/references/skill-authoring.md`; ao fechar, revise e pode duplicados. Mecânica (paths, promoção a skill) em REFERENCE.md.

## Planejamento

- Use `planner` para pesquisa e planos volumosos; use o agente Tab `plan` para planejamento interativo. Ambos são read-only; specs vão para `spec-agent/`.
- Specs em `spec-agent/YYYY-MM-DD-<feature>.md` na raiz do projeto; carregue `writing-plans` + `agent-memory` primeiro.
- Tarefas bite-sized TDD com comandos reais de teste; verifique antes de declarar pronto.

## Serviços

- **systemd:** `systemctl status <svc>`, `journalctl -u <svc> --no-pager -n 50`, `sudo systemctl restart <svc>`
- **Docker:** `docker ps -a`, `docker logs --tail 50 <ctr>`, `docker restart <ctr>`
- **PM2:** `pm2 list`, `pm2 logs <id> --lines 50`, `pm2 restart <id>`

## Referências (leia só se precisar)

- **Qual skill para qual situação:** `~/.config/opencode/SKILL-INDEX.md`
- **Operação do OpenCode** (plugins, DCP, agentes, memória, VPS, snippets): `~/.config/opencode/REFERENCE.md`
- **Skills instaladas:** `~/.config/opencode/skills/<nome>/SKILL.md`
- **Servidores SSH / chaves:** `~/.config/opencode/extras/ssh_servers.md` e `~/.ssh-manager/.env` (inventário local por máquina — NUNCA versionar)
- **Ambiente:** provisionado pelo envctl. Para auditar: `envctl doctor` (0 WARN/0 ERROR = saudável). Para alterar: skill `envctl`.
