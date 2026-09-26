# Federated proof for current carried-forward validation

Date: 2026-09-26. Spec: `pose-federated-carried-forward-proof`.

## Reproduction and cause

The Harne8 consumer pinned POSE at `c19ff2d` and adopted its exact contract,
policy and revision digests. `pose roadmap-check harne8-multirepo-consistency
--strict` then blocked four reviewed source specs with
`source-delivery-evidence-missing`. Their passing module checks were sealed as
`carried-forward`: validation commit `a8a116e` followed each subject commit.
The source bundles were fresh and approved. The source review contract treats
these results as current by provenance, but `federatedSourceProof` admitted
only evidence marked `observed` when checking delivery classes or building the
manifest. The existing synthetic source fixture used identical subject and
validation heads, so it did not cover this case.

## Fix and boundary

Accept a passing, provenance-bearing `carried-forward` result from a fresh,
approved source bundle only when its commit is after the reviewed subject and
within the pinned source history. Keep exact `observed` evidence valid.
Reject pre-subject, missing-commit, outside-pin, failed, missing-provenance
and mismatched semantic-subject variants. Publish only eligible evidence in
the federated manifest. Git ancestry is checked with validated commit IDs,
without shell interpolation, and cached per distinct evidence head.

This preserves the source's own current-evidence decision and adds a temporal
guard; it does not make `done` or a copied attestation sufficient. Reverting
the fix requires reverting the consumer trust pin together with the engine
revision so no consumer accepts a contract it cannot verify.

## Validation

| Command | Result |
| --- | --- |
| `go test ./internal/pose -run TestFederatedAcceptanceCarriedForwardProof -count=1` before fix | Failed on the post-subject case with missing integration and unit evidence; negative cases remained blocked. |
| Same focused test after fix | Passed; manifest included the carried integration evidence. |
| `go test ./internal/pose ./internal/cli -run 'TestFederatedRoadmap|TestFederatedAcceptance' -count=1` | Passed. |
| `go test -race ./internal/pose -run TestFederatedAcceptance -count=1` | Passed. |
| `go vet ./internal/pose` | Passed. |
| `pose validate --strict --module pose-mcp --report` | 25/25 checks passed during implementation; repeat against the committed subject for canonical evidence. |
| `pose assess tech-debt` / `pose assess integrate` | Zero debt markers; 57 contracts and 56 pre-existing unobserved-consumer gaps. |
| `pose check --strict` | One unrelated blocker: missing tracked knowledge artifact declared by `pose-scaffold-self-referential-policy-fix`; source-tree structural warnings were otherwise historical. |

Harne8's source pin remains `c19ff2d` until this source change receives its
own review. The consumer's policy and roadmap still block composition; no
Harne8 acceptance is inferred from these module tests.
