# Spec — `envctl service`: convergedência como serviço de background (Linux, audit-only)

**Data:** 2026-09-27
**Roadmap:** item 5 (`docs/roadmap.md`)
**Escopo:** esta é uma **spec, sem implementação**. O recorte é o que foi julgado implementável
e verificável; o resto fica adiado, com o motivo.
**Decisão:** Linux-only, `systemd --user`, e o serviço **só audita** — não auto-corrige, não
auto-atualiza.

---

## 1. Objetivo

Hoje a convergência do ambiente depende de alguém abrir uma sessão e rodar `envctl run`. Numa
máquina que é serviço, isso significa que a deriva acumula até alguém notar. O objetivo é um
timer `systemd --user` que rode o `doctor` periodicamente e deixe o resultado em log, para que a
deriva seja **visível sem sessão**.

## 2. O que já existe (medido, não assumido)

| peça | estado | evidência |
|---|---|---|
| verbo que o timer chamaria | existe | `doctor` é comando de primeiro nível, 156 checks na VPS e 219 no CachyOS |
| manutenção periódica | existe | `run cleanup`, `update`, `snapshot` |
| onde logar | existe | `~/.envctl/logs/` (o doctor já grava lá; `--log-dir` é flag global) |
| `systemd --user` funcional | **verificado nesta máquina** | `systemd-run --user` executou com runtime de 13ms; `Linger=yes`, então o timer roda sem sessão aberta |
| unit/timer embarcado | não existe | nenhum template `.service`/`.timer` no repo |
| comando de instalar/remover | não existe | os 9 verbos são `run`, `doctor`, `snapshot`, `update`, `version`, `opencode`, `commandcode`, `completion`, `help` |
| **single-flight / lockfile** | **não existe** | grep por `lockfile`/`flock`/`LockFile` no código: **zero** ocorrências |

## 3. Por que o recorte é audit-only

O roadmap lista três cuidados: single-flight com lockfile, logging, e **"nunca auto-`--fix`
destrutivo silencioso — o serviço audita e reporta; corrige só o que estiver numa whitelist"**.

O segundo cuidado é o que decide o recorte: um serviço que só **relata** não pode fazer dano,
mesmo correndo junto de um `envctl run` manual. Um `doctor` (read-only) e um `run` (muta)
disputando a máquina é inofensivo — o doctor só pode relatar o que ele vê no instante.

E é exatamente por isso que **o lockfile não é necessário agora**: single-flight existe para
impedir que dois processos que **mutam** interfiram. Sem mutação, não há o que serializar. O
lockfile fica PRD (abaixo) na fase em que o serviço passar a corrigir, que é onde ele passa a
ser obrigatório — e é também onde ele é caro, porque precisa entrar nos entrypoints de `run`,
`doctor` e `update`, com o risco real de um **lock órfão** travar a ferramenta.

## 4. Recorte proposto

```
envctl service install     # escreve ~/.config/systemd/user/envctl-audit.{service,timer}
                           # habilita e inicia o timer
envctl service uninstall   # para, desabilita e remove os dois arquivos
envctl service status      # se o timer está ativo, quando rodou por último, e o resumo
```

**Unit** (`envctl-audit.service`), `Type=oneshot`:
```ini
ExecStart=<envctl-abs-path> doctor --log-dir %h/.envctl/logs
```
O caminho do binário é **absoluto e resolvido em tempo de install** (`exec.LookPath` + realpath),
porque um `ExecStart` relativo a um serviço --user não resolve por PATH de login.

**Timer** (`envctl-audit.timer`): `OnBootSec=5min` e `OnUnitActiveSec=6h`, `Persistent=true`
(assim uma máquina desligada audita ao subir). `RandomizedDelaySec=15min` para não colidir com
o clock de todo mundo.

**Defaults:** 6h é conservador para `doctor` (156–219 checks, todos read-only). Frequência
ajustável por flag, não por arquivo editado à mão.

## 5. Fora deste recorte, e por quê

| adiado | motivo |
|---|---|
| **auto-`--fix` com whitelist** | exige o lockfile nos entrypoints de `run`/`doctor`/`update`; risco de lock órfão. É a fase 2, e o roadmap já diz que a correção vai numa whitelist — que ainda não existe |
| **auto-update do envctl** | o próprio binário se substitui em uso; e a fase 0 (`run providers`) já é a dona dos CLIs de agente. Automatizar isso exige verificar assinatura do artefato, que o repo ainda não faz |
| **Windows (Task Scheduler)** | mecanismo distinto; o CI tem `windows-latest` mas registrar uma task agendada em CI é invasivo. Não verificável nesta máquina |
| **Termux (`termux-services`/`termux-boot`)** | depende do item 1 do roadmap, que está **indeciso** entre `os: termux` e família detectada por `$PREFIX`/`TERMUX_VERSION` |

## 6. Riscos e guards

| risco | guard |
|---|---|
| timer disparar durante um `run` manual | irrelevante enquanto o job é read-only; quando deixar de ser, o lockfile é obrigatório (fase 2) |
| binário movido/depois apagado → unit quebrada | `service status` mostra `ExecStart` e detecta caminho ausente; `uninstall` funciona mesmo com unit quebrada |
| log crescendo sem parar | `doctor` já nomeia o log por timestamp; o serviço **não** roda `run cleanup` (são coisas diferentes: cleanup é cache de agente, log é histórico) |
| service instalado eesquecido | `doctor` ganha um check que reporta se o timer existe e está ativo, e em cima disso uma **INFO** com a data da última execução — a mesma classe de check que o projeto já usa para `pacnew` |
| dois installs | `install` é idempotente: reescreve os arquivos e re-habilita; não há estado escondido |

## 7. Verificação

O que tem de existir quando isso for implementado:

1. `service install` em HOME temporário cria os dois arquivos com `ExecStart` absoluto
2. `install` duas vezes ⇒ **sem** duplicar linhas nos arquivos (o problema que o
   `configStep` do bootstrap teve, e que o `TestGoPathConfigStepReportsWorkOnlyOnce` agora guarda)
3. `status` com timer ausente ⇒ responde "not installed" sem erro
4. o `ExecStart` gerado roda de verdade: `systemd-analyze verify` no `.service`
5. `doctor` reporta o timer quando instalado e fica quieto quando não está
6. o binário que o `install`.resolve é o que o `status` mostra (sem divergência)
7. `uninstall` remove os dois arquivos e desabilita o timer, com e sem unit quebrada

Os testes 1, 2, 3 e 7 rodam em CI sem systemd. O 4 e o 5 exigem `systemd --user` e são
**pulados** fora de Linux com sessão — mesma convenção de `TestOpenCodePathInstallerPersistsPOSIXProfiles`,
que já pula em Windows.

## 8. Ordem sugerida

1. `status` (read-only, sem instalar nada) — expõe o estado e já é útil
2. `install`/`uninstall` + os templates
3. o check do `doctor`
4. a fase 2: lockfile, depois whitelist, depois Windows
