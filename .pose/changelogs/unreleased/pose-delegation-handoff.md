---
spec: pose-delegation-handoff
category: added
breaking: false
---

Add `pose delegate handoff <run> --to <agent|auto>`: a run that stopped interrupted or failed continues with another agent under the same attempt and the continuation budget of the agent that opened it, with a brief that quotes the stopped run's partial final message and the commands it ran, each with its digest. Only the handoff run concludes; its record names `hands_off_from` and binds the section's digest, and a signed proof verifies the section against the predecessor. A foreground run that stops in a terminal offers waiting, handing off or cancelling; its result, `ls --json` and `wait --json` carry `next_actions` judged by the planning each command runs; and `watch` marks each handoff.
