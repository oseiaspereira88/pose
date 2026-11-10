---
spec: pose-worktree-runtime-resident-cache
category: added
breaking: false
---
`pose runtime run [--envelope] [--timeout <d>] <command>` roda `review-check`, `closeout-check`, `artifact-check`, `surface-check` ou `roadmap-check` no runtime do worktree, com resultado tipado e fallback standalone, e o runtime guarda entre comandos as leituras do Git que são funções puras de um commit; o segundo `review-check` deste repositório levou cerca de 10 s contra cerca de 25 s standalone, com a mesma saída.
