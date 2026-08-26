---
name: phone-e164-normalization
description: Normalizar números de telefone BR para formato E.164 (com DDI 55). Use quando precisar padronizar telefones antes de inserir no banco, comparar duplicados ou enviar para APIs de mensageria/SMS/WhatsApp. Triggers: telefone, phone, e164, normalizar número, ddd, ddi, +55, whatsapp, sms.
license: MIT
compatibility: opencode
---

# Normalização E.164 de Telefones BR

## Quando usar

Sempre que dados de telefone entrarem no sistema (cadastro, import, webhook) sem formato garantido. Evita duplicados (mesmo número em formatos diferentes) e rejeições de APIs de mensageria.

## Passos

1. Remover tudo que não for dígito (espaços, `(`, `)`, `-`, `.`, `+`).
2. Remover `0` inicial (DDD não tem zero à esquerda).
3. Classificar pelo tamanho:
   - **10 dígitos** (`DDD + 8 dígitos`): prefixar `55` → `55DDDnumero`.
   - **11 dígitos** (`DDD + 9 dígitos`): prefixar `55` → `55DDDnumero`.
   - **12-13 dígitos já com `55`** (`55 + DDD + número`): manter.
   - **13 dígitos `55 + 0 + DDD + número`**: remover o 0 (só se `startsWith('55')`).
4. **Caso ambíguo — DDD `55` (RS) vs país `55`**: resolver pelo tamanho — se 10-11 dígitos após remover `55`, é DDD do RS (prefixar `55` de novo); se 12-13, é DDI.
5. Descarte números malformados (resto) — **registrar descarte**, não inserir em silêncio.

## Verificação

- Saída sempre no formato `55DDDNNNNNNNNN` (13 dígitos) para números válidos.
- Tabela de teste: `01123456789` → `551123456789`; `11987654321` → `5511987654321`; `5551987654321` → manter; `55119987654321` → manter; `0` sozinho → descartar.
- Contar descartados no log — malformados não entram no banco sem registro.