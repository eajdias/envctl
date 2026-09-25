---
name: database-ops
description: >-
  Operações com bancos de dados em projetos locais, containers e ambientes remotos: PostgreSQL, MySQL, MariaDB, Firebird, MongoDB, SQLite e Redis. Use para detectar motor e dialeto, inspecionar schema, analisar consultas com EXPLAIN, projetar índices, criar constraints, executar migrações, transacionar, fazer backup/restore e subir bancos com Docker. Triggers: database, banco, banco de dados, sql, postgres, postgresql, mysql, mariadb, firebird, mongodb, mongo, sqlite, redis, migration, migrations, prisma, drizzle, alembic, efcore, schema, query, explain, índice, index, foreign key, transação, backup, restore.
license: MIT
---

# Operações com bancos de dados

## Quando usar

Use esta skill quando precisar investigar ou alterar um banco sem perder a noção de qual motor, versão, schema, cliente e ambiente estão ativos. Ela cobre PostgreSQL, MySQL, MariaDB, Firebird, MongoDB, SQLite e Redis; não trate um comando de outro motor como equivalente automático.

Comece por leitura. Operações que escrevem dados, alteram schema, derrubam bloqueios, restauram backups ou reiniciam containers exigem contexto, plano e confirmação explícita quando houver impacto.

## 1. Descobrir o projeto e o destino

Inspecione a raiz do repositório sem imprimir valores de conexão:

1. Procure a configuração do ORM, driver e migrador: `prisma/schema.prisma`, `drizzle.config.ts`, `knexfile.js`, `alembic.ini`, `manage.py`, arquivos de migração, `sqlc`, GORM ou outro usado pelo projeto.
2. Leia apenas nomes de variáveis e serviços de `.env.example`, Compose e CI. Não copie senha, token, URL completa ou conteúdo de `.env` para a resposta.
3. Identifique se o banco é local, um container do projeto ou um serviço remoto. Use a referência de conexão já configurada pelo projeto ou um inventário local seguro.
4. Confirme o escopo: desenvolvimento, homologação ou produção. Em produção, faça backup e valide a janela de manutenção antes de DDL, restore ou mudança de credencial.

### Checklist antes de escrever

- Motor, versão do servidor, versão do cliente e dialeto foram confirmados.
- Banco, schema e tabela alvo estão explícitos.
- A sessão usa a origem de credencial aprovada, com segredo fora do repositório e do argv.
- Há backup ou uma forma de reverter a mudança.
- O plano identifica locks, downtime, consumidores afetados e como verificar o resultado.

## 2. Operações via Docker Compose

Descubra o nome real do serviço no Compose; pode ser `db`, `postgres`, `mysql` ou `mariadb`.

```bash
docker compose config --services
docker compose up -d <servico-de-banco>
docker compose ps
docker compose logs --tail 80 <servico-de-banco>
```

`up` altera o estado. Aguarde o healthcheck ou um probe real de conexão; o status `running` não garante que o banco aceitou consultas. Não use `down -v`, `volume prune` ou `restart` para mascarar um problema sem antes registrar o impacto.

## 3. MySQL e MariaDB

### Detectar versão, variante e dialeto (dialect)

Não deduza MySQL versus MariaDB pelo nome da imagem, pelo driver ou por uma resposta antiga. Consulte o servidor:

```sql
SELECT VERSION() AS server_version,
       @@version_comment AS version_comment,
       @@sql_mode AS sql_mode;
SHOW VARIABLES LIKE 'version%';
SHOW CREATE DATABASE <database>;
```

O cliente também deve ser verificado:

```bash
mysql --version
mariadb --version
```

Se os dois clientes estiverem instalados, use o que for compatível com o driver do projeto. Em uma imagem MariaDB, prefira `mariadb` e `mariadb-dump` quando eles forem os clientes fornecidos; o cliente `mysql` pode funcionar, mas não deve esconder diferenças de versão ou dialeto. `@@version_comment` e os recursos aceitos pelo servidor ajudam a distinguir a variante; o resultado não deve ser transformado em uma promessa de que outra versão se comporta igual.

Para SQL que depende de recursos, teste `EXPLAIN FORMAT=JSON` e consulte `SHOW CREATE TABLE`, `SHOW INDEX` e `information_schema`. `utf8mb4`, collations, funções de janela, JSON, DDL online e `EXPLAIN ANALYZE` variam entre versões e variantes. Escolha a sintaxe que o servidor realmente suportar.

### Analisar o plano de execução

Comece com a consulta real, sem efeitos colaterais, e com colunas explícitas:

```sql
EXPLAIN
SELECT o.id, o.created_at, c.display_name
FROM orders AS o
JOIN customers AS c ON c.id = o.customer_id
WHERE o.status = 'pending'
  AND o.created_at >= '2026-01-01'
ORDER BY o.created_at DESC;
```

Quando suportado, compare a forma legível com a versão estruturada:

```sql
EXPLAIN FORMAT=JSON
SELECT o.id, o.created_at
FROM orders AS o
WHERE o.customer_id = <id>
ORDER BY o.created_at DESC;
```

`EXPLAIN ANALYZE` pode executar a consulta e só deve ser usado quando isso for aceitável e o servidor declarar suporte. Em uma consulta de leitura, compare estimativas com medições; em qualquer comando mutável, não o use como substituto de uma transação ou de uma revisão de impacto. Observe tipo de acesso, chave escolhida, linhas estimadas, `filtered`, materialização/temporárias e ordenação. A estimativa não é prova de que o índice é adequado.

### Índices por workload

Não adicione um índice para cada coluna. Comece pela carga de trabalho (workload) real:

- **Leitura pontual:** considere a chave usada no `WHERE` e a chave primária/estrangeira da junção.
- **Junções:** as colunas de ligação precisam de índices compatíveis e tipos coerentes.
- **Filtro + ordenação:** em um índice composto, avalie a ordem por seletividade, igualdade, faixa e ordenamento; confirme o uso do prefixo esquerdo.
- **GROUP BY/aggregate:** considere a sequência de agrupamento e o impacto de covering index, sem duplicar cobertura desnecessária.
- **Escritas:** cada índice secundário tem custo de armazenamento e manutenção, além de bloquear ou atrasar operações dependendo do motor.

Exemplo de hipótese, não de regra automática:

```sql
SHOW INDEX FROM orders;

CREATE INDEX idx_orders_customer_created
  ON orders (customer_id, created_at);
```

Compare o plano antes/depois com dados representativos e observe o custo no workload de `INSERT`, `UPDATE` e `DELETE`. Índice coberto demais aumenta a largura da árvore e o custo de manutenção. Use índice funcional, de texto ou de busca textual somente quando a consulta, a expressão e a versão do servidor justificarem; confirme a sintaxe antes de usar.

### Integridade, chaves estrangeiras e constraints

Toda alteração de schema deve explicar a intenção:

- Escolha uma chave primária ou uma chave alternativa coerente com o modelo.
- Use `NOT NULL` e `UNIQUE` quando forem invariantes reais, não por conveniência.
- Antes de criar uma FK, procure registros órfãos e confirme a tabela, coluna, tipo e collation da chave referenciada.
- Escolha `ON DELETE` e `ON UPDATE` conscientemente. `CASCADE` pode apagar ou alterar muitas linhas; `RESTRICT`/`NO ACTION` falham em vez de apagar, mas exigem uma política explícita para os órfãos.
- Verifique se a versão aceita `CHECK` e como trata constraints inválidas; não copie uma cláusula de outro SGBD.

```sql
ALTER TABLE order_items
  ADD CONSTRAINT fk_order_items_order
  FOREIGN KEY (order_id) REFERENCES orders (id)
  ON DELETE RESTRICT
  ON UPDATE RESTRICT;
```

Se a migração adicionar a constraint, faça uma etapa de limpeza/validação antes e um teste com uma operação de escrita e exclusão protegida. Não trate a criação da FK como mera alteração de metadados.

### Transações e consistência

Para operações que alteram várias linhas ou tabelas, delimite a unidade de trabalho e trate erro como rollback:

```sql
START TRANSACTION;
UPDATE account_a
   SET balance = balance - 100
 WHERE id = <account_a_id>;
UPDATE account_b
   SET balance = balance + 100
 WHERE id = <account_b_id>;
COMMIT;
```

Se qualquer instrução SQL falhar, execute `ROLLBACK` e investigue antes de repetir. Para fluxos com muitas etapas, considere savepoints, mas não use uma transação longa para esconder uma operação interativa. Verifique `autocommit`, nível de isolamento, locks e o comportamento de repetição do driver. Deadlocks e lock wait timeout exigem nova tentativa com limite e idempotência; repetir cegamente pode duplicar uma transferência.

### Cuidado com DDL, locks e longo prazo

`CREATE`, `ALTER`, `DROP`, `RENAME`, `TRUNCATE`, criação/remoção de índice e mudanças de coluna podem adquirir locks de metadados, fazer I/O, invalidar planos ou bloquear consultas. `LOCK TABLES`, transações longas e operações concorrentes podem aumentar a espera. MySQL e MariaDB têm capacidades de DDL online diferentes; a mesma sintaxe pode exigir `LOCK`, um algoritmo específico ou uma tabela copiada internamente. Não assuma que `ALTER` é instantâneo.

Antes de DDL:

1. Consulte a versão, o motor, os locks e as transações ativas.
2. Meça tamanho da tabela e crescimento; verifique réplicas, atraso de réplica e consumidores.
3. Teste em cópia representativa e salve o estado anterior.
4. Escolha uma janela compatível; confirme se o DDL faz commit implícito ou invalida a transação.
5. Depois, verifique estrutura, plano, locks e comportamento da aplicação.

Use a ferramenta de migração do projeto quando ela registrar a versão e possuir rollback conhecido. Uma transação SQL não garante rollback de todo DDL.

### Cliente, backup e restore sem senha no argv

Use prompt interativo, um cliente configurado fora do repositório ou um arquivo de opções com permissão `600`. Não embuta senha em opções de linha de comando, em `docker compose exec`, em scripts de CI ou em `ENV` de imagem. Evite imprimir a URL de conexão completa.

```bash
mysql --defaults-extra-file=<client.cnf> -h <host> -u <user> <database> -e \
  "SELECT VERSION(), DATABASE();"
```

Para um dump local, use o cliente compatível e um destino fora do repositório:

```bash
mkdir -p <diretorio-de-backup>
chmod 700 <diretorio-de-backup>
mysqldump --defaults-extra-file=<client.cnf> -h <host> -u <user> <database> \
  > <diretorio-de-backup>/<arquivo>.sql
chmod 600 <diretorio-de-backup>/<arquivo>.sql
```

Um `mysqldump` executado dentro de um container pode usar um arquivo de configuração montado como segredo. O comando exato depende da imagem e do serviço; não cole a senha no comando. Para restore, teste primeiro em uma instância isolada, preserve o dump original e confirme schema, contagem, constraints e consultas de aplicação antes de apontar o serviço real.

## 4. Outros motores

### PostgreSQL

```bash
docker compose exec -T postgres psql -U <user> -d <database> -c '\dt'
docker compose exec -T postgres psql -U <user> -d <database> \
  -c 'SELECT count(*) FROM <tabela>;'
```

Use `.pgpass`, `PGSERVICE` ou o mecanismo de segredos do ambiente em vez de imprimir a senha. Para backup rápido, redirecione `pg_dump` para fora do repositório, restrinja o arquivo e teste o restore.

### MongoDB

```bash
docker compose exec -T mongo mongosh --quiet --eval \
  'db.getCollectionNames()'
```

Inspecione índices e documentos com consultas projetadas e o menor volume necessário. Para DDL/DML, considere transações, journaling, validação de schema e o impacto de índices; confirme recursos da versão do servidor.

### SQLite

```bash
sqlite3 ./dev.db '.tables'
sqlite3 ./dev.db '.schema <tabela>'
sqlite3 ./dev.db 'SELECT <colunas> FROM <tabela> LIMIT 5;'
```

Faça backup consistente do arquivo, respeite locks e teste integridade com `PRAGMA integrity_check` quando fizer sentido. Não trate um arquivo copiado durante escrita como backup consistente.

### Firebird

Use `isql-fb` ou o cliente fornecido pelo projeto e deixe a autenticação para o mecanismo seguro do ambiente:

```bash
isql-fb -user <user> localhost:/<caminho-do-banco> -i <arquivo>.sql
```

Confirme dialeto, charset, transações e caminhos/ferramentas da versão instalada antes de executar DDL.

### Redis

Inspecione `INFO`, memória, persistência, replicação e chaves por padrão, sem varrer o keyspace inteiro em produção. Use `SCAN` quando a operação precisar percorrer chaves. Antes de executar `FLUSH`, expirar chaves ou modificar a persistência, confirme o escopo e faça backup quando o dado puder ser reconstruído.

## 5. Migrações por ecossistema

Os comandos abaixo aplicam a migração e alteram estado; mostre o plano e peça confirmação antes de executá-los:

```bash
npx prisma migrate status
npx prisma migrate dev --name <nome_da_migration>
npx prisma generate

npx drizzle-kit generate
npx drizzle-kit migrate

alembic revision --autogenerate -m "add_user_status_column"
alembic upgrade head

dotnet ef migrations add AddUserProfile
dotnet ef database update

migrate -path ./migrations -database <fonte-de-conexao> up
```

Se a ferramenta aceitar reset, seed destrutivo ou conexão alterada, trate cada uma como operação separada. Leia o diff da migração, valide dados existentes e confirme o destino antes de aplicar.

## Regras de segurança e verificação

- Nunca executar `DROP DATABASE`, `DROP TABLE`, `TRUNCATE`, reset, flush destrutivo ou restore em produção sem plano, backup e confirmação explícita.
- Não versionar dumps, `.env`, chaves ou URLs com credenciais.
- Usar colunas explícitas quando a query precise ser previsível; não impor `SELECT *` como dogma para toda consulta legítima.
- Preferir prepared statements, menor privilégio e credenciais de menor duração.
- Repetir após a mudança: versão/dialeto, `EXPLAIN` quando houver consulta afetada, estrutura/constraints, locks, logs, saúde e uma verificação da aplicação.
- Registrar o resultado sem imprimir segredos, dados pessoais ou inventário de servidores.
