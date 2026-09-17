# CommandCode Environment Manifest

## Ambiente

- **OS:** Windows 11 Pro 25H2 (amd64) · workstation `<hostname>` · usuário `<user>`
- **Shell:** PowerShell 7 (`pwsh.exe`) é o shell do CommandCode — use sintaxe nativa (`Get-ChildItem`, `Test-Path`, `$env:NOME`), não bash. Script POSIX legado: `wsl -e bash -lc "..."` (nunca o contrário).
- **CLIs no PATH:** `rg`, `fd`, `fzf`, `bat`, `delta`, `yq`, `jq`, `ruff`, `gh`, `git`, `docker`, `node`, `npm`, `bun`/`bunx` (substitui `npx`), `go`, `cmdc`.
- **Scratch:** `C:\temp` (`ENVCTL_TEMP`). Todo arquivo temporário vai para lá e é removido ao fim da sessão — nunca em `.commandcode/` nem no projeto.
- **Git:** `fscache`, `preloadindex`, `longpaths`, `autocrlf=input`, pager `delta`.

## Regras

- Código, comentários e commits em inglês; conversa com o usuário em PT-BR.
- Clean Architecture, SOLID, tipagem estrita, padrões do repositório em questão.
- Branches semânticas (`feat/`, `fix/`), conventional commits, PRs via `gh pr create`.
- **Evidência antes de afirmação:** rode build/test/lint e mostre a saída real antes de dizer que terminou.
- **Zero tolerância a WARNING/ERROR** (lint, compilador, ts(6xxx)): corrija na hora, inclusive pré-existente. Ao fechar TODOs, reconcilie a lista e RE-EXECUTE a verificação.
- Nunca hardcode segredos. ACLs restritas em `~/Documents/SSH-keys`, `~/.ssh-manager`, `~/.ssh`.
- Delegue o trabalho barulhento (varredura ampla, output volumoso) para preservar o contexto; o critério completo está na skill `subagent-routing`.
- **Ambiguidade alta** (o alvo, o escopo ou o critério de pronto não estão claros)? **Pergunte antes de agir** — a skill `grill-me` mede isso de 0 a 100. Melhor perguntar a mais do que executar errado e ter que desfazer.

## CommandCode

- **Config:** `~/.commandcode/settings.json` — mude com `cmdc config set`, não à mão. **Regras:** `~/.commandcode/AGENTS.md` (este arquivo), carregado a cada request; os tiers user → projeto (`AGENTS.md` ou `.commandcode/AGENTS.md`) → subdiretório somam.
- **Agentes:** `~/.commandcode/agents/` (frontmatter: `name`, `description`, `tools`, `model`, `reasoningEffort`, `maxTurns`, `permissionMode`, `background`, `showOutput`). Built-ins `general`/`explore`/`plan`/`review`; nomes reservados são ignorados — o custom aqui é `code-reviewer`.
- **MCP:** user-scope `~/.commandcode/mcp.json`. Browser interativo via MCP `chrome-devtools` (`enabled: false` — habilite com `/mcp`); automação determinística via `playwright-cli` no shell.
- **Skills:** carregadas **sob demanda** — o catálogo (nome + descrição) já está no prompt, não há nada a ativar. Para escolher entre elas, veja `~/.commandcode/SKILL-INDEX.md` (tabela situação → skill); não leia por padrão.
- **Taste:** aprende de sinais accept/reject/edit — projeto `.commandcode/taste/`, global `~/.commandcode/taste/`. Não edite à mão; use a tool `taste`.
- **Hot reload:** agentes, skills e memória são re-lidos a cada turno; `settings.json` vale no próximo round. Só um update baixado exige `/reload`.

## Referências (leia só se precisar)

- **Qual skill para qual situação:** `~/.commandcode/SKILL-INDEX.md`
- **Skills instaladas:** `~/.commandcode/skills/<nome>/SKILL.md`
- **Servidores SSH / chaves:** `~/.config/opencode/extras/ssh_servers.md` e `~/.ssh-manager/.env` (inventário local por máquina — NUNCA versionar)
- **Fonte da verdade desta configuração:** repo `https://github.com/eajdias/envctl` (`configs/` + `manifests/`). As cópias locais são gerenciadas — edite no repo e rode `envctl commandcode` (só CommandCode), `envctl opencode` (só OpenCode) ou `envctl run shell` (ambos).
- **Comandos:** `envctl commandcode` · `envctl opencode` · `envctl run all|shell|skills|lsp|cleanup` · `envctl doctor [--fix]` · `envctl snapshot` (sync REVERSO máquina→repo — nunca em VPS remota)
