# Catálogo de Skills de Agentes de IA & Orquestração Remota

O `envctl` embuta e sincroniza **50 Skills de Agentes Especialistas** projetadas para os agentes **CommandCode** e **OpenCode**. As skills fornecem instruções estruturadas, regras determinísticas, scripts utilitários e referências técnicas que capacitam os agentes a executar tarefas de engenharia complexas de ponta a ponta.

> **Carga sob demanda — é o motivo de usar skill em vez de MCP.** A cada turno entra no prompt apenas o par *nome + descrição* de cada skill; o corpo do `SKILL.md` só é lido quando a tarefa casa ou quando você invoca `/<skill>`. Nenhuma skill é carregada antecipadamente.

> **Índice externo.** Cada agente recebe um `SKILL-INDEX.md` (tabela *situação → skill*) que o agente abre **apenas** se precisar decidir qual skill usar — não é auto-carregado. No CommandCode fica em `~/.commandcode/SKILL-INDEX.md`; no OpenCode em `~/.config/opencode/SKILL-INDEX.md`.

> **Deploy por agente.** Cada agente tem seu próprio comando (`envctl commandcode` / `envctl opencode`) e seu diretório (`~/.commandcode/skills/`, `~/.config/opencode/skills/`) — um comando não toca os arquivos do outro. A fonte única da verdade é `configs/skills/` + `manifests/skills.yaml`; propague com `envctl run skills`. Skills de ambiente específico declaram `os:` distro-strict no manifesto (`windows` · `arch,cachyos` · `debian,ubuntu`) e são **podadas** nas demais plataformas — 50 no manifesto, 46 deployadas no Windows, 46 no Ubuntu Server, 48 no CachyOS. O OpenCode recebe ainda um `REFERENCE.md` (detalhe operacional: plugins, DCP, agentes, memória, VPS) — também lido só sob demanda.

> **Créditos.** Boa parte destas skills foi adotada de projetos de terceiros (obra/superpowers, mattpocock/skills, hqhq1025/skill-optimizer, Hardik Pandya) — o autor e o repositório de origem estão no `metadata` de cada `SKILL.md`. Tabela completa, o que foi adaptado e o checklist para adotar skill nova: [**Atribuição de Skills**](skills-attribution.md).

---

## 🎯 Por que Skills em vez de MCPs Pesados?

1. **Economia Massiva de Context Window**: Ferramentas MCP estáticas registram schemas JSON de centenas de parâmetros em *todo* início de sessão, consumindo de 10.000 a 30.000 tokens antes mesmo da primeira interação.
2. **Ativação On-Demand**: As Skills são carregadas sob demanda apenas quando a intenção do usuário ou a complexidade do problema exige o conhecimento especialista.
3. **Multiplataforma e Zero Sobrecarga de Rede**: Como os scripts e templates residem localmente em `~/.config/opencode/skills/` e `~/.commandcode/skills/`, não há latência de chamadas RPC de MCPs locais.

---

## 📋 Categorias das Skills (50)

### 1. Engenharia de Software & Arquitetura (15)
- **`git-workflow`**: Estratégia de branches semânticas, Conventional Commits, ciclo de Pull Requests via `gh` CLI, resolução de conflitos de merge/rebase e gerenciamento de `git worktree`.
- **`database-ops`**: Gestão multi-banco dinâmica (PostgreSQL, MySQL, Firebird, MongoDB, SQLite) via Docker Compose e migrações ORM sem sobrecarga de MCPs estáticos.
- **`universal-test-runner`**: Execução unificada de testes e análise de cobertura para Go, Python (pytest), Node/TypeScript (vitest/jest), .NET e Rust.
- **`test-driven-development`**: Ciclo TDD red-green-refactor em TS/Node, Python e Go — teste falhando primeiro, código mínimo depois (ideia de obra/superpowers, ciclo e exemplos adaptados).
- **`api-contract-design`**: Especificação, validação e linting de contratos OpenAPI 3.x, GraphQL SDL e Protobuf/gRPC.
- **`systematic-debugging`**: Metodologia científica para identificação e resolução de bugs antes de propor correções de código.
- **`variant-analysis`**: Caça às outras instâncias de um bug já encontrado — variantes da mesma causa raiz no resto do código (original, sem upstream).
- **`verification-before-completion`**: Protocolo rigoroso de auto-verificação por evidências empíricas antes de finalizar tarefas.
- **`receiving-code-review`**: Raciocínio crítico para avaliação e implementação de feedbacks de code review.
- **`writing-plans`**: Elaboração de especificações e planos de implementação passo a passo.
- **`docs-sync`**: Auditoria/atualização da documentação contra a implementação real (gaps, incorreções, docstrings TS/PY/GO) — audit-only por default (ideia de openai/openai-agents-python, workflow adaptado).
- **`go-development`**: Desenvolvimento Go idiomático — APIs, erros, context, concorrência, portabilidade, testes e race detector.
- **`python-development`**: Desenvolvimento Python — ambientes, módulos, tipos, async, packaging, pytest, Ruff e type checkers.
- **`mcp-tool-design`**: Design e revisão de servidores MCP — tools, resources, prompts, schemas, segurança, compatibilidade e testes.
- **`frontend-markup`**: HTML, CSS e SCSS semânticos, acessíveis, responsivos e verificados no navegador.

### 2. Orquestração de Infraestrutura & Servidores Remotos (9)
- **`vps-agent-dispatch`**: Orquestrador autônomo que permite ao agente master no notebook delegar tarefas pesadas, builds longos e testes para instâncias remotas OpenCode em servidores VPS (AWS/Oracle) via SSH, trazendo de volta apenas o sumário técnico cristalizado.
- **`ssh-vps`**: Operação, monitoramento e recuperação de serviços (Docker, Systemd, PM2) em parque de servidores remotos.
- **`tailscale`**: Tailnet ops — status/inventário via JSON, up/down, exit nodes, notas de ACL, expor porta local (`serve` tailnet-only ou `funnel` explícito) e teardown; segredos em `~/.config/opencode/secrets/`, nunca no repo.
- **`syncthing-ops`**: Sync de pastas via REST API (status, pasta nova, rescan, conflitos) — nunca editar `config.xml` na mão; sync não é backup.
- **`vps-provisioning`**: Provisionamento e manutenção de VPS/VM com `envctl` (bootstrap de 1 linha + `run all` + `doctor`), idempotente e auditável — nunca configurar servidor à mão.
- **`docker`**: Gerenciamento de containers locais (Docker Desktop/WSL2), volumes, networks, compose stacks, logs e `pull`/`push` no Hub.
- **`windows-admin`**: Administração avançada de sistemas Windows 11 (serviços, registro, tarefas agendadas, firewall, eventos).
- **`windows-debloat`**: Debloat opt-in do Windows 11 (`run debloat`, Tier 1-2 automáticos + Tier 3 manual) — telemetria, privacidade, gaming, Appx, serviços; `doctor` agrega por categoria, nunca `--fix`. `[win]`
- **`aur-headless-install`**: Instalação de pacotes AUR em shells não-interativos (`makepkg` como usuário + `sudo pacman -U`), sem prompts de senha.

### 3. Contexto, Memória & Operação Autônoma (15)
- **`agent-memory`**: Memória persistente de lições e patterns — consulte quando a tarefa parecer repetir algo já resolvido e registre quando aprender.
- **`memory-promotion`**: Classifica cada aprendizado gravado e promove processos reutilizáveis a skills, removendo a entrada da memória (sem redundância memória ↔ skill); aplica a regra de atribuição ao adotar skill de terceiro.
- **`context7-auto`**: Busca obrigatória de documentação atualizada de bibliotecas/frameworks via MCP Context7 antes de escrever código.
- **`technical-research`**: Pesquisa técnica com fontes primárias, versões explícitas, contradições e citações.
- **`grill-me`**: Porteiro de ambiguidade — mede de 0 a 100 a clareza do pedido e, acima do limiar, pergunta **antes** de executar.
- **`ask-questions-if-underspecified`**: A mecânica de perguntar bem — perguntas mínimas, opções com default, resposta compacta.
- **`subagent-routing`**: Roteamento e despacho de subagentes — **inline é o padrão**; só delega quando o contexto bruto da pesquisa/planejamento é volumoso e o retorno é compactável. Qual tipo (`explore`/`general`/`planner`), paralelo vs sequencial, mecânica de dispatch múltiplo na mesma resposta, e orquestração no mesmo repositório (fronteiras disjuntas, base comum, integração final).
- **`subagent-supervision`**: O coordenador vigia subagentes e interrompe pelo **id do runtime em uso** — CommandCode: `agent_output({agent_id, action:"kill"})` (o filho em background sobrevive ao cancelamento do turno pai, e abortar o `wait` não mata o agente); OpenCode V2: `opencode api post /api/session/<sessionID>/interrupt` (lá `sessionID` ≠ PID e não existe `agent_output`). Shell em background só no CommandCode (`run_in_background`, `shell_output`, `task_output`, `kill_shell`, `monitor_command`). Retry no máximo 1× com escopo refinado, depois escala; nunca `reset`/`clean` no worktree.
- **`task-hang-watchdog`**: Previne e recupera comando/tarefa presa com **o toolset do runtime**: timeout real (CommandCode: foreground 30s, clamp 10min, resto em background com log em disco e `shell_output`/`task_output`; OpenCode: `timeout` da tool shell), observada contínua via `monitor_command`+`monitor_events` (só CommandCode, sem poll), recuperação suave antes de forçada e preservação de log/worktree — encerramento por nome de processo nunca.
- **`handoff`**: Compactação e sumarização de contexto de sessão para transferência transparente entre agentes.
- **`grilling`**: Entrevista impiedosa de design (árvore de decisões em rodadas) para validar premissas antes de implementar.
- **`stop-slop`**: Higienização e remoção de clichês e vícios de linguagem em respostas textuais de IA.
- **`skill-miner`**: Mineração de histórico de sessões de agentes para descobrir skills candidatas.
- **`skill-generalizer`**: Preparação de skills locais/privadas para publicação (portabilidade, metadados, licenciamento).
- **`skill-personalizer`**: Auditoria e adaptação de skills recém-criadas/baixadas às preferências e ao ambiente do usuário.

### 4. Domínio & Aplicações (11)
- **`bulk-postgres-import`**: Import/upsert em massa no PostgreSQL (batch multi-VALUES, `ON CONFLICT`, dedupe) tolerante a latência alta (túnel SSH).
- **`jwt-hs256-node`**: Implementação de JWT HS256 sem dependências externas em Node.js (`createHmac` + `timingSafeEqual` com hash duplo).
- **`lsp-smoke-test`**: Smoke test de LSP servers (`--stdio` com stdin fechado; nunca confiar em `--version`) antes de registrar na config.
- **`nextjs-standalone-deploy`**: Deploy de Next.js com `output: standalone` (Dockerfile multi-stage, `NEXT_PUBLIC` no build, `public/`) e migração para v16.
- **`phone-e164-normalization`**: Normalização de telefones BR para E.164 (DDI 55, ambiguidade do DDD 55 do RS, descarte registrado de malformados).
- **`playwright-prod-regression`**: Regressão Playwright contra produção sem mutar dados reais (empty states, timing, filtro de 401 esperados).
- **`simple-feature-flag`**: Feature flags simples e auditáveis (tabela `app_config` + API JWT + leitura no início de cada ciclo).
- **`web-dashboard-automation`**: Automação de dashboards/SPAs autenticados (interceptar o request real via Playwright, CSRF token, replicar chamadas).
- **`cachyos-gaming-setup`**: Tuning de CachyOS para games/emulação (kernel, scheduler, GPU, Steam, emuladores, perfis de controle).
- **`linux-performance-tuning`**: Performance Linux por OS (Ubuntu Server 24.04+ e CachyOS), zram, sysctl, swap planejado, journald, scheduler, governor, benchmark e rollback.
- **`headless-gui-probe`**: Geração de configs e validação de apps GUI (Qt/SDL) sem display (offscreen/dummy, `timeout -k`, log scraping).

---

## 🌐 Automação de Navegador (chrome-devtools MCP + playwright-cli)

Browser interativo (exploratório, 2FA manual, inspeção ao vivo) via MCP `chrome-devtools` (`bunx chrome-devtools-mcp@1.8.0 --no-usage-statistics`, `enabled: false` — opt-in por sessão via `/mcp`; o agente escolhe quando usar). Automação determinística e token-efficient (regressões, fluxos repetíveis, extrações) via `pw` no shell — wrapper versionado (`~/.local/bin/pw.cjs`, provisionado pelo envctl) sobre o `playwright-cli` (`open`, `snapshot`, `click e15`, `screenshot`; `--headed` para acompanhar, headless em VPS/sem display) que evita o hang do Windows — skills `web-dashboard-automation` e `playwright-prod-regression` orientam o uso. Cada browser usa seu próprio build (Chrome do sistema/Chrome for Testing no chrome-devtools, Chromium bundled em `~/.cache/ms-playwright` no CLI) — sem conflito com o navegador do usuário.

---

## 🚀 Como Disparar Tarefas Remotas (`vps-agent-dispatch`)

Para executar uma tarefa em uma VPS remota sem sobrecarregar a memória do notebook:

```bash
# 1. Execução direta não-interativa do OpenCode na VPS via SSH
ssh <SERVER> 'opencode run "Diagnosticar uso de memória dos containers Docker e retornar tabela formatada" < /dev/null'

# 2. Execução desacoplada em background (tarefas longas)
ssh <SERVER> 'nohup opencode run "Executar suíte de testes de integração e gerar relatório em /temp/report.md" < /dev/null > /temp/agent.log 2>&1 &'

# 3. Coleta do resultado consolidado
ssh <SERVER> 'cat /temp/report.md'
```

---

## 🔀 CommandCode × OpenCode

O conjunto base de skills é o mesmo, mas cada agente tem seu próprio **command** de provisionamento, seu **índice** (`SKILL-INDEX.md`) e suas peculiaridades. O que não tem equivalente é diferença de plataforma, não lacuna acidental:

| Recurso | OpenCode | CommandCode |
|---|---|---|
| Skills | `~/.config/opencode/skills/` | `~/.commandcode/skills/` |
| Memória (lessons/patterns) | `~/.config/opencode/memory/` (seed enviado por `envctl`) | Não existe memory-dir — a memória é o `AGENTS.md` (tiers user/project) |
| LSP | Sem bloco `lsp` (removido 2026-09-22 — inerte no runtime v2; binários seguem provisionados p/ shell/IDE) | Sem configuração própria — usa o LSP do IDE conectado (`/ide` + `get_diagnostics`) |
| Context pruning | Nativo do v2 (`compaction`; DCP removido por YAGNI em 2026-09-22) | Nativo (`/compact`, setting `compact-mode`) |
| Regras globais | `~/.config/opencode/AGENTS.md` (variantes Windows/Linux) | `~/.commandcode/AGENTS.md` (variantes Windows/Linux) |
