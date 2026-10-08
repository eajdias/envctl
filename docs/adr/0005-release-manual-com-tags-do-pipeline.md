# ADR 0005: Release manual com tags do pipeline

## Status
Aceito (2026-10-06, PR #77 — substitui a automação anterior).

## Contexto
O release já foi automático por push na main e, depois, gerido pelo bot
release-please. Os dois modelos cobraram preço:

- O release por push gerou poluição de tags histórica (dezenas de tags
  automáticas, ver CHANGELOG/lições 2026-08-26) — publicar por acidente era
  mais fácil que publicar de propósito.
- O release-please burocratizou: o bot era tratado como colaborador externo
  pelo GitHub e cada PR de release exigia aprovação humana de workflow; o CI
  precisou de contorno (`push` na branch do bot + `paths-ignore` nos arquivos
  gerados) para não travar.

Em paralelo, o changelog passou a ser curado à mão (entradas em `[Unreleased]`),
e releases precisavam de um momento deliberado de decisão.

## Decisão
- **Remover release-please** (bot, configs e workflow) e o release automático
  por push.
- **`release.yml` manual**: `workflow_dispatch` com input `version: vX.Y.Z`;
  o pipeline valida o formato, cria a tag no commit atual da main, deixa o
  GoReleaser criar o GitHub Release e anexar os 11 assets (Windows/Linux ×
  amd64/arm64 + bootstraps + SHA256SUMS).
- **CHANGELOG manual**: mover as entradas de `[Unreleased]` para a seção da
  versão **antes** de disparar (a tag precisa conter a seção); a regra está
  explícita no CONTRIBUTING e no topo do CHANGELOG.
- **Tag pertence ao pipeline**: nunca criar tag à mão localmente.

## Consequências
- Publicar virou um ato explícito: quem corta escolhe a versão (feat→minor,
  fix→patch por convenção de commits, com julgamento humano).
- Tags imutáveis e contidas; sem poluição automática.
- Em troca, exige disciplina: esquecer de mover o changelog deixa a tag com a
  seção vazia (`[Unreleased]`) — aconteceu em v1.14.0/v1.14.1; hoje a ordem é
  explicitamente mover → disparar (CONTRIBUTING + topo do CHANGELOG + lição na
  memória do projeto).
- O CI por PR continua automático; só o release é manual.
