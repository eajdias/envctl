# Agent Memory — Lessons (não repetir) — envctl

> Consulte antes de cada tarefa; atualize a cada correção/aprendizado.
> Formato: ❌ <não faça> → ✅ <faça> (porque). Entradas curtas e acionáveis.

## Erros / Lições (não repetir)

<!-- - 2026-08-20 ❌ ... → ✅ ... (porque ...) -->
- 2026-08-21 ❌ Versionar dados de servidores (IPs, usuários, caminhos de chaves) em `configs/skills/*/SKILL.md` do envctl → ✅ Manter inventário APENAS em `~/.config/opencode/extras/ssh_servers.md` (local por máquina) e referenciar dinamicamente (`ssh-manager server list`) (porque o envctl deploya as skills para todas as máquinas e o histórico git preserva PII).
- 2026-08-21 ❌ Provisionar/sincronizar memórias globais (`configs/memory/`) no envctl → ✅ Memórias globais são individuais por PC/VPS: remover do manifest e excluir do snapshot (`snapshot_sync.go` já bloqueia `memory/` e `extras/`) (porque lições de uma máquina vazariam para todas).

## Padrões / Preferências (o que funciona)

<!-- - 2026-08-20 ✅ Quando <situação>, faça <o que funciona> -->
