---
name: test-driven-development
description: >-
  Ciclo TDD red-green-refactor: teste falhando primeiro, código mínimo depois, em Go, TS/Node ou
  Python.
when_to_use: >-
  Vai entrar código novo ou mudar comportamento: feature, bugfix, refatoração, mudança de contrato
  — antes da implementação.
license: MIT
metadata:
  author: obra (superpowers)
  source: https://github.com/obra/superpowers
  adapted: envctl — ideia (lei de ferro + red-green-refactor) adaptada; ciclo e exemplos escritos do zero para TS/PY/GO com vitest/pytest/go test e wiring com universal-test-runner + verification-before-completion
---


# Test-Driven Development


## Triggers (lista estendida)

Viva na lista de catálogo do OpenCode, truncada em 249 chars pelo CommandCode — por isso o resumo
acima é curto. Quando a skill carregar, use esta lista para casar o pedido:

Triggers: tdd, red green refactor, teste primeiro, failing test, ciclo tdd, disciplina tdd, teste antes do código.
**Regra:** não escreva código de produção sem antes observar um teste falhar pelo motivo certo. “Acho que cobre” não é evidência.

Use TDD em feature, bugfix, refatoração e qualquer mudança de comportamento. Para protótipo descartável, código gerado ou configuração, pergunte antes de abrir uma exceção.

## Descoberta obrigatória de framework e convenção

Antes de escrever o teste:

1. Localize testes existentes (`tests/`, `__tests__/`, `*_test.go`, `*.test.*`, `test_*.py` ou padrão equivalente), CI e scripts de verificação.
2. Leia `package.json`, `pyproject.toml`, `go.mod`, arquivos de configuração e Makefile/task runner para identificar o runner, comandos e versão.
3. Examine dois ou três testes próximos: nomenclatura de arquivos e casos, estrutura AAA/Given-When-Then, fixtures, factories, helpers, assertions e forma de lidar com I/O.
4. Descubra a biblioteca de mock/fake e o gerenciador de dependências real do projeto. Não troque `pytest`, `vitest`, `go test` ou qualquer outro por hábito.

Se o projeto não tiver convenção ou runner, pare e registre a necessidade no plano antes de inventar uma configuração. Quando usar exemplos deste documento, adapte-os ao comando descoberto; não copie um comando de outro repositório. Para API ou ferramenta cuja sintaxe possa ter mudado, consulte a documentação atual.

## Ciclo RED — GREEN — REFACTOR

### RED

Escreva um teste pequeno para um comportamento observável, com nome que descreva cenário e expectativa. Ele deve falhar, não passar nem quebrar por erro de setup. Rode o comando exato do projeto e confirme a falha esperada: recurso ausente, comportamento incorreto ou bug reproduzido. Corrija o teste até a mensagem mostrar que ele realmente mede a coisa.

### GREEN

Implemente somente o necessário para o teste passar. Não esconda outro comportamento, não introduza refatoração ou dependência sem necessidade e não contorne o teste. Rode o teste direcionado, a suíte relacionada e a verificação estática prevista pelo projeto; leia erros e warnings, não apenas o código de saída.

### REFACTOR

Depois do verde, remova duplicação, ajuste nomes e extraia helpers. Mantenha a suíte verde e não misture uma mudança de comportamento com a limpeza. Se surgir uma necessidade nova, volte ao ciclo com outro teste.

## Matriz mínima de casos

Escolha os cenários relevantes para o contrato; não invente comportamento só para preencher a matriz. Para cada comportamento, documente ao menos o que é preparado, a ação e o resultado verificável.

| Caso | O que exercitar | Evidência esperada |
|---|---|---|
| **Happy path** | Entrada válida e representativa | Retorno/efeito observável conforme o contrato, sem depender de detalhes internos. |
| **Boundary** | Zero, mínimo, máximo, limite de transição, tamanho grande ou timezone relevante | O valor-limite é aceito, rejeitado ou tratado exatamente como o contrato define. |
| **Empty** | Lista, string, mapa, arquivo, `nil`/ausência ou zero, conforme aplicável | Comportamento explícito — por exemplo, resultado vazio, erro de domínio ou no-op — sem exceção acidental. |
| **Error** | Entrada inválida, violação de regra, estado inconsistente ou erro de domínio | Tipo/código/status de erro e ausência dos efeitos colaterais que importam. |
| **Dependency failure** | Timeout, indisponibilidade, resposta inválida ou falha de recurso externo | Propagação, retry, fallback ou cancelamento conforme o contrato, com a limpeza realizada e mensagem útil. |

A matriz não exige teste artificial para um cenário que não existe no domínio. Nesse caso, registre `N/A` e o motivo no plano ou no teste apropriado.

## Regressão para bugs

Um bug corrigido recebe um teste de regressão permanente:

1. Reproduza o problema com a menor entrada ou sequência que ainda dispare o bug.
2. Execute o teste e confirme que falha pela mesma razão do defeito, não por uma configuração incidental.
3. Corrija a causa, rode o teste de regressão e a suíte que cobre os consumidores afetados.
4. Mantenha o caso no repositório; não o remova quando a implementação parecer estável.

Para um bug já corrigido por outra pessoa, reconstrua a reprodução a partir do relato, do diff e do comportamento observável antes de escrever a regressão. Se a causa permanecer incerta, use `systematic-debugging` em vez de escolher uma correção por tentativa.

## Mocks apenas nas fronteiras

Use mocks, stubs ou fakes para controlar **fronteiras** que sejam lentas, não determinísticas, destrutivas ou externas ao teste: rede, banco, serviço de terceiros, sistema de arquivos, relógio, filas e fontes de evento. Configure cada double para o contrato que a fronteira deve cumprir, incluindo falha, timeout e cancelamento quando forem parte do teste.

Não transforme a integração inteira em mocks: o sistema sob teste, seus adaptadores reais e a composição entre componentes devem continuar exercitados em testes de integração. Use ambiente, container, banco temporário ou servidor de teste real quando a integração for o que precisa ser provado. Não faça mock de lógica pura só para acompanhar chamadas, nem esconda um bug de ciclo de vida atrás de um stub. Um teste que verifica apenas “o mock foi chamado” não prova o comportamento do produto.

## Critério de assertion

Cada teste precisa de uma assertion significativa sobre comportamento observável. Organize-o em Arrange–Act–Assert ou Given–When–Then, com as três partes identificáveis:

- o nome diz o cenário e o resultado esperado;
- a assertion verifica retorno, estado persistido, evento emitido, chamada de fronteira relevante ou erro conforme o contrato — não uma implementação interna;
- “não lançou exceção”, `assert true` ou apenas contar chamadas não bastam;
- se houver várias assertions, todas precisam pertencer ao mesmo comportamento e ter mensagens que ajudem a diagnosticar;
- use verificações estáveis para erros e payloads; não fixe textos voláteis, IDs aleatórios ou detalhes que não fazem parte do contrato;
- um teste de regressão deve falhar novamente quando o bug for reintroduzido.

## Comandos e gates

Use sempre o comando descoberto no projeto. Exemplos apenas ilustrativos:

```bash
# Python, quando pytest for o runner encontrado
pytest path/to/test_file.py -k "nome_do_caso" -v

# Go, quando o runner for go test
go test ./pkg/x -run TestNome -v

# Node/TypeScript, quando o projeto declarar vitest
npx vitest run path/to/file.test.ts
```

Execute o teste RED, o GREEN, a suíte relacionada e o gate completo. Use `universal-test-runner` para suíte, cobertura e comparação entre frameworks; use `verification-before-completion` antes de declarar pronto. Para cobertura, suíte completa e regressões, não substitua comandos por exemplos genéricos deste arquivo.

## Checklist de conclusão

- [ ] framework, convenção e comando foram descobertos no projeto;
- [ ] matriz happy path, boundary, empty, error e dependency failure foi considerada;
- [ ] RED falhou pelo motivo esperado antes da implementação;
- [ ] GREEN usa código mínimo e a suíte relacionada passa sem warnings/erros;
- [ ] mocks ficaram nas fronteiras e a integração real não foi substituída por mocks;
- [ ] toda assertion verifica um contrato observável e falharia se o comportamento regredisse;
- [ ] bug tem teste de regressão, quando aplicável;
- [ ] o gate final foi executado com evidência fresca.
