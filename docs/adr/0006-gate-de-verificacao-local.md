# ADR 0006: Gate de verificação local (envctl-verify)

## Status
Aceito.

## Contexto
Havia dois problemas recorrentes:

- Uma mudança podia escapar sem checagem até o CI — o erro aparecia **minutos
  depois**, num contexto distante; e, com agentes de IA escrevendo código, o
  feedback precisa vir no mesmo turno para o erro não se propagar.
- O `git push` não tinha trava local: falhas óbvias (lint, teste quebrado)
  chegavam ao remoto e viravam custo de CI/PR.

Além disso, o repositório usa múltiplos hooks de git (husky local em projetos
terceiros, hooks do próprio repo), e um hook global podia atropelá-los.

## Decisão
- **Um verificador único** — `configs/bin/envctl-verify` (deployado em
  `~/.local/bin`, embutido no binário) — com três modos:
  - `--hook`: subconjunto **estático** (o que um editor diria ao salvar), para
    fim de turno do agente;
  - `--git-push`: gate **completo**, com a suíte de testes;
  - `--dry-run`: mostra os checks detectados e as severidades, sem rodar.
- **Escopo por tipo de check:** linters/formatadores rodam **só nos arquivos
  que a mudança toca**; type checks e testes rodam **no repositório inteiro**
  (é ali que está o sinal). Mesmo princípio do `--new-from-rev` do
  golangci-lint; `gofmt -l .` é a exceção (operação de módulo).
- **Severidade explícita:** lint/formatação inferidos são **advisory**;
  scripts explícitos do projeto, builds, vets e testes são **blocking**.
  Advisory aparece e registra; blocking aborta o push.
- **Dois pontos de ancoragem:** hook `Stop` do CommandCode (feedback por turno)
  e `pre-push` global do git via `core.hooksPath` — com um **delegator** que
  devolve o controle aos hooks locais do projeto (`pre-commit`,
  `commit-msg`, `post-commit`, `post-checkout`, `pre-rebase`). Husky local
  vence o global por design do próprio git.
- **CI espelha o gate:** os mesmos checks rodam no CI (lint + test nos dois
  OS); manter os dois em sincronia é responsabilidade explícita.

## Consequências
- Erro aparece em segundos no terminal; o push é bloqueado só por falha real.
- O script é contrato com o agente — tem testes próprios
  (`internal/usecase/verify_script_test.go`) que precisam continuar passando.
- O modo `--hook` gateia o **cwd da sessão**: abrir o agente dentro do projeto
  é o que ativa o feedback automático naquele repo.
