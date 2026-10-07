# ADR 0003: Segredos de MCP via arquivo (`{file:}`) e fallback exa → brave

## Status
Aceito (Accepted) — aplicado (PR #79).

## Contexto
As chaves dos MCPs de busca eram injetadas via interpolação de ambiente
(`{env:BRAVE_API_KEY}`, `{env:EXA_API_KEY}` em `configs/opencode.json`).
Probe ao vivo (opencode v2.0.23, Windows) provou que isso nunca funcionou:

- O opencode monta um env **próprio** para o filho MCP a partir do
  `environment` do config, sem mesclar o ambiente do processo — `{env:…}`
  resolve **vazio** (dump do filho confirmou: nenhuma var do pai presente).
- Com a chave vazia, o `brave-search-mcp-server` morre com exit 1; a mesma
  chave passada literalmente conectava. Causa isolada em duas variantes de
  probe (literal passa sempre, `{env:…}` falha sempre) — não era herança de
  processo nem credencial inválida (REST direto retornava `results=1`).
- No CommandCode o problema não existe: `${VAR:-}` expande do ambiente do
  processo (`configs/commandcode/mcp.json` intacto).

## Decisão
- Chaves via `{file:~/.config/opencode/secrets/*.key}`
  (`configs/opencode.json`: `brave.environment`, `exa.headers.x-api-key`) —
  mesmo padrão do `context7.key`. Arquivos locais por máquina, ACL
  restrita, **nunca versionados**; `{env:…}` proibido para segredo de MCP.
- Precedência de busca no lado OpenCode vira **exa → brave → nativo**
  (`AGENTS.md` ×3 + `REFERENCE.md`): exa primário (semântico), brave como
  fallback com `"disabled": true` (habilita via `/mcp`).
- Verificado ao vivo: `opencode mcp list` = exa connected, brave disabled,
  context7 connected.

## Consequências
- **Positivas**: segredo realmente entregue ao filho; ordem reflete o uso
  real (semântico primeiro, keyword como fallback); brave desligado não
  consome startup por default.
- **Negativas / conscientes**: o arquivo **precisa existir**, senão o
  opencode recusa iniciar — máquina fresca sem as chaves não boota até
  criá-los (documentado em `REFERENCE.md`). Sem caminho keyless no template;
  exa keyless só fora do provisioning.
