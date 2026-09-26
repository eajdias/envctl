# Ubuntu Server Baseline — o que o `envctl run performance` ajusta

> Escopo: **Ubuntu Server `VERSION_ID >= 24`** (a frota roda 24.04 e 26.04).
> Cada item abaixo lista o que faz, a evidência medida que justifica o valor,
> o comando exato de reversão e o que ele **nunca** toca.

Este documento substitui a frase que a documentação carregava antes desta branch
("não cria swapfile, não altera journald"): o perfil agora faz as duas coisas, e
a política é declarativa e auditável.

## Como a política é resolvida

```
manifesto (performance_ubuntu.yaml)
        │
        ├─ mede o host ── HardwareProbe: MemTotal, tipo de FS, disco livre, /proc/swaps
        │
        ├─ banda de RAM ── SelectPerformanceTier: tiny | small | medium | large
        │
        ├─ swapfile ── adota o existente OU cria um clampado
        │
        ├─ zram ── ligado conforme a banda; recusado se já existe outro device
        │
        ├─ re-mede ── a topologia de swap mudou depois do zram subir
        │
        └─ sysctls ── base do perfil + banda + vm.swappiness derivado (sempre por último)
```

Nada aqui é chutado: **toda** constante que o manifesto não fixa é detectada, e a
detecção aparece no detalhe do diagnóstico.

## A armadilha de `MemTotal`

`/proc/meminfo` **nunca** reporta o tamanho nominal da instância:

| Host | Nominal | `MemTotal` real |
| --- | --- | --- |
| `vps_oracle_2`, `vps_oracle_1` | "1 GB" | 974092 kB = **951 MiB** |
| `zscan_chatbot` | "16 GB" | 16162396 kB = **15783 MiB** |

Por isso as bandas de `tiers` ficam ~50% acima do nominal, e o resolvedor exige
`MemTotal != 0`: um host não medido é erro, nunca o tier `tiny` por acidente.

---

## 1. `swap` — resiliência, não performance

```yaml
swap:
  policy: auto
  file: /swapfile.envctl
  priority: -2
  size_of: mem_total
  size_min: 1G
  size_max: 8G
  disk_reserve: 5G
  fs_allow: [ext4, xfs]
  fs_btrfs: refuse
  fs_deny: [zfs, overlay, tmpfs]
```

**Adoção é o caso comum.** As duas OCI já nascem com `/swapfile` de 8 GiB criado
pelo operador, em `prio -1`. Quando existe swap em disco, o run é um **no-op
completo**: sem `fallocate`, sem `mkswap`, sem `swapon`, sem escrita em `fstab`.

O path declarado é `/swapfile.envctl`, deliberadamente **não** `/swapfile`, para
que um swapfile do operador nunca seja confundido com um gerenciado pela ferramenta.

**Dimensionamento.** `MemTotal` clampado em `[1G, 8G]`, e recusado se o
filesystem não cobrir o arquivo mais a reserva de 5G. Medido: 951 MiB → 1 GB;
15783 MiB → 8 GB.

**Filesystem.** `btrfs` é recusado por padrão porque a documentação do btrfs chama
um swapfile na raiz de *"especially discouraged"* — com um swapfile ativo,
`balance` e `scrub` pulam blockgroups e a verificação de integridade degrada.

**Reversão:**

```bash
sudo swapoff /swapfile.envctl && sudo rm /swapfile.envctl
sudo cp /etc/fstab.bak.<stamp> /etc/fstab
```

Nos hosts adotados **não há o que reverter**: nada foi escrito.

## 2. `vm.swappiness` — derivado, nunca declarado

O manifesto **não** declara este valor. Ele vem da topologia de swap medida,
depois que o zram subiu:

| Topologia | Valor | Porquê |
| --- | --- | --- |
| zram ativo | **150** | kernel doc: *"For in-memory swap, like zram or zswap … values beyond 100 can be considered"* |
| só disco | **10** | mantém o kernel preferindo page cache a thrash de disco |
| sem swap | **10** | o swap em disco será criado em seguida |

Isto corrige um defeito real do perfil anterior, que instalava
`systemd-zram-generator` **e** fixava `10` — ou seja, criava o dispositivo
comprimido e dizia ao kernel para não usá-lo.

`vfs_cache_pressure` **é** declarado, mas por banda (50/75/100/100), porque é uma
troca entre RAM e syscalls, e o lado certo da troca muda com o tamanho. Os bands
`small` e `medium` estão marcados no manifesto como **derivados, não validados em
hardware** — a frota não tem host de 4 GB nem de 8 GB.

## 3. `zram` — por banda

`zram: policy: tier`. `tiny` e `small` ligam; `medium` e `large` não, porque nessa
faixa o swap em disco é barato frente à RAM que o compressor ocuparia, e o page
cache vale mais.

Um device `zram*` já existente faz o run **recusar** criar um segundo, e um swap em
disco cuja prioridade já supera a do zram é reportado como **conflito** em vez de
ser reordenado em silêncio.

## 4. `sysctls` — com `policy: min` onde o valor é um piso

| Chave | Política | Observação |
| --- | --- | --- |
| `net.core.somaxconn` | set | 65535; medido 4096 nos hosts |
| `net.ipv4.tcp_max_syn_backlog` | set | 4096 |
| `fs.file-max` | **min** | OCI e AWS já vêm em `9223372036854775807` |

`policy: min` significa: escreve só quando o valor efetivo do host é **menor**.
Sem isso, o manifesto antigo escreveria `2097152` por cima do teto de 32 bits dos
dois clouds — uma regressão de muitas ordens de grandeza que ele reportaria como
sucesso.

**Reversão:** restaurar `/etc/sysctl.d/90-envctl-performance.conf.bak.<stamp>` e
`sudo sysctl -p <arquivo>`.

## 5. `limits` — soft alto, hard do host intacto

```yaml
limits:
  nofile_soft: 65536
  nofile_hard: ""     # vazio = preserva o do host
  nproc: 32768
  daemon_reexec: true
```

Medido: soft `1024` em todos os hosts (risco real de `EMFILE` com Docker e Node),
hard `524288` na OCI e `1048576` na AWS. O drop-in systemd escreve
`DefaultLimitNOFILE=65536:` — com o **segundo campo vazio**, forma verificada com
`systemd-analyze cat-config` que faz parse limpo e preserva o hard de cada host.
Fixar `65536:524288` rebaixaria a AWS.

Dois destinos, porque são mecanismos diferentes: `system.conf.d/` cobre as units que
o systemd inicia, `security/limits.d/` cobre sessões de login. Um `limits.d` sozinho
não toca nenhum serviço.

**Atenção declarada em cada run:** `man 5 systemd.exec` avisa que subir o soft acima
de 1024 quebra `select(2)` em software que ainda a usa. O diagnóstico repete isso
sempre que o valor passa do limiar.

`daemon-reexec` é a **única** operação que toca o PID 1, e é necessária porque
`daemon-reload` não relê `system.conf`. `man 1 systemctl` documenta que os sockets
permanecem acessíveis durante o reexec, e `--no-daemon-reexec` desliga.

**Reversão:** remover os dois drop-ins e `sudo systemctl daemon-reexec`.

## 6. `journald` — teto com piso

```yaml
journald:
  values:
    - { key: SystemMaxUse, value: "200M" }
    - { key: SystemKeepFree, value: "1G" }
    - { key: RuntimeMaxUse, value: "50M" }
    - { key: MaxRetentionSec, value: "2week" }
```

Os 3 hosts alcançáveis estão **sem teto**, com 65M / 127M / 105M em uso. O default
do journald é 10% do filesystem: **~4,5 GB** nos discos de 45 GB da OCI. Num host
de 951 MiB, journal cheio impede log, impede SSH e impede diagnóstico.

`SystemKeepFree` é obrigatório, não extra: `man 5 journald.conf` diz que o journald
honra **os dois** limites e usa o **menor**.

O serviço é **reiniciado**, nunca parado: `man 8 systemd-journald` diz que um
`restart` preserva os streams dos clientes e que `stop` não é recomendado.

**Reversão:** remover `/etc/systemd/journald.conf.d/90-envctl-journald.conf` e
`sudo systemctl restart systemd-journald`.

## 7. `timezone` — verifica por padrão

Os 3 hosts estão em `Etc/UTC` com NTP ativo, então o modo `verify` é uma passagem
silenciosa. Um host em outra zona é **INFO** e fica como está: fuso é decisão do
operador, não knob de tuning, e mudar por padrão alteraria timestamps de log e
agendamentos em produção.

`--timezone <IANA>` é a única forma de aplicar, e valida o nome contra
`/usr/share/zoneinfo` antes de escrever.

**Reversão:** `sudo timedatectl set-timezone <anterior>`.

## 8. Debloat — opt-in, com guarda

Só roda com `--allow-debloat` ou `--debloat-only`, porque é a única etapa que
**remove estado visível ao operador**. A lista está em
`manifests/debloat_linux.yaml` e vem da frota **medida**, não de nenhum bootstrap
externo:

| Entrada | Evidência |
| --- | --- |
| `modemmanager` | instalado 3/3; `ModemManager.service` rodando no `vps_oracle_2` |
| `fwupd` | instalado 3/3 |
| `udisks2` | instalado 3/3 |
| `iscsid` | `iscsid.service` rodando no `vps_oracle_2`, sem target iSCSI |
| `multipathd` | rodando no `vps_oracle_2`, disco único |
| `rpcbind` | presente nas 2 OCI, **guardado** por `nfs-common` |

`avahi-daemon`, `cups`, `bluez` e `bluetooth` **não** entram: estão ausentes em
todas as imagens cloud medidas, e listar o que nunca existiu é ruído que esconde o
que importa.

**A guarda vem primeiro, sempre.** O Ubuntu traz `needrestart` (3.11-1ubuntu2
medido no `vps_oracle_2`, sem override) em modo automático, que reinicia serviços
após mudança de pacote — o que, por SSH, pode derrubar a sessão que faz a mudança.
O drop-in `NEEDRESTART_MODE=l` é gravado **antes** do primeiro purge, e a spec é
**rejeitada** sem ele.

O diagnóstico nunca é `WARNING`: ausente é `OK`, pulado por guarda é `INFO`
nomeando o guard. Isso mantém o doctor em 0 WARN num host convergido.

**Reversão:** `sudo apt-get install -y <pacote>`, um por um.

## Precondição: reboot pendente

`/var/run/reboot-required` gera um único `DiagWarning` **antes** de qualquer
escrita, e o run aborta com saída não-zero. Ajustar performance em cima de um
runtime que o host vai substituir descreve um estado que não existirá depois do
boot. `--force-reboot-pending` contorna; o `vps_oracle_2` está com reboot pendente
no momento da escrita deste documento.

## O que este perfil nunca toca

CPU governor · I/O scheduler · mitigations · kernel cmdline · `crashkernel`/`kdump`
· swap em btrfs na raiz · serviços sem relação · hibernação.

Todos continuam atrás de benchmark e aprovação explícita, como decidido em
`spec-agent/2026-09-24-linux-performance-baseline.md`. A knowledge de produto do
envctl mora aqui, não em skill de agente: o commit `d5a2db0` cortou o catálogo de
50 para 12 e movia esse conhecimento para dentro do repo de propósito.

## Unknowns que ficam para o dono

- **`zscan_proxy_prod` não autentica** — 9 de 10 máquinas verificadas.
- **Não existe VPS de 4 GB nem de 8 GB** — os bands `small` e `medium` são
  derivados, declarados como tal no manifesto.
- **`nofile` soft em 65536** é risco documentado (`select(2)`), não testado
  empiricamente na frota. `--no-daemon-reexec` existe para recuar.
- **zram no band `tiny`** é hipótese: `vps_oracle_1` usa 681 MiB de swap, mas o
  ganho não foi medido. É a única mudança da branch que altera comportamento de
  workload em vez de folga.
