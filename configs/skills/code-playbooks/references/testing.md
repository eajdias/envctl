# Testes

## O comando que prova o quê

Antes de afirmar qualquer coisa, identifique **qual comando prova aquele claim**:

| Claim | Comando |
|---|---|
| "compila" | `go build ./...` |
| "passa no vet/lint" | `go vet ./...` / `ruff check` / `tsc --noEmit` |
| "os testes passam" | o runner real do projeto (`go test ./...`, `pytest`, `npm test`) |
| "sem regressão" | suite completa, não só o arquivo tocado |
| "funciona no navegador" | E2E (Playwright), não teste unitário |

## Comandos por ecossistema

```bash
go test ./... && go test -race ./... && go test -cover ./...
pytest -q && pytest --cov
npm test -- --coverage        # ou: bun test, pnpm test
dotnet test && cargo test
```

## Regras

- Teste novo **antes** da implementação (ciclo red-green-refactor), se a mudança for de
  comportamento.
- Falha intermitente quase sempre é estado compartilhado, ordem de teste ou clock.
- Cobertura é mapa, não objetivo: linha coberta que não testa nada não vale.
- Teste que depende de rede/hora real é teste quebrado: injete relógio e use container.
- Saída de processo: signal nunca é exit 0 — trate 128+N como falha.
