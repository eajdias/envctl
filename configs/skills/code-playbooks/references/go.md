# Go

## Antes de escrever

- Leia `go.mod`/`go.sum`: módulo, diretiva `go`, dependências, versão mínima.
- Procure `go.work`, `go.work.sum` e build tags (`_GOOS`, `_GOARCH`).
- Confirme `go version`, `go env GOOS GOARCH CGO_ENABLED` antes de comparar resultado.
- Leia os comentários das APIs exportadas: eles dizem o contrato que o nome esconde.
- Se `CGO_ENABLED` ou `//go:embed` estiver em jogo, a portability é um gate próprio.

## Convenções

- `error` sempre envolvido com contexto (`fmt.Errorf("...: %w", err)`) — sem `errors.New`
  quando existe uma causa para preservar.
- Erro de domínio é sentinela (`var ErrNaoEncontrado = errors.New(...)`) para o caller
  testar com `errors.Is`.
- Concorrência: `context.Context` como primeiro parâmetro, sempre; `errgroup` ou
  `sync.WaitGroup` + canal de erro (nunca `WaitGroup.Wait` com `err` compartilhado sem
  mutex); `sync.Once` para lazy init; nada de goroutine sem dono do cancelamento.
- Portabilidade: build tags para comportamento por SO; nada de Assume `C:\` ou `/tmp`.
- `gofmt` é lei; `go vet` antes de chamar de pronto.

## Gate

```bash
go build ./... && go vet ./... && go test ./...
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build ./cmd/envctl   # se toca em provisionador
```

Mudança de comportamento sem teste atualizado é dívida, não detalhe.
