---
title: Structural delta permanece observação bounded
status: accepted
date: 2026-09-19
decision_type: architecture
---

# Structural delta permanece observação bounded

## Contexto

O programa Anti-Bad-Mechanization precisa tornar a complexidade estrutural
durável visível, mas um contador de classes, LOC ou dependências não consegue
decidir se uma solução é proporcional. Um scanner amplo também aumentaria o
risco de executar conteúdo não confiável, revelar source ou transformar lacuna
de cobertura em aprovação implícita.

## Opções consideradas

1. **Complexity score**: combinar arquivos, dependências e abstrações em um
   número. Rejeitada por falso julgamento e incentivo a Goodhart.
2. **Juiz semântico no engine**: pedir a um LLM que pontue arquitetura.
   Rejeitada por não-determinismo, custo, privacidade e autoridade indevida.
3. **Structural delta bounded sobre o subject selado**: observar apenas
   mudanças duráveis suportadas por parsers locais, com IDs/digests e estados
   de coverage explícitos. Escolhida.

## Decisão

`DesignDeltaReport` compara o subject canônico do review e só afirma fatos
observáveis: dependências Go/npm, manifests de componente, metadata de delivery,
contratos de governança e ações Git (add/remove/change/rename/submodule). Cada
detector é bounded e pode responder `unknown`, `unsupported` ou
`not-applicable`. O engine não cria vínculo causal nem verdict de qualidade;
essa conclusão pertence ao profile/julgamento de review.

## Consequências

- A saída é reconstruível, cacheável pelo digest e não grava estado novo.
- A evidência derivada do bundle usa `SubjectDigest` baseado na identidade de
  implementação (patch/tree/entries), e não apenas no ref Git, para que
  movimento de provider ref ou closeout derivado não stale uma observação do
  mesmo subject.
- Um structural delta sem Decision/Requirement poderá virar finding em um
  profile futuro, mas a projeção sozinha não acusa overengineering.
- Novos ecossistemas exigem parser, fixtures negativos e evidência de cobertura
  antes de deixar de ser `unsupported`.
- O subject do bundle evita TOCTOU sem criar um grafo cíclico de bundle que
  contém a própria avaliação.

## Invariantes

- Nenhum caminho absoluto ou conteúdo integral de manifesto entra no relatório.
- Nenhum comando shell, package manager, rede ou código do repositório é
  executado.
- O cache key inclui subject, parser version, detector inputs e limits.
- `input_digest` não inclui change-set IDs ou refs provider-only; a identidade
  semântica fica ancorada no implementation digest já selado pelo bundle.
- A classe `structure` só pode ser usada depois que CLI, MCP, review bundle,
  catálogo e verificadores a reconhecem.
