# Migrating from GitHub Spec Kit

**Doc type:** How-to &nbsp;·&nbsp; **Applies to:** POSE 5.x (current stable)

If you already write specs with Spec Kit, you do not have to abandon them to
try POSE. `pose import spec-kit` reads your existing feature specs and produces
POSE specs from them.

Before the steps, one thing worth being explicit about, because it changes how
you should read the rest of this page.

## POSE becomes the lifecycle authority

POSE is itself a Spec-Driven Development framework. Once adopted in a
repository, it owns the authoritative lifecycle for specifications:
requirement IDs, status, dependencies, readiness, the definition of done,
closeout and the knowledge that outlives a session.

Running Spec Kit and POSE over the same lifecycle produces competing
authorities for exactly those artifacts. Which tool owns the requirement ID?
Which one decides the work is done? Which one closes it? Those questions have
no good answer, and they surface at the worst moment — when you are trying to
close something.

So the supported model is **migration interoperability, not concurrent
ownership**. The importer is a one-way door by design: it does not synchronize
later Spec Kit changes back, and it is not meant to.

If you would rather keep Spec Kit as your planning tool and never adopt POSE's
lifecycle, that is a legitimate choice — but then POSE is not the right tool
for you, and this guide will not make it one.

## 1. Look before you write

The importer never writes on a dry run. Start there:

```bash
pose import spec-kit .specify/specs --dry-run
```

You get one `import.spec` line per Spec Kit feature — slug, source,
destination, how many requirements were extracted, how many source artifacts
were consumed — plus an `import.curation` line for every gap the importer
found, and a summary confirming `written=0`:

```
import.spec slug=customer-export format=spec-kit source=customer-export/spec.md
  destination=.pose/specs/customer-export/spec.md requirements=2 artifacts=3 action=dry-run
import.curation slug=customer-export warning="unmapped source section \"Out Of Scope\";
  review it in the original artifact"
import.summary specs=1 warnings=1 written=0 dry_run=true
```

Read the `import.curation` lines first. They are the honest part of the
output: every top-level section the importer did not recognize is reported
here rather than dropped silently.

Check `requirements=` against what you expect. If a feature you know has six
`FR-*` items reports `requirements=1`, the importer did not find them and
generated a placeholder — better to learn that now than after writing.

The full rendered spec, including its **Import Provenance** section, is
written to disk on a real run (step 4), not printed here.

## 2. What transfers

| Spec Kit source | Becomes |
|---|---|
| The `# Feature Specification:` heading | The spec title |
| The feature directory name | The spec slug |
| `## User Scenarios & Testing`, or the `Input:` line | Intent |
| `FR-*` requirements | `R1`…`Rn` acceptance criteria |
| `## Success Criteria`, `## Assumptions` | Technical Plan |
| `plan.md` | Technical Plan (Imported Implementation Plan) |
| `tasks.md` | Tasks |

Any top-level section the importer does not recognize is not silently dropped
— it is reported as a curation note so you can decide what to do with it.

## 3. What does not transfer

This is the part worth reading twice. A migration path that hides its losses
produces an angry user at the first gate.

- **Requirement IDs are renumbered.** `FR-003` becomes `R3` only if it happens
  to be third. Anything outside the spec that referenced `FR-003` — a commit
  message, a ticket, a code comment — now points at nothing. Search for those
  references before you delete the source.
- **Status and lifecycle do not carry over.** Every imported spec arrives as
  `status: draft`, regardless of how finished the Spec Kit feature was. POSE
  will not take your word for it; readiness is something it checks.
- **Dependencies do not carry over.** `depends_on` is empty. If your features
  had an ordering, you re-declare it.
- **Validation is a placeholder.** The generated Validation section lists
  "define the module test command" and similar. POSE runs *your repository's*
  checks, and it cannot infer them from a Spec Kit document.
- **Decisions are empty.** Nothing in the Spec Kit format maps to POSE's
  Decisions section. If a trade-off mattered, record it during curation or it
  is lost.
- **Priority defaults to 2** for every imported spec.

## 4. Import, then curate

```bash
pose import spec-kit .specify/specs
```

Then, for each imported spec, work through the Import Provenance notes and
fill what did not transfer. The generated follow-up says the same thing:

```
- [open] Complete import curation and run `pose lint-spec <slug> --ready-check`
  before marking this spec in-progress.
```

That follow-up is not decoration. `--ready-check` is the entry gate, and an
uncurated import will not pass it — which is the intended behaviour, not an
obstacle to route around.

## 5. Run the first governed loop

Pick the smallest imported spec and take it all the way through:

```bash
pose lint-spec <slug> --ready-check   # is it ready to start?
pose suggest feature                   # what applies to this work?
pose validate --strict                 # does the repository agree it is done?
```

Start with the smallest one. The point of the first loop is to see a gate
block for a reason you understand, not to migrate everything.

## 6. Keep the source until you are sure

Nothing forces you to delete `.specify/` on the same day. Keep it until the
imported specs have been curated and at least one has been through a full
loop. The importer does not need it afterwards, and it does not read it again.
