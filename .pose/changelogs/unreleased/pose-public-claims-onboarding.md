---
spec: pose-public-claims-onboarding
category: added
breaking: false
refs:
---

`.pose/templates/public-claims.json` now ships as machinery, so an instance
receives the shape of the public claims contract instead of having to
reverse-engineer it from this repository's own `claims.json`.

`pose public-claims` run without a contract explains that the instance declares
none, that the gate is opt-in, and how to start one from the template, with a command that creates its own destination — `.pose/public` exists in no scaffold, so the copy alone would have failed on the very instances the message is written for. It
previously reported `open .pose/public/claims.json: no such file or directory`
and still exits non-zero, so a pipeline never reads an unrun gate as a passing
one.
