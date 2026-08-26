# Guia de Execução & Provisionamento: Windows 11 PRO

Este guia detalha todos os métodos de execução do `envctl` no **Windows 11 PRO (22H2 / 23H2 / 24H2 / 25H2)** ou **Windows Server 2022+**.

---

## ⚡ 1. Instalação e Execução Direta (Zero Pré-requisitos)

Se você acabou de formatar sua máquina Windows ou está configurando uma nova estação de trabalho, abra o **PowerShell** (como Usuário comum ou Administrador) e execute o comando de uma linha:

```powershell
irm https://raw.githubusercontent.com/eajdias/envctl/main/bootstrap.ps1 | iex
```

### O que o `bootstrap.ps1` faz de forma automatizada:
1. Detecta a arquitetura da sua máquina (`amd64` ou `arm64`).
2. Tenta baixar a release compilada mais recente diretamente via `gh release download` (se o GitHub CLI estiver autenticado) ou via download web.
3. Se nenhum binário estiver disponível, compila o código-fonte Go automaticamente se o Go estiver instalado.
4. Instala o executável em `~/.local/bin/envctl.exe` e adiciona ao seu `PATH` de usuário.
5. Inicia o provisionamento completo chamando `envctl run all` e executa a auditoria `envctl doctor`.

---

## 💻 2. Executando via Binário Pré-Compilado Standalone

Você pode baixar diretamente os executáveis gerados pelos releases automatizados do GitHub:

### Links de Download (Releases):
- **Windows x86_64 (AMD64)**: `envctl-windows-amd64.exe` (ou `envctl.exe`)
- **Windows ARM64**: `envctl-windows-arm64.exe`

### Passos de Execução no Terminal:
```powershell
# 1. Crie uma pasta para seus binários locais (se não existir)
New-Item -ItemType Directory -Force -Path "$HOME\.local\bin"

# 2. Mova o executável baixado para lá e renomeie para envctl.exe
Move-Item -Force .\envctl-windows-amd64.exe "$HOME\.local\bin\envctl.exe"

# 3. Adicione ao PATH da sessão atual (se ainda não estiver)
$env:Path = "$HOME\.local\bin;" + $env:Path

# 4. Execute a auditoria de saúde do sistema
envctl doctor

# 5. Execute o provisionamento completo
envctl run all
```

---

## 🛠️ 3. Compilação e Execução a Partir do Código-Fonte

Caso prefira clonar o repositório e compilar localmente:

### Pré-requisitos:
- **Go 1.22+** instalado.
- **Git** instalado.

### Comandos:
```bash
# 1. Clone o repositório
git clone https://github.com/eajdias/envctl.git
cd envctl

# 2. Execute diretamente sem compilar binário intermediário
go run ./cmd/envctl doctor
go run ./cmd/envctl run all

# 3. Ou compile o binário standalone otimizado
go build -ldflags "-s -w -X main.Version=v1.0.13" -o envctl.exe ./cmd/envctl

# 4. Ou utilize o Taskfile / Makefile
task build   # ou: make build
task doctor  # ou: make doctor
```

---

## 🎛️ 4. Subcomandos e Operações Modulares no Windows

Você pode executar etapas específicas conforme sua necessidade:

```bash
# Apenas pacotes do sistema via Winget (VSCode, Windows Terminal, Ripgrep, etc.)
envctl run winget

# Apenas runtime Node.js LTS e ferramentas globais via Volta
envctl run volta

# Apenas ajustes de Registro, Modo Desenvolvedor, Modo Escuro e Nerd Font
envctl run windows

# Apenas variáveis de ambiente (NODE_PATH, ENVCTL_TEMP) e arquivos de shell
envctl run shell

# Apenas catálogo de 59 Skills do OpenCode
envctl run skills

# Apenas servidores de linguagem (18 LSPs)
envctl run lsp

# Limpeza de acúmulo do OpenCode (cache duplicado, tool-output, scratch >24h)
envctl run cleanup

# Snapshot reverso (salva o estado atual da máquina de volta nos manifestos)
envctl snapshot
```

---

## ⚙️ 5. Particularidades e Ajustes Críticos no Windows

### A. Stack de Shell: PowerShell 7 (primário) + WSL Ubuntu (secundário)
O ambiente Windows é padronizado em **PowerShell 7** como shell default do OpenCode e do Windows Terminal.
- **PowerShell 7**: shell primário (OpenCode, Windows Terminal default, scripts de automação).
- **WSL Ubuntu 26.04**: subshell POSIX secundário para ferramentas Linux — use `wsl -e bash -lc "..."` quando um comando exigir ambiente Linux.
- Comandos Docker rodam nativamente do PowerShell, sem conversão de caminhos ou wrappers.

### B. Resolução Global de Módulos Node (`NODE_PATH`)
Para garantir que scripts autônomos (como Playwright) funcionem a partir de qualquer pasta de projeto:
- `NODE_PATH` aponta para `%USERPROFILE%\node_modules`.

### C. Pasta de Scratch Padrão dos Agentes LLM (`ENVCTL_TEMP`)
Todo arquivo temporário criado por agentes LLM (scripts Playwright, downloads, builds, screenshots) deve ir para `C:\temp` — pasta na raiz do disco, sem relação com o OpenCode, facilitando identificação e exclusão. `envctl run cleanup` remove scratch com mais de 24h.

### D. Automação Headless com Playwright
- Os comandos `pw-screenshot` e `pw-eval` ficam disponíveis instantaneamente em `~/.local/bin/`.
- Navegadores Chromium ficam armazenados em `%LOCALAPPDATA%\ms-playwright`.
