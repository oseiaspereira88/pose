---
spec: pose-dora-five-metrics-v2
category: changed
breaking: true
refs: []
---

`pose dora-metrics` now reports the current five production delivery metrics,
replacing the former Reliability proxy with deployment rework rate and limiting
recovery time to resolved incidents explicitly caused by a deployment. New
deployment and incident events require rework/environment classification;
schema-v1 JSONL remains readable without guessed values.
