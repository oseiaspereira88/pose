---
spec: pose-sbom-negative-coverage
category: changed
breaking: false
refs:
---

The SBOM assertions are now exercised on every CI run, including the
license-coverage floor. They had been unreachable by the negative harness:
inside the artifact-identity gate they sit behind signature verification, and
the synthetic fixtures are unsigned by construction, so every run died before
any SBOM assertion was evaluated. They are their own script now, and the harness
rejects an inventory with no licenses, one below the floor, and one with no
components — then accepts a compliant inventory, so the rejections are not the
harness refusing everything.
