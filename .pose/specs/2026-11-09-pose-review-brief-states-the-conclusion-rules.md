---
slug: pose-review-brief-states-the-conclusion-rules
status: in-progress
created_at: 2026-11-09
completed_at:
supersedes:
depends_on: pose-delegated-review-conclusions
remediates:
priority: 1
components: pose-mcp
task_type: bugfix
delivers: capability:review-brief-states-the-conclusion-rules
---

# Spec: The review brief states the rules the conclusion is checked against

## 1. Intent

### Goal

A delegated reviewer that follows the brief writes a conclusion the attestation preflight accepts, whichever agent runs it.

### Business value

Reviews by Claude Opus 5.5 were refused four times by the preflight although the review itself was sound: judgment criteria passed without a conclusion, evidence given as file and line, a criterion disposed as a finding without the id of a filed finding, a structural fact answered nowhere, tools answered `not-run`, a disposition that does not exist, with the required `artifact-check` and `surface-check` never run because the profile allowed no shell, and, once it could run them, tool evidence given as a bare command line where the engine wants `<class>:<ref>` and a sealed ref of the classes `surface-check` lists. The brief of `review-brief/3` named only `rationale` for not-applicable, `<class>:<ref>` for evidence and no `mappings` at all. Codex had stumbled on the same rules earlier in this repository and learned them from notes the implementer carried between rounds, and it reads `AGENTS.md` and the skills of the checkout, which `claude --safe-mode` does not. The gap is the brief, not the agent.

### Constraints

- The mechanisms are the ones POSE has: the brief template and its versions, and the delegation policy of the agent. No new mechanism, no new file.
- A run briefed with an earlier template keeps verifying and concluding: the brief a run was sent never changes under it.
- The reviewer's access does not widen beyond read-only commands.

### Non-goals

- Changing a rule of the preflight.
- Giving the reviewer a way to write, assess or validate.

## 2. Requirements

### Functional

- R1: The brief of the current template shall state, for a run asked to conclude, each rule the engine checks before it records the conclusion: a passed judgment criterion carries a rationale; the evidence of a passed criterion is one of the refs listed under Sealed evidence; a criterion disposed as a finding carries the id of a finding of the draft; structural facts are answered as mappings by the criterion that asks for them; a not-applicable carries a rationale and is refused where the bundle seals evidence of the class the criterion asks for; every required criterion and tool is answered; a tool disposition is passed, failed, not-used or deferred, a required tool is run and passed, a recommended tool not run is not-used with a rationale, the evidence of a tool is a class:ref (a sealed ref of a class the tool lists, otherwise command: and the command run), and an approving draft leaves no finding open (each is a follow-up with action, owner and review_by, or the draft requests changes); and a draft that breaks one is refused whole.
- R2: A test shall tie each blocker the preflight raises on a conclusion to the words of the brief that state its rule, so that a rule added to the preflight without a sentence in the brief fails the build.
- R3: The template shall be versioned: `review-brief/5` is current (it adds the identifiers of the findings a later round accounts for), `review-brief/4` and `review-brief/3` still render exactly as before, so a run briefed with it still verifies, resumes and concludes.
- R4: The delegation profile of the Claude reviewer shall allow the read-only commands a review needs (Git reads, `go test`, `go vet`, and the read-only checks `artifact-check`, `surface-check`, `review-check`, `closeout-check`, `lint-spec`) in its fresh, resumed and resumed-read commands, and shall allow nothing that writes.
- R5: The manual shall say which template asks for the conclusion and that the brief states the rules.

### Non-functional

- The brief grows by one numbered item; the sealed bundle is unchanged.

### Security

- The allowed commands are read-only by their verbs. `pose assess`, `pose validate`, `pose review bundle --seal` and every `pose` verb that writes stay denied, which a probe of the same flags confirms: an allowed command ran and a write was refused.

### Compatibility

- Runs briefed with `review-brief/3` conclude as before. New runs are briefed with `/4`.

## 3. Technical Plan

### Affected areas

`internal/pose/review_brief.go`, `internal/pose/review_conclusions.go`, `.pose/policy/delegation.json`, the manual.

### Artifacts

- created: .pose/specs/2026-11-09-pose-review-brief-states-the-conclusion-rules.md
- created: .pose/starts/pose-review-brief-states-the-conclusion-rules.json
- created: .pose/changelogs/unreleased/pose-review-brief-states-the-conclusion-rules.md
- modified: .pose/indexes/validation-matrix.json
- modified: .pose/policy/delegation.json
- modified: pose-mcp/internal/pose/review_brief.go
- modified: pose-mcp/internal/pose/review_conclusions.go
- modified: pose-mcp/internal/pose/review_brief_test.go
- created: pose-mcp/internal/pose/review_brief_conclusion_rules_test.go
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md

### Delivery targets

- capability:review-brief-states-the-conclusion-rules module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes

The review brief template moves from `review-brief/3` to `review-brief/4`; the reviewer profile `claude-opus-5-5` allows read-only commands.

## 4. Tasks

- [x] Find why the Claude reviewer's drafts were refused and the Codex reviewer's were not
- [x] State the rules in a new template version and keep the old one rendering
- [x] Tie the preflight blockers to the brief by a test
- [x] Allow the read-only commands of a review to the Claude profile
- [x] Update the manual
- [ ] Independent review and closeout

## 5. Decisions

### Decision D1
- Basis: R1, R3
- Minimal option: keep telling each reviewer in the note file
- Selected option: state the rules in the brief, as a new template version
- Rationale: the note file reaches one run and is written by the implementer, the brief reaches every run and is generated by POSE from what the preflight checks; a template version is the mechanism that lets the text change without changing what a run was sent
- Consequences: new runs carry one more item; old runs verify with the template they were sent
- Falsifier: a draft that follows the brief and is refused by a rule the brief does not state

### Decision D2
- Basis: R4
- Minimal option: leave the Claude reviewer without a shell
- Selected option: allow read-only commands by verb in its profile
- Rationale: a reviewer that cannot run anything cites file and line, which the preflight refuses, and cannot reproduce the tests it is asked to judge; Codex runs under a read-only sandbox and can
- Consequences: the profile lists verbs; `pose assess` and `pose validate` are absent because they write generated files
- Falsifier: a read-only verb in the list that writes to the repository

## 6. Validation

### Strategy

The brief is rendered from a real sealed bundle and read for each rule; the older template is rendered and compared; the profile is probed with the same flags it is launched with: an allowed command runs and a write is refused.

### Deterministic checks

`go test -race` of the tests named in the trace; the strict matrix in a clean clone.

### Requirement trace
- R1 [satisfied] test:TestTheBriefStatesEveryRuleThePreflightEnforces check:review-brief-conclusion-rules-integration
- R2 [satisfied] test:TestTheBriefStatesEveryRuleThePreflightEnforces
- R3 [satisfied] test:TestAnEarlierConclusionBriefStillRendersWithoutTheRules test:TestReviewBriefIsRenderedFromTheBundleDeterministically
- R4 [satisfied] test:TestTheClaudeReviewerProfileAllowsOnlyReadOnlyCommands
- R5 [satisfied] test:TestTheBriefOfALaterRoundStatesTheIdentifiersOfPriorFindings test:TestTheBriefStatesEveryRuleThePreflightEnforces

### Known gaps
- The brief states the rules in words; it does not validate a draft before the run ends. A reviewer that ignores it is still refused by the preflight, as before.
- `delegation.json` is the policy of each instance. This repository's copy is changed; the other instances adopt the profile with `pose update` or by editing theirs.
- The brief rules (R1) are tied to the preflight by the words of the blockers in a test; a change to the wording of a blocker needs the table of that test updated.

---

## 7. Final Report

### Delivered scope
The brief of `review-brief/4` states ten rules (the tenth, the identifiers of prior findings, in review-brief/5) the engine checks on a typed conclusion and that a draft breaking one is refused whole (R1); a test ties each preflight blocker to the sentence of the brief that states its rule (R2); `review-brief/3` still renders as before and runs briefed with it conclude (R3); the Claude reviewer profile allows read-only commands in all three of its commands (R4); the manual names the template (R5). Left out: validating a draft before the run ends, and a change to any rule of the preflight.

### Files and modules changed
- pose-mcp/internal/pose (review_brief.go, review_conclusions.go and their tests)
- .pose/policy/delegation.json (the Claude reviewer profile)
- POSE.md and its pt-BR and scaffold copies; the validation matrix gains `review-brief-conclusion-rules-integration`

### Validation executed
- Command: `pose validate --strict --jobs 4` on a clean clone of the reviewed head, and the suites of `internal/pose`, `internal/cli` and `internal/mcpserver`
- Result: recorded by the sealed validation evidence; independent delegated review by a different agent, concluded with `pose review conclude`.

### Residual risks
- A reviewer that does not read the brief still writes a draft the preflight refuses; the refusal names each broken rule.
- The allowed verbs run the repository's tests, which may write to the test cache of the reviewer's environment but not to the repository.

### Follow-ups

- [open] Let `pose review conclude --dry-run` print the preflight blockers of a run's draft before the attestation is attempted, so a reviewer round is not spent on the shape. (owner:@pose-maintainers crit:low review:2027-01-08)
- [open] Offer `pose delegate review` the same read-only verbs for the Codex profile as an explicit list, so both profiles state what a reviewer may run. (owner:@pose-maintainers crit:low review:2027-01-08)
