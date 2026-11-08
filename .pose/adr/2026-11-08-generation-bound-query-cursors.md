# ADR: Generation-bound query cursors

## Status
Accepted (2026-11-08), by spec `pose-generation-bound-query-cursors`.

## Context

The paginated MCP list tools (`pose_list_specs`, `pose_list_roadmaps`,
`pose_list_knowledge`, `pose_list_reports`, `pose_capability_history`) used a
version 1 cursor that was base64 over an offset. A cursor named no query and no
collection: after an insertion, a removal, a rename or a reordering between two
pages the next page silently skipped or repeated items, a cursor could be
replayed on another tool, filter set or project, and a hand-written token was
accepted. The same collections are recomputed on every call, so a stale
continuation could not be told from a current one.

## Decision

A cursor is a signed token that continues one query over one collection.

- **Payload (version 2):** tool, project binding (logical project id and a digest
  of the root), normalized filters and ordering, the digest of the ordered
  collection together with the index generation it was read from, the position
  (offset and the key of the last item) and an expiry.
- **Protection:** HMAC-SHA256 over the payload with a key held by the server
  process. Base64 is transport, not integrity. A cursor from another process, or
  from before a restart, fails the signature and the caller restarts from the
  first page. No daemon and no resident state: a one-shot CLI needs no cursor.
- **Continuation:** the collection is recomputed and its digest compared; a
  changed collection, generation, tool, project or filter set is refused with a
  `cursor-*` code (`cursor-stale`, `cursor-binding`, `cursor-signature`,
  `cursor-expired`, `cursor-position`, `cursor-unsupported-version`,
  `cursor-malformed`). It never continues over a changed collection and never
  restarts silently at page zero. Version 1 tokens are refused as unsupported,
  not reinterpreted.
- **Bounds:** a token is at most 2048 bytes and lives 15 minutes; a page is at
  most 500 items (a larger `limit` is clamped and the clamp is disclosed).
- **Disclosure:** every page carries its generation, collection digest,
  position, total and a statement that it is a navigation view of a collection
  read for the call: an absent `next_cursor` means that collection was
  exhausted, not that nothing is pending elsewhere, and a page authorizes no
  gate or effect (those read current inputs).
- **Compatibility:** a call without cursor and limit returns everything, as
  before. `limit` without a cursor starts a paginated walk.

## Consequences

- No duplicated or skipped item between pages: either the same collection or an
  explicit restart. The cost of a page is one recomputation of the collection,
  which every call already paid; the page is cut after the domain layer orders
  and filters.
- A cursor never outlives a server restart. That is deliberate and disclosed.
- The total is the size of the recomputed collection and costs a full scan for
  these list tools, which load their whole collection; the cost is stated, not
  hidden behind an optimization that does not exist.
- Adding a paginated tool means calling the shared helper with the tool's
  filters and keys.

Rejected: retaining a snapshot per cursor (unbounded memory, a daemon the CLI
does not have); an unsigned token with a generation field (trivially forged,
replayable across queries); keying the collection by offset alone (cannot tell
a reordering from the same collection); a durable key on disk (a secret in the
repository tree for little gain over a restart).
