# Preferências do dono

> Regras do dono para agentes nesta máquina. Valem em qualquer repo/projeto
> provisionado pelo `envctl`.

## Comunicação

- Com o dono: **PT-BR**, direto, curto. Código/comentários/commits em inglês.
- Code first, explicação depois (máx 3 linhas). Sem ensaio, sem tour de features.

## Trabalho

- **Evidência antes de afirmação**: testar antes de declarar pronto
  (login na API, sha256 de restore, fila do downloader — nunca "deve funcionar").
- **Shell não-interativo**: sem `vim/less/man`, sem `git add -p`/`rebase -i`;
  `sudo -n` (1 tentativa — faillock trava a conta; falhou, delega ao dono).
- **Diff mínimo, poucos arquivos.** Deletar > adicionar. Sem abstração
  não pedida, sem scaffold "p/ depois". Stdlib/nativo antes de dependência.
- **Atenção especial**: validação em trust boundary, erro que evita perda de
  dado, segurança, qualquer coisa explicitamente pedida — nunca simplificar.
- **Scratch**: `/temp/` (`ENVCTL_TEMP`, fora do projeto e do sistema;
  `/tmp/opencode/` é legado — não usar). **Apagar ao concluir.**
- **Memória do agente**: skill `agent-memory` — ler 1x no início (projeto
  `.opencode/memory/` → global `~/.config/opencode/memory/`), gravar lições no
  fim. Erro repetido = bug do processo, não azar.
- **Nunca afirmar ferramenta ausente sem prova**: antes de dizer "não tem",
  rodar `command -v <tool>`, `volta list` (CLIs Node), `opencode mcp list` e
  `envctl doctor`. Memória/doc antiga não é prova — o estado real da máquina é.

## Gostos técnicos

- Tailnet-only, sem porta pública.
- Imagens `:latest`/`:stable` + freshdock; pin só com motivo.
- Mídia (vídeo/música/mangá) no eajdias-cast. Fotos fora de nuvem de terceiro.
- Celular é thin client: só envia (Download, Documents, fotos); lê do PC
  (FileBrowser, Immich).
