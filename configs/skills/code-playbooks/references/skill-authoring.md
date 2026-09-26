# Autoria de skill

## Quando criar uma skill (e quando não)

Crie se: o processo é **reutilizável**, tem **passos** e você já o executou mais de uma vez.
Não crie se: é conhecimento de uma ferramenta que o modelo já tem, é receita de **um**
projeto, ou é uma frase que cabe numa linha do `AGENTS.md`.

Teste: "eu faria isso sozinho, sem aviso?" Se sim, não é skill.

## Escrever

- `name`: kebab-case, verbo no gerúndio quando descreve ação (`processing-pdfs`),
  específico. Evite `helper`, `utils`, `tools`.
- `name` **precisa** ser igual ao diretório (os loaders validam isso e pulam a skill).
- `description`: o que faz **e quando usar**, em 3ª pessoa, imperativo quando couber
  ("Use quando..."), keyword-rich e curto. É o único sinal de ativação que o modelo tem.
- Corpo: regras que **não** são óbvias, o comando real, e o que **não** fazer.
- Corpo curto (objetivo: < 500 linhas); detalhe vira `references/`.

## Adaptar uma skill que não é sua

- Confira se o **ambiente** bate: comandos, paths, gerenciador, runtime.
- Traga só o que é útil aqui; o resto é ruído de contexto.
- Trigger errado é a causa nº 1 de skill que nunca carrega — ajuste a description.

## Minerar

Minere sessões e transcrições quando sentir repetição: mesmo erro, mesma sequência, mesma
pergunta. O workflow que se repetiu e nunca foi escrito é o candidato.

## Generalizar para publicar

Antes de publicar (GitHub, marketplace, time): remova caminho privado, nome de host
interno, credencial, identificador de cliente e hábito pessoal. Documente a licença e a
atribuição do original. O que você só usa localmente **não vai para o git**.
