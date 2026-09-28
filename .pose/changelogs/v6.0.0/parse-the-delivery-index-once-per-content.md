---
spec: parse-the-delivery-index-once-per-content
category: changed
breaking: false
---

The delivery-integrity index is parsed once per distinct content instead of once per
caller, and the structural design delta is computed once per input digest — the value
it already published as `CacheKey`. In this repository one `pose check --strict` was
reading and unmarshalling a 4.5 MB index 528 times and recomputing 36 identical design
deltas. The independent per-spec gate loops also run concurrently now, with each
item's findings replayed in item order so the output stays diffable. The gate goes from
39.3 seconds to 16.6, and from 634.9 across both performance fixes — 38x — with the
findings and their order unchanged. Both memos are keyed on content rather than
on modification time, and both return copies, because callers filter what they receive
in place.
