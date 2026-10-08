# Primeiros Passos: instalando e usando o envctl

> Para quem nunca usou o projeto. Este guia cobre o essencial: o que o `envctl`
> faz, o comando de instalação, como confirmar que deu certo e o uso do dia a
> dia. Para detalhes técnicos, veja os links no final.

## O que é isto, em 3 linhas

O `envctl` deixa uma máquina nova pronta para trabalhar com agentes de IA:
instala os programas do dia a dia (git, ripgrep, Docker, Node.js, Python, Go…),
ajusta o sistema e configura os agentes **OpenCode** e **CommandCode** com as
skills deste repositório.

Suporta **Windows 11**, **Ubuntu/Debian** e **Arch/CachyOS** (64 bits).

## ⚠️ Antes de instalar: o que ele mexe na máquina

- Instala dezenas de programas — alguns passos podem pedir permissão elevada
  (UAC/Administrador no Windows, `sudo` no Linux).
- Configura terminal/shell, variáveis de ambiente e pastas de trabalho.
- **Windows (perfil completo):** tweaks de registro + **debloat** (desativa
  telemetria e alguns recursos, remove aplicativos pré-instalados) + pacotes.
- **Linux (perfil completo):** pacotes + tuning de performance (sysctl/zram) +
  serviços (ex.: Docker). No CachyOS, também a stack de gaming.
- Configura os agentes de IA (OpenCode/CommandCode) e as skills do repositório.
- Nenhum segredo/token fica no repositório — chaves locais ficam fora do git.
- É **idempotente**: pode rodar de novo quando quiser; o que já está certo não
  é refeito.

> O projeto é **opinativo**: ele configura o ambiente do jeito que este
> repositório documenta — não é um instalador genérico que pergunta tudo. Se
> quiser aplicar só partes, veja "Quero só uma parte" abaixo.

## 🚀 1. Instalação (comando único)

### Windows 11 (PowerShell)

```powershell
irm https://raw.githubusercontent.com/eajdias/envctl/main/bootstrap.ps1 | iex
```

### Linux (Ubuntu / Debian / servidor / desktop)

```bash
curl -fsSL https://raw.githubusercontent.com/eajdias/envctl/main/bootstrap.sh | bash
```

O script detecta a sua máquina, baixa o programa mais recente, instala e roda o
**perfil completo do sistema** (`run all`). São muitos pacotes — **pode
demorar** dependendo da rede e da máquina.

### Quero só uma parte (sem o perfil completo)

Windows:

```powershell
& ([scriptblock]::Create((irm https://raw.githubusercontent.com/eajdias/envctl/main/bootstrap.ps1))) -Subsystem shell
# troque "shell" pelo subsistema desejado: shell, skills, lsp, mise, pip,
# winget, tweaks, cleanup… (lista completa no README)
```

Linux:

```bash
curl -fsSL https://raw.githubusercontent.com/eajdias/envctl/main/bootstrap.sh | bash -s -- run shell
```

## ✅ 2. Deu certo? (verificação em 1 minuto)

1. **Abra um terminal novo** — as mudanças de PATH/shell só valem em sessões
   novas.
2. Rode a auditoria:

```bash
envctl doctor
```

O rodapé da tabela mostra a saúde geral: **EXCELLENT** = tudo certo (0 avisos,
0 erros). Se aparecer algum aviso, cada linha traz a dica do que fazer (ex.:
`run 'envctl run X'`).

3. Se algo falhar, o log completo fica em `~/.envctl/logs/`.

## 🔧 3. No dia a dia

```bash
envctl doctor          # audita a saúde da máquina (só relata)
envctl doctor --fix    # corrige o que for possível automaticamente
envctl run all         # reaplica o perfil completo do seu sistema
envctl update          # atualiza as ferramentas gerenciadas (mise/uv)
envctl run cleanup     # limpa caches e temporários dos agentes
envctl --help          # lista todos os comandos
```

## 🩹 4. Problemas comuns

| Sintoma | O que fazer |
| :--- | :--- |
| `envctl` não é reconhecido | Abra um terminal novo (o PATH foi atualizado, mas o shell atual não percebe). |
| Aviso do `mise` sobre versão nova | Informativo — pode ignorar. |
| Terminal diferente após instalar | Feche e abra de novo (configs de shell novas). |
| Instalou a partir de um clone do repo | Faça o rebuild antes de rodar (ver [provisioning](provisioning.md) §3). |
| Quero voltar atrás | Arquivos alterados ganham backup automático (`.bak` com data/hora); os itens mais agressivos do Windows estão no [guia de debloat](windows-debloat-tier3.md). |

## 📚 5. Próximos passos

- [Guia de Provisionamento](provisioning.md) — outras formas de instalar
  (binário pré-compilado, fonte, por subsistema).
- [Guia Windows 11](windows.md) · [Guia Linux](linux.md) ·
  [Guia CachyOS Gaming](cachyos-gaming.md)
- [Doctor, idempotência e logs](../doctor-and-idempotency.md)
- [README](../../README.md) — visão completa do projeto.
