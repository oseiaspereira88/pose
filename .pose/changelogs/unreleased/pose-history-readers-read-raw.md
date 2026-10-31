---
spec: pose-history-readers-read-raw
category: changed
breaking: false
---
Structural review reads raw object history, as the delivery graph does, so a replace ref or graft cannot make it judge another history; a repository with either should rerun `pose index` and reseal. A trailer change set that starts at the repository's first commit now resolves, so a new project's first spec has a change set.
