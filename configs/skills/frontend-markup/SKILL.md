---
name: frontend-markup
description: >-
  Criar e revisar HTML, CSS e SCSS com marcação semântica, interface acessível e styling responsivo. Use em páginas, componentes, formulários, tabelas, sistemas de design ou correções visuais que exigem WCAG, a11y, ARIA, accessibility, navegação por teclado, foco, contraste, zoom ou performance web. Triggers: HTML, CSS, SCSS, semantic markup, responsive, WCAG, a11y, ARIA, accessibility, styling.
license: MIT
---

# HTML e CSS semânticos e acessíveis

## Princípios

1. Comece pela semântica nativa do HTML.
2. Preserve a ordem do documento, a navegação por teclado e o foco.
3. Faça a menor mudança acessível que resolva o problema.
4. Siga os frameworks, tokens e convenções já existentes no projeto.
5. Meça performance e comportamento; não presuma ganhos.

## HTML semântico

- Escolha o elemento pela função: `header`, `nav`, `main`, `aside`, `footer`, `article`, `section`, listas, títulos e controles nativos.
- Use `div` e `span` quando não agregarem semântica; estilo não justifica substituir um elemento nativo melhor.
- Mantenha uma estrutura coerente de títulos: um `h1` principal por documento e descendentes sem saltos sem motivo.
- Use títulos para documentar a estrutura, não para escolher tamanho ou cor. Nomeie agrupamentos não semânticos somente quando necessário.
- Links devem navegar e botões devem executar ações. Não simule um com o outro apenas por conveniência visual.
- Evite landmarks duplicados, listas usadas só para layout e elementos interativos aninhados.

## Controles, conteúdo e dados

- Associe todo campo a um `label`; use `for`/`id` ou aninhamento, conforme o padrão existente.
- Informe ajuda, formato, obrigatoriedade e erros no próprio texto, vinculados ao campo quando necessário.
- Escolha `type`, `inputmode` e `autocomplete` adequados; não desative comportamentos nativos para efeitos cosméticos.
- Dê `alt` que transmita a finalidade da imagem. Use `alt=""` somente quando ela for realmente decorativa.
- Não use placeholder como rótulo nem omita informação essencial disponível apenas no tooltip.
- Use `caption` e `th` com `scope` em tabelas de dados; não use tabelas para layout.
- Preserve o botão nativo, os campos nativos e o comportamento do navegador sempre que atendam ao requisito.

## ARIA e interação por teclado

- ARIA é o último recurso: elemento nativo primeiro, ARIA apenas quando a semântica nativa for insuficiente.
- Nunca acrescente `role` redundante, como `role="button"` em `<button>`, nem anuncie a mesma informação por texto e `aria-label` sem necessidade.
- Um widget customizado precisa de nome acessível, papel correto, estados, atualização coerente e operação completa por teclado.
- Não remova controles nativos da ordem de tabulação com `tabindex="-1"`, exceto quando um padrão de widget real exigir isso.
- Não use `tabindex` positivo. A ordem do DOM deve refletir a ordem visual e de leitura.
- Mantenha o foco visível com `:focus-visible`; não use `outline: none` sem substituto claramente perceptível.
- Trate `Enter` e `Space` em controles customizados que exigem essas teclas, sem duplicar a ação do elemento nativo.
- Confirme foco, retorno de foco e contexto ao abrir, fechar ou navegar por menus, diálogos e conteúdo assíncrono.

## CSS e SCSS

- Use Grid para relações bidimensionais, Flexbox para distribuição unidimensional e layout nativo para o restante; escolha pela estrutura, não por dogmas.
- Prefira dimensões e faixas intrínsecas, `minmax()`, `auto-fit` e dimensionamento por conteúdo a larguras rígidas.
- Reaja ao espaço disponível e ao conteúdo com media queries ou container queries quando isso servir; não imponha mobile-first ou BEM se a base usa outra convenção.
- Reutilize tokens e custom properties existentes. Para código novo, prefira propriedades lógicas como `margin-inline`, `padding-block` e `inset-inline`.
- Evite seletores profundos, `!important` e especificidade frágil. Consuma os estilos do componente em vez de sobrescrevê-los cegamente.
- Teste contraste de texto, ícones, foco e estados que não devam depender apenas de cor.
- Respeite `prefers-reduced-motion`; reduza, substitua ou remova deslocamentos e transições não essenciais.
- Mantenha alvos de clique e de toque confortáveis e espaçados sem impor dimensões que prejudiquem zoom ou conteúdo.

## Resiliência do layout

- Não bloqueie zoom nem fixe largura ou altura que impeçam conteúdo, texto e controles de crescer.
- Prefira componentes que reflowem sem cortes horizontais, truncamento ou sobreposição.
- Não esconda ações essenciais apenas em `hover`; ofereça equivalentes por foco e toque.
- Respeite orientación, espaço disponível e textos traduzidos, não apenas larguras dos dispositivos.
- Teste redimensionamento, zoom e reflow nos limites reais do produto, não só em breakpoints previstos.

## Estabilidade e performance

- Reserve espaço para imagens, vídeo, banners e fontes com dimensões/aspect ratio ou mecanismo equivalente confiável.
- Evite conteúdo injetado que empurre a página e animações JavaScript desnecessárias; use CSS para transições simples.
- Mantenha efeitos de animação restritos a propriedades de baixo custo, como `transform` e `opacity`, quando isso preservar a aparência.
- Meça Core Web Vitals, `PerformanceObserver`, profiler e traces quando a mudança puder afetar renderização ou interação.
- Compare antes/depois em condições equivalentes; não declare performance melhor sem dados.

## Workflow

1. Inspecione o framework, scripts, componentes, estilos globais, tokens e padrões de teste já configurados.
2. Confirme navegadores e dispositivos alvo, nível de suporte e requisitos de acessibilidade do produto.
3. Defina o menor resultado observável: semântica, teclado, foco, estados, reflow e comportamento esperado.
4. Use `test-driven-development` quando houver mudança de comportamento; escreva o teste antes do código.
5. Implemente markup e estilos mínimos. Para confirmar APIs atuais de componentes, consulte `context7-auto` e, quando aplicável, MDN Web Docs.
6. Execute os comandos já configurados de lint, format, typecheck e testes. Use `universal-test-runner` para suítes, regressão e cobertura; não invente comandos paralelos.
7. Se o fluxo passa por build/runtime standalone de Next.js, carregue `nextjs-standalone-deploy`; não duplique o procedimento aqui.
8. Rode a interface real e verifique viewports representativos, além de zoom/reflow e orientação quando forem requisitos.
9. Com browser ou Playwright disponível, navegue somente por teclado, use Tab/Shift+Tab, Enter, Space, setas e Escape, e verifique foco visível e nomes anunciados.
10. Confira overflow, contraste, movimento reduzido, layout shift, console/rede e performance relevante.
11. Corrija falhas encontrados e repita os comandos afetados antes de concluir.

## Checklist final

- [ ] A estrutura nativa representa o documento e as ações.
- [ ] Títulos, landmarks, rótulos, textos alternativos, tabelas e mensagens de erro fazem sentido.
- [ ] Não há ARIA redundante nem estado customizado incompleto.
- [ ] Todo fluxo principal funciona por teclado, com foco previsível e visível.
- [ ] Layout e texto funcionam nos viewports, zoom e reflow exigidos.
- [ ] Contraste, alvos e movimento reduzido respeitam WCAG/a11y do produto.
- [ ] Não há shift evitável, animação JS desnecessária ou regressão de performance.
- [ ] Lint, format, typecheck, testes e validação visual foram executados com evidência.
- [ ] APIs novas foram confirmadas em documentação atual e ficam compatíveis com os alvos.
- [ ] A mudança permaneceu mínima e preservou convenções e componentes existentes.
