# Guia de Execução & Provisionamento: Linux (Ubuntu Server / VPS)

Este guia orienta o uso do `envctl` em **Ubuntu Server 24 ou superior**
(inclusive 26.04 LTS) em instâncias de nuvem como **Oracle Cloud Infrastructure**
e **AWS EC2**.

O alvo do perfil de servidor é **exatamente Ubuntu Server `VERSION_ID >= 24`**.
Ubuntu 20.04/22.04, Debian e WSL2 **não são suportados** por esse perfil: o
`min_distro_version` vive no manifesto, e `envctl run vps` falha com o motivo
em vez de seguir e pular o tuning em silêncio. Para'Arquitectura desktop, veja
[`cachyos-gaming.md`](cachyos-gaming.md).

---

## ⚡ 1. Instalação e Execução Direta (Zero Pré-requisitos)

Em um servidor recém-criado ou na sua máquina Linux de desenvolvimento, execute no terminal Bash:

```bash
curl -fsSL https://raw.githubusercontent.com/eajdias/envctl/main/bootstrap.sh | bash
```

### O que o `bootstrap.sh` faz de forma automatizada:
1. Identifica a arquitetura (`x86_64` -> `amd64`, `aarch64` -> `arm64`).
2. Realiza o download do binário standalone correspondente da release mais recente do GitHub (`envctl-linux-amd64` ou `envctl-linux-arm64`).
3. Instala o executável com permissão `+x` em `~/.local/bin/envctl` e exporta o `PATH`.
4. Executa `envctl run vps` (perfil Ubuntu Server 24+: apt + performance + Volta/Node + LSPs + 12 skills de IA).
5. Roda a auditoria diagnóstica `envctl doctor`.

---

## 💻 2. Executando via Binário Pré-Compilado Standalone

Se preferir baixar o executável manualmente:

### Links de Download (Releases):
- **Linux x86_64 (AMD64)**: `envctl-linux-amd64`
- **Linux ARM64 (aarch64)**: `envctl-linux-arm64`

### Passos de Instalação no Terminal:
```bash
# 1. Crie a pasta de binários do usuário
mkdir -p ~/.local/bin

# 2. Baixe o executável standalone (exemplo para AMD64 via GitHub Release)
curl -fsSL -o ~/.local/bin/envctl https://github.com/eajdias/envctl/releases/latest/download/envctl-linux-amd64

# 3. Dê permissão de execução
chmod +x ~/.local/bin/envctl

# 4. Adicione ao PATH da sessão atual
export PATH="$HOME/.local/bin:$PATH"

# 5. Execute o diagnóstico de conformidade
envctl doctor

# 6. Execute o perfil completo do servidor (Ubuntu/Debian)
envctl run vps
```

---

## 🛠️ 3. Compilação a Partir do Código-Fonte

Caso tenha o toolchain Go instalado na máquina:

```bash
# 1. Clone o repositório
git clone https://github.com/eajdias/envctl.git
cd envctl

# 2. Execute diretamente
go run ./cmd/envctl doctor
go run ./cmd/envctl run vps

# 3. Ou compile o binário standalone
go build -ldflags "-s -w -X main.Version=v1.1.0" -o envctl ./cmd/envctl
sudo mv envctl /usr/local/bin/ # ou mv envctl ~/.local/bin/
```

---

## 🎛️ 4. Subcomandos Modulares no Linux

No Linux, comandos específicos de Windows (como `run winget`, `run tweaks`, `run debloat`) são ignorados de forma limpa e segura:

```bash
# Apenas pacotes do sistema via APT (curl, git, ripgrep, fzf, jq, rsync, tree, etc.)
envctl run apt

# Apenas runtime Node.js LTS e CLIs globais via Volta
envctl run volta

# Apenas configurações de shell (.bashrc, aliases, git configs)
envctl run shell

# Apenas extração e validação das 12 skills de agentes
envctl run skills

# Apenas instalação dos binários de linguagem p/ shell/IDE (15 LSPs; sem efeito no runtime opencode v2)
envctl run lsp

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
O `envctl` instala o runtime `bun` (`bunx`), o `playwright-cli` (via volta, com browsers próprios em `~/.cache/ms-playwright`) e provisiona o MCP `chrome-devtools` (`enabled: false` — ative por sessão via `/mcp`). O agente usa o MCP para o interativo e o CLI no shell para fluxos determinísticos (headless, sem display).
