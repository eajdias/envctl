---
name: python-development
description: >-
  Desenvolver, revisar e depurar aplicações Python respeitando a versão, as dependências e o ambiente do projeto. Use ao criar ou alterar módulos, CLIs, APIs, async, tipos, testes ou packaging. Triggers: python, pyproject.toml, requirements, uv, venv, pytest, ruff, type checker, pathlib, context manager, asyncio, import, entry point, packaging ou debugging Python.
license: MIT
---

# Python Development

## Quando usar

- Implementar, corrigir, revisar ou refatorar código Python.
- Criar módulos, CLIs, APIs, integrações, jobs ou empacotamento.
- Trabalhar com async, cancelamento, tipagem, recursos ou entry points.
- Investigar testes, Ruff, type checker, imports ou incompatibilidade de versão.
- Adaptar um projeto existente sem impor ferramentas de outro repositório.

## Contexto

- Inspecione `pyproject.toml`; quando existir, leia também `setup.cfg`, `setup.py`, `tox.ini`, `noxfile.py`, lockfiles, `.python-version` e CI.
- Confirme a versão de Python suportada em `requires-python`, classifiers, scripts e jobs de CI. Essa é a Python floor do projeto.
- Identifique o gerenciador e o ambiente virtual: `uv`, Poetry, Pipenv, venv, Conda ou outro fluxo declarado.
- Leia imports, callers, testes, scripts de console e convenções existentes antes de escolher biblioteca, formatter, linter ou type checker.
- Respeite as dependências e o lockfile; não atualize versões apenas porque uma versão mais nova parece melhor.
- Verifique se o projeto suporta Windows, Linux e macOS antes de assumir paths, encoding ou subprocess.
- Consulte `context7-auto` antes de usar API, sintaxe ou configuração cuja compatibilidade não esteja clara.
- Para mudança de comportamento, use `test-driven-development`; para falha, `systematic-debugging`; para suíte completa, `universal-test-runner`.
- Leia `pyproject.toml` também para nome, versão, backend, `[project]` e `[tool.*]`; não trate dependências como uma lista solta.
- Inspecione `__init__.py`, layout de `src/`, módulos públicos e efeitos de import antes de mover símbolos ou criar subpacotes.
- Mapeie a ordem de carregamento de configuração e variáveis de ambiente; não revele valores ao imprimir um objeto de settings.
- Confira a matriz de CI para saber quais versões de Python e sistemas operacionais o projeto realmente promete suportar.

## Passos

1. **Enquadre o contrato.** Defina entradas, saídas, erros, a Python floor e o que o entry point deve fazer. Leia o consumidor real antes de adicionar abstrações.
2. **Use o ambiente do projeto.** Ative o venv ou execute com `uv run`/ferramenta declarada. Não instale dependências no Python global para contornar configuração quebrada nem use sintaxe/API acima da versão suportada.
3. **Mantenha dependências estáveis.** Prefira a biblioteca padrão e os padrões já adotados. Só acrescente pacote quando o contrato justificar; atualize dependências e lockfiles separadamente, com revisão explícita.
4. **Marque fronteiras, não tudo.** Use type hints onde dados entram, saem ou atravessam uma API. Escolha `Protocol`, `TypedDict` ou uma classe simples conforme o contrato, sem impor Pydantic ou dataclass apenas para adicionar anotações.
5. **Respeite o modelo existente.** Não force `dataclass`, Pydantic, Black ou mypy quando o projeto usa outra abordagem. Use o formatador e o type checker configurados, inclusive alternativas como Pyright ou `ty`.
6. **Trate exceções com intenção.** Capture apenas erros que a função saiba tratar e acrescente contexto sem revelar segredo. Use `except Exception` apenas para recuperar ou reemitir com informação útil; nunca engula silenciosamente.
7. **Preserve interrupções.** Não use `except:` nem capture amplo de `BaseException`; deixe `KeyboardInterrupt` e `SystemExit` propagarem, salvo cleanup explícito que os reemita.
8. **Use pathlib e context managers.** Prefira `pathlib.Path` a concatenação manual de paths. Use `with open(...)`, gerenciadores de contexto e `contextlib.ExitStack` para arquivos, locks, conexões e recursos temporários; não confie em GC para cleanup essencial.
9. **Use async somente com concorrência real.** I/O concorrente ou integração orientada a eventos pode justificar `async`; tarefas curtas ou CPU-bound normalmente permanecem síncronas. Para operações externas, defina timeout, propague cancelamento e garanta cleanup em `finally` ou `async with`; não engula `CancelledError`.
10. **Proteja segredos.** Nunca registre tokens, senhas, cookies, chaves, valores completos de ambiente ou payloads com credenciais. Redija logs e tracebacks; cite o nome do segredo, nunca seu valor.
11. **Valide packaging.** Confirme o nome do pacote, `__init__`, `[project.scripts]` e entry points configurados. Importar o pacote não deve iniciar trabalho, exigir dependências opcionais nem tocar rede ou disco sem contrato explícito.
12. **Faça transições de erro auditáveis.** Ao adaptar uma exceção, use encadeamento explícito (`raise ... from err`) quando o projeto permitir e mantenha a exceção original disponível para diagnóstico.
13. **Dê lifecycle às tarefas async.** Crie tasks em um escopo do qual possa cancelar e aguardar; não deixe `create_task()` sem referência, timeout ou cleanup.
14. **Respeite a fronteira de I/O.** Não bloqueie o event loop com CPU, filesystem síncrono ou chamadas de rede sem estratégia explícita para executá-las fora dele.
15. **Mantenha compatibilidade.** Escolha `asyncio.timeout`, `wait_for` ou outra primitiva disponível na Python floor do projeto, em vez de copiar uma solução de versão mais nova.
16. **Documente o entry point.** O módulo executado e o console script devem ter o mesmo ambiente, os mesmos erros e a mesma política de logging.

## Verificação

- No ambiente correto, rode o comando de testes do projeto, normalmente `pytest` ou `uv run pytest`; não troque por uma suíte global.
- Rode `ruff check` e `ruff format --check` conforme a configuração existente. Não substitua Ruff por Black só por hábito.
- Rode o type checker explicitamente configurado, como `mypy`, `pyright` ou `ty`, com a versão e as flags do projeto. Se não houver type checker, não adicione mypy só para fechar uma lista.
- Valide imports e entry points no mesmo ambiente, por exemplo `uv run python -c "import nome_do_pacote"` e `uv run python -m pacote --help` ou o console script declarado.
- Se o projeto for distribuível, construa o pacote com a ferramenta configurada (`uv build`, `python -m build` etc.) e teste o artefato conforme o contrato do repositório.
- Confira o diff de `pyproject.toml` e lockfiles: Python floor, dependências e entry points não devem mudar sem solicitação explícita.
- Teste também o caminho de erro, timeout/cancelamento e cleanup quando esses forem parte do contrato.
- Use `pytest -q` ou a opção do projeto para uma execução estável; isole um teste com `-k` antes de depurar a suíte completa.
- Verifique o código de saída e o resumo de testes; um processo que termina sem erro ainda pode ter ignorado testes ou fixtures.
- Rode `ruff` no mesmo ambiente do Python para evitar analisar arquivos com uma versão diferente da usada nos testes.
- Se o projeto usar plugin, marker ou tag de pytest, inclua a opção oficial em vez de desativar testes silenciosamente.
- Para entry points, execute também o comando instalado, se houver, e confirme que o `--help` não importa dependências opcionais.
- `pip check` ou `uv pip check` é útil para dependências resolvidas, mas não substitui testes, Ruff ou o type checker.
- Confira `git diff --check` e o status do repositório para evitar que um comando de verificação tenha alterado arquivos fora do escopo.

## Regras

- Python floor, dependências e gerenciador são contratos do projeto, não padrões do agente.
- Não há async universal: blocking simples permanece síncrono; async exige concorrência real e lifecycle explícito.
- `dataclass`, Pydantic, Black e mypy só entram se a base já os usa ou se o requisito justificar; não force convenções externas.
- Nunca use `except:` ou capture amplo de `BaseException` para esconder falhas.
- `pathlib` e context managers são o padrão para paths e recursos; cleanup não depende de GC.
- Segredos (`secrets`) nunca entram em logs, tracebacks públicos, fixtures versionadas ou mensagens de erro.
- Não publique um modelo de dados ou um entry point sem confirmar o contrato do `pyproject.toml`.
- Não use o Python global, `PYTHONPATH` improvisado ou dependências globais para mascarar um ambiente mal configurado.
- Não adicione `requirements.txt`, lockfile, formatter ou type checker sem evidência de que o projeto os usa.
- Não converta todo I/O síncrono em async; bloquear o event loop é um bug de concorrência, não modernização.
- Não engula `CancelledError`, `KeyboardInterrupt` ou `SystemExit` para fazer a saída parecer limpa.
- Recursos externos, arquivos e tasks precisam de cleanup verificável, inclusive no caminho de exceção.
- Não faça uma refatoração de dependências, packaging ou estilo junto de um bug sem revisão separada do diff.
- Trate `except Exception` como fronteira consciente, não como padrão para esconder toda falha.
- Dê timeout explícito a chamadas externas; um timeout sem cancelamento e cleanup não é lifecycle completo.
- Preserve a ordem dos testes de integração e não confie em ordem de import para tornar o pacote inicializável.
- Não registre objetos de configuração inteiros: prefira campos redigidos e referências a variáveis seguras.
- Verifique a versão do interpretador usada pelos comandos, pois `python` e o Python do venv podem ser diferentes.
- Mantenha o contrato de importação e o entry point estáveis mesmo quando uma refatoração interna parecer mais simples.
- Consulte as skills de apoio sem duplicar seus fluxos: `context7-auto` para documentação atual, `test-driven-development` para RED/GREEN, `systematic-debugging` para causa raiz e `universal-test-runner` para suíte e cobertura.
