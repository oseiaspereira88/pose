---
spec: pose-validation-check-additions-are-not-material
category: changed
breaking: false
refs:
---

The structural delta reads `.pose/indexes/validation-matrix.json` by check. A change that only appends newly named checks is reported as non-material `validation-check` facts, so registering a spec's own check no longer owes a causal mapping; every other matrix change, including an added check that reuses an existing name, stays a material `delivery-metadata` fact. The delta parser version is now `pose-design-delta/v2`.
