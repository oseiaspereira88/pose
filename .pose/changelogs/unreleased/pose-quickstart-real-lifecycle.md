---
spec: pose-quickstart-real-lifecycle
category: fixed
breaking: false
refs:
---

`pose close` no longer marks a spec done with an incomplete requirement trace: the closeout plan stops at a new `trace` step before anything is sealed, and the transition refuses for the same reason, naming each requirement and how to declare it. When the plan stops for a reviewer it prints the sealed evidence refs and an attest command filled with them (`closeout_plan.attest`). The spec template's example delivery target is no longer a declared target that blocked closeout, and an available stack rule extension is a `next` step in `pose doctor`. The quickstart now walks the real lifecycle in five chapters — setup, the entry gate, `pose start`, a decision answered with a signature, and review and close — each executed in CI as written.
