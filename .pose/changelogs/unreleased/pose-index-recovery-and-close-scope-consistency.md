---
spec: pose-index-recovery-and-close-scope-consistency
category: fixed
breaking: false
---
`pose index` repara um conjunto de índices cuja geração publicada está indisponível (o comando que os leitores recomendam passa a funcionar); a varredura de módulos não percorre `.pose` (o registro de governança que cresce a cada fechamento já podia estourar o limite de 10 000 entradas) e o erro do limite diz o que foi contado; e `pose close` de um milestone ou roadmap com `--plan`, `--apply` ou `--resume` sai com 2 dizendo que o escopo fecha com `pose close <escopo>` sem modo, em vez de sugerir a invocação recusada.
