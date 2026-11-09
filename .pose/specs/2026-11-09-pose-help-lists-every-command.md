---
slug: pose-help-lists-every-command
status: done
created_at: 2026-11-09
completed_at: 2026-11-09
supersedes:
depends_on: pose-command-reference-parity
remediates:
priority: 1
components: pose-mcp
task_type: bugfix
delivers: capability:help-lists-every-command
---

# Spec: `pose help` lists every command the binary dispatches

## 1. Intent

### Goal

`pose help` advertises every command `pose` dispatches, in both languages, and the manual's command reference documents every command the help advertises.

### Business value

The help is the public surface an agent reads to decide which command to run. 17 dispatched commands were missing from it (`setup`, `context`, `identity`, `issuer`, `adopt`, `migrate`, `specs`, `spec-format`, `start`, `action`, `assess`, `docs-check`, `docs-init`, `docs-review`, `docs-sync`, `release-package-manifests`, and `public-claims` in pt-BR) although most were documented in `POSE.md`. The gate that exists (`pose-command-reference-parity`) compares the help against the manual and assumes that a command absent from the help is internal, so a command added to the binary and to the manual but not to the help passed it: the help was checked against the manual and never against the dispatch.

### Constraints

- The mechanisms are the ones that exist: the help text, the `helpAdvertisedCommands` reader, `CommandReferenceGaps` and its gate in `pose check`, and tests in `internal/cli`.
- The catalog is read from the dispatch switch itself, not kept as a second list.

### Non-goals

- Per-command help (`pose <command> --help`), which several commands still lack.
- The structural gate between the English and pt-BR manuals (`pose-manual-locale-parity`), which stays as it is.
- Changing what any command does.

## 2. Requirements

### Functional

- R1: `pose help` shall advertise every command `mainCommand` dispatches, except aliases of another command.
- R2: The pt-BR help shall advertise the same commands as the English help.
- R3: `pose help` shall advertise no command the binary does not dispatch.
- R4: The command reference of the manual and of its localized and scaffold copies shall document every command the help advertises, so that `pose check` keeps failing on a missing one.
- R5: The reader of the help shall take a command only from the margin of its group, not from the continuation of a description.

### Non-functional

- The catalog is derived from the Go source of the dispatch by the parser of the language, so a command added to the switch is found without a registration step.

### Security

- None: documentation of existing commands.

### Compatibility

- The manuals gain content, not commands: the English manual takes the fuller knowledge-governance text, the MCP-contract paragraph on cross-project writes, the log bullet of the validation matrix and the path of `new-roadmap` from its pt-BR counterpart, and the pt-BR manual takes the `context` and cross-project bullets and the `--wizard` note from the English one. No command changes. `feedback` stays an alias of `report-limitation` and is not listed twice.

## 3. Technical Plan

### Affected areas

`internal/cli/cli.go` (the help texts), `internal/cli/command_reference.go` (the reader), `POSE.md` and its pt-BR and scaffold copies.

### Artifacts

- created: .pose/specs/2026-11-09-pose-help-lists-every-command.md
- created: .pose/starts/pose-help-lists-every-command.json
- created: .pose/changelogs/unreleased/pose-help-lists-every-command.md
- modified: .pose/indexes/validation-matrix.json
- modified: pose-mcp/internal/cli/cli.go
- modified: pose-mcp/internal/cli/command_reference.go
- created: pose-mcp/internal/cli/command_catalog_parity_test.go
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md

### Delivery targets

- capability:help-lists-every-command module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes

`pose help` lists 17 more commands; the manual's reference lists four more (`action`, `issuer`, `migrate`, `release-package-manifests`).

## 4. Tasks

- [x] Measure the three catalogs: dispatch, help, manual
- [x] Add the missing commands to the help, in both languages
- [x] Gate the help against the dispatch, and the languages against each other
- [x] Document the four commands the manual lacked
- [ ] Independent review and closeout

## 5. Decisions

### Decision D1
- Basis: R1, R3
- Minimal option: add the 17 commands to the help and stop
- Selected option: also gate the help against the dispatch read from the switch
- Rationale: the previous gate trusted the help as the catalog, so a fix without a gate from the dispatch would drift again by the same path
- Consequences: a command added to `mainCommand` without a line in the help fails the build
- Falsifier: a command dispatched, advertised nowhere, that the gate does not report

### Decision D2
- Basis: R2
- Minimal option: compare the pt-BR help with the dispatch only
- Selected option: compare the two languages with each other as well
- Rationale: the pt-BR help had lost `public-claims` and carried a continuation line the reader took for a command; a lagging translation is as unusable to an agent as a missing line
- Consequences: both texts change together
- Falsifier: a command advertised in one language and not the other that the test does not report

## 6. Validation

### Strategy

The gate was written first and seen failing with the 17 and 1 omissions listed; the help and the manuals were then changed until it and the existing reference gate passed.

### Deterministic checks

`go test` of `internal/cli` and `internal/scaffold`; the strict matrix in a clean clone.

### Requirement trace
- R1 [satisfied] test:TestPoseHelpListsEveryDispatchedCommand check:help-lists-every-command-integration
- R2 [satisfied] test:TestPoseHelpAdvertisesTheSameCommandsInEveryLanguage
- R3 [satisfied] test:TestPoseHelpAdvertisesNothingTheBinaryDoesNotDispatch
- R4 [satisfied] test:TestTheManualsDocumentEveryCommandTheHelpAdvertises
- R5 [satisfied] test:TestTheHelpReaderIgnoresDescriptionContinuations

### Known gaps
- Per-command help is a fourth catalog: `pose <command> --help` answers "No specific help found" for several dispatched commands (for example `check`, `review-check`, `closeout-check`, `public-claims`). It is a follow-up.
- The dispatch is read from the `switch cmd` of `mainCommand`; a command registered outside that switch would not be seen by the gate.

---

## 7. Final Report

### Delivered scope
`pose help` advertises every command the binary dispatches in English and pt-BR (17 added), the two languages advertise the same commands, the help advertises nothing that is not dispatched, the reader of the help ignores description continuations, and the manual's reference and its three copies document the four commands they lacked. The gate reads the dispatch from the source. Left out: per-command help.

### Files and modules changed
- pose-mcp/internal/cli (cli.go, command_reference.go, command_catalog_parity_test.go)
- POSE.md and its pt-BR and scaffold copies; the validation matrix gains `help-lists-every-command-integration`

### Validation executed
- Command: `pose validate --strict --jobs 4` on a clean clone of the reviewed head, and the suites of `internal/cli`
- Result: recorded by the sealed validation evidence; independent delegated review concluded with `pose review conclude`.

### Residual risks
- A command dispatched outside the switch read by the gate would not be seen.

### Follow-ups

- [open] Give every dispatched command a `pose <command> --help` entry, and gate that the per-command help covers the dispatch as the general help now does. (owner:@pose-maintainers crit:low review:2027-01-08)
