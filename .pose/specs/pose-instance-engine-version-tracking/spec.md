---
slug: pose-instance-engine-version-tracking
status: draft
created_at: 2026-08-15
completed_at:
supersedes:
depends_on:
priority: 3
components: pose-mcp, cli
delivers:
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
- [ ] Add `EngineVersion string` to `machineryManifest`; populate it in
      `saveMachineryManifest`'s caller(s) from `version.Version`
- [ ] Accept an optional path argument in the `version` dispatch case and
      `cmdVersion`; resolve the instance there instead of cwd when given
- [ ] Print the recorded engine version line only when present

### Validation
- [ ] Regression test: fresh install records `version.Version` in the
      manifest; `pose version <target>` prints it
- [ ] Regression test: a manifest without the new field (simulating a
      pre-existing instance) — `pose version` omits the line, no error
- [ ] `go -C pose-mcp test ./...`, `go vet ./...`, `gofmt -l`

---

## 6. Validation

### Strategy
Unit-level in `pose-mcp/internal/cli`: install fixture, assert the
manifest's new field, assert `pose version <path>` output contains it;
separately, hand-write a manifest without the field and assert the line is
absent, not blank/guessed.

### Known gaps
- None identified.

---

## 7. Final Report

### Follow-ups
- [open] 
