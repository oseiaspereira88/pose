---
spec: pose-index-input-binding-reads-each-transfer-once
category: fixed
breaking: false
---
`review-check`, `closeout-check` e `artifact-check` leem os planos de transferência de um projeto uma vez por operação, não uma vez por artefato resolvido, e as lojas de origem do acceptance federado usam o snapshot de specs da operação. A saída não muda; o `review-check` deste repositório passou de cerca de 56 s para cerca de 38 s.
