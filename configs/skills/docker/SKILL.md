---
name: docker
description: >-
  Docker local e em VPS. Use para inspecionar containers, imagens, volumes, redes e logs; escrever ou revisar Dockerfile e Compose; construir, publicar, transportar e recuperar stacks. Triggers: docker, container, containers, imagem, imagens, compose, docker compose, dockerfile, build, layers, cache, healthcheck, readiness, volume, volumes, rede, networks, backup, restore, secret, secrets, non-root, digest, docker desktop, subir container, derrubar container, logs do container, restart container, docker hub, pull, push, docker ps, stack, vps fraca, build local, wsl exec error, backend.sock.
license: MIT
---

# Docker

## Quando usar

Use esta skill para operar containers ou para revisar a autoria e a segurança de um Dockerfile/Compose. Comece pelo menor procedimento que responde à pergunta: estado do daemon, estado da stack, logs ou origem do problema. Não reinicie, remova, atualize, publique ou reconstrua antes de coletar evidência.

## Primeiro: identificar o ambiente

O cliente e o daemon podem estar em ambientes diferentes. Não assuma que o host é o destino do container:

```bash
docker context show
docker version
docker info --format 'Server={{.ServerVersion}} Root={{.DockerRootDir}}'
```

No Windows com Docker Desktop/WSL2, valide também se o contexto e o daemon estão disponíveis antes de mexer em WSL, serviço ou configuração do Desktop. O comando e os diagnósticos devem seguir o contexto retornado pelo Docker; não reinicie nem desregistre uma distro por tentativa cegamente.

## Segurança e confirmação

Classifique cada comando antes de executá-lo:

- **Leitura:** `ps`, `logs`, `inspect`, `images`, `volume ls`, `network ls`, `compose config` e `compose ps` podem começar a investigação.
- **Alteração:** `build`, `pull`, `push`, `up`, `down`, `start`, `stop`, `restart`, `rm`, `exec` que modifica estado, `prune`, criação/remoção de volume ou rede e alterações de configuração exigem um plano curto e confirmação do solicitante.
- **Destrutivo:** `down -v`, `rm -f`, `volume rm`, `network rm`, `system prune` e `image prune` podem apagar dados ou referências. Mostre o impacto e confirme explicitamente; não trate `--force` como autorização.

Depois de qualquer mudança, execute novamente `docker version`, `docker compose ps` e uma verificação funcional. Registre o que foi alterado, sem colar senhas ou conteúdo de variáveis secretas.

## Dockerfile: decisão antes do código

### Imagem base e reprodutibilidade

Aplique um pin de tag explícita e, para builds que precisam ser reproduzíveis, acrescente um digest verificado para a arquitetura e o registro usados:

```dockerfile
FROM <registry>/<repository>:<tag>@sha256:<digest> AS build
```

O digest deve ser obtido e conferido no registro; não invente um valor. Marque a tag e o digest como dependência de segurança: atualize ambos deliberadamente, rode os testes e registre a mudança. `latest` pode ser útil para uma experimentação local, mas não é uma referência estável para entrega.

### Multi-stage quando agrega valor

A escolha por multi-stage é condicional: use múltiplos estágios quando a compilação, instalação de dependências ou geração de artefatos exigir ferramentas que não devem existir na imagem final. Exemplo genérico:

```dockerfile
FROM <base-pin>@sha256:<digest> AS build
# compila e gera o artefato
FROM <runtime-pin>@sha256:<digest> AS runtime
# instala somente o runtime e copia apenas o artefato
```

Para scripts pequenos ou imagens em que a ferramenta de build já é a runtime, um único estágio pode ser mais simples e auditável. Não adicione estágios só para perseguir um tamanho mínimo: compare tamanho, inicialização, manutenção e risco operacional.

### Usuário sem privilégios

Crie um usuário dedicado com UID/GID conhecido, ajuste a propriedade dos arquivos que precisam ser gravados e termine o Dockerfile com `USER`. Use `COPY --chown=<uid>:<gid>` para artefatos que pertencem ao runtime, em vez de aplicar ownership amplo. Se a imagem não puder ser alterada, defina `user:` no Compose e documente a exceção, os volumes e as permissões. Teste escrita, leitura e o diretório de trabalho em uma execução real; `read_only: true` pode ser combinado com volumes explícitos quando o processo realmente precisar escrever.

### Cache sem confundir camadas com artefatos

- Coloque arquivos que mudam com frequência depois das dependências estáveis.
- Use `.dockerignore` para excluir `.git`, arquivos de ambiente, chaves, dumps, artefatos de build, caches e credenciais. A exclusão evita enviar o arquivo ao daemon, mas não corrige um `COPY` indevido.
- Em builds maiores, considere mounts de cache do BuildKit, como `RUN --mount=type=cache,target=<cache-dir>`. O cache de build não deve ser gravado dentro da imagem final.
- Prefira `COPY` de caminhos explícitos a uma cópia ampla da árvore de trabalho.
- Verifique a configuração com `docker build --pull` e, quando necessário, uma execução limpa sem cache. Não confunda uma camada antiga com uma garantia de reprodutibilidade.

### Segredos fora das camadas

Nunca coloque senha, token, chave privada ou certificado em `ARG`, `ENV`, `COPY`, comando de build ou arquivo versionado. `ARG` e variáveis de ambiente podem aparecer em histórico, metadados e diagnóstico. Para ferramentas de build, use o mecanismo de segredo do BuildKit. O mount fica disponível durante o comando e não grava o conteúdo do segredo na camada:

```dockerfile
RUN --mount=type=secret,id=build_token,target=/run/secrets/build_token \
    <comando-que-le-o-segredo>
```

Para runtime, prefira segredos/arquivos montados pelo orquestrador, com permissões restritas. Se um `.env` for usado apenas como referência local, ele deve estar fora do contexto de build, ignorado pelo Git e nunca ser enviado ao daemon. Depois do build, revise `docker history`, `docker inspect` e a configuração renderizada para confirmar que nenhum segredo ficou exposto.

## Compose: estado, dependências e exposição

Separe o que é configuração versionada do que é segredo local. O esqueleto abaixo mostra apenas campos que precisam ser decididos; adapte nomes e comandos à aplicação:

```yaml
services:
  app:
    image: <registry>/<repository>:<tag>@sha256:<digest>
    build:
      context: .
      target: runtime
    user: "10001:10001"
    read_only: true
    secrets:
      - app_token
    depends_on:
      db:
        condition: service_healthy
    networks:
      - frontend
      - backend

  db:
    image: <registry>/<database>:<tag>@sha256:<digest>
    volumes:
      - db_data:/var/lib/<database>
    healthcheck:
      test: ["CMD-SHELL", "<probe que consulta a prontidão do banco>"]
      interval: 10s
      timeout: 5s
      retries: 12
      start_period: 30s
    networks:
      - backend

volumes:
  db_data:

networks:
  frontend:
  backend:
    internal: true

secrets:
  app_token:
    file: <arquivo-de-segredo-local>
```

### Volumes e redes

- Use volumes nomeados para bancos, filas e estado que precisam sobreviver a `recreate`. Bind mounts são adequados quando o dado deve aparecer em um caminho do host, mas exigem política explícita de permissão e backup.
- `docker compose down` não remove volumes nomeados por padrão; `down -v` remove os volumes do projeto. Trate `down -v` como operação destrutiva.
- Crie redes separadas para o que é exposto ao cliente e para o tráfego interno. Publique no host apenas portas necessárias. Não use `network_mode: host` ou `privileged` por conveniência.
- Para dados importantes, documente retenção, backup, restore testado e o que acontece quando o volume é recriado.

### Readiness, health e dependências

`depends_on` sem condição apenas coordena a ordem de inicialização. Para uma dependência que precisa aceitar conexões, use `condition: service_healthy` e um healthcheck que execute uma consulta real. O healthcheck deve ter timeout, intervalo, número de tentativas e período inicial coerentes com o tempo de inicialização.

Para a aplicação, prefira um endpoint de readiness que valide o que a próxima etapa precisa. Se não existir, use o menor probe real disponível e documente a limitação. Um processo que está `running` ainda pode estar aceitando conexões, pronto para tráfego ou apenas travado. Depois de `up`, confirme `docker compose ps`, logs, o status de saúde e uma requisição ou consulta de negócio.

## Operação, backup e recuperação

Comece com:

```bash
docker ps -a
docker compose ps
docker logs <container> --tail 100
docker inspect <container>
```

Reinicie somente depois de registrar a causa provável e os logs. Para backup, gere o dump para fora do repositório, aplique permissão restrita ao arquivo e teste o restore em uma instância isolada:

```bash
mkdir -p <diretorio-de-backup>
chmod 700 <diretorio-de-backup>
# Grave o dump do serviço no host usando o cliente/secret configurado.
docker compose exec -T <servico> <comando-de-dump-sem-senha> > <diretorio-de-backup>/<arquivo>.dump
chmod 600 <diretorio-de-backup>/<arquivo>.dump
```

O comando de dump, a política de retenção e a criptografia dependem do motor. Para volumes não gerenciados por um dump, documente um snapshot ou cópia consistente e como validar a restauração. Não declare um backup válido só porque o arquivo existe.

Para uma VPS com pouca capacidade de build, uma alternativa é construir localmente, exportar a imagem, carregar na VPS e iniciar o Compose. O transporte da imagem e o `up` alteram estado; confirme ambos e valide a imagem carregada antes de remover qualquer artefato local.

## Reinício do Docker Desktop/WSL2

Se `docker version`, `Wsl/ExecError` ou `backend.sock` indicarem falha do daemon:

1. Consulte o estado do Docker Desktop e do serviço correspondente.
2. Feche o Docker Desktop e, se necessário, use `wsl --shutdown`.
3. Reabra o Desktop e aguarde o daemon ficar disponível.
4. Rode `docker version` e um teste do contexto.

Não execute `wsl --unregister` para diagnosticar uma falha comum; isso pode apagar o ambiente de trabalho. Qualquer reparo persistente precisa de evidência e confirmação.

## Checklist de verificação

- `docker compose config` foi inspecionado sem exibir segredos.
- A imagem de runtime tem tag e, quando aplicável, digest conferido.
- O processo final não é root sem justificativa documentada.
- `.dockerignore` cobre ambiente, chaves, dumps, Git e artefatos.
- Segredos não aparecem em camadas, histórico ou arquivos versionados.
- Volumes nomeados, redes, portas e política de backup estão explícitos.
- Dependências têm readiness e saúde coerentes com o teste real.
- `docker compose ps`, logs e uma verificação funcional foram repetidos após a mudança.
- Comandos destrutivos e alterações de estado só ocorreram após confirmação.
