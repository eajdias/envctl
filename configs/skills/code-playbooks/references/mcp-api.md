# MCP e contratos de API

## Tool MCP

- `description` é documentação, não propaganda: propósito, quando usar, quando **não** usar,
  semântica dos argumentos, limites e efeito observável.
- `inputSchema` obrigatório, raiz é objeto; tool sem parâmetro = objeto vazio com
  `additionalProperties: false`.
- Rejeite valor ambíguo em vez de inventar default; declare unidade, limite e formato.
- Escolha de primitiva pelo controle: `tool` (o modelo decide), `resource` (a aplicação
  injeta), `prompt` (o usuário inicia).
- Efeito colateral e consentimento: escrita destrutiva pede confirmação explícita.
- Paginação e limite de resultado: sem isso, uma tool "lista" estoura o contexto.
- `outputSchema` quando o consumidor precisa validar o resultado; e declare erro.
- Não esconda ação automática (deploy, delete) dentro de uma tool que parece de leitura.

## Contrato de API

- Especificação é o artefato: OpenAPI 3.x, GraphQL SDL, gRPC/Protobuf, REST.
- Versionamento no **caminho** (`/v1`) é o mais previsível; breaking change = versão nova.
- Campo opcional precisa de semântica de ausência definida (ausente ≠ null ≠ vazio).
- Erro: formato único, com código estável e mensagem acionável; 4xx é do cliente, 5xx é seu.
- Depreciação: marcar, avisar, remover em data — nunca sumir em silêncio.
- Compatibilidade: campo novo é seguro; remover/retypesar/obrigatório é breaking.
