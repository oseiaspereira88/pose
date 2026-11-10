---
spec: pose-worktree-runtime-qos-and-measurement
category: added
breaking: false
---
O runtime opcional admite os comandos hospedados em duas faixas (`short` e `heavy`) com slots e fila limitada configuráveis (`--short-slots`, `--heavy-slots`, `--short-queue`, `--heavy-queue`, `--memo-mib`), de modo que uma consulta curta não espera um check pesado; o status traz as métricas por faixa; e há uma medição opt-in do runtime num repositório real, com limiares fixados antes dos números.
