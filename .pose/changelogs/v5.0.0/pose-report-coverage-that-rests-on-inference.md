---
spec: pose-report-coverage-that-rests-on-inference
category: added
breaking: false
refs:
---

A delivery target whose only passing evidence for a class comes from a module
containing it now carries an `inferred-coverage` finding. Nothing is refused.

Evidence answers downward because a module-wide run does exercise its subtree.
Usually — a run configured to skip a directory inside it does not, and nothing
in the result says which. That is the one part of the containment rule the engine
cannot check.

Closing it properly means a check declaring what it walked: a change to the
validation result contract and a migration for every project. Making the
inference visible costs nothing, refuses nothing, and produces what that decision
needs — how many targets in real repositories are gated on a run of something
larger than themselves. In POSE's own the answer is none, so the contract would
have been designed against no case at all.

The finding fires only when no result names the target's own module: a target
with its own check is not resting on the inference, whatever else also matched.
