# ADR: The release version source is declared by the project

## Status

Accepted — 2026-10-02

## Context

`pose release plan`, `prepare` and `check` compared the requested version with the
compiled version of the `pose` binary, recorded as the evidence file
`pose-mcp/internal/version/version.go`. That is the authoritative version only in
the engine repository. In any other project the comparison could only succeed for
the engine's own number. A fixture repository with an adopted policy and a valid
fragment answered `plan --version v0.2.0` with `target v0.2.0 differs from
authoritative version evidence v6.1.0`, and Harne8 accumulated 204 unreleased
fragments because no cut could be planned.

## Decision

The release policy gains an optional `version_source` object naming a
project-relative file: a one-line `text` file, or a `json` file read through one
top-level string `key`. The value must be `X.Y.Z`, with an optional leading `v` and
`-dev` suffix; a pre-release tag is not a release version. The path is confined to
the project, the file is read once and bounded, and the manifest records only the
declared path, the kind and the compared version.

Without `version_source`, the compiled engine version stays authoritative only when
the project contains the engine's version file. Any other project is refused with a
message that names the field. The field is a pointer with `omitempty`, so a policy
that leaves it unset serializes, and digests, as before.

Rejected: reading the latest Git tag, which makes the tag both the input and the
output of the cut; and trusting `--version`, which removes the gate the workflow
describes as "update and review the authoritative version".

## Consequences

A project that adopts release closeout needs a versioned file holding its version,
and updates it in reviewed work before planning, as the workflow already says. The
policy stays at `schema_version` 1; an engine that does not know the field fails
with the same version mismatch as before rather than guessing. Manifests already in
this repository are unchanged and still verify. A project released from tags alone
has no source kind here; that needs its own decision. Rollback is removing the
field from a policy and the selection in `releaseVersionEvidence`.
