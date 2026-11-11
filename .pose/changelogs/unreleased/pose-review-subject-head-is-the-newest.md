---
spec: pose-review-subject-head-is-the-newest
category: fixed
breaking: false
---
A cabeça do sujeito de uma revisão com vários change sets (milestones, roadmaps) é a mais nova das cabeças dos sets, e não a do último por id; o worktree selado da revisão delegada passa a conter o que todos os membros fizeram.
