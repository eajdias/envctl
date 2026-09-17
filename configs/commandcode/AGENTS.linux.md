# CommandCode Environment Manifest (Linux Server)

## Ambiente

- **OS:** Ubuntu Server (LTS) (amd64), provisionada pelo envctl · usuário `ubuntu` (não-root, sudo sem senha)
- **Shell:** Bash (`/bin/bash`) é o shell do CommandCode — use sintaxe POSIX, não PowerShell.
- **CLIs no PATH:** `rg` (ripgrep), `fd`, `fzf`, `bat`, `delta`, `yq`, `gh`, `uv`, `ruff`, `bun`/`bunx` (substitui `npx`), `git`, `docker`, `systemctl`, `cmdc`.
- **Scratch:** `/temp` (`ENVCTL_TEMP`, na raiz do disco, criado pelo envctl). Todo arquivo temporário vai para lá e é removido ao fim da sessão — nunca em `.commandcode/` nem no projeto.
- **Git:** `preloadindex`, `autocrlf=input`, `init.defaultBranch=main`, pager `delta` (sem `fscache`/`longpaths` — são do Windows).

## Regras

- Código, comentários e commits em inglês; conversa com o usuário em PT-BR.
- Clean Architecture, SOLID, tipagem estrita, padrões do repositório em questão.
- Branches semânticas (`feat/`, `fix/`), conventional commits, PRs via `gh pr create`.
- **Evidência antes de afirmação:** rode build/test/lint e mostre a saída real antes de dizer que terminou.
- **Zero tolerância a WARNING/ERROR** (lint, compilador, ts(6xxx)): corrija na hora, inclusive pré-existente. Ao fechar TODOs, reconcilie a lista e RE-EXECUTE a verificação.
- Nunca hardcode segredos. ACLs restritas em `~/.ssh`.
- Delegue o trabalho barulhento (varredura ampla, output volumoso) para preservar o contexto; o critério completo está na skill `subagent-routing`.
- **Ambiguidade alta** (o alvo, o escopo ou o critério de pronto não estão claros)? **Pergunte antes de agir** — a skill `grill-me` mede isso de 0 a 100. Melhor perguntar a mais do que executar errado e ter que desfazer.

## CommandCode

- **Config:** `~/.commandcode/settings.json` — mude com `cmdc config set`, não à mão. **Regras:** `~/.commandcode/AGENTS.md` (este arquivo), carregado a cada request; os tiers user → projeto (`AGENTS.md` ou `.commandcode/AGENTS.md`) → subdiretório somam.
- **Agentes:** `~/.commandcode/agents/` (frontmatter: `name`, `description`, `tools`, `model`, `reasoningEffort`, `maxTurns`, `permissionMode`, `background`, `showOutput`). Built-ins `general`/`explore`/`plan`/`review`; nomes reservados são ignorados — o custom aqui é `code-reviewer`.
- **MCP:** user-scope `~/.commandcode/mcp.json`. Browser interativo via MCP `chrome-devtools` (`enabled: false` — habilite com `/mcp`); automação determinística via `playwright-cli` no shell.
- **Skills:** carregadas **sob demanda** — o catálogo (nome + descrição) já está no prompt, não há nada a ativar. Para escolher entre elas, veja `~/.commandcode/SKILL-INDEX.md` (tabela situação → skill); não leia por padrão.
- **LSP:** o CommandCode não tem configuração de LSP — ele usa o do IDE conectado (`/ide` + tool `get_diagnostics`). Os binários instalados servem ao OpenCode e ao shell.
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
- **Fonte da verdade desta configuração:** repo `https://github.com/eajdias/envctl` (`configs/` + `manifests/`). As cópias locais são gerenciadas — edite no repo e rode `envctl commandcode` (só CommandCode), `envctl opencode` (só OpenCode) ou `envctl run shell` (ambos).
- **Comandos:** `envctl commandcode` · `envctl opencode` · `envctl run all|shell|skills|lsp|cleanup` · `envctl doctor [--fix]` · `envctl snapshot` (sync REVERSO máquina→repo — nunca aqui)
- **Binário:** `envctl` em `~/.local/bin`; atualizar = baixar o release `envctl-linux-amd64`
