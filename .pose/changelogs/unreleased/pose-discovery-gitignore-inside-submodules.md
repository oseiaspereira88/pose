---
spec: pose-discovery-gitignore-inside-submodules
category: fixed
breaking: false
refs:
---

Discovery honours a submodule's own `.gitignore`. The ignored-path lookup ran
`git ls-files --ignored` only at the repository root, and git does not descend
into submodules there, while the walkers do. A cache a submodule ignores reached
`pose index` and `pose validate` as tracked content: on an adopting repository a
pytest cache inside the POSE submodule entered `repo-map.json` on one machine
and not in a clean clone, and a manifest in such a directory would have become a
governed module. Each initialised submodule is now asked for its own ignored
paths; an uninitialised one is skipped.
