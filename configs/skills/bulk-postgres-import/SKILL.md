---
name: bulk-postgres-import
description: Importar/upsert em massa no PostgreSQL de forma rápida. Use quando precisar inserir centenas/milhares de registros (cadastros, contatos, catálogo) vindos de API, CSV ou script, especialmente sobre túnel SSH/latência alta. Triggers: import, bulk, upsert, batch, insert em massa, ON CONFLICT, contatos, milhares de registros.
---

# Import em Massa no PostgreSQL (batch multi-VALUES)

## Quando usar

Inserção/atualização de 100+ registros. Em conexões com latência alta (túnel SSH, VPS remota), INSERT individual é inviável (~200ms cada — 22s/100).

## Passos

1. Dividir em batches de ~500 registros.
2. Montar um único `INSERT ... VALUES (...),(...),(...)` por batch.
3. Upsert com `ON CONFLICT (chave) DO UPDATE`:
   - Campos enriquecidos por fluxos posteriores: `COALESCE(tabela.campo, EXCLUDED.campo)` — **nunca sobrescrever** dados bons com dados vazios.
4. Merge de listas: `[...new Set([...novos, ...existentes])]` (dedupe).
5. Executar com timeout adequado (batch de 500 em ~4s é normal).

## Verificação

- `COUNT(*)` antes/depois bate com o total esperado.
- Registros enriquecidos (fluxo posterior) mantêm seus valores após re-import.
- Listas mescladas sem duplicatas (consulta com `jsonb_array_length`).