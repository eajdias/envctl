# OpenCode Environment Manifest

## Ambiente

- **OS:** Windows 11 Pro 25H2 (amd64) · workstation `<hostname>` · usuário `<user>`
- **Shell:** PowerShell 7 (`pwsh.exe`) é o shell do OpenCode — use sintaxe nativa (`Get-ChildItem`, `Test-Path`, `$env:NOME`), não bash. Script POSIX legado: `wsl -e bash -lc "..."` (nunca o contrário).
- **CLIs no PATH:** `rg`, `fd`, `fzf`, `bat`, `delta`, `yq`, `jq`, `ruff`, `gh`, `git`, `docker`, `node`, `npm`, `bun`/`bunx` (substitui `npx`), `dust`, `hyperfine`, `shellcheck`, `golangci-lint`.
- **Libs globais (sem venv/node_modules por projeto):** Node via `NODE_PATH=%USERPROFILE%\node_modules` (`axios`, `cheerio`, `papaparse`); Python global (`pyyaml`, `requests`, `openpyxl`, `beautifulsoup4`, `pypdf`, `python-docx`, `lxml`, `sqlite3` stdlib).
- **Scratch:** `C:\temp` (`ENVCTL_TEMP`). Todo arquivo temporário vai para lá e é removido ao fim da sessão — nunca em `.opencode/`, no projeto ou no sistema.
- **Git:** `fscache`, `preloadindex`, `longpaths`, `autocrlf=input`, pager `delta`.

## Regras

- Código, comentários e commits em inglês; conversa com o usuário em PT-BR.
- **Tom:** direto, informal, sem rodeio; code first e explicação depois (≤3 linhas). Se um default proposto pelo usuário for subótimo, aponte e proponha o melhor com trade-offs.
- Clean Architecture, SOLID, tipagem estrita, padrões do repositório em questão.
- Branches semânticas (`feat/`, `fix/`), conventional commits, PRs via `gh pr create`.
- **Evidência antes de afirmação:** exiba a saída real de build/test/lint; sem comando rodado, a verificação não conta.
- **Nunca deduza:** não afirme estado, causa ou diagnóstico sem comprovação executada nesta sessão; diante de relato do usuário sobre estado local observável, re-teste na hora e trate a hipótese como hipótese — a contraprova do usuário é evidência de primeira classe, e repetir prescrição sem evidência nova é erro.
- **Zero tolerância a WARNING/ERROR:** corrija no mesmo turno, inclusive pré-existente — falha pré-existente não é desculpa; o que não pôde ser corrigido mantém a tarefa **não concluída** (reporte o bloqueio). Ao fechar TODOs, reconcilie a lista e RE-EXECUTE a verificação.
- Nunca hardcode segredos. ACLs restritas em `~/Documents/SSH-keys`, `~/.ssh-manager`, `~/.ssh`.
- Delegue o trabalho barulhento (varredura ampla, output volumoso) para preservar o contexto; o critério completo está na skill `subagent-routing`.
- **Ambiguidade alta** (o alvo, o escopo ou o critério de pronto não estão claros)? **Pergunte antes de agir** — a skill `grill-me` mede isso de 0 a 100. Melhor perguntar a mais do que executar errado e ter que desfazer.

## OpenCode

- **Config:** `~/.config/opencode/opencode.json` (padrão único, JSON — `opencode.jsonc`/`tui.json` são removidos pelo provisioning). **Regras:** `~/.config/opencode/AGENTS.md` (este arquivo), auto-carregado. **Config não é hot-reload:** reinicie o opencode e valide com `opencode debug config`.
- **Agentes:** `review` (primary explícito) e `plan` (herda primary do built-in) — ambos read-only — use `plan` antes de implementações multi-passos e `review` antes de concluir/commitar. Detalhe em REFERENCE.md.
- **Plugins:** `opencode-goal-plugin` (dcp + ponytail removidos em 2026-09-19: quebram no opencode v2, ver REFERENCE.md). Detalhe em REFERENCE.md.
- **MCP:** `context7` (docs); `ssh-manager` + `chrome-devtools` (`disabled: true` — habilite com `/mcp`); automação determinística via `pw` no shell (wrapper versionado de `playwright-cli`, skills `web-dashboard-automation`, `playwright-prod-regression`). VPS via CLI `ssh-manager` + skills `ssh-vps`/`vps-provisioning`/`vps-agent-dispatch`.
- **Skills:** carregadas **sob demanda** — o catálogo (nome + descrição) já vem no prompt e o corpo só é lido quando a tarefa casa ou você invoca `/<skill>`. Para escolher entre elas, veja `~/.config/opencode/SKILL-INDEX.md`; não leia por padrão.
- **Memória:** no início de toda tarefa carregue `agent-memory` e leia projeto → global (1x por invocação); ao errar, ser corrigido ou descobrir padrão reutilizável, grave lição/pattern na hora (passe `memory-promotion` a cada escrita); ao fechar, revise e pode duplicados. Mecânica (paths, promoção a skill) em REFERENCE.md.

## Planejamento

- Use o agente Tab `plan` (built-in + regra `spec-agent/**`) antes de implementações multi-passos.
- Specs em `spec-agent/YYYY-MM-DD-<feature>.md` na raiz do projeto; carregue `writing-plans` + `agent-memory` primeiro.
- Tarefas bite-sized TDD com comandos reais de teste; verifique antes de declarar pronto.

## Referências (leia só se precisar)

- **Qual skill para qual situação:** `~/.config/opencode/SKILL-INDEX.md`
- **Operação do OpenCode** (plugins, DCP, agentes, memória, VPS, snippets): `~/.config/opencode/REFERENCE.md`
- **Skills instaladas:** `~/.config/opencode/skills/<nome>/SKILL.md`
- **Servidores SSH / chaves:** `~/.config/opencode/extras/ssh_servers.md` e `~/.ssh-manager/.env` (inventário local por máquina — NUNCA versionar)
- **Fonte da verdade:** repo `https://github.com/eajdias/envctl` (`configs/` + `manifests/`). As cópias locais são gerenciadas — edite no repo e rode `envctl opencode` (só OpenCode), `envctl commandcode` (só CommandCode) ou `envctl run shell` (ambos).
- **Comandos:** `envctl opencode` · `envctl commandcode` · `envctl run all|shell|skills|lsp|cleanup` · `envctl doctor [--fix]` · `envctl snapshot` (sync REVERSO máquina→repo — nunca em VPS remota)
