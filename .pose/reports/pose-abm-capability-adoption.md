# ABM capability adoption — shadow and decisions, 2026-10-04

Spec: [pose-abm-capability-adoption](../specs/2026-10-04-pose-abm-capability-adoption.md).
Machine-readable record: [`pose-abm-capability-adoption.json`](../results/pose-abm-capability-adoption.json).

The three capabilities (`contract_nodes_version`, `atomic_start_version`,
`causality_closeout_version`) are implemented and unadopted. Nothing here
reimplements them. This report measures what adopting each would do to this
repository, and the decision is left to the maintainer through three action
requests. The review policy is unchanged.

## Method

Five disposable clones of pose-dist at `ce17db6`, one per configuration: no flag
(baseline), each flag alone, and all three. Each clone ran `lint-spec --all`,
`state --attention --json`, `review bundle spec:<slug> --json` for every one of the
39 in-progress specs, and `start spec:<slug>` (preview) for the 3 drafts. Baseline
and all three also ran `check --strict`. The identity was declared
(`POSE_DEFAULT_PROJECT_ID=proj.pose-dist`).

## Results

| Configuration | lint (errors/warnings) | Attention: start-reconciliation | Bundles stamped with causality-closeout | check --strict |
|---|---|---:|---:|---|
| baseline | 0 / 172 | 0 | 0 | SUCCESS, 17 warnings, 80.9 s |
| contract-nodes | 0 / 172 | 0 | 0 | not run |
| atomic-start | 0 / 172 | 39 | 0 | not run |
| causality | 0 / 172 | 0 | 39 | not run |
| all three | 0 / 172 | 39 | 39 | SUCCESS, 17 warnings, 81.5 s |

Every other Attention count (review blockers 59, review not approved 39, open
follow-ups 112, dependencies 72) and every bundle's blockers and criteria were
identical across configurations. The draft start previews exited 0 everywhere.

## Per capability

**Contract nodes.** No observable effect in this corpus. The gate acts on
amendments, and no flat spec has an amendment journal. Low risk here, unmeasured
where amendments are frequent. Recommendation: adopt.

**Atomic start: stops rollout (R4).** Adoption adds one `start-reconciliation`
obligation per in-progress spec, 39 in all, each restricting closeout: "spec is
in-progress without a recorded start". No command records a baseline for a spec
already in progress, and adoption has no cutoff date, so adopting today would block
39 closeouts with no remedy. The code is not invalidated: what is missing is either
an explicit legacy baseline (`pose start --baseline-existing`, or similar) or an
adoption cutoff like `review_bundles_adopted_at`, so that specs started before
adoption are reported without being blocked. Recommendation: defer, and record that
follow-up.

**Causality closeout.** The contract is stamped on all 39 prepared bundles. Nothing
changes at preparation; the cost shows up when an attestation is concluded under
the contract, and the shadow concluded none. Recommendation: defer until one
attestation is concluded under it and its cost is recorded.

## Decisions requested

| Capability | Action request | Recommendation |
|---|---|---|
| contract-nodes | `act-7d587a8e4f3bf0fc` | adopt |
| atomic-start | `act-fa1d72f029d567a0` | defer |
| causality-closeout | `act-08a9fd2d9a0e50bb` | defer |

Each request is addressed to `human:oseias` and restricts this spec's closeout. Once
a decision is answered, effective governance (`pose state --governance`) shows the
result per capability, because it reads the policy (R5). The policy key changes in a
separate commit after the answer, never before it.
