# Guia de Execução & Provisionamento: macOS (Darwin)

Este guia orienta o uso do `envctl` no **macOS (13+ Ventura / Sonoma / Sequoia)** em arquiteturas **Apple Silicon (M1/M2/M3/M4 - ARM64)** ou **Intel (x86_64 - AMD64)**.

---

## ⚡ 1. Instalação e Execução Direta (Zero Pré-requisitos)

Abra o **Terminal** do macOS (Zsh / Bash) e execute:

```bash
curl -fsSL https://raw.githubusercontent.com/eajdias/envctl/main/bootstrap.sh | bash
```

### O que o `bootstrap.sh` faz no macOS:
1. Detecta se a máquina é Apple Silicon (`arm64`) ou Intel (`amd64`).
2. Baixa o binário standalone compilado (`envctl-darwin-arm64` ou `envctl-darwin-amd64`).
3. Instala o executável em `~/.local/bin/envctl` e exporta o `PATH`.
4. Executa `envctl run all` e roda a auditoria `envctl doctor`.

---

## 💻 2. Executando via Binário Pré-Compilado Standalone

### Links de Download (Releases):
- **macOS Apple Silicon (ARM64)**: `envctl-darwin-arm64`
- **macOS Intel (AMD64)**: `envctl-darwin-amd64`

### Passos de Instalação no Terminal:
```bash
# 1. Crie o diretório de binários
mkdir -p ~/.local/bin

# 2. Baixe o executável correspondente (exemplo para Apple Silicon)
curl -fsSL -o ~/.local/bin/envctl https://github.com/eajdias/envctl/releases/latest/download/envctl-darwin-arm64

# 3. Conceda permissão de execução
chmod +x ~/.local/bin/envctl

# 4. Se o macOS bloquear a execução pelo Gatekeeper:
xattr -d com.apple.quarantine ~/.local/bin/envctl

# 5. Execute a auditoria diagnóstica
envctl doctor

# 6. Execute o provisionamento
envctl run all
```

---

## 🛠️ 3. Compilação a Partir do Código-Fonte

Caso tenha o toolchain Go e Git instalados:

```bash
# 1. Clone o repositório
git clone https://github.com/eajdias/envctl.git
cd envctl

# 2. Execute diretamente
go run ./cmd/envctl doctor
go run ./cmd/envctl run all

# 3. Ou compile o binário standalone
go build -ldflags "-s -w -X main.Version=v1.0.13" -o envctl ./cmd/envctl
mv envctl ~/.local/bin/
```

---

## 🎛️ 4. Subcomandos no macOS

```bash
# Provisionar runtime Node.js LTS e ferramentas globais via Volta
envctl run volta

# Provisionar variáveis de ambiente e arquivos de shell
envctl run shell

# Extrair e sincronizar as 59 Skills do OpenCode
envctl run skills

# Provisionar os 16 servidores de linguagem (LSP)
envctl run lsp

# Executar diagnóstico de saúde
envctl doctor

# Auto-remediar inconsistências detectadas
envctl doctor --fix

# Snapshot e sincronização reversa
envctl snapshot
```
