---
name: api-contract-design
description: Design, especificação, validação e documentação de contratos de API (OpenAPI/Swagger 3.x, GraphQL SDL, gRPC/Protobuf, REST). Use quando o usuário pedir para desenhar endpoints, validar schemas, gerar documentação de API, criar contratos de integração ou verificar breaking changes. Triggers: openapi, swagger, api contract, contrato api, endpoint, endpoints, rest api, graphql, schema, protobuf, grpc, spectral, redoc.
---

# API Contract Design & Schema Validation

## Contexto & Padrões
- **OpenAPI 3.1 / 3.0:** Padrão para REST APIs (JSON/YAML).
- **GraphQL:** Padrão SDL (`schema.graphql`) para APIs orientadas a grafos.
- **Protocol Buffers (Protobuf v3):** Padrão para gRPC e comunicação inter-serviços.

---

## 1. OpenAPI / Swagger Design

### Regras de Qualidade para OpenAPI
1. **Tipagem Estrita:** Sempre definir `type`, `format` (ex: `uuid`, `date-time`, `email`), `required` e `example`.
2. **Respostas Padronizadas:**
   - Sucesso: `200 OK`, `201 Created`, `204 No Content`.
   - Erros: `400 Bad Request`, `401 Unauthorized`, `403 Forbidden`, `404 Not Found`, `422 Unprocessable Entity`, `500 Internal Server Error`.
   - Formato de erro consistente (RFC 7807 Problem Details ou envelope padrão `{ error: { code, message, details } }`).
3. **Segurança:** Declarar explicitamente os esquemas de autenticação (`BearerAuth`, `ApiKeyAuth`, `OAuth2`).

### Validação e Linting de OpenAPI
```bash
# Validar arquivo OpenAPI com spectral (via npx)
npx @stoplight/spectral-cli lint openapi.yaml
# ou
npx @redocly/cli lint openapi.yaml

# Gerar documentação estática ou preview local
npx @redocly/cli preview-docs openapi.yaml
```

---

## 2. GraphQL Schema Design (SDL)

### Padrões
- Queries declarativas e Mutations atômicas.
- Inputs explícitos com `input <Type>Input`.
- Paginação padronizada (Relay Connection ou Cursor-based).

### Validação de GraphQL
```bash
# Validação e checagem de schema
npx graphql-schema-linter schema.graphql
```

---

## 3. Protocol Buffers (gRPC)

### Estrutura Recomendada (`.proto`)
- `syntax = "proto3";`
- Nomenclatura CamelCase para mensagens e snake_case para campos.
- Pacotes com versionamento semântico (ex: `package api.v1;`).

```bash
# Linting de Protobuf com buf (se disponível via npx/binário)
npx @bufbuild/buf lint
```

---

## Detecção de Breaking Changes
Antes de alterar contratos existentes:
1. Nunca renomear ou remover campos sem período de depreciação (`deprecated: true`).
2. Adicionar novos campos sempre como opcionais (não-obrigatórios).
3. Alterações drásticas de contrato devem incrementar a versão da API (`/api/v2/`).
