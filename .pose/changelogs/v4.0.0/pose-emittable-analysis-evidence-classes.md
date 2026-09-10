---
spec: pose-emittable-analysis-evidence-classes
category: added
breaking: true
refs:
---

`lint`, `typecheck`, `security-scan` and `contract` are now evidence classes a
check may emit and a review profile may demand. They were reported as `build`,
or declared nothing at all — so a criterion asking for security assurance was
satisfied by a successful compilation, and the results of `npm run lint` and
`npm run typecheck` were discarded when a review collected evidence.

That is the same indistinction the single vocabulary closed between profiles and
checks, one level down: a gate that reads as met by evidence saying nothing
about it.

`go vet` now emits `lint` rather than `build`; it is a linter, and calling it a
build was the collapse in its clearest form. Because it was the Go stack's only
`build` producer, a `go build ./...` check was added — its absence is why vet
carried the label.

An instance that declares one of the four cannot be validated by an engine
predating them — `pose check --strict` on an older binary fails with `unknown
evidenceClass`. That is the adoption cost of any change to a closed set, and the
set being closed is what makes it worth having. No shipped profile demands any
of the four, so nothing changes for an instance until it opts in.

The stacks catalog exists twice — in a repository's own matrix and in the
shipped scaffold, because the file is excluded from the byte-for-byte sync — and
the two are now held equal by a test. They had agreed by hand, with nothing
checking, and the first edit of this change moved one and not the other.
