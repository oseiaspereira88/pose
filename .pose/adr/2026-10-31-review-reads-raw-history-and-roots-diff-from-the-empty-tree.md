# ADR: Review reads raw history, and a root change set diffs from the empty tree

## Status
Accepted (2026-10-31), spec `pose-history-readers-read-raw`.

## Context

The delivery graph read raw object history while structural review honoured replace refs and grafts, so a commit could mean two things to two gates. A trailer change set was resolved from `<first>^`, which a repository's first commit does not have, so a new project's first spec had none. `resolved_base` is a contract: facts, reports, the review subject and the policy baseline read it, and it must be filled.

## Decision

- Raw history is one rule in `gitCommand`, by environment, so a caller cannot forget it. `RawHistoryGit` is a call of it.
- A root change set has the empty tree of its object format as `resolved_base`: what Git itself diffs a root against. It is the only base that is not a commit, recognised by `IsGitEmptyTree`.
- Rootness is read from the commit object's header, and only after `<first>^` fails. A shallow boundary also has no parent to walk to; only the object tells it from a root, and it stays unresolved.

## Consequences

- A repository with replace refs or grafts gets other review inputs: `pose index` and a reseal settle it.
- A reader of `resolved_base` that needs a commit must first ask `IsGitEmptyTree`; the policy baseline reads it as no contract before the first commit.
- A commit whose parent resolves keeps its identity, and costs no extra Git call.

## Alternatives

- An empty `resolved_base`: readers already treat that as an unresolved change set.
- Flags at each call: the five readers that missed raw history show how that ends.
- A parent walk for rootness: it cannot tell a shallow boundary from a root.
