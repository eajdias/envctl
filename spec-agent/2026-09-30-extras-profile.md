# Spec — extras.yaml: aplicativos opcionais (Windows + CachyOS), zero PII

> Data: 2026-09-30 · Escopo: manifest opt-in `manifests/extras.yaml` + `envctl run extras` (e `--with-extras` no `run all`) · Origem: preferências do dono (repo público: outros podem aproveitar; **nenhum app de negócio/dado pessoal**)

## Decisões do dono (confirmadas)

1. **Manifest próprio `extras.yaml`** — mesmo modelo opt-in do `gaming.yaml`; nunca no `run all` default.
2. **Windows (15 winget)**: Brave, Obsidian, Steam, Tailscale, VLC, ONLYOFFICE, Syncthing, Moonlight, WinSCP, Wireshark, Nmap, Termius, TreeSize Free, BCUninstaller, LockHunter.
   - **FORA (sem ID winget, installer próprio)**: RustDesk, Npcap — nota no guia, sem manifest.
   - WhatsApp/Discord/Spotify: **fora** (pessoal / não instalados).
3. **CachyOS (12 pacman)**: brave-origin-bin, obsidian, onlyoffice-bin, vlc, transmission-qt, picard, rustdesk-bin, anydesk-bin, tailscale, boosteroid, alacritty, mpv.
4. **Rust/Deno: fora** (decisão: não entram agora).
5. **Zero PII**: só IDs de pacote; nenhum dado de conta/negócio (Zscan Evo Server, Path of Exile etc. ficam fora).

## Impacto e contratos

- **Superfície:** `manifests/extras.yaml` (novo); `internal/domain/repository/interfaces.go` (LoadExtrasPackages); `internal/infra/embedded/manifest_repo.go` + teste; `internal/usecase/provision_packages.go` (ExecuteExtras); `internal/ui/cli/run.go` (+ comando `extras` e flag `--with-extras` no all); `docs/manifests.md`; `docs/os-and-agent-matrix.md`; `CHANGELOG.md`.
- **Contrato preservado:** `run all` não muda (extras continuam opt-in); doctor não audita extras (como gaming? gaming é auditado via Steam gate — extras NÃO: são preference, não stack).
- **Dependência:** nenhuma lib nova. Tipos existentes (winget, pacman).
- **Não-regressão:** `expectedDebloat`, `gaming` counts e testes de parcelamento seguem.

## Tarefas

### T1 — Manifests/extras.yaml + loader

**Arquivos:** `manifests/extras.yaml` (novo) · `internal/domain/repository/interfaces.go` · `internal/infra/embedded/manifest_repo.go` · `internal/infra/embedded/manifest_repo_test.go`

**Interfaces:** `LoadExtrasPackages() ([]entity.Package, error)`.

- [ ] RED: `TestLoadExtrasManifest` no `manifest_repo_test.go` — 15 winget + 12 pacman, IDs únicos, tipos válidos, `os` correto.
- [ ] GREEN: manifest + loader (mesmo shape de `LoadGamingPackages`).
- [ ] Verificação: `go test ./internal/infra/embedded/ -run TestLoadExtrasManifest`.
- [ ] Commit: `feat(extras): add the optional apps manifest and loader`

### T2 — ExecuteExtras no usecase

**Arquivos:** `internal/usecase/provision_packages.go` · test

**Interfaces:** `ExecuteExtras(ctx, onProgress)` — igual `ExecuteGaming` sem o gate de distro (winget roda em Windows, pacman em Arch; o filtro de plataforma do provisionList já resolve).

- [ ] RED: teste com mock — winget em windows, pacman em arch.
- [ ] GREEN: implementação.
- [ ] Commit: `feat(extras): provision the optional apps manifest`

### T3 — CLI: `run extras` + flag `--with-extras`

**Arquivos:** `internal/ui/cli/root.go` · `internal/ui/cli/run.go`

- [ ] `envctl run extras` (comando standalone).
- [ ] Flag `--with-extras` no `run all` (Windows e perfil CachyOS) — opt-in explícito.
- [ ] Verificação: `go build ./... && go test ./...`.
- [ ] Commit: `feat(extras): wire run extras and the --with-extras flag`

### T4 — Docs + CHANGELOG

- [ ] `docs/manifests.md` §8: extras.yaml (15 winget + 12 pacman; RustDesk/Npcap manuais).
- [ ] `docs/os-and-agent-matrix.md`: linha Windows/CachyOS com extras opt-in.
- [ ] `CHANGELOG.md` sob `[Unreleased]`.
- [ ] Commit: `docs(extras): document the optional apps profile`

## DoD

- [ ] `envctl run extras` instala os 15 winget no Windows e os 12 pacman no CachyOS, idempotente.
- [ ] `run all --with-extras` inclui extras sem quebrar os perfis existentes.
- [ ] Zero PII no diff (só IDs de pacote).
- [ ] Gates: `go build`, `go vet`, `go test ./...`, `golangci-lint` 0 issues.
- [ ] Validação ao vivo: notebook (winget) + desktop (pacman).

## Execução

Worktree `.worktrees/feat-windows-tier3` (branch `feat/windows-tier3-absorption`)? **Não** — extras é feature própria:

```
git worktree add .worktrees/feat-extras -b feat/extras-optional-apps origin/main
```