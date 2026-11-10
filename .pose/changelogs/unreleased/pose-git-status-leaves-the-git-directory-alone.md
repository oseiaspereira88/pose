---
spec: pose-git-status-leaves-the-git-directory-alone
category: fixed
breaking: false
---
O `git status` do snapshot de integridade deixa de reescrever o índice sob `.git` (`GIT_OPTIONAL_LOCKS=0`); em instâncias com submódulos, os closeouts de review deixam de sair com "review inputs changed during operation: .git" por causa da própria pergunta ao Git.
