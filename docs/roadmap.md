# Roadmap — objetivos acordados, com o contexto já levantado

Este documento existe para que uma sessão futura (LLM ou humana) **retome daqui** em vez de
redescobrir o problema. Cada item traz o objetivo, o que já se sabe hoje e o que falta decidir.
Não é um plano fechado: a ordem sugerida está no fim e nenhum item tem prazo.

- **Antes de propor trabalho "novo"**, confira se ele já está aqui.
- Ao concluir um item, mova o resultado para `CHANGELOG.md` e para
  `docs/os-and-agent-matrix.md` (o roadmap é sobre o que **falta**).

---

## 1. Termux/Android como OS de primeira classe

**Objetivo:** o envctl provisionar e auditar um ambiente Termux, hoje fora do escopo
(`os:` aceita `windows`/`linux`/`darwin` + famílias `arch`/`debian`).

**O que já se sabe:**

- Termux é **userspace Linux comum** (`linux/arm64`), não Android nativo — um binário Go
  compilado para `linux/arm64` roda; `GOOS=android` (NDK/JNI) **não** é o caso.
- Sem `systemd` e sem `apt`/`pacman`: o gerenciador é o `pkg` (repositório próprio, sob
  `$PREFIX`, **sem root**).
- `$PREFIX` = `/data/data/com.termux/files/usr`, `HOME` = `/data/data/com.termux/files/home`
  → qualquer caminho absoluto assumido (`/usr/bin`, `/etc`, semântica de `~/.local/bin`)
  precisa ser revisto.
- Detecção de plataforma: **verificar** o que existe em `/etc/os-release` no Termux e garantir
  que a ausência **degrade** para um caminho POSIX genérico em vez de quebrar.
- `$PREFIX/bin` é um symlink farm (como `/usr/bin`) — a persistência de PATH (`.bashrc`/fish)
  continua válida; fish em Termux usa o mesmo `~/.config/fish`.
- Disponíveis via `pkg`: nodejs, fish, git, openssh, fzf, ripgrep, fd, bat, delta, yq, gh,
  python, golang. A confirmar: `volta` e os pacotes npm dos agentes em `linux/arm64`
  (`command-code`, e o instalador oficial do `opencode` — ver assimetria #9 da matriz).
- No lugar de systemd: `termux-services`/`termux-boot` (ver item 7).

**Falta decidir:** um valor `os: termux` vs. uma família de distro detectada por
`$PREFIX`/`TERMUX_VERSION`; e quais subsistemas têm caminho real de arm64.

## 2. Compatibilidade com o Termux: o que migrar

**Objetivo:** validar o projeto no Termux e decidir o que faz sentido provisionar lá.
Primeiro passo é **somente leitura** (`doctor` no Termux), depois a migração.

Auditoria por subsistema:

| Subsistema | Situação provável | Ação |
| :--- | :--- | :--- |
| Pacotes | só o que existe no `pkg` | filtro por `os:` no manifesto |
| Shell | fish + bash existem | reaproveitar `.profile`/`.bashrc`/fish |
| Configs | são texto portável | deve aplicar quase tudo |
| Skills | portáveis (texto) | aplicar integralmente |
| LSPs | vários sem binário arm64 | lista explícita de pulados |
| Tweaks / gaming | Windows/Arch-only | fora |
| Docker | sem daemon no Termux | fora |
| Verificação (`envctl-verify`) | depende das toolchains | roda o subconjunto presente |

**Entregável:** uma coluna Termux em `docs/os-and-agent-matrix.md` + uma tabela do que
**não** migra e por quê (nada de silêncio: skip explícito).

## 3. Skills de Tailscale e Cloudflared

**Objetivo:** skills novas para controlar/automatizar a camada de rede:
`tailscale status --json` / `up` / `down` / inventário de dispositivos / exit nodes e notas
de ACL; e `cloudflared` (criar/listar túneis, `tunnel route dns`, rodar como serviço com
credenciais 0600).

- **Status 2026-09-24:** skill `tailscale` absorvida do inventário CachyOS (portátil, sem PII);
  `cloudflared` segue pendente.

- Onde: `configs/skills/<nome>/` + `manifests/skills.yaml` (checklist na §5 da matriz).
- Hoje a camada de rede **não** tem skill; o acesso remoto existe pelo skill de SSH/ssh-manager.
- Receita que vale documentar: "expor uma porta local para o tailnet **ou** para a internet
  com segurança, e como derrubar depois".
- Segredos (auth key, credencial de túnel) **nunca** no repositório: documentar o armazenamento
  em `~/.config/opencode/secrets/` (0700) ou no keyring do OS.

## 4. SSH entre os OS: verificação profunda

**Objetivo:** provar com evidência que o SSH funciona em **cada direção** usada de verdade
(Windows ↔ Linux ↔ VM/VPS ↔ celular), em vez de assumir.

Testar e registrar: presença/permissão de chave (0600/0700), agent forwarding, algoritmos de
host key e higiene do `known_hosts`, keepalive e reuso de conexão (`ControlMaster`),
`RemoteCommand`, `BatchMode` (é o modo que o agente usa), e o `sshd` do Termux na porta
**8022** (não privilegiada — os configs atuais assumem 22).

Ponto de atenção: o Android mata processos em background → o `sshd` do Termux morre sem
`termux-wake-lock`; isso precisa estar documentado (e testado) antes de contar com o celular.

**Entregável:** matriz "origem → destino" com o comando exato e o resultado observado, mais a
correção do que falhar (inclusive nos templates de `~/.ssh/config`).

## 5. Provedor local controlando provedor remoto via SSH

**Objetivo:** o agente na estação (opencode/commandcode) despachar trabalho para os agentes
das outras máquinas por SSH, com evidência.

- Baseline: já existem o skill de dispatch remoto e o inventário SSH
  (`~/.ssh-manager`, `ssh_servers.md`). Comece por eles.
- Verificar no destino: PATH/toolchain **não-interativo** (`command -v cmdc/opencode`, shims do
  Volta), configs do agente presentes após o provisionamento (skills/LSP), execução longa
  (`nohup`/`tmux`/`systemd-run`), retorno de artefatos (`scp`/`rsync`) e os modos de falha
  (prompt de host key travando o agente, multiplexação, retry).
- **Entregável:** um cenário de teste por par origem→destino e as correções no skill de
  dispatch e/ou no bootstrap remoto do envctl.

## 6. Renomear o projeto para algo único

**Motivo:** `envctl` é genérico e colide com outros projetos de mesmo nome, o que atrapalha
busca, publicação e identidade do binário. Antes de escolher, checar disponibilidade em:
GitHub (org/repo), npm, PyPI, crates.io, AUR e winget.

O rename é mecânico mas atravessa: módulo Go (`github.com/eajdias/envctl`), diretório
`cmd/envctl`, `bootstrap.ps1`/`bootstrap.sh` (URLs), nome dos artefatos de release, binários
instalados (`~/.local/bin/envctl`, `envctl-verify`), o diretório de estado `~/.envctl/`
(hooks globais e logs), toda a documentação e **os prompts/skills dos agentes que chamam o
envctl**.

**Sugestão:** um commit mecânico só para isso + alias/symlink de compatibilidade por uma
release, para não quebrar máquinas já provisionadas.

## 7. Instalação local como serviço de background

**Objetivo:** nas máquinas usadas como serviço, manter o ambiente convergido e o próprio
envctl atualizado **sem sessão manual**: auditoria periódica e auto-update dentro de uma
whitelist segura.

- Linux: unit + timer de `systemd` (de preferência `--user`).
- Windows: Task Scheduler (ou serviço via WinSW/NSSM) — o Day-0 já tem `bootstrap.ps1`.
- Termux: `termux-services`/`termux-boot` (depende do item 1).
- **Cuidados:** single-flight com lockfile (o timer não pode competir com um `run` manual),
  logging em `~/.envctl/*.log` (já existe) e **nunca** auto-`--fix` destrutivo silencioso — o
  serviço audita e reporta; corrige só o que estiver numa whitelist.
- **Entregável:** arquivos de unit + um comando de instalação/remoção (ex.:
  `envctl self-install --service`) com testes.

## 8. Agenda curta (já discutida, não bloqueia)

- **CLIs extras de dev** (lazygit, `npm-check-updates`/`ncu`; avaliar `xh`, `duf`): entram no
  manifesto de **pacotes** com `check_command` e auditoria — não na fase 0, que é só provedores.
  **Status 2026-09-24:** `lazygit`, `lazydocker` e `duf` absorvidos do inventário CachyOS
  (pacman, `os: arch,cachyos`); `ncu`/`xh` seguem pendentes.
- **Skills por banco** (PostgreSQL/pgvector, MySQL, Redis, SQLite): decisão registrada —
  nenhum cliente/CLI global; cada banco ganha a sua skill quando aparecer a necessidade.
- **`ty`** (Astral) como segundo type checker Python, ao lado do mypy (ver §4 da matriz).
- **Windows:** rodar `envctl run windows` numa sessão Windows para validar o tipo `PSModule`
  (PSScriptAnalyzer + Pester) — pendente de máquina Windows.
- **Windows:** avaliar tweaks de telemetria/Game Bar (desbloat), hoje limitados a 6 DWords de
  Explorer/tema + Developer Mode.

---

**Ordem sugerida:** 4 → 5 (SSH e dispatch remoto são a base) → 3 (skills de rede em cima
disso) → 7 (serviço, começa a render autonomia) → 1/2 (Termux) → 6 (rename por último, quando
o escopo parar de mudar).
