# ADR: Upgrade compatibility lab — populated fixtures over bare-install fixtures

## Status
Accepted (2026-07-19) — spec `pose-upgrade-compatibility-lab`

## Context

`pose-release-compatibility-matrix` already exercises every entry in
`compatibility.json`'s `supported_upgrades` against a real, checksum-pinned
prior release binary — but the fixture it upgrades is a bare fresh install:
empty specs, default locale, no user edits. That proves the schema bump
itself works, but not the property adopters actually care about — that
`pose upgrade` on a real, lived-in repository (populated specs, a
translated `AGENTS.md`, a hand-edited rule) leaves everything untouched
except the schema. `cmdUpgrade` (`internal/cli/maintenance.go`) also had
zero unit test coverage of its own before this spec — every prior
assertion came transitively through the network-dependent shell gate.

Alternatives considered:

1. **Leave the bare-install fixture as-is, add only Go unit tests for
   `cmdUpgrade`.** Fast and network-free, but never proves the exact thing
   R2/R3 ask for: that a *populated* instance's specs, knowledge, locale
   content and user edits survive a real upgrade end to end.
2. **Build a separate, dedicated "upgrade lab" test binary/harness outside
   the existing `internal/cli` package and `compat.sh` script.** More
   ceremony for no real benefit — the upgrade engine already lives in
   `internal/cli`, and the real N-minus proof already lives in
   `compat.sh`; a third home would just fragment ownership.
3. **Deepen both existing homes: network-free Go fixtures for R2/R3, real
   N-minus pairs for R1 in `compat.sh`, both driving the same "populated
   instance" shape** (pt-BR locale install, a real spec, a real knowledge
   note, a hand-edited managed file), and add a symlink-escape guard to
   `cmdUpgrade` itself for the security requirement.

## Decision

Option 3.

- **`internal/cli/upgrade_test.go`** builds a populated fixture
  (`newPopulatedUpgradeFixture`): `pose install --locale pt-BR`, a real
  `new-spec`, a real `new-knowledge handoff`, and an appended marker in
  `AGENTS.md` simulating a user edit — then rewinds `schema-version` to
  simulate a pre-upgrade instance. Four properties are proven directly
  against this fixture, network-free, on every `go test` run:
  - **Dry-run accuracy:** a full tree hash snapshot before/after
    `--dry-run` must be byte-identical (R3).
  - **Apply + idempotent reapply:** on an already-fully-populated instance,
    applying the upgrade changes exactly one file
    (`.pose/schema-version`) — nothing else — and reapplying is a strict
    no-op (R3).
  - **Preservation:** the pt-BR `AGENTS.md` content, the appended user
    marker, the populated spec and the knowledge note all survive
    byte-for-byte (R2, R3).
  - **Explicit remediation, not partial upgrade:** an instance newer than
    the engine fails with an actionable diagnostic and mutates nothing
    (Compatibility requirement).
- **Security — symlink escape:** `ensureManagedDirSafe` (new helper in
  `maintenance.go`) replaces the bare `os.MkdirAll` calls `cmdUpgrade` used
  for its three managed directories. It `Lstat`s the target and every
  existing ancestor under the instance root and refuses (rather than
  silently following) a symlink — proven by
  `TestUpgradeBlocksManagedDirSymlinkEscape`, which plants a symlinked
  `.pose/roadmaps` pointing outside the repository and asserts nothing was
  written through it and the schema version did not advance. `.pose`
  itself gets the same `Lstat` symlink check before `cmdUpgrade` does
  anything else. "Authenticate prior binaries" (the other half of the
  security requirement) was already satisfied by `compat.sh`'s existing
  SHA-256 pin check on `checksums.txt` before executing any downloaded
  prior binary — unchanged by this spec.
- **`tests/release/compat.sh`** gets the same populated-fixture shape for
  real N-minus pairs (R1): `check_upgrade_pair()` installs with
  `--locale pt-BR`, seeds a spec and knowledge note with the *prior*
  binary, edits `AGENTS.md`, upgrades with the *candidate* binary, asserts
  `check --strict` passes, reapplies for idempotency, and hashes
  `AGENTS.md` before/after to prove preservation. Currently exercises zero
  pairs (`supported_upgrades` is empty until the first release), same as
  before this spec — the depth is ready the moment a real pair exists.

## Consequences

- Positive: `cmdUpgrade` went from zero dedicated tests to full R2/R3
  coverage plus a real security fix, without adding a new package or test
  harness — everything lives where the code it tests already lives.
- Positive: the symlink guard is generic (`ensureManagedDirSafe` takes any
  root-relative path) — a future fourth managed directory gets the same
  protection automatically.
- Negative: `check_upgrade_pair` in `compat.sh` is unverified by this spec
  (no real prior release exists yet) — tracked as a follow-up to confirm
  on the first real N-minus pair, consistent with how `pose-slsa-provenance`
  and `pose-reproducible-release-verification` handled the same
  sandbox-unavailable gap.
- Neutral: the populated-fixture shape (locale + user edit + spec +
  knowledge) is now duplicated conceptually between the Go fixture and the
  shell fixture; kept as two implementations rather than one shared
  script because one runs in-process against `cmdUpgrade` directly and the
  other drives two separate real binaries (prior and candidate) — a shared
  abstraction would need to cross that process boundary for no real
  simplification.
