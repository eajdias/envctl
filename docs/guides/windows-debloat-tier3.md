# Guia: Windows Debloat — Tier 3 (manual, opt-in)

A parte automatizada e idempotente mora no `envctl run debloat` (`manifests/debloat.yaml`:
registro de telemetria/privacidade, visuais de gaming, 34 Appx, 9 serviços `Disabled`,
11 serviços `Manual` e 4 startup entries — 94 tweaks). Tudo abaixo é **destrutivo,
exige admin/reboot ou decisão caso a caso** — por isso é manual, nunca auto-fix.
Rode cada bloco só com aprovação explícita do dono.

> Este guia substituiu a skill global `windows-debloat`, removida do catálogo de agentes
> em 2026-09-26: é conhecimento do produto e pertence ao repo, não ao tier global.

Pré-requisito: PowerShell elevado. Checar antes:

```powershell
([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
```

## 1. Pré-visualizar o automatizado (sem mudar nada)

O `doctor` mostra o Tier 1+2 como linhas `Debloat / category <nome>` (`INFO` = drift,
`OK` = aplicado). Detalhe por tweak só no output do próprio comando. Nada é aplicado
sem rodar explicitamente:

```powershell
envctl doctor
envctl run debloat
```

## 2. OneDrive (remoção completa)

Remove o processo, o setup, a chave Run e o namespace do Explorer. A pasta
`$env:USERPROFILE\OneDrive` só apaga com confirmação — **backup antes**:

```powershell
Stop-Process -Name 'OneDrive' -Force -ErrorAction SilentlyContinue
& "$env:SystemRoot\System32\OneDriveSetup.exe" /uninstall
$p = 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Run'
if (Get-ItemProperty -Path $p -Name 'OneDrive' -ErrorAction SilentlyContinue) { Remove-ItemProperty -Path $p -Name 'OneDrive' }
Remove-Item -Path 'HKCR:\CLSID\{018D5C66-4533-4307-9B53-224DE2ED1FE6}' -Recurse -Force -ErrorAction SilentlyContinue
Remove-Item -Path 'HKCR:\Wow6432Node\CLSID\{018D5C66-4533-4307-9B53-224DE2ED1FE6}' -Recurse -Force -ErrorAction SilentlyContinue
```

## 3. Energia (desktop vs laptop — decidir caso a caso)

```powershell
powercfg /setactive 8c5e7fda-e8bf-4a96-9a85-a6e23a8c635c   # High Performance
powercfg /setacvalueindex SCHEME_CURRENT SUB_PROCESSOR PERFBOOSTMODE 2  # turbo boost
powercfg /setactive SCHEME_CURRENT
powercfg.exe /hibernate off   # SÓ desktop com SSD (quebra hibernação em laptop)
```

## 4. Teredo (latência em jogos; quebra party chat do Xbox)

```powershell
netsh interface teredo set state disabled
# reverter: netsh interface teredo set state client
```

## 5. UserPreferencesMask (performance visual, valor binário)

O `run debloat` não escreve `Binary` (só `DWord`/`String`/`Appx`/`Service`):

```powershell
Set-ItemProperty -Path 'HKCU:\Control Panel\Desktop' -Name 'UserPreferencesMask' -Type Binary -Value ([byte[]](144,18,3,128,16,0,0,0))
```

## 6. Docker Desktop + WSL (limites de CPU/memória)

Com backup atômico antes de sobrescrever (padrão do envctl):

```powershell
Copy-Item "$env:USERPROFILE\.wslconfig" "$env:USERPROFILE\.wslconfig.bak.$(Get-Date -Format 'yyyyMMdd-HHmmss')" -ErrorAction SilentlyContinue
Set-Content -Path "$env:USERPROFILE\.wslconfig" -Value "[wsl2]`nmemory=4GB`nprocessors=2" -Force
```

## 7. Copilot / Recall / Widgets (fora do Appx automático)

```powershell
Get-AppxPackage -AllUsers '*Copilot*' | Remove-AppxPackage -AllUsers -ErrorAction SilentlyContinue
winget uninstall -e --name 'Copilot' --silent --force --accept-source-agreements
Disable-WindowsOptionalFeature -FeatureName Recall -Online -NoRestart -ErrorAction SilentlyContinue
Get-Process *Widget* -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue
```

## 8. Excluído do automático: só o Xbox suite

`MSTeams`, `OutlookForWindows` e o provider do Windows AI já entraram no
`debloat.yaml` (Tier 2, 2026-09-26). O que fica de fora é o **Xbox suite**, por
ser gaming:

```powershell
Get-AppxPackage -Name 'Microsoft.GamingApp' -AllUsers | Remove-AppxPackage -AllUsers
Get-AppxPackage -Name 'Microsoft.XboxApp' -AllUsers | Remove-AppxPackage -AllUsers
Get-AppxPackage -Name 'Microsoft.XboxGamingOverlay' -AllUsers | Remove-AppxPackage -AllUsers
```

## 9. Serviços: `Manual` no automático, `Disabled` só manual

`WSearch`, `SysMain`, `NgcSvc`, `wbengine`, `OneSyncSvc`, `Dell*` e `fb*` já
entram no `debloat.yaml` com `value: "Manual"` — o serviço ainda sobe sob demanda
e nada do que a máquina depende quebra. Para o comportamento do legado
(`windows11-clean`, que desliga em vez de relaxar), troque `value` para
`"Disabled"` no manifest ou:

```powershell
Set-Service -Name '<nome>' -StartupType Manual   # sobe sob demanda
Set-Service -Name '<nome>' -StartupType Disabled # nunca sobe
```

Fora do manifest de propósito: `Spooler` (impressão), serviços de acesso remoto
(`AnyDesk`, `Firebird*`, `Tailscale`, `sshd`), `StorSvc`, `gupdate*`,
`EasyAntiCheat*`, `NgcRingFenceSvc`.

## 9.1 Startup entries: o que o `run debloat` remove (e o que nunca remove)

O manifest remove 4 entradas de startup — `BraveSoftware`, `Canva`,
`MicrosoftEdge`, `SecurityHealth` — via `Win32_StartupCommand`. O predicado
`startupLocationRemovable()` limita o escopo a **Run keys e pasta Startup**.

Duas exclusões deliberadas:

- **Serviços nunca são tocados.** A mesma classe `Win32_StartupCommand` também
  enumera serviços; apagar a linha errada quebraria o serviço. Uma linha cujo
  `Location` é `...\Services\...` conta como conforme (verde no doctor) e o apply
  é no-op.
- **Serviços de áudio ficaram de fora**: `WavesSvc` e `RtkAuduService` são
  drivers e desligá-los pode quebrar o áudio da máquina.

Para auditar antes de aplicar:

```powershell
Get-CimInstance -ClassName Win32_StartupCommand |
    Select-Object Name, Location, Command |
    Format-Table -AutoSize
```

Removendo uma entrada à mão:

```powershell
Get-CimInstance -ClassName Win32_StartupCommand |
    Where-Object { $_.Name -eq 'SecurityHealth' } |
    ForEach-Object { $_.Delete() }
```

## 10. Telemetria fora do registro

```powershell
Set-MpPreference -SubmitSamplesConsent 2   # Defender: nunca enviar amostras
[Environment]::SetEnvironmentVariable('POWERSHELL_TELEMETRY_OPTOUT', '1', 'Machine')
```

## Regras

- Nada aqui roda via `doctor --fix` (só `run debloat` + estes comandos manuais).
- Reboot invalida verificação na mesma sessão: re-rode `envctl doctor` depois.
- `mitigations=off` e tuning de kernel ficam fora de propósito (aprovação explícita).
