---
spec: pose-delegated-review-brief
category: added
breaking: false
refs:
---

`pose review brief <bundle|scope> [--kind review|adjudication|smoke]` renders the request a second agent receives to review, adjudicate or smoke-test a scope, generated from its sealed bundle alone: the change, the criteria and tools the plan owes, the structural facts and the sealed evidence, with fixed boundaries. The implementer no longer writes the reviewer's prompt; notes passed with `--note-file` appear only in a section labelled as the implementer's.
