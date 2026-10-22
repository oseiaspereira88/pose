# Command compatibility

**Doc type:** Reference · **Applies to:** POSE 7.x (published baseline v7.4.1)

**Reviewed:** 2026-10-20. This page distinguishes deprecated syntax from supported
compatibility entrypoints. It was checked against the CLI parsers and the
published v7.4.1 binary's help. A command missing from the short top-level help
is not by itself evidence that the command was removed.

## Deprecated, supported and refused

| Older form or confusing name | Status | Use or interpretation |
|---|---|---|
| `pose validate --json <path>` | Deprecated, still accepted with a warning | Use `pose validate --json-out <path>` for a file. In v7.4.1, bare `pose validate --json` is refused because it still requires a path. |
| `pose release-notes --version vX.Y.Z` | Supported compatibility alias | Prefer `pose release notes --version vX.Y.Z`. Both read the prepared immutable snapshot; neither freezes pending fragments. |
| `pose review record <scope> …` | Supported compatibility entrypoint | For the sealed path, prepare a bundle and record a separate attestation. `record` still requires a current sealed bundle when that policy is active. |
| `pose new-spec <slug> --legacy` | Supported layout option | It creates `<slug>/spec.md`. Use the default dated file or `--folder` for new work; existing legacy layouts remain readable. |
| `pose migrate v7 --dry-run` | Supported read-only inventory | Despite its name, it does not apply a version upgrade or rewrite sealed records. Use `pose update` for the supported instance upgrade flow. |
| A validation check with a shell `command` string | Unsupported | Declare `program`, `args` and optional `env` in the validation matrix. |
| `brew install --formula <url>` | Unsupported Homebrew path | Use the verified release archive while a supported tap is unavailable; see [package channels](package-channels.md). |

`pose release-notes --preview` remains a read-only preview of pending fragments;
it does not prepare or publish a release. Prefer the governed release plan when
you also need eligibility, version and blocker checks.

Do not rewrite historical evidence just because it names an older command.
Update executable instructions and examples; preserve dated release notes and
frozen review records as history.

## JSON flags differ by command

For a gate such as `pose check`, `--json` selects the shared gate result.
In the published v7.4.1 binary, `pose validate --json` without a value exits 2;
use `--json-out <path>` and read that file when integrating validation results.
The deprecated spelling is the form that **includes** a path, not a stdout switch.

A review plan, an Attention projection and governance stats have different
schemas because they represent different domain objects. Read each command's
reference before binding an automation to fields.

`--json-out` is supported by the machine-channel gates; it is not a universal
flag for every command. With `validate`, the output path stays inside the
project. Parser errors and progress can still be written to stderr. Preserve
stderr and the exit code when diagnosing automation failures.

New typed closeout operation results, stricter parsers and recovery flags on
the development branch remain [unreleased](evolution.md#on-the-development-branch-not-in-v741).
They must not be inferred from the v7.4.1 command catalog alone.

## Inspect safely

```bash
pose version
pose --help
pose validate --help
pose release --help
pose review --help
```

Use the [CLI reference](cli.md) for the command map, [instance operation](operate.md)
for effective adoption and [releases and development](evolution.md) for availability.
Help explains syntax; a write still needs the applicable preview, current digest,
authority and evidence.
