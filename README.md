# envctl

> **Cross-Platform Environment Provisioner & Autonomous State Replicator**  
> Provisionador determinístico, idempotente e auditável em **Go** para replicação 1:1 de ambientes de desenvolvimento e servidores.

O projeto evoluiu de um script inicial de prototipagem focado exclusivamente em Windows 11 (`win11-new`) para um **ecossistema de provisionamento declarativo e orquestrador de subagentes autônomos multiplataforma em Go (`envctl`)**, cobrindo estações de trabalho e servidores remotos (Windows 11 PRO e Ubuntu / Debian Linux na AWS e Oracle Cloud).  Ele transforma qualquer VPS remota em um trabalhador autônomo de IA, com OpenCode/CommandCode e as mesmas skills do ambiente local.

---

## ⚡ Início Rápido (Comando de 1 Linha)

### 🪟 Windows 11 (PowerShell)
```powershell
irm https://raw.githubusercontent.com/eajdias/envctl/main/bootstrap.ps1 | iex
```

### 🐧 Linux (Ubuntu / Debian / Servidores VPS)
```bash
curl -fsSL https://raw.githubusercontent.com/eajdias/envctl/main/bootstrap.sh | bash
```


---

## 🎯 O que o `envctl` Configura Automaticamente?

- **Shell & Utilitários de Alta Performance**: PowerShell 7 (primário) + WSL Ubuntu (secundário) com `ripgrep`, `fd`, `fzf`, `bat`, `delta`, `tree`, `yq`, `jq`, `rsync`.
- **Toolchains Completas**: Node.js LTS (via Volta), Python 3.14 (`uv` + `ruff`), Go (`golangci-lint` incluso), Docker CLI, Cursor IDE (Windows/Arch).
- **Language Server Protocol (15 binários LSP p/ shell/IDE)**: TypeScript, Pyright, Gopls, Bash-LS, Sqllens, Dockerfile, TOML, PowerShell, etc. (bloco `lsp` removido do `opencode.json` — runtime v2 ignora LSP; diagnósticos do agente via lint/typecheck).
- **Ecossistema OpenCode & CommandCode com 12 Skills**: `opencode.json`, plugins e **12 skills curadas** — só entra no catálogo o que o modelo não faria sozinho; o conhecimento por stack/tool fica em `code-playbooks/references/`. Suporte equivalente a **CommandCode** (agente `code-reviewer`, MCPs, configs) — diferenças de plataforma documentadas na [tabela de paridade](docs/skills.md).
- **Automação Web em Dois Trilhos**: MCP `chrome-devtools` para o interativo (2FA manual, inspeção ao vivo, opt-in por sessão) + `pw` (wrapper versionado de `playwright-cli`, via volta) para automação determinística e token-efficient no shell, sem travar o agente — cada um com seu próprio build de browser, sem conflito com o navegador do usuário.
- **Temp Hygiene & Cleanup Subsystem**: Gerenciamento de diretórios temporários (`C:\temp`, `/temp`), rotação de logs e limpeza de cache/DB/tool-output do OpenCode via `envctl run cleanup`.
- **Quality Gates Locais**: verificador único conectado ao hook `Stop` do CommandCode (diagnóstico de volta ao modelo no mesmo turno) e a um pre-push global do git. Ele detecta a stack do repositório (Go, Node/TS, Python, SQL, shell, Docker, PowerShell), roda linters nos **arquivos alterados** e type checks/testes no repo inteiro. Findings de lint/formatação são informativos; scripts explícitos, builds, vets e testes continuam bloqueantes. Veja [docs/verification.md](docs/verification.md).
- **Worktree por padrão**: worktrees em `.worktrees/<type>-<slug>` e `doctor` reportando `prunable`/`locked`.

---

## 🤖 Workflow Inteligente Multi-VPS (OpenCode)

O envctl transforma o OpenCode local num **orquestrador de frotas**: cada VPS/VM nova vira um agente autônomo, sem configuração manual.

**Fluxo completo (o agente LLM local já sabe fazer):**

1. **Instalação no Windows (recomendado):** `irm https://raw.githubusercontent.com/eajdias/envctl/main/bootstrap.ps1 | iex` → `envctl run all` → `envctl doctor` (Day-0 local, idempotente).
2. **Adicionar VPS/VM nova:** peça ao agente para cadastrar a conexão SSH — ele registra seguindo os padrões (ssh-manager + inventário local `~/.config/opencode/extras/ssh_servers.md`, referência `code-playbooks/references/infra.md`) e **roda o envctl na VPS** (`curl -fsSL .../bootstrap.sh | bash` → `envctl run all`).
3. **Controle:** a VPS passa a ter o próprio OpenCode (plano **Free**) + as mesmas skills; tarefas pesadas podem ser despachadas do local por SSH, mantendo o contexto local enxuto.
4. **Limite Free estourado na VPS:** o orquestrador **PERGUNTA** se você quer registrar um TOKEN (`opencode auth login` na VPS). **Se você não quiser, ele executa os comandos por conta própria via SSH** — a orquestração nunca fica bloqueada.

---

## 💻 Comandos Principais

```bash
# Provisionamento completo do ecossistema (Day-0)
envctl run all

# Agentes — cada um provisiona só o que é dele (não toca o outro)
envctl commandcode      # CommandCode: settings, AGENTS.md, MCP, agentes, SKILL-INDEX e skills
envctl opencode         # OpenCode: opencode.json, plugins, memory seeds, SKILL-INDEX e skills

# Auditoria e diagnóstico de saúde de todo o ecossistema
envctl doctor

# Auto-remediação automática de qualquer divergência
envctl doctor --fix

# Provisionamento por subsistema modular
envctl run providers    # Fase 0: Volta, Node e os CLIs OpenCode/CommandCode prontos e atuais
envctl run winget       # Pacotes Winget (Windows)
envctl run apt          # Pacotes APT (Debian/Ubuntu)
envctl run volta        # Node.js e ferramentas globais
envctl run shell        # Variáveis de ambiente, perfis e configs
envctl run skills       # Extração e sincronização das skills
envctl run lsp          # 15 Servidores de Linguagem (LSP)
envctl run performance  # Perfil opt-in Ubuntu 24.04+ ou CachyOS (zram + auditoria/sysctl)
envctl run windows      # Tweaks de registro, Developer Mode e fontes
envctl run cleanup      # Limpeza de cache/DB/tool-output do OpenCode

# Snapshot reverso e sincronização de estado (Day-2)
envctl snapshot
```

---

## 📚 Documentação Técnica Completa

Para guias passo a passo detalhados, arquitetura e especificações:

### 📖 Guias de Execução por Sistema Operacional:
- 🪟 [**Guia Windows 11 PRO**](docs/guides/windows.md) — Instalação via PowerShell, binários `.exe`, ajustes de registro, PowerShell 7 + WSL Ubuntu.
- 🐧 [**Guia Linux (Ubuntu/Debian/VPS)**](docs/guides/linux.md) — Execução em servidores remotos, instâncias AWS/Oracle, orquestração de subagentes e WSL2.
- 🎮 [**Guia CachyOS Gaming**](docs/guides/cachyos-gaming.md) — Kernel cmdline, scheduler, GPU/AMD, MangoHud, Proton e emuladores (o que o `run gaming` provisiona e o que é manual).
- 🧹 [**Guia Windows Debloat Tier 3**](docs/guides/windows-debloat-tier3.md) — OneDrive, energia, Teredo, Docker/WSL: o que é destrutivo e por isso manual.

### 🏛️ Engenharia & Especificações:
- 🏗️ [**Arquitetura de Software**](docs/architecture.md) — Clean Architecture, camadas internas, abstração de I/O e binário standalone (`//go:embed`).
- 📋 [**Manifestos Declarativos**](docs/manifests.md) — Estrutura e customização dos schemas YAML (`packages.yaml`, `shell.yaml`, `git.yaml`, `performance_*.yaml`, `lsp.yaml`, `windows.yaml`).
- 🤖 [**Catálogo de Skills & Subagentes**](docs/skills.md) — As 12 skills, o critério de entrada no catálogo, roteamento de subagentes e convenção de worktree.
- ©️ [**Atribuição de Skills**](docs/skills-attribution.md) — De onde veio cada skill adotada de terceiros (autor + repositório), o que foi adaptado e como creditar skill nova.
- 🩺 [**Doctor, Idempotência & Logs**](docs/doctor-and-idempotency.md) — diagnóstico de todo o ecossistema (pacotes, configs, skills, agentes, LSPs, ambiente), flag `--fix`, backups atômicos (`.bak.timestamp`) e trilha de auditoria em `~/.envctl/logs/`.
- ✅ [**Verificação Local**](docs/verification.md) — os quality gates rodados na máquina: hook `Stop` do CommandCode, pre-push global do git, checks por stack e variáveis de controle.
- 🧭 [**Matriz OS × Agente**](docs/os-and-agent-matrix.md) — o que é provisionado em cada OS (Windows/Ubuntu/Arch) e em cada agente (OpenCode/CommandCode), assimetrias conhecidas e checklist para adições novas.
- 🗺️ [**Roadmap**](docs/roadmap.md) — os objetivos acordados para o futuro (Termux/Android, skills de Tailscale/Cloudflared, SSH entre os OS, dispatch remoto, rename do projeto, envctl como serviço de background), cada um com o contexto já levantado.
- 📐 [**Princípios & Decisões Arquiteturais (ADRs)**](docs/principles.md) — Diretrizes de idempotência, isolamento e contratos de repositório.

---

## 💎 Princípios Fundamentais
1. **100% Standalone via `//go:embed`**: Todas as skills e templates residem dentro do próprio binário executável compilado.
2. **Idempotência Estrita**: Executar 1 ou 100 vezes produz o mesmo estado final estável sem reinstalações redundantes.
3. **Backup Atômico com Timestamp**: Arquivos modificados sofrem backup automático (`.bak.YYYYMMDD-HHMMSS`) caso haja divergência de hash.
4. **Logging Persistente Estruturado**: Trilha de auditoria completa gerada em `~/.envctl/logs/envctl-YYYYMMDD-HHMMSS.log`.
5. **Zero Segredos**: Chaves e credenciais nunca residem no repositório; permissões seguras são aplicadas via ACLs e POSIX permissions.
