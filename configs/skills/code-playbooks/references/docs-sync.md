# Auditoria de docs

## Regra

> Manifesto e código são a fonte da verdade. A doc que diverge do comportamento está
> errada — conserte a doc. Mas a doc ausente também é erro: quem chega depois vai
> procurar e não achar.

## Roteiro

1. **Escopo**: qual mudança foi feita e o que ela promete (README, `AGENTS.md`, `docs/`).
2. **Código primeiro**: o que a implementação faz agora? (leia o código, não a doc)
3. **Manifesto**: o que está declarado (campos, defaults, escopo por SO/agente).
4. **Divergências**, com `file:line` de cada lado. Separe:
   - doc **errada** (diz o que não é) → corrige a doc;
   - doc **ausente** (feature sem doc) → escreve onde a doc do tema vive;
   - doc **obsoleta** (assunto que não existe mais) → remove.
5. **Números**: contagem de skill/pacote/LSP é o caso que mais mente. Confira a
   verdade no manifesto, não no texto.
6. **Promessa que o código não cumpre**: é bug de produto, não de doc — reporte antes de
   ajustar o texto. Nunca "suavizar" a doc para esconder promessa quebrada.
7. Ao fechar: releia o diff da doc procurando outro trecho que dependia do que mudou.

## Onde a doc de cada assunto vive

- Comportamento do produto → README do repo.
- Detalhe por subsistema → `docs/` do repo.
- Ambiente desta máquina e preferências → `AGENTS.md` (não em skill).
- Conhecimento de domínio → `code-playbooks/references/`, não no README.

## Verificação

- [ ] Cada afirmação nova da doc confere com código ou manifesto (`file:line`).
- [ ] Contagens conferidas no manifesto, não copiadas do texto anterior.
- [ ] Nenhum caminho absoluto, host ou nome de cliente no arquivo versionado.
