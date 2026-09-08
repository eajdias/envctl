---
name: headless-gui-probe
description: Gerar configs e validar apps GUI (Qt/SDL) sem display, para inspeção de arquivos de config e logs. Use ao configurar emuladores, extrair defaults de .ini/.toml ou testar se um binário roda na CPU atual. Triggers: headless, offscreen, gerar config, validar GUI, SIGILL, AVX2.
---

# Headless GUI Probe

## Quando usar
Precisa dos arquivos de config que um app GUI só cria ao abrir, ou quer saber se o binário executa (ex.: crash SIGILL em CPU sem AVX2).

## Passos
1. Tentar plataforma offscreen/dummy primeiro (sem janela, sem risco):
   ```bash
   QT_QPA_PLATFORM=offscreen timeout -k 3 10 <app> > /tmp/probe.log 2>&1
   SDL_VIDEODRIVER=dummy timeout -k 3 10 <app> > /tmp/probe.log 2>&1
   ```
2. SEMPRE redirecionar para arquivo (nunca pipe): filhos que seguram stdout travam o `timeout` e a sessão.
3. SEMPRE `timeout -k <kill> <secs>`: apps GUI ignoram SIGTERM; sem `-k` o processo sobrevive.
4. Se offscreen falhar (plugin ausente/crash), UMA passada curta no display real (`DISPLAY=:0`, 6-10s) e matar por PID específico (nunca `pkill -f` genérico que pode pegar o próprio shell).
5. Ler configs geradas (`~/.config/<app>/`) e logs (`--log`, `Logs/`, stdout do arquivo).
6. Crash `Instrução ilegal` / exit 132 = binário exige instruções ausentes na CPU (ex.: AVX2) → descartar o build, não a config.

## Verificação
- Arquivos de config existem com timestamps novos; nenhum processo residual (`pgrep -f <app>` vazio, descontando o próprio grep).
- `SIGILL` documentado como incompatibilidade, não como bug de config.
