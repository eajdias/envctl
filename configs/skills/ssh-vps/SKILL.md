---
name: ssh-vps
description: >-
  Operação segura de VPS e servidores remotos via SSH, Linux ou Windows OpenSSH. Use para monitorar, diagnosticar, recuperar ou automatizar serviços com ssh-manager, ssh, rsync, systemd, Docker ou PM2, além de organizar chaves, permissões, known_hosts e conexões não interativas. Triggers: ssh, vps, servidor, instância, monitorar, health check, ssh-manager, chave ssh, known_hosts, authorized_keys, BatchMode, systemd, pm2, serviço caiu, túnel, rsync, backup remoto.
license: MIT
---

# SSH em VPS e servidores remotos

## Quando usar

Use para operar um servidor por SSH quando o usuário fornecer o alvo por um inventário local ou por um comando explícito. O inventário pode conter dados de conexão, mas esse arquivo permanece local e nunca deve ser copiado para esta skill, README, docs ou outro arquivo versionado.

Comece com leitura. Reiniciar, parar, editar, enviar arquivos, criar usuários, instalar pacotes, abrir portas e alterar chaves são operações mutáveis: mostre o comando, o impacto e a forma de recuperação, e peça confirmação quando o risco não for evidente.

## Inventário e ferramentas

Consulte, sem imprimir segredos:

1. o inventário local fornecido pelo ambiente;
2. `ssh-manager server list` ou a listagem do MCP quando ele estiver habilitado;
3. a ajuda da ferramenta para confirmar sintaxe e permissões.

O caminho e o formato do inventário são configuração local. Não incorpore hostname, endereço IP, usuário, porta, fingerprint ou caminho de chave real neste arquivo. Use referências genéricas como `<host>`, `<usuario>`, `<porta>` e `<chave>` nos exemplos.

Use o `ssh-manager` CLI quando ele já estiver disponível:

```bash
ssh-manager server list
ssh-manager server test <nome>
ssh-manager exec <nome> "<comando>"
```

Para SSH direto ou transferência, use somente dados obtidos do inventário local:

```bash
ssh -i "<chave-do-inventario>" <usuario>@<host> "uptime"
rsync -avz -e "ssh -i <chave-do-inventario>" ./arquivo/ <usuario>@<host>:<destino>
```

Se um MCP for habilitado no ambiente, valide a lista de servidores antes de usar as ferramentas. Não habilite, desabilhe ou edite configuração do agente como efeito colateral desta skill.

## Higiene de chaves

Esta parte é válida para qualquer usuário ou servidor; não substitui a política de cada ambiente.

### Permissões no cliente

No cliente Linux, use um diretório SSH privado, arquivo de chave privada e arquivos de controle restritos. `chmod` deve ser executado no cliente, no usuário correto:

```bash
umask 077
install -d -m 700 "$HOME/.ssh"
chmod 700 "$HOME/.ssh"
chmod 600 "<chave-privada>"
chmod 644 "<chave-publica>"
for path in "$HOME/.ssh/config" "$HOME/.ssh/known_hosts"; do
  [ -e "$path" ] && chmod 600 "$path"
done
if [ -e "$HOME/.ssh/authorized_keys" ]; then
  chmod 600 "$HOME/.ssh/authorized_keys"
fi
```

Antes de corrigir modos ou ownership, rejeite links simbólicos inesperados; não altere o alvo de um link para completar a receita. Ajuste a ownership sem alterar recursivamente diretórios desconhecidos:

```bash
owner="$(id -un)"
group="$(id -gn)"
chown "$owner:$group" "$HOME/.ssh"
chown "$owner:$group" "<chave-privada>" "<chave-publica>"
for path in "$HOME/.ssh/config" "$HOME/.ssh/known_hosts"; do
  [ -e "$path" ] && chown "$owner:$group" "$path"
done
if [ -e "$HOME/.ssh/authorized_keys" ]; then
  chown "$owner:$group" "$HOME/.ssh/authorized_keys"
fi
```

A chave privada nunca deve ser legível por grupo ou outros usuários. A chave pública pode ser `0644`, desde que a ownership e o conteúdo sejam válidos. Se a máquina cliente também aceitar conexões de entrada, `authorized_keys` é um arquivo de chaves públicas e deve ter `0600`; ele não substitui `known_hosts` e não deve ser confundido com a chave privada do cliente. Se o cliente não atua como servidor, não crie esse arquivo por padrão.

### Permissões no servidor

O `authorized_keys` do servidor deve pertencer ao usuário SSH e ficar dentro de um diretório `.ssh` com `0700`:

```bash
sudo install -d -m 700 -o <usuario> -g <grupo> <home>/.ssh
sudo chown <usuario>:<grupo> <home>/.ssh/authorized_keys
sudo chmod 600 <home>/.ssh/authorized_keys
```

Use o caminho e o grupo efetivamente criados pelo administrador; não use `chown -R` nem troque o dono da home inteira. Verifique sem revelar o conteúdo:

```bash
stat -c '%A %U:%G %n' <home>/.ssh <home>/.ssh/authorized_keys
```

A configuração do `sshd` pode recusar chaves com permissão excessiva. Se um login falha, compare ownership, modo, tipo de arquivo, home, `AuthorizedKeysFile` e logs do servidor; não enfraqueça a segurança global como atalho.

### `known_hosts` e verificação do host

`known_hosts` registra a identidade do servidor. Não desative a verificação para fugir de um erro:

```bash
ssh-keygen -F <host> -f "$HOME/.ssh/known_hosts"
ssh -o StrictHostKeyChecking=yes \
  -o UserKnownHostsFile="$HOME/.ssh/known_hosts" \
  -o BatchMode=yes <usuario>@<host> "uptime"
```

Para uma conexão nova, obtenha o fingerprint por um canal confiável e compare-o antes de aceitar a chave. `accept-new` só é aceitável em uma conexão nova verificada; não o transforme em uma política permanente. Se uma chave foi rotacionada, remova somente a entrada antiga depois de confirmar a nova identidade.

### `BatchMode` e identidade explícita

Para automação, faça a conexão falhar rápido em vez de abrir prompt invisível:

```bash
ssh -o BatchMode=yes \
  -o IdentitiesOnly=yes \
  -o ConnectTimeout=10 \
  -i "<chave-do-inventario>" \
  <usuario>@<host> "<comando>"
```

`BatchMode=yes` desativa prompts de senha e confirmação. Se a chave usa passphrase, carregue-a no agente em uma etapa autorizada ou use um mecanismo de credencial aprovado; não coloque a passphrase no comando, no arquivo de inventário versionado ou no output. `IdentitiesOnly=yes` evita que o cliente tente chaves agentes que não foram solicitadas.

### Gerar chave de forma idempotente

O comando só cria a chave quando ela ainda não existe; nunca sobrescreve material existente:

```bash
set -eu
umask 077
install -d -m 700 "$HOME/.ssh"
key="$HOME/.ssh/<nome-da-chave>"
if [ ! -e "$key" ] && [ ! -e "$key.pub" ]; then
  ssh-keygen -t ed25519 -a 100 -f "$key" -C "<rotulo>"
elif [ -e "$key" ] && [ -e "$key.pub" ]; then
  printf '%s\n' 'Chave existente; nenhuma troca foi feita.'
else
  printf '%s\n' 'Par de chaves incompleto; investigue antes de continuar.' >&2
  exit 1
fi
chmod 600 "$key"
chmod 644 "$key.pub"
```

Se apenas um dos dois arquivos existir, pare e investigue o par incompleto; não force a regeneração. Para automação não interativa, use uma política de passphrase e uma fonte de segredo aprovada, nunca uma senha vazia implícita apenas para conveniência. A chave pública pode ser instalada no `authorized_keys` do servidor; a privada permanece no cliente.

## Diagnóstico remoto

Para um serviço que caiu, comece com uma coleta ampla e sem mudança:

1. `uptime`, `df -h`, `free -h` e o estado geral do sistema;
2. serviços com falha, containers ou processos gerenciados;
3. logs recentes do serviço e do sistema;
4. causa provável, impacto e comando de recuperação;
5. confirmação para reiniciar/alterar;
6. nova consulta de status, logs e uma verificação funcional.

Linux:

```bash
systemctl is-system-running
systemctl list-units --type=service --state=failed
journalctl -u <servico> --no-pager -n 50
systemctl status <servico> --no-pager
```

Docker e PM2:

```bash
docker ps -a
docker logs --tail 50 <container>
pm2 list
pm2 logs <id> --lines 50
```

Em Windows OpenSSH, use PowerShell e comandos nativos (`Get-Service`, `Get-WinEvent`, `Restart-Service`) em vez de `systemctl` ou caminhos Linux. Descubra o shell e o principal da sessão antes de executar.

## Execução remota segura

- Comece com comandos somente leitura e capture a saída completa necessária.
- Use `sudo` só quando a operação realmente precisar de privilégio; em uma cadeia, explicite o privilégio de cada parte em vez de presumir que ele se propaga.
- Evite editores e programas interativos no remoto. Use arquivos temporários controlados, `cat`, `journalctl`, `docker logs` e `pm2 logs`.
- Para processos longos, use uma sessão gerenciada, log persistente e mecanismo de retomada; não dependa de um terminal que possa morrer.
- Para arquivos pequenos, prefira envio controlado; para arquivos grandes, use `rsync` com a identidade correta e verifique soma de verificação/tamanho.
- Nunca exponha chaves, senhas, tokens ou variáveis secretas na saída, nos logs ou em arquivos versionados.
- Alterações no inventário local são permitidas quando solicitadas, mas devem ser testadas e não devem ser sincronizadas para o repositório.

## Checklist de segurança

- O cliente usa a identidade explícita e não depende de senha interativa em automação.
- `~/.ssh` está `0700`; chave privada, `config` e `known_hosts` estão `0600`; ownership pertence ao usuário certo.
- O servidor tem `authorized_keys` com ownership e `0600`, e a configuração do `sshd` foi consultada.
- A identidade do host foi comparada; a verificação de host permanece habilitada.
- A chave foi gerada uma única vez e não foi sobrescrita por idempotência.
- Toda mudança remota tem plano, confirmação quando necessária e verificação posterior.
