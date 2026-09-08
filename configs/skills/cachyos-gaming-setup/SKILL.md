---
name: cachyos-gaming-setup
description: Tune CachyOS for gaming/emulation (kernel params, scheduler, GPU, Steam, emulators, BIOS). Use when setting up or auditing a CachyOS gaming box, or after envctl run gaming. Triggers: gaming setup, otimização jogos, emuladores, Steam Proton, mangohud, LACT, mitigations.
---

# CachyOS Gaming Setup

## Quando usar
Após `envctl run gaming`, ou para auditar tuning existente. Cobre o que manifests não expressam (kernel cmdline, BIOS dumps, perfis de controle, políticas de update).

## Sistema (verificar, não presumir)
1. CPU sem AVX2 (ex.: Sandy Bridge, x86-64-v2)? Então: só repos genéricos (nunca v3/v4), AppImages legacy, e testar SIGILL em qualquer binário novo. Emuladores exigentes (RPCS3, ShadPS4, Switch AAA, xemu, simple64) são inviáveis — não instalar.
2. Kernel cmdline (Limine `/etc/default/limine` + `limine-update`): base + `mitigations=off preempt=full split_lock_detect=off amdgpu.ppfeaturemask=0xffffffff zswap.enabled=0`. `mitigations=off` só com aprovação (trade-off Spectre/Meltdown).
3. `power-profiles-daemon` balanced no desktop; jogos via wrapper `game-performance %command%`. Manter `ananicy-cpp`, NUNCA combinar com `gamemode`.
4. Scheduler: `scx_loader` com `scx_bpfland` modo Auto (`/etc/scx_loader/config.toml`); `game-performance` troca p/ perfil gaming sozinho.
5. GPU AMD: `lactd` ativo + fan curve conservadora em `/etc/lact/config.yaml` (nunca clock/voltagem às cegas); `MESA_SHADER_CACHE_MAX_SIZE=12G` + `RADV_PERFTEST=gpl` em `~/.config/environment.d/gaming.conf`; manter RADV.
6. MangoHud preset (`fps,frametime,cpu/gpu/vram`, toggle Shift+F12); launch padrão Steam: `game-performance mangohud --dlsym %command%`.

## Steam/Proton
- Biblioteca ativa em disco Linux (Btrfs/ext4); NTFS só backup frio (`ntfs-3g`).
- Proton global Valve; por jogo `proton-cachyos-slr` build **x86-64** (nunca `_v3` sem AVX2); shader pre-cache OFF.

## Emuladores
- Vulkan em todos + resolução interna 1080p-classe onde a GPU sustenta (PSP 4x, PS2 3x, PS1 5x, 3DS 4x, DC 1080); Switch 1X (CPU-bound).
- BIOS nos layouts oficiais de cada emu (DuckStation `bios/`, PCSX2 `bios/`, Azahar `sysdata/`+`nand/`, PPSSPP `flash0/`, Dolphin `GC/{USA,EUR,JAP}/`, RetroArch `system/`, melonDS/mGBA/flycast/snes9x/mednafen nos próprios dirs).
- Controles XInput: SDL automático na maioria; exceções manuais na GUI: PCSX2 (Automatic Mapping), Azahar (Controls), Dolphin (perfis por tipo: GC/Nunchuk/Sideways/Upright/Classic).
- Keys/firmware (Switch, Wii U, Vita, 3DS): SEMPRE do próprio console; mods de jogo casam por build ID (@nsobid) — versão errada = crash.

## Política de updates
- Canal único: gerenciador de pacotes. Updaters internos: desligar/ignorar. Exceção: AppImages pinnados (Eden legacy) = update manual, nunca automático.

## Verificação
- `systemctl is-active scx_loader lactd ananicy-cpp power-profiles-daemon` (4/4), `/proc/cmdline` com os params, `vulkaninfo | grep RADV`, 1 jogo Steam + 1 emu com MangoHud (AVG + 1% low).
