---
name: go-development
description: >-
  Desenvolver, revisar e depurar aplicações Go idiomáticas, concorrentes e portáteis. Use ao criar ou alterar código Go, APIs, módulos, goroutines, canais, mutexes ou testes. Triggers: go, golang, go.mod, go.sum, go test, go vet, gofmt, race detector, error wrapping, context, goroutine, deadlock, data race, cross-compilation Windows ou Linux.
license: MIT
---

# Go Development

## Quando usar

- Implementar, corrigir, revisar ou refatorar código Go.
- Criar APIs, integrações de I/O, bibliotecas ou comandos.
- Trabalhar com concorrência, cancelamento, canais, mutexes ou WaitGroup.
- Investigar build, `vet`, testes, race detector ou incompatibilidade de plataforma.
- Preparar um módulo ou binário para Linux e Windows.

## Contexto

- Comece lendo `go.mod` e `go.sum`; confirme módulo, diretiva `go`, dependências e a versão mínima.
- Procure `go.work`, `go.work.sum` e arquivos com build tags ou variantes por `_GOOS`/`_GOARCH`.
- Inspecione a estrutura de pacotes, os consumidores reais, os testes e os padrões já usados no repositório.
- Verifique CI, Makefile, task runner e scripts para aprender os comandos oficiais, GOOS e GOARCH.
- Observe como o código separa domínio, adaptadores, I/O e apresentação antes de criar uma nova abstração.
- Leia a documentação da biblioteca quando uma API, flag ou detalhe de versão não for claro.
- Consulte `context7-auto` antes de usar documentação atual de uma dependência.
- Para mudança de comportamento, use `test-driven-development`; para falha, `systematic-debugging`; para suíte completa, `universal-test-runner`.
- Confirme `go version`, `go env GOOS GOARCH CGO_ENABLED` e a versão efetiva do toolchain antes de comparar um resultado.
- Leia comentários das APIs exportadas, benchmarks e exemplos de uso; eles revelam contratos que o nome do tipo não mostra.
- Verifique código gerado e scripts de geração: gerar arquivos é uma mutação separada e deve ter revisão própria.
- Diferencie o módulo principal de workspaces e pacotes experimentais; não use uma dependência de outro diretório por conveniência.
- Leia o `README` e exemplos de uso para distinguir API pública de detalhe interno antes de mantê-la estável.
- Verifique se o projeto usa código gerado, `//go:embed`, CGO ou syscall; cada um tem um gate de portabilidade próprio.
- Confirme se testes usam `-tags`, variáveis de ambiente ou serviços externos antes de escolher um comando de build paralelo.

## Passos

1. **Enquadre o problema.** Leia o contrato existente, o consumidor real e o comportamento observável. Defina também qual erro o chamador deve conseguir classificar.
2. **Respeite o módulo.** Não mude a versão da linguagem, dependências ou formato do `go.mod` como efeito colateral. Confirme a necessidade de cada biblioteca nova.
3. **Prefira a stdlib.** Use a biblioteca padrão quando ela resolver o caso e mantenha o código próximo dos padrões dos pacotes vizinhos. Uma dependência adicional precisa justificar custo, manutenção e compatibilidade.
4. **Desenhe a interface no consumidor.** Mantenha interfaces pequenas, no lado que consome o comportamento, com os métodos realmente necessários. Prefira tipos concretos no retorno e não crie interfaces espelho da stdlib ou para testá-las artificialmente.
5. **Modele erros explicitamente.** Use `errors.New` para sentinelas, tipos de domínio quando houver dados, e `fmt.Errorf("...: %w", err)` para preservar a causa. O chamador deve classificar com `errors.Is` ou `errors.As`, nunca por string.
6. **Registre erros no limite certo.** Acrescente contexto local sem duplicar o mesmo log em cada camada. Mantenha o erro original disponível para retry, diagnóstico e decisão de política.
7. **Use `context.Context` com suporte real.** Só adicione o parâmetro quando a API e as operações subjacentes suportarem deadline, cancelamento ou valores. Propague-o para chamadas compatíveis e não o armazene em structs para reutilização posterior.
8. **Defina o lifecycle de cada goroutine.** Para cada `go func`, registre o owner, o início, o sinal de stop, o cleanup e a espera com `sync.WaitGroup` ou `errgroup`. O cancelamento do contexto não mata uma goroutine: o código precisa observar `ctx.Done()`, fechar recursos e retornar.
9. **Escolha sincronização pela posse do estado.** Se um único owner possui o estado, prefira canais (`channel`) para transferir mensagens e posse. Se vários chamadores leem ou escrevem o mesmo estado, use mutex para uma região curta e definida. Evite misturar os dois modelos sem necessidade.
10. **Planeje portabilidade.** Use caminhos e APIs portáveis, arquivos de plataforma e build tags explícitos. Se o repositório mirar Windows, valide também o cross-build com a arquitetura declarada no CI.
11. **Trate `go mod tidy` como mutação.** Ele pode alterar `go.mod` e `go.sum`; rode somente quando a atualização for intencional, revise o diff e nunca o trate como verificação.
12. **Escolha quem fecha canais.** Quem cria um canal deve documentar e controlar seu fechamento; não feche um canal pertencente a outro componente.
13. **Evite estado global implícito.** Quando estado precisar ser compartilhado, deixe o owner explícito e limite a região protegida; não esconda singletons atrás de funções sem contrato.
14. **Escreva testes para o lifecycle.** Além do resultado, prove cancelamento, fechamento de recursos e ausência de goroutine pendente quando isso fizer parte da API.
15. **Preserve o padrão do repositório.** Use `errgroup`, benchmarks ou helpers existentes quando forem mais adequados; não introduza uma biblioteca de concorrência para um caso local e simples.

## Verificação

- Formate os arquivos alterados com `gofmt` e confirme que `gofmt -l .` não aponta arquivos inesperados.
- Rode `go build ./...` para provar que todos os pacotes compilam.
- Rode `go vet ./...` para aplicar as análises estáticas do Go.
- Rode `go test ./...` ou os alvos e scripts de teste definidos pelo projeto.
- Use `go test -race ./...` somente para código concorrente relevante e quando a plataforma/arquitetura suportar o race detector; registre a limitação em ambientes incompatíveis.
- Rode o lint próprio do projeto quando existir; não substitua o gate configurado por um comando inventado.
- Se o repositório tiver alvo Windows, rode `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build ./...` ou a combinação exata do CI. Ajuste CGO apenas quando a configuração real exigir.
- Confira separadamente o diff de `go.mod` e `go.sum`; mudanças de dependência não contam como build, vet ou teste.
- Se a mudança for de concorrência, rode testes focados que também provem stop e wait, não apenas o caminho feliz.
- Use `-count=1` quando precisar excluir o cache de testes e `-run` para isolar uma regressão antes da suíte completa.
- Rode as tags de build que o projeto usa em CI; sem elas, um arquivo específico da plataforma pode escapar da validação.
- Confirme que testes de integração que dependem de rede, Docker ou serviços externos foram isolados ou explicitamente ignorados.
- Verifique `git diff --check` para detectar whitespace inválido nos arquivos Go alterados.
- Se o projeto publicar um binário, teste também o artefato final, não apenas os pacotes internos.

## Regras

- Biblioteca padrão e clareza vêm antes de abstração ou dependência.
- Interfaces são pequenas, ficam no consumidor e não existem só para testá-las.
- Erros carregam contexto útil e preservam a causa por `%w`, `errors.Is` e `errors.As`.
- `context.Context` pertence à operação; nunca guarde contexto em uma struct de longa duração.
- Toda goroutine tem owner, stop, cleanup e wait observáveis; não deixe `go` fire-and-forget.
- Um canal é a escolha natural quando coordena posse ou переда mensagens; um mutex protege estado compartilhado curto.
- Não use `time.After` como cronômetro de lifecycle quando um ticker, timer ou contexto controlado resolve o caso.
- Não troque `go mod tidy` por uma falsa sensação de verificação: build, vet e test são gates separados.
- Para cross-platform, não assuma que o build do host prova o alvo; use as variáveis e tags do CI.
- Quem produz um canal ou fecha um recurso deve ser explícito; consumidores não devem adivinhar seu lifecycle.
- Não copie uma struct que contenha `sync.Mutex`, `sync.RWMutex` ou outro estado de sincronização.
- Não crie `context.Background()` no meio da cadeia para “resolver” uma API sem contexto; ajuste a fronteira da API.
- Comentários de APIs exportadas devem dizer o contrato, os erros relevantes e a política de cancelamento quando existir.
- Evite `time.Sleep` para sincronizar produção e consumo; use canal, condição ou outro sinal do owner.
- Um teste de happy path não comprova concorrência segura; race detector e testes de lifecycle são complementares.
- Mantenha o cancelamento cooperativo: `ctx.Done()` é um sinal, não uma chamada de encerramento.
- Se uma goroutine ignorar o contexto, documente a shutdown explícita e garanta que o owner ainda consiga pará-la e aguardá-la.
- Não exponha canais de escrita ou detalhes de implementação que transformem uma API local em contrato global.
- Consulte, sem duplicar seus procedimentos, `context7-auto` para docs atuais, `test-driven-development` para RED/GREEN, `systematic-debugging` para causa raiz e `universal-test-runner` para suíte e cobertura.
