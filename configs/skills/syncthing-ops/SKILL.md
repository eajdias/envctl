---
name: syncthing-ops
description: >-
  Operar o Syncthing do homelab via REST API (aceitar pastas, compartilhar,
  diagnosticar 99% travado, folder marker missing). Use quando precisar criar,
  aceitar ou remover shares, debugar sync parado ou mapear paths do container.
  Triggers: syncthing, 99%, folder marker missing, pending folder, aceitar pasta,
  share, remoteneed, completion, .stfolder.
license: MIT
---

# Syncthing Ops (homelab)

## Quando usar

- Aceitar convite de pasta vindo de outro device (ex.: celular).
- Compartilhar/desvincular pasta com um device.
- Sync travado (%, `error`, `folder marker missing`).
- Paths "ruins": pasta dentro do container ≠ path no host.

## Contexto fixo deste homelab

- GUI/API: `http://<TAILNET_IP>:8384 (IP do host na tailnet — inventário local)` (sem login; auth = header `X-API-Key`).
- API key: `grep -o '<apikey>[^<]*' /mnt/homelab/stack/syncthing/config/config.xml`.
- Compose: `/mnt/homelab/stack/syncthing/docker-compose.yml`.
- Mounts: `/mnt/homelab/docs:/docs`, `/mnt/homelab/phone:/phone`.
  **Toda pasta nova precisa de path sob `/docs` ou `/phone`** — fora disso o
  container não enxerga e dá `folder path missing`.
- NUNCA reescrever `config.xml` com script (parser Go rejeita encoding
  estranho e o container entra em crash-loop). Edição cirúrgica de 1 linha
  no máximo; o resto é API.
- Pastas (desenho 2026-09-30, decidido pelo dono — sem `docs`/`Docs` duplicados):
  `es59v-2cz7r` (`Download-Celular`→`/phone/Download`, **receiveonly** no PC,
  celular só envia) + `vault` (`Vault`→`/docs/vault`, sendreceive PC+notebook,
  sem celular). `20z9g-ukw1g` (Documents do celular) DESATIVADO a pedido do
  dono; reativar só pelo §6.
- Notebook (`notebook`, `VU6EXDW…`, Win): Syncthing v2.1.5 via
  `winget install Syncthing.Syncthing`; roda via atalho na pasta Startup;
  raiz única `C:\syncthing-homelab\` (uma subpasta por share; Vault em
  `C:\syncthing-homelab\Vault`). Tailscale nele estava UP (o `tailscale
  status` do desktop mostrava stale offline).
  Conexão PC↔notebook DIRETA via tailnet: `addresses` pinados dos dois lados
  (`tcp://<TAILNET_IP_DESKTOP>:22000` / `tcp://<TAILNET_IP_NOTEBOOK>:22000` + `dynamic`) —
  sem isso caía em relay (porta 22000 aberta no tailnet mas fechada na LAN do
  notebook pelo firewall; tailnet já roteia pelo fio local em casa).
  Firewall do notebook: regras `Syncthing TCP` (in/TCP 22000) + `Syncthing UDP`
  (in/UDP 22000,21027), `Profile Any` (Wi-Fi é perfil Public). Pura porta, SEM
  escopo de programa — o `syncthing.exe` do PATH é shim do WinGet
  (`Links\`, reparse point) e regra por programa não casa com o binário real
  em `Packages\Syncthing.Syncthing_…\`.

## Passos

### 1. Estado atual

```bash
python3 -c "
import urllib.request, json
key = open('/mnt/homelab/stack/syncthing/config/config/config.xml').read().split('<apikey>')[1].split('<')[0]
def get(p):
    r = urllib.request.Request('http://127.0.0.1:8384' + p, headers={'X-API-Key': key})
    return json.load(urllib.request.urlopen(r, timeout=20))
for f in get('/rest/config/folders'):
    s = get('/rest/db/status?folder=' + f['id'])
    print(f['id'], '|', f.get('label'), '|', f.get('path'), '|', s.get('state'))
"
```

### 2. Aceitar pasta pendente (convite de outro device)

Descobrir o id no `/rest/events` (`PendingFoldersChanged`), depois PUT:

```bash
# PUT /rest/config/folders/{id} com path DENTRO de /docs ou /phone
# body: {"id","label","path","type":"sendreceive","devices":[{"deviceID":mine},{"deviceID":phone}],
#        "rescanIntervalS":3600,"fsWatcherEnabled":true,"fsWatcherDelayS":10}
```

Se o app Android não deixa salvar o aceite (botão inativo), aceitar por aqui
resolve — o celular só precisa ter a pasta compartilhada com este device.

### 3. Compartilhar / desvincular device

`GET /rest/config/folders/{id}`, edita `devices` (adiciona/remove
`{"deviceID": ...}`), `PUT` de volta. Mesma chamada muda `path` e `label`.

### 4. Diagnosticar % travado ou state=error

1. `GET /rest/db/status?folder={id}` → `needFiles`/`needBytes` + `state`.
2. `GET /rest/folder/errors?folder={id}` → erros de item (null = sem erro de arquivo).
3. `GET /rest/db/completion?device={dev}&folder={id}` por device — o "99% nos
   dispositivos remotos" quase sempre é pasta que o OUTRO lado não aceitou ou
   não consegue gravar (Doze, scoped-storage). `remoteneed` lista os arquivos
   pendentes do lado remoto.
4. `folder marker missing` → criar `{path}/.stfolder` no HOST (não no container)
   e forçar `POST /rest/db/scan?folder={id}`. Checar mounts com
   `docker inspect syncthing --format '{{range .Mounts}}{{.Source}} -> {{.Destination}}{{println}}{{end}}'`.

### 5. Adicionar mount novo

Editar `volumes:` no compose + `docker compose up -d` (recreate) + mover `path`
das pastas via PUT + `POST /rest/db/scan`.

## Verificação

- Todas as pastas `idle` + `need: 0` no passo 1.
- `GET /rest/db/completion` = 100% nos dois lados de cada share.
- Prova real de sync bidirecional: criar arquivo de teste de um lado, conferir
  do outro, depois apagar e conferir que sumiu dos dois.

## 6. Re-adicionar pasta cuja cópia do servidor foi perdida (só o remoto tem)

NUNCA criar `.stfolder` + scan no dir vazio com a pasta em `sendreceive` —
o scan gera tombstones e o delete propaga p/ o remoto quando ele voltar
(= APAGA o celular). Protocolo:

1. Celular online + conferir que os arquivos estão intactos nele.
2. Aceitar a pasta pendente (`PUT /rest/config/folders/{id}`) com
   `"type": "receiveonly"` e path com `.stfolder` já criado no host.
3. Aguardar sync 100% (remoto→servidor, sem nada no sentido contrário).
4. `PUT` de volta p/ `"type": "sendreceive"` + `POST /rest/db/scan`.
5. Validar pelo passo de Verificação acima.
