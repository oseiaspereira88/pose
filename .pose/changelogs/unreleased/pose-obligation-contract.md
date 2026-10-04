---
spec: pose-obligation-contract
category: added
breaking: false
refs:
---

POSE defines a versioned obligation read model (`schemas/v1/obligation.schema.json`): one shape for anything still owed, projected from the subsystem that owns it, with a stable id derived from its logical source, project-qualified node references (`xref:<project>/spec:<slug>#requirement:R4`), and separate fields for satisfaction, knowledge, waiting and per-phase effects. Unknown is never a resolution, and a blocker known only as text keeps its origin without an invented actor or target. The documentation page "Obligations and readiness" describes the model.
