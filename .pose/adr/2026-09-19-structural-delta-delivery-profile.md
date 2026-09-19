---
title: Structural delta usa profile de delivery dedicado
status: accepted
date: 2026-09-19
---

# Structural delta usa profile de delivery dedicado

## Contexto

O producer `assess-design` já observa `structure` no subject selado de review,
mas o target local ainda usava `backend-go`. Isso tornava a classificação de
delivery indistinguível de uma governança Go genérica e deixava ambíguo se uma
validação comum deveria fabricar evidência estrutural.

## Decisão

Registrar `structural-delta` como profile de delivery `governance`, exigindo
`integration`, e apontar `governance:structural-delta` para ele.

O profile dedicado não exige `structure` no arquivo de resultados da matriz:
essa classe é derivada pelo review bundle a partir do subject canônico, não por
um check de build que não conhece spec, change set ou subject. A integração
prova que o producer é alcançável; o bundle prova a observação estrutural.

## Alternativas consideradas

- Manter `backend-go`: rejeitada por perder a semântica explícita do target.
- Exigir `structure` em `pose validate`: rejeitada porque o validator não deve
  inventar um subject de review nem duplicar o producer.
- Criar um score de complexidade: rejeitada; julgamento proporcional continua
  pertencendo ao review.

## Consequências

- O target fica semanticamente identificável e pode evoluir sem alterar o
  profile Go genérico.
- `integration` continua sendo uma prova de wiring, não de proporcionalidade.
- A composição Harne8 deve validar separadamente que o gitlink, o profile e o
  producer publicados permanecem alcançáveis; este ADR não declara essa
  composição concluída.
