---
name: envctl
description: Operar o envctl, o padronizador do ambiente de trabalho. Use para auditar a máquina (doctor), provisionar um PC ou VPS novo, instalar skills/configs de agente, ou quando o ambiente estiver fora do padrão.
license: MIT
---

# envctl — operar o ambiente

O envctl padroniza o ambiente de trabalho: pacotes, shells, configs, skills, LSPs,
agentes e quality gates, com provisionamento **idempotente** em Windows 11 e Linux
(Ubuntu/Debian, Arch/CachyOS), incluindo servidores remotos.

> **Escopo desta skill:** *como operar a ferramenta*. Como ela funciona por dentro
> (manifests, doctor, matrizes) é do repositório — leia lá, não aqui.

## Provisionar uma máquina nova

```powershell
# Windows 11
irm https://raw.githubusercontent.com/eajdias/envctl/main/bootstrap.ps1 | iex
```

```bash
# Linux (desktop, VPS, servidor)
curl -fsSL https://raw.githubusercontent.com/eajdias/envctl/main/bootstrap.sh | bash
```

Depois: `envctl run all` e `envctl doctor`. Rodar de novo é seguro (idempotente) e
é também o jeito de **atualizar** uma máquina já provisionada.

## Auditar a máquina atual

```bash
envctl doctor            # inventário completo: o que está fora do padrão
envctl doctor --fix      # corrige o que é corrigível com segurança
```

`doctor` é read-only por padrão; `--fix` só toca no que é idempotente e faz backup
atômico. O estado saudável é **0 WARN / 0 ERROR** — se aparecer warning, investigue
antes de seguir em frente.

## Operação por camada

```bash
envctl run all        # provisionamento completo (o padrão)
envctl run shell      # shell, env vars e configs de agente
envctl run skills     # só as skills dos agentes
envctl run lsp        # binários de language server
envctl run cleanup    # cache, logs, tool-output e tmp acumulada
envctl opencode       # só a camada OpenCode
envctl commandcode    # só a camada CommandCode
```

## Regras de segurança

- **`snapshot` é sync REVERSO (máquina → repo)** e **nunca** deve rodar numa VPS:
  ele existe para quando você editar algo à mão e quiser levar isso ao repo.
- `run all` é o caminho da primeira vez; numa máquina já provisionada, use a camada
  específica (`run shell`, `run skills`) para não gastar tempo à toa.
- Provisionamento é idempotente, mas `--fix` em serviço pode reiniciar processo:
  avise antes.
- O `doctor` é a fonte da verdade do estado da máquina. Quando a doc divergir do
  doctor, o doctor ganha.
- Repositório: `https://github.com/eajdias/envctl` — é lá que mora a documentação
  completa (manifests, matriz OS × agente, verificação, arquitetura).

## Quando algo falha

1. Rode `envctl doctor` **antes** de investigar: na maioria das vezes o diagnóstico
   já está na linha do warning.
2. Leia o log da execução em `~/.envctl/logs/envctl-YYYYMMDD-HHMMSS.log`.
3. Corrija, rode o comando de novo e confirme no `doctor`.
