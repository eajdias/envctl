# Guia de Provisionamento: instalação em qualquer OS

Para quem nunca usou o projeto, comece pelo
[Primeiros Passos](getting-started.md).

Instalação em uma passada só (scripts) ou passo a passo (binário / fonte).
O que é específico de cada OS depois da instalação continua nos guias
próprios: [`linux.md`](linux.md) (§§4–5: subcomandos, performance, subagentes)
e [`windows.md`](windows.md) (§§4–5: subcomandos, shell stack, browser).

## ⚡ 1. Instalação direta (zero pré-requisitos)

### Linux (Ubuntu Server / VPS / desktop)

```bash
curl -fsSL https://raw.githubusercontent.com/eajdias/envctl/main/bootstrap.sh | bash
```

O `bootstrap.sh`:

1. Identifica a arquitetura (`x86_64` → `amd64`, `aarch64` → `arm64`).
2. Baixa o binário standalone da release mais recente (`envctl-linux-amd64` ou `envctl-linux-arm64`).
3. Instala em `~/.local/bin/envctl` (`+x`) e exporta o `PATH`.
4. Executa `envctl run all` (perfil completo do sistema detectado) — rode
   `envctl doctor` em seguida para conferir a saúde. Com argumentos, executa o
   que você pedir (ex.: `bash -s -- run shell`).

### Windows 11 (PowerShell, usuário comum ou admin)

```powershell
irm https://raw.githubusercontent.com/eajdias/envctl/main/bootstrap.ps1 | iex
```

O `bootstrap.ps1`:

1. Detecta a arquitetura (`amd64` ou `arm64`).
2. Baixa a release compilada mais recente via `gh release download` (GitHub CLI autenticado) ou download web; sem binário disponível, compila do fonte se o Go existir.
3. Instala em `~/.local/bin/envctl.exe` e adiciona ao `PATH` de usuário.
4. Executa `envctl run all` (perfil completo do sistema detectado) — rode
   `envctl doctor` em seguida para conferir a saúde. Com o parâmetro
   `-Subsystem`, executa apenas a parte pedida (ex.: `-Subsystem shell`).

## 💻 2. Via binário pré-compilado standalone

### Linux

| Arch | Asset da release |
| :--- | :--- |
| x86_64 (AMD64) | `envctl-linux-amd64` |
| ARM64 (aarch64) | `envctl-linux-arm64` |

```bash
mkdir -p ~/.local/bin
curl -fsSL -o ~/.local/bin/envctl https://github.com/eajdias/envctl/releases/latest/download/envctl-linux-amd64
chmod +x ~/.local/bin/envctl
export PATH="$HOME/.local/bin:$PATH"
envctl doctor
envctl run vps   # perfil servidor; desktop CachyOS usa `run cachyos`
```

### Windows

| Arch | Asset da release |
| :--- | :--- |
| x86_64 (AMD64) | `envctl-windows-amd64.exe` |
| ARM64 | `envctl-windows-arm64.exe` |

```powershell
New-Item -ItemType Directory -Force -Path "$HOME\.local\bin"
Move-Item -Force .\envctl-windows-amd64.exe "$HOME\.local\bin\envctl.exe"
$env:Path = "$HOME\.local\bin;" + $env:Path
envctl doctor
envctl run windows   # perfil workstation (tweaks + debloat + tudo)
```

## 🛠️ 3. A partir do código-fonte

Pré-requisito: toolchain Go (ver `go.mod` para a versão).

### Linux

```bash
git clone https://github.com/eajdias/envctl.git
cd envctl
go run ./cmd/envctl doctor
go run ./cmd/envctl run vps
# ou o binário otimizado:
go build -ldflags "-s -w -X main.Version=$(git describe --tags --always)" -o envctl ./cmd/envctl
```

### Windows

```powershell
git clone https://github.com/eajdias/envctl.git
cd envctl
go run ./cmd/envctl doctor
go run ./cmd/envctl run windows
# ou o binário otimizado:
go build -ldflags "-s -w -X main.Version=v1.0.13" -o envctl.exe ./cmd/envctl
# ou via Makefile:
make build
make doctor
```
