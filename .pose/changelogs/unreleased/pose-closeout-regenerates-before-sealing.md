---
spec: pose-closeout-regenerates-before-sealing
category: changed
breaking: false
refs:
---

The closeout skill now states the order a closeout has to follow — regenerate
the evidence into the path `results_path` names, index, seal, attest, and commit
last — and why. The engine reads that one file and nothing else under
`.pose/results/`, and skipping the regeneration does not fail: the bundle
silently seals the previous run's evidence and the attestation cites checks the
bundle does not contain.

In one adopting repository every sealed bundle carried the same two results from
an unrelated component, whatever the spec covered, because `results_path` named
a file nothing regenerated.
