---
type: handoff
slug: delegation-observation-round4
owner: @oseiaspereira88
sensitivity: public-internal
created_at: 2026-10-26
last_reviewed_at: 2026-10-26
expires_at: 2026-11-25
source_refs:
  spec: pose-delegated-review-completion-signal
  workflow: .pose/workflows/review.md
  commands:
    - pose review verify spec:pose-delegated-review-completion-signal
  external_sources: []
---

# Handoff: observation review residual follow-ups

## Context

Independent round 4 of commit a8ce8cf97acb89815cc8440f028ec319c63ec1c3 closes the milestone `observation` of the roadmap `delegation`. The reviewer is the maintainer-authorized declared identity `agent:independent-claude-sonnet-5-5-review` (Claude Sonnet 5.5, Anthropic), the same vendor and another model than the implementer (Claude Opus 5.5, session `f2a4d9f2-d557-4064-ae96-6ed52bb47cd2`); by the rule of 2026-10-09 another model satisfies `different-actor`, another vendor would be preferred. Rounds 1 and 2 were Codex (OpenAI), round 3 and this one Claude Sonnet 5.5. N02 (a detached run read a relative `--note-file` from the project root instead of the caller's directory) and the follow-ups F-R3-01, F-R3-03, F-R3-04, F-R3-06, F-R3-08 and F-R3-10 were reconciled as fixed; no blocker remains.

## Current state

Open, all owned by @oseiaspereira88 with review on 2026-11-25:

- F01 (low, disclosed under D7): the live view redacts with what the stream has declared so far, and the record is derived again from the whole transcript; a capture the record later withholds can still have been shown live.
- F-R3-02 (low, graded medium in round 3 and lowered because the hand-back now precedes the agent, which leaves a window of about 100ms or a supervisor hung over 30s; the engine accepts only low as accepted risk): the SIGTERM, 30s grace and SIGKILL path of a stopped supervisor has no test (swapping SIGTERM for SIGKILL keeps the delegate suites green), and the SIGKILL fallback on the supervisor group does not reach the agent's own group.
- F-R3-05 (low): after `kill -9` of the supervisor the run is `stalled` at about 35s, but its agent keeps running and the live file, claim and marker stay.
- F-R3-07 (low): `on_terminal` fires for `pose delegate review|resume`, not for the deprecated alias `pose review dispatch --via … --apply` nor for a run recorded `stalled`.
- F-R3-09 (low): a reader racing back-to-back heartbeat rewrites can read "ledger file changed while opening" and list the marker as a loss (8 of 8 under 6 readers and 400 consecutive beats; 0 in 115k reads at a beat every 10ms; the real beat is every 10s).
- F-R4-01 (low): F-R3-03, F-R3-04, F-R3-08 and F-R3-10 have no regression test; reverting each leaves `Delegat|Watch|Stall|Help|Detach` green. F-R3-04 and F-R3-10 are testable.
- F-R4-02 (low): since F-R3-01 a hook claimed less than six minutes ago reads as no hook, so an outcome that could not be published is invisible to `ls` for up to six minutes before it is reported as lost; the foreground command still prints the failure it had.
- F-R4-03 (low): the four manuals do not say how a relative `on_terminal` command is resolved (a name with a separator is a path from the project root, a bare name goes through PATH, `..` leaves the project).
- F-R4-04 (low): `watch` of a `stalled` run prints only the stall marker, not the events its live file still holds, and the paths `--detach` prints are project-root-relative although the run now starts in the caller's directory.
- F-R4-05 (low): the regression tests of the review rounds are named outside the `TestDelegat*`/`TestDelegate*` family (`TestADetachedRunReadsTheCallersFiles`, `TestHookBoundAndRelativeCommand`, `TestAnUnrecordedHookOutcomeIsReported`, `TestADetachedRunThatCannotHandBackDoesNotRun`), so the `delegation-surface-integration` check and the documented race selection do not run them; only the unit lane does.

## Next checks

`go test ./internal/pose ./internal/cli -run 'TestADetachedRun|TestHookBound|TestAnUnrecordedHook' -count=1` and the race selection extended with those names. For F-R3-02 a test that holds a supervisor past the grace. For F-R4-01 a stalled-watch test and a test whose live file cannot be read. Run `pose delegate --help` and the four manuals after any change.

## Risks

None of these changes dispatch, ledger, continuity, provenance or containment; each is either a disclosure gap, a test gap or a window of at most six minutes. F-R3-05 and F-R3-02 can leave an agent running after its supervisor died; the stalled state and `ls` name the run, and `pose delegate cancel` stops one that is still recorded.

## Next owner

@oseiaspereira88, review 2026-11-25. No upstream contribution draft or issue was created.

## References

Spec: `.pose/specs/2026-10-18-pose-delegated-review-completion-signal.md`. ADR: `.pose/adr/2026-10-26-delegation-is-a-first-class-execution-surface.md`. Roadmap: `.pose/roadmaps/delegation.md`. Review report: `.review-work/round4/REPORT.md` (local evidence only).
