> Guia de provisionamento e auditing do stack de jogos do CachyOS.
> Última migração da skill `cachyos-gaming-setup`: 2026-09-26.

# Guia: CachyOS Gaming Stack

## Escopo

O `envctl` provisiona **pacotes** (`gaming.yaml`) e **presets de usuário**
(`gaming.conf`, `MangoHud.conf`) de forma idempotente. Tudo que exige root, reboot ou
decisão de segurança **não** é provisionado: é documentado aqui como orientação manual,
e o `doctor` o audita em modo read-only (nunca `--fix`).

Este guia substituiu a skill global `cachyos-gaming-setup`, removida do catálogo de
agentes em 2026-09-26: é conhecimento do produto e pertence ao repo, não ao tier global.

## Sistema (verificar, não presumir)
1. CPU sem AVX2 (ex.: Sandy Bridge, x86-64-v2)? Então: só repos genéricos (nunca v3/v4), AppImages legacy, e testar SIGILL em qualquer binário novo. Emuladores exigentes (RPCS3, ShadPS4, Switch AAA, xemu, simple64) são inviáveis — não instalar.
2. Kernel cmdline (Limine `/etc/default/limine` + `limine-update` + reboot): `preempt=full split_lock_detect=off amdgpu.ppfeaturemask=0xffffffff zswap.enabled=0`. `mitigations=off` é decisão manual aprovada caso a caso (trade-off Spectre/Meltdown) — o envctl NÃO o gerencia nem audita, de propósito.
3. `power-profiles-daemon` balanced no desktop; jogos via wrapper `game-performance %command%` (vem do `cachyos-settings`, não é pacote próprio). Manter `ananicy-cpp`, NUNCA combinar com `gamemode`.
4. Scheduler: `/etc/scx_loader/config.toml` com `default_sched="scx_bpfland"` + `default_mode="Auto"`; `game-performance` troca p/ perfil gaming sozinho. Rollback: `systemctl disable --now scx_loader`.
5. GPU AMD: `lactd` ativo + fan curve conservadora em `/etc/lact/config.yaml` (exemplo real Polaris: `40:0.2, 55:0.35, 65:0.55, 75:0.8, 85:1.0`, 500ms, `performance_level: auto` — o ID da GPU varia por máquina, nunca copie às cegas); nunca clock/voltagem sem testar estabilidade jogo a jogo. `MESA_SHADER_CACHE_MAX_SIZE=12G` + `RADV_PERFTEST=gpl` em `~/.config/environment.d/gaming.conf` (provisionado pelo envctl); manter RADV, nunca AMDVLK.
6. MangoHud preset (`~/.config/MangoHud/MangoHud.conf`, provisionado pelo envctl): `fps,frametime,cpu/gpu/vram`, toggle Shift+F12, `fps_metrics=avg,0.01` (AVG + 1% low). Launch padrão Steam: `game-performance mangohud --dlsym %command%` (`LD_PRELOAD=""` se overlay/recorder travar).
7. KDE: `[Compositing]` no kwinrc com `AllowBlockCompositing=true` + `UnredirectFullscreen=true` (fullscreen bypassa o compositor); `plasma-x11-session` NÃO puxa o `xorg-server` — instalar o trio (`plasma-x11-session`, `xorg-server`, `xf86-input-libinput`) junto ou a sessão X11 nem sobe (`/usr/bin/X` ausente).

## Steam/Proton
- Biblioteca ativa em disco Linux (Btrfs/ext4); NTFS só backup frio (`ntfs-3g`), exFAT via `exfatprogs`; nunca prefix Proton em NTFS/exFAT.
- Proton global Valve; por jogo `proton-cachyos-slr` build **x86-64** (nunca `_v3` sem AVX2); shader pre-cache OFF; `umu-launcher` + `wine-cachyos-opt` como backend fora do Steam.
- AAA pesado e GPU-bound: `gamescope -W 1920 -H 1080 -w 1280 -h 720 -F fsr -- %command%`.

## Emuladores (Vulkan em todos; resolução = teto sensato p/ i7-2600 + RX 580)
| Emulador | Renderer | Res. interna | Notas |
|---|---|---|---|
| Dolphin | Vulkan (`GFXBackend=Vulkan`, `ShaderCompilationMode=2` async) | Auto (segue a janela) | 10 perfis X360 embutidos (`profiles/`): carregar na GUI por jogador |
| RetroArch | `video_driver="vulkan"` | nativa | só cores sem-standalone-bom (genesis-plus-gx, mupen64plus-next); resto é standalone |
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
- Dolphin: 10 perfis Xbox 360 embutidos nesta skill (`profiles/GCPad/`, `profiles/Wiimote/`: GameCube, Nunchuk, Sideways, Upright, Classic × P1/P2). Instalar os necessários em `~/.local/share/dolphin-emu/Config/Profiles/{GCPad,Wiimote}/` e carregar na GUI (Controllers > Configure > Profile > Load). Mapeamento sem giroscópio: mira/swing no direcional direito (flick = golpe), sacudir = LB, R3 recentraliza.
- Keys/firmware (Switch, Wii U, Vita, 3DS): SEMPRE do próprio console; mods de jogo casam por build ID (@nsobid) — versão errada = crash. Pastas de ROMs: um dir por sistema em `~/Games/` (NVMe).
- 360 sem gyro: shrines de movimento (BOTW) e afins precisam workaround por jogo (motion source via UDP, ou skip).

## Política de updates
- Canal único: gerenciador de pacotes. Updaters internos: desligar/ignorar (binários root-owned nem aceitam escrita interna). Exceção: AppImages pinnados (Eden legacy) = update manual, nunca automático.

## Verificação
- `envctl doctor`: seção Gaming (pacotes do `gaming.yaml` + 4 serviços + `sched_ext` + cmdline sem mitigations + RADV + preset shader + `/usr/bin/X` + multilib). Silenciosa sem Steam instalado.
- Manual: `systemctl is-active scx_loader lactd ananicy-cpp power-profiles-daemon` (4/4), `/proc/cmdline` com os params, `vulkaninfo | grep RADV`, 1 jogo Steam + 1 emu com MangoHud (AVG + 1% low). Tetos: GPU 85°C, CPU 80°C.
