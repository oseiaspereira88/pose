---
spec: pose-review-subject-and-scope-precision
category: fixed
breaking: false
refs:
---

A submodule is now recognised wherever it sits. Previously the gitlink was only
resolved when no other rule claimed the path, so a submodule under a mapped
component or a known prefix was classified as implementation and the bundle then
failed trying to read a directory. Component-scoped `validate` tools also stop
accepting evidence from components they do not govern: an overlay's evidence
classes now apply only to the components its selectors matched, while the
repository-wide tool keeps the union.
