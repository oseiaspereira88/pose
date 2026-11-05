# B34: measured dependency budgets

Measured on 2026-11-05 from implementation checkpoint
`f54e92857cdf7ed7460ec00dcd98a66409812c85`, before the graph-scope correction.
This report records a diagnostic and the resulting contract clarification;
it does not assert that the corrected index, review or closeout has passed.

## Native observation

The canonical strict validation at that checkpoint passed 189/189 checks in
994.8 seconds. The subsequent real `pose index` preflight refused before
publication with:

```text
index input closure could not be proved:
index.generate.preflight: artifact dependency graph exceeded its depth or node budget
```

`ArtifactResolver.ValidateGraph` normally limits a dependency path to 64 edges
and a graph to 256 distinct identities. B34 installed `governanceDepthLimit=8`
and the implementation substituted eight for the total path limit, including
local edges. The accepted ADR describes eight as the federated depth bound.
This mismatch is the correction target; it is not evidence for a larger budget.

## Canonical spec-catalog measurement

The input was generated with the native `pose specs --json`, using the
production spec parser. A separate read-only traversal inspected only its
`slug` and `depends_on` fields. Its topology projection is the lexically sorted
array of `[slug, sorted(depends_on)]` pairs, encoded as compact UTF-8 JSON.
SHA-256 of that path-independent projection:

```text
cbbeaa3f22233d345540ea57b803d459efa4c374e7252a37393161c5f049bd1c
```

| Measurement | Value |
| --- | ---: |
| Parsed specs | 482 |
| Declared spec dependency edges | 553 |
| Excluded/unresolved references in this spec-only traversal | 0 |
| Spec dependency cycles | 0 |
| Longest local spec path, edges | 27 |
| Origins whose longest local path exceeds eight edges | 78 |
| Largest reachable local spec set, including its origin | 111 |

The longest path begins at `pose-integrated-agency-and-runtime-pilot` and ends
at `pose-hierarchical-review-closeout`. Its 27 edges exceed eight while remaining
within the existing 64-edge bound. The reachable spec sets remain below 256.

This traversal excludes roadmap/milestone expansion, transfer redirects,
foreign resolution, authorization and Git proofs. It is not a replay of the
canonical resolver or a proof of captured read closure. The native index
refusal is the direct observation of the production failure.

## Auxiliary roadmap model

A separate delegated read-only model included 482 specs and 23 roadmaps:
653 identities including milestones, 1,169 edges, a maximum path of 30 edges
and a maximum reachable set of 237 identities. In the model, eight caused
97 depth refusals and zero node refusals; 64 caused neither depth nor node
refusals. These are model results, not Go-parser or authorization results.

Its first modeled witness was the local chain from
`milestone:delegation/continuity` through `pose-delegation-handoff`,
`pose-delegation-routing`, `pose-delegation-surface`,
`pose-delegated-adapter-containment`,
`pose-git-and-dispatch-snapshot-integrity`, `pose-recoverable-closeout-plan`,
`pose-typed-producer-diagnostics`, `pose-obligation-contract` and
`pose-blocked-semantics-alignment`. The attempted tenth identity has depth
nine; the full modeled closure has 29 identities and a longest path of 12.
No project crossing is needed for this witness.

The model also found absent milestone references in historical roadmap
content. They were retained as absent; no completeness or remediation claim
is made for them by this report.

## Decision and verification cases

Decision D5 and the ADR clarify independent bounds: 64 total dependency edges,
256 distinct graph identities, 256 declared dependency entries per artifact,
eight project crossings along a composed path and at most 16 authorized roots
per operation. Existing byte, observation, metadata, retry and redirect-chain
limits remain. A cache hit must check its remaining path cost; duplicate queue
entries cannot consume another distinct-node slot. Real excess remains unknown
or incomplete for the affected producer.

Synthetic production-parser/resolver cases cover local paths over eight and at
64/65, federated paths at eight/nine, shared-suffix DAGs with a shortcut visited
first, layered DAGs, 256/257 unique nodes, cycles, repeated input edges and a
real index-material preflight. The composed selection passed in domain (15.681 s)
and CLI (0.973 s). Its first run failed only the new transfer fixtures, which
omitted the transfer-directory parent normally prepared by ApplySpecTransfer;
that fixture prerequisite was added without changing production checks.

The expanded B34 integration then passed with race detection: domain 55.154 s,
CLI 15.652 s and MCP 1.313 s. It includes composed redirect/dependency costs,
historical identity/proof controls and the real queue boundary. Current canonical
validation and actual index publication still require execution after the final
patch. Historical 189/189 results do not qualify later source edits.
