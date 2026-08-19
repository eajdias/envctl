# Doctor Diagnóstico, Idempotência & Trilha de Auditoria

O `envctl` oferece garantias estritas de estabilidade operacional, idempotência matemática e rastreabilidade total de todas as ações executadas no sistema hospedeiro.

---

## 🩺 O Subsistema `doctor` (160+ Verificações)

O comando `envctl doctor` realiza uma varredura completa em todos os subsistemas da máquina para verificar a conformidade com o estado desejado definido nos manifestos:

```bash
# Executa a auditoria completa
envctl doctor
```

### Categorias de Diagnóstico Inspecionadas:
1. **Ambiente & Sistema Operacional**:
   - Detecção de SO, arquitetura, privilégios de execução.
   - Ajustes de Registro do Windows (Win32 Long Paths, Developer Mode, Dark Mode, Explorer extensions).
   - Fonte tipográfica (MesloLGM Nerd Font).
2. **Gerenciadores de Pacotes & Toolchains**:
   - Winget, APT, MSYS2 Pacman, Volta, Go, Rustup, Dotnet CLI, Python UV/Pip.
   - Presença de todos os 50+ binários essenciais no `PATH` (`rg`, `fd`, `fzf`, `bat`, `delta`, `tree`, `yq`, `jq`, `rsync`, etc.).
3. **Variáveis de Ambiente & Shell**:
   - `NODE_PATH` resolvido e validado contra módulos globais.
   - `MSYS2_ENV_CONV_EXCL` e `MSYS2_ARG_CONV_EXCL` configurados para evitar corrupção de caminhos Docker e Node.
   - Integridade de `.bashrc`, `.bash_profile`, `settings.json` do Terminal e perfis do PowerShell.
4. **Language Servers (16 LSPs)**:
   - Verificação de binários e capacidade de resposta via `--version` ou `--stdio` para cada um dos 16 servidores de linguagem registrados.
5. **Runtime do Playwright & Navegadores Headless**:
   - Carregamento assíncrono do módulo `playwright` via Node.js em qualquer diretório.
   - Presença dos binários do Chromium instalados (`%LOCALAPPDATA%/ms-playwright` no Windows ou `~/.cache/ms-playwright` no Linux).
   - Scripts utilitários `pw-screenshot` e `pw-eval` operacionais.
6. **Catálogo de 59 Skills de Agentes**:
   - Existência e conformidade de todas as 59 skills em `~/.config/opencode/skills/` e `~/.agents/skills/`.

---

## 🛠️ Auto-Remediação com `envctl doctor --fix`

Quando o `doctor` detecta qualquer divergência (`WARN` ou `ERROR`) em relação aos manifestos declarativos, o parâmetro `--fix` entra em ação:

```bash
# Executa a auditoria e corrige automaticamente qualquer inconsistência
envctl doctor --fix
```

### O que o `--fix` executa de forma autônoma:
- Aplica chaves de registro ausentes ou incorretas.
- Instala pacotes faltantes via gerenciador nativo correspondente.
- Reinstala variáveis de ambiente do usuário.
- Restaura templates de shell e configurações com backup atômico.
- Extrai e sincroniza skills ausentes ou desatualizadas.
- Baixa runtimes ou componentes de LSP faltantes (ex: `rust-analyzer` ou binários do Chromium).

---

## 🔄 Idempotência Estrita & Backup Atômico

Todas as operações de escrita de arquivos e alterações no sistema são **estritamente idempotentes**:

### 1. Detecção de Hash SHA-256
Antes de tocar em qualquer arquivo no disco:
1. O `envctl` calcula o hash SHA-256 do arquivo em disco e do template embutido.
2. Se os hashes forem idênticos, a operação é pulada (`[IDEMPOTENT-SKIP]`), evitando tocar na data de modificação (`mtime`) ou gerar I/O desnecessário.
3. Se houver divergência real de conteúdo, o arquivo original é renomeado para `<nome>.bak.YYYYMMDD-HHMMSS` antes de gravar o novo conteúdo.

### 2. Idempotência em Gerenciadores de Pacotes
- **Winget**: Consulta o catálogo local (`winget list --exact --id <name>`) antes de invocar o instalador.
- **APT**: Utiliza `dpkg-query -W` para verificar se o pacote já está instalado.
- **Pacman**: Utiliza o parâmetro `-S --needed` para não reinstalar pacotes atualizados.
- **Volta / Go / Dotnet / Rustup**: Inspecionam o `PATH` e a versão do binário antes de disparar instalações remotas.

---

## 📜 Trilha de Auditoria & Logging Persistente

Toda execução de qualquer comando do `envctl` gera automaticamente um log estruturado em:
```
~/.envctl/logs/envctl-YYYYMMDD-HHMMSS.log
```

### Formato do Log:
```
2026-08-18 20:30:15.120 [INFO]  === envctl Session Started (Command: run all, OS: windows, Arch: amd64) ===
2026-08-18 20:30:15.125 [DEBUG] [IDEMPOTENT-SKIP] Package 'BurntSushi.ripgrep.MSVC' already installed and verified at 'C:\Program Files\ripgrep\rg.exe'
2026-08-18 20:30:15.140 [INFO]  [APPLY-CHANGE] Writing file '~/.bashrc' (Hash mismatch detected)
2026-08-18 20:30:15.142 [DEBUG] Created backup at '~/.bashrc.bak.20260818-203015'
2026-08-18 20:30:15.145 [DEBUG] Command executed: 'pacman -S --needed --noconfirm tree' -> Exit Code: 0
2026-08-18 20:30:16.890 [INFO]  === envctl Session Finished Successfully (Elapsed: 1.765s, Errors: 0) ===
```

- Sanitiza automaticamente null bytes e caracteres de controle oriundos de consoles UTF-16 no Windows.
- Captura comandos executados, códigos de saída e payloads completos em caso de falha para diagnóstico imediato.
