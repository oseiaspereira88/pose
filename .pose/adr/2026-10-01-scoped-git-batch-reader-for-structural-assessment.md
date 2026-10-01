# ADR: Scoped Git batch reader for structural assessment

## Status

Accepted — 2026-10-01

## Context

Structural assessment starts separate existence and content processes per blob.
The check-worker-count follow-up identified process creation as remaining work.
A shared global stream would require concurrency, root and lifetime coordination.

## Decision

Own one lazily started `git cat-file --batch` process per structural collector.
Use the default Git framing, validate request delimiters, inspect the object size
before allocation, and close/kill/wait explicitly. Give the process a bounded
lifetime. Preserve the existing path/revision and symlink checks before requests.
Do not add a persistent cache, global pool, shell expansion or content filters.

Protocol source: [Git cat-file batch output](https://git-scm.com/docs/git-cat-file#_batch_output).
The default input treats the whole line as one object expression; output carries
object identity, type and size, then exactly that many content bytes and a newline.

## Consequences

Successful reads share a process within one assessment. Each collector remains
single-consumer, matching its existing mutable counters/maps. Corrupt framing or
oversized data terminates the process; failure is never a successful observation.
This optimization does not promise to remove every Git spawn in pose check:
federation, subject classification and attribution have independent callers.
Rollback is reverting the collector integration and returning to separate reads.
