# ADR 0002: Toolchain Volta → mise

## Status
Aceito (Accepted) — migração concluída (PR #74, série M1–M9).

## Contexto
O toolchain Node era gerenciado pelo Volta (`VoltaManager`,
`PackageTypeVolta`, `run volta`, pacotes `type: volta`). Na prática, o Volta
acumulou modos de falha caros, todos registrados no CHANGELOG:

- `VoltaManager.IsInstalled` com falso-positivo para pacotes scoped
  (`@playwright/cli`): qualquer linha do `volta list` com token isolado `"@"`
  casava via prefixo, então `run volta` pulava instalações que o doctor
  acusava ausentes (CHANGELOG, seção `VoltaManager.IsInstalled`).
- Shells não-login (ssh/systemd/agente) não herdavam `~/.volta/bin`, e como
  `exec.LookPath` ignora `cmd.Env`, o Volta recém-instalado era reportado
  como ausente a cada run (probes `resolveOnToolchainPath`).
- Escopo só Node/npm: Go, Python e demais runtimes exigiam um instalador
  próprio cada (tarball Go, etc.).

## Decisão
Trocar o Volta pelo **mise** como gerenciador único de toolchain:

- `MiseManager` (`internal/infra/toolchain/mise_manager.go`): instala via
  `mise install <id>`, inventaria via `mise ls --json`.
- Node LTS e globals via `mise use -g` (`manifests/packages.yaml`: `jdx.mise`,
  `Node.js LTS (via mise)`, `type: mise`); globals npm pelo `NpmManager`.
- Fase 0 (`run providers`) garante mise + Node + CLIs de agente antes de
  tudo (`internal/ui/cli/run.go`); shims em `~/.local/share/mise/shims`
  entram no PATH de toolchain (`internal/infra/executil/toolchain.go`).
- `VoltaManager`, `PackageTypeVolta` e `run volta` removidos; docs migradas
  (M6–M8). Validado ao vivo na homologação (M9: `run vps` convergiu via
  mise, doctor verde, 2ª passada idempotente).

## Consequências
- **Positivas**: um resolvedor de PATH para todos os runtimes; mesma spec do
  manifesto para máquina e agente (`mise use -g node@…`); fim dos
  falso-positivos de `volta list`.
- **Negativas**: máquinas antigas ainda podem ter shims Volta órfãos no PATH
  — o doctor não os caça; limpeza manual se morder.
- ADR 0001 §1 cita `Volta` como adaptador: registro histórico, não reescrever
  (convenção ADR); esta decisão o sucede nesse ponto.
