# ADR: Make delegated review adoption explicit and recoverable

## Status
Accepted for implementation (2026-10-29), spec `pose-delegated-review-capability`.
The maintainer explicitly selected the live B17 barrier in action
`act-31f0849a7ec6712c` (declared assurance, matching current action policy).

## Context

B17 introduced a live signed-run-proof barrier behind `delegated_review`.
B21 adds catalog adoption, an interactive init offer and a disclosed fallback
when explicitly disabled. A missing legacy key cannot mean a new refusal;
merely adding an agent map cannot make automatic conclusions usable. Fresh
init must provision the native machine signer without conferring human
authority, and a failed multi-file setup must remain diagnosable and recoverable.

The accepted B17 ADR deliberately makes opt-in invalidate prefix-only agent
approval under current verification, including an older bundle. B21's
constraint that adoption changes no earlier review can mean either immutable
records or preserved historical validity. The maintainer clarified that historical records stay immutable while
current verification retains the live B17 barrier.

## Decision

Use the following composition of the accepted delegation contracts:

- Preserve legacy behavior when `delegated_review` is absent. Treat explicit
  false as a disabled decision for new plans, sealing the
  `delegated-review-disabled` limitation into the plan and attestation. Keep
  mandatory-human and stronger protected-baseline requirements intact.
- With the capability on, establish a different-actor floor for new plans.
  Automatic routing must obtain another vendor or model; otherwise open the
  existing authorized maintainer decision request without consuming a run.
- On a fresh, accepted interactive init, discover supported CLIs without
  executing them, provision or reuse a project-specific native executor key
  outside the repository, pin it only for attestations, preserve project and
  audience binding, configure provenance and conclusion signing, and enable
  last. A refusal, headless init, EOF or zero supported CLIs records false and
  creates no delegation policy or signing artifacts. An existing instance is
  never treated as fresh merely because its schema stamp is absent.
- Validate a review-capable agent, executor-attested provenance, explicit
  executor identities, the loadable conclusion key and its exact current pin
  before activation. Existing-project adoption provisions no new issuer.
- Preserve key/config after partial failure and report the stage to recover.
  Distinguish pre-publication failure from published-but-unsynced state. Use
  the native review-policy compare-and-swap under an adoption lock, revalidate
  configuration at apply, and check the same configuration bytes after
  publication. On observed drift, disable only our own still-unchanged decision;
  a conflicting rollback reports an unknown state for explicit inspection.
- Let update only suggest adoption. Keep decline/defer as configuration
  decisions rather than silently weakening the review policy. Explicit off
  records false and its date, preserving trust pins and execution history.

Historical approval decision and alternatives:

1. Selected: keep the B17 live barrier. Preserve records without promising
   that an old prefix-only approval satisfies a newly adopted trust bar.
2. Rejected: freeze this barrier per sealed bundle. Preserve the validity of old
   declared approvals; this requires superseding the deliberate B17 exception
   and validating a trustworthy historical policy snapshot.

The native request records the maintainer's explicit selection; neither code
nor tests weaken the existing live barrier.

## Consequences

A local native executor signature remains an executor declaration, never a
vendor receipt, human approval or proof of cognitive independence. No old proof
is reconstructed. Multi-file setup is recoverable rather than falsely atomic;
uncooperative external filesystem writers cannot be excluded by an advisory
lock. Unsafe linked governance paths are refused before provisioning trust.

Rejected alternatives: enabling an unusable agent map (no automatic proof),
granting the machine signer human authority (wrong principal boundary),
deleting partial files on failure (could remove another writer's changes),
treating every absent key as refusal (silently relaxes legacy policy), and
using decline/defer as an implicit off (changes a trust contract as metadata).

Implementation: `delegated_review_adoption.go`, `init_delegation.go`, the
review plan/bundle/assurance path and the native delegation routing surface.
The capability spec contains the negative and integrated journey tests.
