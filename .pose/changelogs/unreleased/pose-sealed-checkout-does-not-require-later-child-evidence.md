---
spec: pose-sealed-checkout-does-not-require-later-child-evidence
category: fixed
breaking: false
---
A revisão delegada de um roadmap (ou de um milestone com filhos) deixa de falhar com "sealed checkout blocked: child … has no sealed review bundle": o checkout selado não contém a evidência de review dos filhos, que é commitada depois do último commit do sujeito, e o selamento e o fechamento já a julgam.
