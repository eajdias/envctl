# Catálogo de Skills de Agentes de IA & Orquestração Remota

O `envctl` embuta e sincroniza **59 Skills de Agentes Especialistas** projetadas para o ecossistema OpenCode. As skills fornecem instruções estruturadas, regras determinísticas, scripts utilitários e referências técnicas que capacitam agentes de IA a executar tarefas de engenharia complexas de ponta a ponta.

---

## 🎯 Por que Skills em vez de MCPs Pesados?

1. **Economia Massiva de Context Window**: Ferramentas MCP estáticas registram schemas JSON de centenas de parâmetros em *todo* início de sessão, consumindo de 10.000 a 30.000 tokens antes mesmo da primeira interação.
2. **Ativação On-Demand**: As Skills são carregadas sob demanda apenas quando a intenção do usuário ou a complexidade do problema exige o conhecimento especialista.
3. **Multiplataforma e Zero Sobrecarga de Rede**: Como os scripts e templates residem localmente em `~/.config/opencode/skills/`, não há latência de chamadas RPC de MCPs locais.

---

## 📋 Categorias das 59 Skills Catalogadas

### 1. Engenharia de Software & Arquitetura
- **`git-workflow`**: Estratégia de branches semânticas, Conventional Commits, ciclo de Pull Requests via `gh` CLI, resolução de conflitos de merge/rebase e gerenciamento de `git worktree`.
- **`database-ops`**: Gestão multi-banco dinâmica (PostgreSQL, MySQL, Firebird, MongoDB, SQLite) via Docker Compose e migrações ORM sem sobrecarga de MCPs estáticos.
- **`universal-test-runner`**: Execução unificada de testes e análise de cobertura de código para Go, Python (pytest), Node/TypeScript (vitest/jest), .NET e Rust.
- **`api-contract-design`**: Especificação, validação e linting de contratos OpenAPI 3.x, GraphQL SDL e Protobuf/gRPC.
- **`systematic-debugging`**: Metodologia científica para identificação e resolução de bugs antes de propor correções de código.
- **`verification-before-completion`**: Protocolo rigoroso de auto-verificação por evidências empíricas antes de finalizar tarefas.
- **`receiving-code-review`**: Raciocínio crítico para avaliação e implementação de feedbacks de code review.
- **`using-git-worktrees`**: Isolamento determinístico de workspaces através de árvores de trabalho do Git.
- **`writing-plans`**: Elaboração de especificações e planos de implementação passo a passo.

### 2. Orquestração de Infraestrutura & Servidores Remotos
- **`vps-agent-dispatch`**: Orquestrador autônomo que permite ao agente master no notebook delegar tarefas pesadas, builds longos e testes para instâncias remotas OpenCode em servidores VPS (AWS/Oracle) via SSH, trazendo de volta apenas o sumário técnico cristalizado.
- **`ssh-vps`**: Operação, monitoramento e recuperação de serviços (Docker, Systemd, PM2) em parque de servidores remotos.
- **`docker`**: Gerenciamento de containers locais e remotos, volumes, networks, compose stacks e logs.
- **`windows-admin`**: Administração avançada de sistemas Windows 11 (serviços, registro, tarefas agendadas, firewall, eventos).

### 3. Extração Web & Automação Inteligente (Firecrawl Suite)
- **`firecrawl-search`** & **`firecrawl-scrape`**: Busca web em tempo real e extração limpa de markdown.
- **`firecrawl-agent`**: Extração de dados estruturados com schemas JSON complexos.
- **`firecrawl-crawl`** & **`firecrawl-map`**: Mapeamento de sitemaps e rastreamento em massa de documentações.
- **`firecrawl-deep-research`** & **`firecrawl-research-papers`**: Síntese aprofundada de literatura científica e acadêmica (arXiv, PubMed, bioRxiv).
- **`firecrawl-interact`**: Interação com páginas dinâmicas (login, paginação, formulários, SPA).
- **`firecrawl-monitor`**: Monitoramento contínuo de alterações em páginas web com alertas inteligentes.

### 4. Automação de Navegador & Testes Visuais (Playwright)
- **`playwright-cli`**: Automação de navegador headless via Node.js API.
- **Utilitários de Linha de Comando Embutidos** (`~/.local/bin/`):
  - `pw-screenshot <url> [output.png]`: Captura instantânea de tela em alta resolução em menos de 1 segundo.
  - `pw-eval <url> [expression]`: Avaliação rápida de JavaScript e extração do DOM de páginas web sem necessidade de abrir browser gráfico.

### 5. Meta-Skills & Refinamento de IA
- **`grilling`** / **`grill-me`**: Entrevistas impiedosas de design de software para validação de premissas e tomada de decisões técnicas.
- **`ask-questions-if-underspecified`**: Esclarecimento proativo de requisitos antes de iniciar implementações ambíguas.
- **`stop-slop`**: Higienização e remoção de clichês e vícios de linguagem em respostas textuais de IA.
- **`skill-miner`** & **`skill-personalizer`**: Descoberta e customização de novos padrões de fluxo de trabalho para criação de novas skills.
- **`handoff`**: Compactação e sumarização de contexto de sessão para transferência transparente entre agentes.

---

## 🚀 Como Disparar Tarefas Remotas (`vps-agent-dispatch`)

Para executar uma tarefa em uma VPS remota sem sobrecarregar a memória do notebook:

```bash
# 1. Execução direta não-interativa do OpenCode na VPS via SSH
ssh <SERVER> 'opencode run "Diagnosticar uso de memória dos containers Docker e retornar tabela formatada" < /dev/null'

# 2. Execução desacoplada em background (tarefas longas)
ssh <SERVER> 'nohup opencode run "Executar suíte de testes de integração e gerar relatório em /tmp/report.md" < /dev/null > /tmp/agent.log 2>&1 &'

# 3. Coleta do resultado consolidado
ssh <SERVER> 'cat /tmp/report.md'
```
