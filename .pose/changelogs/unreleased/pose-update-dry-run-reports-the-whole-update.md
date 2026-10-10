---
spec: pose-update-dry-run-reports-the-whole-update
category: fixed
breaking: false
refs:
---

`pose update --dry-run` now shows the whole update: the files it would create, modify or remove, the engine version it would stamp and the configuration review it would open, by running the update on a disposable copy. It used to print only schema migrations, so an instance could look current while still stamped by an older engine. `pose doctor` and `pose check` now warn when the instance was last updated by an older engine.
