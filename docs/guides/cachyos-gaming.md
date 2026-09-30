> Guia de provisionamento e auditing do stack de jogos do CachyOS.
> Última migração da skill `cachyos-gaming-setup`: 2026-09-26.

# Guia: CachyOS Gaming Stack

## Escopo

O `envctl` provisiona **tudo** do stack de gaming de forma idempotente num PC
formatado:

- **Pacotes** (`gaming.yaml`): Steam, emuladores, Proton, LACT, scx, quarteto X11.
- **Presets de usuário** (`gaming.conf`, `MangoHud.conf`, configs dos 9
  emuladores em `configs/emulators/`): seed só quando o arquivo não existe —
  ajustes manuais seus vencem.
- **Tuning privilegiado** (via sudo interativo, `sudo -v` no início do run):
  kernel cmdline (`/etc/default/limine` + `limine-update` + reboot), LACT fan
  curve (GPU AMD detectada via sysfs, curve conservadora embutida),
  `scx_loader` (bpfland/Auto) e o bypass de compositing no `kwinrc` (merge de
  seção, preserva suas outras preferências).
- **Eden AppImage**: download do build pinnado (legacy ou standard conforme a
  CPU), smoke test SIGILL, launcher `.desktop`, config seed.

O `doctor` audita tudo isso em modo read-only (nunca `--fix`): os checks
reportam o que o `run gaming` provisiona, em `INFO` quando ausente e em `WARN`
apenas quando a ausência quebra o stack (ex.: pacotes faltando).

**O que continua manual (decisão de segurança/legal, sem PII no repo):**
- BIOS/firmware/keys dos emuladores (dump do seu próprio console).
- ROMs em `~/Games/*` (estrutura de pastas é seedada; conteúdo é seu).
- Login Steam, launch options e perfis de controle Dolphin (GUI).

Este guia substituiu a skill global `cachyos-gaming-setup`, removida do catálogo de
agentes em 2026-09-26: é conhecimento do produto e pertence ao repo, não ao tier global.

## Hardware (host validado: i7-2600 + RX 580 2048SP)

O stack abaixo foi **medido e validado num host específico** (Sandy Bridge
x86-64-v2 + Polaris). As limitações deste hardware são o teto honesto do stack:

- **Sem AVX2 (x86-64-v2):** só repositórios genéricos (nunca v3/v4), AppImages
  legacy, e um binário novo pode morrer com SIGILL — testar antes de confiar.
  Emuladores exigentes são **inviáveis** e não devem ser instalados: RPCS3/PS3,
  ShadPS4/PS4, Switch AAA, xemu (Xbox) e simple64 (N64) — todos exigem AVX2.
- **Placa-mãe Sandy Bridge:** sem ReBAR, PCIe 2.0 → perda de ~5-10% na RX 580,
  normal, não é drift.
- **Limite de emulação realista:** até PS2/GC/Wii/PSP/3DS confortável; Switch
  só 2D/indie a 720p/30fps (AAA = 10-20fps slideshow, limite de silício).
- **RAM DDR3:** 3x8GB @1333 flex dual-channel; se os pentes forem 1600, ativar
  o perfil no BIOS (ganho pequeno).
- **BIOS (checklist manual):** XMP/DOCP, HPET off, C-states/EIST on, CSM/UEFI
  como está se boota.

## Sistema (verificar, não presumir)
1. CPU sem AVX2 (ex.: Sandy Bridge, x86-64-v2)? Então: só repos genéricos (nunca v3/v4), AppImages legacy, e testar SIGILL em qualquer binário novo. Emuladores exigentes (RPCS3, ShadPS4, Switch AAA, xemu, simple64) são inviáveis — não instalar.
2. Kernel cmdline (Limine `/etc/default/limine` + `limine-update` + reboot). Três tiers, porque os params não têm o mesmo significado:
   - **Universal (auditado com WARN se faltar):** `preempt=full split_lock_detect=off zswap.enabled=0`.
   - **Só com AMD (`amdgpu` carregado; auditado só nessa máquina, e ignorado em Intel/NVIDIA):** `amdgpu.runpm=0 amdgpu.aspm=0 pcie_aspm=off amdgpu.gpu_recovery=0`.
   - **Estabilidade, opcional (INFO, nunca WARN):** `oops=panic panic=10` — transformar oops em panic e limitar o loop de reboot; é escolha de crash visibility, não de performance. Kernel de fábrica sem eles não é gap.
   - `mitigations=off` é decisão manual aprovada caso a caso (trade-off Spectre/Meltdown) — o envctl NÃO o gerencia nem audita, de propósito.
3. `power-profiles-daemon` balanced no desktop; jogos via wrapper `game-performance %command%` — o binário vem do pacote `cachyos-settings`, declarado no `gaming.yaml` (é ele quem traz o ruleset `cachyos-ananicy-rules` e os defaults em `modprobe.d`/`sysctl.d`/`modules-load.d`). Manter `ananicy-cpp` **junto com** `cachyos-ananicy-rules`: o daemon sozinho sobe com zero regras. NUNCA combinar com `gamemode`.
4. Scheduler: `/etc/scx_loader/config.toml` com `default_sched="scx_bpfland"` + `default_mode="Auto"`; `game-performance` troca p/ perfil gaming sozinho. Rollback: `systemctl disable --now scx_loader`.
5. GPU AMD: `lactd` ativo + fan curve conservadora em `/etc/lact/config.yaml` (exemplo real Polaris: `40:0.2, 55:0.35, 65:0.55, 75:0.8, 85:1.0`, 500ms, `performance_level: auto` — o ID da GPU varia por máquina, nunca copie às cegas); nunca clock/voltagem sem testar estabilidade jogo a jogo. `MESA_SHADER_CACHE_MAX_SIZE=12G` + `RADV_PERFTEST=gpl` em `~/.config/environment.d/gaming.conf` (provisionado pelo envctl); manter RADV, nunca AMDVLK.
6. MangoHud preset (`~/.config/MangoHud/MangoHud.conf`, provisionado pelo envctl): `fps,frametime,cpu/gpu/vram`, toggle Shift+F12, `fps_metrics=avg,0.01` (AVG + 1% low). Launch padrão Steam: `game-performance mangohud --dlsym %command%` (`LD_PRELOAD=""` se overlay/recorder travar).
7. KDE: `[Compositing]` no kwinrc com `AllowBlockCompositing=true` + `UnredirectFullscreen=true` (fullscreen bypassa o compositor); `plasma-x11-session` NÃO puxa o `xorg-server` — instalar o quarteto (`plasma-x11-session`, `xorg-server`, `xf86-input-libinput`, `xf86-video-amdgpu`) junto ou a sessão X11 nem sobe (`/usr/bin/X` ausente) e, sem o driver AMD, cai em modesetting genérico.

## Steam/Proton
- Biblioteca ativa em disco Linux (Btrfs/ext4); NTFS só backup frio (`ntfs-3g`), exFAT via `exfatprogs`; nunca prefix Proton em NTFS/exFAT.
- Proton global Valve; por jogo `proton-cachyos-slr` build **x86-64** (nunca `_v3` sem AVX2); shader pre-cache OFF; `umu-launcher` + `wine-cachyos-opt` como backend fora do Steam.
- AAA pesado e GPU-bound: `gamescope -W 1920 -H 1080 -w 1280 -h 720 -F fsr -- %command%`.
- Controles: as regras udev vêm de `steam-devices` (dependência de `steam`, já declarado) — sem elas o pad aparece mas não mapeia. `protontricks` para ajuste fino de Proton/Wine por jogo.

## Emuladores (Vulkan em todos; resolução = teto sensato p/ i7-2600 + RX 580)
| Emulador | Renderer | Res. interna | Notas |
|---|---|---|---|
| Dolphin | Vulkan (`GFXBackend=Vulkan`, `ShaderCompilationMode=2` async) | Auto (segue a janela) | perfis de controle são configurados na GUI, não versionados aqui (ver abaixo) |
| RetroArch | `video_driver="vulkan"` | nativa | menus Ozone/XMB vêm de `retroarch-assets-ozone` e `retroarch-assets-xmb` (declarados); só cores sem-standalone-bom (genesis-plus-gx, mupen64plus-next); resto é standalone |
| PPSSPP | Vulkan | 4x | sem frameskip |
| PCSX2 | Vulkan (`Renderer=14`), `mtvu=true` | 3x | ⚠️ Pad 1 vem no TECLADO: mapear na GUI (Automatic Mapping) |
| DuckStation | Vulkan, PGXP off (CPU fraca) | 5x | BIOS `scph550x` em `bios/` |
| Azahar | Vulkan (`graphics_api=2`), async + disk cache | 4x | ⚠️ perfil só-teclado: mapear na GUI |
| Eden (Switch) | Vulkan, async GPU + shaders | 1X (CPU-bound, não subir) | AppImage legacy PINNADO, auto-update desligado (troca p/ build AVX2 e quebra tudo) |
| Vita3K | Vulkan | 2x (=1920x1088) | firmware via GUI |
| Cemu | Vulkan (`api=1`), AsyncCompile | packs 1080p via download in-app | `keys.txt` na pasta do Cemu; update SÓ via paru |
| melonDS/mGBA/snes9x/flycast/mednafen | nativo/2D (flycast GL) | nativa/janela (flycast 1080) | standalone > core (link-cable, layouts, netplay, debugger) |

- BIOS nos layouts oficiais de cada emu (DuckStation `bios/`, PCSX2 `bios/`, Azahar `sysdata/`+`nand/`, PPSSPP `flash0/`, Dolphin `GC/{USA,EUR,JAP}/`, RetroArch `system/`, melonDS/mGBA/flycast/snes9x/mednafen nos próprios dirs).
- Controles XInput: SDL automático na maioria; exceções manuais na GUI: PCSX2 (Automatic Mapping), Azahar (Controls), Dolphin (perfis abaixo).
- Dolphin: **perfis de controle NÃO são versionados pelo envctl.** Não existe `profiles/` no repo (a skill `cachyos-gaming-setup`, que os trazia, foi removida em 2026-09-26) — a máquina tem 2 criados à mão (`X360-GameCube-P1/P2.ini`). Para multi-jogador, criar os perfis na GUI (Controllers > Configure > Profile) e salvar em `~/.local/share/dolphin-emu/Config/Profiles/{GCPad,Wiimote}/`; o carregamento é manual por jogador, não automático. Mapeamento sem giroscópio: mira/swing no direcional direito (flick = golpe), sacudir = LB, R3 recentraliza.
- Keys/firmware (Switch, Wii U, Vita, 3DS): SEMPRE do próprio console; mods de jogo casam por build ID (@nsobid) — versão errada = crash. Pastas de ROMs: um dir por sistema em `~/Games/` (NVMe).
- 360 sem gyro: shrines de movimento (BOTW) e afins precisam workaround por jogo (motion source via UDP, ou skip).

## Política de updates
- Canal único: gerenciador de pacotes. Updaters internos: desligar/ignorar (binários root-owned nem aceitam escrita interna). Exceção: AppImages pinnados (Eden legacy) = update manual, nunca automático.

## Verificação
- `envctl doctor`, seção Gaming — silenciosa sem Steam instalado (opt-in):
  - pacotes do `gaming.yaml` (inclui `cachyos-settings` e `cachyos-ananicy-rules`);
  - regras do ananicy (`/etc/ananicy.d/00-types.types` presente → o daemon não está inerte);
  - 4 serviços (`scx_loader`, `lactd`, `ananicy-cpp`, `power-profiles-daemon`);
  - `sched_ext` habilitado;
  - cmdline em 3 tiers: universal, AMD (só com `amdgpu`), panic (INFO);
  - RADV ativo, preset `gaming.conf` com as 2 chaves, preset `MangoHud.conf` presente;
  - tuning privilegiado em `INFO` (contexto, nunca WARN): `/etc/lact/config.yaml`,
    `/etc/scx_loader/config.toml` e o bypass de compositing no kwinrc
    (`[Compositing]` com `AllowBlockCompositing` + `UnredirectFullscreen`);
  - configs de emuladores em `INFO`: cada emulador com o renderer Vulkan
    aplicado (Dolphin, RetroArch, PPSSPP, PCSX2, DuckStation, Azahar, Eden,
    Vita3K, Cemu) — ausência significa que o dono não fez o ajuste, não é drift;
  - `/usr/bin/X` e `[multilib]` ativo.
- `envctl doctor`, seção Performance: `.pacnew` pendentes em `/etc` (INFO, via `pacdiff`), swap/zram, governor, scheduler de I/O, journald, `fstrim.timer` e os 5 daemons.
- Manual (o que nenhum check cobre por depender de arquivo privilegiado ou de gameplay): `systemctl is-active` dos 4 serviços, launch options do Steam, `vulkaninfo | grep RADV`, 1 jogo Steam + 1 emu com MangoHud (AVG + 1% low). Tetos: GPU 85°C, CPU 80°C.
