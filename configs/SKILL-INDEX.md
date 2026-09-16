# OpenCode — índice de skills

**Consulte sob demanda.** Não leia este arquivo por padrão: abra-o apenas quando precisar decidir *qual* skill carregar para a tarefa em mãos. O catálogo (nome + descrição) já vem no seu prompt automaticamente.

- **Skills instaladas:** `~/.config/opencode/skills/<nome>/SKILL.md` — invoque com `/<skill>`; a tool do modelo é `skill` (com `name: <skill>`).
- **Marcadores:** `[win]` só existe em máquinas Windows · `[linux]` só existe em Linux · sem marcador = qualquer ambiente.
- **Regra de ouro:** se a tarefa é "como fazer X neste ambiente", veja se existe uma skill para X antes de improvisar.

| Situação | Skill |
|---|---|
| Escrever, alterar ou configurar código com lib/framework/API | `context7-auto` |
| Bug, teste vermelho, erro intermitente, comportamento inesperado | `systematic-debugging` |
| Antes de declarar "pronto"/corrigido/passando (inclusive antes de commitar ou abrir PR) | `verification-before-completion` |
| Implementar algo multi-passo, feature nova ou refatoração ampla | `writing-plans` |
| Pedido ambíguo, incompleto ou com premissas não ditas | `ask-questions-if-underspecified` |
| Endurecer, questionar ou stress-testar um plano/decisão | `grilling` |
| Receber code review ou comentários de PR | `receiving-code-review` |
| Commit, branch, PR, rebase, conflito, tag/release | `git-workflow` |
| Isolar o trabalho num workspace próprio (feature paralela) | `using-git-worktrees` |
| Decidir delegar / explorar codebase / pesquisar na web | `subagent-routing` |
| Executar 2+ tarefas independentes em paralelo | `dispatching-parallel-agents` |
| Vários subagentes no mesmo repositório git | `parallel-agent-orchestration` |
| Schema, migration, query, backup/restore de banco | `database-ops` |
| Inserir/atualizar muitos registros no PostgreSQL | `bulk-postgres-import` |
| Rodar testes, medir cobertura, rodar benchmark | `universal-test-runner` |
| Desenhar/validar contrato de API (OpenAPI, GraphQL, gRPC) | `api-contract-design` |
| Servidor/VPS remoto: monitorar, diagnosticar, reiniciar serviço | `ssh-vps` |
| Provisionar, atualizar ou auditar VPS/VM com envctl | `vps-provisioning` |
| Rodar tarefa pesada (build, suíte, crawler) numa VPS | `vps-agent-dispatch` |
| Containers, compose, imagens, volumes, Docker Hub | `docker` |
| Windows: serviços, registro, tarefas agendadas, firewall, winget | `windows-admin` `[win]` |
| Docker Desktop não sobe / erro de backend WSL2 | `docker-desktop-wsl-restart` `[win]` |
| Deploy de imagem Docker em VPS fraca (build local) | `docker-build-local-vps-deploy` |
| Deploy de Next.js standalone / migração v15→v16 | `nextjs-standalone-deploy` |
| Ligar/desligar funcionalidade sem re-deploy | `simple-feature-flag` |
| JWT HS256 em Node.js sem dependências | `jwt-hs256-node` |
| Padronizar telefone BR (E.164) | `phone-e164-normalization` |
| Regressão/validação contra produção sem mutar dados | `playwright-prod-regression` |
| Dashboard/SPA autenticado (login, extrair dados, executar ação via HTTP) | `web-dashboard-automation` |
| Automação de browser (abrir, clicar, extrair, screenshot) | MCPs `playwright` / `chrome-devtools` — habilite via `/mcp` |
| Validar servidor de linguagem (LSP) que não responde | `lsp-smoke-test` |
| Texto, README, doc ou resposta com cara de IA | `stop-slop` |
| Encerrar sessão longa / transferir contexto para outra sessão ou agente | `handoff` |
| Início de qualquer tarefa (ler lições) e fim (gravar lições) | `agent-memory` |
| Vai gravar entrada em lessons/patterns | `memory-promotion` |
| Descobrir/criar skill a partir de uso repetido | `skill-miner` |
| Ajustar skill recém-baixada ao ambiente do usuário | `skill-personalizer` |
| Publicar/compartilhar skill (remover dados privados) | `skill-generalizer` |
| Editar `opencode.json` / `dcp.jsonc` / agentes | `customize-opencode` |
| Jogos/emulação no CachyOS (kernel, GPU, Steam, emuladores) | `cachyos-gaming-setup` `[linux]` |
| Instalar pacote AUR sem TTY/senha | `aur-headless-install` `[linux]` |
| Validar app GUI (Qt/SDL) sem display | `headless-gui-probe` `[linux]` |

**Delegação:** `subagent-routing` decide *quem* delegar; `dispatching-parallel-agents` e `parallel-agent-orchestration` dão a *mecânica*.
