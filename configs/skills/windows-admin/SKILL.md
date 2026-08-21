---
name: windows-admin
description: Administração local do Windows 11. Use quando o usuário pedir para gerenciar serviços Windows, registro, tarefas agendadas, firewall, usuários locais, eventos, instalar/atualizar programas, ou qualquer operação de administração do sistema operacional. Triggers: windows, serviço windows, Get-Service, sc.exe, registry, registro, schtasks, tarefa agendada, winget, firewall, evento, event log, Get-WinEvent, usuário local, Get-LocalUser, restart service, serviço parado, iniciar serviço.
---

# Administração Windows 11 (máquina local)

## Contexto

- Máquina local: Windows 11 (win32), shell default MSYS2 bash (`C:\msys64\usr\bin\bash.exe`).
- Operações nativas do Windows devem preferir **PowerShell 5.1/pwsh 7.6.5** — use `bash -lc "powershell -NoProfile -Command '...'"` ou, se o bash tool estiver em PS, os cmdlets diretamente. Nunca tente imitar cmdlets do Windows em bash puro.
- LSP PowerShell configurado (PowerShellEditorServices v4.7.0 em `~/Documents/PowerShell/Modules/PowerShellEditorServices`).
- Sessão pode não ser elevada: verificar com `([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)` antes de operações que exijam admin.

## Operações comuns

### Serviços
- Listar: `Get-Service` | Ver um: `Get-Service <nome>` | Iniciar/parar/reiniciar: `Start-Service`, `Stop-Service`, `Restart-Service` (ou `sc.exe start/stop <nome>`).
- Mudar tipo de inicialização: `Set-Service <nome> -StartupType Automatic|Manual|Disabled`.

### Registro
- `Get-ItemProperty -Path 'HKLM:\...'`, `Set-ItemProperty`, `New-Item`, `Remove-Item` — sempre com `-ErrorAction Stop` e `-LiteralPath` quando houver caracteres especiais.

### Tarefas agendadas
- `schtasks /query /tn <nome>` / `schtasks /create ...` / `schtasks /run` / `schtasks /end` — ou cmdlets `Get-ScheduledTask`, `New-ScheduledTask` (requer módulo ScheduledTasks).

### Firewall
- `Get-NetFirewallRule -DisplayName '*nome*'`, `New-NetFirewallRule -DisplayName ... -Direction Inbound -Action Allow -Protocol TCP -LocalPort <porta>` (requer admin).

### Instalação de software
- Preferir **winget**: `winget install -e --id <Id> --accept-source-agreements --accept-package-agreements --disable-interactivity`. Funciona no MSYS2 bash (verificado).
- Chocolatey existe (2.7.3) mas só o choco em si está instalado — usar apenas se winget não tiver o pacote.

### Eventos
- `Get-WinEvent -FilterHashtable @{LogName='Application'; Level=2; StartTime=(Get-Date).AddHours(-1)}` — para diagnóstico de falhas recentes.

### Usuários locais
- `Get-LocalUser`, `Get-LocalGroup`, `New-LocalUser` (admin).

## Regras

- Operações que mudam estado (iniciar serviço, instalar, editar registro/firewall): confirmar com o usuário antes se houver dúvida de impacto.
- Não editar configuração do opencode (opencode.jsonc) por conta própria — se algo exigir mudança de config, parar e pedir ao usuário.
- Para diagnóstico de serviços: sempre coletar `Get-Service` + eventos do Application log antes de reiniciar.