# ADR: Carry unchanged checks into the canonical validation result

## Status

Accepted on 2026-11-22 for `pose-validate-carries-unchanged-checks`.

## Context

The canonical structured result at `results_path` is the only validation evidence that review bundles and closeout load. Producing it means running the whole matrix. `--changed-from` already selects the modules a change can affect: it uses changed paths, declared `dependsOn` edges and the high-criticality policy, and falls back to every module for a change outside all modules. Its record, though, marks the other checks as skipped, so it cannot stand in for the canonical result. In an instance with a slow and partly intermittent matrix, every review round pays for the full run.

## Decision

Add `validate --carry`. The flag uses the existing canonical result as a base when that base is provably comparable:
- the same schema, mode, matrix digest and engine version;
- a head that is an ancestor of the current head;
- not cancelled;
- a clean material working copy.

With a usable base, the run executes the changed-scope selection between the two heads. That selection ignores the canonical result file and the derived administrative records that material bindings already ignore: the engine's closed list, still regular files at `HEAD`. It copies the base results of every unselected module. Each copied result keeps its outcome and states the head where it really ran (`carried_from`) and when (`carried_generated_at`). The delivery reading uses that head for the result. Any unmet condition runs everything and records the reason.

A module may declare `"selfContained": true` in its metadata. A change outside every module then stops selecting it. The declaration is the instance's statement that the module's checks read only its own directory and its declared dependencies, so it applies to `--changed-from` too. Without it, the existing safe fallback stays.

## Consequences

A change confined to one module reruns that module, the modules enclosing it (the root module included), the modules beneath the changed path and its declared dependents. Everything else is reused with its true origin visible. Only a changed file that no module owns widens the run to every module that does not declare itself self-contained. Consumers that compare a result's head with a subject head now see carried results as observations of the earlier head, which is accurate. An incorrect `selfContained` declaration can carry a stale result. That risk sits with the instance's explicit declaration and is documented next to it.

## Alternatives considered

- Hashing each check's declared inputs would be more precise. It needs every check's inputs declared and verified, and it is a larger contract.
- Merging a partial `--changed-from` record into the canonical file by hand would hide which results were reused.
- Carrying across matrix or engine changes would make a definition change invisible to the evidence.
