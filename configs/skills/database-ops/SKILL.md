---
name: database-ops
description: Operações dinâmicas com múltiplos bancos de dados (PostgreSQL, MySQL, Firebird, MongoDB, SQLite, Redis), migrações, containers Docker e consultas. Use quando o usuário pedir para inspecionar schemas, rodar migrations, validar queries, fazer backup/restore, manipular dados ou subir bancos em Docker. Triggers: database, banco, banco de dados, sql, postgres, postgresql, mysql, firebird, mongodb, mongo, sqlite, redis, migration, migrations, prisma, drizzle, alembic, efcore, schema, query.
---

# Operações com Múltiplos Bancos de Dados (Dinâmico & Multi-projeto)

## Contexto & Filosofia
- **Dinâmico por Projeto:** Em vez de conexões estáticas globais, as operações adaptam-se ao contexto do projeto atual (`.env`, `docker-compose.yml`, configurações de ORM).
- **Suporte Multi-Engine:** PostgreSQL, MySQL, Firebird, MongoDB, SQLite, Redis.
- **Isolamento em Docker:** Preferência por rodar instâncias de bancos via Docker Compose local ou conectar em serviços remotos autenticados.

## Detecção do Banco de Dados no Projeto Atual

Antes de qualquer operação, inspecione a raiz do repositório:
1. **Verificar ORMs/Migrators:**
   - Node/TS: `prisma/schema.prisma`, `drizzle.config.ts`, `typeorm`, `knexfile.js`, `mikro-orm`
   - Python: `alembic.ini`, `manage.py` (Django), `tortoise`, `peewee`
   - .NET: `*.csproj` com Entity Framework Core (`dotnet ef`)
   - Go: `golang-migrate`, `gorm`, `sqlc`, `ent`
2. **Verificar Variáveis de Conexão:**
   - Ler `.env` ou `.env.example` para identificar strings de conexão (`DATABASE_URL`, `POSTGRES_DB`, `MONGO_URI`, etc.).

---

## 1. Operações via Docker Compose (Padrão Recomendado)

Subir e checar containers de banco de dados locais:
```bash
# Subir apenas o serviço de banco
docker compose up -d db
# ou
docker compose up -d postgres

# Verificar logs e prontidão (ready to accept connections)
docker compose logs --tail 50 postgres
```

---

## 2. Execução Rápida de Queries e Inspeção via CLI / Container

### PostgreSQL
```bash
# Executar query direta via container Docker
docker compose exec -T postgres psql -U postgres -d <dbname> -c "\dt"
docker compose exec -T postgres psql -U postgres -d <dbname> -c "SELECT count(*) FROM users;"

# Via cliente local se disponível
psql "$DATABASE_URL" -c "\d"
```

### MySQL / MariaDB
```bash
# Listar tabelas e schemas via container
docker compose exec -T mysql mysql -u root -p<senha> <dbname> -e "SHOW TABLES;"
```

### MongoDB
```bash
# Inspecionar collections e índices via mongosh no container
docker compose exec -T mongo mongosh --eval "db.getCollectionNames()"
```

### SQLite
```bash
# Inspecionar banco local SQLite
sqlite3 ./dev.db ".tables"
sqlite3 ./dev.db ".schema users"
sqlite3 ./dev.db "SELECT * FROM users LIMIT 5;"
```

### Firebird
```bash
# Executar script ou consulta via isql-fb / container
isql-fb -user SYSDBA -password <senha> localhost:/path/to/database.fdb -i query.sql
```

---

## 3. Execução e Validação de Migrações por Ecossistema

### Prisma (Node / TS)
```bash
# Visualizar status de migrações
npx prisma migrate status

# Gerar e aplicar nova migração em desenvolvimento
npx prisma migrate dev --name <nome_da_migracao>

# Gerar Prisma Client
npx prisma generate
```

### Drizzle ORM (Node / TS)
```bash
# Gerar arquivos SQL a partir do schema TS
npx drizzle-kit generate
# Aplicar migrações no banco
npx drizzle-kit migrate
```

### Alembic (Python / SQLAlchemy / FastAPI)
```bash
# Gerar migração automática
alembic revision --autogenerate -m "add_user_status_column"
# Aplicar migrações
alembic upgrade head
```

### Entity Framework Core (.NET / C#)
```bash
# Adicionar e aplicar migração
dotnet ef migrations add AddUserProfile
dotnet ef database update
```

### Go (golang-migrate)
```bash
migrate -path ./migrations -database "$DATABASE_URL" up
```

---

## Regras de Segurança e Integridade
1. **NUNCA executar `DROP DATABASE`, `DROP TABLE` ou `TRUNCATE` em ambientes produtivos.**
2. Em desenvolvimento, sempre avise o usuário antes de resetar bancos de dados (`prisma migrate reset` ou `docker volume prune`).
3. Nunca comitar arquivos de dump contendo dados pessoais ou senhas reais.
4. Para backups rápidos de segurança antes de migrações complexas:
   - PostgreSQL: `docker compose exec -T postgres pg_dump -U postgres <dbname> > backup_pre_migration.sql`
   - MySQL: `docker compose exec -T mysql mysqldump -u root -p<senha> <dbname> > backup_pre_migration.sql`
