---
name: writing-plans
description: >-
  Especificação e plano passo a passo antes de tocar produção: arquivos afetados, riscos, rollback
  e comandos de verificação.
when_to_use: >-
  Tarefa multi-passo ou multi-arquivo, escopo ambíguo, ou pedido de plano, spec, design,
  estratégia, roadmap.
license: MIT
metadata:
  author: obra (superpowers)
  source: https://github.com/obra/superpowers
  adapted: envctl — descricao/triggers em PT-BR; planos vao para spec-agent/ (o upstream usava docs/superpowers/plans/)
---


# Escrita de planos


## Triggers (lista estendida)

Viva na lista de catálogo do OpenCode, truncada em 249 chars pelo CommandCode — por isso o resumo
acima é curto. Quando a skill carregar, use esta lista para casar o pedido:

Triggers: plano, planejar, planejamento, especificação, spec, design da implementação, passo a passo, antes de codar, tarefa grande, quebrar em etapas, roadmap.
Escreva um plano executável antes de modificar produção. O objetivo não é produzir um relatório bonito: é deixar claro o que muda, por quê, em qual ordem, como provar e como desfazer.

**Anuncie no início:** “Estou usando a skill `writing-plans` para criar o plano de implementação.”

## Regras de contexto e escopo

- Se a execução estiver em um worktree isolado, use a skill `git-workflow` para verificar ou preparar o isolamento. Detecte o worktree antes de criar outro e peça consentimento quando a criação exigir uma decisão do usuário.
- Salve o plano em `spec-agent/YYYY-MM-DD-<feature-name>.md`, na raiz do projeto. Uma preferência explícita do usuário por outro local prevalece.
- Não implemente enquanto o requisito, o impacto ou um unknown bloqueante não estiver claro.
- Se o pedido misturar subsistemas independentes, proponha specs/planos separados; cada um deve entregar software testável por conta própria.

## 1. Descoberta antes do desenho

Não adivinhe a estrutura do projeto. Antes de escrever tarefas, leia as instruções do repositório, README, arquitetura disponível, manifests/configs, CI e os pontos de entrada. Depois mapeie:

- arquivos que serão criados, modificados ou testados, com a responsabilidade de cada um;
- interfaces, consumidores, contratos e dados que atravessam as fronteiras;
- padrões já existentes de testes, lint, formatação, tipos, commits e verificação;
- dependências, variáveis de ambiente, migrações, feature flags e restrições de plataforma que precisam ser considerados.

Registre a fonte de cada decisão importante. Quando um detalhe não puder ser confirmado no código, ele é um unknown — não uma suposição escondida no plano.

## 2. Impacto e contratos

Descreva o efeito da mudança antes da sequência de passos:

- **Superfície alterada:** componentes e chamadas que podem ser afetados.
- **Contratos:** assinaturas, tipos, status/códigos de erro, configurações, formatos de arquivo e compatibilidade.
- **Dependências:** bibliotecas, serviços, dados, migrações e limites de plataforma.
- **Critério de não regressão:** quais comportamentos existentes precisam continuar funcionando.

Mantenha a nomenclatura, a linguagem e as convenções do repositório. Divida uma unidade grande quando cada parte puder ser revisada e testada de forma independente; não divida só para criar etapas sem resultado observável.

## 3. Riscos, unknowns e breaking changes

Inclua uma seção explícita com estes três blocos:

- **Riscos:** o que pode quebrar, afetar segurança, dados, performance, disponibilidade, UX ou operação; registre a probabilidade/impacto quando isso orientar a ordem das tarefas e a mitigação.
- **Unknowns:** perguntas concretas, o motivo de cada uma bloquear ou mudar o desenho, quem precisa responder e quando. Um unknown sem dono e próximo passo continua pendente.
- **Breaking changes:** mudanças incompatíveis em API pública, CLI, flags, exit codes, configuração, schema, serialização, protocolo, arquivos ou comportamento padrão. Para cada uma, liste consumidores afetados, estratégia de migração/compatibilidade, janela de depreciação e como a mudança será verificada.

Se não houver breaking change, declare isso e explique como a compatibilidade foi verificada. Não esconda uma mudança incompatível dentro de “ajuste interno”.

## 4. Rollback e reversibilidade

Para cada tarefa ou mudança com efeito persistente, indique:

1. o ponto de rollback ou a forma de reverter;
2. o backup/snapshot necessário e onde ele fica;
3. a ordem segura para desfazer código, configuração e dados;
4. feature flag, compatibilidade temporária ou outra estratégia de transição;
5. o que acontece se a reversão não for segura ou não for possível.

Priorize operações atômicas, migrações reversíveis, flags e implantações compatíveis com o estado anterior. Marque explicitamente qualquer passo irreversível e peça aprovação antes de executá-lo. Não trate “apagar e refazer” como rollback se o estado original não estiver preservado.

## 5. Formato de cada tarefa

Uma tarefa é a menor unidade que entrega um resultado verificável e passa por uma revisão útil. Cada tarefa deve conter:

- **Arquivos:** criar/modificar/testar, com caminhos exatos;
- **Interfaces:** o que consome e o que produz, com nomes e tipos que as tarefas vizinhas devem usar;
- **Passos pequenos:** uma ação por item, com checkboxes `- [ ]`;
- **TDD:** teste RED, comando e motivo esperado da falha; implementação mínima; comando GREEN;
- **Verificação:** teste direcionado, suíte relacionada, lint/typecheck/build conforme o gate real do projeto;
- **Histórico:** commits atômicos quando a política do repositório permitir, sem substituir os gates;
- **Rollback:** como desfazer a tarefa ou onde está o backup;
- **Resultado esperado:** condição observável que encerra a tarefa.

Inclua comandos reais, não apenas “rode os testes”. Informe o resultado esperado do RED e do GREEN, mas não declare PASS antes de executar. Evite `TBD`, “implementar depois”, “adicionar validação”, “handle edge cases”, “similar à tarefa N” e qualquer passo que não diga como verificar o resultado. Quando uma tarefa exigir decisão do usuário, marque isso explicitamente em vez de preencher a lacuna com texto genérico.

## 6. Definition of Done

Inclua no plano uma checklist de Definition of Done adaptada ao repositório. Ela deve verificar, no mínimo:

- [ ] requisito e escopo estão cobertos por tarefas;
- [ ] interfaces, nomes, tipos e contratos são consistentes entre tarefas;
- [ ] testes cobrem comportamento novo, limites, entradas vazias, erros e falhas de dependência relevantes;
- [ ] cada mudança de comportamento teve RED observado antes do GREEN;
- [ ] suíte relacionada, testes de regressão, lint, typecheck, build e demais gates aplicáveis passaram com evidência fresca;
- [ ] riscos, unknowns, breaking changes, migrações e consumidores foram tratados ou foram aceitos explicitamente;
- [ ] rollback/reversibilidade foi testado ou o limite irreversível foi documentado e aprovado;
- [ ] documentação, configuração e artefatos de implantação foram atualizados quando necessários;
- [ ] o diff não contém segredos, arquivos gerados sem motivo ou mudanças fora do escopo;
- [ ] o resultado final pode ser reproduzido por outra pessoa a partir dos comandos do plano.

## 7. Revisão do plano

Revise o plano contra a spec e o código encontrado, sem delegar essa conferência a um relatório genérico:

- confirme cobertura de cada requisito e cada consumidor;
- procure placeholders, nomes desatualizados e referências a arquivos inexistentes;
- confira assinaturas, tipos, paths e ordem de dependências;
- confirme que cada risco tem mitigação e cada unknown tem resposta/dono;
- confirme que breaking changes, rollback e DoD aparecem no ponto em que são relevantes;
- reduza tarefas sem resultado ou qualquer trabalho fora do escopo.

## Execução do plano

Depois de salvar, informe o caminho e ofereça execução inline ou por `subagent`, quando o ambiente disponibilizar despacho. Para execução por subagentes, use subagentes `general` com contexto autocontido, fronteira de arquivos e resultado esperado; revise o retorno antes de avançar. Não presuma uma API, um nome de ferramenta ou argumentos de despacho que não estejam documentados no ambiente atual. Na execução inline, rode os testes da tarefa antes de avançar e pare nos checkpoints definidos.
