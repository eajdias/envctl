# SQL e bancos

## Antes de escrever qualquer query

1. **Detecte o motor e o dialeto**: PostgreSQL, MySQL/MariaDB, SQLite, Mongo? `SELECT version()`.
2. Dialeto diferente muda sintaxe de upsert, tipos de data, JSON e DDL.
3. Inspecione o schema real (`\d`, `information_schema`) antes de supor coluna.

## Diagnóstico, não chute

- `EXPLAIN (ANALYZE, BUFFERS)` antes de otimizar; veja seq scan, rows estimado vs real.
- Índice: column que filtra, depois ordem; índice composto na ordem das colunas da query.
- Evite `SELECT *` em hot path: coluna explícita é cache de plano e banda.
- N+1 é a causa nº 1 de lentidão: faça `JOIN`/batch, não query por item.

## Migrar e transacionar

- Toda migração tem **down** ou é irreversível de forma explícita e documentada.
- Wrappe em transação quando o DDL permitir; `CREATE INDEX CONCURRENTLY` não.
- Deploy de coluna: adicione → backfill → leia → **só então** remova (3 deploys).

## Armadilhas que já custaram tempo

- **asyncpg/psycopg**: passar `date`/`datetime` como `str` quebra em silêncio; converta
  antes. `DataError` vem mascarado por fallback de query.
- Filtro de timestamp sem timezone: sempre `timestamptz` + UTC.
- Import em massa: **batch multi-VALUES** ou `COPY`; nunca `INSERT` por linha. Lote de
  500–5000 e commit por lote; em túnel SSH a diferença é de ordens de magnitude.

## Backup

Backup sem teste de restauração não é backup. `pg_dump` → restaure num banco descartável
e compare contagem de linhas.
