---
spec: pose-machinery-distribution-contract
category: changed
breaking: false
refs:
---

`pose upgrade` now delivers rules, workflows, templates and skills, not just the
manuals. Until now they only reached an instance through `pose upgrade --force`,
so engine improvements never arrived and instances drifted indefinitely. The
contract is per file: unchanged files are skipped, a file you edited is backed
up as `<file>.pose-backup` before being refreshed, and a file you deleted stays
deleted. Expect a burst of `.pose-backup` files on the first upgrade if you have
customized shipped machinery — nothing is lost. One exception applies once: on
an instance created before this release, a machinery file you had deleted comes
back on the first upgrade, because the engine has no record of it having been
delivered; from the second upgrade on, deletions are respected.

`pose skills-check` now also validates the `locales/*/.agents/skills` mirrors,
which immediately caught a translated skill linking a file that does not exist.
