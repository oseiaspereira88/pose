---
spec: pose-skill-command-parity
category: added
breaking: false
refs:
---

A contract test now fails when an English skill and its translation teach
different POSE commands, comparing only the commands each side tells an agent to
run — flags, wording and format are deliberately out of scope, because English
skills are terse and translations example-rich by design. Applying it settled
that open question with evidence: the terse rewrite had dropped nine commands
the translations still taught, including `pose new-spec` from the feature skill
and `pose validate` from the closeout skill. They are restored.
