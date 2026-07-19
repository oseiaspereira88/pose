# Kit: direct adoption

`fixture/` is a small pre-existing repository — a README and a Go module
under `service/` — with no spec system of any kind. This is the plain
"add POSE to an existing repo" path.

Verified end to end by `TestBrownfieldDirectAdoptionKit`.

## Stage 1 — visibility

```bash
pose doctor
```

Before adoption this reports `.pose/ not found` and fails — an honest,
immediate signal rather than a silent no-op.

## Stage 2 — adoption

```bash
pose install . --skip-mcp
```

`pose install` ends with its own `check --strict` gate — it does not
report success unless the newly-installed structure is already
internally consistent. `README.md` and `service/` are untouched; nothing
your repository already tracked is rewritten.

## Stage 3 — blocking gate

```bash
pose init --wizard --yes   # detects service/go.mod, registers it
pose validate --tolerant   # visibility: run the module's checks, don't block
pose validate --strict     # blocking gate: now enforced
pose check --strict        # POSE's own structural gate, enforced
```

## Rollback

Nothing pre-existing was modified (only `.pose/`, `.agents/`, `AGENTS.md`,
`POSE.md` were added). `git clean -fdx` before committing removes the
adoption entirely; `git revert` after committing does the same.
