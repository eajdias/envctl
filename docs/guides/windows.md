# Guia de Execução & Provisionamento: Windows 11 PRO

Este guia detalha todos os métodos de execução do `envctl` no **Windows 11 PRO (22H2 / 23H2 / 24H2 / 25H2)** ou **Windows Server 2022+**.

> Instalação (1-liner, binário ou fonte): ver o guia único
> [`provisioning.md`](provisioning.md). Abaixo, só o que é específico do Windows.

---

## 🎛️ 4. Subcomandos e Operações Modulares no Windows

Você pode executar etapas específicas conforme sua necessidade:

```bash
# Apenas pacotes do sistema via Winget (VSCode, Windows Terminal, Ripgrep, etc.)
envctl run winget

# Apenas runtime Node.js LTS e ferramentas globais via Volta
envctl run volta

# Perfil completo da workstation (tweaks + debloat + pacotes + shell + skills + LSPs)
envctl run windows

# Apenas ajustes de Registro, Modo Desenvolvedor e Modo Escuro
envctl run tweaks

# Apenas variáveis de ambiente (NODE_PATH, ENVCTL_TEMP) e arquivos de shell
envctl run shell

# Apenas catálogo de 12 skills do OpenCode/CommandCode
envctl run skills

# Apenas binários de linguagem p/ shell/IDE (15 LSPs; sem efeito no runtime opencode v2)
envctl run lsp

# Limpeza de acúmulo do OpenCode (cache duplicado, tool-output, scratch >24h)
envctl run cleanup

# Snapshot reverso (salva o estado atual da máquina de volta nos manifestos)
envctl snapshot
```

---

## ⚙️ 5. Particularidades e Ajustes Críticos no Windows

### A. Stack de Shell: PowerShell 7 (primário) + WSL Ubuntu (secundário)
O ambiente Windows é padronizado em **PowerShell 7** como shell default do OpenCode e do Windows Terminal.
- **PowerShell 7**: shell primário (OpenCode, Windows Terminal default, scripts de automação).
- **WSL Ubuntu 26.04**: subshell POSIX secundário para ferramentas Linux — use `wsl -e bash -lc "..."` quando um comando exigir ambiente Linux.
- Comandos Docker rodam nativamente do PowerShell, sem conversão de caminhos ou wrappers.

### B. Resolução Global de Módulos Node (`NODE_PATH`)
Para garantir que scripts autônomos (como Playwright) funcionem a partir de qualquer pasta de projeto:
- `NODE_PATH` aponta para `%USERPROFILE%\node_modules`.

### C. Pasta de Scratch Padrão dos Agentes LLM (`ENVCTL_TEMP`)
Todo arquivo temporário criado por agentes LLM (downloads, builds, extrações, screenshots) deve ir para `C:\temp` — pasta na raiz do disco, sem relação com o OpenCode, facilitando identificação e exclusão. `envctl run cleanup` remove scratch com mais de 24h.

### D. Automação de Browser (MCP chrome-devtools + playwright-cli)
- O `envctl` provisiona o MCP `chrome-devtools` (`chrome-devtools-mcp`, `enabled: false` — ative por sessão via `/mcp`) para o interativo (o agente escolhe quando usar; 2FA manual e inspeção ao vivo) e o `playwright-cli` (via volta) para automação determinística no shell. Cada um usa seu próprio build de browser — sem conflito com o navegador do usuário.
