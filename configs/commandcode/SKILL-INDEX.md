# CommandCode — índice de skills

**Consulte sob demanda.** Não leia este arquivo por padrão: abra-o apenas quando precisar decidir *qual* skill carregar para a tarefa em mãos. O catálogo de skills (nome + descrição) já vem no seu prompt automaticamente — não há nada a "ativar".

- **Skills instaladas:** `~/.commandcode/skills/<nome>/SKILL.md` — invoque com `/<skill>` (ou `/skill:<nome>`); a tool do modelo é `activate_skill`.
- **Marcadores:** `[win]` só existe em máquinas Windows · `[linux]` só existe em Linux · sem marcador = qualquer ambiente.
- **Regra de ouro:** se a tarefa é "como fazer X neste ambiente", veja se existe uma skill para X antes de improvisar.

| Situação | Skill |
|---|---|
| Escrever, alterar ou configurar código com lib/framework/API | `context7-auto` |
| Pesquisar ou verificar fato técnico atual com fontes, versões e citações | `technical-research` |
| Bug, teste vermelho, erro intermitente, comportamento inesperado | `systematic-debugging` |
| Achou um bug e quer saber onde mais o mesmo padrão ocorre | `variant-analysis` |
| Antes de declarar "pronto"/corrigido/passando (inclusive antes de commitar ou abrir PR) | `verification-before-completion` |
| Implementar algo multi-passo, feature nova ou refatoração ampla | `writing-plans` |
| Pedido ambíguo/vago — medir a clareza (0-100) e perguntar ANTES de agir | `grill-me` |
| Endurecer, questionar ou stress-testar um plano/decisão | `grilling` |
| Formular as perguntas de esclarecimento (opções, defaults, resposta curta) | `ask-questions-if-underspecified` |
| Receber code review ou comentários de PR | `receiving-code-review` |
| Commit, branch, PR, rebase, conflito, tag/release + isolamento via worktree | `git-workflow` |
| Roteamento e despacho de subagentes (quando/como delegar + mecânica paralela + mesmo repo) | `subagent-routing` |
| Schema, migration, query, backup/restore de banco | `database-ops` |
| Inserir/atualizar muitos registros no PostgreSQL | `bulk-postgres-import` |
| Rodar testes, medir cobertura, rodar benchmark | `universal-test-runner` |
| Feature nova, bugfix, refatoração em TS/PY/GO (teste antes do código) | `test-driven-development` |
| Go: módulo, APIs, concorrência, testes, race/vet/gofmt | `go-development` |
| Python: ambiente, módulos, async, tipos, packaging e pytest/ruff | `python-development` |
| Auditar/atualizar docs contra a implementação real | `docs-sync` |
| Desenhar/validar contrato de API (OpenAPI, GraphQL, gRPC) | `api-contract-design` |
| Projetar/revisar tools, resources, prompts, schemas e erros de servidor MCP | `mcp-tool-design` |
| Servidor/VPS remoto: monitorar, diagnosticar, reiniciar serviço | `ssh-vps` |
| Rede tailnet: status/inventário, exit node, expor porta (serve/funnel) | `tailscale` |
| Sync de pastas entre devices via API (status, pasta nova, conflitos) | `syncthing-ops` |
| Provisionar, atualizar ou auditar VPS/VM com envctl | `vps-provisioning` |
| Rodar tarefa pesada (build, suíte, crawler) numa VPS | `vps-agent-dispatch` |
| Containers, compose, imagens, volumes, Docker Hub, build local/transporte (VPS fraca) e restart WSL2 | `docker` |
| Windows: serviços, registro, tarefas agendadas, firewall, winget | `windows-admin` `[win]` |
| Windows: debloat opt-in (telemetria, Appx, serviços + Tier 3 manual) | `windows-debloat` `[win]` |
| Deploy de Next.js standalone / migração v15→v16 | `nextjs-standalone-deploy` |
| Ligar/desligar funcionalidade sem re-deploy | `simple-feature-flag` |
| JWT HS256 em Node.js sem dependências | `jwt-hs256-node` |
| Padronizar telefone BR (E.164) | `phone-e164-normalization` |
| Regressão/validação contra produção sem mutar dados | `playwright-prod-regression` |
| Dashboard/SPA autenticado (login, extrair dados, executar ação via HTTP) | `web-dashboard-automation` |
| HTML/CSS/SCSS semântico, acessível e responsivo | `frontend-markup` |
| Automação de browser interativa (abrir, clicar, 2FA manual, inspeção) | MCP `chrome-devtools` — habilite via `/mcp` |
| Automação de browser determinística (regressão, scripts repetíveis) | `pw` via shell (wrapper de `playwright-cli`; skills `web-dashboard-automation`, `playwright-prod-regression`) |
| Validar servidor de linguagem (LSP) que não responde | `lsp-smoke-test` |
| Texto, README, doc ou resposta com cara de IA | `stop-slop` |
| Encerrar sessão longa / transferir contexto para outra sessão ou agente | `handoff` |
| Início de qualquer tarefa (ler lições) e fim (gravar lições) | `agent-memory` |
| Vai gravar entrada em lessons/patterns | `memory-promotion` |
| Descobrir/criar skill a partir de uso repetido | `skill-miner` |
| Ajustar skill recém-baixada ao ambiente do usuário | `skill-personalizer` |
| Publicar/compartilhar skill (remover dados privados) | `skill-generalizer` |
| Performance Linux por OS (Ubuntu Server 24.04+, CachyOS, zram, sysctl, swap) | `linux-performance-tuning` `[linux]` |
| Jogos/emulação no CachyOS (kernel, GPU, Steam, emuladores) | `cachyos-gaming-setup` `[linux]` |
| Instalar pacote AUR sem TTY/senha | `aur-headless-install` `[linux]` |
| Validar app GUI (Qt/SDL) sem display | `headless-gui-probe` `[linux]` |
| Prevenir/recover comando ou tarefa presa (timeout, background, log, interrupção por id) | `task-hang-watchdog` |
| Coordenador vigiar subagente e interromper pelo id do runtime (CommandCode: `agent_output` · OpenCode: `sessionID`) | `subagent-supervision` |

**Delegação:** `subagent-routing` cobre roteamento, despacho paralelo e orquestração no mesmo repo.
