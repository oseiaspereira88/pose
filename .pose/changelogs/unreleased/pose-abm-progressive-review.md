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
