# Simplificação do envctl — placar (checkbox de referência)

**Atualizado:** 2026-10-06 · Spec completa (detalhes, TDD, rollback): `2026-10-05-simplify-envctl.md`
**Gate de cada entrega:** `go build ./... && go vet ./... && go test ./... && golangci-lint run --new-from-rev=origin/main`

## ✅ Feito

| Fase | O quê (1 linha) |
|---|---|
| 1 T1–T3 | `.goreleaser.yml` + `release.yml` 1 step + snapshot 11/11 (`36c355b`) |
| 1b | 59 findings zerados, `only-new-issues` fora (`33d799b`) |
| 1c | pins, actionlint via action, steps subsumidos fora, `paths-ignore`, Makefile dev-only, paridade verify (`60dd152`) |
| 1d T1–T2 | ps1 aprende arm64 + contrato espelhado (`80588e2`) |
| 2 T1–T4 | 17→2 interfaces; só `PackageManager` + `Logger` restam |
| 2b | `run.go` table-driven + teste de contrato do help |
| 2c | `doctor_audit.go` 2135→168 linhas, `Execute` só despacha |
| 2d T1 | `opencode.json` base única + overlay Windows |
| 2e T1 | `pathStep` via `EnsurePathEntry`; follow-up feito: reusa `fs.BackupPathFor` |
| 2f | `pacman`+`paru` → `internal/infra/arch/` |
| 2g A–H | `ToolchainEnv`/`ExecTool`/`LookPathIn`/`ProbeCheckCommand`/`version.go`/`BackupPathFor`/`PSQuote`/ctors `diag.*` |
| 2h T1–T6, T8–T9 | `ensureStep`, archive V2 parcial, temp dono único, perf verbatim, profiles em tabela, tiny folds, `idempotencyRecorder` fora, `fakes_test.go`; T8 = MANTER os 4 |
| 2i | LSP fonte única, mcp paridade, SKILL-INDEX único, ssh base+overlay, CommandCode AGENTS, skills.yaml trava |
| 2j | `os_values` no loader (U3 resolvido), rename `debloat_windows.yaml`, shim `_envctl-shim` + `instances:`, embed-only |
| 3 | `gaming.go`, `skill_contract.go`, `cleanup_agents.go` + varredura de citações |
| 6 | manifests.md sem fantasmas, priority, guia único, contagens, arch link |
| extra | `snapshot` removido (usecase+CLI+make+docs); `bootstrap.ps1` sem `Write-Host` + BOM (advisory zerado); 17 specs antigas deletadas |

## ⏳ Falta

- [ ] **1-T4** — PR + merge na janela de release; validar 11 assets (`gh release view`)
- [ ] **1d-T3** — `bootstrap.sh` + `bootstrap.ps1` instalando da release do goreleaser
- [ ] **2h-T7 (resto)** — `provisionList` genérico (`provision_tweaks` vs `provisionListMode`); longo prazo
- [x] **5-T1** — baseline `homologacaochatbot` (Ubuntu 24.04.4, 2026-10-06): 154 checks, 145 pass, 6 warn, 3 err. Erros = 3 agents commandcode ausentes (nunca provisionado); warns = 3 sysctl host-wins (`99-sysctl.conf`), 1 reboot-required, 2 skills drift (41 deployados × 12 manifesto). T2 pulado de propósito (ciclo volta jogado fora; convergência única no M9)
- [ ] **5-T3/T4** — VM dockur + ciclo Windows (inclui teste real do `bootstrap.ps1` reescrito)
- [ ] **DoD** — checkboxes §§10 da spec estão stale (fases prontas marcadas `[ ]`); sincronizar ao fechar

## 🅿️ Estacionado (não fazer agora)

- **2d-T2** — AGENTS por OS ficam separados (delta real); unificar só via base+overlay
- **Fase 4 (mise, plano em `2026-10-06-volta-to-mise.md`)** — após baseline VPS (ver ▶️); fnm descartado como plano B
- **`verify_script_test.go`** — testa o bash do gate, fora de escopo por decisão
- **`statfs_*`** — split por build-tag é idiomático, ninguém funde

## ▶️ Próximos 3 (ordem sugerida)

1. **5-T1/T2** (VPS: baseline + idempotência) — prova a base antes de churnear
2. **Migração mise M1–M8** (`2026-10-06-volta-to-mise.md`) — sobre base provada
3. **5-T3/T4 + M9 juntos** (dockur) — 1 ciclo valida migração + Windows
