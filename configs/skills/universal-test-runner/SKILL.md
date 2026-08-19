---
name: universal-test-runner
description: Execução de testes unitários, integração, E2E e cálculo de cobertura de código em múltiplos ecossistemas (Node/TS, Python, Go, .NET, Rust). Use quando o usuário pedir para rodar testes, verificar regressões, medir cobertura ou validar código antes de finalizar. Triggers: test, testes, testar, coverage, cobertura, pytest, vitest, jest, go test, dotnet test, cargo test, tdd, benchmark.
---

# Universal Test Runner & Coverage Analysis

## Filosofia de Auto-Verificação
Antes de dar qualquer tarefa de implementação ou refatoração por concluída, execute o suite de testes correspondente para garantir que:
1. Novos testes cobrem a funcionalidade implementada.
2. Nenhum teste existente quebrou (regressão zero).
3. Linters e typechecks passam sem erros.

---

## 1. Node.js / TypeScript (Vitest, Jest, Playwright)

```bash
# Vitest
npx vitest run                           # Execução única
npx vitest run --coverage                # Com relatório de cobertura
npx vitest run path/to/file.test.ts      # Teste específico

# Jest / pnpm
pnpm test
pnpm test -- --coverage
pnpm test -- path/to/file.spec.ts

# Type-check antes dos testes
npx tsc --noEmit
```

---

## 2. Python (Pytest)

```bash
# Execução padrão
pytest -v

# Com cobertura de código
pytest --cov=src --cov-report=term-missing

# Testar arquivo ou função específica
pytest tests/test_auth.py
pytest tests/test_auth.py -k "test_jwt_login"

# Verificação estática e linter
ruff check . && ruff format --check .
mypy .
```

---

## 3. Go (go test)

```bash
# Todos os pacotes
go test ./... -v

# Cobertura detalhada
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out

# Pacote específico
go test ./pkg/auth -v -run TestAuthenticateUser

# Verificação estática
golangci-lint run
```

---

## 4. .NET / C# (dotnet test)

```bash
# Executar todos os testes da solution
dotnet test

# Verboso com log estruturado
dotnet test --logger "console;verbosity=detailed"

# Com coleta de cobertura
dotnet test --collect:"XPlat Code Coverage"

# Filtrar por classe ou método
dotnet test --filter "FullyQualifiedName~AuthServiceTests"
```

---

## 5. Rust (Cargo test)

```bash
# Todos os testes
cargo test

# Teste específico
cargo test test_user_creation -- --nocapture

# Clippy (linter estático do Rust)
cargo clippy -- -D warnings
```

---

## Práticas Recomendadas para o Agente
- **Isolamento de Erros:** Quando um teste falhar, isole o teste específico com a flag de filtro (`-k`, `-run`, `--filter`) para depurar com logs limpos antes de rodar a suite completa.
- **Evidência Antes de Conclusão:** Cole a linha de sumário do teste (ex: `Tests: 12 passed, 12 total` ou `100% passed`) para demonstrar sucesso.
