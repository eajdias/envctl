# Python

## Antes de escrever

- Versão e gerenciador do projeto (`pyproject.toml`, `requirements.txt`, `uv.lock`):
  respeite a versão declarada, não a do PATH.
- Use o ambiente do projeto (venv/`uv`), nunca a lib global da máquina.
- Descubra o runner de teste real antes de escrever teste: `pytest`, `unittest`?
  Config em `pyproject.toml`/`pytest.ini`/`setup.cfg`?

## Convenções

- Tipos: anotações em código público; `pyproject.toml` com `ruff` (lint+format) e um
  type checker (`mypy`/`ty`/`pyright`) — escolha o que o projeto já usa.
- `async` só com lib async de verdade; `await` em I/O, nunca em loop de CPU.
- Dataclass/`pydantic` para estrutura, dict para payload efêmero.
- Context manager (`with`) para recurso; `try/finally` quando não houver.
- Segredos: variável de ambiente ou arquivo de secrets, nunca no código.

## Armadilhas que já custaram tempo

- Concorrência com `ThreadPoolExecutor` para I/O, `ProcessPoolExecutor` para CPU.
- Path: `pathlib.Path`, não concatenação de string.
- Timestamps em UTC com timezone explícito; comparação de `date` vs `datetime` é
  fonte clássica de bug.
- Pydantic v2: `model_validate` para payload externo; coerção implícita é bug latente.
