---
slug: pose-compat-gate-candidate-integrity
status: done
created_at: 2026-08-07
completed_at: 2026-08-07
supersedes:
depends_on: pose-installer-local-binary-precedence
priority: 1
components: release, installer
delivers: governance:compat-gate-candidate-integrity
---

# Spec: The compatibility gate tests the candidate, and upgrades never lose text

## 1. Intent

### Goal
Stop the release compatibility gate from replacing the candidate binary with the
last published release mid-run, and stop `pose upgrade` from silently discarding
manual text an instance wrote without its own heading.

### Business value
`tests/release/compat.sh` ran `"$candidate" upgrade`, and a bare `upgrade`
self-updates to the latest published release. In the v0.18.1 run it printed
`updating pose binary: v0.18.1 -> v0.17.0` while processing the first pair, so
the three remaining pairs validated v0.17.0 and passed. Their `PASS` lines
asserted nothing about the candidate. What the contaminated gate hid is the
second defect: merging a managed manual refreshes engine-owned section bodies,
so a note the instance appended without a heading of its own disappears on
upgrade.

### Constraints
- Upgrades from releases before 0.18.2 are out of support: those versions have
  no installed base, so the gate must not spend network and time on them.
- Release history stays intact. Narrowing the support window is a matrix change,
  never a deletion of published tags, manifests or notes.

### Non-goals
- Changing what `MergeManagedDoc` considers engine-owned. Refreshing those
  bodies is the point of the merge; only losing them silently is the defect.
- Removing published releases or their POSE artifacts.

---

## 2. Requirements

### Functional
- R1: The compatibility gate shall run every candidate invocation with
  `--no-self`, so the candidate binary is the one under test for every pair.
- R2: The supported-upgrade matrix shall start at the first release that
  actually publishes, and the gate shall report an empty matrix as a declared
  support window rather than a skip.
- R3: When merging a managed manual would drop a non-blank line the instance
  wrote, `pose install`/`pose upgrade` shall keep the pre-merge file as
  `<doc>.pose-backup` and say so, unless `--no-backup` is passed.

### Non-functional
- The gate must not need the network when the matrix is empty.

### Security
- The backup inherits the manual's own permissions and stays inside the target;
  no new path is accepted from the caller.

### Compatibility
- `MergeManagedDoc` keeps its signature and behaviour; the drop detection is a
  separate, additive query.

---

## 3. Technical Plan

### Affected areas
- `tests/release/compat.sh` — candidate integrity across pairs.
- `compatibility.json` — support window.
- `pose-mcp/internal/cli/managed_docs.go` — drop detection.
- `pose-mcp/internal/cli/install.go` — backup on a lossy merge.

### Artifacts
- created: .pose/specs/pose-compat-gate-candidate-integrity/spec.md
- modified: tests/release/compat.sh
- modified: compatibility.json
- modified: pose-mcp/internal/cli/managed_docs.go
- modified: pose-mcp/internal/cli/managed_docs_test.go
- modified: pose-mcp/internal/cli/install.go
- created: pose-mcp/internal/scaffold/dist/.pose/specs/pose-compat-gate-candidate-integrity/spec.md

### Delivery targets
- governance:compat-gate-candidate-integrity module:pose-mcp profile:release-governance entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes
- `MergeDropsLocalContent` is new and exported for the installer; no existing
  signature changes.

### Technical risks
- Line-multiset comparison reports a drop when the instance duplicated a line
  that the canonical manual also carries. The failure mode is an extra backup
  file, never a lost one.

---

## 4. Tasks

### Planning
- [x] Reproduce the self-update mid-gate and confirm which pairs it contaminates
- [x] Separate the contaminated failure from the real one

### Implementation
- [x] Pass `--no-self` on every candidate invocation in the gate
- [x] Narrow the support window to the publishing release without deleting history
- [x] Detect a lossy merge and back the manual up before rewriting it

### Validation
- [x] Run the compatibility gate against the candidate
- [x] Run the mandatory checks

---

## 5. Decisions

### Decision 1
- Date: 2026-08-07
- Context: upgrades from 0.9.0–0.17.0 leave the instance failing `check --strict`
  because the delivered `AGENTS.md` references `.pose/assessments` and
  `.pose/state`, which those instances never had.
- Options considered: (a) make the upgrade create the directories the delivered
  manual references; (b) narrow the support window to the first release that publishes.
- Decision: (b).
- Rationale: the project owner confirmed there is no installed base before
  0.18.2, so (a) would be machinery maintained for nobody. Narrowing is also
  honest: the gate stops claiming to exercise upgrades the project will not
  support.
- Consequences: the matrix is empty until 0.18.2 has a successor; the first pair
  it exercises again will be 0.18.2 → next. Published tags and their POSE
  artifacts are untouched.

---

## 6. Validation

### Strategy
Run the real gate. With the matrix narrowed it must reach `COMPATIBLE` without
network access, and its contract gates must still exercise the candidate tree.
Cover the drop detection with a unit test over the three cases that matter —
text without a heading, a section the engine does not know, and an untouched
manual — then confirm end to end that a 0.17.0 instance upgrading to the
candidate keeps a recoverable copy.

### Deterministic checks

#### Test
- Command: `go -C pose-mcp test ./internal/cli -run "MergeDrops|ManagedDoc" -count=1`
- Scope: managed-manual merge and drop detection
- Expected: ok

#### Security / Contract
- Command: `bash tests/release/compat.sh v0.18.2`
- Scope: release compatibility gate
- Expected: `Result: COMPATIBLE — release gate passed.`

### Execution log
- Date: 2026-08-07
- Environment: linux/amd64, Go 1.26.5, pose 0.18.2-dev.
- Notes: the contaminated run is CI run 31147297512, where the gate printed
  `updating pose binary: v0.18.1 -> v0.17.0` and then passed three pairs against
  the wrong engine. Re-running the 0.14.0 pair with `--no-self` reproduced the
  real failure: `customizacao PERDIDA` plus two broken references.

### Results summary
- Successes: compatibility gate (COMPATIBLE), `go test ./...`, `go vet ./...`,
  `pose check --strict`, installer E2E
- Failures: none
- Warnings: none

### Requirement trace
- R1 [satisfied] check:compat-gate test:tests/release/compat.sh — the gate reaches COMPATIBLE with the candidate intact; before the fix it rewrote the candidate to v0.17.0
- R2 [satisfied] check:compat-gate report:compatibility.json — the gate reports "none declared: support window starts at 0.18.2"
- R3 [satisfied] governance:compat-gate-candidate-integrity evidence:integration check:delivery-integration test:TestMergeDropsLocalContentDetectsTextWithoutItsOwnHeading — a 0.17.0 instance upgrading to the candidate keeps AGENTS.md.pose-backup with the dropped note

### Known gaps
- The empty matrix means the gate currently proves nothing about upgrade pairs.
  That is accurate — there are none in support — but it also means the pair
  machinery itself is unexercised until the next release adds one back.

---

## 7. Final Report

### Delivered scope
The compatibility gate now runs every candidate invocation with `--no-self`, so
no pair can be validated by a previously published engine. The support window
starts at 0.18.2, with published releases left untouched. A merge that would
drop instance-written text writes `<doc>.pose-backup` first and reports it.

### Files and modules changed
- tests/release/compat.sh
- compatibility.json
- pose-mcp/internal/cli/managed_docs.go
- pose-mcp/internal/cli/install.go

### Validation executed
- Command: `bash tests/release/compat.sh v0.18.2`
- Result: COMPATIBLE

### Residual risks
- The drop detection compares line multisets, so a manual whose instance
  duplicated a canonical line produces a spurious backup. Harmless, but noisy.

### Follow-ups

- [spawned: pose-release-cycle-debt-closure] Re-add an upgrade pair to `supported_upgrades` at the first release after 0.18.2, so the pair machinery in `compat.sh` stops being dead code. (owner:@pose-maintainers crit:high review:2026-09-04)
- [duplicate: pose-mcp-active-context] `pose release prepare` archives changelog fragments without rewriting the artifact claims of the specs it consumes, so every cut breaks the structural gate for those specs. Seen a third time in this cycle; `pose-mcp-active-context` already carries the fuller diagnosis and is the item that spawned pose-release-cycle-debt-closure. (owner:@pose-maintainers crit:high review:2026-09-04)
