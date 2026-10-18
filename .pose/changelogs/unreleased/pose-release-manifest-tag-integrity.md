---
spec: pose-release-manifest-tag-integrity
category: changed
breaking: false
---

A release manifest is now read through a strict contract: a supported `schema_version`, no
unknown fields, a version that is the release's own, clean paths, and the redundant fields
(`specs`, `categories`, `breaking`, `release_input_digest`) recomposed from the fragments the
manifest freezes. A manifest whose projections were edited, or that is only a parseable JSON
object, is refused with a localized diagnostic instead of being treated as the cut.

`pose release check` and `pose release status` verify a tag from its own tree. The tag, light or
annotated, is peeled to its commit; the manifest, the notes, the frozen fragments and the
version source are read as blobs of that commit, and a tag whose tree holds no manifest is
refused even when the working tree holds a valid one. A later commit, provider evidence or the
next cycle does not make a correct cut stale. `status` reports the tag's snapshot (`snapshot`),
and `check --json` the version sources it could not recompute (`snapshot_limits`).

Stricter, classified minor: a manifest that was only loosely valid is refused, and a governed tag
needs its manifest in the tagged tree.

The manifest is also read as exactly one JSON document with the schema's exact field names, every
required field and no repeated key, and its repeated fields are recomposed from the fragment blobs
and the release policy of the tagged tree. Release versions are ordered by Semantic Versioning
precedence (`v1.0.0-rc.2` before `v1.0.0-rc.10` before `v1.0.0`), a manifest's `previous_version`
must be older than its release, and a release whose tag does not hold an intact snapshot is no
longer treated as tagged, published or verified by `release status`, the obligation queue, the
public-claims gate or `release open-next`.
