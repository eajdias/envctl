# Visão: o modelo do envctl

Este documento responde "o que é o projeto, por que ele existe e como ele
pensa". Diferente de [architecture.md](architecture.md) (como o código se
organiza) e [principles.md](principles.md) (diretrizes de engenharia), aqui
fica o **porquê** e o **contrato de produto** — as decisões estruturais estão
registradas, uma a uma, em [`adr/`](adr/).

---

## 1. Por que o projeto existe

Um ambiente de desenvolvimento moderno não é só um conjunto de programas: é
shell, toolchains, git, agentes de IA (OpenCode/CommandCode), skills,
permissões, memória e verificação. Mantido à mão, esse estado:

- **diverge** entre máquinas — funciona no notebook, quebra na VPS;
- **envelhece em silêncio** — uma configuração que ninguém audita;
- **não se replica com fidelidade** — a "receita" vive na cabeça de quem fez.

O envctl existe para tornar esse estado **declarável, convergível, auditável e
replicável**: em vários sistemas operacionais, para vários agentes de IA, sem
depender da memória de quem opera.

## 2. O que ele é — e o que não é

**É:** um provisionador declarativo e um auditor de conformidade; um pacote
autocontido (binário com manifests, configs e skills embutidos) que converge
Windows 11, Ubuntu/Debian e Arch/CachyOS ao estado declarado — incluindo o
ambiente dos agentes; um orquestrador de frota (VPS como workers via SSH).

**Não é:** um instalador genérico interativo; um dotfiles manager; um
gerenciador de estado remoto (não há servidor central — cada máquina converge
sozinha a partir do mesmo binário); um framework de deploy genérico.

**É opinativo por design.** O estado desejado está nos manifestos deste
repositório. Mudar o comportamento de uma máquina = mudar o manifesto (o repo
é a fonte de verdade). Qualquer pessoa pode rodar o projeto, mas o conteúdo
segue as decisões documentadas aqui — não é um configurador que pergunta tudo.

## 3. Modelo de operação: declarar → convergir → auditar

1. **Declarar** — `manifests/*.yaml` descrevem o estado desejado por OS/distro
   (pacotes, toolchains, configs, tweaks, skills, performance).
2. **Convergir** — `envctl run [perfil|subsistema]` aplica apenas o que
   diverge. Contratos idempotentes por gerenciador
   (`IsAvailable`/`IsInstalled`/`Install`) garantem que rodar N vezes não
   duplica trabalho; toda escrita divergente é precedida de backup atômico.
3. **Auditar** — `envctl doctor` é **read-only**: compara a máquina real contra
   os manifestos e reporta OK/WARN/ERROR (+INFO para contexto), com a correção
   sugerida em cada linha. `--fix` existe, mas é opt-in.

A meta operacional é **0 WARN / 0 ERROR** no doctor. Um WARN nunca é ruído:
ou é drift real a corrigir, ou o manifesto precisa mudar — a decisão é
explícita, nunca silenciosa.

**Dia-0** (máquina nova): one-liner de bootstrap → perfil completo
(`run all`). **Dia-2** (já provisionada): subsistemas específicos
(`run shell`, `run skills`, …) e `doctor` como fonte da verdade do estado.

## 4. Duas dimensões, um contrato: OS × agente

- **Multi-OS:** Windows 11, Ubuntu/Debian e Arch/CachyOS, cada um com seu
  perfil (`run windows` / `run vps` / `run cachyos`) e adaptadores próprios.
- **Multi-agente:** OpenCode e CommandCode recebem o mesmo catálogo de skills e
  as mesmas regras, respeitando (e documentando) as diferenças de runtime.

A paridade é um **contrato auditado**, não uma intenção:
[os-and-agent-matrix.md](os-and-agent-matrix.md) registra o que é provisionado
em cada eixo, as assimetrias conhecidas e o status de cada uma.

## 5. Princípios (o que não se negocia)

1. **Manifesto é a fonte de verdade.** Quando doc e manifesto divergem, o
   manifesto ganha e a doc muda. O mesmo vale para o código.
2. **Idempotência estrita.** Rodar 1 ou 100 vezes produz o mesmo estado final.
   Gerenciador ausente = skip limpo, nunca falha barulhenta.
3. **Nada do usuário é sobrescrito sem regra explícita.** `merge:` ou
   `seed_if_missing`; escrita divergente gera backup atômico
   (`.bak.YYYYMMDD-HHMMSS`, podado por `keep_newest`).
4. **Auditoria antes de mutação.** Diagnóstico é read-only por padrão;
   correção automática só quando pedida (`--fix`, flags opt-in) — e itens
   sensíveis nunca são `--fix` (ficam em INFO/manual).
5. **Zero segredos no repositório.** Chaves vivem por máquina
   (`~/.config/opencode/secrets/`, ACL restrita); o repo só as referencia.
6. **Determinismo de versão.** Manifests, configs e skills são embutidos no
   binário (`//go:embed`): uma versão = um conjunto coerente, executável
   offline.
7. **Feedback em segundos, não em minutos.** O gate local (`envctl-verify`) é
   ferramenta de **invocação explícita** — desde a v2 nada é plugado
   automaticamente (hooks globais interferiam nos hooks dos outros repositórios);
   o CI espelha os mesmos checks — ver [verification.md](verification.md) e o
   [ADR 0009](adr/0009-sem-hooks-globais.md).
8. **Decisão vira registro.** Decisão estrutural → ADR; lição → memória do
   projeto; mudança → CHANGELOG. Nada de conhecimento tribal.

## 6. Arquitetura em uma passada

Clean Architecture em Go: `domain` (entidades + contratos) ← `usecase`
(orquestração de perfis/subsistemas) ← `infra` (adaptadores: winget/apt/
pacman/paru, mise/uv, filesystem/ACL, registro, git, logger) ← `ui/cli`
(Cobra + PTerm). O mapa de arquivos e o detalhamento de cada camada estão em
[architecture.md](architecture.md).

O "pacote" do produto é o **binário + o repositório**: binário determinístico
(assets embutidos) e o repo como fonte (manifests, docs, ADRs). O release
amarra versão, changelog e assets em um commit tagueado — ver
[ADR 0005](adr/0005-release-manual-com-tags-do-pipeline.md).

## 7. Como uma mudança acontece

Fluxo padrão para implementações multi-arquivo:

```
spec (spec-agent/) → branch → implementar → excluir a spec/pasta → PR → merge → release
```

- Specs são **temporárias**: guiam a implementação e são deletadas quando
  incorporadas — as decisões migram para ADRs, memória e CHANGELOG.
- Conventional commits, merge commit (sem squash), CI (lint/test em dois OS)
  e gate local espelhado.
- Releases são manuais e deliberadas; a tag pertence ao pipeline.

## 8. Segurança e dados

- **Repo público, zero PII:** inventário de servidores e chaves vivem fora
  (`~/.config/opencode/extras/`, `~/.ssh-manager/` — locais por máquina).
- **Memória em tiers:** projeto (`.opencode/memory/`, versionado) e global
  (`~/.config/opencode/memory/`, por máquina, nunca sincronizado).
- **Segredos por referência:** chaves entram por arquivo local (`{file:…}`)
  em vez de env ou literal — ver
  [ADR 0003](adr/0003-mcp-segredos-via-arquivo.md).

## 9. Evolução

O que falta está no [roadmap.md](roadmap.md) (Termux/Android, serviço em
background, dispatch remoto profundo, rename do projeto). A regra para propor
algo novo: confira o roadmap, escreva a spec — e, se a decisão for estrutural,
ela nasce como ADR.

## Mapa da documentação

| Pergunta | Documento |
|---|---|
| Por que o projeto existe / modelo | **este documento** |
| Como o código se organiza | [architecture.md](architecture.md) |
| Diretrizes de engenharia | [principles.md](principles.md) |
| Decisões estruturais | [adr/](adr/) |
| O que é provisionado por OS × agente | [os-and-agent-matrix.md](os-and-agent-matrix.md) |
| Doctor, idempotência e backups | [doctor-and-idempotency.md](doctor-and-idempotency.md) |
| Gate local de verificação | [verification.md](verification.md) |
| Estrutura dos manifestos | [manifests.md](manifests.md) |
| Catálogo de skills | [skills.md](skills.md) |
| Instalação e uso (novatos) | [guides/getting-started.md](guides/getting-started.md) |
| Futuro acordado | [roadmap.md](roadmap.md) |
