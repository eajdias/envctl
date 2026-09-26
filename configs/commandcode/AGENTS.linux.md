# CommandCode Environment Manifest (Linux Server)

## Ambiente

- **OS:** Ubuntu Server (LTS) (amd64) · usuário `ubuntu` (não-root, sudo sem senha)
- **Shell:** Bash (`/bin/bash`) é o shell do CommandCode — use sintaxe POSIX, não PowerShell.
- **CLIs no PATH:** `rg` (ripgrep), `fd`, `fzf`, `bat`, `delta`, `yq`, `gh`, `uv`, `ruff`, `bun`/`bunx` (substitui `npx`), `git`, `docker`, `systemctl`, `cmdc`.
- **Scratch:** `/temp` (`ENVCTL_TEMP`, na raiz do disco). Todo arquivo temporário vai para lá e é removido ao fim da sessão — nunca em `.commandcode/` nem no projeto.
- **Git:** `preloadindex`, `autocrlf=input`, `init.defaultBranch=main`, pager `delta` (sem `fscache`/`longpaths` — são do Windows).  ·  **Worktrees:** o runtime gerencia em `~/.commandcode/worktrees/<repo>-<hash>/` (fora do repo) via `/worktree` e `enter_worktree`; para cair na convenção do projeto use `cmdc -w "$PWD/.worktrees/<slug>"` (path absoluto é usado verbatim). `doctor` reporta `prunable`/`locked` nas duas rotas.

## Regras

- Código, comentários e commits em inglês; conversa com o usuário em PT-BR.
- Clean Architecture, SOLID, tipagem estrita, padrões do repositório em questão.
- Branches semânticas (`feat/`, `fix/`), conventional commits, PRs via `gh pr create`.
- **Evidência antes de afirmação:** rode build/test/lint e mostre a saída real antes de dizer que terminou.
- **Nunca deduza:** não afirme estado, causa ou diagnóstico sem comprovação executada nesta sessão; diante de relato do usuário sobre estado local observável, re-teste na hora e trate a hipótese como hipótese — a contraprova do usuário é evidência de primeira classe, e repetir prescrição sem evidência nova é erro.
- **Docs:** após mexer em código/config, confira a doc que descreve isso; manifesto e código ganham da doc.
- **Catálogo:** carregue `code-playbooks` e leia `references/<tema>.md` antes de seguir convenção de stack.
- **Regra de ouro:** use Context7 para documentação atual de qualquer lib antes de escrever código.
- **Sem clichê:** nada de filler, hedging ou frase de efeito em resposta ou texto.
- **Contexto cheio:** produza handoff e siga, em vez de resumir de memória.
- **Review:** valide o feedback antes de implementar; peça clarificação quando vago.
- **Docs:** após mexer em código/config, confira a doc que descreve isso (ler `code-playbooks/references/docs-sync.md`); manifesto e código ganham da doc.
- **Zero tolerância a WARNING/ERROR** (lint, compilador, ts(6xxx)): corrija na hora, inclusive pré-existente. Ao fechar TODOs, reconcilie a lista e RE-EXECUTE a verificação.
- Nunca hardcode segredos. ACLs restritas em `~/.ssh`.
- Delegue o trabalho barulhento (varredura ampla, output volumoso) para preservar o contexto; o critério completo está na skill `subagent-routing`.
- **Ambiguidade alta** (o alvo, o escopo ou o critério de pronto não estão claros)? **Pergunte antes de agir** — a skill `clarify-before-acting` mede isso de 0 a 100. Melhor perguntar a mais do que executar errado e ter que desfazer.

## CommandCode

- **Config:** `~/.commandcode/settings.json` — mude com `cmdc config set`, não à mão. **Regras:** `~/.commandcode/AGENTS.md` (este arquivo), carregado a cada request; os tiers user → projeto (`AGENTS.md` ou `.commandcode/AGENTS.md`) → subdiretório somam.
- **Agentes:** `~/.commandcode/agents/` (frontmatter: `name`, `description`, `tools`, `disallowedTools`, `model`, `reasoningEffort`, `maxTurns`, `permissionMode`, `background`, `showOutput`). Built-ins `general`/`explore`/`plan`/`review`; nomes reservados são ignorados — os customs aqui são `code-reviewer`, `verifier`, `docs-writer` e `memory-keeper`. Não existe campo `mode`: todo agente custom **é** subagent, e `plan` já é dispatchável (read-only, `tools: [read_file]`) — use-o para planejamento e deixe o coordenador gravar a spec em `spec-agent/`. `agent`/`agent_output` nunca podem ser concedidos (delegação tem um nível); o `doctor` valida esse schema e nomeia o campo inválido.
- **MCP:** user-scope `~/.commandcode/mcp.json`. Browser interativo via MCP `chrome-devtools` (`enabled: false` — habilite com `/mcp`); automação determinística via `pw` no shell (wrapper versionado de `playwright-cli`).
- **Skills:** carregadas **sob demanda** — o catálogo (nome + descrição) já está no prompt, não há nada a ativar. Para escolher entre elas, veja `~/.commandcode/SKILL-INDEX.md` (tabela situação → skill); não leia por padrão. **Proactive:** Se a descrição de uma skill casar com a tarefa, carregue-a com a tool `skill` antes de agir; o índice é apenas para desempate.
- **LSP:** o CommandCode não tem configuração de LSP — ele usa o do IDE conectado (`/ide` + tool `get_diagnostics`). Os binários instalados servem ao OpenCode e ao shell.
- **Memória:** no início de toda tarefa carregue `/agent-memory` e leia os tiers user → projeto (1x por sessão); ao errar, ser corrigido ou descobrir padrão reutilizável, grave lição/pattern na hora classificando antes: ver `code-playbooks/references/skill-authoring.md`; ao fechar, revise e pode duplicados. A memória vive nos arquivos `AGENTS.md` dos tiers (o CommandCode não tem memory-dir).
- **Taste:** aprende de sinais accept/reject/edit — projeto `.commandcode/taste/`, global `~/.commandcode/taste/`. Não edite à mão; use a tool `taste`.
- **Hot reload:** agentes, skills e memória são re-lidos a cada turno; `settings.json` vale no próximo round. Só um update baixado exige `/reload`.

## Serviços

- **systemd:** `systemctl status <svc>`, `journalctl -u <svc> --no-pager -n 50`, `sudo systemctl restart <svc>`
- **Docker:** `docker ps -a`, `docker logs --tail 50 <ctr>`, `docker restart <ctr>`
- **PM2:** `pm2 list`, `pm2 logs <id> --lines 50`, `pm2 restart <id>`

## Referências (leia só se precisar)

- **Qual skill para qual situação:** `~/.commandcode/SKILL-INDEX.md`
- **Skills instaladas:** `~/.commandcode/skills/<nome>/SKILL.md`
- **Servidores SSH / chaves:** `~/.config/opencode/extras/ssh_servers.md` e `~/.ssh-manager/.env` (inventário local por máquina — NUNCA versionar)
- **Ambiente:** provisionado pelo envctl. Para auditar: `envctl doctor` (0 WARN/0 ERROR = saudável). Para alterar ou provisionar: skill `envctl`.
