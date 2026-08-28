# Guia de Execução & Provisionamento: Linux (Ubuntu / Debian / VPS)

Este guia orienta o uso do `envctl` em distribuições **Ubuntu (20.04 / 22.04 / 24.04 LTS)**, **Debian (11 / 12)**, **WSL2** ou instâncias de nuvem **AWS EC2 / Oracle Cloud Infrastructure**.

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
4. Executa `envctl run all` instalando pacotes via `apt-get`, Volta/Node, LSPs e implantando as 73 Skills de IA.
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

# 6. Execute o provisionamento completo
envctl run all
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
go run ./cmd/envctl run all

# 3. Ou compile o binário standalone
go build -ldflags "-s -w -X main.Version=v1.1.0" -o envctl ./cmd/envctl
sudo mv envctl /usr/local/bin/ # ou mv envctl ~/.local/bin/
```

---

## 🎛️ 4. Subcomandos Modulares no Linux

No Linux, comandos específicos de Windows (como `run winget`, `run windows`) são ignorados de forma limpa e segura:

```bash
# Apenas pacotes do sistema via APT (curl, git, ripgrep, fzf, jq, rsync, tree, etc.)
envctl run apt

# Apenas runtime Node.js LTS e CLIs globais via Volta
envctl run volta

# Apenas configurações de shell (.bashrc, aliases, git configs)
envctl run shell

# Apenas extração e validação das 73 Skills de Agentes
envctl run skills

# Apenas instalação dos servidores de linguagem (18 LSPs)
envctl run lsp

# Auditoria completa do ambiente
envctl doctor

# Auto-remediação de avisos e pendências
envctl doctor --fix
```

---

## 🌐 5. Orquestração de Subagentes OpenCode na VPS

O `envctl` transforma qualquer VPS remota em um **trabalhador autônomo de IA** via a skill `vps-agent-dispatch`:

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

### 3. Automação Headless com Playwright no Linux:
O `envctl` instala as dependências nativas de sistema (`sudo npx playwright install-deps chromium`) para que a extração web headless funcione de forma nativa:
```bash
pw-screenshot https://meu-servico-web.internal /tmp/dashboard.png
pw-eval https://meu-servico-web.internal "document.title"
```
