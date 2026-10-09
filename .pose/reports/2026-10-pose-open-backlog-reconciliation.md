# Open backlog reconciliation — 2026-10-04

Spec: [pose-open-backlog-reconciliation](../specs/2026-10-04-pose-open-backlog-reconciliation.md).
Machine-readable classification: [`pose-open-backlog-reconciliation.json`](../results/pose-open-backlog-reconciliation.json).

## Snapshot

| Field | Value |
|---|---|
| Repository head when measured | `4133b56` (pose-dist `main`, after the wave-0 implementation commits) |
| `spec-graph.json` sha256 prefix | `00d8ab366fd0e641` |
| Non-terminal specs read | 12 (the 30 planning specs created on 2026-10-04 excluded) |
| Open follow-ups (`pose followups --open --json`) | 120 open, 11 overdue, 52 unowned, of 289 total |

The analysis cited 123 open follow-ups from an earlier snapshot. The two numbers
come from different heads and are not a measure of progress.

## Method

Each remaining requirement was read against its spec's trace, tasks and the
tree at the snapshot. Each follow-up was read in full and classified into one of
the analysis's seven classes. A disposition is applied here only when the
evidence is in the tree and the disposition is one the closeout skill lets the
agent record without confirmation (`done`). Every `covered`, `duplicate` or
`spawned` proposal is listed for the maintainer's decision and left `[open]`.
Code existing is never treated as the requirement being met.

## Non-terminal specs, requirement by requirement

| Spec | Requirement | Class | Evidence / reason |
|---|---|---|---|
| pose-agy-smoke-test | R1, R2 | acceptance (external) | Trace marks both satisfied, but the requirement is that the AGY agent produces the file; `examples/smoke-test-agy.txt` is not in the tree. Needs a real AGY + Harne8 Desktop run; Harne8 side tracked by `harne8-pose-open-integrations-reconciliation`. |
| pose-community-contribution-surfaces | R1, R5 | evidence review | Satisfied in trace (`CONTRIBUTING.md`, `scripts/verify.sh`). |
| pose-community-contribution-surfaces | R2, R3, R4 | external operation | Issues and Discussions are actions on the public repository; not performed by this program. |
| pose-docs-canonical-route | R1, R2, R4, R5 | evidence review | Canonical base recorded in `.pose/public/claims.json`; `docs-site/mkdocs.yml` sets `site_url` so pages emit `rel="canonical"`; `public-claims` fails on the deprecated host. Trace not yet written. |
| pose-docs-canonical-route | R3 | external operation, deferred | Host-level redirect; the deferral stands and is not reversed here. |
| pose-first-governed-loop-quickstart | R1, R2, R3, R5 | evidence review | Satisfied in trace with `check:quickstart-loop`. |
| pose-first-governed-loop-quickstart | R4 | acceptance (human) — decision needed | The retained 6.964 s run measures installation and the automated loop only, excluding reading and human development. It satisfies the automated part; the requirement's intent (a first-use experience) needs either a human observation or an amendment of R4's scope. |
| pose-readme-evaluation-path | R1–R5 | evidence review, closeout candidate | All tasks checked; no trace written. |
| pose-launch-proof-demo | R1, R2, R3 | evidence review | Satisfied in trace (`examples/demo/record.sh`). |
| pose-launch-proof-demo | R4 | acceptance | The recording is a capture step; the script exists, the asset needs a capture run. |
| pose-launch-proof-demo | R5 | Harne8 authority | Landing embedding belongs to the site repository (`harne8-pose-open-integrations-reconciliation`). |
| pose-release-recovery-verification | R1, R3, R4, R5 | evidence review | Satisfied in trace. |
| pose-release-recovery-verification | R2 | evidence review | Deferred to "a recovery tag"; releases v6.0.0–v6.2.0 were published since with retained publication evidence and complete asset sets (`.pose/releases/v6.2.0/publication-evidence.json`). Candidate satisfied by that evidence; needs review, not new work. |
| pose-release-security-gate-integrity | R1–R6 | evidence review, closeout candidate | All satisfied in trace; two tasks unchecked; four follow-ups triaged below. |
| pose-sdd-migration-acquisition | R1–R4 | evidence review | Satisfied in trace. |
| pose-sdd-migration-acquisition | R5 | Harne8 authority | Landing link belongs to the site repository. |
| pose-canonical-positioning | R1–R5 | evidence review, closeout candidate | All tasks checked; no trace written. |
| pose-cli-output-rendering-system | R1, R3–R7, R9–R11 | evidence review | Satisfied in trace. |
| pose-cli-output-rendering-system | R2, R8, R12 | implementation or evidence to verify | No trace line. The renderer exists and records fields (`Renderer.RecordField`), but whether every command emits one `--json` document (R8) and whether the docs describe both channels (R12) was not verified here; residual follow-up 094 (localisation parity) stays open. |
| pose-package-channels-deferred-native-verification | R1, R2 | evidence review | Satisfied in trace. |
| pose-package-channels-deferred-native-verification | R3 | external operation, deferred | The final native round is deliberately deferred; not a defect of the engine. |

No requirement above calls for a new feature spec. The closeout candidates
(readme-evaluation-path, canonical-positioning, release-security-gate-integrity)
need a trace and a review, not code.

## Follow-ups

Counts by class, of 120 open:

| Class | Count |
|---|---|
| Material implementation still owed | 32 |
| Evidence review or acceptance of work already produced | 9 |
| Adoption, pilot or rollout | 14 |
| External or human operation | 12 |
| Accepted or deferred residual (revisit-if) | 36 |
| Satisfaction candidate (evidence in tree) | 9 |
| Coverage candidate (another spec delivers it) | 8 |

The per-item class, reason and proposed disposition are in the JSON. The
unowned items (52) are older deferred-scope notes written before the ownership
group existed; they keep their text and gain an owner only when someone takes
them, rather than being assigned by default.

### Applied: `[done]` with evidence

| # | Follow-up | Evidence |
|---|---|---|
| 070 | pose-contract-adoption-registry: report a policy whose `contract_adoptions` and legacy field disagree | `doctor` check `review.adoption-conflict`; test `TestDoctorReportsConflictingAdoptionDates`. |
| 076 | pose-bundles-seal-the-contracts-that-govern-them: count bundles without `governing_contracts` | `doctor` check `review.legacy-bundles`; test `TestDoctorCountsLegacySealedBundleFields`. |
| 084 | pose-bundle-findings-take-the-contract-the-legacy-path-had: count bundles without `gates` | Same check and test as 076. |
| 086 | pose-changelog-adoption-is-the-instances: shipped review policy carries this repository's dates | `pose-mcp/internal/scaffold/dist/.pose/policy/review.json` carries no dates; its comment states that install/update stamp them. |
| 102 | pose-abm-review-authority: claim schema compared with the wrong constant | `verifiedAuthorityBlockers` compares `claim.SchemaVersion` with `ReviewBundleSchemaVersion`; test `TestABMReviewAuthorityRejectsUnsupportedClaimSchema`. |
| 103 | pose-abm-review-authority: `HumanAuthorityIssuers` pin form unchecked | Policy parsing validates the `<issuer>#sha256:<digest>` form; test `TestHumanAuthorityIssuerPinsValidated`. |
| 111 | pose-federated-milestone-scoped-acceptance: close `milestone:harne8-multirepo-consistency/adoption` | Harne8 roadmap `harne8-multirepo-consistency` is `status: done`. |
| 113 | pose-roadmap-gate-scopes-milestones-and-external-members: same milestone | Same. |

### Proposed, awaiting the maintainer's confirmation

| # | Follow-up | Proposed disposition |
|---|---|---|
| 104 | Reviewer authority satisfied by a declared prefix asserts identity it does not verify | `[covered: pose-review-assurance-disclosure]` — the record now discloses declared vs verified separation; whether a stricter default is wanted is a separate decision. |
| 115, 116, 117 | Adopt atomic start, causality closeout and contract nodes after the pilot | `[covered: pose-abm-capability-adoption]` |
| 025, 065 | Dependabot bumps break CI until `action-runtimes.json` is refreshed | `[covered: pose-dependabot-runtime-repair]` |
| 071 | Disable git auto gc for every test fixture | `[covered: test-git-repos-run-no-background-maintenance]` |
| 100 | Harne8 issuer adapter that feeds `verified` mode | `[covered: xref:proj.harne8/spec:harne8-action-request-confirmation-channel]` |

### Items the program's new specs depend on

- 018 (governance-gate-activation R6, overdue): the first quarterly audit and the
  first recurrence intervention still need a human verdict; it is a review of
  evidence that exists, not implementation.
- 063, 064 (release-security-gate-integrity): a failed release or a red `main`
  reaches no human. The daily `release-liveness` workflow now catches an
  unreachable installer, but not a failed tag run; this stays material.
- 097 (abm-design-basis): `lint-spec --strict` counts trace refs without
  resolving `test:` refs, so an invented test name passes. This is exactly the
  formal-compliance shape the adversarial corpus (`pose-mechanization-adversarial-corpus`)
  exists to catch; recorded there as a case, kept open here.

## Limits

The classification reads the tree at one head. It does not re-run each cited
check, does not verify Harne8-side items beyond the roadmap status named above,
and does not publish or close anything outside this repository.

## Maintainer confirmation — 2026-10-05

The maintainer confirmed the proposals above in the Claude Code session
(Decision 5, alternative A-extended), after each coverage claim was checked
against the code:

| # | Disposition recorded | Check made before recording |
|---|---|---|
| 025, 065 | `[covered: pose-dependabot-runtime-repair]` | The repair workflow exists; its seven tests run in `go test ./...` through `TestDependabotRuntimeRepairTrustBoundary`. It has not yet processed a real Dependabot PR: the last actions PR (#128) predates it. The `dependabot.yml` comment that still prescribed the manual refresh was corrected. |
| 071 | `[covered: test-git-repos-run-no-background-maintenance]` | The three packages whose tests commit isolate git in `TestMain`; the only other fixture that inits a repository never commits. |
| 100, and "compose `governance:verified-review-authority` in Harne8" | `[covered: xref:proj.harne8/spec:harne8-action-request-confirmation-channel]` | That spec's R2 and R4 are the issuer for action requests and review drafts. Qualified dispositions were refused by `lint-spec` until pose-followup-dispositions-accept-qualified-refs. |
| "`Project` and `Audience` compared with the same value" | `[spawned: pose-authority-claim-project-is-not-the-audience]` | Resolved before 7.0.0: `project` binds `authority_project`, `audience` binds `authority_audience`. |
| 104 | `[covered: pose-review-assurance-disclosure]` | Outputs state the identity as declared and the separation as not verified. Requiring verified identity by default is decided with the Harne8 confirmation channel. |
| 115, 116, 117 | `[done]` | Resolved by the adoption decisions of pose-abm-capability-adoption. |

Each covering spec now names the follow-ups it covers in its Final Report, so the
covered-anchor check finds them.

## Second pass — 2026-10-09

| Field | Value |
|---|---|
| Repository head when measured | `191e568e` |
| `spec-graph.json` sha256 prefix | `0703bd3e11fa0372` |
| Open follow-ups (`pose followups --open --json`) | 107 open, 10 overdue, 43 unowned, of 307 total |

The counts come from a different head than the first pass and are not a measure
of progress.

The maintainer set the scope of this pass: close every non-terminal spec and
deliver the high and medium material follow-ups before the next release;
low-criticality items stay open with owners as the next version's backlog;
requirements that depend on an operation outside the repository are recorded as
deferred integrations with an owner, never as satisfied; the quickstart's R4 is
amended to what was measured.

Delivered in new specs, each closed with its own review: red signals reach a
person (`pose-red-signals-reach-a-person`), trace `test:` refs resolve
(`pose-trace-test-refs-resolve`), locale parity compares obligation flags
(`pose-locale-obligation-flags`), a contaminated range names its other work
(`pose-range-names-its-other-work`), `pose check` spawns 70% fewer Git
processes (`pose-check-spawns-fewer-git-processes`), Attention projects every
source (`pose-attention-projects-every-source`) and answers in about a second
(`pose-attention-within-a-second`), and recurrence-check reports flapping tasks
(`pose-recurrence-flapping-signal`). Two discovery follow-ups were recorded as
covered by `pose-discovery-gitignore-and-root-alias-fix` after the maintainer's
confirmation.

Eleven of the twelve specs of the first pass were closed with requirement
traces. The twelfth, `pose-package-channels-deferred-native-verification`,
closes with the next release, whose native round it exists to record. External
parts deferred with owners: the old docs host redirect, the canonical link on
the Harne8-served docs, the GitHub repository description, scoped issues and
Discussions, the demo recording and its embedding, and the AGY smoke, whose
earlier trace claimed a file that was never committed.
