---
spec: pose-governance-audit-skips-pure-git-reads
category: fixed
breaking: false
---
A auditoria final de uma operação de leitura governada não reexecuta mais os comandos do Git cuja saída é função pura de ids de objeto ou do commit do HEAD, enquanto esse commit se mantém; o `review-check` deste repositório passou de cerca de 52 s para cerca de 25 s, com a mesma saída e o mesmo veredito da auditoria.
