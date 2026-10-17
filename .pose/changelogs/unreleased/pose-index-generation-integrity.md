---
spec: pose-index-generation-integrity
category: changed
breaking: false
---

`pose index` now publishes its seven files as one generation. The files are
written under `.pose/indexes/generations/<id>/`, the existing names are
published from the same bytes, and a descriptor,
`.pose/indexes/generation.json`, naming each file's SHA-256 and the inputs
the set was computed from is published last: that is the commit. A write that
fails in the middle (an output that cannot be staged, a crash) leaves the
previous generation in force and a retry converges on the same identity; before,
the files written so far were new and the rest old, and every reader accepted
the mixture.

Readers check the set against the descriptor. A delivery graph, a review plan's
component map, the obligation snapshot's index digest and `pose suggest` read a
published set; a file that is missing, replaced, from another generation or
under a descriptor that cannot be read makes the set `unavailable`, with the
reason, in the CLI and over MCP, never an empty or mixed result. A repository
that has not run `pose index` since this release keeps working and is reported
`legacy-unversioned`. A generated index edited by hand has no effect while the
generation's copy exists, and makes the set unavailable in a clone without it;
`pose index --verify` names the file.

`pose index --verify` reads without writing and exits 1 unless the set is
current, the index files are those of the generation, and the specs, roadmaps,
policy, module metadata and validation matrix it was computed from still hold. `pose artifact-check` and `pose review bundle`
refresh `delivery-integrity.json` inside a new generation instead of editing the
file alone. Old generations are kept up to three and never removed within ten
minutes; a retention failure is a warning. `generation.json` and `generations/`
are not synced by `pose update`.
