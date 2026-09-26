# Padrões de backend

## Feature flag

- Objetivo: ligar/desligar por **usuário operacional**, sem deploy e sem redeploy.
- Fonte da verdade: banco (auditado) ou serviço de flags; env var não serve (exige restart).
- Chave = nome legível; valor com default seguro; cache com TTL curto.
- **Kill switch** testado: o caminho de desligamento precisa ser mais fácil que o de ligar.
- Limpe flag morta: flag que ninguém lê vira condição permanente.

## Auth sem dependência nova

- JWT HS256 é ~30 linhas com `crypto` nativo (`createHmac` + `timingSafeEqual` com hash
  duplo) em Node. Uma lib para um algoritmo simples vira supply-chain e peso de install.
- Verificação: compare **hash** doHMAC, nunca string; rejeite `alg` que você não assinou
  (evita `alg: none`); expiração e audience sempre verificados; clock skew com folga.
- Senha: hash com custo (argon2/bcrypt), nunca reversível.

## Idempotência

- Endpoint de escrita que o cliente pode repetir precisa de chave de idempotência.
- Guarde a chave com o resultado; replay devolve o mesmo resultado, não duplica.
- Operações com efeito externo (pagamento, envio) exigem estado intermediário persistido.

## Paginação e volume

- Offset para página pequena e estável; cursor (`keyset`) para lista grande ou que muda.
- Limite máximo sempre; `count` total é caro em tabela grande — evite em listagem.
