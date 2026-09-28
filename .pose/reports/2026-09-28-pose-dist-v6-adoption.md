# POSE source repository: published v6.0.0 machinery adoption

Date: 2026-09-28. Scope: pose-dist-adopt-published-v6.

## Authentication

Published tag: v6.0.0, commit 3faf1fed5ea9dfa8dcaae385573740bb6099dfd5.
Archive SHA-256: fdeb6fed9f19d81ba6ba21fa9bbec2665266bb6382a5c1eb42d694134442d401.
Binary SHA-256: 2f29fc5054f8a6f8523ba64d0dc5936d4541093820e0724873bd3948a50c7d35.
Both match .pose/releases/v6.0.0/verified-evidence.json. Cosign verify-blob returns Verified OK with the exact release.yml@refs/tags/v6.0.0 identity and GitHub Actions OIDC issuer. The global CLI reports 6.0.0.

## Native delivery and preservation

pose update --no-self delivered machinery without force and recorded engine_version 6.0.0. Repeating the update produced the same managed file content. Git diff contains no policy, release ledger, distributed runtime or contributor-mode changes. The source POSE.md is a distribution template: update reports its unresolved project placeholder and leaves it intact, as expected.

The component assessment was recalculated at task entry. No release tag or runtime provenance is changed by this adoption.

## Deterministic results

- pose check --strict: SUCCESS, zero errors; 15 preexisting changelog/assessment warnings.
- pose skills-check: SUCCESS, 22 checked, zero errors/warnings.
- pose lint-spec pose-dist-adopt-published-v6 --ready-check: SUCCESS.
- pose doctor --json: installed binary 6.0.0; instance schema 1. Informational legacy assessment/policy diagnostics remain visible.

Review and lifecycle transition follow these implementation checks. Existing runtime closure defects are outside this machinery adoption and must be fixed under dedicated engine specs.
