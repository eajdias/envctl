# Doctor Diagnóstico, Idempotência & Trilha de Auditoria

O `envctl` oferece garantias estritas de estabilidade operacional, idempotência matemática e rastreabilidade total de todas as ações executadas no sistema hospedeiro.

---

## 🩺 O Subsistema `doctor`

O comando `envctl doctor` realiza uma varredura completa em todos os subsistemas da máquina para verificar a conformidade com o estado desejado definido nos manifestos:

```bash
# Executa a auditoria completa
envctl doctor
```

### Categorias de Diagnóstico Inspecionadas:
1. **Ambiente & Sistema Operacional**:
   - Detecção de SO, arquitetura, privilégios de execução.
   - Ajustes de Registro do Windows (Win32 Long Paths, Developer Mode, Dark Mode, Explorer extensions).
2. **Gerenciadores de Pacotes & Toolchains**:
   - Winget, APT, Pacman, Paru, Volta, Go, Python UV/Pip.
   - Presença de 45–55 binários conforme o OS (45 Ubuntu / 48 Arch / 55 Win — matrix §1) no `PATH` (`rg`, `fd`, `fzf`, `bat`, `delta`, `tree`, `yq`, `jq`, `rsync`, etc.).
3. **Variáveis de Ambiente & Shell**:
   - `NODE_PATH` resolvido e validado contra módulos globais.
   - `ENVCTL_TEMP` apontando para a pasta de scratch padrão (`C:\temp` no Windows, `/temp` no Linux).
   - Integridade de `settings.json` do Terminal, perfis do PowerShell e `opencode.json`.
4. **Language Servers (15 no manifesto, 14 aplicáveis no Linux — `pwsh` é windows-only)**:
   - Presença do binário no `PATH` + handshake stdio de stdin fechado para cada servidor — check de **toolchain** (shell/IDE), não de runtime do agente: o bloco `lsp` foi removido do `opencode.json` (runtime v2 ignora LSP; diagnósticos do agente via lint/typecheck).
5. **Runtime do usuário (npm libs)**: dependências de automação (`axios`, `cheerio`, `papaparse`) instaladas em `~/node_modules` via `npm install` quando `~/package.json` é mais novo.
6. **Catálogo de Skills por OS (38 Win / 38 Ubuntu / 40 CachyOS, + espelho CommandCode)**:
   - Existência e conformidade das Skills em `~/.config/opencode/skills/`.
7. **Verificação Local (`Verify`)**:
   - `~/.local/bin/envctl-verify` e `~/.config/git/hooks/pre-push` presentes e executáveis, e `core.hooksPath` apontando para o diretório de hooks (ver [verification.md](./verification.md)).

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
- Baixa runtimes ou componentes de LSP faltantes (ex: `pylsp`, `docker-langserver` ou binários do Chromium).

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
- **Volta / Go**: Inspecionam o `PATH` e a versão do binário antes de disparar instalações remotas.

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
2026-08-18 20:30:15.140 [INFO]  [APPLY-CHANGE] Writing file '~/.config/opencode/opencode.json' (Hash mismatch detected)
2026-08-18 20:30:15.142 [DEBUG] Created backup at '~/.config/opencode/opencode.json.bak.20260818-203015'
2026-08-18 20:30:15.145 [DEBUG] Command executed: 'winget install --exact --id BurntSushi.ripgrep.MSVC --silent' -> Exit Code: 0
2026-08-18 20:30:16.890 [INFO]  === envctl Session Finished Successfully (Elapsed: 1.765s, Errors: 0) ===
```

- Sanitiza automaticamente null bytes e caracteres de controle oriundos de consoles UTF-16 no Windows.
- Captura comandos executados, códigos de saída e payloads completos em caso de falha para diagnóstico imediato.
