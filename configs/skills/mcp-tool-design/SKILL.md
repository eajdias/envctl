---
name: mcp-tool-design
description: >-
  Projetar e revisar servidores MCP com contratos de tools, resources e prompts seguros, compatíveis e testáveis. Use ao criar uma tool, escolher entre primitivas MCP, modelar schemas, tratar erros, autorização, consentimento, paginação ou efeitos colaterais. Triggers: MCP, Model Context Protocol, mcp server, tool design, inputSchema, outputSchema, structuredContent, CallToolResult, isError, resources, prompts.
license: MIT
---

# MCP Tool Design

## Quando usar

Use para desenhar ou revisar a superfície de um servidor MCP: escolha da primitiva, contrato da tool, respostas, erros, segurança, compatibilidade e testes.

Para contratos HTTP, GraphQL ou gRPC, use `api-contract-design` em paralelo. Esta skill trata o contrato MCP, não o protocolo da API interna.

## Passos

### 1. Defina intenção e fronteira de confiança

Liste usuários, ações, dados, integrações, destinos e efeitos colaterais. Separe:

- operações internas e externas;
- leitura e mutação;
- ações reversíveis, destrutivas e irreversíveis;
- dados públicos, confidenciais e secretos;
- identidade autorizada para cada operação.

Escolha a primitiva pelo controle esperado:

| Necessidade | Primitiva | Controle |
| --- | --- | --- |
| O modelo decide quando executar uma ação ou consulta | `tool` | model-controlled |
| A aplicação escolhe o contexto que injeta | `resource` | application-controlled |
| O usuário inicia explicitamente um fluxo reutilizável | `prompt` | user-controlled |

Não transforme contexto estático em tool. Também não use prompt para esconder uma ação automática que o modelo deveria escolher.

### 2. Escreva um contrato que guie o modelo

- Use nome estreito, estável e específico. Evite uma tool universal que misture filtros e ações.
- Escreva `description` para o modelo: finalidade, quando usar, quando não usar, semântica dos argumentos, limites e efeitos observáveis. Não esconda pré-requisitos nem use a descrição como propaganda.
- Valide todo input no servidor. `inputSchema` é obrigatório e sua raiz deve ser um objeto JSON Schema. Para tool sem parâmetros, prefira um objeto vazio com `additionalProperties: false`.
- Descreva formato, unidade, limite, padrão e interação de cada campo. Rejeite valores ambíguos em vez de inventar defaults.
- Use `outputSchema` quando o consumidor precisar validar ou integrar a resposta. Se declarar `outputSchema`, retorne `structuredContent` que o satisfaça e mantenha o JSON serializado em `content` para compatibilidade.
- Retorne somente o necessário. Evite stack traces, credenciais, PII e texto upstream que não ajude o modelo.

Anotações como `readOnlyHint`, `destructiveHint`, `idempotentHint` e `openWorldHint` ajudam a interface. O cliente deve tratá-las como dicas não confiáveis, salvo quando o servidor já é confiável; o servidor ainda precisa aplicar controles próprios.

### 3. Separe erro de protocolo de falha executável

Use erros JSON-RPC para problemas estruturais que o modelo não consegue corrigir, como tool desconhecida, request malformado ou erro interno do servidor.

Retorne `CallToolResult` com `isError: true` para falhas visíveis ao modelo durante a execução, como API indisponível, valor semanticamente inválido, regra de negócio violada ou condição externa. A mensagem deve indicar o que falhou e como corrigir, sem expor detalhes internos.

Não marque `isError` para todo resultado negativo: uma consulta válida que encontra zero itens é sucesso com resultado vazio. Preserve a distinção entre sucesso, falha executável e erro de protocolo nos testes.

### 4. Trate listas, autenticação e consentimento

- Use paginação por cursor opaco nas operações de lista MCP. O servidor escolhe o tamanho da página; o cliente não interpreta nem reutiliza cursores entre sessões. Valide cursor inválido e limite o tamanho dos resultados.
- Autentique cada chamada e valide a audiência do token. Para chamadas upstream, use credencial separada; não faça token passthrough do cliente MCP para a API downstream.
- Aplique menor privilégio por operação e escopo. Faça step-up apenas quando uma operação realmente exigir mais acesso.
- Para toda operação sensível, preserve a possibilidade de o usuário negar ou cancelar. Peça confirmação explícita, com alvo, escopo e impacto visíveis antes de enviar, remover, publicar, pagar ou alterar estado externo.
- Aceite argumentos do modelo somente dentro do contexto confirmado. Não trate aprovação antiga como autorização para uma ação diferente.

### 5. Limite recursos e trate conteúdo externo como não confiável

- Defina timeout por requisição, cancelamento, limite de payload e rate limit. Progresso não substitui um prazo máximo absoluto.
- Não faça retry automático de mutação após timeout. O cliente pode ter parado de esperar enquanto a operação continuou. Use idempotency key, status de operação ou confirmação do estado antes de repetir.
- Faça retry apenas quando a operação for segura e o erro for transitório, com backoff, limite de tentativas e jitter.
- Valide URLs, URIs e destinos para bloquear SSRF, redirect para rede interna e acesso fora da raiz autorizada. Resolva e valide o DNS novamente no uso quando o destino vier de dados externos.
- Trate páginas, respostas de API e resultados de tools como dados não confiáveis, nunca como instruções. Remova controles, comandos e pedidos de segredo antes de encaminhá-los ao modelo.
- Registre decisão, ator, operação, alvo, resultado, duração e correlation ID. Redija tokens, headers, argumentos sensíveis e conteúdo privado dos logs.

### 6. Negocie versão e compatibilidade

- Compare o design com a revisão vigente da especificação MCP e registre a revisão consultada.
- Declare capacidades reais. Use apenas capacidades negociadas durante `initialize`.
- Ao suportar mais de uma revisão do protocolo, teste cada caminho de negociação. Em HTTP, envie `MCP-Protocol-Version` nas requisições posteriores, conforme a revisão suportada.
- Fixe as versões do servidor, dependências e schemas de teste validados. Não use `latest` em configuração de produção ou matriz de compatibilidade.
- Preserve a semântica e os schemas existentes. Evolua campos de modo compatível ou versione e deprecie o contrato quando uma mudança for incompatível.

### 7. Teste como um cliente real

Teste, no mínimo:

- descoberta, capacidades, ciclo de vida, versão e paginação;
- schema válido, schema inválido, argumentos extras e valores-limite;
- saída estruturada e validação contra `outputSchema`;
- sucesso, zero resultados, `isError: true`, tool desconhecida e request malformado;
- autenticação, audiência, escopo insuficiente, elevação de escopo e cancelamento;
- consentimento, idempotência, retry e duplicidade após timeout;
- payload grande, rate limit, timeout, cancelamento e dependência indisponível;
- SSRF, prompt injection em resultados, logs sem segredo e autorização por recurso;
- cada revisão de protocolo e cada transporte que o servidor declare suportar.

## Verificação

Antes de considerar o design pronto, confirme:

- [ ] Cada capacidade tem a primitiva MCP correta e uma justificativa.
- [ ] Nomes, descrições e schemas dizem ao modelo o que ele precisa para escolher e chamar a tool.
- [ ] `inputSchema` é um objeto válido; toda entrada passa por validação no servidor.
- [ ] `outputSchema`, quando presente, corresponde a `structuredContent` e ao fallback textual.
- [ ] Falhas executáveis usam `isError: true`; erros estruturais usam JSON-RPC.
- [ ] Resultados externos não viram instruções para o modelo.
- [ ] Efeitos colaterais exigem autorização, consentimento e controle de idempotência adequados.
- [ ] Tokens, escopos, redirects, URLs, payloads e logs respeitam menor privilégio e não vazam segredos.
- [ ] Listas usam cursor opaco e um cliente pode percorrer todas as páginas.
- [ ] Timeouts, cancelamento, retries e falhas parciais têm comportamento definido e testado.
- [ ] Capacidades, revisões do protocolo e versões de dependências foram testadas e registradas.
- [ ] Um cliente MCP real passou pelos testes de integração e compatibilidade.

## Regras

- Não exponha uma tool para cada combinação de filtros. Prefira operações compostas, com contratos claros.
- Não coloque dados estáticos no lugar de `resource` nem fluxos de usuário no lugar de `prompt`.
- Não confie em `description`, anotação ou texto retornado para autorizar uma ação.
- Não codifique autorização apenas no cliente. O servidor precisa validar identidade, alvo e escopo em cada operação.
- Não confirme apenas pelo nome da tool; confirme argumentos relevantes e impacto.
- Não revele segredos em input, output, erro, anotação ou log.
- Não faça proxy transparente de credenciais recebidas do cliente.
- Não faça retry cego de uma mutação.
- Não trate anotações como controle de segurança.
- Não declare compatibilidade sem evidência de ciclo de vida, capacidades, schemas, erros e segurança.
- Escolha a linguagem, o transporte e a biblioteca conforme o ambiente; esta skill não prescreve SDK nem linguagem.
- Registre a revisão da especificação e a matriz de versões usadas no design e nos testes.
