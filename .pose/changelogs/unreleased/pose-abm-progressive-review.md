---
spec: pose-abm-progressive-review
category: added
breaking: false
---

Ship opt-in engineering-judgment and high-criticality-review profiles for declared
delivery targets. High and critical scopes can require a different reviewer;
overlapping profiles deduplicate the three explicit judgments without adding tools.
Installation does not activate the profiles.

`review-plan` now explains the effective plan: a `band` with one entry per reason
(trigger, basis, source, policy, obligations), an `unknown` entry for an overlay
that could not decide a component, and a `declared-forecast` projection that
separates the scope the spec declared from the obligations only delivery
provenance added. Both are derived through the existing selectors and composer,
add no obligation, and stay out of the plan digest, so no sealed review is
superseded.

A scope that changes review policy, profiles or their schemas is now resolved under
the contract at the change set's resolved base, and obligations the diff weakened
are restored for that review: reviewer independence, dropped criteria and criteria
softened from required to optional or from judged to collected. `policy_baseline`
reports what was weakened and restored. A diff that disables review, or that
removes the profile the policy points at, no longer makes itself unreviewable; an
unresolvable base is reported unprotected rather than implied protected.

Review profiles can select on observed structure. `structural-materiality@1` ships
opt-in and matches what the sealed subject was observed to do — direct
dependencies, component boundaries, delivery metadata, governance contracts,
public contracts and submodules — so an undeclared scope can still be material
while a transitive bump, a lock file or an unreadable manifest is reported without
becoming an obligation. Under the new `structural-causality` contract, a criterion
declaring `requires_structural_mapping` must answer for each material fact with a
decision basis that reaches a requirement, or with an explicit missing-evidence,
not-applicable or accepted-risk and a rationale; record them with
`pose review attest --mapping`.

Fixed: the structural detector treated a directory prefix as proof, so every file
under a top-level `api/` — including a README — was observed as a public-contract
change. The prefix rules now require a contract-bearing file shape.
