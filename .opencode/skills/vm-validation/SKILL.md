---
name: vm-validation
description: Validar o envctl em uma VM Linux (Hyper-V) antes de declarar uma release multiplataforma pronta. Use quando precisar provar que mudanças de provisionamento/doctor funcionam em Linux real: build cross, upload, doctor 100%, idempotência, smoke test de LSPs e permissões. Triggers: validar na VM, testar linux, cross-compile, hyper-v, vm test, doctor na vm, idempotência.
compatibility: opencode
license: MIT
---

# Validação do envctl em VM Linux

## Quando usar

- Antes de finalizar/releasear mudanças de provisionamento, doctor ou manifests com impacto Linux.
- Para provar 0 WARN/0 ERRO e idempotência em ambiente Linux real (Hyper-V `Ubuntu Server`).

## Passos

1. **Cross-compile** (binário Windows NÃO roda na VM — "Exec format error"):
   ```powershell
   $env:GOOS="linux"; $env:GOARCH="amd64"; $env:CGO_ENABLED="0"
   go build -o C:\temp\envctl-vm\envctl-linux ./cmd/envctl
   Remove-Item Env:GOOS, Env:GOARCH, Env:CGO_ENABLED
   ```
2. **Upload + exec via paramiko** (senha, sem chave): script helper com `sftp.put` e `exec_command` (ex.: `C:\temp\envctl-vm\vmhelper.py` — recriar se não existir; dados de conexão no inventário local `~/.config/opencode/extras/ssh_servers.md`).
3. **Doctor**: `bash -lc '/tmp/envctl doctor'` → meta `EXCELLENT` (0 WARN / 0 ERRO). Filtre ANSI com `sed 's/\x1b\[[0-9;]*m//g'`.
4. **Idempotência**: rodar `bash -lc '/tmp/envctl run all'` pela 2ª vez → grep por `WARNING|ERROR|FAIL` deve ser **0** e sem linhas `Installing` (nada re-instala).
5. **Smoke test LSPs** (sempre via `bash -lc` — PATH do volta só existe no login shell):
   ```bash
   for b in gopls rust-analyzer typescript-language-server pylsp intelephense taplo; do "$b" --version; done
   ```
6. **Permissões**: `stat -c "%a %n" ~/.ssh ~/.ssh-manager ~/Documents/SSH-keys` → `700`.

## Verificação

- `envctl doctor` na VM: `Passed` = Total, Warnings 0, Errors 0.
- 2ª execução de `run all`: zero WARNING/ERROR e zero re-instalações.
- Permissões 700 nos diretórios restritos.
- Rodar `go build ./... && go vet ./... && go test ./...` localmente antes do upload.

## Observações

- Comandos com aspas aninhadas via paramiko quebram — subir script `.sh` por SFTP e executar.
- `gopls --version` falha (usa `gopls version`) — resposta ≠ "not found" conta como OK.
- Dados de conexão da VM (IP/senha/usuário) ficam NO helper local (`C:\temp\envctl-vm\vmhelper.py` ou inventário `~/.config/opencode/extras/ssh_servers.md`) — nunca no repo.