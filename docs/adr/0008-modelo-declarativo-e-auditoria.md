# ADR 0008: Modelo declarativo e auditoria read-only

## Status
Aceito (retroativo — registra o modelo que o código pratica desde o início,
como o ADR 0004 fez para a migração V2).

## Contexto
O projeto nasceu de scripts imperativos ad-hoc ("instala isso, ajusta aquilo").
Esse formato tem dois defeitos estruturais:

- **Não audita**: depois de rodar, ninguém sabe se a máquina ainda corresponde
  ao que se queria — o script não guarda a intenção.
- **Não replica**: ambientes divergem entre si sem que ninguém perceba, e a
  "receita" só existe no histórico de execução.

## Decisão
O envctl opera em um modelo **declarativo com auditoria contínua**:

1. **Manifestos são a fonte de verdade.** `manifests/*.yaml` descrevem o
   estado desejado por OS/distro (pacotes, toolchains, configs, tweaks,
   skills, performance), incluindo filtros (`os:`, `target_distro`,
   `min_distro_version`) para que uma entrada só se aplique onde faz sentido.
   Os manifestos são embutidos no binário (determinismo de versão).
2. **Convergência idempotente por contratos.** Todo gerenciador implementa o
   mesmo contrato — `IsAvailable` (existe neste host?), `IsInstalled` (já está
   no estado desejado?), `Install` (converge) — e o uso caso a caso
   (`provision_*`) filtra pelo OS corrente. Rodar 1 ou 100 vezes = mesmo
   estado; gerenciador ausente = skip limpo.
3. **`doctor` é read-only e é o contrato de auditoria.** Ele compara a máquina
   real contra os manifestos e reporta OK/WARN/ERROR (+INFO para contexto),
   com correção sugerida em cada linha. `--fix` e demais mutações são opt-in;
   itens sensíveis (kernel cmdline, serviços de terceiros, decisões de
   segurança) nunca são `--fix` — viram INFO/manual. Meta operacional:
   **0 WARN / 0 ERROR**.
4. **Docs subordinadas à fonte.** Quando doc e manifesto/código divergem, o
   manifesto/código ganha e a doc é corrigida.

## Consequências
- Drift é sempre **visível**: ou aparece como WARN no doctor, ou o manifesto
  está desatualizado — nos dois casos a decisão é explícita.
- Máquinas convergem sozinhas a partir do mesmo binário; não há servidor
  central nem estado remoto.
- O custo é disciplina de manifesto: tudo que o provisionamento promete precisa
  ter representação declarada + check de auditoria correspondente.
