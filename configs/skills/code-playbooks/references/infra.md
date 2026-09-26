# SSH e infra

> **Fronteira:** isto é *operar o que já existe*. Criar/auditar máquina é da skill
> `envctl`; provisionar serviço de um repo é do repo.

## Chaves e acesso

- Uma chave por destino, com ACL restrita (`chmod 600`, diretório `700`); nunca chave
  compartilhada entre ambientes.
- Cadastre servidor com `ssh-manager` e mantenha o inventário **local**
  (`~/.config/opencode/extras/ssh_servers.md`). IP, usuário e caminho de chave nunca
  entram no git — o histórico preserva PII para sempre.
- `ssh` interativo lento em reconexão: multiplexing no `~/.ssh/config` do cliente
  (`ControlMaster auto`, `ControlPath ~/.ssh/cm-%r@%h:%p`, `ControlPersist 10m`).

## Operação segura

- Banner em toda chamada remota; sessão que não é login tem PATH mínimo — use
  `bash -lc` e resolvam toolchain por caminho absoluto.
- Alteração de serviço: `status` antes, `restart` com causa registrada, `status` depois.
  Se o serviço não voltou, **não** fique em loop de restart: colete evidência.
- Comando destrutivo remoto sempre com confirmação e com o valor repetido no comando.

## Tailscale

- Status e inventário via JSON; `serve`/`funnel` expõem porta — confirme se é tailnet
  ou internet antes de usar.
- Exit node muda o roteamento da máquina inteira: documente e reverta.

## Syncthing

- **Nunca edite `config.xml` na mão**: quebra encoding e o boot do parser. Use a REST
  API com header de API key; padrão é add de folder → rescan → conferir conflito.

## Regras que evitam erro

- Segredo no shell remoto vai para arquivo de secrets na máquina alvo, não para o comando.
- Não rode provisionamento de máquina a partir de sessão sem token; se pedir token ao
  usuário, ele digita — o agente não manuseia credencial.
