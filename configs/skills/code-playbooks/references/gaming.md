# Gaming e emulação

> **Fronteira:** isto é *operar e ajustar* o stack de games. *Provisionar*
> pacotes e AUR é do `envctl` (`manifests/gaming.yaml`, perfil `cachyos`).

## Frontend e organização

- **ES-DE** (AppImage) é o frontend único; ROMs em `/mnt/homelab/games/roms`
  (padrão do homelab; ajuste por máquina); overrides p/ standalones em
  `~/ES-DE/custom_systems/es_systems.xml`.
- `steam-sync.sh` gera entradas Steam a partir do ES-DE (arte via Akamai —
  Cloudflare costuma estar bloqueado no AdGuard).
- Regra de engine: **core libretro só quando o standalone não tem render HW
  ou não existe** — não duplicar engine à toa.

## Emuladores

- Standalones por sistema: dolphin (GC/Wii), mgba-qt (GBA), ppsspp (PSP),
  snes9x-gtk (SNES), mednafen (PSX/Saturn), cemu (Wii U), duckstation (PS1),
  flycast (Dreamcast), melonds (DS), pcsx2 (PS2), azahar (3DS), vita3k (Vita),
  hactool (chaves/conteúdo de ROM Switch), skyscraper (scrape de capa/vídeo
  para o ES-DE).
- retroarch + cores (genesis-plus-gx, mupen64plus-next, mame) só onde não há
  standalone bom (16-bit, arcade, N64).
- **Eden** (Switch) AppImage pinnado em `Games/switch/`, auto-update
  desligado, fora dos timers de update.

## Controle e streaming

- `emu-wrap`: L3+R3 por 0,6s mata qualquer emulador travado em fullscreen
  (onde teclado/mouse não alcançam). Confirms off, fullscreen default.
- **Sunshine** (`encoder=vaapi`, h264+hevc via VCE) + **Moonlight**: stream do
  PC para notebook/celular — clientes fracos nunca emulam pesado.
- RetroAchievements: login configurado nos emuladores que suportam.
- Overlay e limit de FPS: gamescope/mangohud/goverlay.
- GPU AMD: `lact` + `amdgpu_top` (power-limit, fans, VRAM) — freezes longos
  com suspeita de hardware se investigam aqui antes de culpar software.
- Boosteroid (assinatura): streaming de AAA que o host não roda.

## Saves

- Ativos no NVMe (latência); **espelho semanal** em `games/saves/`
  (`.srm`, memcards, `eden/nand`, `melonDS/saves`).
- **Nunca** sync vivo nem states entre builds (`sstates/`/`savestates/` não cruzam).
