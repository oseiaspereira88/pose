---
spec: pose-git-repository-detection-ignores-locale
category: fixed
breaking: false
---

Recognise a directory outside every Git repository under a translated git. Release readers no longer turn "no repository" into a git failure in non-English sessions, so the release phase of a project without releases is not unknown and an archived fragment in a project without Git can be intact.
