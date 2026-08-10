---
spec: pose-scaffold-exclusion-policy
category: fixed
breaking: false
refs:
---

The published composition contract is no longer embedded in the scaffold, so it
will not be distributed to instances that do not run the services it describes.
The scaffold generator includes by default, which is why a new product-level file
at the repository root became distribution without anyone deciding it. Fixing it
surfaced that the exclusion list existed twice — in the generator and in the
drift guard, the latter annotated as mirroring the former — so an exclusion added
on one side made the generator emit a tree the guard rejected. The list now has a
single home that both consume.
