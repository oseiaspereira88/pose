---
slug: pose-instance-engine-version-tracking
status: in-progress
created_at: 2026-08-15
completed_at:
supersedes:
depends_on:
priority: 3
components: pose-mcp, cli
delivers: capability:pose-mcp
---

# Spec: pose-instance-engine-version-tracking

---

## 1. Intent

### Goal
Record which `pose` engine semver last delivered machinery to an instance,
and let `pose version [path]` report it, so "is this repo current" is
answerable without diffing content against a reference checkout.

### Business value
`github.com/oseiaspereira88/pose#20`: no persisted marker anywhere in a
POSE instance records the delivering engine's semver. `.pose/schema-version`
is an integer *migration* counter (currently `1` since adoption) that
doesn't change between minor/patch releases, so an instance last synced by
v1.2.0 and one synced by v1.2.2 both report identical `schema: 1` even
though their machinery content differs. Found while updating several
downstream projects (codass, audio-relay, harne8) to v1.2.2 in one
session: the only way to confirm each was actually in sync was `pose
update --no-self` (a live merge, not a check) followed by `git diff`
against whatever it produced — no cheap "check first" existed.

### Constraints
- The recorded version must be the version that actually *delivered*
  machinery — i.e. written by `deliverMachinery`'s own success path
  (`saveMachineryManifest`), not by `pose version` itself or anything
  read-only.
- Must not fabricate a version for instances that predate this feature —
  `pose version [path]` omits the line entirely rather than guessing when
  no engine version was ever recorded (an untouched pre-existing
  `machinery-manifest.json` has no such field).
- `pose version` (no path) keeps today's exact behavior — current-directory
  resolution only changes when a path argument is given.

### Non-goals
- Tracking version history (when each past update ran) — only the most
  recent delivering version.
- Changing `.pose/schema-version`'s own migration-counter semantics.

---

## 2. Requirements

### Functional
- R1: On every successful `deliverMachinery` call (both `pose update`'s
  non-force path and `pose install`, force or not), `saveMachineryManifest`
  shall record the delivering engine's `version.Version` string in
  `.pose/state/machinery-manifest.json`.
- R2: `pose version [path]` shall accept an optional target path argument;
  when given, it resolves the instance rooted there instead of the current
  directory (mirroring how `pose install <target>` takes an explicit
  target rather than relying on cwd).
- R3: When the resolved instance's `machinery-manifest.json` carries a
  recorded engine version, `pose version` shall print it alongside the
  existing `schema:` line.
- R4: When no engine version was ever recorded (pre-existing instance,
  feature not yet exercised there), `pose version` shall omit that line —
  never print a guessed or empty value.

### Non-functional
- No new file: reuse the existing `machinery-manifest.json`, adding one
  field rather than a second marker file that could drift from it.

### Compatibility
- `machineryManifest`'s `schema_version` field (the manifest *format*
  version, currently `1`, unrelated to this spec) is unaffected; the new
  field is purely additive JSON, so an older `pose` binary reading a
  manifest written by a newer one just ignores the unknown key.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/cli/machinery.go` (`machineryManifest` struct,
  `saveMachineryManifest`, call sites)
- `pose-mcp/internal/cli/cli.go` (`version` dispatch, `cmdVersion`)

### Artifacts
- modified: pose-mcp/internal/cli/machinery.go
- modified: pose-mcp/internal/cli/cli.go
- modified: pose-mcp/internal/cli/cli_test.go

### Delivery targets
- capability:pose-mcp module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### Technical risks
- Low: additive JSON field and an optional CLI argument; no change to
  existing call signatures used elsewhere without updating them.

---

## 4. Tasks

### Planning
- [x] Confirm the exact gap: `.pose/schema-version` is a migration
      counter, `machinery-manifest.json`'s `schema_version` is a fixed
      manifest-format version — neither tracks engine semver
- [x] Confirm `cmdVersion(stdout)` today takes no arguments and `pose
      version <anything>` silently ignores extra args

### Implementation
- [x] Added `EngineVersion string \`json:"engine_version,omitempty"\`` to
      `machineryManifest`; `saveMachineryManifest` populates it from the
      package-level `Version` (its only call site, inside
      `deliverMachinery`, covers both `pose update`'s non-force path and
      `pose install`, force or not)
- [x] Refactored `projectRoot()` into `projectRootAt(dir)` (git top-level
      from `dir`, falling back to `dir`'s absolute path) with
      `projectRoot()` now calling `projectRootAt(".")` — preserves exact
      prior behavior for the no-arg case
- [x] `version` dispatch in `cli.go` passes an optional first arg through
      to `cmdVersion(w, target)`; prints the recorded engine version line
      only when `instanceEngineVersion(root)` returns non-empty

### Validation
- [x] `TestVersionWithPathArgReportsRecordedEngineVersion`: fresh install
      records `Version` in the manifest; `pose version <path>` — run from
      an unrelated cwd — resolves the target and prints
      `instance last updated by: pose <Version>`
- [x] `TestVersionOmitsEngineVersionForPreExistingManifest`: a
      hand-written manifest without `engine_version` produces no such line
- [x] `go -C pose-mcp test ./...`, `go -C pose-mcp vet ./...`, `gofmt -l`:
      all clean
- [x] Manually verified against `~/GolandProjects/codass` (real project,
      not a fixture): `pose version <path>` before any update showed no
      engine-version line; after `pose update --no-self`,
      `instance last updated by: pose 1.2.2-dev` appeared, and the only
      diff was the one new JSON field — reverted after verification (not
      requested to actually update codass again right now)

---

## 6. Validation

### Strategy
Unit-level in `pose-mcp/internal/cli`: install fixture, assert the
manifest's new field, assert `pose version <path>` output contains it;
separately, hand-write a manifest without the field and assert the line is
absent, not blank/guessed.

### Requirement trace
- R1 [satisfied] `TestVersionWithPathArgReportsRecordedEngineVersion`
  (manifest side); also manually reverified against codass.
- R2 [satisfied] `TestVersionWithPathArgReportsRecordedEngineVersion`
  (resolves `repo` from an unrelated cwd via the path argument).
- R3 [satisfied] same test, asserts the printed line.
- R4 [satisfied] `TestVersionOmitsEngineVersionForPreExistingManifest`.

### Known gaps
- None identified.

---

## 7. Final Report

### Delivered scope
`saveMachineryManifest` now records the delivering engine's `Version` in
`.pose/state/machinery-manifest.json` (`engine_version`, additive/
omitempty). `pose version` accepts an optional target-path argument
(`pose version <path>`, resolving that instance instead of cwd, via a new
`projectRootAt(dir)` that `projectRoot()` itself now delegates to) and
prints `instance last updated by: pose <version>` when the resolved
instance's manifest carries one — omitted entirely, never guessed, for a
manifest that predates this field.

### Files and modules changed
- `pose-mcp/internal/cli/machinery.go`: `EngineVersion` field,
  `instanceEngineVersion` helper, `saveMachineryManifest` populates it.
- `pose-mcp/internal/cli/cli.go`: `projectRootAt`, `version` dispatch
  takes an optional arg, `cmdVersion` signature and output.
- `pose-mcp/internal/cli/cli_test.go`: two new regression tests.

### Validation executed
- `go -C pose-mcp test ./internal/cli/... -run TestVersion`: SUCCESS.
- `go -C pose-mcp test ./...`: SUCCESS.
- `go -C pose-mcp vet ./...`: SUCCESS.
- `gofmt -l`: clean.
- Manual end-to-end against `~/GolandProjects/codass`: confirmed the full
  record-then-report loop works on a real, previously-untouched instance.

### Residual risks
- None identified; purely additive JSON field and an optional CLI
  argument, no existing call site's behavior changed.

### Follow-ups
- [wont-do: no further work identified — R1-R4 fully closes the gap issue #20 reported]
