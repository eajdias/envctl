# Simplificação do envctl + migração mise — placar (checkbox de referência)

**Atualizado:** 2026-10-06 · Specs completas (detalhes, TDD, rollback): `2026-10-05-simplify-envctl.md` (simplificação) + `2026-10-06-volta-to-mise.md` (migração M1–M9)
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
| mise M1 | `MiseManager` isolado + `PackageTypeMise` (TDD, `mise ls --json` confirmado na doc) |
| mise M2+M5 | bootstrap + providers via mise/npm (tarball Go deletado); `ToolchainDirs` sem `.volta/bin`; taxonomia `sourceMise` |

## ✅ Tudo feito (só resta o PR para `main`)

- [x] **1-T4** — v1.13.1 publicada pelo goreleaser (PR #73 mergeado, job attach success, 11/11 assets)
- [x] **1d-T3** — `bootstrap.sh` (homolog, via gh) + `bootstrap.ps1` (estação, via HTTPS) instalaram v1.13.1 e rodaram `version`; bug `Test-Path -and` achado e corrigido no ciclo
- [x] **DoD** — §§10 da spec sincronizados 2026-10-06; restam só "PR separado" e "rollback pós-merge", que fecham com o PR da branch para `main`
- [x] **5-T1** — baseline `homologacaochatbot` (Ubuntu 24.04.4, 2026-10-06): 154 checks, 145 pass, 6 warn, 3 err. Erros = 3 agents commandcode ausentes (nunca provisionado); warns = 3 sysctl host-wins (`99-sysctl.conf`), 1 reboot-required, 2 skills drift (41 deployados × 12 manifesto). T2 pulado de propósito (ciclo volta jogado fora; convergência única no M9)
- [x] **M3** — manifests: 9 packages (`node`→mise, globals→npm) + 11 LSPs (`install_type`→npm)
- [x] **M4** — `update.go`: `GroupVolta` → `GroupMise` + `GroupNpm`
- [x] **M6** — doctor: tabela toolchain + refs volta
- [x] **M7** — deletar volta de vez (`volta_manager`, `PackageTypeVolta`, testes)
- [x] **M8** — docs/matriz (canal mise por OS, `Volta.Volta` fora)
- [x] **M9** — validação viva (feito 2026-10-06: homolog `run vps` convergiu via mise, doctor 152/148/4/0; 2ª run idempotente; **Windows estação do dono**: `run windows` convergiu 15→2 warns, ambos artefato-by-design de dev-machine — envctl fora do PATH e freshness `vdev`; **bug real achado e corrigido no ciclo**: audit comparava `opencode.json` com o template sem overlay → WARN eterno no Windows; agora aplica `withWindowsShellOverlay` igual ao deploy)
- [x] **DoD** — §§10 da spec sincronizados 2026-10-06; restam só "PR separado" e "rollback pós-merge", que fecham com o PR da branch para `main`

## 🅿️ Estacionado (não fazer agora)

- **2h-T7 (resto)** — `provisionList` genérico: generics sobre tweaks (registro Windows!) por ~-60 linhas; risco/benefício não fecha

- **2d-T2** — AGENTS por OS ficam separados (delta real); unificar só via base+overlay
- **fnm** — descartado como plano B (mise decidido; ver análise 2026-10-06)
- **`verify_script_test.go`** — testa o bash do gate, fora de escopo por decisão
- **`statfs_*`** — split por build-tag é idiomático, ninguém funde

## ▶️ Resta (fora do meu alcance agora)

1. **PR da branch para `main`** — fecha "PR separado" + "rollback pós-merge" do DoD
