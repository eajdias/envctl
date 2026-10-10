# Guia de Execução & Provisionamento: Linux (Ubuntu Server / VPS)

Este guia orienta o uso do `envctl` em **Ubuntu Server 24 ou superior**
(inclusive 26.04 LTS) em instâncias de nuvem como **Oracle Cloud Infrastructure**
e **AWS EC2**.

O alvo do perfil de servidor é **exatamente Ubuntu Server `VERSION_ID >= 24`**.
Ubuntu 20.04/22.04, Debian e WSL2 **não são suportados** por esse perfil: o
`min_distro_version` vive no manifesto, e `envctl run vps` falha com o motivo
em vez de seguir e pular o tuning em silêncio. Para desktop CachyOS, veja
[`cachyos-gaming.md`](cachyos-gaming.md).

> Instalação (1-liner, binário ou fonte): ver o guia único
> [`provisioning.md`](provisioning.md). Abaixo, só o que é específico do Linux.

---

## 🎛️ 4. Subcomandos Modulares no Linux

No Linux, comandos específicos de Windows (como `run winget`, `run tweaks`, `run debloat`) são ignorados de forma limpa e segura:

```bash
# Apenas pacotes do sistema via APT (curl, git, ripgrep, fzf, jq, rsync, tree, etc.)
envctl run apt

# Apenas runtimes gerenciados pelo mise (Node.js LTS)
envctl run mise

# Apenas configurações de shell (.bashrc, aliases, git configs)
envctl run shell

# Apenas extração e validação das 12 skills de agentes
envctl run skills

# Auditoria completa do ambiente
envctl doctor

# Auto-remediação de avisos e pendências
envctl doctor --fix
```

### Performance Linux por SO (aplicado pelo perfil `run vps`, standalone abaixo)

O comando `run performance` seleciona um perfil exato e nunca mistura Ubuntu
com CachyOS:

```bash
# Ubuntu Server 24+ ou CachyOS: mostra o plano sem alterar o host
envctl run performance --dry-run

# Aplica somente o perfil do sistema detectado
envctl run performance

# Flags que mudam o que é escrito no host
envctl run performance --no-daemon-reexec      # não re-executa o PID 1
envctl run performance --timezone Etc/UTC       # aplica, em vez de só verificar
envctl run performance --allow-debloat          # inclui a remoção de pacotes
```

O perfil `ubuntu-server` mede o host antes de decidir: banda de RAM, tipo de
filesystem, espaço livre e topologia de swap vêm do probe, e o manifesto declara
o resto. Ele instala `systemd-zram-generator` e `tzdata`, cria o swapfile quando
não existe algum, limita o journald, eleva o soft de descritores de arquivo
preservando o hard do host, e grava o drop-in de sysctl.

`vm.swappiness` é **derivado** da topologia medida (150 com zram ativo, 10 só em
disco) em vez de declarado — é a correção da contradição anterior, em que o perfil
instalava zram e fixava 10, dizendo ao kernel para não usar o dispositivo criado.

CachyOS garante o pacote `zram-generator` quando ausente e preserva o tuning já
existente. Nenhum dos perfis toca governor, scheduler, mitigations, kernel
cmdline ou serviços sem relação; remoção de pacotes é **opt-in**
(`--allow-debloat`) e usa a lista medida em `manifests/debloat_linux.yaml`.

Detalhes por item, evidência que justifica cada valor e o comando exato de
reversão: [`ubuntu-server-baseline.md`](ubuntu-server-baseline.md).

---

## 🌐 5. Orquestração de Subagentes OpenCode na VPS

O `envctl` transforma qualquer VPS remota em um **trabalhador autônomo de IA** — a mesma skill tree e os mesmos agentes, via SSH:

### 1. Preparação da VPS (Executado apenas uma vez):
```bash
# No notebook local, execute o provisionamento na VPS via SSH:
ssh minha-vps 'curl -fsSL https://raw.githubusercontent.com/eajdias/envctl/main/bootstrap.sh | bash'
```

### 2. Execução Não-Interativa de Tarefas (`opencode run`):
O OpenCode no Linux opera sem necessidade de ambiente gráfico:
```bash
# Exemplo 1: Diagnóstico de containers e rede
ssh minha-vps 'opencode run "Diagnosticar uso de disco e containers Docker com alto consumo de memória" < /dev/null'

# Exemplo 2: Execução de testes de carga em background
ssh minha-vps 'nohup opencode run "Executar testes de carga no endpoint /api/v1/auth e salvar resumo em /tmp/summary.md" < /dev/null > /tmp/agent.log 2>&1 &'
```

### 3. Automação de Browser Headless no Linux:
O `envctl` instala o runtime `bun` (`bunx`), o `playwright-cli` (via mise/npm, wrapper `pw`, com browsers próprios em `~/.cache/ms-playwright`) e provisiona o MCP `chrome-devtools` (`enabled: false` — ative por sessão via `/mcp`). O agente usa o MCP para o interativo e o CLI no shell para fluxos determinísticos (headless, sem display).
