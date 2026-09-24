---
name: git-workflow
description: >-
  Fluxo de trabalho com Git e GitHub CLI (gh): branches semânticas, conventional commits, ciclo de PRs, resolução de conflitos, stashes, tags, rebase, isolamento via git worktree (criação, detecção, segurança) e inspecção de histórico. Triggers: git, commit, commits, pull request, pr, gh pr, branch, branches, rebase, merge, stash, conflict, conflito, changelog, tag, release, worktree, isolar workspace.
license: MIT
---

# Git & GitHub CLI Workflow

## Contexto

- Git for Windows v2.55+ com otimizações (`fscache`, `preloadindex`, `longpaths`, `autocrlf input`).
- `gh` autenticado globalmente. Shell padrão PowerShell 7 (WSL via `wsl -e bash -lc "..."`).

## Branches & Commits

Branches: `feat/<escopo>-<curta>`, `fix/...`, `refactor/...`, `docs/...`, `test/...`, `chore/...`.
Conventional commits: `tipo(<escopo opcional>): mensagem imperativa em minúsculas`.
Commits atômicos; `git status` + `git diff --staged` antes de cada commit.

## Ciclo de PR (`gh`)

`git push -u origin HEAD` → `gh pr create` → `gh pr checks` → `gh pr merge --squash --delete-branch`.
Conflitos: `git fetch origin main` + `git rebase origin/main`, resolvendo a favor do remoto. **NUNCA** force-push em `main`/`master`.

## Releases (release-please, automático)

- NUNCA crie tags (`git tag`, `gh release create`) — um workflow antigo criava 1 tag por push e poluíu o repo com 74 tags (apagadas em 2026-09-24).
- A versão nasce dos conventional commits: `feat:` → minor, `fix:` → patch, `feat!:`/`BREAKING CHANGE:` → major. `chore:`/`docs:`/`test:` não geram release.
- O bot abre o PR `chore(main): release X.Y.Z` (CHANGELOG + manifest). Confira o CI e dê merge (squash, sem deletar o branch do bot) — o merge cria tag + release e o pipeline anexa os binários.
- `CHANGELOG.md` é do bot: nunca escreva seção de versão à mão; notas breves podem ir em `[Unreleased]`.
- Release sem binários (pipeline falhou)? Re-dispare: `gh workflow run "Release Pipeline" --ref main -f version=vX.Y.Z`.

## Isolamento com git worktree

Quando precisar de workspace isolado (feature, plano, paralelo no mesmo repo) — use como complemento ao fluxo acima.

**Detecção (sempre primeiro):**
```bash
GIT_DIR=$(cd "$(git rev-parse --git-dir)" 2>/dev/null && pwd -P)
GIT_COMMON=$(cd "$(git rev-parse --git-common-dir)" 2>/dev/null && pwd -P)
```
- Se `GIT_DIR != GIT_COMMON` (e não submodule: `git rev-parse --show-superproject-working-tree`): já está num linked worktree — siga direto.
- Senão: peça consentimento antes de criar. O usuário pode trabalhar no lugar.

**Criar (sem ferramenta nativa de worktree):** siga preferência declarada → `.worktrees/` (preferido, oculto) → `worktrees/` → default `.worktrees/`. **Obrigatório** verificar `git check-ignore -q <dir>`; se ignorado, adicione ao `.gitignore` e commit. Crie com `git worktree add <path> -b <branch>`. Em erro de permissão (sandbox), trabalhe no diretório atual e reporte.

**Baseline:** rode os testes do projeto no worktree antes de implementar. Reporte falhas e aguarde decisão.

**Importante:** se tiver ferramenta nativa (`/worktree`, `enter_worktree`), use-a em vez do `git worktree add` — ela cuida posicionamento, branch e limpeza.

## Regras de segurança

- NUNCA force-push em `main`/`master`. Em divergência, `rebase` a favor do upstream.
- NUNCA comitar segredos, `.pem`, `.env`, `.env.local`.
