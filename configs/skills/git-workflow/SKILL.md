---
name: git-workflow
description: >-
  Fluxo seguro de Git/GitHub: status, diff, branches, commits convencionais, PR, rebase, merge,
  stash, tags e worktrees.
when_to_use: >-
  Pedido explícito ou implícito de Git: commitar, abrir PR, branch, rebase, conflito, worktree,
  tag, release.
license: MIT
---


# Git e GitHub CLI


## Triggers (lista estendida)

Viva na lista de catálogo do OpenCode, truncada em 249 caracteres pelo CommandCode — por isso o
resumo da description acima é curto. Quando a skill carregar, use esta lista para casar o pedido:

Triggers: git, commit, commits, pull request, pr, gh pr, branch, branches, rebase, merge, stash,
conflict, conflito, untracked, staged, unstaged, diff, changelog, tag, release, worktree, isolar
workspace.
## Quando usar

Use para investigar, preparar ou executar um fluxo Git. O estado real do repositório vem antes de qualquer plano: um `status` resumido, uma lista de branches ou um log antigo não substituem a leitura do índice, da árvore de trabalho e dos arquivos não rastreados.

Não assuma que `gh` está autenticado, que o remoto é `origin`, que a branch de integração é `main` ou que o shell é Bash. Descubra esses fatos no ambiente atual.

## Primeiro: inspeção somente leitura

Execute e mostre ao solicitante, antes de alterar o repositório:

```bash
git status --short --branch --untracked-files=all
git status --porcelain=v1 -uall
git diff --stat
git diff
git diff --cached --stat
git diff --cached
git diff --staged --stat
git diff --staged
git diff --name-status
git diff --cached --name-status
git ls-files --others --exclude-standard
git log --oneline --decorate -n 12
git branch --show-current
git branch -vv
```

Leia separadamente:

- alterações **staged**, que já podem entrar em um commit;
- alterações **unstaged**, que ainda não estão no índice;
- arquivos **untracked**, que `git diff` não mostra por padrão;
- submodules, links simbólicos, arquivos ignorados e operações Git em andamento.

Não use `git add -A` ou `git add .` por reflexo. Selecione caminhos explícitos com `git add -- <arquivo>` e confira `git diff --cached` novamente. `git add -A` pode capturar trabalho de outra pessoa ou de outra sessão.

## Branches e commits

Branches semânticas: `feat/<escopo>-<curta>`, `fix/<escopo>-<curta>`, `refactor/...`, `docs/...`, `test/...` e `chore/...`. Uma branch deve ter um objetivo pequeno e verificável.

Use Conventional Commits: `tipo(escopo opcional): mensagem imperativa em minúsculas`. Prefira `feat:`, `fix:`, `docs:`, `refactor:`, `test:`, `build:` e `chore:` conforme a intenção real. Antes de commitar:

1. confirme que só os arquivos da tarefa entraram no índice;
2. rode a verificação exigida pelo projeto e mostre o resultado;
3. leia `git diff --cached --check` e `git diff --cached`;
4. proponha a mensagem e peça confirmação para o commit.

Não misture refatoração, correção e alteração de documentação em um commit que finge ser atômico.

## Mudanças de estado, plano e confirmação

Antes de `checkout`, `switch`, `merge`, `rebase`, `pull`, `stash`, `commit`, `push`, criação de worktree ou alteração de referência, descreva:

- estado atual e origem da mudança;
- comandos exatos;
- arquivos afetados, incluindo untracked;
- risco de perda, rebase, lock ou conflito;
- como verificar e como desfazer sem apagar trabalho.

A confirmação deve ser explícita para a operação proposta. Se o plano mudar, pare e apresente o novo plano. Uma autorização para `git add` não autoriza `push --force`; uma autorização para `commit` não autoriza rebase ou reset.

## Conflitos: inspecionar os três lados

Quando merge ou rebase parar, não escolha um lado por padrão. Primeiro mostre o estado:

```bash
git status --short --untracked-files=all
git diff --name-only --diff-filter=U
git diff --cc
git ls-files -u
```

Para cada caminho em conflito, leia a base comum e as duas versões:

```bash
git show :1:<caminho>   # base comum
git show :2:<caminho>   # ours: estado da branch atual
git show :3:<caminho>   # theirs: commit sendo incorporado
```

`:1:`, `:2:` e `:3:` são os stages 1, 2 e 3 do índice. O nome `ours` é a branch que está sendo reescrita ou mesclada; `theirs` é a ponta que está sendo incorporada. Confirme essa orientação no log, no comando que iniciou a operação e no conteúdo — não assuma pelo nome.

Para um arquivo real, compare também o contexto antes dos marcadores. Para conflito de delete/rename, verifique os dois lados em commits e o impacto de arquivos vizinhos/renomeados. Para binários, submodules ou conflito de modo de arquivo, não tente resolver como texto sem uma estratégia específica.

### Plano de resolução

Apresente um plano por arquivo ou grupo coerente:

1. explique a intenção funcional de cada lado e o conflito semântico;
2. escolha a combinação que preserva a mudança pretendida, sem apagar silenciosamente dados, migrações ou compatibilidade;
3. indique testes e critérios de aceite;
4. mostre o diff que será aplicado;
5. peça confirmação antes de editar, `git add`, `rebase --continue`, `merge --continue` ou `commit`.

Depois da confirmação, aplique a menor mudança que resolve o conflito, rode as verificações e leia `git diff`/`git diff --cached`. Se o resultado não puder ser provado, reverta apenas a resolução proposta, não o trabalho original; `git merge --abort` ou `git rebase --abort` também mudam o estado e precisam de plano e autorização.

Nunca trate `checkout --ours`, `checkout --theirs`, `restore --ours` ou `restore --theirs` como solução automática. Eles podem ser úteis depois de comparar os três lados, mas devem ser parte do plano explícito.

## Fetch, pull, rebase e PR

Inspecione a divergência antes de integrar:

```bash
git remote -v
git fetch <remoto>
git log --oneline --left-right HEAD...<remoto>/<branch>
git diff --stat HEAD..<remoto>/<branch>
```

`fetch` altera referências, mas não a árvore de trabalho; `pull`, `merge` e `rebase` mudam arquivos e/ou histórico. Não faça force-push em `main`, `master` ou em uma branch compartilhada. Se precisar republicar uma branch, discuta `push --force-with-lease` como uma operação separada, com confirmação e impacto explícito.

Fluxo usual de PR:

1. confirme branch, remote e commits que entrarão;
2. publique a branch com `git push -u <remoto> HEAD` somente após autorização;
3. crie ou revise o PR com `gh pr create`/`gh pr view`;
4. rode `gh pr checks` e leia os logs, sem aceitar falha como sucesso;
5. faça merge somente com a estratégia e o botão aprovados pelo projeto.

## Stash

Stash não é backup garantido: pode omitir arquivos ignorados, deixar refs em um commit específico e esconder o trabalho da revisão normal. Antes de `git stash push`, mostre `git status --short --untracked-files=all` e explique se `-u` ou `-a` é necessário. Confirme a operação, registre o nome do stash e restaure com revisão do diff; não aplique um stash desconhecido.

## Operações destrutivas

Não execute sem autorização específica e plano de recuperação:

- `git reset --hard` e `git reset --merge` em trabalho com mudanças;
- `git clean -fd`, `git clean -fdx` ou variantes que apagam untracked/ignored;
- `git checkout .`, `git restore .` ou `git restore --worktree .` sem revisão arquivo a arquivo;
- `git branch -D`, `git push --delete`, `git push --force` e `--force-with-lease` em branch compartilhada;
- `git rebase` com `--onto`, `--exec` ou opções que reescrevem ou descartam commits;
- `git filter-branch`, `filter-repo`, `worktree remove --force` e scripts de limpeza de refs.

`git reflog`, `git fsck` e um stash/branch de resgate podem ajudar, mas não anulam o risco. Primeiro registre o estado e mostre o comando de recuperação. Nunca use uma opção destrutiva para esconder um conflito.

## Releases e histórico

Neste repositório, o `release-please` é dono de `CHANGELOG.md` e das versões publicadas:

- `feat:` produz uma mudança minor; `fix:` produz patch; `feat!:` ou `BREAKING CHANGE:` produz major;
- `chore:`, `docs:` e `test:` normalmente não publicam uma versão;
- não crie tag, seção de versão ou release manualmente;
- confira o PR de release, o CI e a política de merge antes de aprovar;
- se um pipeline de release falhar, investigue a causa e use o procedimento oficial do repositório; não repita publicação às cegas.

Para changelog narrativo, derive fatos de `git log` e dos diffs, não de nomes de branch ou de uma suposição sobre o conteúdo.

## Worktrees

A convenção do repositório é `.worktrees/<type>-<slug>` (por exemplo,
`.worktrees/feat-agent-skills`). Um branch fica em apenas um worktree; dois
agentes não dividem a mesma árvore. **Cada agente chega lá por um caminho
diferente:**

| | OpenCode V2 | CommandCode |
|---|---|---|
| config | `.opencode/opencode.json` → `worktree.directory: .worktrees` | não existe chave de diretório em `settings.json` |
| rota da nossa convenção | worktree nativo do projeto | `cmdc -w "$PWD/.worktrees/<slug>"` (path absoluto é usado verbatim) |
| rota default do runtime | — | `~/.commandcode/worktrees/<repo>-<hash>/<name>` (fora do repo), via `/worktree` e `enter_worktree` |
| `.gitignore` | `/.worktrees/` (obrigatório, é dentro do repo) | desnecessário (dir gerenciado é externo) |

As duas rotas do CommandCode servem: a ponte por `-w` mantém a convenção do
projeto; `/worktree` e `enter_worktree` usam o diretório gerenciado. Ambas são
`git worktree` de verdade, então `envctl doctor` enxerga as duas (entradas
`prunable` viram WARN, `locked` viram INFO).

Detecte antes de criar outro workspace:

```bash
GIT_DIR=$(cd "$(git rev-parse --git-dir)" 2>/dev/null && pwd -P)
GIT_COMMON=$(cd "$(git rev-parse --git-common-dir)" 2>/dev/null && pwd -P)
git rev-parse --show-superproject-working-tree
```

Se `GIT_DIR` e `GIT_COMMON` forem diferentes e não for um submodule, já existe um linked worktree. Confirme branch, path e alterações antes de continuar. Se for necessário criar um, peça consentimento, use o path da convenção, verifique que está ignorado e faça o baseline antes da implementação:

```bash
git worktree list --porcelain
git check-ignore -q .worktrees/<name> || true
git worktree add .worktrees/<name> -b <branch> <base-ref>
```

O branch deve ser criado a partir de uma base explícita (normalmente
`origin/main`) e carregado no prompt do agente. `git worktree` isola arquivos,
não containers, portas, volumes ou serviços externos; esses recursos precisam de
nomes separados quando houver concorrência.

O comando de criação muda o estado do repositório. Não remova worktrees com
`--force` enquanto houver alterações não salvas. Use `git worktree prune` somente
depois de revisar cada entrada `prunable`; entradas `locked` são trabalho
intencional e devem ser preservadas.

## Verificação de uma operação

Depois de uma mudança autorizada, reexecute:

```bash
git status --short --branch --untracked-files=all
git diff --check
git diff --stat
git diff
git diff --cached --stat
git diff --cached
```

Acompanhe com os testes, lint e checks da aplicação. Se qualquer comando falhar, preserve a saída, não esconda o problema com reset/clean e descreva o estado real antes da próxima ação.
