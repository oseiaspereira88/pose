---
spec: project-id-from-any-directory-name
category: fixed
breaking: false
---

`pose install` and `pose index` work again in a repository whose directory name is not a slug, such as `MyApp` or `Acme Portal`. On 6.0.0 the id derived from the directory name was refused and both commands failed with `invalid-project-configuration`. The derived id is now folded into a slug (`proj.myapp`); a name that was already valid keeps its id. A declared id is never rewritten: an invalid one is refused with the value to declare instead, and `pose doctor --fix --yes` replaces one an earlier engine stamped into `.mcp.json`.
