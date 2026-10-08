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
   - Debloat opt-in (`debloat_windows.yaml`): 6 linhas agregadas por categoria (`Debloat / category <nome>`),
     `OK` quando aplicada, `INFO` quando há drift com `run 'envctl run debloat'` — nunca `WARN`/`ERROR`,
     nunca no `--fix` (o stack só aplica sob invocação explícita).
2. **Gerenciadores de Pacotes & Toolchains**:
   - Winget, APT, Pacman, Paru, mise, npm, Go, Python UV/Pip.
   - Presença dos binários do manifesto no `PATH` (ver `manifests/packages.yaml` — matrix §1) (`rg`, `fd`, `fzf`, `bat`, `delta`, `tree`, `yq`, `jq`, `rsync`, etc.).
   - Entradas `type: mise` são auditadas pelo registro `mise ls` (`installed` **e** `active`): binário no PATH sem registro é cópia legada (o sweep arquiva após o install), e registro `installed` sem `active` é shim órfão (`mise install` sem `mise use -g` — ver matriz §3, assimetria #28). Os dois viram `WARN` com `run 'envctl run mise'`.
3. **Variáveis de Ambiente & Shell**:
   - `NODE_PATH` resolvido e validado contra módulos globais.
   - `ENVCTL_TEMP` apontando para a pasta de scratch padrão (`C:\temp` no Windows, `/temp` no Linux).
   - `Envctl / PATH`: o próprio `envctl` resolvível no `PATH` (bootstrap persiste o install dir; senão `WARN`).
   - `PATH (mise shims)`: o dir de shims do mise (`%LOCALAPPDATA%\mise\shims` no Windows, `~/.local/share/mise/shims` no POSIX) no `PATH` do usuário — sem ele nenhum binário `type: mise` resolve por nome bare (`run 'envctl run shell'`).
   - `Envctl / Binary freshness`: quando o binário é executado a partir de um checkout do repo,
     compara a versão embutida com `git describe --tags --always` — binário mais antigo que o
     checkout (ou build `dev`) vira `WARN` pedindo rebuild, porque os templates são `//go:embed`.
   - `TempFolder`: pasta acima de 500 MB vira `WARN` quando o dono dominante é scratch do agente
     (`opencode`, `commandcode`, `node-compile-cache`, `tsx-*`), e `INFO` quando o dominante é
     cache de terceiros (Docker Desktop, WinGet, Brave updaters) — regenerável pelo app dono.
   - Integridade de `settings.json` do Terminal, perfis do PowerShell e `opencode.json`.
   - `ConfigFile` com `merge: markdown_sections` (ex.: `AGENTS.md` do CommandCode) é reportado
     como "merged with user content" — o bloco `envctl:user` é preservado, nunca drift.
4. **Language Servers (ver `manifests/lsp.yaml`; `powershell` é windows-only)**:
   - Presença do binário no `PATH` + handshake stdio de stdin fechado para cada servidor — check de **toolchain** (shell/IDE), não de runtime do agente: o bloco `lsp` foi removido do `opencode.json` (runtime v2 ignora LSP; diagnósticos do agente via lint/typecheck).
5. **Runtime do usuário (npm libs)**: dependências de automação (`axios`, `cheerio`, `papaparse`) instaladas em `~/node_modules` via `npm install` quando `~/package.json` é mais novo.
6. **Catálogo de Skills (12 portáteis, + espelho CommandCode)**:
   - Existência e conformidade das Skills em `~/.config/opencode/skills/`.
   - **Quarentena com janela de recuperação**: diretório de skill que saiu do manifesto é
     **movido** (nunca apagado) para `~/.config/opencode/.envctl-trash/skills/<nome>-<stamp>`,
     irmão da árvore de skills — então não é re-varrido, re-podado nem contado pelo doctor.
     Entradas com mais de **30 dias** são removidas no deploy seguinte, para que o caminho de
     recuperação não vire armazenamento permanente. Sem esse limite a árvore crescia 1 diretório
     por skill removida, para sempre (39 diretórios por runtime na máquina em que o catálogo foi
     de 50 para 12). Arquivos soltos no trash não são tocados: a árvore não é exclusivamente nossa.
7. **Agentes & Config do OpenCode**:
   - `Config shape` (read-only): valida o formato V2 nativo do `~/.config/opencode/opencode.json` —
     sem `agent`/`permission` de V1, sem ações de permissão `bash`/`task`, `mode` em
     `primary|subagent|all` e `description` obrigatória em agente dispatchable. `OK` no shape
     nativo, `WARN` nomeando cada problema (config inválida passava pelo doctor verde).
   - `Agents` do CommandCode: valida cada `~/.commandcode/agents/*.md` em duas camadas —
     carga (`name` == nome do arquivo, nomes reservados ignorados) e **schema documentado**
     (docs/agents): `tools`/`disallowedTools` (aceita `"a, b"`, lista YAML ou `"*"`; id
     fora do catálogo = `INFO`, `agent`/`agent_output` = `WARN` porque nunca podem ser
     concedidos), `permissionMode` no conjunto válido, `maxTurns` inteiro positivo,
     `background`/`showOutput` booleanos, `model`/`reasoningEffort` não vazios. `WARN` nomeia
     o campo, porque o runtime **ignora** valor inválido em silêncio e o agente carrega com
     menos capacidades do que o frontmatter pede; chave desconhecida não gera diagnóstico
     (o runtime também a ignora).
   - `git worktree`: parse de `git worktree list --porcelain` — entrada `prunable` vira `WARN`
     com hint de `git worktree prune` **após revisão manual**; entrada `locked` vira `INFO`
     (trabalho intencional). Vale para worktrees do OpenCode **e** do CommandCode
     (`~/.commandcode/worktrees/`), porque ambas são `git worktree` do mesmo repo. O
     `doctor` nunca poda, destrava ou remove worktree.
8. **Performance Linux (read-only)**:
   - `Performance` agrega swap, zram, governor, scheduler, journald, `fstrim.timer` e serviços.
   - Estado opcional ausente é `INFO`, nunca warning/error; `run performance` e `doctor --fix` não aplicam governors, schedulers ou journald. O único lifecycle automático é o serviço gerador do zram quando o device está ausente.
9. **Verificação Local (`Verify`)**:
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
- Baixa runtimes ou componentes de LSP faltantes (ex: `pyright`, `docker-langserver` ou binários do Chromium).

---

## 🔄 Idempotência Estrita & Backup Atômico

O `doctor` só **relata**: não muta a máquina sem `--fix`. O `envctl update` é o oposto — ele
muda, e é por isso que tem escopo próprio: só mecanismos user-local (`mise` runtimes + ferramentas, `uv tool`), nunca gerenciador de SO, porque *partial upgrade* no Arch quebra o sistema.
Para saber o que está atrás sem mudar nada: `envctl update --list` (nem toca a rede) ou
`envctl update --dry-run`.


Todas as operações de escrita de arquivos e alterações no sistema são **estritamente idempotentes**:

### 1. Detecção de Hash SHA-256
Antes de tocar em qualquer arquivo no disco:
1. O `envctl` calcula o hash SHA-256 do arquivo em disco e do template embutido.
2. Se os hashes forem idênticos, a operação é pulada (`[IDEMPOTENT-SKIP]`), evitando tocar na data de modificação (`mtime`) ou gerar I/O desnecessário.
3. Se houver divergência real de conteúdo, o arquivo original é renomeado para `<nome>.bak.YYYYMMDD-HHMMSS` antes de gravar o novo conteúdo.

### 2. Poda de Backups (`keep_newest`)
O backup atômico é ilimitado por padrão, então cada execução que diverge deixa um arquivo. Itens de manifest com `keep_newest: N` podam esse histórico:

- A poda é **recursiva**: as árvores de skills são aninhadas (`~/.config/opencode/skills/<skill>/SKILL.md`), e uma varredura só do topo nunca as alcançava — o resultado era 1 backup por redeploy acumulando indefinidamente.
- O agrupamento é por **caminho do arquivo original**, não por nome base: dois `SKILL.md` em diretórios diferentes não disputam o mesmo slot de `keep_newest`.
- Aplicada em `~/.config/opencode` e `~/.commandcode` com `keep_newest: 1` (um backup por arquivo, o suficiente para rollback de edição manual).
- Arquivos com conteúdo **idêntico** não geram backup nenhum (diff-gate por hash), então redeploy sem mudança não deixa rastro.

### 3. Idempotência em Gerenciadores de Pacotes
- **Winget**: Consulta o catálogo local (`winget list --exact --id <name>`) antes de invocar o instalador.
- **APT**: Utiliza `dpkg-query -W` para verificar se o pacote já está instalado.
- **Pacman**: Utiliza o parâmetro `-S --needed` para não reinstalar pacotes atualizados.
- **mise / npm / Go**: Inspecionam o `PATH` e a versão do binário antes de disparar instalações remotas.

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
