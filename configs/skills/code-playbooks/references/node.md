# Node / TypeScript

## Antes de escrever

- `package.json`: versão de Node, tipo de módulo (`"type": "module"`), scripts.
- Qual gerenciador: `pnpm`, `bun`, `npm`, `yarn`? Use o que o lockfile indica.
- `tsconfig.json`: `strict`, `target`, `moduleResolution` — o build usa o que está lá.

## Convenções

- `tsc --noEmit` é o gate de tipos; `eslint` só nos arquivos alterados se o projeto
  assim definir; `prettier` formata.
- ESM: import explícito com extensão onde o projeto exige; não misture `require`.
- Dependência nova precisa justificar: bundle, suporte da versão de Node, manutenção.
- `node:` prefix no core modules; `fetch` nativo em runtime moderno.
- Ambiente: `.env` via loader do projeto; segredo nunca no bundle do cliente.

## Armadilhas

- `npm`/`bun` installando por accident sem lockfile atualizado → CI diverge.
- `any` para calar o compilador: use `unknown` + narrow.
- Client component tocando `window`/`localStorage` no servidor.
