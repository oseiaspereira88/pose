# Migrating from OpenSpec

**Doc type:** How-to &nbsp;·&nbsp; **Applies to:** POSE 5.x (current stable)

`pose import openspec` reads OpenSpec capability specs and change folders and
produces POSE specs from them, so existing work comes with you.

Read [the authority note in the Spec Kit
guide](spec-kit.md#pose-becomes-the-lifecycle-authority) first if you have not
— it applies identically here. POSE owns the specification and delivery
lifecycle once adopted; the importer is a one-way door by design and does not
synchronize later OpenSpec changes back.

## 1. Look before you write

```bash
pose import openspec openspec/changes/add-2fa --dry-run
```

Point it at a change folder or at a capability spec. Nothing is written on a
dry run.

## 2. What transfers

| OpenSpec source | Becomes |
|---|---|
| `### Requirement:` sections | `R1`…`Rn` acceptance criteria |
| `## Purpose` | Intent |
| `proposal.md` (when importing a change) | Intent, replacing Purpose |
| `design.md` | Technical Plan (Imported Design) |
| `tasks.md` | Tasks |
| The capability directory name | The spec slug |
| A change folder | Slug becomes `<change>-<capability>` |

When you import from inside a change folder, the importer walks up to the
change root and picks up `proposal.md`, `design.md` and `tasks.md` alongside
the capability spec. Importing a bare capability spec gets you the capability
only.

## 3. What does not transfer

- **`### Requirement:` sections are mandatory.** Unlike the Spec Kit importer,
  which invents a placeholder criterion when it finds no `FR-*`, this one
  fails outright: *"OpenSpec file … has no '### Requirement:' sections"*. A
  capability described only in prose does not import. That is deliberate — a
  spec with no acceptance criteria cannot pass POSE's entry gate anyway, so
  producing one would only move the failure later.
- **Requirement IDs are renumbered** into `R1`…`Rn` by order of appearance.
  External references to the original requirement names break.
- **Scenarios do not become tests.** The behaviour described under each
  requirement is carried as text. POSE runs your repository's real checks; it
  does not synthesize them from a scenario.
- **Status, dependencies and priority do not carry over.** Every imported spec
  arrives `status: draft`, with empty `depends_on` and `priority: 2`.
- **Deltas and archives have no equivalent.** OpenSpec's change/archive model
  and POSE's spec lifecycle are different shapes. What imports is the content;
  the archival relationship is not reconstructed.
- **Decisions are empty.** Anything in `design.md` lands in the Technical Plan,
  not in POSE's Decisions section. If a rejected alternative matters, move it
  during curation — or record it as an ADR, which is where POSE keeps
  decisions that must outlive the spec.

## 4. Import, then curate

```bash
pose import openspec openspec/changes/add-2fa
```

Work through the Import Provenance curation notes on each generated spec. Any
top-level section the importer did not recognize is listed there rather than
dropped silently.

## 5. Run the first governed loop

```bash
pose lint-spec <slug> --ready-check
pose suggest feature
pose validate --strict
```

`--ready-check` is the entry gate. An uncurated import is expected to fail it;
that is the gate doing its job, not a migration defect.

## 6. A note on multiple changes

Importing several change folders at once produces several specs. POSE requires
slugs to be unique, and the importer checks the whole batch before writing
anything: on a collision it refuses everything rather than writing half, and a
partial write failure rolls back the directories it created. If two changes
touch the same capability, you will get a collision — resolve it by importing
them one at a time and giving the second a distinct slug during curation.

The same preflight refuses to overwrite a spec that already exists. Re-running
an import is therefore safe: it will stop rather than clobber curation work you
have already done.
