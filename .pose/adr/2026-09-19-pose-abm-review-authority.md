---
title: Autoridade verificável separada da independência declarada
status: Proposed
date: 2026-09-19
---

# Context

O review plan já expressa `same-actor-separate-execution`, `different-actor` e
`mandatory-human`, mas um attestation local consegue satisfazer esses valores
com uma string que o próprio reviewer escolheu. Isso é legível como declaração,
não como prova de identidade ou autoridade.

# Decision

Manter independência e assurance como eixos distintos. Bundles novos podem selar
`identity_assurance: verified`; nesse modo a attestation precisa carregar uma
`ReviewAuthorityClaim` assinada pelo envelope Ed25519 já confiado pela policy,
vinculada ao digest do bundle, audience do projeto, principal, papel e executions.
Afirmações de papel humano exigem um grant separado de issuer. Bundles antigos e
policy `declared` conservam o comportamento histórico, explicitamente sem
prometer autoridade verificada.

# Consequences

O POSE continua offline e não precisa conhecer pessoas ou modelos. A integração
que emite a claim passa a ser responsável por identidade, proteção da policy e
rotação de pins. Claims podem ser rejeitadas por replay, expiração, audience,
principal divergente ou grant ausente. A solução não prova independência
cognitiva e não transforma um issuer autorizado em fonte infalível.

# Alternatives rejected

- Tratar o prefixo como identidade: barato, mas reproduz exatamente a brecha.
- Exigir Harne8 para todo review local: acopla disponibilidade e viola o modo
  offline do POSE.
- Exigir diversidade de modelo: não há propriedade verificável de pensamento
  independente.
