---
spec: pose-tool-dispositions-must-be-supported
category: fixed
breaking: true
refs:
---

A tool disposition recorded `passed` must now cite evidence the sealed bundle
contains, for tools that declare evidence classes. The class was already
checked; presence was not, so a disposition could name a class the tool asks for
and an id that appears nowhere.

`pose review auto-attest` no longer synthesises tool evidence either — no
`validation:auto-attest`, no `<class>:auto-attest`. Where the scope is expected
to carry validation evidence it refuses, naming the tool and the missing class;
where it is not, it defers the tool with a rationale. A tool that declares no
evidence class, such as `artifact-check`, cites its own run rather than an
unrelated sealed result.

A tool gated on `delivery-target-declared`, in a scope that has none, is now
deferrable rather than required to have passed. Without that, removing the
fabrication would make a documentation-only closeout impossible: the engine
would demand a disposition only a fabricated one could satisfy.
