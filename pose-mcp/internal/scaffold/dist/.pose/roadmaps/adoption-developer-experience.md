---
slug: adoption-developer-experience
status: done
created_at: 2026-07-18
depends_on:
---

# Roadmap: Adoption and developer experience

**Portfolio order:** 6 of 7
**Outcome:** reduce time-to-first-governed-delivery for greenfield and brownfield teams while preserving local-first operation.

Distribution channels begin only after the install and signing contracts are stable. Adoption kits prove real workflows rather than adding marketing-only examples.

## Milestone: trusted-install
- after:
- target_start: 2026-09-21
- target_due: 2026-10-30
- specs: pose-package-manager-distribution, pose-upgrade-compatibility-lab

**Exit gate:** clean environments install a verified release and traverse supported upgrade paths.

## Milestone: guided-adoption
- after: trusted-install
- target_start: 2026-11-02
- target_due: 2026-12-11
- specs: pose-doctor-guided-remediation, pose-brownfield-reference-kits

**Exit gate:** new teams reach a first passing gate and can act on failures without private support.

## Milestone: product-polish
- after: guided-adoption
- target_start: 2026-12-14
- target_due: 2027-01-29
- specs: pose-localization-docs-contract

**Exit gate:** every supported locale and public command stays synchronized with release behavior.

## Risk controls

- Test package channels from clean machines and pin the artifact digest.
- Keep examples executable and versioned with the release they document.
- Never let translation lag hide a safety or compatibility warning.
