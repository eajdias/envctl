# Catálogo de Skills de Agentes de IA & Orquestração Remota

O `envctl` embuta e sincroniza **40 Skills de Agentes Especialistas** (+ 1 skill built-in do opencode) projetadas para os ecossistemas **OpenCode e CommandCode**. As skills fornecem instruções estruturadas, regras determinísticas, scripts utilitários e referências técnicas que capacitam agentes de IA a executar tarefas de engenharia complexas de ponta a ponta.

> O mesmo conjunto é deployado em `~/.config/opencode/skills/` e `~/.commandcode/skills/`. A fonte única da verdade é `configs/skills/` + `manifests/skills.yaml` — edite lá e propague com `envctl run skills`.

---

## 🎯 Por que Skills em vez de MCPs Pesados?

1. **Economia Massiva de Context Window**: Ferramentas MCP estáticas registram schemas JSON de centenas de parâmetros em *todo* início de sessão, consumindo de 10.000 a 30.000 tokens antes mesmo da primeira interação.
2. **Ativação On-Demand**: As Skills são carregadas sob demanda apenas quando a intenção do usuário ou a complexidade do problema exige o conhecimento especialista.
3. **Multiplataforma e Zero Sobrecarga de Rede**: Como os scripts e templates residem localmente em `~/.config/opencode/skills/` e `~/.commandcode/skills/`, não há latência de chamadas RPC de MCPs locais.

---

## 📋 Categorias das Skills (40)

### 1. Engenharia de Software & Arquitetura (9)
- **`git-workflow`**: Estratégia de branches semânticas, Conventional Commits, ciclo de Pull Requests via `gh` CLI, resolução de conflitos de merge/rebase e gerenciamento de `git worktree`.
- **`database-ops`**: Gestão multi-banco dinâmica (PostgreSQL, MySQL, Firebird, MongoDB, SQLite) via Docker Compose e migrações ORM sem sobrecarga de MCPs estáticos.
- **`universal-test-runner`**: Execução unificada de testes e análise de cobertura para Go, Python (pytest), Node/TypeScript (vitest/jest), .NET e Rust.
- **`api-contract-design`**: Especificação, validação e linting de contratos OpenAPI 3.x, GraphQL SDL e Protobuf/gRPC.
- **`systematic-debugging`**: Metodologia científica para identificação e resolução de bugs antes de propor correções de código.
- **`verification-before-completion`**: Protocolo rigoroso de auto-verificação por evidências empíricas antes de finalizar tarefas.
- **`receiving-code-review`**: Raciocínio crítico para avaliação e implementação de feedbacks de code review.
- **`using-git-worktrees`**: Isolamento determinístico de workspaces através de árvores de trabalho do Git.
- **`writing-plans`**: Elaboração de especificações e planos de implementação passo a passo.

### 2. Orquestração de Infraestrutura & Servidores Remotos (6)
- **`vps-agent-dispatch`**: Orquestrador autônomo que permite ao agente master no notebook delegar tarefas pesadas, builds longos e testes para instâncias remotas OpenCode em servidores VPS (AWS/Oracle) via SSH, trazendo de volta apenas o sumário técnico cristalizado.
- **`ssh-vps`**: Operação, monitoramento e recuperação de serviços (Docker, Systemd, PM2) em parque de servidores remotos.
- **`vps-provisioning`**: Provisionamento e manutenção de VPS/VM com `envctl` (bootstrap de 1 linha + `run all` + `doctor`), idempotente e auditável — nunca configurar servidor à mão.
- **`docker`**: Gerenciamento de containers locais (Docker Desktop/WSL2), volumes, networks, compose stacks, logs e `pull`/`push` no Hub.
- **`windows-admin`**: Administração avançada de sistemas Windows 11 (serviços, registro, tarefas agendadas, firewall, eventos).
- **`aur-headless-install`**: Instalação de pacotes AUR em shells não-interativos (`makepkg` como usuário + `sudo pacman -U`), sem prompts de senha.

### 3. Contexto, Memória & Meta-Agente (13)
- **`agent-memory`**: Memória persistente de lições e patterns — carregada no início de toda tarefa e atualizada a cada aprendizado (`.opencode/memory/` no OpenCode; `AGENTS.md` no CommandCode).
- **`memory-promotion`**: Classifica cada aprendizado gravado e promove processos reutilizáveis a skills, removendo a entrada da memória (sem redundância memória ↔ skill).
- **`context7-auto`**: Busca obrigatória de documentação atualizada de bibliotecas/frameworks via MCP Context7 antes de escrever código.
- **`ask-questions-if-underspecified`**: Esclarecimento proativo de requisitos antes de iniciar implementações ambíguas.
- **`subagent-routing`**: Roteamento de delegação — quando delegar, qual subagente (`explore`/`general`), paralelo vs sequencial e quando **não** paralelizar.
- **`dispatching-parallel-agents`**: Mecânica de execução de tarefas independentes em paralelo, preservando o contexto do coordenador.
- **`parallel-agent-orchestration`**: Subagentes paralelos no **mesmo** repositório git — base comum primeiro, fronteiras disjuntas, integração final pelo orquestrador.
- **`handoff`**: Compactação e sumarização de contexto de sessão para transferência transparente entre agentes.
- **`grilling`**: Entrevista impiedosa de design (árvore de decisões em rodadas) para validar premissas antes de implementar.
- **`stop-slop`**: Higienização e remoção de clichês e vícios de linguagem em respostas textuais de IA.
- **`skill-miner`**: Mineração de histórico de sessões de agentes para descobrir skills candidatas.
- **`skill-generalizer`**: Preparação de skills locais/privadas para publicação (portabilidade, metadados, licenciamento).
- **`skill-personalizer`**: Auditoria e adaptação de skills recém-criadas/baixadas às preferências e ao ambiente do usuário.

### 4. Domínio & Aplicações (12)
- **`bulk-postgres-import`**: Import/upsert em massa no PostgreSQL (batch multi-VALUES, `ON CONFLICT`, dedupe) tolerante a latência alta (túnel SSH).
- **`docker-build-local-vps-deploy`**: Build de imagem Docker local + transporte (`save`/`load`) para VPS fraca que não aguenta build.
- **`docker-desktop-wsl-restart`**: Restart limpo do Docker Desktop no Windows quando o backend WSL2 falha (`Wsl/ExecError`, `backend.sock`).
- **`jwt-hs256-node`**: Implementação de JWT HS256 sem dependências externas em Node.js (`createHmac` + `timingSafeEqual` com hash duplo).
- **`lsp-smoke-test`**: Smoke test de LSP servers (`--stdio` com stdin fechado; nunca confiar em `--version`) antes de registrar na config.
- **`nextjs-standalone-deploy`**: Deploy de Next.js com `output: standalone` (Dockerfile multi-stage, `NEXT_PUBLIC` no build, `public/`) e migração para v16.
- **`phone-e164-normalization`**: Normalização de telefones BR para E.164 (DDI 55, ambiguidade do DDD 55 do RS, descarte registrado de malformados).
- **`playwright-prod-regression`**: Regressão Playwright contra produção sem mutar dados reais (empty states, timing, filtro de 401 esperados).
- **`simple-feature-flag`**: Feature flags simples e auditáveis (tabela `app_config` + API JWT + leitura no início de cada ciclo).
- **`web-dashboard-automation`**: Automação de dashboards/SPAs autenticados (interceptar o request real via Playwright, CSRF token, replicar chamadas).
- **`cachyos-gaming-setup`**: Tuning de CachyOS para games/emulação (kernel, scheduler, GPU, Steam, emuladores, perfis de controle).
- **`headless-gui-probe`**: Geração de configs e validação de apps GUI (Qt/SDL) sem display (offscreen/dummy, `timeout -k`, log scraping).

---

## 🌐 Automação de Navegador (MCP-only)

Browser automation usa exclusivamente os MCPs `playwright` (`bunx @playwright/mcp@0.0.79 --browser chrome`) e `chrome-devtools` (`bunx chrome-devtools-mcp@1.8.0 --no-usage-statistics`), ambos `enabled: false` (opt-in por sessão via `/mcp`). Não há skill de browser/CLI nem scripts utilitários — o bundle do Chrome acompanha os MCPs.

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

## 🔀 Paridade OpenCode ↔ CommandCode

As skills são deployadas idênticas para os dois agentes. O que **não** tem equivalente é por diferença de plataforma, não por lacuna acidental:

| Recurso | OpenCode | CommandCode |
|---|---|---|
| Skills | `~/.config/opencode/skills/` | `~/.commandcode/skills/` |
| Memória (lessons/patterns) | `~/.config/opencode/memory/` (seed enviado por `envctl`) | Não existe memory-dir — a memória é o `AGENTS.md` (tiers user/project) |
| LSP | Bloco `lsp` do `opencode.json` (18 servidores) | Sem configuração própria — usa o LSP do IDE conectado (`/ide` + `get_diagnostics`) |
| Context pruning | Plugin DCP (`dcp.jsonc`, banda 90%/80%) | Nativo (`/compact`, setting `compact-mode`) |
| Regras globais | `~/.config/opencode/AGENTS.md` (variantes Windows/Linux) | `~/.commandcode/AGENTS.md` (variantes Windows/Linux) |
