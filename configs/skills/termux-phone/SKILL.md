---
name: termux-phone
description: >-
  Operar o celular Android (Redmi Note 9S) via adb/Termux: sshd, wake-lock,
  pacotes, rclone para o homelab, backup e recovery do Tailscale. Use quando
  precisar mexer no celular, debugar sshd, subir backup ou reacender o Tailscale.
  Triggers: celular, termux, adb, sshd phone, wake-lock, rclone phone,
  backup celular, tailscale celular, redmi.
license: MIT
---

# Termux Phone Ops (redmi-note-9s)

## Quando usar

- `ssh celular` caiu / sshd do Termux morreu.
- Instalar/atualizar pacotes no Termux.
- Backup do celular para o homelab (rclone ou Syncthing).
- Tailscale do celular sumiu da tailnet.
- Ver a tela do celular sem pedir foto ao dono.

## Contexto fixo

- Device: Redmi Note 9S (curtana), MIUI 14.0.3 Global. Modelo = **thin client**:
  o celular ENVIA (Download, Documents, fotos) e ACESSA do PC (FileBrowser, Immich).
- Termux: sshd `:8022`, user `u0_a258`, chave `~/.ssh/celular` (sem senha).
  `health_check` do ssh-manager FALHA no Termux (sem `/proc/net/dev`) — coletar
  métricas na mão.
- Tailscale IP `<TAILNET_IP_CELULAR>`, LAN `<LAN_IP_CELULAR>`. **Sem autostart confiável.**
- rclone remote `homelab` no celular exige `host_key_algorithms ssh-ed25519`
  (stack SSH do Go misnegocia mldsa).
- Root: caminho aprovado = Magisk com boot patchado, sem trocar ROM. Mi bind
  2026-09-24 ~03:30 → unlock após ~2026-10-01. Backups pré-root em
  `/mnt/homelab/backups/celular-20260924/`.

## Passos

### 1. Recovery do sshd (via USB)

```bash
adb devices                      # confirmar
adb shell monkey -p com.termux -c android.intent.category.LAUNCHER 1
sleep 5; ssh -o ConnectTimeout=8 celular 'echo SSH_OK'
```

### 2. Recovery do Tailscale (regra permanente)

```bash
adb shell monkey -p com.tailscale.ipn -c android.intent.category.LAUNCHER 1
# se não voltar: escalar ao dono (toggle Always-on VPN manual no app)
```

### 3. Pacotes (validado: tmux, jq, htop, rsync, mosh, rclone, ncdu, fastfetch)

```bash
ssh celular 'pkg update -y && pkg install -y <pkgs>'
```

### 4. Backup para o homelab

- `~/phone-backup.sh` no Termux (rclone → homelab). Dry-run antes.
- Shares Syncthing (skill `syncthing-ops`): Download→`/phone/Download`,
  Documents→`/docs/from-phone`. Fotos/vídeos = Immich, não Syncthing.

### 5. Ver a tela sem o dono

`adb shell uiautomator dump` falha com app ativo (`could not get idle state`) →
usar sempre:

```bash
adb shell screencap -p /sdcard/screen.png
adb pull -a /sdcard/screen.png /temp/screen.png
adb shell rm /sdcard/screen.png
```

### 6. Debloat via adb (só com aprovação explícita do dono)

`adb shell pm uninstall -k --user 0 <pkg>` por pacote, um de cada vez, com
`Success` confirmado. Manter: WhatsApp Business (+ Databases p/ restore),
apps bancários (Inter, Nubank, BTG), apps vetados antes.

## Verificação

- `ssh celular 'echo OK'` responde.
- `tailscale status | grep <TAILNET_IP_CELULAR>` no PC mostra o celular.
- Backup: contar arquivos origem × destino (ex.: `ls | wc -l`, `du -sh`).
- Ao mexer em developer options/USB debugging: desligar no fim (compatibilidade
  com apps bancários).
