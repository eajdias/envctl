# Atribuição de Skills do envctl

**Não existe obrigação de fidelidade 1:1 com nenhum upstream.** Toda skill deste repositório é sua para adaptar, reescrever e ajustar às suas preferências, ao seu estilo de programação e ao que fizer sentido para o seu ambiente — independentemente de onde a ideia original veio. O que este documento registra é apenas **de onde a ideia veio** (para dar o crédito devido) e **o que foi adaptado** em cada cópia. Nenhuma skill aqui está presa a "espelhar" o autor original: se amanhã você quiser reescrever uma derivada do zero, pode.

O crédito ao upstream (declaro no `metadata` de cada `SKILL.md`) já cumpre o papel de reconhecer quem trouxe a ideia; o resto é seu.

```yaml
# formato usado no frontmatter das que vieram de terceiro
license: MIT
metadata:
  author: <autor/handle do upstream>
  source: https://github.com/<owner>/<repo>
  adapted: <o que foi mudado — ver a tabela>
```

---

## 1. Com upstream conhecido (ideia veio daqui)

| Upstream | Licença | Skills | Como foi adaptado (medido) |
|---|---|---|---|
| [obra/superpowers](https://github.com/obra/superpowers) | MIT | `writing-plans` | Ideia e estrutura originalmente preservadas; o corpo foi adaptado ao fluxo local, com riscos, unknowns, rollback, Definition of Done e `spec-agent/`. |

Licenças verificadas via API do GitHub em 2026-09-16 (`license.spdx_id = MIT`).

## 2. Sem upstream confirmado (suas)

Nenhuma skill atual se enquadra aqui — as 12 do catálogo derivam do upstream acima ou de autoria própria já documentada em `docs/skills.md`.

## 3. Licença

Repositório com `LICENSE` (MIT) na raiz. Todas as skills declaram `license: MIT` no frontmatter. Como todos os upstreams são MIT, o conjunto é compatível — você pode adaptar qualquer uma livremente.

## 4. Ao adotar skill de terceiro daqui pra frente

1. Verifique a licença do upstream (`gh api repos/<owner>/<repo> --jq .license.spdx_id`). Prefira MIT/Apache-2.0/BSD.
2. Adapte o corpo como quiser — fidelidade ao upstream não é necessária.
3. Registre na tabela da seção 1.
4. Rode `envctl run skills`; confirme no `cmdc skills list`.
