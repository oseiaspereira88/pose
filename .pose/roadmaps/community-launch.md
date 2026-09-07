---
slug: community-launch
status: active
created_at: 2026-09-06
depends_on:
---

# Roadmap: POSE Community Launch

**Outcome:** an external developer who has never spoken to the maintainer can
understand what POSE is, install it, and reach a first governed delivery in a
real repository — and the project can prove that path works before any traffic
is sent to it.

This roadmap does not add product capability. POSE 1.7.x is already further
along technically than its public presentation. The gap being closed here is
between **what the product does** and **what a stranger can discover, install,
understand and repeat**.

Three findings set the ordering.

1. **Distribution is broken, not merely stale.** Releases v1.7.1 through
   v1.7.10 all failed. `releases/latest/download/install.sh` — the exact
   command published on the landing page and in the README — returns HTTP 404.
   The last release carrying artifacts is v1.6.0. Nothing else on this roadmap
   matters until a visitor who copies the first command gets a working binary.
2. **The public surfaces disagree about which product this is.** The landing
   page, the homepage and the structured metadata describe POSE 1.4.3; the
   docs home is marked "applies to 1.4.x"; the README carries a "What's new in
   v1.4.3" section. Version drift is the symptom. The cause is that no gate
   ties a public claim to a released fact.
3. **The category is under-claimed.** POSE is described as a governance layer,
   which invites the reading that it sits on top of someone else's SDD
   framework. POSE owns the specification and delivery lifecycle once adopted.
   Migration interoperability, not concurrent lifecycle ownership, is the
   actual model, and no public surface says so.

Feature work is deliberately frozen for the duration. A spec that adds
capability does not belong on this roadmap.

## Milestone: distribution-repair
- after:
- target_start: 2026-09-06
- target_due: 2026-09-13
- specs: pose-release-security-gate-integrity, pose-release-recovery-verification

**Exit gate:** a tagged release publishes its full artifact set, and a clean
machine installs it from the published command and runs `pose version` and
`pose doctor` successfully.

## Milestone: single-public-truth
- after: distribution-repair
- target_start: 2026-09-13
- target_due: 2026-09-24
- specs: pose-public-claims-contract, pose-canonical-positioning, pose-docs-canonical-route

**Exit gate:** every public surface derives its version and canonical links
from one machine-checked source, and CI fails when a surface contradicts a
released fact.

## Milestone: evaluation-path
- after: single-public-truth
- target_start: 2026-09-24
- target_due: 2026-10-08
- specs: pose-readme-evaluation-path, pose-first-governed-loop-quickstart, pose-sdd-migration-acquisition

**Exit gate:** a first-time reader decides whether POSE is worth their time
from the README alone, and a measured clean-environment run reaches a first
deterministic gate inside the published time budget.

## Milestone: community-surfaces
- after: evaluation-path
- target_start: 2026-10-08
- target_due: 2026-10-15
- specs: pose-launch-proof-demo, pose-community-contribution-surfaces

**Exit gate:** the value is legible in under a minute without reading prose,
and an outside contributor can find a real, scoped, self-contained task.

## Release cut criteria

The launch announcement is gated on facts, not on calendar position. Every
item below is verifiable by command, not by opinion:

- `curl -fsSL .../releases/latest/download/install.sh` returns 200 and the
  installed binary reports the released version.
- `pose public-claims --strict` passes: no public surface asserts a version,
  docs route or download URL that contradicts the latest release.
- The measured clean-environment quickstart reaches a first deterministic
  gate, and the published time budget is that measurement rather than an
  aspiration.
- Migration from at least one external SDD framework is documented and
  exercised by a test, not merely described.
- The demo shows a delivery that is blocked and then legitimately closed —
  an all-green recording does not demonstrate why POSE exists.

## Risk controls

- **Do not measure launch by stars.** The signal that matters is a third party
  running POSE a second time in a repository the maintainer does not control.
- **Do not let the version live in prose.** Any surface that hardcodes a patch
  number will drift again; the fix is to remove the claim or generate it.
- **Do not publish a time budget that has not been measured.** The previous
  "under 10 minutes" claim predates the current lifecycle and was never
  re-measured against a broken installer.
- **Do not announce before distribution is proven from a clean machine.** A
  visitor who copies the first command and gets a 404 does not return.
