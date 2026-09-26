# Frontend

## HTML semântico

- Elemento pela **função**, não pela aparência: `button` para ação, `a` para navegação,
  `label` ligado a input, `table` para dado tabular.
- Landmarks (`header`, `nav`, `main`, `footer`) e hierarquia de heading sem pular nível.
- `alt` descritivo ou `alt=""` em decorativo — nunca alt com nome de arquivo.

## Acessibilidade (WCAG)

- Contraste 4.5:1 em texto, 3:1 em texto grande e em borda de componente interativo.
- Foco visível e ordem de tabulação igual à ordem visual; nada de `outline: none` sem
  substituto.
- Todo input tem label; erro associado por `aria-describedby` + `aria-invalid`.
- Alvo de toque mínimo 24×24 px; não dependa de hover.
- `aria-*` só quando o HTML semântico não resolve; ARIA errada piora.

## CSS

- Custom properties para token de cor/espaço; nada de valor mágico repetido.
- Mobile-first; breakpoints por **conteúdo**, não por dispositivo.
- `prefers-reduced-motion` respeitado; nada de animação que bloqueie interação.
- Estado de foco e de erro **visuais**, não só semânticos.
