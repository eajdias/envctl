---
name: aur-headless-install
description: Instalar pacotes AUR de shells não-interativos (agentes, SSH, CI). Use quando paru pedir senha/TTY, falhar no sudo interno ou ao instalar com dependências make faltando. Triggers: paru, AUR, makepkg, sudo sem TTY, "terminal é necessário para ler a senha".
---

# AUR Headless Install

## Quando usar
Shell sem TTY (agente, SSH non-interactive, CI) no Arch/CachyOS: o `sudo` interno do paru falha e trava a conta no faillock após retries.

## Passos
1. NUNCA pipear senha repetidamente. Uma tentativa; se falhar, parar e delegar.
2. Baixar/buildar como usuário (sem sudo):
   ```bash
   paru -G <pacote>  # ou: cd ~/.cache/paru/clone/<pacote>
   cd <pacote> && makepkg --noconfirm
   ```
3. Se faltar makedepends, instalar via pacman (`sudo pacman -S --noconfirm <dep>`) e repetir o `makepkg`.
4. Instalar o artefato com UMA chamada sudo:
   ```bash
   sudo pacman -U --noconfirm <pacote>-<versão>-<arch>.pkg.tar.zst
   ```
5. Limpar o diretório de build após instalar.

## Verificação
- `pacman -Q <pacote>` lista a versão; `which <binário>` resolve.
- Nenhuma tentativa extra de sudo após a primeira falha (checar `faillock --user $USER` se houver dúvida).
