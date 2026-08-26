# Agent Memory — Lessons (não repetir) — envctl

> Consulte antes de cada tarefa; atualize a cada correção/aprendizado.
> Formato: ❌ <não faça> → ✅ <faça> (porque). Entradas curtas e acionáveis.

## Erros / Lições (não repetir)

<!-- - 2026-08-20 ❌ ... → ✅ ... (porque ...) -->
- 2026-08-21 ❌ Versionar dados de servidores (IPs, usuários, caminhos de chaves) em `configs/skills/*/SKILL.md` do envctl → ✅ Manter inventário APENAS em `~/.config/opencode/extras/ssh_servers.md` (local por máquina) e referenciar dinamicamente (`ssh-manager server list`) (porque o envctl deploya as skills para todas as máquinas e o histórico git preserva PII).
- 2026-08-21 ❌ Provisionar/sincronizar memórias globais (`configs/memory/`) no envctl → ✅ (regra refinada em 2026-08-26): memórias globais SEM dados pessoais são versionadas como **templates seed** em `configs/memory/*.md` e provisionadas com `seed_if_missing: true` (escreve SÓ se o destino não existe — baseline para máquinas novas; adições locais por máquina nunca são sobrescritas nem reverse-syncadas — `snapshot_sync.go` continua bloqueando `memory/`).
- 2026-08-26 ❌ Manter config do OpenCode em DOIS arquivos (`opencode.jsonc` + `opencode.json` sobreposto) → ✅ Fonte ÚNICA `configs/opencode.json` (e `opencode.linux.json`) + seção `cleanup` no `manifests/shell.yaml` que remove `opencode.jsonc`/`tui.json` stale no provisioning (porque arrays de `plugin` não mesclam entre arquivos — o boot carregava 18 LSPs + 6 plugins; jsonc não suporta merge com json).
- 2026-08-26 ❌ Editar `configs/*` e rodar `envctl run shell` de diretório errado → ✅ Rebuild (`go build -o envctl.exe`) + rodar do root do repo (porque o `fsManager.ReadFile` lê `configs/` do CWD e cai no FS embutido ANTIGO se rodar de outro diretório — silenciosamente "Already up to date").
- 2026-08-26 ❌ Adicionar pacotes pip/novos ao manifest e instalar com binário desatualizado → ✅ Rebuild antes de `envctl run pip`/`run shell` (porque o binário embute os manifests — o run pip "Processed 2 packages" com o binário velho pulou os pacotes novos).
- 2026-08-26 ✅ Push na main do envctl dispara Release automático (`v1.0.<commit-count>`, workflow release.yml) + CI — atualizar o CHANGELOG.md com a versão ANTES do push (o nome da release vem do commit count; tag manual desnecessária).

## Padrões / Preferências (o que funciona)

<!-- - 2026-08-20 ✅ Quando <situação>, faça <o que funciona> -->
