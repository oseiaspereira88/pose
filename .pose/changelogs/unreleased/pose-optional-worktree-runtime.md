---
spec: pose-optional-worktree-runtime
category: added
breaking: false
---
`pose runtime serve|start|attach|status|stop|restart`: a runtime opcional por worktree, identificado pelo usuário, pelo worktree canônico e pelo diretório comum do Git, num socket Unix protegido, com handshake que recusa outro engine, ciclo de vida drenado e fallback standalone. É opt-in e nesta fatia não hospeda operações do domínio; no Windows os comandos rodam sempre standalone.
