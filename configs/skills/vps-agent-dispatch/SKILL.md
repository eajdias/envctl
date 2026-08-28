---
name: vps-agent-dispatch
description: Orquestrar e despachar tarefas e subagentes do OpenCode em servidores remotos (VPS Ubuntu/Debian/Oracle/AWS) via SSH. Use quando o usuário pedir para rodar tarefas pesadas, testes, builds, benchmarks ou subagentes remotos em VPS para economizar contexto local e paralelizar execuções ("despachar no vps", "executar agente remoto", "rodar subagente no servidor", "vps agent", "delegar para o vps", "dispatch agent", "executar remoto no opencode").
---

# VPS Agent Dispatch (Remote OpenCode Orchestration)

## Propósito e Benefícios
Permite que o agente local (no notebook) delegue tarefas autônomas, testes pesados, compilações, crawlers ou tarefas multi-etapas para instâncias remotas do OpenCode rodando em VPS Ubuntu/Debian (AWS, Oracle Cloud, etc.) via SSH.

### Vantagens:
1. **Economia Extrema de Contexto**: A exploração pesada e centenas de tool calls rodam no VPS remoto; o agente local recebe apenas o resumo técnico cristalizado.
2. **Poder Computacional e Rede**: VPS em data centers têm links de altíssima velocidade (1Gbps+) e recursos dedicados.
3. **Isolamento de Ambiente**: Mudanças de pacotes, compilações de containers e testes destrutivos rodam isolados no VPS sem poluir o notebook local.

---

## Modos de Execução

### 1. Despacho Síncrono Direto (Headless OpenCode Run)
Para tarefas curtas a médias (1 a 5 minutos):

```bash
# Via SSH direto com chave
ssh -i "$KEYPATH" $USER@$HOST "opencode run '$PROMPT'"

# Ou via ssh-manager CLI
ssh-manager exec <servidor> "opencode run '$PROMPT'"
```

### 2. Despacho com Sincronização de Workspace (Rsync + Execution + Return)
Para rodar testes ou refatoração em código local transferido para o VPS:

```bash
# 1. Sincronizar workspace local para pasta temporária no VPS
rsync -avz --exclude '.git' --exclude 'node_modules' --exclude 'target' -e "ssh -i $KEYPATH" ./ $USER@$HOST:/tmp/workspace-$TASK_ID/

# 2. Executar OpenCode no diretório do workspace remoto
ssh -i "$KEYPATH" $USER@$HOST "cd /tmp/workspace-$TASK_ID && opencode run '$PROMPT'"

# 3. Trazer de volta resultados/patches (se necessário)
rsync -avz -e "ssh -i $KEYPATH" $USER@$HOST:/tmp/workspace-$TASK_ID/results/ ./results/

# 4. Limpar workspace remoto
ssh -i "$KEYPATH" $USER@$HOST "rm -rf /tmp/workspace-$TASK_ID"
```

### 3. Despacho Assíncrono para Tarefas Longas (Nohup / Background)
Para tarefas de longa duração (benchmarks, crawls, grandes compilações):

```bash
# Iniciar execução desacoplada
ssh -i "$KEYPATH" $USER@$HOST "nohup opencode run '$PROMPT' > /tmp/agent-$TASK_ID.log 2>&1 & echo \$!"

# Checar progresso / logs
ssh -i "$KEYPATH" $USER@$HOST "tail -n 30 /tmp/agent-$TASK_ID.log"

# Aguardar conclusão e obter resultado final
ssh -i "$KEYPATH" $USER@$HOST "wait <PID> 2>/dev/null; cat /tmp/agent-$TASK_ID.log"
```

---

## Protocolo de Orquestração do Subagente

Quando você (agente local) for instruído a despachar uma tarefa para um VPS:

1. **Identificar Alvo**:
   - Consulte `~/.ssh-manager/.env` ou `~/.ssh/config` para identificar os VPS disponíveis e credenciais.
   - Verifique a conectividade básica: `ssh-manager server test <servidor>` ou `ssh -o BatchMode=yes <servidor> uptime`.

2. **Verificar Pré-requisitos no VPS**:
   - Verifique se o OpenCode está instalado: `ssh <servidor> "which opencode || command -v opencode"`.
   - Se ausente, sugira provisionar via `envctl run all` ou `npm install -g @opencode-ai/cli` / `curl -fsSL ...`.

3. **Montar o Prompt do Subagente Remoto**:
   - O prompt enviado deve ser **autocontido, explícito e com critérios claros de sucesso**.
   - Exija formato de saída estruturado para facilitar a leitura no retorno (ex: "Emita um resumo técnico final em Markdown com seções: Ações Realizadas, Métricas, Erros e Status").

4. **Executar e Capturar Saída**:
   - Execute o comando remoto.
   - Monitore erros de conexão ou timeout.

5. **Cristalizar e Sintetizar**:
   - Processe a resposta recebida do VPS.
   - Apresente ao usuário local uma síntese limpa e objetiva, mantendo a janela de contexto local compacta e de alta qualidade.

---

## Limites Free & Fallback de Token

O OpenCode na VPS roda no plano **Free** (limites de uso/modelo). Quando o limite estoura, o `opencode run` remoto falha com erro de quota/rate-limit (ex.: "usage limit", "quota exceeded", "rate limit").

**Protocolo ao detectar limite Free estourado na VPS:**

1. **PARE e PERGUNTE ao usuário** se ele quer registrar um TOKEN/API key no OpenCode da VPS (`opencode auth login` na VPS, ou `OPENCODE_API_KEY` no `.bashrc`). O agente NUNCA manuseia/armazena o token — o registro é do usuário.
2. **Se o usuário não quiser registrar token:** execute a tarefa **por conta própria**, direto via SSH (`ssh-manager exec <vps> "..."` ou `ssh -i <chave> user@host`), sem depender do OpenCode remoto — mesmo fluxo: read-only → diagnóstico → execução → verificação → reporte.
3. **Se o usuário registrar:** prossiga com `opencode run` normal na VPS.
