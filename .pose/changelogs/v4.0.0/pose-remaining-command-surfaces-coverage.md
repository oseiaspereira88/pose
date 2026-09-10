---
spec: pose-remaining-command-surfaces-coverage
category: fixed
breaking: false
refs:
---

`review-check`, `history-check`, `docs-sync`, `docs-review`, `contribute
submit`, `roadmap-check` and the helpers the Definition of Ready is built from
now execute under test, through the dispatch a user reaches them by. None of
them had run a statement under the suite, so every refusal each one makes was
an assumption — and a refusal is an exit code a script branches on.

`contribute submit` reaches `gh issue create` and `docs-sync push` writes to a
Conductor: both are covered up to the outward call and no further. The
contribute assertion also requires the refusal to happen *before* the command
announces a submission.

Three fixtures had to be corrected against what the commands actually require
rather than against what they happened to answer, each found by an assertion
failing. The last one surfaced something worth its own change: `roadmap-check`
returns before loading any roadmap when the delivery profile index is absent, so
it reports zero cut criteria and exits 0 — a gate reading as met because the
path that would fail is never reached. The behaviour is pinned here and recorded
as a follow-up.
