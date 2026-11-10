---
spec: pose-commit-gate-refuses-index-without-descriptor
category: added
breaking: false
---
`pose check` reporta um conjunto de índices que o descritor da geração não descreve (geração indisponível, ou arquivos que diferem da geração) como erro no modo estrito e aviso no tolerante, nomeando `pose index`, em vez de deixar a causa aparecer só como "input closure is unknown" em cada review closeout.
