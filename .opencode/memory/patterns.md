# Agent Memory — Patterns (o que funciona) — envctl

> Padrões reutilizáveis e preferências do usuário que já funcionaram.
> Formato: ✅ Quando <situação>, faça <o que funciona>. Entradas curtas e acionáveis.

## Padrões / Preferências (o que funciona)

<!-- - 2026-08-20 ✅ Quando <situação>, faça <o que funciona> -->
- 2026-08-26 ✅ Quando validar mudanças no envctl: `go build ./... && go vet ./... && go test ./...` + `envctl.exe doctor` (163 checks, meta 0 WARN/0 ERRO) + deploy real com `envctl run shell` do root do repo (porque evidência > afirmação; o doctor cobre manifestos, env vars, configs, skills, LSPs, WSL, temp).
- 2026-08-26 ✅ Quando inspecionar estado do OpenCode na máquina: `envctl run cleanup` (configs legados, tool-output >10MB, scratch `C:\temp` >24h) + `envctl run shell` (re-sincroniza configs e remove stale) antes de mexer manualmente (porque o provisioning é idempotente e deixa a máquina alinhada ao repo).
