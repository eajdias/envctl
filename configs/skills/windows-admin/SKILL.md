---
name: windows-admin
description: >-
  Administração local do Windows com PowerShell. Use quando o usuário pedir para diagnosticar ou alterar serviços, registro, tarefas agendadas, firewall, usuários locais, eventos ou instalação de programas. Triggers: windows, serviço windows, Get-Service, sc.exe, registry, registro, reg export, rollback, schtasks, tarefa agendada, winget, firewall, New-NetFirewallRule, RemoteAddress, evento, event log, Get-WinEvent, usuário local, Get-LocalUser, instalar, atualizar, GUI do Windows.
license: MIT
---

# Administração do Windows

## Quando usar

Use esta skill para uma tarefa administrativa no Windows. O alvo pode ser a máquina local ou outro host; confirme o host e o escopo antes de executar. Não use um comando apenas porque ele existe em outra versão do Windows: descubra a versão real, os cmdlets disponíveis e o nível de elevação.

## Contexto e verificação inicial

Descubra o que existe no ambiente em vez de assumir uma edição, um build ou um caminho específico:

```powershell
$PSVersionTable
Get-ComputerInfo -Property WindowsProductName,WindowsVersion,OsBuildNumber
Get-Command Get-Service,Get-NetFirewallRule,Get-WinEvent -ErrorAction SilentlyContinue
```

Verifique elevação antes de uma mudança sensível:

```powershell
$principal = [Security.Principal.WindowsPrincipal]::new(
  [Security.Principal.WindowsIdentity]::GetCurrent()
)
$principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
```

Se o resultado for falso, explique que a operação precisa ser repetida em um terminal elevado. Não tente contornar UAC, não copie credenciais e não aplique `--disable-interactivity` a um instalador que possa pedir uma decisão sem mostrar o que fará.

## Protocolo de segurança

Comece com leitura e colete evidência antes de reiniciar, parar, editar, remover ou instalar. Para qualquer comando que mude estado:

1. mostre o comando e o impacto esperado;
2. peça confirmação quando houver dúvida, risco ou dados envolvidos;
3. execute uma etapa por vez;
4. verifique o resultado e registre a falha, se houver.

Não edite configuração do OpenCode, CommandCode ou outro agente por conta própria para resolver uma tarefa do Windows. Se a correção depender dessa configuração, pare e explique ao usuário.

## Serviços

Consulta inicial:

```powershell
Get-Service | Sort-Object Status,Name
Get-CimInstance Win32_Service |
  Select-Object Name,State,StartMode,PathName
Get-WinEvent -FilterHashtable @{
  LogName='Application'
  StartTime=(Get-Date).AddHours(-1)
} -ErrorAction SilentlyContinue |
  Select-Object TimeCreated,ProviderName,Id,LevelDisplayName,Message
```

Depois do diagnóstico, `Start-Service`, `Stop-Service` e `Restart-Service` são operações mutáveis. Confirme o serviço, o impacto e a forma de recuperação. Para alterar inicialização, use `Set-Service -StartupType` somente com o valor explicitamente aprovado e registre o valor anterior.

## Registro do Windows

Trate qualquer escrita no registro como mudança de sistema. Faça backup do escopo exato antes de editar e valide o arquivo de backup:

```powershell
$stamp = Get-Date -Format 'yyyyMMdd-HHmmss'
$backup = Join-Path ([Environment]::GetFolderPath('MyDocuments')) `
  "windows-registry-$stamp.reg"
reg.exe export 'HKLM\Software\<chave-alvo>' "$backup"
if ($LASTEXITCODE -ne 0) { throw 'Falha ao exportar o registro' }
Get-Item -LiteralPath $backup | Select-Object FullName,Length,LastWriteTime
```

Use `Get-ItemProperty -LiteralPath` para ler e `Set-ItemProperty -LiteralPath`/`New-Item` para alterar, sempre com `-ErrorAction Stop`. Não use `Remove-Item` em uma chave ou valor sem identificar os impactos e sem backup.

### Rollback

O rollback precisa ser uma operação consciente; importar o arquivo inteiro pode sobrescrever alterações feitas depois do backup:

```powershell
# Revise a data, o caminho e as diferenças antes de importar.
reg.exe import $backup
if ($LASTEXITCODE -ne 0) { throw 'Falha ao restaurar o registro' }
```

Confirme que o arquivo é a exportação da chave correta e que não houve mudanças posteriores. Se a mudança fez parte de um conjunto maior, restaure somente o escopo afetado ou use o procedimento de reparo do fabricante. Depois do rollback, releia a chave e valide o serviço/aplicação dependente.

## Firewall

Primeiro leia perfis e regras existentes:

```powershell
Get-NetFirewallProfile |
  Select-Object Name,Enabled,DefaultInboundAction,DefaultOutboundAction
Get-NetFirewallRule -DisplayName '*<nome>*>' -ErrorAction SilentlyContinue
Get-NetFirewallPortFilter -ErrorAction SilentlyContinue
```

Ao criar uma regra, declare todos os limites relevantes. `-Profile` evita que a regra valha em todos os perfis sem revisão; `-RemoteAddress` limita quem pode alcançar a porta:

```powershell
New-NetFirewallRule `
  -DisplayName '<nome-da-regra>' `
  -Direction Inbound `
  -Action Allow `
  -Protocol TCP `
  -LocalPort <porta> `
  -LocalAddress <endereco-local> `
  -RemoteAddress <escopo-remoto> `
  -Profile Domain,Private `
  -ErrorAction Stop
```

Escolha `Domain`, `Private` ou `Public` conforme a política do ambiente. Use `LocalSubnet`, uma faixa necessária ou outro escopo explícito; `Any` só com justificativa e aprovação. Para uma regra já existente, altere uma cópia revisada ou edite as propriedades após exportar a configuração; não apague regras sem identificar a política que dependem delas.

A regra pode exigir admin. Depois de criar, consulte:

```powershell
Get-NetFirewallRule -DisplayName '<nome-da-regra>' |
  Get-NetFirewallAddressFilter
Get-NetFirewallRule -DisplayName '<nome-da-regra>' |
  Get-NetFirewallPortFilter
```

### GUI sob demanda

Use a interface apenas quando o usuário pedir passos visuais, a política exigir documentação ou o cmdlet não estiver disponível. Não abra ferramentas GUI como parte de toda alteração automatizada:

```powershell
Start-Process wf.msc
```

No Windows, o caminho típico é **Windows Security → Firewall & network protection → Advanced settings**, ou a página de configurações de firewall. Confirme Profile, direção, ação, protocolo, porta, endereço local e escopo remoto antes de **Concluir** ou **Aplicar**. Se usar a GUI, registre a configuração resultante para comparação.

## Instalação de software

Verifique qual gerenciador existe:

```powershell
Get-Command winget,choco -ErrorAction SilentlyContinue
```

Se `winget` estiver disponível e a política permitir, prefira o pacote exato e visível:

```powershell
winget install --exact --id <Package.Id> `
  --accept-source-agreements `
  --accept-package-agreements
```

Não presuma que o gerenciador alternativo esteja instalado, nem fixe uma versão de winget ou PowerShell sem medir a máquina. Se `winget` não tiver o pacote, verifique outra fonte aprovada e peça confirmação antes de instalar por `choco`, MSI ou outro método. Não use um instalador de origem desconhecida só para concluir a tarefa.

## Tarefas agendadas, eventos e usuários

- `Get-ScheduledTask`, `Get-ScheduledTaskInfo`, `Register-ScheduledTask`, `Unregister-ScheduledTask` e `schtasks.exe` têm permissões e impactos diferentes. Inspecione a tarefa, seus gatilhos e o principal de execução antes de alterar.
- `Get-WinEvent -FilterHashtable` é adequado para uma janela inicial. Para volume, filtre no servidor e limite a quantidade de eventos para não despejar dados sensíveis.
- `Get-LocalUser` e `Get-LocalGroupMember` ajudam a inspecionar contas locais. `New-LocalUser`, `Remove-LocalUser` e alterações de grupo são mutações; exigem admin e aprovação explícita.

Exemplos de leitura:

```powershell
Get-ScheduledTask | Select-Object TaskName,State
Get-ScheduledTaskInfo -TaskName '<tarefa>'
Get-LocalUser | Select-Object Name,Enabled,LastLogon
Get-LocalGroupMember -Group '<grupo>'
Get-WinEvent -LogName Application -MaxEvents 50
```

## Verificação

Depois de uma mudança autorizada, repita a leitura e valide o efeito:

```powershell
Get-Service -Name '<servico>' | Select-Object Name,Status,StartType
Get-ItemProperty -LiteralPath 'HKLM:\Software\<chave-alvo>' -ErrorAction SilentlyContinue
Get-NetFirewallRule -DisplayName '<nome-da-regra>' | Format-List
Get-WinEvent -LogName Application -MaxEvents 20
```

Confirme também o comportamento do serviço ou aplicativo e a possibilidade de rollback. Se a saída indicar erro, preserve a mensagem, pare a sequência e não tente outro comando destrutivo para mascará-la.
