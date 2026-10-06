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

## Causality closeout — measured cost and decision, 2026-10-05

Two measurements on disposable clones, both with `causality_closeout_version: 1`.

**Without a structural profile** the contract had nothing to check: `spec-closeout@1`
has no criterion answering for observed structure, so plans sealed `structure: null`
and two real attestations passed with no mapping. Adopting the flag alone would have
shown the capability effective while protecting nothing.

**With `structural-materiality@1`**, as the contract was designed, the attestation
obligation is real. Every malformed answer was rejected: no mapping, an integrity fact
accepted as a risk, a basis the scope does not declare, a mapping without a reason;
one rationale per fact passed. Over the specs with commits since 2026-09-20 in a full
history clone (176), the overlay selected 100 specs and charged 150 material facts;
40 of those specs were selected only because registering a check rewrites
`.pose/indexes/validation-matrix.json`.

The maintainer answered act-08a9fd2d9a0e50bb with adoption as designed after four
engine remediations, each its own spec:

| Remediation | Spec | Measured effect |
|---|---|---|
| An added validation check is reported, not charged | pose-validation-check-additions-are-not-material | selected specs 100 → 71, facts 150 → 108; matrix-only selections 40 → 6 |
| Adoption cutoff for the contract and for a late overlay | pose-causality-closeout-adoption-cutoff | specs created before the date are not held to it |
| Attest refuses what verify would reject | pose-attest-refuses-what-verify-rejects | every refused attempt above wrote nothing (0 files) |
| New instances adopt; existing ones toggle with `pose adopt` | pose-governed-capabilities-default-on-new-instances | applies to all four capabilities |

The 108 remaining facts are governance and public contracts: ADRs added or changed
(35), skills (19), templates (10), review profiles (11), policy (7), workflows and
rules (4), public contracts (10), and 12 real edits of the validation matrix.

Adopted with `pose adopt causality-closeout --date 2026-10-06 --apply`: the contract
and `structural-materiality@1` apply to specs created on or after 2026-10-06.
