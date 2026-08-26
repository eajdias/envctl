---
name: git-workflow
description: Fluxo de trabalho avançado com Git e GitHub CLI (gh). Use quando o usuário pedir para criar branches, fazer commits convencionais, criar/revisar Pull Requests, resolver conflitos, gerenciar stashes, tags, rebase ou inspecionar histórico. Triggers: git, commit, commits, pull request, pr, gh pr, branch, branches, rebase, merge, stash, conflict, conflito, changelog, tag, release.
---

# Git & GitHub CLI Workflow

## Contexto
- Git for Windows v2.55+ com otimizações ativas (`core.fscache`, `core.preloadindex`, `core.longpaths`, `core.autocrlf input`).
- GitHub CLI (`gh`) autenticado globalmente (`gh version 2.97+`).
- Shell padrão: PowerShell 7 (WSL Ubuntu disponível via `wsl -e bash -lc "..."` para comandos POSIX).

## Convenções de Branches & Commits

### 1. Nomenclatura de Branches
- `feat/<escopo>-<descricao-curta>` — Novas funcionalidades
- `fix/<escopo>-<descricao-curta>` — Correção de bugs
- `refactor/<escopo>-<descricao-curta>` — Refatoração sem alteração de comportamento
- `docs/<escopo>-<descricao-curta>` — Documentação
- `test/<escopo>-<descricao-curta>` — Criação ou ajuste de testes
- `chore/<escopo>-<descricao-curta>` — Ajustes de build, dependências ou configs

### 2. Padrão Conventional Commits
Formato obrigatório: `<tipo>(<escopo opcional>): <mensagem imperativa em minúsculas>`
- `feat(auth): add jwt refresh token rotation`
- `fix(docker): resolve volume permission mapping on windows`
- `refactor(db): optimize connection pool settings`
- `chore(deps): update volta node runtime to v24.19.0`

## Comandos Essenciais

### Status, Diffs e Histórico
```bash
git status -s
git diff
git diff --staged
git log --oneline --graph --decorate -n 15
```

### Criação e Troca Segura de Branches
```bash
git checkout -b feat/nova-feature
# ou com git switch
git switch -c fix/correcao-bug
```

### Staging e Commits Atômicos
```bash
# Inspecionar o que está modificado antes de stagear
git status
# Adicionar apenas arquivos intencionais (NUNCA credenciais ou arquivos .env)
git add <arquivo1> <arquivo2>
git commit -m "feat(api): add endpoint for user profile export"
```

### Ciclo de Vida de Pull Request com `gh`
```bash
# 1. Enviar branch para o remote
git push -u origin HEAD

# 2. Criar Pull Request interativo ou direto
gh pr create --title "feat(escopo): resumo da funcionalidade" --body "## Descrição\n...\n\n## Testes Realizados\n..."

# 3. Inspecionar status de PRs, checks e reviews
gh pr list
gh pr view
gh pr checks

# 4. Fazer merge após aprovação e checks verdes
gh pr merge --squash --delete-branch
```

### Resolução Segura de Conflitos
```bash
# Atualizar base e fazer rebase
git fetch origin main
git rebase origin/main

# Em caso de conflito, verificar arquivos em conflito:
git status
# Resolver marcadores <<<<<<< ======= >>>>>>> nos arquivos
git add <arquivos-resolvidos>
git rebase --continue
```

## Regras de Segurança
- NUNCA fazer force-push (`git push -f` ou `--force`) na branch `main`/`master`.
- NUNCA comitar segredos, chaves de API, arquivos `.pem`, senhas ou arquivos de ambiente (`.env`, `.env.local`).
- SEMPRE rodar `git status` e `git diff --staged` antes de finalizar qualquer commit.
