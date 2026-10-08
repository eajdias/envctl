# ADR 0007: Catálogo de skills enxuto (12) + code-playbooks

## Status
Aceito.

## Contexto
O catálogo de skills é injetado no prompt a **cada turno** — cada skill custa
contexto permanentemente. Medido com `cmdc -p`:

- 48 skills custavam ~6,4k tokens/turn; com o budget default do CommandCode, o
  runtime **degrada o catálogo para nome + localização** — e nesse modo a
  auto-ativação **não acontece** (0 chamadas de `activate_skill` observadas).
- Boa parte do conteúdo não pedia slot: receitas de stack, conhecimento de
  ferramenta e regras obrigatórias — que pertencem a outros lugares.

## Decisão
- **Critério de entrada estreito:** uma skill só ocupa slot se contiver algo
  que o modelo **não faria sozinho** — preferência do usuário, procedimento
  não-óbvio ou armadilha já paga.
- **Catálogo de 12 skills** (ver [../skills.md](../skills.md)), cabendo no
  budget default com a description chegando ao modelo.
- **Um catálogo-ponte para conhecimento:** `code-playbooks` é a única entrada;
  o conhecimento por stack/tema vive em `references/*.md`, carregado sob
  demanda (1 hop), com a regra dura "não responda sobre um tema sem ler o
  arquivo correspondente".
- **Destinos alternativos:** comportamento obrigatório → `AGENTS.md`;
  conhecimento do produto → repo; receita de projeto → `references/`.
- **Contrato de tamanho medido:** `description` + `when_to_use` ≤ 247
  caracteres no catálogo (truncagem do CommandCode), com a description curta e
  os gatilhos no corpo — validado por teste de conteúdo.
- **Skills servidas a dois runtimes** documentam uma seção por runtime
  (OpenCode / CommandCode) e são travadas por teste por seção.

## Consequências
- O catálogo cabe no default; a auto-ativação volta a funcionar.
- Crescer o catálogo agora custa caro **de propósito** — a pergunta "isto é
  preferência/procedimento/armadilha?" filtra antes de poluir o prompt.
- Conteúdo de stack continua disponível (via playbooks), mas fora do custo
  fixo por turno.
