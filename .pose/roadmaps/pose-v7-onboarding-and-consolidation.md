---
slug: pose-v7-onboarding-and-consolidation
status: draft
created_at: 2026-10-05
depends_on:
---

# Roadmap: Onboarding, configuration and identity, consolidated for 7.0.0

**Outcome:** a person adopting POSE — alone or through Harne8 — enters through one command, sees one map of what is in force and what is new, answers decisions with a proof POSE can check, and learns the real lifecycle from a first spec that is itself governed. Nothing is turned on in an existing instance without the maintainer's confirmation.

Source: the maintainer's review of Decision 7 on 2026-10-05, after a fresh
install and a 6.2.0 → current upgrade were walked by hand. Findings:
`pose init` leaves an instance `pose check` rejects; install ends without a next
step; a fresh `doctor` opens with four warnings; `pose update` announces none of
the capabilities it brings ("Nothing to do"); `pose adopt` covers four of eight
adoptable capabilities; answers can only be declared without Harne8; the
quickstart bypasses `pose start`, review and `pose close`.

## Milestone: entry-and-update
- after:
- target_start: 2026-10-05
- target_due:
- specs: pose-project-identity-file, pose-capability-catalog, pose-init-is-install, pose-fresh-install-doctor-is-clean, pose-setup-command, pose-update-configuration-review

**Exit gate:** `pose init` and `pose install` produce the same working instance; a fresh `doctor` reports no warning; every adoptable capability is toggled by `pose adopt`; `pose setup` shows what is in force, what is new and what is missing; an update that brings capabilities scaffolds a configuration-review spec whose decisions are applied only after the maintainer answers.

## Milestone: identity
- after: entry-and-update
- target_start: 2026-10-05
- target_due:
- specs: pose-signed-action-answers, pose-mcp-action-resolve-signed-only

**Exit gate:** without Harne8, a maintainer answers with a signature from a key registered in the policy and POSE verifies it offline; a key that requires presence is distinguished from one that does not; the Harne8 issuer is the same contract; MCP resolves only signed answers.

## Milestone: guided-onboarding
- after: identity
- target_start: 2026-10-05
- target_due:
- specs: pose-onboarding-spec, pose-quickstart-real-lifecycle, pose-install-and-upgrade-journeys

**Exit gate:** a fresh install carries a `pose-onboarding` spec that `pose setup` drives to done; the quickstart chapters follow start, decisions, review and close and each is verified in CI; the install and upgrade journeys run in CI against the previous release.

## Release cut criteria

7.0.0 needs the three milestones verified. The Harne8 side (issuer key and
confirmation screen) is tracked by `xref:proj.harne8/spec:harne8-action-request-confirmation-channel`
and is not a cut criterion: POSE must be complete without it.

## Risk controls

- Nothing is adopted in an existing instance without a recorded answer.
- A signature from a key without presence is never presented as proof of a person.
