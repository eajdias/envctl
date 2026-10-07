# ADR 0004: Migração OpenCode V1 → V2 (retrospectiva)

## Status
Aceito (Accepted) — migração concluída 2026-09-22/24. ADR retroativo:
registra decisão já executada porque o "porquê" estava disperso em memória
e CHANGELOG, e reverter qualquer ponto abaixo quebra o boot do v2.

## Contexto
O runtime v2 mudou formato de config, ciclo de plugins e diagnóstico
(LSP inerte no runtime). Manter o formato V1 produzia boot quebrado ou
config silenciosamente ignorada.

## Decisão (o que foi feito e por quê)
- **Formato nativo V2** (`configs/opencode.json`): `agents` com
  `system`/`permissions[]`/`request.body`, `skills[]`, `mcp.servers` com
  `disabled` + `timeout`, `experimental.subagent_depth: 2`;
  `opencode debug config` com zero diagnostics como gate (CHANGELOG,
  seção "OpenCode configs em formato nativo V2").
- **Bloco `lsp` removido**: runtime v2 ignora LSP — binários seguem
  provisionados (`manifests/lsp.yaml` + `run lsp` + doctor como toolchain),
  diagnóstico do agente via lint/typecheck.
- **`dcp.jsonc` removido do provisioning** (YAGNI): plugin V1 quebra o boot
  do v2; dcp + ponytail saíram dos plugins (resta só o goal-plugin);
  pruning agora é o compaction nativo.
- **Canal oficial V2**: `https://opencode.ai/v2/install` (`~/.opencode/bin`),
  instalador PowerShell no Windows; **nunca** npm/mise (canal npm travado
  na linha 1.x — assimetria #9 da matriz). Segunda cópia user-local no
  Arch é arquivada: pacman continua autoritativo.
- **Custom com ID de built-in vira merge silencioso**: `agents.plan`
  reduzido à exceção mínima (`spec-agent/** allow`); exceção análoga
  vale para qualquer override futuro de built-in.

## Consequências
- **Positivas**: boot determinístico no v2; superfície de config mínima
  (menos merge silencioso para apodrecer a cada release upstream).
- **Negativas**: sem pruning custom até a API de plugins sair de beta;
  sem LSP no runtime (monitoramento mensal via `pacman -Si opencode`,
  assimetria #17 da matriz).
- Reversão de qualquer item = boot quebrado ou diagnóstico mudo; mudar
  exige revalidar `debug config` + `mcp list` ao vivo.
