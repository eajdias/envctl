# ~/.bashrc - Configuração unificada do MSYS2 Bash no Windows 11

# Habilitar expansão de aliases
shopt -s expand_aliases 2>/dev/null

# --- Aliases Ergonômicos ---
# Listagem
alias ls='ls --color=auto -F'
alias ll='ls -lh --color=auto'
alias la='ls -lah --color=auto'

# Git
alias gs='git status -s'
alias gd='git diff'
alias gds='git diff --staged'
alias gl='git log --oneline --graph --decorate -n 15'
alias gp='git push'
alias gpl='git pull'
alias gwc='git worktree add'
alias gwl='git worktree list'
alias gwr='git worktree remove'

# Docker & Container Pathconv Bypass
alias dc='docker compose'
alias dps='docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Image}}"'
alias dlogs='docker compose logs --tail=50 -f'
alias docker='MSYS_NO_PATHCONV=1 docker'
alias docker-compose='MSYS_NO_PATHCONV=1 docker-compose'
alias kubectl='MSYS_NO_PATHCONV=1 kubectl'
alias podman='MSYS_NO_PATHCONV=1 podman'

# Python & Ruff
alias py='python'
alias rc='ruff check .'
alias rf='ruff format .'

# --- Environment Variables ---
export PATH="$HOME/.local/bin:$PATH"
export MSYS2_ARG_CONV_EXCL="/bin;/usr;/var;/etc;/app;/tmp;/opt;--entrypoint;-v;--volume;--mount;--workdir;-w"
if command -v cygpath >/dev/null 2>&1 && [ -n "$USERPROFILE" ]; then
  export NODE_PATH="$(cygpath -m "$USERPROFILE/node_modules")"
else
  export NODE_PATH="C:/Users/eajdias-note/node_modules"
fi

# --- FZF & Bat Integration ---
if command -v fd >/dev/null 2>&1; then
  export FZF_DEFAULT_COMMAND='fd --type f --hidden --exclude .git'
  export FZF_CTRL_T_COMMAND="$FZF_DEFAULT_COMMAND"
fi
export FZF_DEFAULT_OPTS='--height 40% --layout=reverse --border'

# --- Shell Interativo (Prompt & Completions) ---
if [[ "$-" == *i* ]]; then
  # Oh My Posh
  if [ -f "C:/Users/eajdias-note/oh-my-posh/themes/jandedobbeleer.omp.json" ] && command -v oh-my-posh >/dev/null 2>&1; then
    eval "$(oh-my-posh init bash --config 'C:/Users/eajdias-note/oh-my-posh/themes/jandedobbeleer.omp.json')"
  fi

  # GitHub CLI autocompletion
  if command -v gh >/dev/null 2>&1; then
    eval "$(gh completion -s bash)"
  fi
fi
