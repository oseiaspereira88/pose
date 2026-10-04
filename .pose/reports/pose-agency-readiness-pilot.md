# Agency-readiness pilot — automated rehearsal, 2026-10-04

Spec: [pose-agency-readiness-pilot](../specs/2026-10-04-pose-agency-readiness-pilot.md).
Machine-readable record: [`pose-agency-readiness-pilot.json`](../results/pose-agency-readiness-pilot.json).

## What this is, and what it is not

This is an **automated rehearsal** of the vertical slice on a disposable clone of
pose-dist at `768e6c7`. An agent ran a script. The "human" principal
(`human:pilot-rehearsal`) is a script fixture: no person answered anything, so this
report says nothing about friction for a person. Adoption
(`agency_readiness_version: 1` and a role map) was made in the clone only. The real
repository's policy is unchanged; it changes only after the stop/go below.

## The slice

A real in-progress spec (`pose-v7-legacy-cleanup-plan`) received a human decision
restricting its closeout. Every step's exit code and duration are in the results
file.

| Step | Observed |
|---|---|
| Attention before | no request; 2.6 s |
| Open the decision (`--effect closeout:block`, subject `requirement:R5`) | recorded with id and digest; 36 ms |
| Attention and `action list --present` | the request shows, marked as able to wait until closeout |
| `pose close` | refused: `close.refused.action-request-pending` names the request |
| The requesting agent answers its own decision | refused: `action-actor-not-authorized` |
| An answer against a stale revision | refused: `action-revision-conflict` |
| The authorized answer, then the same call again | recorded once; the replay is a no-op |
| Attention after, closeout plan | the request is gone; the plan continues to evidence and sealing |
| An unrelated commit | the answer stands; nothing is asked again |
| The subject (`R5`) changes | the answer is invalidated and the request is asked again at revision 2 |
| An external operation, advisory on release | recorded; does not restrict closeout |
| `stats governance --waits` | age, attributed wait and blocking reported separately |

No subsystem had to be searched by hand: open, Attention, refusal, resolve,
revalidation and continuation each pointed at the next step (requirement R1). The
trivial change kept the minimal flow and did not re-ask (R2).

## Baseline

The engine before this roadmap (`133478d`) has no channel for the decision: it would
have lived in chat or in a follow-up line, `pose close` would not have refused, and
nothing would have shown it as pending. There is no equivalent operation to time, so
the comparison is of capability, not speed.

| Measure | Baseline | Rehearsal |
|---|---|---|
| Commands to raise, see, answer and verify a decision | not possible | 6 (open, attention, resolve, attention, show, close plan) |
| Scripted human interventions | — | 2 answers |
| Invalidations by cause | — | 1 subject-changed, 0 unrelated |
| Refusals | — | 4 expected, 4 observed |

## Defects found and fixed

Each fix landed under the spec that owns the code, with a test that fails without it.

- **`stats governance --waits` took 88 s**, recomputing the whole outcomes report
  first. Fixed in `2a0fdc8`: 13 ms.
- **A directory-derived project identity was silent.** The clone recorded its request
  as `xref:proj.pilot-clone/…`. The obligation snapshot now lists the limitation
  whenever no identity is declared, and `action open --apply` warns. Fixed in
  `768e6c7`.

## Unknowns

- Friction for a person: nobody ran the flow.
- Authority transfer and Harne8 consumption were not exercised here; unit tests cover
  them (`pose-transfer-preserves-obligations`, Harne8 specs not yet implemented).
- Attention takes about 2.2 to 2.6 s on this corpus, above the 1 s target recorded in
  `pose-obligation-projection`.

No gate was adjusted to make the rehearsal pass (R3).

## Stop/go

**Pending: a maintainer's decision.** It was raised through the slice itself, as an
action request on this spec addressed to `human:oseias`. The agent's
recommendation, which is not the decision:

- **Go, delimited:** adopt `agency_readiness_version: 1` in pose-dist only, with a
  role map naming the maintainer. Harne8 waits for its own adoption spec.
- **Rollback condition:** revert the policy key if an action request blocks a closeout
  that the maintainer judges should have proceeded and the cause is the engine, not
  the request; or if Attention misses a request that exists.
- **Before any wider adoption:** one run of the slice by a person, and the Harne8
  attention surface.
